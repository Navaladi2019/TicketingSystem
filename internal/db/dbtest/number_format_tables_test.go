//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// P6-06 (TICK-671, ADR-005 §3.2, §3.3, §3.6, §3.7 C8, C9, C12): the number
// format tables, the epoch registry, the pinned link on
// entity_definition_versions, the tenant time zone and the number_format
// audit subject.

const (
	numberFormatTablesMigration = "20261113100000_number_format_tables.sql"
	// numberFormatBefore is the version just before every P6-06 migration.
	numberFormatBefore = 20261113095959

	// insertFormat: tenant $1, format id $2, key $3.
	insertFormat = `INSERT INTO number_formats (tenant_id, id, key, current_version) VALUES ($1, $2, $3, 1)`
	// insertFormatVersion: tenant $1, format $2, version $3, created_by $4.
	insertFormatVersion = `INSERT INTO number_format_versions (tenant_id, format_id, version, document, compiled, compiled_sha256, created_by)
		VALUES ($1, $2, $3, '{"template": "{prefix}-{yyyy}{MM}{seq:4}"}', '{"head": "-"}', sha256(int4send($3)), $4)`
	// insertEpoch: tenant $1, type $2, epoch $3, hash seed $4, head $5.
	insertEpoch = `INSERT INTO numbering_epochs (tenant_id, entity_definition_id, epoch, compiled_sha256, literal_head)
		VALUES ($1, $2, $3, sha256(convert_to($4::text, 'UTF8')), $5)`
	// linkVersion publishes version 2 of type $1 (tenant $1, so seedEntityTypes'
	// type), pinned to (tenant $1, format $2, version $3), with its config entry.
	linkVersion = `WITH v AS (INSERT INTO entity_definition_versions
			(tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by,
			 number_format_id, number_format_version, number_compiled)
			VALUES ($1, gen_random_uuid(), $1, 2, '{}'::json, sha256(convert_to('{}', 'UTF8')), '{}', '{}', 'compatible', $1,
			 $2, $3, '{"head": "-"}')) ` + configEntry + `
		VALUES ($1, gen_random_uuid(), 'config', 'publish', 'entity_definition', $1, 2, 'system', 'system', sha256(''::bytea))`
)

// numberFormatTables are the three tables 20261113100000 creates.
var numberFormatTables = []string{"number_formats", "number_format_versions", "numbering_epochs"}

// seedNumberFormats writes, as the superuser, for each tenant: seedEntityTypes'
// rows (a principal and a type whose id is the tenant id), a format whose id
// is the tenant id under the key "monthly" at version 1, and epoch 1 of the
// tenant's type.
func seedNumberFormats(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	seedEntityTypes(t, super, tenants...)
	for _, id := range tenants {
		exec(t, super, insertFormat, id, id, "monthly")
		exec(t, super, insertFormatVersion, id, id, 1, id)
		exec(t, super, insertEpoch, id, id, 1, "monthly", "TCK-")
	}
}

// beforeNumberFormats drops the P6-06 tables from today's grants and audited
// manifests when fsys stops before 20261113100000, so a test that migrates to
// an earlier version can still apply them.
func beforeNumberFormats(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	if _, err := fs.Stat(fsys, numberFormatTablesMigration); err == nil {
		return m, a
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		r.Tables = maps.Clone(r.Tables)
		for _, tbl := range numberFormatTables {
			delete(r.Tables, tbl)
		}
		gm.Roles[name] = r
	}
	am := *a
	am.Audited = maps.Clone(a.Audited)
	for _, tbl := range numberFormatTables {
		delete(am.Audited, tbl)
	}
	return &gm, &am
}

