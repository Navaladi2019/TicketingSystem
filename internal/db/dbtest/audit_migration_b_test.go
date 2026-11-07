//go:build integration

package dbtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// P5-32 (TICK-442): migration B, ADR-004 R25 (B half), R26, R27 and the R16
// seeding for existing tenants. One transaction re-runs the S8 backfill,
// attaches the commit check to the entity-definition tables, and seeds every
// tenant's chain head and first audit.seal start row with the rows onboarding
// (P5-33) and outbox.EnqueueSeal (C5) write.

const migrationB = "20261107053707_audit_migration_b.sql"

// migrateBeforeB runs every migration before B, then the grants manifest and
// the catalogue gate with the audited manifest as it was then: the
// entity-definition tables are not attached: true until B.
func migrateBeforeB(t *testing.T, m *grants.Manifest) (migrate.Result, error) {
	t.Helper()
	a := *auditedManifest(t)
	a.Audited = make(map[string]audited.Table, len(a.Audited))
	for name, e := range auditedManifest(t).Audited {
		if name == "entity_definitions" || name == "entity_definition_versions" {
			e.Attached = false
		}
		a.Audited[name] = e
	}
	log, _ := newLog()
	return migrate.Run(context.Background(), roleConfig(t, m.Owner(), m.PlatformDatabase),
		migrationsBefore(t, migrationB), m, &a, globalTables(t), log, migrate.Options{LockRetrySeconds: 1})
}

// entriesPerVersion lists every S8 version that does not have exactly one
// entry for its subject and version, as the superuser sees both tables.
const entriesPerVersion = `
	SELECT coalesce(string_agg(v.tenant_id || '/' || v.version || ': ' || n, '; '), '')
	FROM entity_definition_versions v
	CROSS JOIN LATERAL (SELECT count(*) AS n FROM audit_log l
		WHERE l.tenant_id = v.tenant_id AND l.subject_type = 'entity_definition'
		  AND l.subject_id = v.definition_id AND l.version_after = v.version) e
	WHERE n <> 1`

// sealShape counts the tenant's heads that are at the column defaults and
// point at a start row in the C5 shape of EnqueueSeal(ctx, tx, 1), with no
// workflow_ref. Both ids are UUIDv7.
const sealShape = `SELECT count(*) FROM outbox o JOIN audit_chain_head h
	ON h.tenant_id = o.tenant_id AND h.next_seal_outbox_id = o.id
	WHERE o.tenant_id = $1 AND o.kind = 'audit.seal' AND o.kind_version = 1 AND o.dedupe_key = 'audit.seal:1'
	  AND o.target_key = 'audit.seal:' || o.id::text AND o.target_seq = 1
	  AND o.causation_id = o.id AND o.causation_depth = 0 AND o.payload = '{}' AND o.trace_context IS NULL
	  AND o.available_at = o.created_at AND o.attempts = 0 AND o.dispatched_at IS NULL AND o.dead_at IS NULL
	  AND h.version = 1 AND h.seq = 0 AND h.entry_hash = decode(repeat('00', 32), 'hex') AND h.seal_interval_s = 10
	  AND h.next_seal_at = o.created_at AND h.export_lease_until IS NULL
	  AND substr(o.id::text, 15, 1) = '7' AND substr(h.id::text, 15, 1) = '7'
	  AND NOT EXISTS (SELECT 1 FROM workflow_ref w WHERE w.tenant_id = o.tenant_id AND w.outbox_id = o.id)`

