// Command nava-purge deletes expired idempotency_keys, outbox and workflow_ref
// rows on a ticker, connected as nava_purge (ADR-001 A.3). It serves /metrics,
// /healthz and /readyz on the admin listener.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/purge"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.LookupEnv, os.Stdout)
	stop()
	if err != nil {
		slog.ErrorContext(ctx, "nava-purge stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

// run purges until ctx is cancelled or the admin listener fails, then drains
// the listener and flushes telemetry.
func run(ctx context.Context, lookup config.LookupFunc, stdout io.Writer) error {
	cfg, err := config.LoadPurge(lookup)
	if err != nil {
		return err
	}
	log := logging.New(stdout, cfg.Log)
	telemetry.RouteErrors(log)

	tel, err := telemetry.New(ctx, cfg.Telemetry)
	if err != nil {
		return err
	}
	log = logging.WithExport(log, tel.LogHandler(), cfg.Log.Level)

	pool, err := newPool(ctx, cfg.DatabaseURL)
	if err != nil {
		_ = tel.Shutdown(context.WithoutCancel(ctx))
		return err
	}
	defer pool.Close()
	if err := purge.AssertRole(ctx, pool); err != nil {
		_ = tel.Shutdown(context.WithoutCancel(ctx))
		return err
	}
	purger, err := purge.New(pool, tel.MeterProvider, log, purge.Options{
		Interval: cfg.Interval,
		Windows: purge.Windows{
			OutboxDispatched: cfg.OutboxDispatched,
			OutboxDead:       cfg.OutboxDead,
			WorkflowRef:      cfg.WorkflowRef,
		},
	})
	if err != nil {
		_ = tel.Shutdown(context.WithoutCancel(ctx))
		return err
	}
	mon, err := health.NewMonitor(log, tel.MeterProvider,
		[]health.Check{{Name: "postgres", Critical: true, Probe: pool.Ping}}, health.Options{})
	if err != nil {
		_ = tel.Shutdown(context.WithoutCancel(ctx))
		return err
	}
	admin := telemetry.NewAdminServer(cfg.AdminAddr, cfg.HTTP, telemetry.NewAdminHandler(log, tel, mon.Routes), log)

	log.InfoContext(ctx, "nava-purge running", slog.String("admin_addr", cfg.AdminAddr),
		slog.Duration("interval", cfg.Interval))
	stopCtx, stopAll := context.WithCancel(ctx)
	defer stopAll()
	var serveErr, purgeErr error
	var wg sync.WaitGroup
	wg.Go(func() { mon.Run(stopCtx) })
	wg.Go(func() {
		defer stopAll()
		purgeErr = recovered(func() error { return purger.Run(stopCtx) })
	})
	wg.Go(func() {
		defer stopAll()
		errc := make(chan error, 1)
		go func() { errc <- recovered(admin.ListenAndServe) }()
		select {
		case serveErr = <-errc:
		case <-stopCtx.Done():
		}
		drainCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.ShutdownTimeout)
		defer cancel()
		shutErr := admin.Shutdown(drainCtx)
		if serveErr == nil {
			serveErr = <-errc
		}
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		serveErr = errors.Join(serveErr, shutErr)
	})
	wg.Wait()

	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.Telemetry.ShutdownTimeout)
	defer cancel()
	telErr := tel.Shutdown(flushCtx)
	if telErr != nil {
		log.WarnContext(flushCtx, "telemetry shutdown failed", slog.Any("error", telErr))
	}
	return errors.Join(serveErr, purgeErr, telErr)
}

// newPool connects lazily as the URL's role. Two connections are enough: one
// batch at a time plus the readiness ping. QueryExecModeExec keeps the
// statements unnamed, so the pool also works behind PgBouncer's transaction
// mode.
func newPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.New("NAVA_DATABASE_URL is not a valid PostgreSQL connection string")
	}
	cfg.MaxConns = 2
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("cannot create the PostgreSQL pool")
	}
	return pool, nil
}

// recovered turns a panic in fn into an error carrying the stack, so a
// panicking goroutine leads to a graceful shutdown.
func recovered(fn func() error) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("panic: %v\n%s", v, debug.Stack())
		}
	}()
	return fn()
}
