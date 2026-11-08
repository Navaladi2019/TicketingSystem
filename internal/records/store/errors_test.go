package store_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

// The statements run against real PostgreSQL in store_integration_test.go;
// these pin the error paths around them on a transaction whose statements
// answer from a script: every database error is wrapped naming its step,
// and a value that cannot be encoded fails before any statement.

var errBoom = errors.New("boom")

type rowFunc func(dest ...any) error

func (f rowFunc) Scan(dest ...any) error { return f(dest...) }

func failRow(err error) rowFunc { return func(...any) error { return err } }

// fakeTx answers QueryRow with row, Exec with execErr and Query with rows
// (or queryErr), and records the SQL and arguments it was given.
type fakeTx struct {
	pgx.Tx   // nil: any method not overridden panics, which fails the test
	row      rowFunc
	execErr  error
	rows     pgx.Rows
	queryErr error
	sql      []string
	args     [][]any
}

func (x *fakeTx) note(sql string, args []any) {
	x.sql = append(x.sql, sql)
	x.args = append(x.args, args)
}

func (x *fakeTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	x.note(sql, args)
	return x.row
}

func (x *fakeTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	x.note(sql, args)
	return pgconn.NewCommandTag("OK"), x.execErr
}

func (x *fakeTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	x.note(sql, args)
	return x.rows, x.queryErr
}

// checkErr fails unless err wraps is and reads want.
func checkErr(t *testing.T, err, is error, want string) {
	t.Helper()
	if !errors.Is(err, is) {
		t.Errorf("err = %v, want it to wrap %v", err, is)
	}
	if err == nil || err.Error() != want {
		t.Errorf("message %v, want %q", err, want)
	}
}

func TestStatementErrorsAreWrapped(t *testing.T) {
	ctx := context.Background()
	tenant, id := uuid.New(), uuid.New()

	_, err := store.DefinitionID(ctx, &fakeTx{row: failRow(errBoom)}, tenant, "asset")
	checkErr(t, err, errBoom, "records/store: read entity type: boom")

	_, err = store.DefinitionID(ctx, &fakeTx{row: failRow(pgx.ErrNoRows)}, tenant, "asset")
	checkErr(t, err, store.ErrNotFound, "records/store: not found")

	_, err = store.NextNumber(ctx, &fakeTx{row: failRow(errBoom)}, tenant, id)
	checkErr(t, err, errBoom, "records/store: take record number: boom")

	_, err = store.Get(ctx, &fakeTx{row: failRow(errBoom)}, tenant, id)
	checkErr(t, err, errBoom, "records/store: read record: boom")

	_, err = store.Update(ctx, &fakeTx{row: failRow(errBoom)}, &store.Record{TenantID: tenant, ID: id}, 1)
	checkErr(t, err, errBoom, "records/store: update record: boom")

	err = store.Insert(ctx, &fakeTx{execErr: errBoom}, &store.Record{TenantID: tenant, ID: id})
	checkErr(t, err, errBoom, "records/store: insert record: boom")

	// The v0 audit statements (InsertAudit, AuditByKey) left the store with
	// P5-13b/P5-17; their idempotency successor is record_write_keys (D-5-3).
	err = store.InsertWriteKey(ctx, &fakeTx{execErr: errBoom}, &store.WriteKey{TenantID: tenant, RecordID: id, Action: "create"})
	checkErr(t, err, errBoom, "records/store: insert write key: boom")

	_, err = store.WriteKeyByKey(ctx, &fakeTx{row: failRow(errBoom)}, tenant, nil, "k")
	checkErr(t, err, errBoom, "records/store: read write key: boom")

	_, err = store.PayloadJSON(ctx, &fakeTx{row: failRow(errBoom)}, tenant, id, 1, nil)
	checkErr(t, err, errBoom, "records/store: read payload as jsonb: boom")

	_, err = store.GroupLive(ctx, &fakeTx{row: failRow(errBoom)}, tenant, id)
	checkErr(t, err, errBoom, "records/store: read group: boom")

	err = store.LockUnique(ctx, &fakeTx{execErr: errBoom}, tenant, id, "serial", `"SN-1"`)
	checkErr(t, err, errBoom, "records/store: lock unique value: boom")

	_, err = store.UniqueTaken(ctx, &fakeTx{row: failRow(errBoom)}, tenant, id, uuid.Nil, "serial", "SN-1", nil)
	checkErr(t, err, errBoom, "records/store: check unique value: boom")
}

