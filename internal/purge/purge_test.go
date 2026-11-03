package purge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

// The statements and the nava_purge policies run against real PostgreSQL in
// internal/db/dbtest (TestT13Purge*, including the two-tenant and isolation
// cases); these cover the batch loop and its failure handling on a scripted
// database.

var errBoom = errors.New("boom")

type rowFunc func(dest ...any) error

func (f rowFunc) Scan(dest ...any) error { return f(dest...) }

// script is how every batch transaction of batchDB answers.
type script struct {
	beginErr  error
	lockErr   error
	locked    bool
	deleteErr error
	commitErr error
	// deleted is the rows each successive DELETE reports; past its end, 0.
	deleted []int64
}

type batchDB struct {
	mu      sync.Mutex
	s       script
	begins  int
	deletes int
	args    [][]any
	onBegin func(n int)
}

func (d *batchDB) Begin(context.Context) (pgx.Tx, error) {
	d.mu.Lock()
	d.begins++
	n := d.begins
	d.mu.Unlock()
	if d.onBegin != nil {
		d.onBegin(n)
	}
	if d.s.beginErr != nil {
		return nil, d.s.beginErr
	}
	return &batchTx{db: d}, nil
}

type batchTx struct {
	pgx.Tx // nil: any method not overridden panics, which fails the test
	db     *batchDB
}

func (x *batchTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	return rowFunc(func(dest ...any) error {
		if !strings.Contains(sql, "pg_try_advisory_xact_lock") {
			return fmt.Errorf("unexpected statement %q", sql)
		}
		if x.db.s.lockErr != nil {
			return x.db.s.lockErr
		}
		*dest[0].(*bool) = x.db.s.locked
		return nil
	})
}

func (x *batchTx) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	d := x.db
	d.mu.Lock()
	defer d.mu.Unlock()
	d.args = append(d.args, args)
	if d.s.deleteErr != nil {
		return pgconn.CommandTag{}, d.s.deleteErr
	}
	var n int64
	if d.deletes < len(d.s.deleted) {
		n = d.s.deleted[d.deletes]
	}
	d.deletes++
	return pgconn.NewCommandTag(fmt.Sprintf("DELETE %d", n)), nil
}

func (x *batchTx) Commit(context.Context) error   { return x.db.s.commitErr }
func (x *batchTx) Rollback(context.Context) error { return nil }

func newTestPurger(t *testing.T, db Beginner, opts Options) (*Purger, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	p, err := New(db, metricnoop.NewMeterProvider(), slog.New(slog.NewTextHandler(&logs, nil)), opts)
	if err != nil {
		t.Fatal(err)
	}
	return p, &logs
}

// A failure at any step of a batch is that table's Result.Err, wrapped
// naming the step, logged as a failure; the other tables still run.
func TestOnceReportsEachFailingStep(t *testing.T) {
	cases := []struct {
		name string
		s    script
		want string
	}{
		{"begin", script{beginErr: errBoom}, "begin: boom"},
		{"advisory lock", script{lockErr: errBoom}, "advisory lock: boom"},
		{"delete", script{locked: true, deleteErr: errBoom}, "delete: boom"},
		{"commit", script{locked: true, commitErr: errBoom, deleted: []int64{3, 3, 3}}, "commit: boom"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := &batchDB{s: c.s}
			p, logs := newTestPurger(t, db, Options{})
			rs := p.Once(t.Context())
			if len(rs) != 3 {
				t.Fatalf("%d results, want one per table", len(rs))
			}
			for i, table := range []string{TableIdempotencyKeys, TableOutbox, TableWorkflowRef} {
				r := rs[i]
				if r.Table != table || !errors.Is(r.Err, errBoom) || r.Err.Error() != c.want || r.Deleted != 0 || r.Skipped {
					t.Errorf("result %d = %+v, want %s failed with %q", i, r, table, c.want)
				}
			}
			if got := strings.Count(logs.String(), `msg="purge failed"`); got != 3 {
				t.Errorf("%d failure logs, want 3:\n%s", got, logs)
			}
			if db.begins != 3 {
				t.Errorf("%d transactions, want one per table (a failure ends the table)", db.begins)
			}
		})
	}
}

