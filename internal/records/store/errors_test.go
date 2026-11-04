package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

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

	err = store.InsertAudit(ctx, &fakeTx{execErr: errBoom}, &store.Audit{TenantID: tenant, ID: id, Action: "create"})
	checkErr(t, err, errBoom, "records/store: insert audit: boom")

	_, err = store.AuditByKey(ctx, &fakeTx{row: failRow(errBoom)}, tenant, "k")
	checkErr(t, err, errBoom, "records/store: read audit by key: boom")

	err = store.LockUnique(ctx, &fakeTx{execErr: errBoom}, tenant, id, "serial", `"SN-1"`)
	checkErr(t, err, errBoom, "records/store: lock unique value: boom")

	_, err = store.UniqueTaken(ctx, &fakeTx{row: failRow(errBoom)}, tenant, id, uuid.Nil, "serial", "SN-1", nil)
	checkErr(t, err, errBoom, "records/store: check unique value: boom")

	_, err = store.History(ctx, &fakeTx{queryErr: errBoom}, tenant, id, time.Time{}, uuid.Nil, 10)
	checkErr(t, err, errBoom, "records/store: read history: boom")
}

// Data that does not encode to JSON fails before any statement runs.
func TestUnencodableValuesFailBeforeAnyStatement(t *testing.T) {
	ctx := context.Background()
	nan := map[string]any{"x": math.NaN()}
	snap := &store.Snapshot{Data: nan}
	cases := map[string]func(tx pgx.Tx) error{
		"insert": func(tx pgx.Tx) error { return store.Insert(ctx, tx, &store.Record{Data: nan}) },
		"update": func(tx pgx.Tx) error { _, err := store.Update(ctx, tx, &store.Record{Data: nan}, 1); return err },
		"audit before": func(tx pgx.Tx) error {
			return store.InsertAudit(ctx, tx, &store.Audit{Before: snap})
		},
		"audit after": func(tx pgx.Tx) error {
			return store.InsertAudit(ctx, tx, &store.Audit{After: snap})
		},
		"unique value": func(tx pgx.Tx) error {
			_, err := store.UniqueTaken(ctx, tx, uuid.New(), uuid.New(), uuid.Nil, "tag", math.Inf(1), nil)
			return err
		},
	}
	prefix := map[string]string{
		"insert":       "records/store: encode data: ",
		"update":       "records/store: encode data: ",
		"audit before": "records/store: encode audit snapshot: ",
		"audit after":  "records/store: encode audit snapshot: ",
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

// An audit row holds a snapshot or a detail, never both.
func TestInsertAuditRefusesSnapshotAndDetail(t *testing.T) {
	tx := &fakeTx{}
	err := store.InsertAudit(context.Background(), tx, &store.Audit{
		After: &store.Snapshot{}, Detail: map[string]string{"reason": "cascade_cycle"},
	})
	if err == nil || err.Error() != "records/store: an audit row has a snapshot or a detail, not both" {
		t.Fatalf("err = %v", err)
	}
	if len(tx.sql) != 0 {
		t.Errorf("statements ran: %q", tx.sql)
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

// keyedRow scans an AuditByKey row.
func keyedRow(record *uuid.UUID, entityKey *string, after []byte) rowFunc {
	return func(dest ...any) error {
		*dest[0].(**uuid.UUID) = record
		*dest[1].(**string) = entityKey
		*dest[2].(*string) = "create"
		*dest[3].(*[]byte) = after
		return nil
	}
}

func TestAuditByKeyRefusesRowsOfNoPipelineRecord(t *testing.T) {
	ctx := context.Background()
	rec, key := uuid.New(), "asset"
	const noRecord = "records/store: keyed audit row names no pipeline record"
	for name, row := range map[string]rowFunc{
		"no record":      keyedRow(nil, &key, []byte(`{}`)),
		"no entity type": keyedRow(&rec, nil, []byte(`{}`)),
		"no snapshot":    keyedRow(&rec, &key, nil),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := store.AuditByKey(ctx, &fakeTx{row: row}, uuid.New(), "k"); err == nil || err.Error() != noRecord {
				t.Fatalf("err = %v, want %q", err, noRecord)
			}
		})
	}
	_, err := store.AuditByKey(ctx, &fakeTx{row: keyedRow(&rec, &key, []byte(`"text"`))}, uuid.New(), "k")
	if err == nil || !strings.HasPrefix(err.Error(), "records/store: decode audit snapshot: ") {
		t.Fatalf("bad snapshot: %v", err)
	}
	_, err = store.AuditByKey(ctx, &fakeTx{row: failRow(pgx.ErrNoRows)}, uuid.New(), "k")
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

// fakeRows yields one row per scan function.
type fakeRows struct {
	pgx.Rows
	scans []rowFunc
	i     int
	err   error
}

func (r *fakeRows) Next() bool             { r.i++; return r.i <= len(r.scans) }
func (r *fakeRows) Scan(dest ...any) error { return r.scans[r.i-1](dest...) }
func (r *fakeRows) Close()                 {}
func (r *fakeRows) Err() error             { return r.err }

func historyRow(before, after string) rowFunc {
	return func(dest ...any) error {
		*dest[0].(*uuid.UUID) = uuid.New()
		*dest[3].(*string) = "user"
		*dest[4].(*string) = "update"
		if before != "" {
			*dest[5].(*[]byte) = []byte(before)
		}
		if after != "" {
			*dest[6].(*[]byte) = []byte(after)
		}
		return nil
	}
}

func TestHistoryErrors(t *testing.T) {
	ctx := context.Background()
	good := `{"number":"AST1","version":1,"schema_version":1,"data":{"n":1}}`
	cases := []struct {
		name string
		rows *fakeRows
		want string
	}{
		{"scan fails", &fakeRows{scans: []rowFunc{failRow(errBoom)}}, "records/store: scan history: boom"},
		{"before not JSON", &fakeRows{scans: []rowFunc{historyRow(`{`, good)}}, "records/store: decode audit snapshot: "},
		{"after data not an object", &fakeRows{scans: []rowFunc{historyRow(good, `{"data":[1]}`)}}, "records/store: decode audit snapshot: "},
		{"rows fail", &fakeRows{err: errBoom}, "records/store: read history: boom"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := store.History(ctx, &fakeTx{rows: c.rows}, uuid.New(), uuid.New(), time.Time{}, uuid.Nil, 10)
			if err == nil || !strings.HasPrefix(err.Error(), c.want) || got != nil {
				t.Fatalf("History = %v, %v; want %q", got, err, c.want)
			}
		})
	}

	// A row with no before (a create) decodes to a nil Before, numbers exact.
	got, err := store.History(ctx, &fakeTx{rows: &fakeRows{scans: []rowFunc{historyRow("", good)}}},
		uuid.New(), uuid.New(), time.Time{}, uuid.Nil, 10)
	if err != nil || len(got) != 1 || got[0].Before != nil || got[0].After.Number != "AST1" || got[0].After.Data["n"] != json.Number("1") {
		t.Fatalf("History = %+v, %v", got, err)
	}
}
