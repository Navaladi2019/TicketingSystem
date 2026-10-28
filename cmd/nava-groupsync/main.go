// Command nava-groupsync mirrors each tenant's users, groups, memberships and
// org units from Keycloak into the identity tables every few minutes,
// connected as nava_groupsync (ADR-001 A.3, PLAN-001 T25). It serves
// /metrics, /healthz and /readyz on the admin listener.
//
// With --once it runs one cycle and exits, non-zero if any tenant failed
// (T19 and T26 use it).
package main

import (
	"context"
	"errors"
	"flag"
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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

func main() {
	once := flag.Bool("once", false, "run one sync cycle and exit")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.LookupEnv, os.Stdout, *once)
	stop()
	if err != nil {
		slog.ErrorContext(ctx, "nava-groupsync stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

// run syncs until ctx is cancelled or the admin listener fails (or, with
// once, for one cycle), then drains the listener and flushes telemetry.
func run(ctx context.Context, lookup config.LookupFunc, stdout io.Writer, once bool) error {
	cfg, err := config.LoadGroupsync(lookup)
	if err != nil {
		return err
	}
	log := logging.New(stdout, cfg.Log)
	telemetry.RouteErrors(log)

	tel, err := telemetry.New(ctx, cfg.Telemetry)
	if err != nil {
		return err
	}
	log = logging.WithExport(log, tel.LogHandler("nava-groupsync"), cfg.Log.Level)
	flush := func() error {
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.Telemetry.ShutdownTimeout)
		defer cancel()
		return tel.Shutdown(flushCtx)
	}

	pool, err := newPool(ctx, cfg.DatabaseURL)
	if err != nil {
		_ = flush()
		return err
	}
	defer pool.Close()
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		_ = flush()
		return err
	}
	dir, err := keycloak.NewDirectory(keycloak.DirectoryConfig{
		BaseURL:  cfg.KeycloakURL,
		Realm:    cfg.KeycloakRealm,
		Runtime:  keycloak.ClientCredentials{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret},
		Registry: noRegistry{},
	}, out)
	if err != nil {
		_ = flush()
		return err
	}
	syncer, err := groupsync.New(pool, dir, tel.MeterProvider, log, groupsync.Options{Interval: cfg.Interval})
	if err != nil {
		_ = flush()
		return err
	}

	if once {
		return runOnce(ctx, log, syncer, flush)
	}

	mon, err := health.NewMonitor(log, tel.MeterProvider,
		[]health.Check{{Name: "postgres", Critical: true, Probe: pool.Ping}}, health.Options{})
	if err != nil {
		_ = flush()
		return err
	}
	admin := telemetry.NewAdminServer(cfg.AdminAddr, cfg.HTTP, telemetry.NewAdminHandler(log, tel, mon.Routes), log)

	log.InfoContext(ctx, "nava-groupsync running", slog.String("admin_addr", cfg.AdminAddr),
		slog.Duration("interval", cfg.Interval))
	stopCtx, stopAll := context.WithCancel(ctx)
	defer stopAll()
	var serveErr, syncErr error
	var wg sync.WaitGroup
	wg.Go(func() { mon.Run(stopCtx) })
	wg.Go(func() {
		defer stopAll()
		syncErr = recovered(func() error { return syncer.Run(stopCtx) })
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

	telErr := flush()
	if telErr != nil {
		log.WarnContext(ctx, "telemetry shutdown failed", slog.Any("error", telErr))
	}
	return errors.Join(serveErr, syncErr, telErr)
}

// runOnce runs one cycle and returns an error if the cycle or any tenant
// failed. Each failure is already logged with its tenant id.
func runOnce(ctx context.Context, log *slog.Logger, syncer *groupsync.Syncer, flush func() error) error {
	results, err := syncer.Once(ctx)
	failed := 0
	for _, r := range results {
		if r.Err != nil {
			failed++
		}
	}
	log.InfoContext(ctx, "nava-groupsync --once done", slog.Int("tenants", len(results)), slog.Int("failed", failed))
	if telErr := flush(); telErr != nil {
		log.WarnContext(ctx, "telemetry shutdown failed", slog.Any("error", telErr))
	}
	if err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d tenants failed to sync", failed, len(results))
	}
	return nil
}

// newPool connects lazily as the URL's role. One connection syncs and one
// serves the readiness ping. QueryExecModeExec keeps the statements unnamed,
// so the pool also works behind PgBouncer's transaction mode.
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

// noRegistry is the keycloak.Registry of a process that only reads the
// directory: group sync calls ChangesSince, which asks the registry nothing.
// Every method refuses, so a later change that makes sync call one fails
// loudly instead of answering from nothing.
type noRegistry struct{}

var errNoRegistry = errors.New("nava-groupsync: the registry is not available")

var _ keycloak.Registry = noRegistry{}

func (noRegistry) UserKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", errNoRegistry
}

func (noRegistry) GroupKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", errNoRegistry
}

func (noRegistry) BoundIdPAliases(context.Context, uuid.UUID) ([]string, error) {
	return nil, errNoRegistry
}

func (noRegistry) TenantOwnsEmailDomain(context.Context, uuid.UUID, string) (bool, error) {
	return false, errNoRegistry
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
