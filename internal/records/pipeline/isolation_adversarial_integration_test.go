//go:build integration

package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

// Adversarial (Security & Multi-Tenancy Reviewer, PR #356): tenant B's system
// writer reuses an idempotency key that tenant A has already used. The replay
// lookup must not return A's Result (record id, number, version), and the
// partial unique index on (tenant_id, idempotency_key) must not turn A's key
// into a 23505 that tells B the key exists.
func TestAdversarialIdempotencyKeyDoesNotCrossTenants(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	const key = "wf:shared-workflow-id:s1"

	cmdA := createCmd(a, map[string]any{"title": "tenant a"})
	cmdA.Actor, cmdA.Source, cmdA.IdempotencyKey = a.system(), pipeline.SourceWorkflow, key
	resA, err := write(ctx, e, p, a.id, cmdA)
	if err != nil {
		t.Fatal(err)
	}

	cmdB := createCmd(b, map[string]any{"title": "tenant b"})
	cmdB.Actor, cmdB.Source, cmdB.IdempotencyKey = b.system(), pipeline.SourceWorkflow, key
	resB, err := write(ctx, e, p, b.id, cmdB)
	if err != nil {
		t.Fatalf("tenant B with tenant A's key: %v, want a fresh write", err)
	}
	if resB.RecordID == resA.RecordID || resB.Number == resA.Number {
		t.Fatalf("tenant B replayed tenant A's result: %+v", resB)
	}
	for _, f := range []tenantFixture{a, b} {
		if n := count(t, e.super, "SELECT count(*) FROM records WHERE tenant_id = $1", f.id); n != 1 {
			t.Fatalf("tenant %s has %d records, want 1", f.id, n)
		}
		if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND idempotency_key = $2", f.id, key); n != 1 {
			t.Fatalf("tenant %s has %d keyed audit rows, want 1", f.id, n)
		}
	}

	// An update from B that names A's record under A's key: the replay finds
	// B's own row for the key and refuses the different record (no write and
	// no leak of A's result), and the record itself is not visible to B.
	upB := updateCmd(b, resA.RecordID, 1, map[string]any{"summary": "x"})
	upB.Actor, upB.Source, upB.IdempotencyKey = b.system(), pipeline.SourceWorkflow, key
	if res, err := write(ctx, e, p, b.id, upB); err == nil || res.RecordID == resA.RecordID {
		t.Fatalf("tenant B update of A's record under A's key = %+v, %v; want refused", res, err)
	}
	upB.IdempotencyKey = "wf:other:s1"
	if _, err := write(ctx, e, p, b.id, upB); !errors.Is(err, pipeline.ErrNotFound) {
		t.Fatalf("tenant B update of A's record: %v, want ErrNotFound", err)
	}

	// The store lookup itself, inside B's transaction, never returns A's row,
	// even with A's tenant id passed explicitly: RLS hides it.
	inTx(t, e, b.id, func(tx pgx.Tx) error {
		if _, _, err := store.AuditByKey(ctx, tx, a.id, key); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("AuditByKey(tenant A) in tenant B's transaction: %v, want ErrNotFound", err)
		}
		return nil
	})
}

// Adversarial: tenant A's user puts tenant B's principal id in a property
// mapped to a fixed uuid column (requester -> requester_id). The composite
// foreign key (tenant_id, requester_id) must refuse it, on create and on
// update, and nothing is written. The refusal must not depend on whether the
// id exists in B, so it is no existence oracle for B's principals.
func TestAdversarialFixedColumnCannotReferenceAnotherTenant(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()

	foreign := b.principal.PrincipalID.String()
	_, errExisting := write(ctx, e, p, a.id, createCmd(a, map[string]any{"title": "x", "requester": foreign}))
	if errExisting == nil {
		t.Fatal("create naming tenant B's principal as requester succeeded")
	}
	if n := count(t, e.super, "SELECT count(*) FROM records WHERE tenant_id = $1", a.id); n != 0 {
		t.Fatalf("tenant A has %d records after a refused create", n)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", a.id); n != 0 {
		t.Fatalf("tenant A has %d audit rows after a refused create", n)
	}

	rec := mustCreate(t, e, p, a, map[string]any{"title": "ok", "requester": a.principal.PrincipalID.String()})
	if _, err := write(ctx, e, p, a.id, updateCmd(a, rec.RecordID, 1, map[string]any{"requester": foreign})); err == nil {
		t.Fatal("update naming tenant B's principal as requester succeeded")
	}
	if _, _, requester, v, _ := storedData(t, e, rec.RecordID); v != 1 || requester == nil || *requester != a.principal.PrincipalID {
		t.Fatalf("tenant A's record changed: version %d, requester %v", v, requester)
	}
}
