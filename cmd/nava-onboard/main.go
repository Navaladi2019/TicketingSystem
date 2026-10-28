// Command nava-onboard serves the operator API (POST /api/v1/operator/tenants,
// ADR-001 A.3, PLAN-001 T21) in its own process, with its own listener. It is
// the only process that reads the onboarding service account's secret and the
// nava_onboard database URL: nava-core, which parses every tenant's input,
// never does, and answers the operator routes 501.
//
// The admin listener serves /metrics, /healthz and /readyz.
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

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.LookupEnv, os.Stdout)
	stop()
	if err != nil {
		slog.ErrorContext(ctx, "nava-onboard stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, lookup config.LookupFunc, stdout io.Writer) error {
	cfg, err := config.LoadOnboard(lookup)
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
	flush := func() error {
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.Telemetry.ShutdownTimeout)
		defer cancel()
		return tel.Shutdown(flushCtx)
	}

	handler, appPool, closeAll, err := build(ctx, log, tel, cfg)
	if err != nil {
		_ = flush()
		return err
	}
	defer closeAll()

	mon, err := health.NewMonitor(log, tel.MeterProvider,
		[]health.Check{{Name: "postgres", Critical: true, Probe: appPool.Ping}}, health.Options{})
	if err != nil {
		_ = flush()
		return err
	}
	listeners := []struct {
		name string
		srv  *http.Server
	}{
		{"operator", server.New(cfg.Addr, cfg.HTTP, handler, log)},
		{"admin", telemetry.NewAdminServer(cfg.AdminAddr, cfg.HTTP, telemetry.NewAdminHandler(log, tel, mon.Routes), log)},
	}
	log.InfoContext(ctx, "nava-onboard listening", slog.String("addr", cfg.Addr), slog.String("admin_addr", cfg.AdminAddr))

	stopCtx, stopAll := context.WithCancel(ctx)
	defer stopAll()
	errs := make([]error, len(listeners))
	var wg sync.WaitGroup
	wg.Go(func() { mon.Run(stopCtx) })
	for i, l := range listeners {
		wg.Go(func() {
			defer stopAll()
			errs[i] = serve(stopCtx, l.srv, cfg.HTTP.ShutdownTimeout)
		})
	}
	wg.Wait()
	telErr := flush()
	if telErr != nil {
		log.WarnContext(ctx, "telemetry shutdown failed", slog.Any("error", telErr))
	}
	return errors.Join(append(errs, telErr)...)
}

// build wires the onboarding service: a nava_app pool (tenant-token refusal,
// the invite's registry), a nava_onboard pool, and the Keycloak adapters.
func build(ctx context.Context, log *slog.Logger, tel *telemetry.Telemetry, cfg config.Onboard) (http.Handler, *pgxpool.Pool, func(), error) {
	appPool, err := newPool(ctx, tel, cfg.DatabaseURL, "NAVA_DATABASE_URL", 4)
	if err != nil {
		return nil, nil, nil, err
	}
	onboardPool, err := newPool(ctx, tel, cfg.OnboardDatabaseURL, "NAVA_ONBOARD_DATABASE_URL", 8)
	if err != nil {
		appPool.Close()
		return nil, nil, nil, err
	}
	var adapter *keycloak.Adapter
	closeAll := func() {
		if adapter != nil {
			_ = adapter.Close(context.Background())
		}
		onboardPool.Close()
		appPool.Close()
	}
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	ops := make([]keycloak.OperatorSubject, 0, len(cfg.Operators))
	for _, o := range cfg.Operators {
		ops = append(ops, keycloak.OperatorSubject{Issuer: o.Issuer, Subject: o.Subject})
	}
	adapter, err = keycloak.New(ctx, keycloak.Config{
		Issuers:       cfg.OIDCIssuers,
		FetchBaseURL:  cfg.OIDCFetchBaseURL,
		Operators:     ops,
		Logger:        log,
		MeterProvider: tel.MeterProvider,
		Store:         pgstore.New(appPool),
	}, out)
	if err != nil {
		closeAll()
		return nil, nil, nil, fmt.Errorf("identity: %w", err)
	}
	registry := onboard.NewRegistry(appPool)
	store, err := onboard.NewPGStore(onboardPool, registry)
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	specs := onboard.NewSpecs()
	dir, err := keycloak.NewDirectory(keycloak.DirectoryConfig{
		BaseURL:    cfg.KeycloakURL,
		Realm:      cfg.KeycloakRealm,
		Runtime:    keycloak.ClientCredentials{ClientID: cfg.RuntimeClientID, ClientSecret: cfg.RuntimeClientSecret},
		Onboarding: &keycloak.ClientCredentials{ClientID: cfg.OnboardClientID, ClientSecret: cfg.OnboardClientSecret},
		Registry:   registry,
		IdPs:       specs.Source,
		Invites:    store,
		InviteTTL:  cfg.InviteTTL,
	}, out)
	if err != nil {
		closeAll()
		return nil, nil, nil, fmt.Errorf("directory: %w", err)
	}
	svc, err := onboard.New(onboard.Options{
		Store: store, Directory: dir, Finder: dir, Domains: registry, Specs: specs,
		Resolver: onboard.NewNetResolver(), Issuer: cfg.Issuer, Log: log,
	})
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	h, err := server.NewHandler(log, tel, adapter, nil, server.WithOperator(adapter, svc), server.OperatorOnly())
	if err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	return h, appPool, closeAll, nil
}

// newPool connects lazily as the URL's role, with unnamed statements so it
// works behind PgBouncer's transaction mode. The error never carries the URL:
// it holds a password.
func newPool(ctx context.Context, tel *telemetry.Telemetry, dsn, name string, maxConns int32) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid PostgreSQL connection string", name)
	}
	pc.MaxConns = maxConns
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	return db.NewPool(ctx, pc, tel)
}

// serve runs srv until it fails or ctx is done, then drains it.
func serve(ctx context.Context, srv *http.Server, drain time.Duration) error {
	errc := make(chan error, 1)
	go func() { errc <- recovered(srv.ListenAndServe) }()
	var serveErr error
	served := false
	select {
	case serveErr = <-errc:
		served = true
	case <-ctx.Done():
	}
	drainCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), drain)
	defer cancel()
	shutErr := srv.Shutdown(drainCtx)
	if !served {
		serveErr = <-errc
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, shutErr)
}

// recovered turns a panic in fn into an error carrying the stack.
func recovered(fn func() error) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("panic: %v\n%s", v, debug.Stack())
		}
	}()
	return fn()
}
