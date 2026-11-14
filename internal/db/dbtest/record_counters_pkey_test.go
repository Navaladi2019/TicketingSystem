//go:build integration

package dbtest

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
)

// S18 P6-05 (TICK-670), ADR-005 §4.2 step M3: record_counters_pkey becomes
// (tenant_id, entity_definition_id, scope_key), promoted from
// record_counters_scoped_key. Runbook: docs/data/record-counters-scope-key.md.

const (
	// m3Version is m3Migration's version; m3Before the one just before it.
	m3Version = 20261113034010
	m3Before  = m3Version - 1

	counterPrimaryKey = `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'record_counters'::regclass AND contype = 'p'`
	counterFKey = `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'record_counters'::regclass AND conname = 'record_counters_entity_definition_fkey'`
	m3Applied = `SELECT count(*) FROM goose_db_version WHERE version_id = 20261113034010`

	pkeyTwoColumns   = "PRIMARY KEY (tenant_id, entity_definition_id)"
	pkeyThreeColumns = "PRIMARY KEY (tenant_id, entity_definition_id, scope_key)"
	fkeyDef          = "FOREIGN KEY (tenant_id, entity_definition_id) REFERENCES entity_definitions(tenant_id, id)"
)

// postM3Indexes is counterIndexes after M3: one unique index, the key.
var postM3Indexes = strings.Join([]string{
	"record_counters_period_end valid=true btree (period_end) WHERE (period_end IS NOT NULL)",
	"record_counters_pkey valid=true btree (tenant_id, entity_definition_id, scope_key)",
}, "\n")

// preM3Indexes is counterIndexes before M3 (and after its Down).
var preM3Indexes = strings.Join([]string{
	"record_counters_period_end valid=true btree (period_end) WHERE (period_end IS NOT NULL)",
	"record_counters_pkey valid=true btree (tenant_id, entity_definition_id)",
	"record_counters_scoped_key valid=true btree (tenant_id, entity_definition_id, scope_key)",
}, "\n")

// seededBefore is a fresh cluster migrated, with the grants, by
// fsys(m3Migration) (migrationsBefore or migrationsThrough), seeded with
// tenants A and B and a default-scope counter each, A's at next 43.
func seededBefore(t *testing.T, fsys func(*testing.T, string) fs.FS) *grants.Manifest {
	t.Helper()
	m := freshCluster(t)
	if _, err := migrateFS(t, m, fsys(t, m3Migration)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	exec(t, super, "UPDATE record_counters SET next = 43 WHERE tenant_id = $1", tenantA)
	return m
}

// Pinned test 3, second half (ADR-005 §4.2): after M3 the key is
// (tenant_id, entity_definition_id, scope_key) under the old name, the FK is
// intact, R1's 3-column upsert succeeds, R0's 2-column ON CONFLICT fails with
// 42P10, and a non-empty scope_key row coexists with the type's ” row, as
// nava_app and nava_worker. The existing counters keep their next.
func TestP605CounterKeyContract(t *testing.T) {
	m := seededBefore(t, migrationsBefore)
	super := superNava(t)
	fileBefore := queryString(t, super, `SELECT pg_relation_filenode('record_counters')::text`)
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate through M3: %v", err)
	}

	if got := queryString(t, super, counterPrimaryKey); got != pkeyThreeColumns {
		t.Errorf("primary key after M3: %s, want %s", got, pkeyThreeColumns)
	}
	if got := queryString(t, super, counterIndexes); got != postM3Indexes {
		t.Errorf("record_counters indexes after M3:\n%s\nwant:\n%s", got, postM3Indexes)
	}
	if got := queryString(t, super, counterFKey); got != fkeyDef {
		t.Errorf("FK after M3: %s, want %s", got, fkeyDef)
	}
	if got := queryString(t, super, `SELECT pg_relation_filenode('record_counters')::text`); got != fileBefore {
		t.Errorf("record_counters filenode %s -> %s: M3 rewrote the table", fileBefore, got)
	}
	// What S18's publish gate reads (ADR-005 §4.2): does the key cover scope_key?
	if !queryBool(t, super, `SELECT EXISTS (SELECT 1 FROM pg_constraint c JOIN pg_attribute a
		ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
		WHERE c.conname = 'record_counters_pkey' AND c.conrelid = 'record_counters'::regclass AND a.attname = 'scope_key')`) {
		t.Error("record_counters_pkey does not cover scope_key after M3")
	}

	want := int64(42)
	for i, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		want++
		if got, err := takeIn(conn, tenantA, takeNumberR1, tenantA, tenantA); err != nil || got != want {
			t.Fatalf("%s: 3-column upsert after M3: %d, %v; want %d, nil", role, got, err, want)
		}
		_, err := takeIn(conn, tenantA, takeNumberR0, tenantA, tenantA)
		wantSQLState(t, role+": 2-column ON CONFLICT after M3", err, "42P10")

		scope := []string{"e1|d:202611", "e1|f:BLR"}[i]
		if got, err := takeIn(conn, tenantA, takeScoped, tenantA, tenantA, scope); err != nil || got != 1 {
			t.Fatalf("%s: first take of scope %q: %d, %v; want 1, nil", role, scope, got, err)
		}
		if got, err := takeIn(conn, tenantA, takeScoped, tenantA, tenantA, scope); err != nil || got != 2 {
			t.Fatalf("%s: second take of scope %q: %d, %v; want 2, nil", role, scope, got, err)
		}
	}
	if got := queryString(t, super, `SELECT string_agg('[' || scope_key || ']=' || next, ',' ORDER BY scope_key)
		FROM record_counters WHERE tenant_id = $1`, tenantA); got != "[]=45,[e1|d:202611]=3,[e1|f:BLR]=3" {
		t.Errorf("tenant A counters = %s, want []=45,[e1|d:202611]=3,[e1|f:BLR]=3", got)
	}
	if got := queryString(t, super, `SELECT string_agg('[' || scope_key || ']=' || next, ',') FROM record_counters WHERE tenant_id = $1`, tenantB); got != "[]=2" {
		t.Errorf("tenant B counters = %s, want []=2", got)
	}

	// RLS holds on the new key at a non-empty scope.
	_, err := takeIn(asRole(t, "nava_app"), tenantB, takeScoped, tenantA, tenantA, "e1|d:202611")
	wantSQLState(t, "tenant B takes from tenant A's scoped counter", err, "42501")
	_, err = execIn(asRole(t, "nava_worker"), tenantA, `INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES ($1, $2, 'e1|d:202611', 2)`, tenantA, tenantB)
	wantSQLState(t, "tenant A starts a scoped counter on tenant B's type", err, "23503")
}

