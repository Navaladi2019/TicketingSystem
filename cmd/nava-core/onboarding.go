package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// newOnboarding builds the operator onboarding service (T21): a nava_onboard
// pool, a nava_app pool for the invite's registry, and the Keycloak directory
// with the onboarding account. With onboarding unconfigured it returns no
// options, and the operator routes answer 501.
func newOnboarding(ctx context.Context, log *slog.Logger, tel *telemetry.Telemetry, cfg config.Config, ops identity.OperatorResolver) ([]server.Option, func(), error) {
	o := cfg.Deps.Onboarding
	if !o.Enabled() || ops == nil || cfg.Deps.DatabaseURL == "" {
		log.InfoContext(ctx, "operator onboarding not configured: the operator routes answer 501")
		return nil, func() {}, nil
	}
	appPool, err := onboardPool(ctx, tel, cfg.Deps.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	onboardDB, err := onboardPool(ctx, tel, o.DatabaseURL)
	if err != nil {
		appPool.Close()
		return nil, nil, err
	}
	closeAll := func() { onboardDB.Close(); appPool.Close() }
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	specs := onboard.NewSpecs()
	dir, err := keycloak.NewDirectory(keycloak.DirectoryConfig{
		BaseURL:    o.KeycloakURL,
		Realm:      o.KeycloakRealm,
		Runtime:    keycloak.ClientCredentials{ClientID: o.RuntimeClientID, ClientSecret: o.RuntimeClientSecret},
		Onboarding: &keycloak.ClientCredentials{ClientID: o.OnboardClientID, ClientSecret: o.OnboardClientSecret},
		Registry:   onboard.NewRegistry(appPool),
		IdPs:       specs.Source,
	}, out)
	if err != nil {
		closeAll()
		return nil, nil, fmt.Errorf("onboarding directory: %w", err)
	}
	svc, err := onboard.New(onboard.Options{
		Store: onboard.NewPGStore(onboardDB), Directory: dir, Finder: dir, Specs: specs,
		Resolver: onboard.NewNetResolver(), Issuer: o.Issuer, Log: log,
	})
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return []server.Option{server.WithOperator(ops, svc)}, closeAll, nil
}

// onboardPool opens a pool for url that works behind PgBouncer's transaction
// mode. The error never carries the URL: it holds a password.
func onboardPool(ctx context.Context, tel *telemetry.Telemetry, url string) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("config: a database URL is not a valid PostgreSQL connection string")
	}
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	return db.NewPool(ctx, pc, tel)
}
