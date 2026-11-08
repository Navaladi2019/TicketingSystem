package db_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/multitracer"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
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

// TestNewPoolClosesFailedParses: the pool's tracer keeps otelpgx on every
// hook and adds ParseFailureCloser as a prepare tracer (TICK-550).
func TestNewPoolClosesFailedParses(t *testing.T) {
	t.Parallel()
	tel := &telemetry.Telemetry{TracerProvider: tracenoop.NewTracerProvider(), MeterProvider: noop.NewMeterProvider()}
	pool, err := db.NewPool(t.Context(), config(t), tel)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	mt, ok := pool.Config().ConnConfig.Tracer.(*multitracer.Tracer)
	if !ok {
		t.Fatalf("tracer is %T, want *multitracer.Tracer", pool.Config().ConnConfig.Tracer)
	}
	if !slices.ContainsFunc(mt.PrepareTracers, func(p pgx.PrepareTracer) bool { _, ok := p.(db.ParseFailureCloser); return ok }) {
		t.Error("no ParseFailureCloser among the prepare tracers")
	}
	if !slices.ContainsFunc(mt.QueryTracers, func(q pgx.QueryTracer) bool { _, ok := q.(*otelpgx.Tracer); return ok }) {
		t.Error("no otelpgx tracer among the query tracers")
	}
	if len(mt.PoolAcquireTracers) != 1 {
		t.Errorf("%d pool acquire tracers, want 1 (otelpgx)", len(mt.PoolAcquireTracers))
	}
}

// errMeterProvider's meters refuse every observable counter, as an SDK
// with a broken view or exporter can.
type errMeterProvider struct{ noop.MeterProvider }

func (errMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { return errMeter{} }

type errMeter struct{ noop.Meter }

func (errMeter) Int64ObservableCounter(string, ...metric.Int64ObservableCounterOption) (metric.Int64ObservableCounter, error) {
	return nil, errors.New("instrument refused")
}

// TestNewPoolErrorsQuoteNoCredential: neither failure echoes the connection
// string, which carries the password.
func TestNewPoolErrorsQuoteNoCredential(t *testing.T) {
	t.Parallel()
	const secret = "s3cret-in-the-dsn"
	tests := map[string]struct {
		mp   metric.MeterProvider
		edit func(*pgxpool.Config)
		want string
	}{
		"invalid pool config": {noop.NewMeterProvider(), func(c *pgxpool.Config) { c.MaxConns = 0 }, "db: cannot create the PostgreSQL pool"},
		"metrics refused":     {errMeterProvider{}, func(*pgxpool.Config) {}, "db: cannot register the pool metrics"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg, err := pgxpool.ParseConfig("postgres://nava:" + secret + "@127.0.0.1:1/nava?connect_timeout=1")
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(cfg)
			tel := &telemetry.Telemetry{TracerProvider: tracenoop.NewTracerProvider(), MeterProvider: tc.mp}
			pool, err := db.NewPool(t.Context(), cfg, tel)
			if pool != nil {
				pool.Close()
				t.Fatal("got a pool, want nil")
			}
			if err == nil || err.Error() != tc.want {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("error quotes the password: %v", err)
			}
		})
	}
}
