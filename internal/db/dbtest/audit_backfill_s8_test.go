//go:build integration

package dbtest

import (
	"context"
	"crypto/sha1" //nolint:gosec // FIPS test vectors for the SQL SHA-1, not a security use.
	"encoding/hex"
	"regexp"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// P5-15 (TICK-433): migration A's backfill, ADR-004 R25 (A half) and R26.
// Every S8 version written before the migration gets exactly one
// backfilled entry in its own tenant's log, through canonical per-tenant
// policies that are gone when the migration ends.

const (
	auditBackfillMigration = "20261103120000_audit_backfill_s8.sql"
	// auditBackfillNamespace is the UUIDv5 namespace the migration fixes.
	auditBackfillNamespace = "3767d3ad-0a89-4b41-847c-9e4bf30c2055"
	// tenantC has no entity types: the backfill must give it nothing.
	tenantC = "0190a000-0000-7000-8000-00000000000c"
)

// backfillID is the R25 id, computed the way Go code would.
func backfillID(tenant, version string) string {
	t, v := uuid.MustParse(tenant), uuid.MustParse(version)
	return uuid.NewSHA1(uuid.MustParse(auditBackfillNamespace), append(t[:], v[:]...)).String()
}

// The plpgsql SHA-1 is FIPS 180-4 (one-, two- and zero-block vectors, and
// every padding boundary against crypto/sha1), and audit_backfill_id is
// Go's uuid.NewSHA1 over tenant ‖ version.
func TestP515UUIDv5MatchesGo(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	for msg, want := range map[string]string{
		"":    "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		"abc": "a9993e364706816aba3e25717850c26c9cd0d89d",
		"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq": "84983e441c3bd26ebaae4aa1f95129e5e54670f1",
	} {
		if got := queryString(t, super, "SELECT encode(audit_backfill_sha1(convert_to($1, 'UTF8')), 'hex')", msg); got != want {
			t.Errorf("sha1(%q) = %s, want %s", msg, got, want)
		}
	}
	for n := 0; n <= 130; n++ {
		msg := make([]byte, n)
		for i := range msg {
			msg[i] = byte(i*7 + n)
		}
		sum := sha1.Sum(msg) //nolint:gosec // see import
		if got := queryString(t, super, "SELECT encode(audit_backfill_sha1($1), 'hex')", msg); got != hex.EncodeToString(sum[:]) {
			t.Errorf("sha1 of %d bytes = %s, want %x", n, got, sum)
		}
	}
	for i := 0; i < 200; i++ {
		tenant, version := uuid.NewString(), uuid.NewString()
		if got, want := queryString(t, super, "SELECT audit_backfill_id($1, $2)::text", tenant, version), backfillID(tenant, version); got != want {
			t.Fatalf("audit_backfill_id(%s, %s) = %s, want %s", tenant, version, got, want)
		}
	}
}

// seedBackfillVersions writes, before migration A, the S8 history the
// backfill must cover: tenant A has one type with versions 1 to 3, all with
// a publisher; tenant B has one type whose version 1 onboarding seeded with
// no publisher; tenant C has nothing. It returns the version ids by tenant.
func seedBackfillVersions(t *testing.T, super *pgx.Conn) map[string][]string {
	t.Helper()
	seedTenants(t, super, []string{tenantA, tenantB, tenantC}, nil)
	seedEntityTypes(t, super, tenantA)
	ids := map[string][]string{tenantA: {tenantA}}
	for v, compat := range map[int]string{2: "compatible", 3: "needs_backfill"} {
		id := uuid.NewString()
		exec(t, super, insertVersionRaw, tenantA, id, tenantA, v, testSchema, schemaSum(testSchema), compat, tenantA)
		ids[tenantA] = append(ids[tenantA], id)
	}
	seedIdentity(t, super, tenantB)
	exec(t, super, insertTypeRaw, tenantB, "ticket", "TCK")
	exec(t, super, insertVersionRaw, tenantB, tenantB, tenantB, 1, testSchema, schemaSum(testSchema), "initial", nil)
	ids[tenantB] = []string{tenantB}
	// Past, distinct publish times, so occurred_at = published_at is visible.
	exec(t, super, "UPDATE entity_definition_versions SET published_at = now() - make_interval(days => version * 10)")
	return ids
}

// entryMismatches lists every version whose backfilled entries are not
// exactly one R25 entry, as the superuser sees both tables.
const entryMismatches = `
	SELECT coalesce(string_agg(v.tenant_id || '/' || v.version || ': ' || e.problem, '; '), '')
	FROM entity_definition_versions v
	LEFT JOIN LATERAL (
		SELECT CASE
			WHEN count(*) <> 1 THEN count(*) || ' entries'
			WHEN bool_or(l.id <> audit_backfill_id(v.tenant_id, v.id)) THEN 'id'
			WHEN bool_or(l.occurred_at <> v.published_at) THEN 'occurred_at'
			WHEN bool_or(l.domain <> 'config' OR l.entity_definition_id IS DISTINCT FROM v.definition_id) THEN 'domain'
			WHEN bool_or(l.action <> CASE WHEN v.version = 1 THEN 'create' ELSE 'publish' END) THEN 'action'
			WHEN bool_or(l.version_before IS DISTINCT FROM nullif(v.version - 1, 0)) THEN 'version_before'
			WHEN bool_or(l.content_sha256 <> v.schema_sha256) THEN 'content_sha256'
			WHEN bool_or(NOT l.backfilled) THEN 'backfilled'
			WHEN bool_or(l.actor_principal_id IS DISTINCT FROM v.published_by) THEN 'actor_principal_id'
			WHEN bool_or(v.published_by IS NULL AND (l.actor_system, l.source) IS DISTINCT FROM ('onboarding', 'onboarding'))
				THEN 'onboarding actor'
			WHEN bool_or(v.published_by IS NOT NULL AND (l.actor_system IS NOT NULL OR l.source <> 'user')) THEN 'user actor'
			WHEN bool_or(l.diff IS NOT NULL OR l.seq IS NOT NULL) THEN 'diff or seal'
		END AS problem
		FROM audit_log l
		WHERE l.subject_type = 'entity_definition' AND l.subject_id = v.definition_id AND l.version_after = v.version
	) e ON true
	WHERE e.problem IS NOT NULL`

// CI-A9 (A half) and the grooming addition: after migration A each version
// has exactly one entry, in its own tenant's log, with tenant C empty; a
// re-run (P5-32's call) adds none, and adds exactly one for a version
// published in between.
func TestP515BackfillOnePerVersion(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, auditBackfillMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", auditBackfillMigration, err)
	}
	super := superNava(t)
	ids := seedBackfillVersions(t, super)
	if err := migrateOnlyFS(t, m, migrationsThrough(t, auditBackfillMigration)); err != nil {
		t.Fatalf("migrate through %s: %v", auditBackfillMigration, err)
	}

	if bad := queryString(t, super, entryMismatches); bad != "" {
		t.Errorf("after migration A: %s", bad)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM audit_log"); got != "4" {
		t.Errorf("audit_log has %s entries, want 4 (one per version)", got)
	}
	// Each entry's tenant is its source version's tenant (the id and the
	// subject pin the version; the join above matched on subject only).
	if got := queryString(t, super, `SELECT count(*)::text FROM audit_log l
		JOIN entity_definition_versions v ON l.id = audit_backfill_id(v.tenant_id, v.id)
		WHERE l.tenant_id = v.tenant_id`); got != "4" {
		t.Errorf("entries whose tenant_id is their version's = %s, want 4", got)
	}
	for tenant, want := range map[string]int{tenantA: len(ids[tenantA]), tenantB: len(ids[tenantB]), tenantC: 0} {
		if got := queryString(t, super, "SELECT count(*)::text FROM audit_log WHERE tenant_id = $1", tenant); got != strconv.Itoa(want) {
			t.Errorf("tenant %s has %s entries, want %d", tenant, got, want)
		}
	}
	for tenant, vs := range ids {
		for _, v := range vs {
			if got := queryString(t, super, "SELECT count(*)::text FROM audit_log WHERE tenant_id = $1 AND id = $2", tenant, backfillID(tenant, v)); got != "1" {
				t.Errorf("entry %s for version %s of %s: %s rows, want 1", backfillID(tenant, v), v, tenant, got)
			}
		}
	}

	mig := asRole(t, "nava_migrator")
	if got := queryString(t, mig, "SELECT audit_backfill_s8()::text"); got != "0" {
		t.Errorf("re-run added %s entries, want 0", got)
	}
	exec(t, super, insertVersionRaw, tenantB, uuid.NewString(), tenantB, 2, testSchema, schemaSum(testSchema), "compatible", tenantB)
	if got := queryString(t, mig, "SELECT audit_backfill_s8()::text"); got != "1" {
		t.Errorf("re-run after one new publish added %s entries, want 1", got)
	}
	if bad := queryString(t, super, entryMismatches); bad != "" {
		t.Errorf("after the re-runs: %s", bad)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles)"); got != "0" {
		t.Errorf("%s nava_migrator policies left after the re-runs, want 0", got)
	}

	// The rest of the migrations, then the grants manifest and the gate.
	migrateAll(t, m)
}

