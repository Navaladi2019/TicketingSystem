//go:build integration

package dbtest

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// TICK-498 (AR-TICK-480-01): audit_log's idempotency key is unique per
// (tenant, key, actor), NULL actors not distinct.

const auditIdempotencyPerActorMigration = "20261103032000_audit_log_idempotency_key_per_actor.sql"

// insertAuditBy: tenant $1, record $2, idempotency key $3, actor $4 (NULL for
// a principal-less system writer).
const insertAuditBy = `INSERT INTO audit_log (tenant_id, id, record_id, action, actor_id, source, after, idempotency_key)
	VALUES ($1, gen_random_uuid(), $2, 'create', $4, 'workflow', '{}', $3)`

// Acceptance: in one tenant, two members each write under the same key; the
// same member twice, or no principal twice, still conflicts.
func TestTICK498IdempotencyKeyPerActor(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA) // a NULL-actor row keyed seed-key
	worker := asRole(t, "nava_worker")
	rec := pipelineRecord(tenantA)
	memberA, memberB := tenantX, tenantY // any two uuids: actor_id has no foreign key
	const key = "wf:shared:s1"

	if _, err := execIn(worker, tenantA, insertAuditBy, tenantA, rec, key, memberA); err != nil {
		t.Fatalf("member A's key: %v", err)
	}
	if _, err := execIn(worker, tenantA, insertAuditBy, tenantA, rec, key, memberB); err != nil {
		t.Fatalf("member B under member A's key: %v; want its own row, no conflict", err)
	}
	_, err := execIn(worker, tenantA, insertAuditBy, tenantA, rec, key, memberA)
	wantSQLState(t, "member A's key twice", err, "23505")
	_, err = execIn(worker, tenantA, insertAuditBy, tenantA, rec, "seed-key", nil)
	wantSQLState(t, "no principal, seed-key twice (NULLS NOT DISTINCT)", err, "23505")
	if _, err := execIn(worker, tenantA, insertAuditBy, tenantA, rec, "seed-key", memberA); err != nil {
		t.Fatalf("member A under the system writer's key: %v", err)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM audit_log WHERE idempotency_key = $1", key); got != "2" {
		t.Errorf("rows keyed %s = %s, want 2 (one per member)", key, got)
	}
	if got := sqlStateConstraint(t, worker, rec, key, memberB); got != "audit_log_idempotency_key" {
		t.Errorf("the 23505 names constraint %q, want audit_log_idempotency_key (store.InsertAudit maps that name)", got)
	}
}

// Acceptance: the migration builds over existing keyed rows, leaves one valid
// index under the old name, and its definition is the per-actor one.
func TestTICK498MigrationOverExistingRows(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, auditIdempotencyPerActorMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", auditIdempotencyPerActorMigration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	exec(t, super, insertAuditBy, tenantA, pipelineRecord(tenantA), "k-before", tenantX)

	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	got := queryString(t, super, `SELECT string_agg(c.relname || ' valid=' || i.indisvalid || ' ' || pg_get_indexdef(i.indexrelid), E'\n' ORDER BY c.relname)
		FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
		WHERE i.indrelid = 'audit_log'::regclass AND c.relname LIKE 'audit_log_idempotency%'`)
	want := "audit_log_idempotency_key valid=true CREATE UNIQUE INDEX audit_log_idempotency_key ON public.audit_log USING btree (tenant_id, idempotency_key, actor_id) NULLS NOT DISTINCT WHERE (idempotency_key IS NOT NULL)"
	if got != want {
		t.Fatalf("idempotency indexes:\n%s\nwant:\n%s", got, want)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM audit_log WHERE idempotency_key IS NOT NULL"); got != "3" {
		t.Errorf("keyed rows after the migration = %s, want 3", got)
	}
}

// sqlStateConstraint writes a duplicate keyed row for actor in tenant A and
// returns the constraint its 23505 names.
func sqlStateConstraint(t *testing.T, conn *pgx.Conn, rec, key, actor string) string {
	t.Helper()
	_, err := execIn(conn, tenantA, insertAuditBy, tenantA, rec, key, actor)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23505" {
		t.Fatalf("duplicate keyed row: %v, want SQLSTATE 23505", err)
	}
	return pe.ConstraintName
}
