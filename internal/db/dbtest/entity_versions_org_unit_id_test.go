//go:build integration

package dbtest

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
)

// TICK-1115 (ADR-010 D8): no stored entity-type version has a top-level
// property named org_unit_id.

const (
	orgUnitIDReservedMigration  = "20261125040000_entity_versions_org_unit_id_reserved.sql"
	orgUnitIDReservedConstraint = "entity_definition_versions_org_unit_id_reserved"
	// leakMarker is in every refused schema; no error text may hold it.
	leakMarker = "leak-marker-1115"

	cleanSchema    = `{"type": "object", "properties": {"a": {"type": "string"}}}`
	reservedSchema = `{"title": "` + leakMarker + `", "type": "object", "properties": {"org_unit_id": {"type": "string"}}}`
)

// esc writes a backslash for each ~. The fixtures name their JSON escapes
// this way so that no editor or tool can decode one: a decoded fixture passes
// without testing anything.
func esc(s string) string { return strings.ReplaceAll(s, "~", string(rune(92))) }

type storedVersion struct {
	tenant  string
	version int
	schema  string
}

// orgUnitIDCase is the versions one upgrade finds. The stored text of the
// last one must hold stored.
type orgUnitIDCase struct {
	name     string
	versions []storedVersion
	stored   string
}

// oneVersion is tenant A with only a version 1; esc applies to schema and
// stored.
func oneVersion(name, schema, stored string) orgUnitIDCase {
	return orgUnitIDCase{name, []storedVersion{{tenantA, 1, esc(schema)}}, esc(stored)}
}

// orgUnitIDRefused are stored versions the migration refuses.
func orgUnitIDRefused() []orgUnitIDCase {
	titled := func(rest string) string { return `{"title": "` + leakMarker + `", ` + rest + `}` }
	return []orgUnitIDCase{
		{"in the current version", []storedVersion{{tenantA, 1, cleanSchema}, {tenantA, 2, reservedSchema}}, ""},
		{"in an older version only", []storedVersion{{tenantA, 1, reservedSchema}, {tenantA, 2, cleanSchema}}, ""},
		{"in tenant B only", []storedVersion{{tenantA, 1, cleanSchema}, {tenantB, 1, reservedSchema}}, ""},
		oneVersion("value false", titled(`"properties": {"org_unit_id": false}`), ""),
		oneVersion("value null", titled(`"properties": {"org_unit_id": null}`), ""),
		oneVersion("a NUL escape beside it", titled(`"description": "x~u0000y", "properties": {"org_unit_id": {}}`), "~u0000"),
		oneVersion("an escaped letter in org_unit_id", titled(`"properties": {"~u006frg_unit_id": {}}`), "~u006f"),
		oneVersion("an escaped letter in properties", titled(`"~u0070roperties": {"org_unit_id": {}}`), "~u0070"),
	}
}

// orgUnitIDAllowed are stored versions the migration accepts. Publish accepts
// neither a NUL escape nor a lone surrogate, but a row stored before it
// refused them must not block an upgrade.
func orgUnitIDAllowed() []orgUnitIDCase {
	described := func(d string) string {
		return `{"description": "x` + d + `y", "properties": {"a": {"type": "string"}}}`
	}
	return []orgUnitIDCase{
		oneVersion("no such property", cleanSchema, ""),
		oneVersion("only a nested org_unit_id", `{"properties": {"a": {"type": "object", "properties": {"org_unit_id": {}}}}}`, ""),
		oneVersion("no properties key", `{"type": "object"}`, ""),
		{"no versions", nil, ""},
		oneVersion("a NUL escape", described("~u0000"), "~u0000"),
		oneVersion("a lone high surrogate", described("~ud800"), "~ud800"),
		oneVersion("a lone high surrogate in upper case", described("~uD83D"), "~uD83D"),
		oneVersion("a lone low surrogate", described("~udc00"), "~udc00"),
		oneVersion("an escaped surrogate pair", described("~ud83d~ude00"), "~ud83d~ude00"),
		oneVersion("an escaped backslash then u0000", described("~~u0000"), "~~u0000"),
		oneVersion("a key org_unit_id plus a NUL escape", `{"properties": {"org_unit_id~u0000": {}}}`, "org_unit_id~u0000"),
	}
}

