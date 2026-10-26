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

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The namespace and the only task queue on the Authorizer's allow-list
// (docs/temporal-server.md, D9: platformTaskQueues = {nava-tq}).
const (
	namespace = "nava"
	taskQueue = "nava-tq"
)

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
	log = logging.WithExport(log, tel.LogHandler(), cfg.Log.Level)
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
	w, mon, closeDeps, err := newWorker(ctx, log, tel, cfg, fatal)
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

// newWorker builds the guarded client and worker, the workflow_ref activities'
// pool and the readiness checks. The returned func closes the client and the
// pools; call it after the worker has stopped.
func newWorker(ctx context.Context, log *slog.Logger, tel *telemetry.Telemetry, cfg config.Config, fatal chan<- error) (worker.Worker, *health.Monitor, func(), error) {
	tlsConfig, err := temporalTLS(cfg.Deps.TemporalTLS)
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

	w := worker.New(c, taskQueue, workerOptions(g, fatal))
	(&workerguard.Activities{Pool: pool}).Register(w)
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

// workerOptions puts the guard last, through workerguard.WorkerOptions. No
// workflow is registered yet (T32b adds the first), so the worker polls
// activity tasks only: a workflow poller with nothing registered would take
// workflow tasks on nava-tq and fail them.
//
// The SDK stops the worker on a fatal poll error (NamespaceNotFound,
// InvalidArgument, ...) and reports it only through OnFatalError, which sends
// it to fatal (the first error; fatal has room for one) for serve.
func workerOptions(g *workerguard.Guard, fatal chan<- error) worker.Options {
	return workerguard.WorkerOptions(worker.Options{
		DisableWorkflowWorker: true,
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
