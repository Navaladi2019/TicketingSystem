//go:build integration

package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// ownerDSN is the platform database's owner role (the migrator), which RLS
// does not bind on a table without FORCE ROW LEVEL SECURITY.
func ownerDSN(t *testing.T, database string) string {
	t.Helper()
	m, err := grants.Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := testPassword(m.Owner())
	u.User = url.UserPassword(m.Owner(), pw)
	u.Path = "/" + database
	return u.String()
}

// TestRecordLocksNameTheTenantWithoutRLS (TICK-480): with RLS bypassed (the
// owner role, records not forced), neither record lock, cascade.ClaimStart
// nor outbox.EnqueueAtVersion, finds or locks tenant B's record from tenant
// A's transaction. The tenant filter in the SQL is what holds, not RLS.
func TestRecordLocksNameTheTenantWithoutRLS(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	a, b := seedTenant(t, e), seedTenant(t, e)
	p := newPipeline(e, nil)
	recB := mustCreate(t, e, p, b, map[string]any{"title": "b's"})

	owner := appPool(t, ownerDSN(t, e.platform))
	// The owner is bound by RLS only while records is forced; lift that for
	// this test (the database is recreated by the next migrated call).
	mustExec(t, e.super, "ALTER TABLE records NO FORCE ROW LEVEL SECURITY")
	t.Cleanup(func() {
		_, _ = e.super.Exec(context.Background(), "ALTER TABLE records FORCE ROW LEVEL SECURITY")
	})

	locked := func() bool {
		t.Helper()
		// A second session: NOWAIT fails at once if the row is locked.
		_, err := e.super.Exec(ctx, "SELECT 1 FROM records WHERE id = $1 FOR UPDATE NOWAIT", recB.RecordID)
		return err != nil
	}

	err := tenancy.InTenantTx(ctx, owner, a.id, func(tx pgx.Tx) error {
		// Control: RLS is really off here, so the owner sees B's row.
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM records WHERE id = $1", recB.RecordID).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			t.Fatalf("owner in tenant A's transaction sees %d of B's records; RLS is not bypassed, the test proves nothing", n)
		}

		_, _, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: recB.RecordID, DefinitionID: uuid.New()})
		if !errors.Is(err, cascade.ErrRecordNotFound) {
			t.Errorf("cascade.ClaimStart(B's record) in A's transaction: %v, want ErrRecordNotFound", err)
		}
		if locked() {
			t.Error("cascade.ClaimStart locked tenant B's record from tenant A's transaction")
		}

		payload, _ := json.Marshal(map[string]string{"record_id": recB.RecordID.String()})
		_, err = outbox.EnqueueAtVersion(ctx, tx, outbox.Entry{Kind: "ticket.approval", KindVersion: 1,
			DedupeKey: "tick-480:" + uuid.NewString(), RecordID: recB.RecordID,
			Target: outbox.TargetStart{Kind: "ticket.approval"}, Causation: workflowCausation(), Payload: payload}, recB.Version)
		if !errors.Is(err, outbox.ErrRecordNotFound) {
			t.Errorf("outbox.EnqueueAtVersion(B's record) in A's transaction: %v, want ErrRecordNotFound", err)
		}
		if locked() {
			t.Error("outbox.EnqueueAtVersion locked tenant B's record from tenant A's transaction")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", a.id); n != 0 {
		t.Errorf("tenant A has %d outbox rows; want none", n)
	}
	if n := count(t, e.super, "SELECT count(*) FROM cascade_start"); n != 0 {
		t.Errorf("%d cascade_start rows; want none", n)
	}
}

// TestIdempotencyKeyIsScopedByPrincipal (TICK-480): a key replays only the
// principal's own write. Member B's write under member A's key, in the same
// tenant, never returns A's record: it is refused as a conflict and writes
// nothing. A's own replay and a principal-less system writer's (as
// cmd/demo-outbox's) still return their earlier write.
func TestIdempotencyKeyIsScopedByPrincipal(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	memberA := identity.Principal{TenantID: f.id, PrincipalID: uuid.New()}
	memberB := identity.Principal{TenantID: f.id, PrincipalID: uuid.New()}
	for _, m := range []identity.Principal{memberA, memberB} {
		mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
			VALUES ($1, $2, $3, true, now())`, f.id, m.PrincipalID, "kc-"+m.PrincipalID.String())
	}
	keyed := func(actor identity.Principal, key, title string) pipeline.Command {
		c := createCmd(f, map[string]any{"title": title})
		c.Actor, c.Source, c.IdempotencyKey, c.Causation = actor, pipeline.SourceWorkflow, key, workflowCausation()
		return c
	}
	const key = "wf:shared:s1"

	resA, err := write(ctx, e, p, f.id, keyed(memberA, key, "a's"))
	if err != nil {
		t.Fatal(err)
	}
	again, err := write(ctx, e, p, f.id, keyed(memberA, key, "a's"))
	if err != nil || again != resA {
		t.Fatalf("member A's replay = %+v, %v; want %+v", again, err, resA)
	}

	resB, err := write(ctx, e, p, f.id, keyed(memberB, key, "b's"))
	var conflict *pipeline.ErrIdempotencyConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("member B's write under A's key = %+v, %v; want *ErrIdempotencyConflict", resB, err)
	}
	if resB.RecordID == resA.RecordID {
		t.Fatalf("member B's write under A's key returned A's record %s", resA.RecordID)
	}
	if n := count(t, e.super, "SELECT count(*) FROM records WHERE tenant_id = $1", f.id); n != 1 {
		t.Errorf("tenant has %d records; want A's only", n)
	}

	// The store lookup itself: B's principal never finds A's row; no
	// principal (a system writer) does not either.
	inTx(t, e, f.id, func(tx pgx.Tx) error {
		if kw, err := store.AuditByKey(ctx, tx, f.id, &memberB.PrincipalID, key); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("AuditByKey(member B, A's key) = %+v, %v; want ErrNotFound", kw, err)
		}
		if kw, err := store.AuditByKey(ctx, tx, f.id, nil, key); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("AuditByKey(no principal, A's key) = %+v, %v; want ErrNotFound", kw, err)
		}
		kw, err := store.AuditByKey(ctx, tx, f.id, &memberA.PrincipalID, key)
		if err != nil || kw.RecordID != resA.RecordID {
			t.Errorf("AuditByKey(member A, A's key) = %+v, %v; want record %s", kw, err, resA.RecordID)
		}
		return nil
	})

	// A system writer with no principal, as cmd/demo-outbox writes, still
	// replays its own write.
	sys, err := write(ctx, e, p, f.id, keyed(f.system(), "wf:system:s1", "system's"))
	if err != nil {
		t.Fatal(err)
	}
	if again, err := write(ctx, e, p, f.id, keyed(f.system(), "wf:system:s1", "system's")); err != nil || again != sys {
		t.Fatalf("system writer's replay = %+v, %v; want %+v", again, err, sys)
	}
}
