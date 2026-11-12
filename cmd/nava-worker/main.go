// Command nava-worker is the platform's Temporal worker (PLAN-001 T31b). main
// only wires configuration, telemetry, the guarded Temporal client and worker,
// and the admin listener together.
//
// The worker's client and worker come only from workerguard.ConfigureClient
// and workerguard.WorkerOptions; activity code gets no Temporal client
// (AR-025). /metrics, /healthz and /readyz are served on NAVA_ADMIN_ADDR by
// telemetry.NewAdminHandler, which is not wrapped in httpobs.Middleware
// (TICK-94 item 1). There is no public listener: NAVA_HTTP_ADDR is ignored.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	tlog "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/temporaltls"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The namespace and the only task queue on the Authorizer's allow-list
// (docs/temporal-server.md, D9: platformTaskQueues = {nava-tq}).
const (
	namespace = "nava"
	taskQueue = "nava-tq"
)

// service names the binary on the logs it exports.
const service = "nava-worker"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.LookupEnv, os.Stdout)
	stop()
	if err != nil {
		slog.ErrorContext(ctx, "nava-worker stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

// run starts the worker, then serves the admin listener until ctx is
// cancelled, the listener fails or the worker stops on a fatal error. It stops
// the worker before draining the listener, and flushes telemetry last. A fatal
// worker error is returned, so the process exits non-zero and compose
// restarts it instead of /readyz staying 200 with no poller.
func run(ctx context.Context, lookup config.LookupFunc, stdout io.Writer) error {
	cfg, err := config.Load(lookup)
	if err != nil {
		return err
	}
	if cfg.Deps.TemporalAddress == "" || cfg.Deps.DatabaseURL == "" {
		return errors.New("config: nava-worker needs NAVA_TEMPORAL_ADDRESS and NAVA_DATABASE_URL")
	}
	log := logging.New(stdout, cfg.Log)
	telemetry.RouteErrors(log)

	tel, err := telemetry.New(ctx, cfg.Telemetry)
	if err != nil {
		return err
	}
	log = exportLogs(log, tel, cfg.Log.Level)
	flush := func() error {
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.Telemetry.ShutdownTimeout)
		defer cancel()
		err := tel.Shutdown(flushCtx)
		if err != nil {
			log.WarnContext(flushCtx, "telemetry shutdown failed", slog.Any("error", err))
		}
		return err
	}

	fatal := make(chan error, 1)
	w, mon, closeDeps, err := newWorker(ctx, log, tel, cfg, lookup, fatal)
	if err != nil {
		return errors.Join(err, flush())
	}
	defer closeDeps()
	// Start polls nothing until the namespace answers, so /readyz is served
	// only once the worker runs.
	if err := w.Start(); err != nil {
		return errors.Join(fmt.Errorf("temporal worker start: %w", err), flush())
	}
	// TODO(TODO-004): no alert rule fires on the guard denies this listener
	// exports (nava_worker_guard_decisions_total{outcome!="allow"}).
	admin := telemetry.NewAdminServer(cfg.HTTP.AdminAddr, cfg.HTTP, telemetry.NewAdminHandler(log, tel, mon.Routes), log)
	log.InfoContext(ctx, "nava-worker running",
		slog.String("task_queue", taskQueue), slog.String("admin_addr", cfg.HTTP.AdminAddr),
		slog.Bool("trace_export", tel.Exporting()))

	monCtx, stopMon := context.WithCancel(ctx)
	monDone := make(chan struct{})
	go func() { defer close(monDone); mon.Run(monCtx) }()
	serveErr := serve(ctx, log, admin, w, fatal, cfg)
	stopMon()
	<-monDone
	return errors.Join(serveErr, flush())
}

// exportLogs adds OTLP export, under the nava-worker name, to log. It returns
// log unchanged when log export is off.
func exportLogs(log *slog.Logger, tel *telemetry.Telemetry, level slog.Leveler) *slog.Logger {
	return logging.WithExport(log, tel.LogHandler(service), level)
}

// newWorker builds the guarded client and worker, the workflow_ref and sealer
// activities' pool and the readiness checks, registers the audit sealer, and
// registers the interpreter when
// interpreterActivities returns its ports (the database DefinitionStore in a
// production build, the fixtures in an interptest build). The
// returned func closes the client and the pools; call it after the worker has
// stopped.
func newWorker(ctx context.Context, log *slog.Logger, tel *telemetry.Telemetry, cfg config.Config, lookup config.LookupFunc, fatal chan<- error) (worker.Worker, *health.Monitor, func(), error) {
	tlsConfig, err := temporaltls.Load(cfg.Deps.TemporalTLS)
	if err != nil {
		return nil, nil, nil, err
	}
	workflows, err := workflowsMode(lookup)
	if err != nil {
		return nil, nil, nil, err
	}
	tt, err := tel.NewTemporal(log)
	if err != nil {
		return nil, nil, nil, err
	}
	c, err := client.NewLazyClient(clientOptions(log, cfg.Deps.TemporalAddress, tlsConfig, tt))
	if err != nil {
		return nil, nil, nil, errors.New("temporal: cannot create the client")
	}
	g, err := workerguard.New(log, tel.MeterProvider)
	if err != nil {
		c.Close()
		return nil, nil, nil, err
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.Deps.DatabaseURL)
	if err != nil {
		c.Close()
		// The error quotes the connection string, which is a credential.
		return nil, nil, nil, errors.New("config: NAVA_DATABASE_URL is not a valid PostgreSQL connection string")
	}
	pool, err := db.NewPool(ctx, poolCfg, tel)
	if err != nil {
		c.Close()
		return nil, nil, nil, err
	}
	pg, closePG, err := health.Postgres(ctx, cfg.Deps.DatabaseURL)
	if err != nil {
		c.Close()
		pool.Close()
		return nil, nil, nil, err
	}
	closeAll := func() { closePG(); pool.Close(); c.Close() }

	acts, err := interpreterActivities(lookup, pool)
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	if acts != nil && !workflows {
		closeAll()
		return nil, nil, nil, errors.New("config: " + WorkflowsEnv + "=off cannot run the interpreter")
	}

	// Temporal gates readiness here: a worker that cannot reach the frontend
	// does nothing. The probe uses the worker's own client.
	tc := health.Check{Name: "temporal", Critical: true, Probe: func(ctx context.Context) error {
		_, err := c.WorkflowService().GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
		return err
	}}
	mon, err := health.NewMonitor(log, tel.MeterProvider, []health.Check{pg, tc}, health.Options{})
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}

	sm, err := seal.NewMetrics(tel.MeterProvider)
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}

	// The sealer is registered unless WorkflowsEnv is off (a stack-only
	// activity worker), so the worker polls workflow tasks on nava-tq.
	w := worker.New(c, taskQueue, workerOptions(g, workflows, fatal))
	(&workerguard.Activities{Pool: pool}).Register(w)
	registerSealer(w, &seal.Activities{Pool: pool, Metrics: sm, Log: log}, workflows)
	registerInterpreter(w, acts)
	return w, mon, closeAll, nil
}

