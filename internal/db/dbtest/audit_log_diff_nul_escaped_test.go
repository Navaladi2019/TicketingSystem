//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"maps"
	"slices"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// TICK-1170 (TICK-1121, ADR-004-E10 / TICK-1169): audit_log.diff_nul_escaped.

// diffNULEscapedMigration adds the column; diffNULEscapedBefore is the
// version just before it.
const (
	diffNULEscapedMigration = "20261126150000_audit_log_diff_nul_escaped.sql"
	diffNULEscapedBefore    = 20261126145959
)

// beforeDiffNULEscaped returns m and a without audit_log.diff_nul_escaped when
// fsys lacks the migration that adds it: no role's grant names it and the
// generated writer INSERT leaves it out, so a run over fsys applies a
// manifest whose columns exist.
func beforeDiffNULEscaped(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	if _, err := fs.Stat(fsys, diffNULEscapedMigration); err == nil {
		return m, a
	}
	const column = "diff_nul_escaped"
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		if privs, ok := r.Tables[audited.LogTable]; ok {
			tables := maps.Clone(r.Tables)
			p := make(map[string][]string, len(privs))
			for priv, cols := range privs {
				p[priv] = slices.DeleteFunc(slices.Clone(cols), func(c string) bool { return c == column })
			}
			tables[audited.LogTable] = p
			r.Tables = tables
		}
		gm.Roles[name] = r
	}
	am := *a
	am.AbsentLogColumns = append(slices.Clone(a.AbsentLogColumns), column)
	return &gm, &am
}

// The column is added to a populated audit_log without a rewrite: the
// partition keeps its relfilenode, existing rows read false, and the R11
// guard refuses any UPDATE that flips it, the superuser's included; the
// E10-6 gate refuses a true INSERT. Down drops it and the rows stay.
func TestTICK1170DiffNULEscapedColumn(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	ctx := context.Background()

	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.DownTo(ctx, diffNULEscapedBefore); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_attribute
		WHERE attrelid = 'audit_log'::regclass AND attname = 'diff_nul_escaped' AND NOT attisdropped`); got != "0" {
		t.Fatalf("after down: diff_nul_escaped still on audit_log")
	}

	// Populate under the old shape.
	const seed = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
		actor_system, source) SELECT $1, gen_random_uuid(), 'record', 'update', 'record', gen_random_uuid(),
		'system', 'system' FROM generate_series(1, 1000)`
	exec(t, super, seed, tenantA)
	const node = `SELECT relfilenode::text FROM pg_class WHERE oid = 'audit_log_default'::regclass`
	before := queryString(t, super, node)
	const total = `SELECT count(*)::text FROM audit_log WHERE tenant_id = $1`
	rows := queryString(t, super, total, tenantA) // the seed plus any entry seedTenants wrote

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}
	if got := queryString(t, super, node); got != before {
		t.Errorf("audit_log_default relfilenode %s -> %s: ADD COLUMN rewrote the partition", before, got)
	}
	if got := queryString(t, super, `SELECT count(*) FILTER (WHERE NOT diff_nul_escaped)::text
		FROM audit_log WHERE tenant_id = $1`, tenantA); got != rows {
		t.Errorf("existing rows reading false = %s, want all %s", got, rows)
	}

	// R11: nobody flips it, the owner and the superuser included.
	id := queryString(t, super, `SELECT id::text FROM audit_log WHERE tenant_id = $1 LIMIT 1`, tenantA)
	_, err = super.Exec(ctx, `UPDATE audit_log SET diff_nul_escaped = true WHERE tenant_id = $1 AND id = $2`, tenantA, id)
	wantSQLState(t, "superuser flips diff_nul_escaped", err, "42501")

	// E10-6: until step 2 the gate refuses a true INSERT with 23514, for
	// every role (the superuser here; the writers in
	// TestAppendDiffNULEscapedGated).
	_, err = super.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
		actor_system, source, diff_nul_escaped) VALUES ($1, gen_random_uuid(), 'record', 'update', 'record',
		gen_random_uuid(), 'system', 'system', true)`, tenantA)
	wantSQLState(t, "superuser inserts diff_nul_escaped = true", err, "23514")

	// Down and up again: the drop keeps the rows.
	rows = queryString(t, super, total, tenantA)
	if _, err := p.DownTo(ctx, diffNULEscapedBefore); err != nil {
		t.Fatalf("down with rows: %v", err)
	}
	if got := queryString(t, super, total, tenantA); got != rows {
		t.Errorf("rows after down = %s, want %s", got, rows)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
}
