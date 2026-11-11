//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// TICK-626 (AR-TICK-480-01 on D-5-3): record_write_keys carries the writer's
// principal, is backfilled from the write's audit entry, and is unique per
// (tenant, key, principal), NULLs not distinct.

const (
	rwkActorMigration    = "20261108120000_record_write_keys_actor.sql"
	rwkActorKeyMigration = "20261108120100_record_write_keys_actor_key.sql"

	// insertKeyRaw: tenant $1, key $2, action $3, record $4, audit entry $5.
	// The definition is the tenant's seeded one (id = tenant id).
	insertKeyRaw = `INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id, number,
		version, schema_version, audit_entry_id) VALUES ($1, $2, $1, $3, $4, 'TCK0000001', 1, 1, $5)`
	// insertKeyBy is insertKeyRaw with actor_principal_id $6.
	insertKeyBy = `INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id, number,
		version, schema_version, audit_entry_id, actor_principal_id) VALUES ($1, $2, $1, $3, $4, 'TCK0000001', 1, 1, gen_random_uuid(), $5)`
	// insertRecordEntryBy: a record entry of tenant $1, id $2, subject $3,
	// by principal $4.
	insertRecordEntryBy = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after,
		actor_principal_id, source)
		VALUES ($1, $2, 'record', 'update', 'record', $3, 1, $4, 'user')`
	// insertRecordEntrySystem: the same by a system writer, no principal.
	insertRecordEntrySystem = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after,
		actor_system, source)
		VALUES ($1, $2, 'record', 'update', 'record', $3, 1, 'workflow', 'workflow')`
)

// Acceptance: keys written before the migration get the principal of their
// audit entry; a system writer's stays NULL; a create whose entry is gone
// (audit_log_v0) gets the record's creator; an update whose entry is gone
// stays NULL; a key never takes another tenant's entry. Then the per-principal
// index replaces the primary key, and the migrations go down and up again.
func TestTICK626RecordWriteKeysActorBackfill(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, rwkActorMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", rwkActorMigration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB) // principal and created_by = the tenant id
	recA, recB := pipelineRecord(tenantA), pipelineRecord(tenantB)
	const (
		entryMember = "0190a000-0000-7000-8000-00000000e001"
		entrySystem = "0190a000-0000-7000-8000-00000000e002"
		entryB      = "0190a000-0000-7000-8000-00000000e003"
		goneEntry   = "0190a000-0000-7000-8000-00000000e0ff"
	)
	exec(t, super, insertRecordEntryBy, tenantA, entryMember, recA, tenantA)
	exec(t, super, insertRecordEntrySystem, tenantA, entrySystem, recA)
	exec(t, super, insertRecordEntryBy, tenantB, entryB, recB, tenantB)
	exec(t, super, insertKeyRaw, tenantA, "k-member", "update", recA, entryMember)
	exec(t, super, insertKeyRaw, tenantA, "k-system", "update", recA, entrySystem)
	exec(t, super, insertKeyRaw, tenantA, "k-v0-create", "create", recA, goneEntry)
	exec(t, super, insertKeyRaw, tenantA, "k-v0-update", "update", recA, goneEntry)
	exec(t, super, insertKeyRaw, tenantB, "k-member", "update", recB, entryB)
	// Tenant B's key names tenant A's entry id: it must not take A's actor.
	exec(t, super, insertKeyRaw, tenantB, "k-cross", "update", recB, entryMember)

	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	got := queryString(t, super, `SELECT string_agg(tenant_id || ' ' || idempotency_key || ' ' || coalesce(actor_principal_id::text, 'NULL'), E'\n'
		ORDER BY tenant_id, idempotency_key) FROM record_write_keys`)
	want := tenantA + " k-member " + tenantA + "\n" +
		tenantA + " k-system NULL\n" +
		tenantA + " k-v0-create " + tenantA + "\n" +
		tenantA + " k-v0-update NULL\n" +
		tenantB + " k-cross NULL\n" +
		tenantB + " k-member " + tenantB
	if got != want {
		t.Errorf("backfilled actors:\n%s\nwant:\n%s", got, want)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles)`); got != "0" {
		t.Errorf("%s nava_migrator policies left over", got)
	}
	if got := queryString(t, super, `SELECT coalesce(string_agg(c.relname || ' valid=' || i.indisvalid || ' ' || pg_get_indexdef(i.indexrelid), E'\n' ORDER BY c.relname), '')
		FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
		WHERE i.indrelid = 'record_write_keys'::regclass`); got != "record_write_keys_actor_key valid=true CREATE UNIQUE INDEX record_write_keys_actor_key ON public.record_write_keys USING btree (tenant_id, idempotency_key, actor_principal_id) NULLS NOT DISTINCT" {
		t.Errorf("record_write_keys indexes:\n%s", got)
	}

	// Down past both migrations, then up again: the tenant-wide primary key comes
	// back, the column goes, and the re-run fills it again.
	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// DownTo the version before both, not two Down steps: a later migration
	// (20261108124249_acl_tables.sql, TICK-615) sorts after them.
	if _, err := p.DownTo(ctx, 20261108115959); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := queryString(t, super, `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'record_write_keys'::regclass AND conname = 'record_write_keys_pkey'`); got != "PRIMARY KEY (tenant_id, idempotency_key)" {
		t.Errorf("after down: record_write_keys_pkey = %q", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_attribute
		WHERE attrelid = 'record_write_keys'::regclass AND attname = 'actor_principal_id' AND NOT attisdropped`); got != "0" {
		t.Errorf("after down: actor_principal_id still exists")
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM record_write_keys WHERE actor_principal_id IS NOT NULL`); got != "3" {
		t.Errorf("after up again: %s keys with an actor, want 3", got)
	}
}

// Acceptance: two principals of one tenant each hold the same key; the same
// principal twice, or no principal twice, is 23505 on
// record_write_keys_actor_key, the name the store maps to ErrDuplicateKey.
func TestTICK626RecordWriteKeysUniquePerPrincipal(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	rec := pipelineRecord(tenantA)
	memberA, memberB := tenantX, tenantY // any two uuids: the column has no foreign key
	const key = "wf:shared:s1"
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		k := key + ":" + role
		for _, actor := range []any{memberA, memberB, nil} {
			if _, err := execIn(conn, tenantA, insertKeyBy, tenantA, k, "create", rec, actor); err != nil {
				t.Fatalf("%s: actor %v claims %s: %v; want its own row", role, actor, k, err)
			}
		}
		for _, actor := range []any{memberA, nil} {
			_, err := execIn(conn, tenantA, insertKeyBy, tenantA, k, "create", rec, actor)
			if code, con := pgErrDetail(err); code != "23505" || con != "record_write_keys_actor_key" {
				t.Errorf("%s: actor %v claims %s twice: %v, want 23505 on record_write_keys_actor_key", role, actor, k, err)
			}
		}
	}
}
