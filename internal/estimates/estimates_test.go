package estimates

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// fakeTx records its statements; failTenant makes that tenant's upsert fail.
type fakeTx struct {
	pgx.Tx
	db *fakeDB
}

func (t *fakeTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.db.log = append(t.db.log, sql)
	if t.db.onExec != nil && sql == refreshSQL {
		t.db.onExec()
		return pgconn.CommandTag{}, context.Canceled
	}
	if t.db.failTenant != "" && sql == refreshSQL && args[0] == t.db.failTenant {
		return pgconn.CommandTag{}, errors.New("boom")
	}
	return pgconn.CommandTag{}, nil
}

// fakeDB is a Store.
type fakeDB struct {
	tenants    []uuid.UUID
	listErr    error
	txErr      error
	failTenant string
	onExec     func()
	lists      int
	txs        int
	log        []string
}

func (d *fakeDB) Tenants(context.Context) ([]uuid.UUID, error) {
	d.lists++
	return d.tenants, d.listErr
}

func (d *fakeDB) InTenant(_ context.Context, _ uuid.UUID, fn func(pgx.Tx) error) error {
	d.txs++
	if d.txErr != nil {
		return d.txErr
	}
	return fn(&fakeTx{db: d})
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newRefresher(t *testing.T, db Store, c *clock, interval time.Duration) (*Refresher, *strings.Builder) {
	t.Helper()
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, nil))
	r, err := New(db, noop.NewMeterProvider(), log, Options{Interval: interval, Now: c.now})
	if err != nil {
		t.Fatal(err)
	}
	return r, &buf
}

const (
	tenant1 = "0190a000-0000-7000-8000-00000000000a"
	tenant2 = "0190a000-0000-7000-8000-00000000000b"
)

// Each tenant gets one transaction with the upsert, then the zero statement.
func TestEveryTenantGetsItsOwnTransaction(t *testing.T) {
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1), uuid.MustParse(tenant2)}}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, time.Hour)
	if n := r.Cycle(context.Background()); n != 0 {
		t.Fatalf("failed = %d, want 0", n)
	}
	want := []string{refreshSQL, zeroSQL, refreshSQL, zeroSQL}
	if db.txs != 2 || strings.Join(db.log, "|") != strings.Join(want, "|") {
		t.Errorf("transactions %d, statements %q; want 2 and %q", db.txs, db.log, want)
	}
}

// A failing tenant is counted, logged by id only, and the next is refreshed.
func TestFailingTenantIsSkippedAndAgesTheGauge(t *testing.T) {
	c := &clock{t: time.Unix(1000, 0)}
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1), uuid.MustParse(tenant2)}, failTenant: uuid.MustParse(tenant1).String()}
	r, logs := newRefresher(t, db, c, time.Hour)
	c.t = c.t.Add(time.Hour)
	if n := r.Cycle(context.Background()); n != 1 {
		t.Fatalf("failed = %d, want 1", n)
	}
	if !strings.Contains(logs.String(), uuid.MustParse(tenant1).String()) || strings.Contains(logs.String(), uuid.MustParse(tenant2).String()) {
		t.Errorf("log = %s, want only the failed tenant", logs)
	}
	// tenant 1 never refreshed: its age counts from the start.
	if got := r.Age(); got != 3600 {
		t.Errorf("Age = %v, want 3600", got)
	}
	db.failTenant = ""
	c.t = c.t.Add(time.Minute)
	r.Cycle(context.Background())
	if got := r.Age(); got != 0 {
		t.Errorf("Age after a clean cycle = %v, want 0", got)
	}
}

func TestAgeBeforeAnyCycleCountsFromStart(t *testing.T) {
	c := &clock{t: time.Unix(1000, 0)}
	r, _ := newRefresher(t, &fakeDB{}, c, time.Hour)
	c.t = c.t.Add(3 * time.Hour)
	if got := r.Age(); got != 10800 {
		t.Errorf("Age = %v, want 10800", got)
	}
}

func TestListingFailureIsLoggedNotCounted(t *testing.T) {
	r, logs := newRefresher(t, &fakeDB{listErr: errors.New("denied")}, &clock{t: time.Unix(0, 0)}, time.Hour)
	if n := r.Cycle(context.Background()); n != 0 {
		t.Errorf("failed = %d, want 0", n)
	}
	if !strings.Contains(logs.String(), "cannot list tenants") {
		t.Errorf("log = %s", logs)
	}
}

func TestTransactionErrorIsAFailure(t *testing.T) {
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1)}, txErr: errors.New("commit failed")}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, time.Hour)
	if n := r.Cycle(context.Background()); n != 1 {
		t.Errorf("failed = %d, want 1", n)
	}
}