// Data that does not encode to JSON fails before any statement runs.
func TestUnencodableValuesFailBeforeAnyStatement(t *testing.T) {
	ctx := context.Background()
	nan := map[string]any{"x": math.NaN()}
	cases := map[string]func(tx pgx.Tx) error{
		"insert": func(tx pgx.Tx) error { return store.Insert(ctx, tx, &store.Record{Data: nan}) },
		"update": func(tx pgx.Tx) error { _, err := store.Update(ctx, tx, &store.Record{Data: nan}, 1); return err },
		"unique value": func(tx pgx.Tx) error {
			_, err := store.UniqueTaken(ctx, tx, uuid.New(), uuid.New(), uuid.Nil, "tag", math.Inf(1), nil)
			return err
		},
	}
	prefix := map[string]string{
		"insert":       "records/store: encode data: ",
		"update":       "records/store: encode data: ",
		"unique value": "records/store: encode unique value: ",
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			tx := &fakeTx{}
			err := run(tx)
			if err == nil || !strings.HasPrefix(err.Error(), prefix[name]) {
				t.Fatalf("err = %v, want %q...", err, prefix[name])
			}
			if len(tx.sql) != 0 {
				t.Errorf("statements ran: %q", tx.sql)
			}
		})
	}
}

// A key another write by the same principal holds is ErrDuplicateKey (D-5-3,
// TICK-626: the per-principal index); a unique violation on any other
// constraint, the dropped tenant-wide primary key included, stays a wrapped
// database error.
func TestInsertWriteKeyMapsTheDuplicateKey(t *testing.T) {
	ctx := context.Background()
	dup := &pgconn.PgError{Code: "23505", ConstraintName: "record_write_keys_actor_key"}
	if err := store.InsertWriteKey(ctx, &fakeTx{execErr: dup}, &store.WriteKey{}); !errors.Is(err, store.ErrDuplicateKey) {
		t.Fatalf("err = %v, want ErrDuplicateKey", err)
	}
	for _, name := range []string{"some_other_key", "record_write_keys_pkey"} {
		other := &pgconn.PgError{Code: "23505", ConstraintName: name}
		err := store.InsertWriteKey(ctx, &fakeTx{execErr: other}, &store.WriteKey{})
		if errors.Is(err, store.ErrDuplicateKey) || !errors.Is(err, other) {
			t.Fatalf("%s: err = %v, want the wrapped database error", name, err)
		}
	}
	if err := store.InsertWriteKey(ctx, &fakeTx{}, &store.WriteKey{}); err != nil {
		t.Fatalf("insert = %v", err)
	}
}

// A stored data column that is not a JSON object fails the read.
func TestGetRefusesUndecodableData(t *testing.T) {
	tx := &fakeTx{row: func(dest ...any) error {
		*dest[14].(*[]byte) = []byte(`[1,2]`)
		return nil
	}}
	_, err := store.Get(context.Background(), tx, uuid.New(), uuid.New())
	if err == nil || !strings.HasPrefix(err.Error(), "records/store: decode data: ") {
		t.Fatalf("err = %v, want a decode error", err)
	}
}

// WriteKeyByKey returns the scanned row, and ErrNotFound for no row.
func TestWriteKeyByKey(t *testing.T) {
	ctx := context.Background()
	tenant, rec := uuid.New(), uuid.New()
	tx := &fakeTx{row: func(dest ...any) error {
		*dest[1].(*string) = "update"
		*dest[2].(*uuid.UUID) = rec
		*dest[3].(*string) = "AST1"
		*dest[4].(*int64) = 3
		return nil
	}}
	actor := uuid.New()
	k, err := store.WriteKeyByKey(ctx, tx, tenant, &actor, "k")
	if err != nil || k.TenantID != tenant || k.IdempotencyKey != "k" || k.Action != "update" || k.RecordID != rec || k.Number != "AST1" || k.Version != 3 || k.ActorPrincipalID == nil || *k.ActorPrincipalID != actor {
		t.Fatalf("WriteKeyByKey = %+v, %v", k, err)
	}
	_, err = store.WriteKeyByKey(ctx, &fakeTx{row: failRow(pgx.ErrNoRows)}, tenant, nil, "k")
	checkErr(t, err, store.ErrNotFound, "records/store: not found")
}

