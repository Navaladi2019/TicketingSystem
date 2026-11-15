package tenancy

// TICK-909: the refusal paths of a BeginInFirstBatch transaction that need
// no server. Each lazyTx here has a nil connection, so anything that tried to
// send would panic: a passing test is also proof that nothing was sent.

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// scoped wraps lazy as InTenantTx does.
func scoped(lazy *lazyTx) *scopedTx {
	return &scopedTx{Tx: lazy, tenant: lazy.tenant, memo: &memo{m: map[any]any{}}}
}

// assertClosed checks that every way of sending on tx refuses with
// pgx.ErrTxClosed, before anything reaches the (nil) connection.
func assertClosed(t *testing.T, tx pgx.Tx) {
	t.Helper()
	ctx := t.Context()
	check := func(what string, err error) {
		t.Helper()
		if !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("%s after the transaction ended: err = %v, want pgx.ErrTxClosed", what, err)
		}
	}
	_, err := tx.Exec(ctx, "SELECT 1")
	check("Exec", err)
	rows, err := tx.Query(ctx, "SELECT 1")
	check("Query", err)
	if rows != nil {
		t.Error("Query after the transaction ended returned rows")
	}
	check("QueryRow", tx.QueryRow(ctx, "SELECT 1").Scan(new(int)))
	sd, err := tx.Prepare(ctx, "s", "SELECT 1")
	check("Prepare", err)
	if sd != nil {
		t.Error("Prepare after the transaction ended returned a statement")
	}
	n, err := tx.CopyFrom(ctx, pgx.Identifier{"t"}, []string{"a"}, pgx.CopyFromRows([][]any{{1}}))
	check("CopyFrom", err)
	if n != 0 {
		t.Errorf("CopyFrom after the transaction ended copied %d rows", n)
	}
	sp, err := tx.Begin(ctx)
	check("Begin", err)
	if sp != nil {
		t.Error("Begin after the transaction ended returned a savepoint")
	}

	b := &pgx.Batch{}
	b.Queue("SELECT 1")
	br := tx.SendBatch(ctx, b)
	_, err = br.Exec()
	check("SendBatch: Exec", err)
	_, err = br.Query()
	check("SendBatch: Query", err)
	check("SendBatch: QueryRow", br.QueryRow().Scan(new(int)))
	check("SendBatch: Close", br.Close())

	check("Commit", tx.Commit(ctx))
	check("Rollback", tx.Rollback(ctx))
}

// TestFirstBatchNothingSentEndsWithoutARoundTrip: a BeginInFirstBatch
// transaction that sent nothing commits or rolls back without sending
// COMMIT or ROLLBACK, and is then closed to everything: no statement can
// open a transaction (and so run without a tenant) after it ended.
func TestFirstBatchNothingSentEndsWithoutARoundTrip(t *testing.T) {
	for name, end := range map[string]func(pgx.Tx) error{
		"commit":   func(tx pgx.Tx) error { return tx.Commit(t.Context()) },
		"rollback": func(tx pgx.Tx) error { return tx.Rollback(t.Context()) },
	} {
		t.Run(name, func(t *testing.T) {
			lazy := &lazyTx{tenant: uuid.New()}
			tx := scoped(lazy)
			if err := end(tx); err != nil {
				t.Fatalf("%s of a transaction that sent nothing: %v", name, err)
			}
			if lazy.begun || !lazy.closed {
				t.Fatalf("after %s: begun %v, closed %v; want not begun, closed", name, lazy.begun, lazy.closed)
			}
			assertClosed(t, tx)
			if lazy.begun {
				t.Error("a refused call marked the transaction begun")
			}
			if tx.memo.open != 0 {
				t.Errorf("a refused Begin opened a memo scope (open = %d)", tx.memo.open)
			}
			if tx.Conn() != nil {
				t.Error("Conn is not the transaction's connection")
			}
		})
	}
}

