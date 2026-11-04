//go:build integration

package dbtest

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	// No tenant predicate, aimed at B's entry by its own id. Not by subject:
	// pipelineRecord gives tenants A and B the same record id, so a subject
	// predicate would also reach A's seeded record entry (P5-13c), which RLS
	// lets the caller see.
	if n, err := execIn(asRole(t, "nava_erasure"), tenantA,
		`UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $1 WHERE id = $2`,
		ref, victim); err != nil || n != 0 {
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

// R23: the transition roles (nava_erasure, nava_worker) have no SELECT grant
// on diff or diff_salt, so they may not learn their contents: not through a
// WHERE predicate on an UPDATE (a row-count oracle), not through RETURNING,
// not through an expression in SET. Since ADR-004-A1 only nava_audit_reader
// holds SELECT on those columns; TestP510bAppReaderGrantAndTenantBoundary
// pins nava_app's denial and the reader's tenant boundary, and the isolation
// registry's ADR-004-A1/diff-denied probes every runtime role.
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

// P5-17 (TICK-435), inverted by ADR-004-A1 (A1-1, A1-2): nava_app holds
// SELECT on exactly its reader column list of db/grants.yaml (with the
// commit-check columns the merge adds), which no longer names diff or
// diff_salt, so reading either is 42501; no UPDATE on any audit_log column,
// and no table-level DELETE or TRUNCATE. The tenant-boundary half moved to
// nava_audit_reader, the one runtime role that reads the raw diff: the grant
// is not the tenant boundary, row-level security is. With tenant A set it
// reads none of tenant B's rows, diff and diff_salt included, with or
// without a tenant predicate, and with no tenant set it reads nothing.
func TestP510bAppReaderGrantAndTenantBoundary(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")
	reader := asRole(t, "nava_audit_reader")

	m, _ := manifests(t)
	want := slices.Clone(m.Roles["nava_app"].Tables["audit_log"]["SELECT"])
	slices.Sort(want)
	got := strings.Fields(queryString(t, super, `SELECT coalesce(string_agg(a.attname, ' ' ORDER BY a.attname), '')
		FROM pg_attribute a WHERE a.attrelid = 'audit_log'::regclass AND a.attnum > 0 AND NOT a.attisdropped
		  AND has_column_privilege('nava_app', a.attrelid, a.attnum, 'SELECT')`))
	if !slices.Equal(got, want) {
		t.Errorf("nava_app SELECT columns on audit_log = %v, want the reader list %v", got, want)
	}
	if slices.Contains(got, "diff") || slices.Contains(got, "diff_salt") {
		t.Errorf("nava_app holds SELECT on diff or diff_salt: %v (A1-2)", got)
	}
	if bad := queryString(t, super, `SELECT coalesce(string_agg(a.attname, ' '), '')
		FROM pg_attribute a WHERE a.attrelid = 'audit_log'::regclass AND a.attnum > 0 AND NOT a.attisdropped
		  AND (has_column_privilege('nava_app', a.attrelid, a.attnum, 'UPDATE'))`); bad != "" {
		t.Errorf("nava_app can UPDATE audit_log columns: %s", bad)
	}
	for _, priv := range []string{"DELETE", "TRUNCATE", "UPDATE"} {
		if queryString(t, super, "SELECT has_table_privilege('nava_app', 'audit_log', $1)::text", priv) != "false" {
			t.Errorf("nava_app holds table-level %s on audit_log", priv)
		}
	}

	a := appendEntry(t, app, tenantA, 1)
	b := appendEntry(t, app, tenantB, 1)
	for _, q := range []string{"SELECT diff FROM audit_log WHERE id = $1", "SELECT diff_salt FROM audit_log WHERE id = $1"} {
		_, err := execIn(app, tenantA, q, a)
		wantSQLState(t, "nava_app: "+q, err, "42501")
	}
	count := func(tenant, q string, args ...any) int {
		t.Helper()
		var n int
		if err := inTenantTx(reader, tenant, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), q, args...).Scan(&n)
		}); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return n
	}
	const read = `SELECT count(*) FROM (SELECT diff, diff_salt FROM audit_log`
	if n := count(tenantA, read+` WHERE tenant_id = $1) r`, tenantB); n != 0 {
		t.Errorf("the reader in tenant A reads %d of tenant B's diff rows by tenant_id, want 0", n)
	}
	if n := count(tenantA, read+` WHERE id = $1) r`, b); n != 0 {
		t.Errorf("the reader in tenant A reads tenant B's entry by id: %d rows, want 0", n)
	}
	if n := count(tenantA, `SELECT count(*) FROM audit_log WHERE tenant_id <> $1`, tenantA); n != 0 {
		t.Errorf("the reader in tenant A reads %d rows of other tenants with no tenant predicate, want 0", n)
	}
	if n := count(tenantA, read+` WHERE id = $1) r`, a); n != 1 {
		t.Errorf("control: the reader in tenant A reads its own entry: %d rows, want 1", n)
	}
	if n := count("", read+`) r`); n != 0 {
		t.Errorf("with app.tenant_id unset nava_audit_reader reads %d rows, want 0", n)
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