// Acceptance (C9): nava_app may SELECT and INSERT all three tables and UPDATE
// only current_version and updated_at of number_formats; nava_worker may only
// SELECT; no other role (every role in the cluster, read from pg_roles) has
// any privilege. RLS is ENABLEd and FORCEd, and every policy is for nava_app
// or nava_worker with the canonical predicate. The UPDATE grant on
// entity_definitions is unchanged.
func TestP606Grants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	roles := strings.Split(queryString(t, super, `SELECT string_agg(rolname, ',' ORDER BY rolname) FROM pg_roles
		WHERE NOT rolsuper AND rolname NOT LIKE 'pg\_%' AND rolname <> 'nava_migrator'`), ",")
	for _, must := range []string{"nava_app", "nava_worker", "nava_purge", "nava_onboard"} {
		if !slices.Contains(roles, must) {
			t.Fatalf("roles in the cluster = %v, want %s among them", roles, must)
		}
	}
	for _, table := range numberFormatTables {
		for _, role := range roles {
			for _, priv := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"} {
				want := (role == "nava_app" && (priv == "SELECT" || priv == "INSERT")) ||
					(role == "nava_worker" && priv == "SELECT")
				got := queryBool(t, super, "SELECT has_table_privilege($1, $2, $3)", role, "public."+table, priv)
				if got != want {
					t.Errorf("%s %s on %s = %v, want %v", role, priv, table, got, want)
				}
			}
			// Column-level UPDATE: nava_app on number_formats' two columns only.
			got := queryString(t, super, `SELECT coalesce(string_agg(attname, ',' ORDER BY attname), '') FROM pg_attribute
				WHERE attrelid = $1::regclass AND attnum > 0 AND NOT attisdropped
				  AND has_column_privilege($2, attrelid, attnum, 'UPDATE')`, table, role)
			want := ""
			if role == "nava_app" && table == "number_formats" {
				want = "current_version,updated_at"
			}
			if got != want {
				t.Errorf("%s UPDATE columns of %s = %q, want %q", role, table, got, want)
			}
		}
		if got := queryString(t, super, `SELECT relrowsecurity::text || relforcerowsecurity::text FROM pg_class
			WHERE oid = $1::regclass`, table); got != "truetrue" {
			t.Errorf("%s RLS enabled and forced = %s, want truetrue", table, got)
		}
		// Every policy names nava_app or nava_worker and tests exactly the
		// canonical predicate; none is for UPDATE or DELETE on the
		// insert-only tables.
		rows, err := super.Query(context.Background(), `SELECT policyname, cmd, array_to_string(roles, ','),
				coalesce(qual, ''), coalesce(with_check, '')
			FROM pg_policies WHERE schemaname = 'public' AND tablename = $1 ORDER BY policyname`, table)
		if err != nil {
			t.Fatal(err)
		}
		var cmds []string
		for rows.Next() {
			var name, cmd, to, qual, check string
			if err := rows.Scan(&name, &cmd, &to, &qual, &check); err != nil {
				t.Fatal(err)
			}
			cmds = append(cmds, cmd)
			const canon = "(tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)"
			for _, expr := range []string{qual, check} {
				if expr != "" && expr != canon {
					t.Errorf("%s policy %s: predicate %s, want %s", table, name, expr, canon)
				}
			}
			for _, r := range strings.Split(to, ",") {
				if r != "nava_app" && r != "nava_worker" {
					t.Errorf("%s policy %s is for %s", table, name, r)
				}
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		slices.Sort(cmds)
		want := "INSERT,SELECT"
		if table == "number_formats" {
			want = "INSERT,SELECT,UPDATE"
		}
		if got := strings.Join(cmds, ","); got != want {
			t.Errorf("%s policy commands = %s, want %s", table, got, want)
		}
	}
	if got := queryString(t, super, `SELECT string_agg(attname, ',' ORDER BY attname) FROM pg_attribute
		WHERE attrelid = 'entity_definitions'::regclass AND attnum > 0 AND NOT attisdropped
		  AND has_column_privilege('nava_app', attrelid, attnum, 'UPDATE')`); got != "current_version,numbering_mode,updated_at" {
		t.Errorf("nava_app UPDATE columns of entity_definitions = %s, want current_version,numbering_mode,updated_at", got)
	}
}

