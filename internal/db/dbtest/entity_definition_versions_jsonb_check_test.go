//go:build integration

package dbtest

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TICK-378: fixed_columns and states on entity_definition_versions are JSON
// objects. The store scans them into map[string]string and StateModel, and
// pgx refuses any other top-level type, so a '[]' row would turn every read
// of its type into a 500. Versions are immutable, so the insert is the only
// place to stop it.

// insertVersionShapes: tenant $1 (also the version id and type id),
// fixed_columns $2, states $3, as text cast to jsonb.
const insertVersionShapes = `INSERT INTO entity_definition_versions
	(tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
	VALUES ($1, $1, $1, 1, '{"type": "object"}', sha256('{"type": "object"}'), $2::jsonb, $3::jsonb, 'initial', $1)`

func TestTICK378VersionJSONBMustBeObjects(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	exec(t, super, insertType, tenantA, "ticket", "TCK")
	app := asRole(t, "nava_app")

	// Both constraints exist and are validated, not left NOT VALID.
	if got := queryString(t, super, `SELECT string_agg(conname || ':' || convalidated, ',' ORDER BY conname)
		FROM pg_constraint WHERE conrelid = 'entity_definition_versions'::regclass
		AND conname IN ('entity_definition_versions_fixed_columns_check', 'entity_definition_versions_states_check')`); got !=
		"entity_definition_versions_fixed_columns_check:true,entity_definition_versions_states_check:true" {
		t.Errorf("constraints = %q; want both present and validated", got)
	}

	const obj = `{}`
	for _, bad := range []string{`[]`, `["open","closed"]`, `null`, `"x"`, `1`, `true`} {
		for _, c := range []struct{ column, fixed, states string }{
			{"fixed_columns", bad, obj},
			{"states", obj, bad},
		} {
			_, err := execIn(app, tenantA, insertVersionShapes, tenantA, c.fixed, c.states)
			wantSQLState(t, c.column+" = "+bad, err, "23514")
			var pe *pgconn.PgError
			if want := "entity_definition_versions_" + c.column + "_check"; errors.As(err, &pe) && pe.ConstraintName != want {
				t.Errorf("%s = %s: refused by %s, want %s", c.column, bad, pe.ConstraintName, want)
			}
		}
	}
	// The superuser too: a seed or a hand-written repair is refused alike.
	_, err := execIn(super, "", insertVersionShapes, tenantA, `[]`, `[]`)
	wantSQLState(t, "superuser inserts '[]'", err, "23514")

	// Control: objects, empty or not, are accepted.
	if _, err := execIn(app, tenantA, insertVersionShapes, tenantA, `{"state": "state"}`,
		`{"initial": "open", "states": ["open", "closed"], "transitions": [{"from": "open", "to": "closed"}]}`); err != nil {
		t.Errorf("control: nava_app publishes object-shaped columns: %v; want accepted", err)
	}
}
