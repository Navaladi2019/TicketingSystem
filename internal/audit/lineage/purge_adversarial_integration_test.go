//go:build integration

// Security review of PR #489 (TICK-590): adversarial guards on nava_purge's
// cross-tenant read of audit_chain_head, beyond TestPurgeRoleReadsOnlyTheCount.
// The attacker holds the nava_purge connection (a compromised nava-purge) and
// tries to reach live heads or to block other tenants' sealers.

package lineage_test

import (
	"context"
	"testing"
)

// nava_purge cannot lock any head. A row lock needs UPDATE privilege; without
// this guard a compromised nava-purge could hold FOR UPDATE/SHARE on every
// dead head and stall chain:reseed (ReseedGateSQL locks the head) and the
// sealer for every tenant.
func TestPurgeRoleCannotLockHeads(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	seedHead(t, e, past)
	for _, lock := range []string{"FOR UPDATE", "FOR NO KEY UPDATE", "FOR SHARE", "FOR KEY SHARE"} {
		_, err := e.purge.Exec(ctx, "SELECT tenant_id FROM audit_chain_head "+lock)
		wantSQLState(t, err, "42501", "SELECT ... "+lock)
	}
}

// nava_purge cannot create a head for any tenant.
func TestPurgeRoleCannotInsertHead(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	_, err := e.purge.Exec(ctx,
		"INSERT INTO audit_chain_head (tenant_id, next_seal_at) VALUES (gen_random_uuid(), now())")
	wantSQLState(t, err, "42501", "INSERT")
}

// A user predicate cannot widen the policy: RLS quals run before the query's
// own WHERE, so a live head stays invisible however the query asks, and no
// tenant setting (app.tenant_id) widens it either.
func TestPurgeRoleCannotSeeLiveHeadsByPredicate(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	alive := seedHead(t, e, inside)
	never := seedHead(t, e, nil)
	for _, q := range []string{
		"SELECT count(*) FROM audit_chain_head WHERE tenant_id = ANY($1::uuid[]) OR next_seal_at IS NULL",
		"SELECT count(*) FROM audit_chain_head WHERE tenant_id = ANY($1::uuid[]) AND (next_seal_at IS NULL OR next_seal_at > now() - interval '1 day')",
	} {
		var n int64
		if err := e.purge.QueryRow(ctx, q, []string{alive.String(), never.String()}).Scan(&n); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		if n != 0 {
			t.Fatalf("%s: nava_purge sees %d live or lineage-less heads, want 0", q, n)
		}
	}
	tx, err := e.purge.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", alive.String()); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM audit_chain_head WHERE tenant_id = $1", alive).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("nava_purge with app.tenant_id set sees the tenant's live head: %d", n)
	}
}