func TestCancelledContextStopsTheCycle(t *testing.T) {
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1), uuid.MustParse(tenant2)}}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if n := r.Cycle(ctx); n != 0 || db.txs != 0 {
		t.Errorf("failed %d, transactions %d; want 0, 0", n, db.txs)
	}
}

func TestRunDisabledReturnsAtOnce(t *testing.T) {
	db := &fakeDB{}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, 0)
	r.Run(context.Background())
	if db.lists != 0 {
		t.Errorf("listings = %d, want 0", db.lists)
	}
}

func TestRunCyclesAtStartAndOnTheTicker(t *testing.T) {
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1)}}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, 5*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	r.Run(ctx)
	if db.lists < 2 {
		t.Errorf("listings = %d, want at least two cycles", db.lists)
	}
}

func TestNewRegistersBothMetrics(t *testing.T) {
	r, _ := newRefresher(t, &fakeDB{}, &clock{t: time.Unix(0, 0)}, time.Hour)
	if r == nil {
		t.Fatal("nil refresher")
	}
	if _, err := New(&fakeDB{}, failingProvider{noop.NewMeterProvider(), FailuresMetric}, slog.New(slog.DiscardHandler), Options{}); err == nil {
		t.Error("counter registration error not returned")
	}
	if _, err := New(&fakeDB{}, failingProvider{noop.NewMeterProvider(), AgeMetric}, slog.New(slog.DiscardHandler), Options{}); err == nil {
		t.Error("gauge registration error not returned")
	}
}

func TestIntervalFromEnv(t *testing.T) {
	for _, c := range []struct {
		in      string
		set     bool
		want    time.Duration
		wantErr bool
	}{
		{"", false, DefaultInterval, false},
		{"", true, DefaultInterval, false},
		{"0", true, 0, false},
		{"30m", true, 30 * time.Minute, false},
		{"-1h", true, 0, true},
		{"soon", true, 0, true},
	} {
		got, err := IntervalFromEnv(func(string) (string, bool) { return c.in, c.set })
		if got != c.want || (err != nil) != c.wantErr {
			t.Errorf("%q: got %v, %v; want %v, err %v", c.in, got, err, c.want, c.wantErr)
		}
	}
}

// failingProvider's meters refuse to create the instrument named fail.
type failingProvider struct {
	metric.MeterProvider
	fail string
}

func (p failingProvider) Meter(name string, o ...metric.MeterOption) metric.Meter {
	return failingMeter{p.MeterProvider.Meter(name, o...), p.fail}
}

type failingMeter struct {
	metric.Meter
	fail string
}

func (m failingMeter) Int64Counter(name string, o ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	if name == m.fail {
		return nil, errors.New("refused")
	}
	return m.Meter.Int64Counter(name, o...)
}

func (m failingMeter) Float64ObservableGauge(name string, o ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error) {
	if name == m.fail {
		return nil, errors.New("refused")
	}
	return m.Meter.Float64ObservableGauge(name, o...)
}

// An empty stack does not age after a good cycle; a listing that keeps
// failing does.
func TestAgeWithNoTenants(t *testing.T) {
	c := &clock{t: time.Unix(1000, 0)}
	db := &fakeDB{}
	r, _ := newRefresher(t, db, c, time.Hour)
	c.t = c.t.Add(4 * time.Hour)
	r.Cycle(context.Background())
	c.t = c.t.Add(time.Minute)
	if got := r.Age(); got != 60 {
		t.Errorf("Age after a good cycle with no tenants = %v, want 60", got)
	}
	db.listErr = errors.New("denied")
	for range 3 {
		c.t = c.t.Add(time.Hour)
		r.Cycle(context.Background())
	}
	if got := r.Age(); got < 10800 {
		t.Errorf("Age with the listing failing = %v, want over 10800", got)
	}
}

// A cycle that shutdown aborts counts and logs nothing, and is not clean.
func TestAbortedTenantIsNotAFailure(t *testing.T) {
	c := &clock{t: time.Unix(1000, 0)}
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1)}}
	r, logs := newRefresher(t, db, c, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	db.onExec = cancel
	if n := r.Cycle(ctx); n != 0 {
		t.Errorf("failed = %d, want 0", n)
	}
	if strings.Contains(logs.String(), "refresh failed") {
		t.Errorf("log = %s, want no failure", logs)
	}
	if !r.cleanAt.IsZero() {
		t.Error("an aborted cycle was recorded as clean")
	}
}