// clientOptions is the worker's Temporal client: namespace nava over mTLS,
// with what workerguard.ConfigureClient sets (Propagator, tracing first, SDK
// metrics, no worker heartbeats).
func clientOptions(log *slog.Logger, address string, tlsConfig *tls.Config, tt *telemetry.Temporal) client.Options {
	o := client.Options{
		HostPort:          address,
		Namespace:         namespace,
		Logger:            tlog.NewStructuredLogger(log),
		ConnectionOptions: client.ConnectionOptions{TLS: tlsConfig},
	}
	workerguard.ConfigureClient(&o, tt)
	return o
}

// workerOptions puts the guard last, through workerguard.WorkerOptions. The
// worker polls workflow tasks only when workflows is set; otherwise it polls
// activity tasks only: a workflow poller with nothing registered would take
// workflow tasks on nava-tq and fail them. newWorker always sets it, since
// audit.seal is always registered.
//
// The SDK stops the worker on a fatal poll error (NamespaceNotFound,
// InvalidArgument, ...) and reports it only through OnFatalError, which sends
// it to fatal (the first error; fatal has room for one) for serve.
func workerOptions(g *workerguard.Guard, workflows bool, fatal chan<- error) worker.Options {
	return workerguard.WorkerOptions(worker.Options{
		DisableWorkflowWorker: !workflows,
		OnFatalError: func(err error) {
			select {
			case fatal <- err:
			default:
			}
		},
	}, g)
}