// Acceptance: tenant A reads none of tenant B's rows, by id or by key, as
// nava_app or nava_worker, and nothing with no tenant or an empty one; an
// insert for tenant B is refused; nava_worker inserts nothing; nava_app
// neither updates nor deletes a version or an epoch, not even its own.
func TestP606TenantIsolation(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedNumberFormats(t, super, tenantA, tenantB)

	lookups := map[string][]string{
		"number_formats":         {"id = $1", "key = 'monthly'"},
		"number_format_versions": {"format_id = $1", "version = 1"},
		"numbering_epochs":       {"entity_definition_id = $1", "compiled_sha256 = sha256('monthly'::bytea)"},
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, table := range numberFormatTables {
			for _, where := range lookups[table] {
				for _, tenant := range []string{tenantA, "", "unset"} {
					var own, other int
					err := inTenantTx(conn, "", func(tx pgx.Tx) error {
						if tenant != "unset" {
							if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", tenant); err != nil {
								return err
							}
						}
						args := []any{tenantB}
						if !strings.Contains(where, "$1") {
							args = nil
						}
						return tx.QueryRow(context.Background(),
							"SELECT count(*) FILTER (WHERE tenant_id = '"+tenantA+"'), count(*) FILTER (WHERE tenant_id <> '"+tenantA+"') FROM "+
								table+" WHERE "+where+" OR tenant_id = '"+tenantA+"'", args...).Scan(&own, &other)
					})
					wantOwn := 0
					if tenant == tenantA {
						wantOwn = 1
					}
					if err != nil || own != wantOwn || other != 0 {
						t.Errorf("%s reads %s WHERE %s with tenant %q: own %d, other %d, %v; want %d, 0",
							role, table, where, tenant, own, other, err, wantOwn)
					}
				}
			}
		}
	}

	app, worker := asRole(t, "nava_app"), asRole(t, "nava_worker")
	newID := "0c000000-0000-4000-8000-0000000000c9"
	_, err := execIn(app, tenantA, insertFormat, tenantB, newID, "iso")
	wantSQLState(t, "nava_app in tenant A creates tenant B's format", err, "42501")
	_, err = execIn(app, tenantA, insertFormatVersion, tenantB, tenantB, 2, tenantB)
	wantSQLState(t, "nava_app in tenant A publishes tenant B's format version", err, "42501")
	_, err = execIn(app, tenantA, insertEpoch, tenantB, tenantB, 2, "iso", "TCK/")
	wantSQLState(t, "nava_app in tenant A takes tenant B's epoch", err, "42501")
	_, err = execIn(app, "", insertFormat, tenantA, newID, "iso")
	wantSQLState(t, "nava_app with no tenant creates a format", err, "42501")

	_, err = execIn(worker, tenantA, insertFormat, tenantA, newID, "iso")
	wantSQLState(t, "nava_worker creates a format", err, "42501")
	_, err = execIn(worker, tenantA, insertFormatVersion, tenantA, tenantA, 2, tenantA)
	wantSQLState(t, "nava_worker publishes a format version", err, "42501")
	_, err = execIn(worker, tenantA, insertEpoch, tenantA, tenantA, 2, "iso", "TCK/")
	wantSQLState(t, "nava_worker takes an epoch", err, "42501")

	for _, stmt := range []string{
		"UPDATE number_format_versions SET compiled = '{}' WHERE tenant_id = $1",
		"DELETE FROM number_format_versions WHERE tenant_id = $1",
		"UPDATE numbering_epochs SET literal_head = 'TCK/' WHERE tenant_id = $1",
		"DELETE FROM numbering_epochs WHERE tenant_id = $1",
		"DELETE FROM number_formats WHERE tenant_id = $1",
		"UPDATE number_formats SET key = 'renamed' WHERE tenant_id = $1",
	} {
		_, err = execIn(app, tenantA, stmt, tenantA)
		wantSQLState(t, "nava_app: "+stmt, err, "42501")
	}
	// nava_app moves its own format's current_version, and none of tenant B's.
	if n, err := execIn(app, tenantA, "UPDATE number_formats SET current_version = 2, updated_at = now() WHERE id IN ($1, $2)",
		tenantA, tenantB); err != nil || n != 1 {
		t.Errorf("nava_app moves current_version of tenant A's and B's formats: %d rows, %v; want 1, nil", n, err)
	}
	if got := queryString(t, super, `SELECT string_agg(current_version::text, ',' ORDER BY tenant_id) FROM number_formats`); got != "2,1" {
		t.Errorf("current_version after the update = %s, want 2,1", got)
	}

	for _, role := range []string{"nava_onboard", "nava_groupsync", "nava_relay", "nava_purge", "nava_audit_reader", "nava_erasure"} {
		for _, table := range numberFormatTables {
			_, err := execIn(asRole(t, role), tenantA, "SELECT count(*) FROM "+table)
			wantSQLState(t, role+" reads "+table, err, "42501")
		}
	}
}

