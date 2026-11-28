package store

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// fakeRows is a pgx.Rows whose Next, Scan and Err are scripted.
type fakeRows struct {
	pgx.Rows
	rows    int
	scanErr error
	err     error
	closed  bool
}

func (f *fakeRows) Next() bool {
	f.rows--
	return f.rows >= 0
}
func (f *fakeRows) Scan(...any) error { return f.scanErr }
func (f *fakeRows) Err() error        { return f.err }
func (f *fakeRows) Close()            { f.closed = true }

func TestCollectIndexFieldsReportsReadFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	for name, c := range map[string]struct {
		rows *fakeRows
		want string
	}{
		"scan":     {&fakeRows{rows: 1, scanErr: boom}, "scan field index registry"},
		"iterator": {&fakeRows{rows: 0, err: boom}, "read field index registry"},
	} {
		got, err := collectIndexFields(c.rows)
		if err == nil || !errors.Is(err, boom) || !strings.Contains(err.Error(), c.want) || got != nil {
			t.Errorf("%s: = %v, %v; want a wrapped %q error and no rows", name, got, err, c.want)
		}
		if !c.rows.closed {
			t.Errorf("%s: the rows were not closed", name)
		}
	}
	got, err := collectIndexFields(&fakeRows{rows: 2})
	if err != nil || len(got) != 1 { // both scripted rows scan to the zero key
		t.Errorf("clean read = %v, %v", got, err)
	}
}

// failingTx is a pgx.Tx whose Query fails before it reaches the server, as it
// does on a closed transaction.
type failingTx struct {
	pgx.Tx
	err error
}

func (f failingTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, f.err }

func TestIndexFieldsWrapsAQueryFailure(t *testing.T) {
	t.Parallel()
	got, err := IndexFields(t.Context(), failingTx{err: pgx.ErrTxClosed}, uuid.New(), uuid.New())
	if got != nil || !errors.Is(err, pgx.ErrTxClosed) || !strings.Contains(err.Error(), "records/store: read field index registry") {
		t.Errorf("IndexFields = %v, %v; want a wrapped pgx.ErrTxClosed and no rows", got, err)
	}
}