// serve runs the admin listener until it fails, the worker reports a fatal
// error on fatal, or ctx is done; then it stops the worker (in-flight
// activities finish or are cancelled) and drains the listener, so /readyz
// answers while the worker stops. A fatal worker error is returned.
//
// On a fatal error serve does not call w.Stop: the SDK calls Stop itself right
// after OnFatalError, and two concurrent AggregatedWorker.Stop calls can both
// pass its stopC check and close stopC twice, which panics (TICK-171).
func serve(ctx context.Context, log *slog.Logger, admin *http.Server, w worker.Worker, fatal <-chan error, cfg config.Config) error {
	errc := make(chan error, 1)
	go func() { errc <- recovered(admin.ListenAndServe) }()

	var serveErr, fatalErr error
	served := false
	select {
	case err := <-fatal:
		fatalErr = fmt.Errorf("temporal worker stopped: %w", err)
		log.ErrorContext(ctx, "nava-worker fatal worker error", slog.Any("error", err))
	case serveErr = <-errc:
		served = true
		if !errors.Is(serveErr, http.ErrServerClosed) {
			log.ErrorContext(ctx, "nava-worker admin listener failed", slog.Any("error", serveErr))
		}
	case <-ctx.Done():
	}
	log.InfoContext(ctx, "nava-worker shutting down")
	if fatalErr == nil {
		w.Stop()
	}

	drainCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	shutErr := admin.Shutdown(drainCtx)
	if !served {
		serveErr = <-errc
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(fatalErr, serveErr, shutErr)
}

// recovered runs fn and turns a panic into an error carrying the stack, so a
// panic in the serve loop leads to a graceful shutdown.
func recovered(fn func() error) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("panic: %v\n%s", v, debug.Stack())
		}
	}()
	return fn()
}

// registerSealer registers the A1 and A2 activities on w, which polls nava-tq
// (ADR-004 R16, R18), behind the guard workerOptions installs, and audit.seal
// itself when workflows is set.
func registerSealer(w worker.Registry, acts *seal.Activities, workflows bool) {
	if workflows {
		seal.RegisterWorkflow(w)
	}
	acts.Register(w)
}

// WorkflowsEnv switches the worker's workflow side: "on" (the default) or
// "off". Off, the worker registers no workflow and polls activity tasks
// only, as it did before audit.seal (TICK-453). Only the stack's
// nava-worker-activities service (deploy/compose/compose.test.yaml) sets it:
// the stack cases that run their own workflow worker on nava-tq swap the
// always-on nava-worker for it, so they still have the service's activities
// (nava.workflow_ref.Close, A1, A2) but are the only workflow poller (the
// nava-tq rule). compose.yaml never sets it (TestComposeWorkerWorkflowsOn).
const WorkflowsEnv = "NAVA_WORKER_WORKFLOWS"

// workflowsMode reads WorkflowsEnv; any value but on, off or unset is an
// error, so a typo never turns the sealer off.
func workflowsMode(lookup config.LookupFunc) (bool, error) {
	v, ok := lookup(WorkflowsEnv)
	switch {
	case !ok || v == "" || v == "on":
		return true, nil
	case v == "off":
		return false, nil
	default:
		return false, errors.New("config: " + WorkflowsEnv + " must be on or off")
	}
}