// runM3 runs the migrator over the cluster m with every migration (M3 the
// only one pending) with production retry settings.
func runM3(t *testing.T, m *grants.Manifest) error {
	t.Helper()
	log, _ := newLog()
	_, err := migrate.Up(context.Background(), roleConfig(t, m.Owner(), m.PlatformDatabase),
		db.Migrations(), log, migrate.Options{LockRetrySeconds: 1})
	return err
}

// holdCounterLock takes ACCESS SHARE on record_counters in a transaction of
// its own, which conflicts with M3's ACCESS EXCLUSIVE, and releases it after
// hold. It returns a channel closed once the lock is released.
func holdCounterLock(t *testing.T, hold time.Duration) <-chan struct{} {
	t.Helper()
	ctx := context.Background()
	conn := superNava(t)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "LOCK TABLE record_counters IN ACCESS SHARE MODE"); err != nil {
		t.Fatal(err)
	}
	released := make(chan struct{})
	go func() {
		defer close(released)
		time.Sleep(hold)
		_ = tx.Rollback(context.Background())
	}()
	t.Cleanup(func() { <-released })
	return released
}

// Acceptance: a session holding a conflicting lock for 3 s makes M3's first
// attempt time out (lock_timeout 2s); M3 retries in its DO block and then
// succeeds, in one migrator attempt.
func TestP605M3RetriesALockAndSucceeds(t *testing.T) {
	m := seededBefore(t, migrationsBefore)
	super := superNava(t)
	released := holdCounterLock(t, 3*time.Second)

	start := time.Now()
	if err := runM3(t, m); err != nil {
		t.Fatalf("M3 with the table locked for 3 s: %v, want success after a retry", err)
	}
	took := time.Since(start)
	select {
	case <-released:
	default:
		t.Fatalf("M3 finished in %v, before the 3 s lock was released", took)
	}
	if got := queryString(t, super, counterPrimaryKey); got != pkeyThreeColumns {
		t.Errorf("primary key: %s, want %s", got, pkeyThreeColumns)
	}
	if got := queryString(t, super, counterIndexes); got != postM3Indexes {
		t.Errorf("indexes:\n%s\nwant:\n%s", got, postM3Indexes)
	}
	if n := countRows(t, m3Applied); n != 1 {
		t.Errorf("M3 version rows = %d, want 1", n)
	}
}

