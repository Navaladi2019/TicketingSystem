//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// P3-36f (TICK-659, PLAN-003 D-3-9): entity_definitions.numbering_mode, text
// NOT NULL DEFAULT 'gaps', CHECK IN ('gaps', 'gapless'), changeable by
// nava_app under the audit commit check.

const (
	numberingModeMigration = "20261110040000_entity_definitions_numbering_mode.sql"
	// numberingModeBefore is the version just before both migrations.
	numberingModeBefore = 20261110035959
	// setMode sets tenant $1's type's mode to $2 and writes its config
	// entry, so the commit passes the check.
	setMode = "UPDATE entity_definitions SET numbering_mode = $2, updated_at = now() WHERE tenant_id = $1"
)

// commitCheckColumns is the UPDATE OF list of entity_definitions' commit check.
func commitCheckColumns(t *testing.T, conn *pgx.Conn) string {
	t.Helper()
	return queryString(t, conn, `SELECT coalesce(string_agg(a.attname, ',' ORDER BY a.attname), '')
		FROM pg_trigger tr CROSS JOIN unnest(tr.tgattr::int2[]) u(attnum)
		JOIN pg_attribute a ON a.attrelid = tr.tgrelid AND a.attnum = u.attnum
		WHERE tr.tgrelid = 'entity_definitions'::regclass AND tr.tgname = 'entity_definitions_audit_commit_check'`)
}

// Acceptance: the migration adds the column to a table that already has
// types without rewriting it, every existing type reads gaps, the CHECK ends
// validated, the commit check covers the column, and the migrations go down
// and up again.
func TestP336fNumberingModeMigration(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, numberingModeMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", numberingModeMigration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	fileBefore := queryString(t, super, `SELECT pg_relation_filenode('entity_definitions')::text`)

	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// A rewrite gives the table a new file; a constant default does not.
	if got := queryString(t, super, `SELECT pg_relation_filenode('entity_definitions')::text`); got != fileBefore {
		t.Errorf("entity_definitions filenode %s -> %s: the migration rewrote the table", fileBefore, got)
	}
	if got := queryString(t, super, `SELECT string_agg(tenant_id || ' ' || numbering_mode, ',' ORDER BY tenant_id) FROM entity_definitions`); got != tenantA+" gaps,"+tenantB+" gaps" {
		t.Errorf("existing types read %q, want gaps for both", got)
	}
	if got := queryString(t, super, `SELECT format_type(atttypid, atttypmod) || ' notnull=' || attnotnull || ' default=' || pg_get_expr(d.adbin, d.adrelid)
		FROM pg_attribute a JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE a.attrelid = 'entity_definitions'::regclass AND a.attname = 'numbering_mode'`); got != "text notnull=true default='gaps'::text" {
		t.Errorf("numbering_mode column: %s", got)
	}
	if got := queryString(t, super, `SELECT convalidated || ' ' || pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'entity_definitions'::regclass AND conname = 'entity_definitions_numbering_mode_check'`); got != "true CHECK ((numbering_mode = ANY (ARRAY['gaps'::text, 'gapless'::text])))" {
		t.Errorf("entity_definitions_numbering_mode_check: %s", got)
	}
	if got := commitCheckColumns(t, super); got != "current_version,numbering_mode,updated_at" {
		t.Errorf("commit check UPDATE OF %s, want current_version,numbering_mode,updated_at", got)
	}

	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.DownTo(ctx, numberingModeBefore); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_attribute
		WHERE attrelid = 'entity_definitions'::regclass AND attname = 'numbering_mode' AND NOT attisdropped`); got != "0" {
		t.Errorf("after down: numbering_mode still exists")
	}
	if got := commitCheckColumns(t, super); got != "current_version,updated_at" {
		t.Errorf("after down: commit check UPDATE OF %s, want current_version,updated_at", got)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := queryString(t, super, `SELECT string_agg(numbering_mode, ',') FROM entity_definitions`); got != "gaps,gaps" {
		t.Errorf("after up again: %q, want gaps,gaps", got)
	}
	if got := commitCheckColumns(t, super); got != "current_version,numbering_mode,updated_at" {
		t.Errorf("after up again: commit check UPDATE OF %s", got)
	}
}

// Acceptance: the CHECK refuses any other value on insert and update; nava_app
// changes its own tenant's mode only with a config entry; it can neither read
// nor change another tenant's; nava_worker cannot change it at all.
func TestP336fNumberingModeChecksAndTenancy(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	app, worker := asRole(t, "nava_app"), asRole(t, "nava_worker")

	for _, bad := range []string{"", "Gaps", "GAPLESS", "none", "gapless "} {
		_, err := execIn(app, tenantA, withDefinitionEntry("UPDATE entity_definitions SET numbering_mode = $2, updated_at = now() WHERE tenant_id = $1"), tenantA, bad)
		wantSQLState(t, "nava_app sets numbering_mode "+bad, err, "23514")
	}
	_, err := execIn(app, tenantA, `WITH n AS (SELECT gen_random_uuid() AS id),
		d AS (INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, numbering_mode, current_version)
			SELECT $1, n.id, 'incident', 'Incident', 'INC', 'sometimes', 1 FROM n) `+
		configEntry+` SELECT $1, gen_random_uuid()`+typeEntryValues+`, n.id`+typeEntryTail+` FROM n`, tenantA)
	wantSQLState(t, "nava_app inserts a type with numbering_mode sometimes", err, "23514")

	// The commit check covers the column: a change with no entry fails at COMMIT.
	_, err = execIn(app, tenantA, setMode, tenantA, "gapless")
	wantSQLState(t, "nava_app changes numbering_mode with no config entry", err, "23514")
	if n, err := execIn(app, tenantA, withDefinitionEntry(setMode), tenantA, "gapless"); err != nil || n != 1 {
		t.Fatalf("nava_app sets its own type to gapless: %d rows, %v; want 1, nil", n, err)
	}

	// Tenant A's transaction sees and changes none of tenant B's rows.
	var seen int
	if err := inTenantTx(app, tenantA, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), `SELECT count(*) FROM entity_definitions WHERE tenant_id = $1`, tenantB).Scan(&seen)
	}); err != nil || seen != 0 {
		t.Errorf("tenant A reads tenant B's types: %d, %v; want 0, nil", seen, err)
	}
	if n, err := execIn(app, tenantA, withDefinitionEntry(setMode), tenantB, "gapless"); err != nil || n != 0 {
		t.Errorf("tenant A sets tenant B's mode: %d rows, %v; want 0, nil", n, err)
	}
	if got := queryString(t, super, `SELECT string_agg(tenant_id || ' ' || numbering_mode, ',' ORDER BY tenant_id) FROM entity_definitions`); got != tenantA+" gapless,"+tenantB+" gaps" {
		t.Errorf("modes after the writes: %q, want A gapless, B gaps", got)
	}

	_, err = execIn(worker, tenantB, "UPDATE entity_definitions SET numbering_mode = 'gapless' WHERE tenant_id = $1", tenantB)
	wantSQLState(t, "nava_worker sets numbering_mode", err, "42501")
	if err := inTenantTx(worker, tenantB, func(tx pgx.Tx) error {
		var mode string
		return tx.QueryRow(context.Background(), `SELECT numbering_mode FROM entity_definitions WHERE tenant_id = $1`, tenantB).Scan(&mode)
	}); err != nil {
		t.Errorf("nava_worker reads its tenant's numbering_mode: %v", err)
	}
}
