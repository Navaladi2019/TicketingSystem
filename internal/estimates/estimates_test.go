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

// fakeRows yields tenants.
type fakeRows struct {
	pgx.Rows
	ids []uuid.UUID
	i   int
}

func (r *fakeRows) Next() bool { r.i++; return r.i <= len(r.ids) }
func (r *fakeRows) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = r.ids[r.i-1]
	return nil
}
func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

// fakeTx records its statements; failOn makes Exec of a statement fail.
type fakeTx struct {
	pgx.Tx
	db *fakeDB
}

func (t *fakeTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	t.db.log = append(t.db.log, sql)
	if t.db.queryErr != nil {
		return nil, t.db.queryErr
	}
	return &fakeRows{ids: t.db.tenants}, nil
}

func (t *fakeTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.db.log = append(t.db.log, sql)
	if t.db.failTenant != "" && sql == refreshSQL && args[0] == t.db.failTenant {
		return pgconn.CommandTag{}, errors.New("boom")
	}
	return pgconn.CommandTag{}, nil
}

func (t *fakeTx) Commit(context.Context) error { return t.db.commitErr }

func (t *fakeTx) Rollback(context.Context) error {
	t.db.rollbacks++
	if t.db.rollbacks == t.db.rollbackFailAt {
		return errors.New("rollback failed")
	}
	return pgx.ErrTxClosed
}

type fakeDB struct {
	tenants        []uuid.UUID
	beginErr       error
	queryErr       error
	commitErr      error
	failTenant     string
	rollbackFailAt int
	rollbacks      int
	begins         int
	log            []string
}

func (d *fakeDB) Begin(context.Context) (pgx.Tx, error) {
	d.begins++
	if d.beginErr != nil {
		return nil, d.beginErr
	}
	return &fakeTx{db: d}, nil
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newRefresher(t *testing.T, db Beginner, c *clock, interval time.Duration) (*Refresher, *strings.Builder) {
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

// Every statement but the listing runs after set_config, per transaction.
func TestEveryStatementFollowsSetConfig(t *testing.T) {
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1), uuid.MustParse(tenant2)}}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, time.Hour)
	if n := r.Cycle(context.Background()); n != 0 {
		t.Fatalf("failed = %d, want 0", n)
	}
	want := []string{listSQL, setTenantSQL, refreshSQL, zeroSQL, setTenantSQL, refreshSQL, zeroSQL}
	if strings.Join(db.log, "|") != strings.Join(want, "|") {
		t.Errorf("statements = %q, want %q", db.log, want)
	}
}

// A failing tenant is counted, logged by id only, and the next is refreshed.
func TestFailingTenantIsSkippedAndAgesTheGauge(t *testing.T) {
	c := &clock{t: time.Unix(1000, 0)}
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1), uuid.MustParse(tenant2)}, failTenant: uuid.MustParse(tenant1).String(), rollbackFailAt: 0}
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

func TestListingFailuresAreLoggedNotCounted(t *testing.T) {
	for name, db := range map[string]*fakeDB{
		"begin": {beginErr: errors.New("no conn")},
		"query": {queryErr: errors.New("denied")},
	} {
		r, logs := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, time.Hour)
		if n := r.Cycle(context.Background()); n != 0 {
			t.Errorf("%s: failed = %d, want 0", name, n)
		}
		if !strings.Contains(logs.String(), "cannot list tenants") {
			t.Errorf("%s: log = %s", name, logs)
		}
	}
}

func TestRefreshBeginCommitAndRollbackErrors(t *testing.T) {
	// begin fails for the tenant transaction only: the listing is begin #1.
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1)}, commitErr: errors.New("commit failed"), rollbackFailAt: 2}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, time.Hour)
	if n := r.Cycle(context.Background()); n != 1 {
		t.Errorf("commit failure: failed = %d, want 1", n)
	}
	db = &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1)}}
	r, _ = newRefresher(t, db, &clock{t: time.Unix(0, 0)}, time.Hour)
	if err := r.refresh(context.Background(), uuid.MustParse(tenant1)); err != nil {
		t.Fatal(err)
	}
	db.beginErr = errors.New("no conn")
	if err := r.refresh(context.Background(), uuid.MustParse(tenant1)); err == nil {
		t.Error("refresh with no connection succeeded")
	}
}

func TestCancelledContextStopsTheCycle(t *testing.T) {
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1), uuid.MustParse(tenant2)}}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if n := r.Cycle(ctx); n != 0 || db.begins != 1 {
		t.Errorf("failed %d, begins %d; want 0 and only the listing", n, db.begins)
	}
}

func TestRunDisabledReturnsAtOnce(t *testing.T) {
	db := &fakeDB{}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, 0)
	r.Run(context.Background())
	if db.begins != 0 {
		t.Errorf("begins = %d, want 0", db.begins)
	}
}

func TestRunCyclesAtStartAndOnTheTicker(t *testing.T) {
	db := &fakeDB{tenants: []uuid.UUID{uuid.MustParse(tenant1)}}
	r, _ := newRefresher(t, db, &clock{t: time.Unix(0, 0)}, 5*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	r.Run(ctx)
	if db.begins < 4 {
		t.Errorf("begins = %d, want at least two cycles", db.begins)
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