// TestFirstBatchClosedSavepointRefuses: a savepoint released or rolled back
// refuses Begin, Commit and Rollback; an open savepoint of a transaction that
// already ended cannot open another.
func TestFirstBatchClosedSavepointRefuses(t *testing.T) {
	ctx := t.Context()
	sp := &lazySavepoint{lazyTx: &lazyTx{tenant: uuid.New(), begun: true}, n: 1, closed: true}
	if tx, err := sp.Begin(ctx); !errors.Is(err, pgx.ErrTxClosed) || tx != nil {
		t.Errorf("Begin on a closed savepoint = %v, %v; want nil, pgx.ErrTxClosed", tx, err)
	}
	if err := sp.Commit(ctx); !errors.Is(err, pgx.ErrTxClosed) {
		t.Errorf("Commit on a closed savepoint = %v, want pgx.ErrTxClosed", err)
	}
	if err := sp.Rollback(ctx); !errors.Is(err, pgx.ErrTxClosed) {
		t.Errorf("Rollback on a closed savepoint = %v, want pgx.ErrTxClosed", err)
	}

	parent := &lazyTx{tenant: uuid.New(), begun: true, closed: true, sp: 1}
	open := &lazySavepoint{lazyTx: parent, n: 1}
	if tx, err := open.Begin(ctx); !errors.Is(err, pgx.ErrTxClosed) || tx != nil {
		t.Errorf("Begin on a savepoint of an ended transaction = %v, %v; want nil, pgx.ErrTxClosed", tx, err)
	}
	if parent.sp != 1 {
		t.Errorf("a refused Begin numbered a savepoint (sp = %d)", parent.sp)
	}
}

// fakeResults is the pgx.BatchResults of a first batch: execErrs answers
// the leading Exec calls in order (the BEGIN, then the tenant), and calls
// records what the caller's side read.
type fakeResults struct {
	execErrs []error
	calls    []string
}

func (f *fakeResults) Exec() (pgconn.CommandTag, error) {
	f.calls = append(f.calls, "exec")
	if len(f.execErrs) > 0 {
		err := f.execErrs[0]
		f.execErrs = f.execErrs[1:]
		return pgconn.CommandTag{}, err
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (f *fakeResults) Query() (pgx.Rows, error) {
	f.calls = append(f.calls, "query")
	return nil, nil
}

func (f *fakeResults) QueryRow() pgx.Row {
	f.calls = append(f.calls, "queryrow")
	return errRow{nil}
}

func (f *fakeResults) Close() error { return nil }

// TestFirstBatchHeaderFailureHidesCallerResults: when the first batch's
// BEGIN or set_config fails, the caller's first read returns that error and
// never reaches the caller's own statement's result, which PostgreSQL did
// not run under a tenant; when both succeed the caller reads its own result
// and never the BEGIN's or the tenant's.
func TestFirstBatchHeaderFailureHidesCallerResults(t *testing.T) {
	failed := errors.New("header statement failed")
	read := map[string]func(*firstResults) error{
		"Exec": func(r *firstResults) error { _, err := r.Exec(); return err },
		"Query": func(r *firstResults) error {
			rows, err := r.Query()
			if err != nil && rows != nil {
				return errors.New("Query returned rows with its error")
			}
			return err
		},
		"QueryRow": func(r *firstResults) error { return r.QueryRow().Scan() },
	}
	for name, fn := range read {
		for _, tc := range []struct {
			header    string
			execErrs  []error
			wantCalls []string
		}{
			{"BEGIN fails", []error{failed}, []string{"exec"}},
			{"tenant fails", []error{nil, failed}, []string{"exec", "exec"}},
		} {
			t.Run(name+"/"+tc.header, func(t *testing.T) {
				f := &fakeResults{execErrs: tc.execErrs}
				r := &firstResults{BatchResults: f, pending: 2}
				if err := fn(r); !errors.Is(err, failed) {
					t.Errorf("err = %v, want the header's", err)
				}
				if got, want := len(f.calls), len(tc.wantCalls); got != want {
					t.Errorf("reads %v, want %v: the caller's result was read past a failed header", f.calls, tc.wantCalls)
				}
				if r.pending != 0 {
					t.Errorf("pending = %d after a failed header, want 0", r.pending)
				}
			})
		}
		t.Run(name+"/header succeeds", func(t *testing.T) {
			f := &fakeResults{}
			r := &firstResults{BatchResults: f, pending: 2}
			if err := fn(r); err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"Exec": "exec", "Query": "query", "QueryRow": "queryrow"}[name]
			if len(f.calls) != 3 || f.calls[0] != "exec" || f.calls[1] != "exec" || f.calls[2] != want {
				t.Errorf("reads %v, want the BEGIN, the tenant, then the caller's %s", f.calls, want)
			}
			// The header is read once: the next read is the caller's next result.
			if err := fn(r); err != nil {
				t.Fatal(err)
			}
			if len(f.calls) != 4 {
				t.Errorf("reads %v, want the header skipped only once", f.calls)
			}
		})
	}
}
