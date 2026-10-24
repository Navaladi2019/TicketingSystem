package db_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

func config(t *testing.T) *pgxpool.Config {
	t.Helper()
	// Never dialled: NewPool does not connect.
	cfg, err := pgxpool.ParseConfig("postgres://nava@127.0.0.1:1/nava?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// TestNewPoolRejectsNilProviders: without a provider otelpgx would use the
// global one, whose spans skip the export allow-list.
func TestNewPoolRejectsNilProviders(t *testing.T) {
	t.Parallel()
	tp, mp := tracenoop.NewTracerProvider(), noop.NewMeterProvider()
	tests := map[string]*telemetry.Telemetry{
		"nil telemetry":       nil,
		"nil tracer provider": {MeterProvider: mp},
		"nil meter provider":  {TracerProvider: tp},
		"both nil":            {},
	}
	for name, tel := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pool, err := db.NewPool(t.Context(), config(t), tel)
			if !errors.Is(err, db.ErrNilProvider) {
				t.Errorf("err = %v, want ErrNilProvider", err)
			}
			if pool != nil {
				pool.Close()
				t.Error("got a pool, want nil")
			}
		})
	}
}

func TestNewPoolRejectsNilConfig(t *testing.T) {
	t.Parallel()
	tel := &telemetry.Telemetry{TracerProvider: tracenoop.NewTracerProvider(), MeterProvider: noop.NewMeterProvider()}
	if _, err := db.NewPool(t.Context(), nil, tel); err == nil {
		t.Error("NewPool(nil config) succeeded")
	}
}

// TestNewPoolTracesACopy: the pool gets the tracer; the caller's config is
// left as it was.
func TestNewPoolTracesACopy(t *testing.T) {
	t.Parallel()
	tel := &telemetry.Telemetry{TracerProvider: tracenoop.NewTracerProvider(), MeterProvider: noop.NewMeterProvider()}
	cfg := config(t)
	pool, err := db.NewPool(t.Context(), cfg, tel)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if pool.Config().ConnConfig.Tracer == nil {
		t.Error("the pool has no tracer")
	}
	if cfg.ConnConfig.Tracer != nil {
		t.Error("NewPool changed the caller's config")
	}
}