// UniqueTaken compares a fixed-column property on its column, and refuses a
// mapping to a column it does not know.
func TestUniqueTakenColumns(t *testing.T) {
	ctx := context.Background()
	fixed := map[string]string{
		"assignee": metadata.ColumnAssigneeID,
		"group":    metadata.ColumnAssignmentGroupID,
		"bogus":    "created_by",
	}
	for field, want := range map[string]string{
		"assignee": "to_jsonb(r.assignee_id) = $4::jsonb",
		"group":    "to_jsonb(r.assignment_group_id) = $4::jsonb",
	} {
		tx := &fakeTx{row: func(dest ...any) error { *dest[0].(*bool) = true; return nil }}
		v := uuid.NewString()
		taken, err := store.UniqueTaken(ctx, tx, uuid.New(), uuid.New(), uuid.Nil, field, v, fixed)
		if err != nil || !taken {
			t.Fatalf("%s: %v, %v", field, taken, err)
		}
		if !strings.Contains(tx.sql[0], want) {
			t.Errorf("%s: SQL %q does not compare on %s", field, tx.sql[0], want)
		}
		if got := tx.args[0][3]; got != `"`+v+`"` {
			t.Errorf("%s: compared value %v, want the JSON string", field, got)
		}
	}
	tx := &fakeTx{}
	_, err := store.UniqueTaken(ctx, tx, uuid.New(), uuid.New(), uuid.Nil, "bogus", "x", fixed)
	if err == nil || err.Error() != `records/store: unknown fixed column "created_by"` {
		t.Fatalf("err = %v", err)
	}
	if len(tx.sql) != 0 {
		t.Errorf("statements ran: %q", tx.sql)
	}
}

func TestPayloadAndSplitOfEmptyInput(t *testing.T) {
	r := store.Record{State: "open"}
	if got := r.Payload(map[string]string{"status": metadata.ColumnState}); !reflect.DeepEqual(got, map[string]any{"status": "open"}) {
		t.Fatalf("Payload of nil data = %v", got)
	}
	if got := r.Payload(nil); got == nil || len(got) != 0 {
		t.Fatalf("Payload of nil data, no columns = %v, want an empty object", got)
	}
	if err := r.Split(nil, nil); err != nil || r.Data == nil || len(r.Data) != 0 {
		t.Fatalf("Split(nil) = %v, data %v; want an empty object", err, r.Data)
	}
	err := r.Split(map[string]any{"owner": uuid.NewString()}, map[string]string{"owner": "created_by"})
	if err == nil || err.Error() != `records/store: unknown fixed column "created_by"` {
		t.Fatalf("Split to an unknown column: %v", err)
	}
}

// PayloadJSON builds the payload from the four known fixed columns only,
// passes property names as parameters, and is ErrStale for no row.
func TestPayloadJSON(t *testing.T) {
	ctx := context.Background()
	fixed := map[string]string{
		"assignee": metadata.ColumnAssigneeID,
		"group":    metadata.ColumnAssignmentGroupID,
		"reporter": metadata.ColumnRequesterID,
		"status":   metadata.ColumnState,
	}
	tx := &fakeTx{row: func(dest ...any) error { *dest[0].(*string) = `{"status":"open"}`; return nil }}
	got, err := store.PayloadJSON(ctx, tx, uuid.New(), uuid.New(), 2, fixed)
	if err != nil || string(got) != `{"status":"open"}` {
		t.Fatalf("PayloadJSON = %s, %v", got, err)
	}
	if !strings.Contains(tx.sql[0], "jsonb_build_object($8::text, state)") ||
		!strings.Contains(tx.sql[0], "CASE WHEN assignee_id IS NULL") {
		t.Errorf("SQL %q", tx.sql[0])
	}
	if want := []any{"assignee", "group", "reporter", "status"}; !reflect.DeepEqual(tx.args[0][4:], want) {
		t.Errorf("property args = %v, want %v", tx.args[0][4:], want)
	}

	_, err = store.PayloadJSON(ctx, &fakeTx{row: failRow(pgx.ErrNoRows)}, uuid.New(), uuid.New(), 2, nil)
	if !errors.Is(err, store.ErrStale) {
		t.Fatalf("no row: %v, want ErrStale", err)
	}

	bad := &fakeTx{}
	_, err = store.PayloadJSON(ctx, bad, uuid.New(), uuid.New(), 2, map[string]string{"owner": "created_by"})
	if err == nil || err.Error() != `records/store: unknown fixed column "created_by"` {
		t.Fatalf("unknown column: %v", err)
	}
	if len(bad.sql) != 0 {
		t.Errorf("statements ran: %q", bad.sql)
	}
}

// GroupLive reports the EXISTS answer it reads.
func TestGroupLive(t *testing.T) {
	for _, want := range []bool{true, false} {
		tx := &fakeTx{row: func(dest ...any) error { *dest[0].(*bool) = want; return nil }}
		if live, err := store.GroupLive(context.Background(), tx, uuid.New(), uuid.New()); err != nil || live != want {
			t.Fatalf("GroupLive = %v, %v; want %v", live, err, want)
		}
	}
}