// CI-A9 across A, the deploy and B, and the R16 seeding: a version published
// between A and the deploy gets its backfilled entry from B; one published
// after the deploy keeps the entry the publish path wrote; every tenant ends
// with one head and one start row; a tenant onboarding already seeded keeps
// its own; and nothing is left of the migrator's access path.
func TestP532MigrationB(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, auditBackfillMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", auditBackfillMigration, err)
	}
	super := superNava(t)
	seedBackfillVersions(t, super)
	if err := migrateOnlyFS(t, m, migrationsThrough(t, auditBackfillMigration)); err != nil {
		t.Fatalf("migrate through %s: %v", auditBackfillMigration, err)
	}
	if _, err := migrateBeforeB(t, m); err != nil {
		t.Fatalf("migrate to before %s: %v", migrationB, err)
	}

	// Between A and the deploy: tenant B publishes version 2 with no entry.
	gap := uuid.NewString()
	exec(t, super, insertVersionRaw, tenantB, gap, tenantB, 2, testSchema, schemaSum(testSchema), "compatible", tenantB)
	// After the deploy: tenant A publishes version 4 and the publish path
	// writes its entry (not backfilled) in the same transaction.
	deployed := uuid.NewString()
	exec(t, super, `WITH v AS (`+insertVersionRaw+` RETURNING tenant_id, definition_id, version, schema_sha256)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
			version_before, version_after, actor_principal_id, source, content_sha256)
		SELECT tenant_id, $9::uuid, 'config', 'publish', 'entity_definition', definition_id, definition_id,
			version - 1, version, $8, 'user', schema_sha256 FROM v`,
		tenantA, uuid.NewString(), tenantA, 4, testSchema, schemaSum(testSchema), "compatible", tenantA, deployed)

	// A tenant onboarding before B: its own head and start row.
	s := onboardStore(t, 4)
	onboarded := uuid.New()
	if _, err := s.Begin(t.Context(), onboarded, []byte("h"), nil); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedAuditChain(t.Context(), onboarded); err != nil {
		t.Fatal(err)
	}
	const lineage = "SELECT h.id::text || '/' || o.id::text FROM audit_chain_head h JOIN outbox o ON o.tenant_id = h.tenant_id AND o.kind = 'audit.seal' WHERE h.tenant_id = $1"
	before := queryString(t, super, lineage, onboarded)
	// A tenant still provisioning when B runs: B seeds it, and onboarding's
	// SeedAuditChain after B writes nothing more (Security, PR #492).
	provisioning := uuid.New()
	if _, err := s.Begin(t.Context(), provisioning, []byte("h"), nil); err != nil {
		t.Fatal(err)
	}

	migrateAll(t, m)

	if bad := queryString(t, super, entriesPerVersion); bad != "" {
		t.Errorf("versions without exactly one entry after B: %s", bad)
	}
	if n := countRows(t, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND id = $2 AND backfilled", tenantB, backfillID(tenantB, gap)); n != 1 {
		t.Errorf("version published between A and the deploy: %d backfilled entries, want 1", n)
	}
	if n := countRows(t, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND id = $2 AND NOT backfilled", tenantA, deployed); n != 1 {
		t.Errorf("version published after the deploy: %d publish-path entries, want 1 (B must not add its own)", n)
	}

	for _, tenant := range []string{tenantA, tenantB, tenantC, onboarded.String()} {
		if n := countRows(t, "SELECT count(*) FROM audit_chain_head WHERE tenant_id = $1", tenant); n != 1 {
			t.Errorf("tenant %s: %d heads, want 1", tenant, n)
		}
		if n := countRows(t, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND kind = 'audit.seal'", tenant); n != 1 {
			t.Errorf("tenant %s: %d seal rows, want 1", tenant, n)
		}
		if n := countRows(t, sealShape, tenant); n != 1 {
			t.Errorf("tenant %s: head and start row are not the onboarding/C5 shape", tenant)
		}
	}
	seededByB := queryString(t, super, lineage, provisioning)
	if err := s.SeedAuditChain(t.Context(), provisioning); err != nil {
		t.Fatalf("SeedAuditChain after B for a tenant B seeded: %v", err)
	}
	if n := countRows(t, "SELECT count(*) FROM audit_chain_head WHERE tenant_id = $1", provisioning); n != 1 {
		t.Errorf("provisioning tenant: %d heads after B and SeedAuditChain, want 1", n)
	}
	if n := countRows(t, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND kind = 'audit.seal'", provisioning); n != 1 {
		t.Errorf("provisioning tenant: %d seal rows after B and SeedAuditChain, want 1", n)
	}
	if n := countRows(t, sealShape, provisioning); n != 1 {
		t.Errorf("provisioning tenant: head and start row are not the onboarding/C5 shape")
	}
	if after := queryString(t, super, lineage, provisioning); after != seededByB {
		t.Errorf("provisioning tenant's head/start row changed from B's %s to %s", seededByB, after)
	}
	if after := queryString(t, super, lineage, onboarded); after != before {
		t.Errorf("onboarded tenant's head/start row changed from %s to %s", before, after)
	}

	if got := queryString(t, super, `SELECT string_agg(t.tgrelid::regclass::text || ':' || t.tgname, ',' ORDER BY t.tgname COLLATE "C")
		FROM pg_trigger t JOIN pg_proc p ON p.oid = t.tgfoid WHERE p.proname = 'audit_commit_check'`); got !=
		"entity_definition_versions:entity_definition_versions_audit_commit_check,entity_definitions:entity_definitions_audit_commit_check,records:records_audit_commit_check" {
		t.Errorf("audit_commit_check triggers after B: %s", got)
	}
	// R26.3, and both functions are idempotent: a re-run adds nothing.
	if n := countRows(t, "SELECT count(*) FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles)"); n != 0 {
		t.Errorf("%d nava_migrator policies left after B, want 0", n)
	}
	mig := asRole(t, "nava_migrator")
	for _, fn := range []string{"audit_backfill_s8", "audit_seed_chain_heads"} {
		if got := queryString(t, mig, "SELECT "+fn+"()::text"); got != "0" {
			t.Errorf("re-run of %s added %s, want 0", fn, got)
		}
	}
	if n := countRows(t, "SELECT count(*) FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles)"); n != 0 {
		t.Errorf("%d nava_migrator policies left after the re-runs, want 0", n)
	}
	// No runtime role can run the seeder.
	_, err := asRole(t, "nava_app").Exec(context.Background(), "SELECT audit_seed_chain_heads()")
	wantSQLState(t, "nava_app runs audit_seed_chain_heads", err, "42501")
}

