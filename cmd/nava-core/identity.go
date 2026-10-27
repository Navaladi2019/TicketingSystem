package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// newResolver builds the tenant resolver behind every /api/v1 operation: the
// Keycloak adapter for NAVA_OIDC_ISSUERS, reading bindings and principals as
// nava_app through NAVA_DATABASE_URL. With either unset it returns a
// resolver that refuses every token, so the API fails closed. The returned
// func closes the adapter and the pool.
func newResolver(ctx context.Context, log *slog.Logger, tel *telemetry.Telemetry, cfg config.Config) (identity.IdentityResolver, identity.GroupPathReader, func(), error) {
	if len(cfg.Deps.OIDCIssuers) == 0 || cfg.Deps.DatabaseURL == "" {
		log.WarnContext(ctx, "tenant resolution not configured: every API request is refused",
			slog.Bool("issuers_set", len(cfg.Deps.OIDCIssuers) > 0), slog.Bool("database_set", cfg.Deps.DatabaseURL != ""))
		return refuseAll{}, refuseAll{}, func() {}, nil
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.Deps.DatabaseURL)
	if err != nil {
		// The error quotes the connection string, which is a credential.
		return nil, nil, nil, errors.New("config: NAVA_DATABASE_URL is not a valid PostgreSQL connection string")
	}
	// Unnamed statements, so the pool works behind PgBouncer's transaction
	// mode.
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	poolCfg.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, err := db.NewPool(ctx, poolCfg, tel)
	if err != nil {
		return nil, nil, nil, err
	}
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		pool.Close()
		return nil, nil, nil, err
	}
	store := pgstore.New(pool)
	a, err := keycloak.New(ctx, keycloak.Config{
		Issuers:       cfg.Deps.OIDCIssuers,
		Logger:        log,
		MeterProvider: tel.MeterProvider,
		Store:         store,
	}, out)
	if err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("identity: %w", err)
	}
	return a, store, func() { _ = a.Close(context.WithoutCancel(ctx)); pool.Close() }, nil
}

// refuseAll is the resolver when tenant resolution is not configured.
type refuseAll struct{}

func (refuseAll) Resolve(context.Context, string) (identity.Principal, error) {
	return identity.Principal{}, fmt.Errorf("%w: tenant resolution not configured", identity.ErrInvalidToken)
}

// GroupPaths is never reached: the resolver refuses every token first.
func (refuseAll) GroupPaths(context.Context, uuid.UUID, uuid.UUID) ([]identity.GroupPath, error) {
	return nil, errors.New("tenant resolution not configured")
}

// AC8 red step: a disallowed GroupPaths call, removed in the next commit.
func redStep(ctx context.Context, r identity.GroupPathReader) {
	_, _ = r.GroupPaths(ctx, uuid.Nil, uuid.Nil)
}