// Full batches run again until one comes back short; the total is the sum.
func TestOnceBatchesUntilShort(t *testing.T) {
	db := &batchDB{s: script{locked: true, deleted: []int64{2, 2, 1}}}
	p, logs := newTestPurger(t, db, Options{BatchSize: 2})
	rs := p.Once(t.Context())
	if rs[0].Deleted != 5 || rs[0].Err != nil || rs[1].Deleted != 0 || rs[2].Deleted != 0 {
		t.Fatalf("results = %+v, want 5 from idempotency_keys and 0 from the rest", rs)
	}
	if db.deletes != 5 {
		t.Errorf("%d DELETEs, want 3 for the first table and 1 for each other", db.deletes)
	}
	// Each statement got the batch size and the windows, nothing else: no
	// tenant or other caller-chosen row selector reaches it.
	w := DefaultWindows()
	want := [][]any{
		{2}, {2}, {2},
		{2, w.OutboxDispatched.Seconds(), w.OutboxDead.Seconds()},
		{2, w.WorkflowRef.Seconds()},
	}
	if fmt.Sprint(db.args) != fmt.Sprint(want) {
		t.Errorf("arguments %v, want %v", db.args, want)
	}
	if !strings.Contains(logs.String(), `msg=purged table=idempotency_keys deleted=5`) {
		t.Errorf("no purged log for idempotency_keys:\n%s", logs)
	}
}

// A cancelled context ends a table with the context's error and no batch.
func TestOnceWithCancelledContext(t *testing.T) {
	db := &batchDB{s: script{locked: true}}
	p, _ := newTestPurger(t, db, Options{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, r := range p.Once(ctx) {
		if !errors.Is(r.Err, context.Canceled) || r.Deleted != 0 {
			t.Errorf("%s: %+v, want context.Canceled", r.Table, r)
		}
	}
	if db.begins != 0 {
		t.Errorf("%d transactions after cancel, want none", db.begins)
	}
}

// Run purges at once, again every Interval, and returns nil when ctx ends.
func TestRunRepeatsUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	db := &batchDB{s: script{locked: true}}
	// Three tables per run: the seventh transaction is the third run's.
	db.onBegin = func(n int) {
		if n == 7 {
			cancel()
		}
	}
	p, _ := newTestPurger(t, db, Options{Interval: time.Millisecond})
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run = %v, want nil on cancel", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	if db.begins < 7 {
		t.Errorf("%d transactions, want at least three runs", db.begins)
	}
}

// A meter that refuses the counter fails New.
func TestNewCounterFailure(t *testing.T) {
	_, err := New(&batchDB{}, errMeterProvider{}, slog.New(slog.DiscardHandler), Options{})
	if !errors.Is(err, errBoom) || err.Error() != "purge: boom" {
		t.Fatalf("err = %v, want purge: boom", err)
	}
}

// New's defaults: out-of-range batch sizes take MaxBatch, and zero
// windows and interval take the ADR-001 values.
func TestNewDefaults(t *testing.T) {
	for _, b := range []int{0, -1, MaxBatch + 1} {
		p, _ := newTestPurger(t, &batchDB{}, Options{BatchSize: b})
		if p.opts.BatchSize != MaxBatch || p.opts.Interval != time.Hour || p.opts.Windows != DefaultWindows() {
			t.Errorf("BatchSize %d: options %+v", b, p.opts)
		}
	}
}

type errMeterProvider struct{ metricnoop.MeterProvider }

func (errMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { return errMeter{} }

type errMeter struct{ metricnoop.Meter }

func (errMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errBoom
}