// TICK-449 grooming: B's start row is the row outbox.EnqueueSeal writes, and
// B's head and start row equal what onboarding's SeedAuditChain writes, column
// by column, apart from the ids and the clock.
func TestP532SealRowMatchesEnqueueSeal(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateBeforeB(t, m); err != nil {
		t.Fatalf("migrate to before %s: %v", migrationB, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	migrateAll(t, m) // B seeds tenant A

	// The C5 producer for a tenant created after B, at head version 1.
	exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenantB)
	exec(t, super, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, gen_random_uuid())", tenantB)
	ctx := context.Background()
	var enqueued uuid.UUID
	if err := inTenantTx(asRole(t, "nava_app"), tenantB, func(tx pgx.Tx) error {
		var err error
		enqueued, err = outbox.EnqueueSeal(ctx, tx, 1)
		return err
	}); err != nil {
		t.Fatalf("EnqueueSeal: %v", err)
	}
	// Onboarding's path for a tenant created after B.
	s := onboardStore(t, 4)
	onboarded := uuid.New()
	if _, err := s.Begin(ctx, onboarded, []byte("h"), nil); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedAuditChain(ctx, onboarded); err != nil {
		t.Fatal(err)
	}

	// Every outbox column except the ids, which C5 ties to the row's own id,
	// and the clock, which only has to be the transaction's now().
	const sealRow = `SELECT (to_jsonb(o) - ARRAY['id', 'tenant_id', 'target_key', 'causation_id', 'created_at', 'available_at'])::text
		|| ' self=' || (o.target_key = 'audit.seal:' || o.id::text AND o.causation_id = o.id)::text
		|| ' now=' || (o.available_at = o.created_at)::text
		|| ' v7=' || (substr(o.id::text, 15, 1) = '7')::text
		FROM outbox o WHERE o.tenant_id = $1 AND o.kind = 'audit.seal'`
	fromB := queryString(t, super, sealRow, tenantA)
	if got := queryString(t, super, sealRow, tenantB); got != fromB {
		t.Errorf("seal row: EnqueueSeal wrote\n  %s\nmigration B wrote\n  %s", got, fromB)
	}
	if got := queryString(t, super, sealRow, onboarded); got != fromB {
		t.Errorf("seal row: onboarding wrote\n  %s\nmigration B wrote\n  %s", got, fromB)
	}
	if n := countRows(t, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND id = $2", tenantB, enqueued); n != 1 {
		t.Fatalf("EnqueueSeal's row %s not found", enqueued)
	}

	const headRow = `SELECT (to_jsonb(h) - ARRAY['tenant_id', 'id', 'next_seal_outbox_id', 'next_seal_at'])::text
		|| ' points=' || (h.next_seal_outbox_id = o.id)::text
		|| ' now=' || (h.next_seal_at = o.created_at)::text
		|| ' v7=' || (substr(h.id::text, 15, 1) = '7')::text
		FROM audit_chain_head h JOIN outbox o ON o.tenant_id = h.tenant_id AND o.kind = 'audit.seal'
		WHERE h.tenant_id = $1`
	if got, want := queryString(t, super, headRow, onboarded), queryString(t, super, headRow, tenantA); got != want {
		t.Errorf("head: onboarding wrote\n  %s\nmigration B wrote\n  %s", got, want)
	}
	// B's UUIDv7 time prefix is the transaction's clock, to the millisecond.
	if n := countRows(t, `SELECT count(*) FROM outbox WHERE tenant_id = $1
		AND abs(extract(epoch FROM created_at) * 1000 - ('x' || translate(substr(id::text, 1, 13), '-', ''))::bit(48)::bigint) < 60000`, tenantA); n != 1 {
		t.Errorf("migration B's seal id does not carry a UUIDv7 timestamp near created_at")
	}
}

// CI-A2 (config): after B, a raw write to either entity-definition table by
// a writer role fails at COMMIT, naming the trigger; with the entry the
// publish path writes in the same transaction, it commits.
func TestP532RawConfigWriteFailsAtCommit(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX})
	seedEntityTypes(t, super, tenantA)
	app := asRole(t, "nava_app")
	ctx := context.Background()
	const entry = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
		version_before, version_after, actor_principal_id, source, content_sha256)
		VALUES ($1, gen_random_uuid(), 'config', 'publish', 'entity_definition', $1, $1, 1, 2, $1, 'user', $2)`

	for _, tc := range []struct {
		name, trigger string
		writes        []string
	}{
		{"insert a version", "entity_definition_versions_audit_commit_check", []string{insertVersionRaw}},
		{"move current_version", "entity_definitions_audit_commit_check",
			[]string{"UPDATE entity_definitions SET current_version = 2 WHERE id = $1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := func(withEntry bool) error {
				return inTenantTx(app, tenantA, func(tx pgx.Tx) error {
					for _, w := range tc.writes {
						var err error
						if w == insertVersionRaw {
							_, err = tx.Exec(ctx, w, tenantA, uuid.NewString(), tenantA, 2, testSchema, schemaSum(testSchema), "compatible", tenantA)
						} else {
							_, err = tx.Exec(ctx, w, tenantA)
						}
						if err != nil {
							return err
						}
					}
					if withEntry {
						_, err := tx.Exec(ctx, entry, tenantA, schemaSum(testSchema))
						return err
					}
					return nil
				})
			}
			err := run(false)
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "23514" || pg.ConstraintName != tc.trigger {
				t.Fatalf("raw write as nava_app: %v, want SQLSTATE 23514 from %s at COMMIT", err, tc.trigger)
			}
			if err := run(true); err != nil {
				t.Fatalf("write with its entry: %v", err)
			}
		})
	}

	// nava_onboard, the other writer: a raw type insert for its own
	// provisioning tenant fails at COMMIT too.
	seedIdentity(t, super, tenantX)
	_, err := execIn(asRole(t, audited.OnboardRole), tenantX, insertTypeRaw, tenantX, "ticket", "TCK")
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "23514" || pg.ConstraintName != "entity_definitions_audit_commit_check" {
		t.Fatalf("raw type insert as nava_onboard: %v, want SQLSTATE 23514 at COMMIT", err)
	}
}

// R27: B takes its six locks in the fixed order before any other work, and a
// lock that times out is retried, holding none of the set while it waits.
// The blocker holds outbox, the last of the six, for longer than one
// lock_timeout (5 s), so B succeeds only by retrying.
func TestP532LocksInOrderAndRetries(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateBeforeB(t, m); err != nil {
		t.Fatalf("migrate to before %s: %v", migrationB, err)
	}
	ctx := context.Background()
	blocker := superNava(t)
	blockTx, err := blocker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blockTx.Exec(ctx, "LOCK TABLE outbox IN ACCESS SHARE MODE"); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := migrateFS(t, m, db.Migrations())
		done <- err
	}()

	// The migrating backend waits on outbox holding the other five.
	watch := superNava(t)
	const waiting = `SELECT pid FROM pg_locks WHERE relation = 'outbox'::regclass AND mode = 'AccessExclusiveLock' AND NOT granted`
	const held = `SELECT string_agg(relation::regclass::text, ',' ORDER BY relation::regclass::text COLLATE "C") FROM pg_locks
		WHERE pid = $1 AND locktype = 'relation' AND mode = 'AccessExclusiveLock' AND granted
		  AND relation::regclass::text IN ('tenants', 'entity_definitions', 'entity_definition_versions', 'audit_log', 'audit_chain_head', 'outbox')`
	var pid int32
	deadline := time.Now().Add(30 * time.Second)
	for pid == 0 {
		if time.Now().After(deadline) {
			t.Fatal("migration B never waited on outbox")
		}
		if err := watch.QueryRow(ctx, waiting).Scan(&pid); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got := queryString(t, watch, held, pid); got != "audit_chain_head,audit_log,entity_definition_versions,entity_definitions,tenants" { // C order
		t.Errorf("while waiting on outbox, B holds %q, want the other five", got)
	}
	// Outlast one lock_timeout; between attempts B holds none of the six.
	sawGap := false
	for end := time.Now().Add(7 * time.Second); time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
		if queryString(t, watch, `SELECT coalesce((`+held+`), '')`, pid) == "" {
			sawGap = true
		}
	}
	if err := blockTx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("migration B after the blocker let go: %v", err)
	}
	if !sawGap {
		t.Error("B never released the set between attempts: the first attempt did not time out and retry")
	}
	if n := countRows(t, "SELECT count(*) FROM pg_trigger WHERE tgname = 'entity_definitions_audit_commit_check'"); n != 1 {
		t.Errorf("entity_definitions trigger count %d after the retried run, want 1", n)
	}
}

// Security (PR #492): no runtime writer can switch the commit check off.
// ALTER TABLE ... DISABLE TRIGGER needs the table owner (nava_migrator), so
// nava_app and nava_onboard get 42501 on either table.
func TestP532RuntimeRolesCannotDisableCommitCheck(t *testing.T) {
	migratedPlatform(t)
	seedTenants(t, superNava(t), []string{tenantA}, []string{tenantX})
	for _, role := range []string{"nava_app", audited.OnboardRole} {
		conn := asRole(t, role)
		tenant := tenantA
		if role == audited.OnboardRole {
			tenant = tenantX
		}
		for _, tr := range []struct{ table, name string }{
			{"entity_definitions", "entity_definitions_audit_commit_check"},
			{"entity_definition_versions", "entity_definition_versions_audit_commit_check"},
		} {
			for _, verb := range []string{"DISABLE", "ENABLE"} {
				q := "ALTER TABLE " + tr.table + " " + verb + " TRIGGER " + tr.name
				_, err := execIn(conn, tenant, q)
				wantSQLState(t, role+": "+q, err, "42501")
			}
			if _, err := execIn(conn, tenant, "DROP TRIGGER "+tr.name+" ON "+tr.table); err == nil {
				t.Errorf("%s dropped %s", role, tr.name)
			} else {
				wantSQLState(t, role+": DROP TRIGGER "+tr.name, err, "42501")
			}
		}
	}
	if got := queryString(t, superNava(t), `SELECT string_agg(tgname || ':' || tgenabled::text, ',' ORDER BY tgname COLLATE "C")
		FROM pg_trigger WHERE tgname IN ('entity_definitions_audit_commit_check', 'entity_definition_versions_audit_commit_check')`); got !=
		"entity_definition_versions_audit_commit_check:O,entity_definitions_audit_commit_check:O" {
		t.Errorf("triggers after the attempts: %s, want both present and enabled", got)
	}
}