// storeVersions replaces every version with c's, as the superuser with
// triggers off: these are rows an earlier release stored.
func storeVersions(t *testing.T, super *pgx.Conn, c orgUnitIDCase) {
	t.Helper()
	exec(t, super, "SET session_replication_role = replica")
	exec(t, super, "DELETE FROM entity_definition_versions")
	for _, v := range c.versions {
		compat := "compatible"
		if v.version == 1 {
			compat = "initial"
		}
		exec(t, super, `INSERT INTO entity_definition_versions
			(tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
			VALUES ($1, gen_random_uuid(), $1, $2, $3::text::json, $4, '{}', '{}', $5, $1)`,
			v.tenant, v.version, v.schema, schemaSum(v.schema), compat)
		exec(t, super, "UPDATE entity_definitions SET current_version = $2 WHERE tenant_id = $1", v.tenant, v.version)
	}
	exec(t, super, "SET session_replication_role = DEFAULT")
	if c.stored != "" {
		last := c.versions[len(c.versions)-1]
		wantStored(t, super, last.tenant, last.version, c.stored)
	}
}

// wantStored fails unless the version's stored text holds part.
func wantStored(t *testing.T, super *pgx.Conn, tenant string, version int, part string) {
	t.Helper()
	if !queryBool(t, super, `SELECT strpos(schema::text, $3) > 0 FROM entity_definition_versions
		WHERE tenant_id = $1 AND version = $2`, tenant, version, part) {
		t.Fatalf("the stored schema does not hold %s: the fixture tests nothing", part)
	}
}

func hasOrgUnitIDConstraint(t *testing.T, conn *pgx.Conn) bool {
	t.Helper()
	return queryBool(t, conn, `SELECT EXISTS (SELECT 1 FROM pg_constraint
		WHERE conrelid = 'entity_definition_versions'::regclass AND conname = $1 AND convalidated)`, orgUnitIDReservedConstraint)
}

// The upgrade stops on a stored top-level org_unit_id, in whatever tenant and
// version, and on nothing else. The refusal names the constraint and no
// tenant, type or schema text.
func TestTICK1115MigrationRefusesStoredOrgUnitID(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, orgUnitIDReservedMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", orgUnitIDReservedMigration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	const key = "leak_key_1115"
	exec(t, super, "SET session_replication_role = replica")
	exec(t, super, "UPDATE entity_definitions SET key = $1", key)
	exec(t, super, "SET session_replication_role = DEFAULT")

	for _, c := range orgUnitIDRefused() {
		t.Run("refused/"+c.name, func(t *testing.T) {
			storeVersions(t, super, c)
			log, out := newLog()
			_, err := migrate.Up(context.Background(), roleConfig(t, m.Owner(), m.PlatformDatabase),
				migrationsThrough(t, orgUnitIDReservedMigration), log, migrate.Options{LockRetrySeconds: 1})
			pe := refusal(t, "migrate", err)
			if pe == nil {
				undoOrgUnitIDReserved(t, m, super)
				return
			}
			if pe.ConstraintName != orgUnitIDReservedConstraint {
				t.Errorf("refused by %q, want %s", pe.ConstraintName, orgUnitIDReservedConstraint)
			}
			for _, text := range []string{err.Error(), pe.Message, pe.Detail, pe.Hint, pe.Where, pe.InternalQuery, out.String()} {
				for _, secret := range []string{tenantA, tenantB, key, leakMarker} {
					if strings.Contains(text, secret) {
						t.Errorf("the refusal holds %q: %s", secret, text)
					}
				}
			}
			if hasOrgUnitIDConstraint(t, super) {
				t.Error("the refused migration left the constraint")
			}
		})
	}
	for _, c := range orgUnitIDAllowed() {
		t.Run("applies/"+c.name, func(t *testing.T) {
			storeVersions(t, super, c)
			if err := migrateOnlyFS(t, m, migrationsThrough(t, orgUnitIDReservedMigration)); err != nil {
				t.Fatalf("migrate: %v", err)
			}
			if !hasOrgUnitIDConstraint(t, super) {
				t.Error("the constraint is missing or not validated")
			}
			undoOrgUnitIDReserved(t, m, super)
			if got := queryString(t, super, "SELECT count(*)::text FROM entity_definition_versions"); (got == "0") != (len(c.versions) == 0) {
				t.Errorf("%s versions after Up and Down, stored %d", got, len(c.versions))
			}
		})
	}
	storeVersions(t, super, oneVersion("clean", cleanSchema, ""))
	migrateAll(t, m)
}