// Acceptance (C8, the FK case): a tenant A definition version pinned to
// tenant B's format id fails exactly as one pinned to a made-up id does: the
// same SQLSTATE on the same constraint, so the link is no existence oracle.
// Pinned to its own format it publishes, and a half-set link is refused.
func TestP606LinkToAnotherTenantsFormat(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedNumberFormats(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")

	_, errB := execIn(app, tenantA, linkVersion, tenantA, tenantB, 1)
	_, errMade := execIn(app, tenantA, linkVersion, tenantA, "0c000000-0000-4000-8000-0000000000c8", 1)
	codeB, conB := pgErrDetail(errB)
	codeMade, conMade := pgErrDetail(errMade)
	if codeB != "23503" || conB != "entity_definition_versions_number_format_fkey" {
		t.Errorf("tenant A pins tenant B's format: %v (constraint %q), want 23503 on entity_definition_versions_number_format_fkey", errB, conB)
	}
	if codeB != codeMade || conB != conMade {
		t.Errorf("tenant B's id gives %s/%s, a made-up id %s/%s: they must not differ", codeB, conB, codeMade, conMade)
	}
	_, err := execIn(app, tenantA, linkVersion, tenantA, tenantA, 2)
	wantSQLState(t, "tenant A pins a version its own format does not have", err, "23503")

	// The epoch and the format version also carry tenant_id in their keys.
	_, err = execIn(app, tenantA, insertEpoch, tenantA, tenantB, 2, "iso", "TCK/")
	wantSQLState(t, "tenant A takes an epoch for tenant B's type", err, "23503")
	_, err = execIn(app, tenantA, insertFormatVersion, tenantA, tenantB, 2, tenantA)
	wantSQLState(t, "tenant A publishes a version of tenant B's format", err, "23503")
	_, err = execIn(app, tenantA, insertFormatVersion, tenantA, tenantA, 2, tenantB)
	wantSQLState(t, "tenant A names tenant B's principal as created_by", err, "23503")

	if n, err := execIn(app, tenantA, linkVersion, tenantA, tenantA, 1); err != nil || n != 1 {
		t.Fatalf("tenant A pins its own format: %d, %v; want 1 entry, nil", n, err)
	}
	if got := queryString(t, super, `SELECT number_format_id || ' ' || number_format_version || ' ' || number_compiled::text
		FROM entity_definition_versions WHERE tenant_id = $1 AND version = 2`, tenantA); got != tenantA+` 1 {"head": "-"}` {
		t.Errorf("pinned link reads %q", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM entity_definition_versions
		WHERE version = 1 AND number_format_id IS NULL AND number_format_version IS NULL AND number_compiled IS NULL`); got != "2" {
		t.Errorf("default versions with a NULL link = %s, want 2", got)
	}

	for name, cols := range map[string]string{
		"id only":             `$2, NULL, NULL`,
		"id and version only": `$2, 1, NULL`,
		"compiled only":       `NULLIF($2::uuid, $2::uuid), NULL, '{"head": "-"}'`,
		"compiled not object": `$2, 1, '[]'`,
	} {
		stmt := `INSERT INTO entity_definition_versions (tenant_id, id, definition_id, version, schema, schema_sha256,
			fixed_columns, states, compat, published_by, number_format_id, number_format_version, number_compiled)
			VALUES ($1, gen_random_uuid(), $1, 3, '{}'::json, sha256(convert_to('{}', 'UTF8')), '{}', '{}', 'compatible', $1, ` + cols + `)`
		_, err := execIn(app, tenantA, withDefinitionEntry(stmt), tenantA, tenantA)
		if code, con := pgErrDetail(err); code != "23514" || con != "entity_definition_versions_number_format_check" {
			t.Errorf("half-set link (%s): %v (constraint %q), want 23514 on entity_definition_versions_number_format_check", name, err, con)
		}
	}
}

// Acceptance (§3.3): the two UNIQUE keys of the epoch registry, epoch 0 has no
// row, and the shape CHECKs on the three tables.
func TestP606ShapeChecks(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedNumberFormats(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")

	type refusal struct {
		what, stmt, code, constraint string
		args                         []any
	}
	for _, r := range []refusal{
		{"epoch number taken twice", insertEpoch, "23505", "numbering_epochs_pkey", []any{tenantA, tenantA, 1, "other", "TCK/"}},
		{"compiled template registered twice", insertEpoch, "23505", "numbering_epochs_compiled_sha256_key", []any{tenantA, tenantA, 2, "monthly", "TCK/"}},
		{"epoch 0 registered", insertEpoch, "23514", "numbering_epochs_epoch_check", []any{tenantA, tenantA, 0, "zero", "TCK"}},
		{"literal head without the prefix", insertEpoch, "23514", "numbering_epochs_literal_head_check", []any{tenantA, tenantA, 2, "x", "-TCK"}},
		{"literal head with a lowercase letter", insertEpoch, "23514", "numbering_epochs_literal_head_check", []any{tenantA, tenantA, 2, "x", "TCKa"}},
		{"format key taken twice", insertFormat, "23505", "number_formats_key_key", []any{tenantA, "0c000000-0000-4000-8000-0000000000c7", "monthly"}},
		{"format key with an uppercase letter", insertFormat, "23514", "number_formats_key_check", []any{tenantA, "0c000000-0000-4000-8000-0000000000c7", "Monthly"}},
		{"format version taken twice", insertFormatVersion, "23505", "number_format_versions_pkey", []any{tenantA, tenantA, 1, tenantA}},
		{"format version 0", insertFormatVersion, "23514", "number_format_versions_version_check", []any{tenantA, tenantA, 0, tenantA}},
		{"compiled_sha256 of 31 bytes", `INSERT INTO numbering_epochs (tenant_id, entity_definition_id, epoch, compiled_sha256, literal_head)
			VALUES ($1, $2, 2, substring(sha256('x'::bytea) from 2), 'TCK')`, "23514", "numbering_epochs_compiled_sha256_check", []any{tenantA, tenantA}},
		{"document not an object", `INSERT INTO number_format_versions (tenant_id, format_id, version, document, compiled, compiled_sha256, created_by)
			VALUES ($1, $1, 2, '[]', '{}', sha256('x'::bytea), $1)`, "23514", "number_format_versions_document_check", []any{tenantA}},
	} {
		_, err := execIn(app, tenantA, r.stmt, r.args...)
		if code, con := pgErrDetail(err); code != r.code || con != r.constraint {
			t.Errorf("%s: %v (constraint %q), want %s on %s", r.what, err, con, r.code, r.constraint)
		}
	}
	// The same key, hash and head in another tenant are not a clash.
	if got := queryString(t, super, `SELECT count(DISTINCT tenant_id)::text FROM numbering_epochs WHERE literal_head = 'TCK-'`); got != "2" {
		t.Errorf("tenants holding epoch head TCK- = %s, want 2", got)
	}
	if _, err := execIn(app, tenantA, insertEpoch, tenantA, tenantA, 2, "daily", "TCK-"); err != nil {
		t.Errorf("nava_app takes epoch 2: %v", err)
	}
}

// Acceptance (C12): the audit CHECK accepts number_format and still refuses
// an unknown subject type; it ends validated.
func TestP606AuditSubjectNumberFormat(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	const entry = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after,
		actor_system, source, content_sha256) VALUES ($1, gen_random_uuid(), 'config', 'publish', $2, $1, 1, 'system', 'system', sha256(''::bytea))`
	if _, err := super.Exec(context.Background(), entry, tenantA, "number_format"); err != nil {
		t.Errorf("audit entry with subject_type number_format: %v", err)
	}
	_, err := super.Exec(context.Background(), entry, tenantA, "number_formats")
	if code, con := pgErrDetail(err); code != "23514" || con != "audit_log_subject_type_check" {
		t.Errorf("audit entry with subject_type number_formats: %v (constraint %q), want 23514 on audit_log_subject_type_check", err, con)
	}
	if got := queryString(t, super, `SELECT convalidated::text FROM pg_constraint
		WHERE conrelid = 'audit_log'::regclass AND conname = 'audit_log_subject_type_check'`); got != "true" {
		t.Errorf("audit_log_subject_type_check validated = %s, want true", got)
	}
}

// Acceptance (§3.6): tenants.time_zone defaults to UTC, takes IANA-shaped
// names and refuses others.
func TestP606TenantTimeZone(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	if got := queryString(t, super, `SELECT time_zone FROM tenants WHERE id = $1`, tenantA); got != "UTC" {
		t.Errorf("new tenant's time_zone = %q, want UTC", got)
	}
	for _, ok := range []string{"Asia/Kolkata", "America/Argentina/Buenos_Aires", "America/Port-au-Prince", "Etc/GMT+5", "UTC"} {
		if _, err := super.Exec(context.Background(), `UPDATE tenants SET time_zone = $2 WHERE id = $1`, tenantA, ok); err != nil {
			t.Errorf("time_zone %q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "/UTC", "Asia/Kolkata; DROP", "+05:30", strings.Repeat("A", 65)} {
		_, err := super.Exec(context.Background(), `UPDATE tenants SET time_zone = $2 WHERE id = $1`, tenantA, bad)
		if code, con := pgErrDetail(err); code != "23514" || con != "tenants_time_zone_check" {
			t.Errorf("time_zone %q: %v (constraint %q), want 23514 on tenants_time_zone_check", bad, err, con)
		}
	}
}

// Acceptance: the P6-06 migrations apply to a database that already holds
// types and versions without rewriting entity_definition_versions or tenants,
// every existing version reads the default (NULL) link and every tenant UTC,
// every new constraint ends validated, and the migrations go down and up
// again.
func TestP606Migration(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, numberFormatTablesMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", numberFormatTablesMigration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	filenodes := `SELECT pg_relation_filenode('entity_definition_versions')::text || ',' || pg_relation_filenode('tenants')::text`
	before := queryString(t, super, filenodes)

	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if got := queryString(t, super, filenodes); got != before {
		t.Errorf("entity_definition_versions,tenants filenodes %s -> %s: the migration rewrote a table", before, got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM entity_definition_versions
		WHERE number_format_id IS NULL AND number_format_version IS NULL AND number_compiled IS NULL`); got != "2" {
		t.Errorf("existing versions with the default link = %s, want 2", got)
	}
	if got := queryString(t, super, `SELECT string_agg(time_zone, ',') FROM tenants`); got != "UTC,UTC" {
		t.Errorf("existing tenants' time_zone = %s, want UTC,UTC", got)
	}
	constraints := `SELECT string_agg(conname || '=' || convalidated, ',' ORDER BY conname) FROM pg_constraint
		WHERE conname IN ('entity_definition_versions_number_format_fkey', 'entity_definition_versions_number_format_check',
		                  'tenants_time_zone_check', 'audit_log_subject_type_check')
		  AND conrelid IN ('entity_definition_versions'::regclass, 'tenants'::regclass, 'audit_log'::regclass)`
	const allValid = "audit_log_subject_type_check=true,entity_definition_versions_number_format_check=true," +
		"entity_definition_versions_number_format_fkey=true,tenants_time_zone_check=true"
	if got := queryString(t, super, constraints); got != allValid {
		t.Errorf("constraints = %s, want %s", got, allValid)
	}

	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.DownTo(ctx, numberFormatBefore); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_class WHERE relnamespace = 'public'::regnamespace
		AND relname IN ('number_formats', 'number_format_versions', 'numbering_epochs')`); got != "0" {
		t.Errorf("after down: %s of the three tables remain", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_attribute WHERE NOT attisdropped
		AND ((attrelid = 'entity_definition_versions'::regclass AND attname IN ('number_format_id', 'number_format_version', 'number_compiled'))
		  OR (attrelid = 'tenants'::regclass AND attname = 'time_zone'))`); got != "0" {
		t.Errorf("after down: %s of the new columns remain", got)
	}
	if got := queryString(t, super, `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'audit_log'::regclass AND conname = 'audit_log_subject_type_check'`); strings.Contains(got, "number_format") {
		t.Errorf("after down: audit_log_subject_type_check still names number_format: %s", got)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := queryString(t, super, constraints); got != allValid {
		t.Errorf("after up again: constraints = %s, want %s", got, allValid)
	}
}
