//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
)

// Security review of P5-11 (TICK-428): adversarial cases for the generated
// R12 audit_log policies. The attacker holds a runtime login for a writer
// role of an audited table and a tenant context of its own, and goes after
// another tenant's audit entries: reading them (the commit check's SELECT
// grant makes them readable in principle) or appending entries in that
// tenant's name.

// countEntries counts the audit_log entries conn sees under tenant.
func countEntries(t *testing.T, conn *pgx.Conn, tenant string) int {
	t.Helper()
	var n int
	if err := inTenantTx(conn, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT count(*) FROM audit_log").Scan(&n)
	}); err != nil {
		t.Fatalf("count audit_log as tenant %s: %v", tenant, err)
	}
	return n
}

// countOtherTenants counts the audit_log entries conn sees under tenant that
// belong to any other tenant. Tenant A's own entries (seedTenants writes one
// per seeded record, P5-13c) are not a leak.
func countOtherTenants(t *testing.T, conn *pgx.Conn, tenant string) int {
	t.Helper()
	var n int
	if err := inTenantTx(conn, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT count(*) FROM audit_log WHERE tenant_id <> $1::uuid", tenant).Scan(&n)
	}); err != nil {
		t.Fatalf("count other tenants' audit_log as tenant %s: %v", tenant, err)
	}
	return n
}

// Every canonical writer, with tenant A set, sees none of tenant B's
// entries and cannot append one in B's name.
func TestP511AdversarialCanonicalWritersStayInTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	exec(t, super, insertEntryAs, tenantB, uuid.NewString(), uuid.NewString(), 1, "record")

	for _, role := range auditedManifest(t).Writers() {
		if role == audited.OnboardRole {
			continue
		}
		conn := asRole(t, role)
		if n := countOtherTenants(t, conn, tenantA); n != 0 {
			t.Errorf("%s (tenant A) sees %d of other tenants' audit entries, want 0 (B's entry leaked)", role, n)
		}
		_, err := execIn(conn, tenantA, insertEntryAs, tenantB, uuid.NewString(), uuid.NewString(), 1, "record")
		wantSQLState(t, role+" (tenant A) appends an entry for tenant B", err, "42501")
	}
}

// nava_onboard's pinned provisioning predicate: with its own provisioning
// tenant X set, it neither reads nor appends entries of an active tenant
// (A) or of another provisioning tenant (Y); and with an active tenant's
// context set, it can append nothing even for that tenant.
func TestP511AdversarialOnboardOwnProvisioningOnly(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX, tenantY})
	exec(t, super, insertEntryAs, tenantA, uuid.NewString(), uuid.NewString(), 1, "entity_definition")
	exec(t, super, insertEntryAs, tenantY, uuid.NewString(), uuid.NewString(), 1, "entity_definition")

	onboard := asRole(t, audited.OnboardRole)
	if n := countEntries(t, onboard, tenantX); n != 0 {
		t.Errorf("nava_onboard (tenant X) sees %d audit entries, want 0 (A's or Y's leaked)", n)
	}
	if n := countEntries(t, onboard, tenantA); n != 0 {
		t.Errorf("nava_onboard (active tenant A set) sees %d audit entries, want 0", n)
	}
	for _, tc := range []struct{ what, ctx, target string }{
		{"(tenant X) appends for active tenant A", tenantX, tenantA},
		{"(tenant X) appends for provisioning tenant Y", tenantX, tenantY},
		{"(active tenant A set) appends for A", tenantA, tenantA},
		{"(no tenant set) appends for X", "", tenantX},
	} {
		_, err := execIn(onboard, tc.ctx, insertEntryAs, tc.target, uuid.NewString(), uuid.NewString(), 1, "entity_definition")
		wantSQLState(t, "nava_onboard "+tc.what, err, "42501")
	}
	// Positive control: its own provisioning tenant works, so the denials
	// above are the predicate and not a missing grant.
	if _, err := execIn(onboard, tenantX, insertEntryAs, tenantX, uuid.NewString(), uuid.NewString(), 1, "entity_definition"); err != nil {
		t.Errorf("nava_onboard (tenant X) appends for X: %v, want success", err)
	}
}

// The commit check cannot be satisfied across tenants: a writer that writes
// tenant A's row, then switches the transaction to tenant B and appends the
// matching entry there (same txid, subject type and subject id), still fails
// at COMMIT; an entry in the wrong tenant never counts.
func TestP511AdversarialCommitCheckIgnoresOtherTenantsEntry(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	writers := auditedManifest(t).Writers()
	createTestTable(t, super, writers)
	ctx := context.Background()

	conn := asRole(t, "nava_app")
	row := uuid.NewString()
	err := inTenantTx(conn, tenantA, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO "+testTable+" (tenant_id, id, val) VALUES ($1, $2, 'a')", tenantA, row); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantB); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, insertEntryAs, tenantB, uuid.NewString(), row, 1, "record")
		return err
	})
	if code, constraint := pgErrDetail(err); code != "23514" || constraint != testTable+"_audit_commit_check" {
		t.Errorf("A's row with B's entry: err = %v (constraint %q), want 23514 from the commit check", err, constraint)
	}
}
