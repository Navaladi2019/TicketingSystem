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

	// locked guards a lock that reports not-found yet holds B's row. An
	// unfiltered lock fails the error check first: its later insert breaks
	// RLS on cascade_start or outbox, and the statement error is what the
	// test sees.
	locked := func() bool {
		t.Helper()
		// A second session: NOWAIT fails at once if the row is locked.
		_, err := e.super.Exec(ctx, "SELECT 1 FROM records WHERE id = $1 FOR UPDATE NOWAIT", recB.RecordID)
		return err != nil
	}

	// Each lock runs in its own owner transaction in tenant A, so one
	// failing cannot abort the other's.
	errRollback := errors.New("roll back")
	inA := func(name string, wantErr error, lock func(pgx.Tx) error) {
		t.Helper()
		err := tenancy.InTenantTx(ctx, owner, a.id, func(tx pgx.Tx) error {
			// Control: RLS is really off here, so the owner sees B's row.
			var n int
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM records WHERE id = $1", recB.RecordID).Scan(&n); err != nil {
				return err
			}
			if n != 1 {
				t.Fatalf("owner in tenant A's transaction sees %d of B's records; RLS is not bypassed, the test proves nothing", n)
			}
			if err := lock(tx); !errors.Is(err, wantErr) {
				t.Errorf("%s(B's record) in A's transaction: %v, want %v", name, err, wantErr)
			}
			if locked() {
				t.Errorf("%s locked tenant B's record from tenant A's transaction", name)
			}
			return errRollback
		})
		if !errors.Is(err, errRollback) {
			t.Fatal(err)
		}
	}

	inA("cascade.ClaimStart", cascade.ErrRecordNotFound, func(tx pgx.Tx) error {
		_, _, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: recB.RecordID, DefinitionID: uuid.New()})
		return err
	})
	inA("outbox.EnqueueAtVersion", outbox.ErrRecordNotFound, func(tx pgx.Tx) error {
		payload, _ := json.Marshal(map[string]string{"record_id": recB.RecordID.String()})
		_, err := outbox.EnqueueAtVersion(ctx, tx, outbox.Entry{Kind: "ticket.approval", KindVersion: 1,
			DedupeKey: "tick-480:" + uuid.NewString(), RecordID: recB.RecordID,
			Target: outbox.TargetStart{Kind: "ticket.approval"}, Causation: workflowCausation(), Payload: payload}, recB.Version)
		return err
	})
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", a.id); n != 0 {
		t.Errorf("tenant A has %d outbox rows; want none", n)
	}
	if n := count(t, e.super, "SELECT count(*) FROM cascade_start"); n != 0 {
		t.Errorf("%d cascade_start rows; want none", n)
	}
}

// TestIdempotencyKeyIsScopedByPrincipal (TICK-480, TICK-498): a key belongs
// to one principal. Member B's write under member A's key, in the same
// tenant, creates B's own record: no 409, so B learns nothing about A's key
// (AR-TICK-480-01). Each member's retry replays its own write, a different
// write under a member's own key is still *ErrIdempotencyConflict, and a
// concurrent duplicate by the same principal, or by two principal-less
// system writers, still hits the unique index (ErrDuplicateKey).
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
	resB, err := write(ctx, e, p, f.id, keyed(memberB, key, "b's"))
	if err != nil {
		t.Fatalf("member B's write under A's key = %+v, %v; want B's own record, no conflict", resB, err)
	}
	if resB.RecordID == resA.RecordID {
		t.Fatalf("member B's write under A's key returned A's record %s", resA.RecordID)
	}
	if n := count(t, e.super, "SELECT count(*) FROM records WHERE tenant_id = $1", f.id); n != 2 {
		t.Errorf("tenant has %d records; want A's and B's", n)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND idempotency_key = $2", f.id, key); n != 2 {
		t.Errorf("%d audit rows keyed %s; want one per member", n, key)
	}

	// Each member's retry replays its own write and writes nothing.
	for _, c := range []struct {
		name  string
		actor identity.Principal
		title string
		want  pipeline.Result
	}{{"A", memberA, "a's", resA}, {"B", memberB, "b's", resB}} {
		if again, err := write(ctx, e, p, f.id, keyed(c.actor, key, c.title)); err != nil || again != c.want {
			t.Errorf("member %s's replay = %+v, %v; want %+v", c.name, again, err, c.want)
		}
	}
	// A different write (an update, not a create) under A's own key is
	// still a conflict.
	upd := keyed(memberA, key, "a's")
	upd.Op, upd.RecordID, upd.ExpectedVersion = pipeline.OpUpdate, resA.RecordID, resA.Version
	var conflict *pipeline.ErrIdempotencyConflict
	if res, err := write(ctx, e, p, f.id, upd); !errors.As(err, &conflict) {
		t.Errorf("member A's update under A's create key = %+v, %v; want *ErrIdempotencyConflict", res, err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM records WHERE tenant_id = $1", f.id); n != 2 {
		t.Errorf("tenant has %d records after the replays; want 2", n)
	}

	// The store lookup: each principal finds only its own row; no principal
	// (a system writer) finds neither.
	inTx(t, e, f.id, func(tx pgx.Tx) error {
		for _, c := range []struct {
			name  string
			actor *uuid.UUID
			want  uuid.UUID
		}{{"member A", &memberA.PrincipalID, resA.RecordID}, {"member B", &memberB.PrincipalID, resB.RecordID}} {
			kw, err := store.AuditByKey(ctx, tx, f.id, c.actor, key)
			if err != nil || kw.RecordID != c.want {
				t.Errorf("AuditByKey(%s) = %+v, %v; want record %s", c.name, kw, err, c.want)
			}
		}
		if kw, err := store.AuditByKey(ctx, tx, f.id, nil, key); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("AuditByKey(no principal, the members' key) = %+v, %v; want ErrNotFound", kw, err)
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

	// The concurrent path, where the replay lookup has not seen the other
	// write: the unique index refuses a second keyed audit row by the same
	// principal, and by a second NULL actor (NULLS NOT DISTINCT), and the
	// store reports it as ErrDuplicateKey under the index's name.
	dup := func(name string, actor *uuid.UUID, k string, rec uuid.UUID) {
		t.Helper()
		err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
			return store.InsertAudit(ctx, tx, &store.Audit{TenantID: f.id, ID: uuid.New(), RecordID: rec,
				Action: "create", ActorID: actor, Source: string(pipeline.SourceWorkflow),
				After:          &store.Snapshot{Number: "x", Version: 1, SchemaVersion: 1, Data: map[string]any{}},
				IdempotencyKey: &k})
		})
		if !errors.Is(err, store.ErrDuplicateKey) {
			t.Errorf("%s: a second audit row keyed %s = %v; want ErrDuplicateKey", name, k, err)
		}
	}
	dup("member A", &memberA.PrincipalID, key, resA.RecordID)
	dup("member B", &memberB.PrincipalID, key, resB.RecordID)
	dup("no principal", nil, "wf:system:s1", sys.RecordID)
}
