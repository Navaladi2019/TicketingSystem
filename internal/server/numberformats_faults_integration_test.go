//go:build integration

// P6-07 (TICK-672): a failing statement in the number-format store, on a real
// PostgreSQL. The store runs several statements (head, version, the audit
// reads and writes); a failure of any one of them must come back as an error
// and leave no head, version or audit row, and must not move current_version.
// Helpers are in entitytypes_integration_test.go and numberformats_integration_test.go.

package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/numberformats"
	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

var errNFInjected = errors.New("injected database fault")

// nfFaults numbers the statements the store sends, in call order, and fails
// exactly the failAt-th one.
type nfFaults struct {
	failAt, n int
	fired     bool
}

func (f *nfFaults) hit() bool {
	f.n++
	if f.n-1 == f.failAt {
		f.fired = true
		return true
	}
	return false
}

type nfFaultTx struct {
	pgx.Tx
	f *nfFaults
}

func (x *nfFaultTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if x.f.hit() {
		return pgconn.CommandTag{}, errNFInjected
	}
	return x.Tx.Exec(ctx, sql, args...)
}

func (x *nfFaultTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if x.f.hit() {
		return nfErrRow{}
	}
	return x.Tx.QueryRow(ctx, sql, args...)
}

func (x *nfFaultTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if x.f.hit() {
		return nil, errNFInjected
	}
	return x.Tx.Query(ctx, sql, args...)
}

type nfErrRow struct{}

func (nfErrRow) Scan(...any) error { return errNFInjected }

// rows is the tenant's number-format state: heads, versions and audit rows,
// and the sum of the heads' current_version.
func nfState(t *testing.T, e *etEnv) [4]int {
	t.Helper()
	var s [4]int
	if err := e.super.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM number_formats WHERE tenant_id = $1),
		(SELECT count(*) FROM number_format_versions WHERE tenant_id = $1),
		(SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND subject_type = 'number_format'),
		(SELECT coalesce(sum(current_version), 0) FROM number_formats WHERE tenant_id = $1)`, e.a.id).
		Scan(&s[0], &s[1], &s[2], &s[3]); err != nil {
		t.Fatal(err)
	}
	return s
}

// sweep runs run with each statement failing in turn, in a fresh tenant
// transaction. Every injected fault must be returned as an error wrapping
// errNFInjected and leave the state as it was. The first run that injects
// nothing must succeed and change the state.
func nfSweep(t *testing.T, e *etEnv, run func(tx pgx.Tx) error) {
	t.Helper()
	before := nfState(t, e)
	for failAt := 0; ; failAt++ {
		f := &nfFaults{failAt: failAt}
		err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
			return run(&nfFaultTx{Tx: tx, f: f})
		})
		if !f.fired {
			if err != nil {
				t.Fatalf("no fault injected, but run failed: %v", err)
			}
			if nfState(t, e) == before {
				t.Fatal("the unfaulted run changed nothing")
			}
			if failAt < 4 {
				t.Fatalf("only %d statements were faulted; the sweep is not reaching the audit statements", failAt)
			}
			return
		}
		if !errors.Is(err, errNFInjected) {
			t.Fatalf("fault %d: err = %v, want the injected fault", failAt, err)
		}
		if got := nfState(t, e); got != before {
			t.Fatalf("fault %d left rows behind: %v -> %v", failAt, before, got)
		}
	}
}

func nfDoc() json.RawMessage {
	return json.RawMessage(`{"template":"{prefix}-{yyyy}-{seq:4}","overflow":"grow"}`)
}

func TestNumberFormatCreateIsAtomicUnderAStatementFailure(t *testing.T) {
	e := newETEnv(t)
	c, err := numbering.ParseFormat(nfDoc())
	if err != nil {
		t.Fatal(err)
	}
	nfSweep(t, e, func(tx pgx.Tx) error {
		_, err := numberformats.Store{}.Create(t.Context(), tx, e.a.id, numberformats.NewInput{
			Key: "atomic", Document: nfDoc(), Compiled: c, CreatedBy: e.a.admin.PrincipalID})
		return err
	})
}

func TestNumberFormatPublishIsAtomicUnderAStatementFailure(t *testing.T) {
	e := newETEnv(t)
	c, err := numbering.ParseFormat(nfDoc())
	if err != nil {
		t.Fatal(err)
	}
	var id uuid.UUID
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		v, err := numberformats.Store{}.Create(t.Context(), tx, e.a.id, numberformats.NewInput{
			Key: "atomic", Document: nfDoc(), Compiled: c, CreatedBy: e.a.admin.PrincipalID})
		if v != nil {
			id = v.ID
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	nfSweep(t, e, func(tx pgx.Tx) error {
		_, err := numberformats.Store{}.Publish(t.Context(), tx, e.a.id, numberformats.PublishInput{
			ID: id, Document: nfDoc(), Compiled: c, CreatedBy: e.a.admin.PrincipalID, ExpectedVersion: []int{1}})
		return err
	})
}

// A stale ExpectedVersion reaches the locked-row check only when two
// publishes pass the handler's pre-check together; here the store is called
// directly with a version that is not current.
func TestNumberFormatPublishRefusesAStaleVersionOnTheLockedRow(t *testing.T) {
	e := newETEnv(t)
	c, err := numbering.ParseFormat(nfDoc())
	if err != nil {
		t.Fatal(err)
	}
	var id uuid.UUID
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		v, err := numberformats.Store{}.Create(t.Context(), tx, e.a.id, numberformats.NewInput{
			Key: "stale", Document: nfDoc(), Compiled: c, CreatedBy: e.a.admin.PrincipalID})
		if v != nil {
			id = v.ID
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	before := nfState(t, e)
	err = tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		_, err := numberformats.Store{}.Publish(t.Context(), tx, e.a.id, numberformats.PublishInput{
			ID: id, Document: nfDoc(), Compiled: c, CreatedBy: e.a.admin.PrincipalID, ExpectedVersion: []int{7}})
		return err
	})
	var conflict *numberformats.ConflictError
	if !errors.As(err, &conflict) || conflict.Current != 1 {
		t.Fatalf("err = %v, want a ConflictError at version 1", err)
	}
	if nfState(t, e) != before {
		t.Error("a stale publish wrote")
	}
}