// Two tenants: the backfill's own canonical policies, taken from the
// function source so the test cannot drift from it, refuse a planted
// cross-tenant write and hide the other tenant's versions.
func TestP515CanonicalPoliciesRefuseCrossTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)

	src := queryString(t, super, "SELECT prosrc FROM pg_proc WHERE proname = 'audit_backfill_s8'")
	policies := regexp.MustCompile(`CREATE POLICY [^;]+;`).FindAllString(src, -1)
	if len(policies) != 4 {
		t.Fatalf("found %d CREATE POLICY statements in audit_backfill_s8, want 4", len(policies))
	}

	ctx := context.Background()
	mig := asRole(t, "nava_migrator")
	inMigratorTx := func(fn func(tx pgx.Tx) error) error {
		tx, err := mig.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		for _, p := range policies {
			if _, err := tx.Exec(ctx, p); err != nil {
				t.Fatalf("%s: %v", p, err)
			}
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantA); err != nil {
			t.Fatal(err)
		}
		return fn(tx)
	}

	// The backfill's INSERT shape with B's version joined in while A is set.
	planted := `INSERT INTO audit_log (tenant_id, id, occurred_at, domain, action, subject_type, subject_id,
			entity_definition_id, version_after, actor_system, source, backfilled, content_sha256)
		SELECT v.tenant_id, audit_backfill_id(v.tenant_id, v.id), v.published_at, 'config', 'create',
			'entity_definition', v.definition_id, v.definition_id, v.version, 'onboarding', 'onboarding', true, v.schema_sha256
		FROM entity_definition_versions v WHERE v.tenant_id = $1`
	err := inMigratorTx(func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, planted, tenantB)
		if err == nil && tag.RowsAffected() != 0 {
			t.Errorf("cross-tenant INSERT ... SELECT wrote %d rows; B's versions must be invisible", tag.RowsAffected())
		}
		return err
	})
	if err != nil {
		t.Errorf("cross-tenant INSERT ... SELECT: %v, want 0 rows and no error", err)
	}
	// A literal B row, as a broken join would produce it, fails WITH CHECK.
	err = inMigratorTx(func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, occurred_at, domain, action, subject_type,
				subject_id, version_after, actor_system, source, backfilled, content_sha256)
			VALUES ($1, gen_random_uuid(), now() - interval '1 day', 'config', 'create', 'entity_definition',
				$1, 1, 'onboarding', 'onboarding', true, sha256('x'))`, tenantB)
		return err
	})
	wantSQLState(t, "planted tenant B entry with tenant A set", err, "42501")
	// A's own version goes through: exactly one row, so a policy that hid
	// A's versions too (0 rows, no error) fails this control.
	if err := inMigratorTx(func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, planted, tenantA)
		if err == nil && tag.RowsAffected() != 1 {
			t.Errorf("own-tenant backfill insert wrote %d rows, want 1", tag.RowsAffected())
		}
		return err
	}); err != nil {
		t.Errorf("own-tenant backfill insert: %v", err)
	}
}

// R26.3: after the real migrations no policy names nava_migrator, and only
// the migrator can run the backfill.
func TestP515NoMigratorPolicyLeft(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	if got := queryString(t, super, `SELECT coalesce(string_agg(polrelid::regclass || '.' || polname, ', '), '')
		FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles)`); got != "" {
		t.Errorf("nava_migrator policies left after migrating: %s", got)
	}
	for _, role := range []string{"nava_app", "nava_worker", "nava_onboard", "nava_erasure"} {
		_, err := asRole(t, role).Exec(context.Background(), "SELECT audit_backfill_s8()")
		wantSQLState(t, role+" runs the backfill", err, "42501")
	}
}
