// Package db builds the platform's traced PostgreSQL pool. Tenant-scoped work
// on the pool goes through tenancy.InTenantTx.
package db

import (
	"context"
	"errors"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// ErrNilProvider is returned when the telemetry or one of its providers is
// nil. otelpgx would fall back to the global providers, whose spans skip the
// export allow-list.
var ErrNilProvider = errors.New("db: telemetry providers must not be nil")

// NewPool returns a pool for cfg, traced and measured through tel's
// providers. It does not connect: the first query does. cfg is copied, not
// changed.
//
// The tracer records no SQL text and no query parameters: span names are
// SQL keywords (telemetry.DBSpanName), and the statement attribute is off.
// otelpgx.WithIncludeQueryParameters is banned by lint.
func NewPool(ctx context.Context, cfg *pgxpool.Config, tel *telemetry.Telemetry) (*pgxpool.Pool, error) {
	if tel == nil || tel.TracerProvider == nil || tel.MeterProvider == nil {
		return nil, ErrNilProvider
	}
	if cfg == nil {
		return nil, errors.New("db: nil pool config")
	}
	cfg = cfg.Copy()
	cfg.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTracerProvider(tel.TracerProvider),
		otelpgx.WithMeterProvider(tel.MeterProvider),
		otelpgx.WithSpanNameCtxFunc(func(ctx context.Context, stmt string) string {
			return telemetry.DBSpanName(otelpgx.SQLOperationName(ctx, stmt))
		}),
		otelpgx.WithDisableSQLStatementInAttributes(),
	)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		// The error can quote the connection string, which is a credential.
		return nil, errors.New("db: cannot create the PostgreSQL pool")
	}
	if err := otelpgx.RecordStats(pool, otelpgx.WithStatsMeterProvider(tel.MeterProvider)); err != nil {
		pool.Close()
		return nil, errors.New("db: cannot register the pool metrics")
	}
	return pool, nil
}
