//go:build integration

package dbtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
)

// Security review of PR #503 (TICK-606), R26.3 on the failure path. The
// migrator owns B's lock_timeout retry. When every attempt times out, the
// run fails. Each failed attempt must leave nothing behind:
//   - no nava_migrator policy (R26.3);
//   - no change to any grant on the six R27 tables;
//   - no B version row, trigger or seed function.
//
// So a later run starts clean. TestP532LocksInOrderAndRetries covers the
// success path. This test covers the run that gives up.
func TestP532RetryExhaustedLeavesNoPolicyOrGrant(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateBeforeB(t, m); err != nil {
		t.Fatalf("migrate to before %s: %v", migrationB, err)
	}
	super := superNava(t)
	const policies = `SELECT count(*) FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles)`
	const acls = `SELECT string_agg(c.relname || '=' || coalesce(c.relacl::text, ''), ';' ORDER BY c.relname)
		FROM pg_class c WHERE c.relnamespace = 'public'::regnamespace
		  AND c.relname IN ('tenants', 'entity_definitions', 'entity_definition_versions', 'audit_log', 'audit_chain_head', 'outbox')`
	const bVersion = `SELECT count(*) FROM goose_db_version WHERE version_id = 20261107053707`
	if n := countRows(t, policies); n != 0 {
		t.Fatalf("%d nava_migrator policies before B, want 0", n)
	}
	aclsBefore := queryString(t, super, acls)

	ctx := context.Background()
	blocker := superNava(t)
	blockTx, err := blocker.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blockTx.Rollback(ctx) }()
	if _, err := blockTx.Exec(ctx, "LOCK TABLE outbox IN ACCESS SHARE MODE"); err != nil {
		t.Fatal(err)
	}

	log, _ := newLog()
	_, err = migrate.Run(ctx, roleConfig(t, m.Owner(), m.PlatformDatabase), db.Migrations(), m,
		auditedManifest(t), globalTables(t), log,
		migrate.Options{LockRetrySeconds: 1, LockTimeout: 500 * time.Millisecond, DDLAttempts: 2, DDLBackoff: 100 * time.Millisecond})
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "55P03" {
		t.Fatalf("migrate with outbox held throughout: err %v, want 55P03 after the retries", err)
	}

	if n := countRows(t, policies); n != 0 {
		t.Errorf("%d nava_migrator policies left after B gave up, want 0 (R26.3)", n)
	}
	if got := queryString(t, super, acls); got != aclsBefore {
		t.Errorf("grants on the R27 tables changed after B gave up:\n before %s\n after  %s", aclsBefore, got)
	}
	if n := countRows(t, bVersion); n != 0 {
		t.Errorf("B recorded as applied after it gave up: %d rows", n)
	}
	if n := countRows(t, `SELECT count(*) FROM pg_trigger WHERE tgname IN
		('entity_definitions_audit_commit_check', 'entity_definition_versions_audit_commit_check')`); n != 0 {
		t.Errorf("%d commit-check triggers left after B gave up, want 0", n)
	}
	if n := countRows(t, `SELECT count(*) FROM pg_proc WHERE proname IN ('audit_seed_chain_heads', 'audit_seed_uuidv7')`); n != 0 {
		t.Errorf("%d B seed functions left after B gave up, want 0", n)
	}

	// Once the blocker lets go, a fresh run applies B and still leaves no policy.
	if err := blockTx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate after the blocker let go: %v", err)
	}
	if n := countRows(t, bVersion); n != 1 {
		t.Errorf("B version rows after the clean re-run: %d, want 1", n)
	}
	if n := countRows(t, policies); n != 0 {
		t.Errorf("%d nava_migrator policies after the clean re-run, want 0 (R26.3)", n)
	}
}
