// Command nava-relay is the outbox relay (PLAN-001 T30b): it claims pending
// outbox rows as the nava_relay database role and starts or signals their
// Temporal workflows with the nava-relay certificate (ADR-001 A.2). main only
// wires configuration, telemetry, the database pool, the Temporal client and
// the admin listener together; the work is internal/relay.
//
// The Temporal client comes only from telemetry.NewTemporal and
// workerguard.ConfigureClient, so the nava-tenant header and the SDK's
// tracing and metrics are the worker's. /metrics, /healthz and /readyz are
// served on NAVA_ADMIN_ADDR by telemetry.NewAdminHandler, not wrapped in
// httpobs.Middleware (TICK-94 item 1). There is no public listener.
//
// A binary built with -tags relaytest has the T33 crash seam: with
// NAVA_RELAYTEST_PAUSE=1 it pauses before every StartWorkflow (see
// docs/architecture/outbox-relay.md). The image is built without the tag.
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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	tlog "go.temporal.io/sdk/log"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/temporaltls"
	"github.com/Navaladi2019/TicketingSystem/internal/relay"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

const (
	// namespace is the platform's Temporal namespace.
	namespace = "nava"
	// pollInterval is the wait after a poll that claimed nothing.
	pollInterval = 500 * time.Millisecond
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.LookupEnv, os.Stdout)
	stop()
	if err != nil {
		slog.ErrorContext(ctx, "nava-relay stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

// run starts the dispatch loop and serves the admin listener until ctx is
// cancelled or the listener fails. It stops the loop (an in-flight poll
// settles its rows) before draining the listener, and flushes telemetry last.
func run(ctx context.Context, lookup config.LookupFunc, stdout io.Writer) error {
	cfg, err := config.Load(lookup)
	if err != nil {
		return err
	}
	if cfg.Deps.TemporalAddress == "" || cfg.Deps.DatabaseURL == "" {
		return errors.New("config: nava-relay needs NAVA_TEMPORAL_ADDRESS and NAVA_DATABASE_URL")
	}
	log := logging.New(stdout, cfg.Log)
	telemetry.RouteErrors(log)

	tel, err := telemetry.New(ctx, cfg.Telemetry)
	if err != nil {
		return err
	}
	log = logging.WithExport(log, tel.LogHandler("nava-relay"), cfg.Log.Level)
	flush := func() error {
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.Telemetry.ShutdownTimeout)
		defer cancel()
		err := tel.Shutdown(flushCtx)
		if err != nil {
			log.WarnContext(flushCtx, "telemetry shutdown failed", slog.Any("error", err))
		}
		return err
	}

	d, mon, closeDeps, err := newDispatcher(ctx, log, tel, cfg)
	if err != nil {
		return errors.Join(err, flush())
	}
	defer closeDeps()
	admin := telemetry.NewAdminServer(cfg.HTTP.AdminAddr, cfg.HTTP, telemetry.NewAdminHandler(log, tel, mon.Routes), log)
	log.InfoContext(ctx, "nava-relay running", slog.String("admin_addr", cfg.HTTP.AdminAddr),
		slog.Bool("trace_export", tel.Exporting()))

	monCtx, stopMon := context.WithCancel(ctx)
	monDone := make(chan struct{})
	go func() { defer close(monDone); mon.Run(monCtx) }()
	serveErr := serve(ctx, log, admin, d, cfg)
	stopMon()
	<-monDone
	return errors.Join(serveErr, flush())
}

// newDispatcher builds the nava_relay pool, the Temporal client and the
// readiness checks. The returned func closes the client and the pools.
func newDispatcher(ctx context.Context, log *slog.Logger, tel *telemetry.Telemetry, cfg config.Config) (*relay.Dispatcher, *health.Monitor, func(), error) {
	tlsConfig, err := temporaltls.Load(cfg.Deps.TemporalTLS)
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

	tc := health.Check{Name: "temporal", Critical: true, Probe: func(ctx context.Context) error {
		_, err := c.WorkflowService().GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
		return err
	}}
	mon, err := health.NewMonitor(log, tel.MeterProvider, []health.Check{pg, tc}, health.Options{})
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	r, err := relay.New(pool, tel.MeterProvider, log, relay.Options{})
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	d, err := relay.NewDispatcher(r, c, tel.TracerProvider, tel.MeterProvider, log, relay.DispatcherOptions{})
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	return d, mon, closeAll, nil
}

// clientOptions is the relay's Temporal client: namespace nava over mTLS,
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

// serve runs the dispatch loop and the admin listener until the listener
// fails or ctx is done; then it waits for the loop's last poll and drains the
// listener, so /readyz answers while the relay stops.
func serve(ctx context.Context, log *slog.Logger, admin *http.Server, d *relay.Dispatcher, cfg config.Config) error {
	loopCtx, stopLoop := context.WithCancel(ctx)
	loopDone := make(chan struct{})
	go func() { defer close(loopDone); d.Run(loopCtx, pollInterval) }()

	errc := make(chan error, 1)
	go func() { errc <- recovered(admin.ListenAndServe) }()

	var serveErr error
	served := false
	select {
	case serveErr = <-errc:
		served = true
		if !errors.Is(serveErr, http.ErrServerClosed) {
			log.ErrorContext(ctx, "nava-relay admin listener failed", slog.Any("error", serveErr))
		}
	case <-ctx.Done():
	}
	log.InfoContext(ctx, "nava-relay shutting down")
	stopLoop()
	<-loopDone

	drainCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	shutErr := admin.Shutdown(drainCtx)
	if !served {
		serveErr = <-errc
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, shutErr)
}

// recovered runs fn and turns a panic into an error carrying the stack.
func recovered(fn func() error) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("panic: %v\n%s", v, debug.Stack())
		}
	}()
	return fn()
}
