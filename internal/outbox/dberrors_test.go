package outbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// scanFunc answers one QueryRow.
type scanFunc func(dest ...any) error

func (f scanFunc) Scan(dest ...any) error { return f(dest...) }

func noRows(...any) error { return pgx.ErrNoRows }

func fail(err error) scanFunc { return func(...any) error { return err } }

func ids(id uuid.UUID) scanFunc {
	return func(dest ...any) error { *dest[0].(*uuid.UUID) = id; return nil }
}

func locked(tenant uuid.UUID, version int64) scanFunc {
	return func(dest ...any) error {
		*dest[0].(*uuid.UUID), *dest[1].(*int64) = tenant, version
		return nil
	}
}

// fakeTx answers enqueue's statements from a script: each QueryRow of a
// statement takes the next answer for it, and Exec returns execErr. Any
// other method, or a statement with no answer left, fails the test.
type fakeTx struct {
	pgx.Tx
	t       *testing.T
	answers map[string][]scanFunc
	execErr error
	execs   []string
}

func (f *fakeTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	// The record lock names the transaction's tenant (TICK-480):
	// tenancy.TenantOf reads it first. Unless a case scripts it, the
	// transaction is scoped to a tenant.
	if strings.Contains(sql, "current_setting(") && len(f.answers[sql]) == 0 {
		return scanFunc(func(dest ...any) error { *dest[0].(*string) = uuid.NewString(); return nil })
	}
	a := f.answers[sql]
	if len(a) == 0 {
		f.t.Fatalf("unexpected QueryRow: %s", sql)
	}
	f.answers[sql] = a[1:]
	return a[0]
}

func (f *fakeTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	f.execs = append(f.execs, sql)
	return pgconn.NewCommandTag("INSERT 0 1"), f.execErr
}

func signalEntry() Entry {
	e := validEntry()
	e.Kind, e.Target = "approve", TargetSignal{Kind: "ticket.approval", SignalName: "approve"}
	return e
}

// Every database failure on the enqueue path is returned wrapped with the
// step that failed, and nothing after it runs.
func TestEnqueueDatabaseErrors(t *testing.T) {
	boom := errors.New("connection reset")
	tenant, existing := uuid.New(), uuid.New()
	cases := []struct {
		name    string
		entry   Entry
		answers map[string][]scanFunc
		execErr error
		prefix  string
		execs   int
	}{
		{"dedupe lookup", validEntry(), map[string][]scanFunc{dedupeLookupSQL: {fail(boom)}}, nil, "outbox: dedupe lookup: ", 0},
		{"record lock", validEntry(), map[string][]scanFunc{
			dedupeLookupSQL: {noRows}, lockRecordSQL: {fail(boom)},
		}, nil, "outbox: lock record: ", 0},
		{"open workflow", signalEntry(), map[string][]scanFunc{
			dedupeLookupSQL: {noRows}, lockRecordSQL: {locked(tenant, 1)}, openRefSQL: {fail(boom)},
		}, nil, "outbox: find open workflow: ", 0},
		{"insert", validEntry(), map[string][]scanFunc{
			dedupeLookupSQL: {noRows}, lockRecordSQL: {locked(tenant, 1)}, insertSQL: {fail(boom)},
		}, nil, "outbox: insert: ", 0},
		{"dedupe lookup after conflict", validEntry(), map[string][]scanFunc{
			dedupeLookupSQL: {noRows, fail(boom)}, lockRecordSQL: {locked(tenant, 1)}, insertSQL: {noRows},
		}, nil, "outbox: dedupe lookup after conflict: ", 0},
		{"workflow_ref", validEntry(), map[string][]scanFunc{
			dedupeLookupSQL: {noRows}, lockRecordSQL: {locked(tenant, 1)}, insertSQL: {ids(existing)},
		}, boom, "outbox: insert workflow_ref: ", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tx := &fakeTx{t: t, answers: c.answers, execErr: c.execErr}
			id, err := EnqueueAtVersion(context.Background(), tx, c.entry, 1)
			if id != uuid.Nil || !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), c.prefix) {
				t.Fatalf("EnqueueAtVersion = %s, %v; want nil id and %q wrapping %v", id, err, c.prefix, boom)
			}
			if len(tx.execs) != c.execs {
				t.Errorf("%d Execs, want %d", len(tx.execs), c.execs)
			}
		})
	}
	// ErrNoRows from the lock is the record's absence, not a database error.
	tx := &fakeTx{t: t, answers: map[string][]scanFunc{dedupeLookupSQL: {noRows}, lockRecordSQL: {noRows}}}
	if _, err := EnqueueAtVersion(context.Background(), tx, validEntry(), 1); !errors.Is(err, ErrRecordNotFound) {
		t.Errorf("no record: %v, want ErrRecordNotFound", err)
	}
}

// Duplicate delivery of one dedupe key: a concurrent writer committed the
// key between this write's dedupe lookup and its INSERT, so the INSERT's
// ON CONFLICT DO NOTHING returns no row. The write returns the winner's id,
// writes no workflow_ref and is not an error.
func TestEnqueueConcurrentDuplicateReturnsWinner(t *testing.T) {
	winner := uuid.New()
	tx := &fakeTx{t: t, answers: map[string][]scanFunc{
		dedupeLookupSQL: {noRows, ids(winner)}, lockRecordSQL: {locked(uuid.New(), 1)}, insertSQL: {noRows},
	}}
	id, err := EnqueueAtVersion(context.Background(), tx, validEntry(), 1)
	if err != nil || id != winner {
		t.Fatalf("EnqueueAtVersion = %s, %v; want %s, nil", id, err, winner)
	}
	if len(tx.execs) != 0 {
		t.Errorf("the duplicate wrote %d workflow_ref rows, want 0", len(tx.execs))
	}
}
