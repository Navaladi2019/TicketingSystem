package idempotency_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

var errMeter = errors.New("meter refused the instrument")

type failingMeterProvider struct{ noop.MeterProvider }

func (failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { return failingMeter{} }

type failingMeter struct{ noop.Meter }

func (failingMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errMeter
}

func TestNewMeterError(t *testing.T) {
	t.Parallel()
	pool, err := pgxpool.New(t.Context(), "postgres://nobody@127.0.0.1:1/none")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	mw, err := idempotency.New(pool, &problem.Registry{}, slog.New(slog.DiscardHandler), failingMeterProvider{})
	if !errors.Is(err, errMeter) || mw != nil {
		t.Fatalf("New = %v, %v; want nil, %v", mw, err, errMeter)
	}
}

// A write whose key cannot be stored is a 500 and never runs the handler:
// running it without a key row would let a retry run it again.
func TestUnreachableDatabaseIs500WithoutRunning(t *testing.T) {
	t.Parallel()
	h, runs := newMW(t) // its pool dials a closed port
	rec := call(h, http.MethodPost, http.Header{idempotency.Header: {"k"}}, `{"a":1}`, true)
	var b problem.Body
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("%d %s: %v", rec.Code, rec.Body, err)
	}
	if rec.Code != http.StatusInternalServerError || b.Code != problem.InternalError || *runs != 0 {
		t.Errorf("= %d %s, runs %d; want 500 internal_error and no handler run", rec.Code, b.Code, *runs)
	}
}