// Acceptance: a lock held past all 6 attempts (the first and 5 retries,
// about 18 s) fails M3 with 55006, which the migrator does not retry, and
// leaves the table unchanged: the 2-column key, record_counters_scoped_key
// (the same relation), the FK, the counters, and no version row. Once the
// lock is gone a re-run applies M3.
func TestP605M3GivesUpCleanly(t *testing.T) {
	m := seededBefore(t, migrationsBefore)
	super := superNava(t)
	scopedOID := queryString(t, super, `SELECT to_regclass('record_counters_scoped_key')::oid::text`)
	countersBefore := queryString(t, super, `SELECT string_agg(tenant_id || '[' || scope_key || ']=' || next, ',' ORDER BY tenant_id) FROM record_counters`)
	released := holdCounterLock(t, 20*time.Second)

	start := time.Now()
	err := runM3(t, m)
	took := time.Since(start)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "55006" || !strings.Contains(pe.Message, "after 6 attempts") {
		t.Fatalf("M3 with the table locked throughout: %v, want 55006 after 6 attempts", err)
	}
	// 6 x 2 s lock waits + 6.2 s backoff, in one migrator attempt: the
	// migrator's own 55P03 retry did not run it again.
	if took < 18*time.Second || took > 23*time.Second {
		t.Errorf("M3 gave up after %v, want about 18.2 s (one migrator attempt)", took)
	}

	if got := queryString(t, super, counterPrimaryKey); got != pkeyTwoColumns {
		t.Errorf("primary key after the failed M3: %s, want %s", got, pkeyTwoColumns)
	}
	if got := queryString(t, super, counterIndexes); got != preM3Indexes {
		t.Errorf("indexes after the failed M3:\n%s\nwant:\n%s", got, preM3Indexes)
	}
	if got := queryString(t, super, `SELECT to_regclass('record_counters_scoped_key')::oid::text`); got != scopedOID {
		t.Errorf("record_counters_scoped_key oid %s -> %s", scopedOID, got)
	}
	if got := queryString(t, super, counterFKey); got != fkeyDef {
		t.Errorf("FK after the failed M3: %s", got)
	}
	if got := queryString(t, super, `SELECT string_agg(tenant_id || '[' || scope_key || ']=' || next, ',' ORDER BY tenant_id) FROM record_counters`); got != countersBefore {
		t.Errorf("counters after the failed M3: %s, want %s", got, countersBefore)
	}
	if n := countRows(t, m3Applied); n != 0 {
		t.Errorf("M3 recorded as applied after it gave up: %d rows", n)
	}

	<-released
	if err := runM3(t, m); err != nil {
		t.Fatalf("M3 once the lock is gone: %v", err)
	}
	if got := queryString(t, super, counterPrimaryKey); got != pkeyThreeColumns {
		t.Errorf("primary key after the re-run: %s", got)
	}
}

// Acceptance: M3 Down restores the S17 schema while every scope_key is ”,
// and Up again promotes the index again; once a non-empty scope_key row
// exists Down refuses with a message and changes nothing. Up is a no-op when
// the key already covers scope_key (a hand re-run).
func TestP605M3DownAndUp(t *testing.T) {
	m := seededBefore(t, migrationsThrough)
	super := superNava(t)
	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	// With P6-10's purge policies, which migrateFS applied out of order, so
	// goose knows every applied version.
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, withCounterPurge(t, migrationsThrough(t, m3Migration)),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if got := queryString(t, super, counterPrimaryKey); got != pkeyThreeColumns {
		t.Fatalf("primary key after M3: %s", got)
	}

	if _, err := p.DownTo(ctx, m3Before); err != nil {
		t.Fatalf("down with only default scopes: %v", err)
	}
	if got := queryString(t, super, counterPrimaryKey); got != pkeyTwoColumns {
		t.Errorf("primary key after down: %s, want %s", got, pkeyTwoColumns)
	}
	if got := queryString(t, super, counterIndexes); got != preM3Indexes {
		t.Errorf("indexes after down:\n%s\nwant:\n%s", got, preM3Indexes)
	}
	if got := queryString(t, super, counterFKey); got != fkeyDef {
		t.Errorf("FK after down: %s", got)
	}
	if n := countRows(t, `SELECT count(*) FROM pg_constraint WHERE conname = 'record_counters_m3_down_default_scope'`); n != 0 {
		t.Errorf("down left its check constraint behind")
	}
	if got := queryString(t, super, `SELECT string_agg(next::text, ',' ORDER BY tenant_id) FROM record_counters`); got != "43,2" {
		t.Errorf("next after down: %s, want 43,2", got)
	}

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := queryString(t, super, counterIndexes); got != postM3Indexes {
		t.Errorf("indexes after up again:\n%s", got)
	}

	// A hand re-run over the applied M3 changes nothing.
	pkeyOID := queryString(t, super, `SELECT to_regclass('record_counters_pkey')::oid::text`)
	rerun, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrationsWhile(t, func(n string) bool { return n == m3Migration }),
		goose.WithDisableVersioning(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rerun.Up(ctx); err != nil {
		t.Fatalf("re-run of M3: %v", err)
	}
	if got := queryString(t, super, `SELECT to_regclass('record_counters_pkey')::oid::text`); got != pkeyOID {
		t.Errorf("re-run of M3 replaced the key (oid %s -> %s)", pkeyOID, got)
	}

	// A scoped counter makes M3 irreversible.
	if _, err := takeIn(asRole(t, "nava_app"), tenantB, takeScoped, tenantB, tenantB, "e1|d:202611"); err != nil {
		t.Fatalf("scoped take: %v", err)
	}
	_, err = p.DownTo(ctx, m3Before)
	if err == nil || sqlState(err) != "55000" || !strings.Contains(err.Error(), "M3 down refused: record_counters holds rows with a non-empty scope_key") {
		t.Fatalf("down with a scoped counter: %v, want 55000 M3 down refused", err)
	}
	if got := queryString(t, super, counterPrimaryKey); got != pkeyThreeColumns {
		t.Errorf("primary key after the refused down: %s", got)
	}
	if got := queryString(t, super, counterIndexes); got != postM3Indexes {
		t.Errorf("indexes after the refused down:\n%s", got)
	}
	if n := countRows(t, m3Applied); n != 1 {
		t.Errorf("M3 version rows after the refused down = %d, want 1", n)
	}
}