// undoOrgUnitIDReserved runs the migration's Down section as nava_migrator and
// forgets its version, so the next case upgrades again.
func undoOrgUnitIDReserved(t *testing.T, m *grants.Manifest, super *pgx.Conn) {
	t.Helper()
	b, err := fs.ReadFile(db.Migrations(), orgUnitIDReservedMigration)
	if err != nil {
		t.Fatal(err)
	}
	_, down, ok := strings.Cut(string(b), "-- +goose Down")
	if !ok {
		t.Fatalf("%s has no Down section", orgUnitIDReservedMigration)
	}
	exec(t, connect(t, roleConfig(t, m.Owner(), m.PlatformDatabase)), down)
	if hasOrgUnitIDConstraint(t, super) {
		t.Fatal("Down left the constraint")
	}
	version, _, _ := strings.Cut(orgUnitIDReservedMigration, "_")
	exec(t, super, "DELETE FROM goose_db_version WHERE version_id = $1::text::bigint", version)
}

// Once applied, the constraint refuses the same schemas on a direct INSERT by
// nava_app, as a replica of the previous release would make, and accepts the
// rest.
func TestTICK1115InsertRefusesOrgUnitID(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")
	if !hasOrgUnitIDConstraint(t, super) {
		t.Error("the constraint is missing or not validated")
	}
	insert := func(conn *pgx.Conn, tenant, session string, version int, schema string) error {
		_, err := execIn(conn, session, insertVersion, tenant, queryString(t, super, "SELECT gen_random_uuid()::text"),
			tenant, version, schema, schemaSum(schema), "compatible", tenant)
		return err
	}
	version := 1
	for _, c := range orgUnitIDRefused() {
		for _, v := range c.versions {
			if v.schema == cleanSchema {
				continue
			}
			for _, as := range []struct {
				name    string
				conn    *pgx.Conn
				session string
			}{{"nava_app", app, v.tenant}, {"superuser", super, ""}} {
				version++
				err := insert(as.conn, v.tenant, as.session, version, v.schema)
				var pe *pgconn.PgError
				if !errors.As(err, &pe) || pe.Code != "23514" || pe.ConstraintName != orgUnitIDReservedConstraint {
					t.Errorf("%s: %s inserts: %v; want 23514 on %s", c.name, as.name, err, orgUnitIDReservedConstraint)
				}
			}
		}
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM entity_definition_versions"); got != "2" {
		t.Errorf("%s versions after the refused inserts, want the 2 seeded", got)
	}
	for _, c := range orgUnitIDAllowed() {
		for _, v := range c.versions {
			version++
			if err := insert(app, tenantA, tenantA, version, v.schema); err != nil {
				t.Errorf("%s: %v; want accepted", c.name, err)
				continue
			}
			if c.stored != "" {
				wantStored(t, super, tenantA, version, c.stored)
			}
		}
	}
}
