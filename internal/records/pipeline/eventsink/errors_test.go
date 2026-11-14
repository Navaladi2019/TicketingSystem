package eventsink_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline/eventsink"
)

// The statements run against real PostgreSQL in
// internal/records/pipeline/eventsink_integration_test.go; these pin the
// error paths around them on a transaction whose statements answer from a
// script: every failure rolls the write back with an error naming its step,
// and the sink writes nothing after a failed read.

var errBoom = errors.New("boom")

type rowFunc func(dest ...any) error

func (f rowFunc) Scan(dest ...any) error { return f(dest...) }

// tenantRow answers tenancy.TenantOf's read of the transaction's tenant.
func tenantRow(tenant uuid.UUID) rowFunc {
	return func(dest ...any) error {
		*dest[0].(*string) = tenant.String()
		return nil
	}
}

// noRows is a result with no rows that ends with err.
type noRows struct {
	pgx.Rows // nil: any method not overridden panics, which fails the test
	err      error
}

func (noRows) Close()                        {}
func (noRows) Next() bool                    { return false }
func (r noRows) Err() error                  { return r.err }
func (noRows) CommandTag() pgconn.CommandTag { return pgconn.NewCommandTag("SELECT 0") }

// result is one scripted answer to Query.
type result struct {
	rows pgx.Rows
	err  error
}

// fakeTx answers QueryRow with row and the n-th Query with results[n]; it
// has no Exec, so any write panics and fails the test.
type fakeTx struct {
	pgx.Tx
	row     rowFunc
	results []result
	queries int
}

func (x *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row { return x.row }

func (x *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	r := x.results[x.queries]
	x.queries++
	return r.rows, r.err
}

func TestRecordChangedErrorsAreWrapped(t *testing.T) {
	ctx := context.Background()
	tenant := uuid.New()
	empty := result{rows: noRows{}}
	for _, tc := range []struct {
		name    string
		tx      *fakeTx
		op      pipeline.Op
		want    string
		queries int
	}{
		{
			name: "tenant read fails",
			tx:   &fakeTx{row: func(...any) error { return errBoom }},
			want: "eventsink: tenancy: read tenant: boom",
		},
		{
			name:    "triggered query fails",
			tx:      &fakeTx{row: tenantRow(tenant), results: []result{{err: errBoom}}},
			op:      pipeline.OpCreate,
			want:    "eventsink: find triggered definitions: boom",
			queries: 1,
		},
		{
			name:    "triggered rows fail",
			tx:      &fakeTx{row: tenantRow(tenant), results: []result{{rows: noRows{err: errBoom}}}},
			want:    "eventsink: find triggered definitions: boom",
			queries: 1,
		},
		{
			name:    "waiting query fails",
			tx:      &fakeTx{row: tenantRow(tenant), results: []result{empty, {err: errBoom}}},
			want:    "eventsink: find waiting runs: boom",
			queries: 2,
		},
		{
			name:    "waiting rows fail",
			tx:      &fakeTx{row: tenantRow(tenant), results: []result{empty, {rows: noRows{err: errBoom}}}},
			want:    "eventsink: find waiting runs: boom",
			queries: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ev := pipeline.Event{Op: tc.op, EntityDefinitionID: uuid.New(), RecordID: uuid.New(), Version: 1}
			err := eventsink.Sink{}.RecordChanged(ctx, tc.tx, ev)
			if !errors.Is(err, errBoom) {
				t.Errorf("err = %v, want it to wrap %v", err, errBoom)
			}
			if err == nil || err.Error() != tc.want {
				t.Errorf("message %v, want %q", err, tc.want)
			}
			if tc.tx.queries != tc.queries {
				t.Errorf("ran %d queries, want %d (nothing after the failure)", tc.tx.queries, tc.queries)
			}
		})
	}
}

// With nothing triggered and nobody waiting, a write enqueues nothing: the
// fake has no Exec, so any write would panic.
func TestRecordChangedWithNothingToDoWritesNothing(t *testing.T) {
	tx := &fakeTx{row: tenantRow(uuid.New()), results: []result{{rows: noRows{}}, {rows: noRows{}}}}
	ev := pipeline.Event{Op: pipeline.OpUpdate, EntityDefinitionID: uuid.New(), RecordID: uuid.New(), Version: 2}
	if err := (eventsink.Sink{}).RecordChanged(context.Background(), tx, ev); err != nil {
		t.Fatalf("RecordChanged: %v", err)
	}
	if tx.queries != 2 {
		t.Errorf("ran %d queries, want 2", tx.queries)
	}
}
