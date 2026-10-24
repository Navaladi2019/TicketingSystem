//go:build integration

package dbtest

import (
	"context"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/purge"
)

// T13 (TICK-68) adversarial guards for the cross-tenant purge. nava_purge
// reaches every tenant, so its reach must stop at the six *_purge policies
// and the nava_purge column grants, whatever the purge code asks for.

// The only policies that name nava_purge are the six paired purge policies.
func TestT13PurgeRoleHasOnlySixPolicies(t *testing.T) {
	migratedPlatform(t)
	got := queryString(t, superNava(t), `
		SELECT string_agg(tablename || '.' || policyname || ':' || cmd, ',' ORDER BY tablename, policyname)
		FROM pg_policies WHERE 'nava_purge' = ANY (roles) OR 'public' = ANY (roles)`)
	want := "idempotency_keys.idempotency_purge:DELETE,idempotency_keys.idempotency_purge_select:SELECT," +
		"outbox.outbox_purge:DELETE,outbox.outbox_purge_select:SELECT," +
		"workflow_ref.workflow_ref_purge:DELETE,workflow_ref.workflow_ref_purge_select:SELECT"
	if got != want {
		t.Fatalf("policies reaching nava_purge:\n got %s\nwant %s", got, want)
	}
}

// A purger given windows far shorter than the policies' (which config.LoadPurge
// rejects, but purge.New accepts) still deletes only what the policies allow:
// the floor is in Postgres, not in Go.
func TestT13PurgePolicyIsTheFloorForShortWindows(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPurgeable(t, super, tenantA, tenantB)

	p := newPurger(t, purgePool(t), purge.Options{Windows: purge.Windows{
		OutboxDispatched: time.Nanosecond, OutboxDead: time.Nanosecond, WorkflowRef: time.Nanosecond,
	}})
	p.Once(context.Background())

	// Same survivors as with the default windows: in-window rows of both
	// tenants are untouched.
	for table, want := range map[string]int{"idempotency_keys": 2, "outbox": 6, "workflow_ref": 4} {
		if got := count(t, super, table); got != want {
			t.Errorf("%s holds %d rows, want %d", table, got, want)
		}
	}
}

// nava_purge cannot widen its reach: it cannot age a live row of tenant B
// into the purge window, cannot write rows, and cannot touch any other table.
func TestT13PurgeRoleCannotWidenItsReach(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	purgeConn := asRole(t, "nava_purge")
	ctx := context.Background()

	for _, q := range []string{
		// Age tenant B's live rows so the DELETE policy would match them.
		"UPDATE idempotency_keys SET expires_at = now() - interval '1 day'",
		"UPDATE outbox SET dispatched_at = now() - interval '8 days'",
		"UPDATE workflow_ref SET closed_at = now() - interval '31 days'",
		// Write rows.
		"INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at) VALUES ('" + tenantB + "', '" + tenantB + "', 'x', '\\x00', now())",
		// Reach other tables.
		"SELECT id FROM records",
		"DELETE FROM records",
		"DELETE FROM tenants",
		"DELETE FROM tenant_identity_binding",
		"SELECT org_id FROM tenant_identity_binding",
		"TRUNCATE outbox",
		// Read the secret-bearing columns.
		"SELECT request_hash FROM idempotency_keys",
		"SELECT dedupe_key FROM outbox",
		"SELECT record_id FROM workflow_ref",
	} {
		_, err := purgeConn.Exec(ctx, q)
		wantSQLState(t, q, err, "42501")
	}
	// Nothing changed for either tenant.
	for table, want := range map[string]int{"idempotency_keys": 2, "outbox": 2, "workflow_ref": 2, "records": 2, "tenants": 2} {
		if got := count(t, super, table); got != want {
			t.Errorf("%s holds %d rows, want %d", table, got, want)
		}
	}
}
