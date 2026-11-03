//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// P3-12 (TICK-88), ADR-001 A5-2: cascade_start, the start key. One claim per
// (tenant, origin, record, definition); a repeat returns no row; the origin
// is never NULL; the table is append-only and per tenant.

// claimStart is cascade.ClaimStart's insert: tenant $1, origin $2, record $3,
// definition $4, outbox id $5.
const claimStart = `INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id)
	VALUES ($1, $2, $3, $4, $5) ON CONFLICT ON CONSTRAINT cascade_start_pkey DO NOTHING RETURNING outbox_id`

const (
	originX = "0e000000-0000-4000-8000-0000000000e1"
	defX    = "0e000000-0000-4000-8000-0000000000e2"
	outboxX = "0e000000-0000-4000-8000-0000000000e3"
	outboxY = "0e000000-0000-4000-8000-0000000000e4"
)

// claimRows runs claimStart in tenant's transaction and returns how many
// rows it returned, committing.
func claimRows(conn *pgx.Conn, tenant string, args ...any) (int, error) {
	n := 0
	err := inTenantTx(conn, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(context.Background(), claimStart, args...)
		if err != nil {
			return err
		}
		for rows.Next() {
			n++
		}
		rows.Close()
		return rows.Err()
	})
	return n, err
}

func TestP312CascadeStartKey(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	recA, recB := pipelineRecord(tenantA), pipelineRecord(tenantB) // the same id in both tenants

	for i, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		outbox := []string{outboxX, outboxY}[i]
		def := []string{defX, tenantA}[i] // a key of each role's own
		if n, err := claimRows(conn, tenantA, tenantA, originX, recA, def, outbox); err != nil || n != 1 {
			t.Fatalf("%s: the first claim returned %d rows, %v; want 1", role, n, err)
		}
		if n, err := claimRows(conn, tenantA, tenantA, originX, recA, def, outbox); err != nil || n != 0 {
			t.Errorf("%s: the repeat claim returned %d rows, %v; want 0 (a cycle)", role, n, err)
		}
		// Tenant B's key with the same uuids is its own.
		if n, err := claimRows(conn, tenantB, tenantB, originX, recB, def, outbox); err != nil || n != 1 {
			t.Errorf("%s: tenant B's claim of the same uuids returned %d rows, %v; want 1", role, n, err)
		}

		_, err := claimRows(conn, tenantB, tenantA, originX, recA, defX, outbox)
		wantSQLState(t, role+" in tenant B claims a key for A", err, "42501")
		_, err = claimRows(conn, "", tenantA, originX, recA, defX, outbox)
		wantSQLState(t, role+" with no tenant claims a key", err, "42501")
		_, err = claimRows(conn, tenantA, tenantA, nil, recA, outboxY, outbox)
		wantSQLState(t, role+" claims with a NULL origin", err, "23502")
		for _, q := range []string{
			"UPDATE cascade_start SET outbox_id = gen_random_uuid() WHERE tenant_id = $1",
			"DELETE FROM cascade_start WHERE tenant_id = $1",
		} {
			_, err := execIn(conn, tenantA, q, tenantA)
			wantSQLState(t, role+": "+q, err, "42501")
		}
	}
	for _, role := range []string{"nava_onboard", "nava_groupsync", "nava_relay", "nava_purge"} {
		_, err := execIn(asRole(t, role), tenantA, "SELECT count(*) FROM cascade_start WHERE tenant_id = $1", tenantA)
		wantSQLState(t, role+" reads cascade_start", err, "42501")
	}

	// Tenant A cannot claim on tenant B's record: the composite foreign key
	// holds the tenant, though the policy passes the row.
	other := "0b000000-0000-4000-8000-0000000000b7"
	exec(t, super, "INSERT INTO records (tenant_id, id) VALUES ($1, $2)", tenantB, other)
	_, err := claimRows(asRole(t, "nava_app"), tenantA, tenantA, originX, other, defX, outboxX)
	wantSQLState(t, "tenant A claims on tenant B's record", err, "23503")

	if got := queryString(t, super, "SELECT count(*)::text FROM cascade_start"); got != "4" {
		t.Errorf("%s claims in all, want 4", got)
	}
	if got := queryString(t, super, `SELECT string_agg(relrowsecurity::text || relforcerowsecurity::text, '') FROM pg_class
		WHERE oid = 'cascade_start'::regclass`); got != "truetrue" {
		t.Errorf("cascade_start RLS enabled and forced = %s, want truetrue", got)
	}
}
