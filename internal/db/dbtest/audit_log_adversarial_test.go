//go:build integration

package dbtest

import (
	"testing"

	"github.com/google/uuid"
)

// Security review of P5-10b (TICK-450): adversarial cases that the
// contract tests do not already cover. The attacker holds a valid runtime
// login and a tenant context of its own (tenant A) and goes after tenant B's
// audit rows, or after diff contents it has no grant to read.

// The R11 transitions stay inside the caller's tenant: the sealer and the
// erasure job, with tenant A set, reach no row of tenant B's, and B's diff
// survives.
func TestP510bAdversarialTransitionsStayInTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	victim := appendEntry(t, asRole(t, "nava_app"), tenantB, 1)
	ref := uuid.NewString()

	if n, err := execIn(asRole(t, "nava_erasure"), tenantA, eraseEntry, tenantB, victim, ref); err != nil || n != 0 {
		t.Errorf("nava_erasure (tenant A) erases tenant B's entry: %d rows, %v; want 0 rows", n, err)
	}
	if n, err := execIn(asRole(t, "nava_erasure"), tenantA,
		`UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $1`, ref); err != nil || n != 0 {
		t.Errorf("nava_erasure (tenant A, no tenant predicate) erases: %d rows, %v; want 0 rows", n, err)
	}
	if n, err := execIn(asRole(t, "nava_worker"), tenantA, sealEntry, tenantB, victim); err != nil || n != 0 {
		t.Errorf("nava_worker (tenant A) seals tenant B's entry: %d rows, %v; want 0 rows", n, err)
	}
	// Moving A's own row into B through an allowed transition is refused too.
	own := appendEntry(t, asRole(t, "nava_app"), tenantA, 2)
	_, err := execIn(asRole(t, "nava_erasure"), tenantA,
		`UPDATE audit_log SET tenant_id = $3, diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $4
		 WHERE tenant_id = $1 AND id = $2`, tenantA, own, tenantB, ref)
	wantSQLState(t, "nava_erasure moves an entry to tenant B", err, "42501")

	if got := queryString(t, super, `SELECT (diff IS NOT NULL AND diff_salt IS NOT NULL AND erased_at IS NULL AND seq IS NULL)::text
		FROM audit_log WHERE tenant_id = $1 AND id = $2`, tenantB, victim); got != "true" {
		t.Errorf("tenant B's entry changed: %s", got)
	}
}

// R23: no role without a SELECT grant on diff or diff_salt may learn their
// contents, not through a WHERE predicate on an UPDATE (a row-count oracle),
// not through RETURNING, not through an expression in SET.
func TestP510bAdversarialNoDiffOracle(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	app := asRole(t, "nava_app")
	ref := uuid.NewString()

	probes := map[string][]string{
		"nava_erasure": {
			// Oracle: only rows whose diff matches get erased.
			`UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $3
			 WHERE tenant_id = $1 AND id = $2 AND diff::text LIKE '%"op"%'`,
			`UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $3
			 WHERE tenant_id = $1 AND id = $2 AND diff_salt IS NOT NULL`,
			eraseEntry + ` RETURNING diff`,
			eraseEntry + ` RETURNING diff_salt`,
			eraseEntry + ` RETURNING *`,
			`UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $3
			 WHERE tenant_id = $1 AND id = $2 AND diff_sha256 IS NOT NULL`,
		},
		"nava_worker": {
			sealEntry + ` AND diff::text LIKE '%"op"%'`,
			sealEntry + ` AND diff_salt IS NOT NULL`,
			sealEntry + ` RETURNING diff`,
			sealEntry + ` RETURNING diff_salt`,
		},
		"nava_app": {
			`SELECT diff FROM audit_log WHERE tenant_id = $1 AND id = $2`,
			`SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND id = $2 AND diff::text LIKE '%"op"%'`,
			`SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND id = $2 AND action = 'update'`,
		},
	}
	for role, qs := range probes {
		conn := asRole(t, role)
		for _, q := range qs {
			id := appendEntry(t, app, tenantA, 1)
			args := []any{tenantA, id}
			if role == "nava_erasure" {
				args = append(args, ref)
			}
			_, err := execIn(conn, tenantA, q, args...)
			wantSQLState(t, role+": "+q, err, "42501")
			exec(t, super, "DELETE FROM audit_log WHERE tenant_id = $1", tenantA)
		}
	}
}

// An upsert is not a side door to UPDATE: ON CONFLICT DO UPDATE needs the
// UPDATE grant on every SET column, which nava_app lacks entirely and
// nava_worker and nava_erasure lack outside their transition columns.
func TestP510bAdversarialUpsertIsNoSideDoor(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	for role, set := range map[string]string{
		"nava_app":     "seq = 1, prev_hash = sha256('p'), entry_hash = sha256('e'), sealed_at = now()",
		"nava_worker":  "action = 'delete'",
		"nava_erasure": "action = 'delete'",
	} {
		_, err := execIn(asRole(t, role), tenantA,
			`INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
			 VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', $2, 1, 'workflow', 'workflow')
			 ON CONFLICT ON CONSTRAINT audit_log_commit_key DO UPDATE SET `+set, tenantA, pipelineRecord(tenantA))
		wantSQLState(t, role+" upserts "+set, err, "42501")
	}
}
