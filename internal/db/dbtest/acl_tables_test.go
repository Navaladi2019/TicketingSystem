//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// P3-30 (TICK-615): acl_versions and record_shares, the S10 authorization
// tables. Per tenant under FORCE RLS, insert-only for nava_app and
// nava_worker, nothing for any other role, and composite foreign keys that
// hold the tenant.

const (
	// insertACL: tenant $1, type $2, version $3, publisher $4.
	insertACL = `INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks, published_by)
		VALUES ($1, $2, $3, '{"roles": {}}', '{"fields": []}', $4)`
	// insertShare: tenant $1, record $2, principal $3, kind $4, granter $5.
	insertShare = `INSERT INTO record_shares (tenant_id, record_id, principal_id, principal_kind, granted_by)
		VALUES ($1, $2, $3, $4, $5)`
)

var aclTables = []string{"acl_versions", "record_shares"}

// seedACL writes, as the superuser, for each tenant: seedPipeline's rows, ACL
// version 1 of the tenant's type and a share of its pipeline record with its
// principal (every id is the tenant id but the record's).
func seedACL(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	seedPipeline(t, super, tenants...)
	for _, id := range tenants {
		exec(t, super, insertACL, id, id, 1, id)
		exec(t, super, insertShare, id, pipelineRecord(id), id, "user", id)
	}
}

// Acceptance: both roles may SELECT and INSERT both tables and nothing more;
// no other role has any privilege on either. RLS is ENABLEd and FORCEd.
//
// "No other role" is every role in the cluster but superusers, the
// built-in pg_* roles and the owner nava_migrator, read from pg_roles rather
// than listed by hand, so a role a later migration adds (as ADR-004 added
// nava_audit_reader and nava_erasure) is checked without editing this test.
func TestP330Grants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	roles := queryString(t, super, `SELECT string_agg(rolname, ',' ORDER BY rolname) FROM pg_roles
		WHERE NOT rolsuper AND rolname NOT LIKE 'pg\_%' AND rolname <> 'nava_migrator'`)
	for _, must := range []string{"nava_app", "nava_worker", "nava_audit_reader", "nava_erasure"} {
		if !slices.Contains(strings.Split(roles, ","), must) {
			t.Fatalf("roles in the cluster = %s, want %s among them", roles, must)
		}
	}
	for _, table := range aclTables {
		for _, role := range strings.Split(roles, ",") {
			for _, priv := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"} {
				want := (role == "nava_app" || role == "nava_worker") && (priv == "SELECT" || priv == "INSERT")
				got := queryBool(t, super, "SELECT has_table_privilege($1, $2, $3)", role, "public."+table, priv)
				if got != want {
					t.Errorf("%s %s on %s = %v, want %v", role, priv, table, got, want)
				}
			}
		}
		if got := queryString(t, super, `SELECT relrowsecurity::text || relforcerowsecurity::text FROM pg_class
			WHERE oid = $1::regclass`, table); got != "truetrue" {
			t.Errorf("%s RLS enabled and forced = %s, want truetrue", table, got)
		}
	}
}

// Acceptance: tenant A cannot read or insert tenant B's rows as nava_app or
// nava_worker, and with no tenant or an empty one they see nothing.
func TestP330TenantIsolation(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedACL(t, super, tenantA, tenantB)

	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, table := range aclTables {
			for _, tenant := range []string{tenantA, "", "unset"} {
				var own, other int
				err := inTenantTx(conn, "", func(tx pgx.Tx) error {
					if tenant != "unset" {
						if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", tenant); err != nil {
							return err
						}
					}
					return tx.QueryRow(context.Background(),
						"SELECT count(*) FILTER (WHERE tenant_id = $1), count(*) FILTER (WHERE tenant_id <> $1) FROM "+table,
						tenantA).Scan(&own, &other)
				})
				wantOwn := 0
				if tenant == tenantA {
					wantOwn = 1
				}
				if err != nil || own != wantOwn || other != 0 {
					t.Errorf("%s reads %s with tenant %q: own %d, other %d, %v; want %d, 0", role, table, tenant, own, other, err, wantOwn)
				}
			}
		}

		// An insert for tenant B from tenant A's transaction fails the policy.
		_, err := execIn(conn, tenantA, insertACL, tenantB, tenantB, 2, tenantB)
		wantSQLState(t, role+" in tenant A publishes tenant B's ACL", err, "42501")
		_, err = execIn(conn, tenantA, insertShare, tenantB, pipelineRecord(tenantB), tenantB, "group", tenantB)
		wantSQLState(t, role+" in tenant A shares for tenant B", err, "42501")
		_, err = execIn(conn, "", insertACL, tenantA, tenantA, 2, tenantA)
		wantSQLState(t, role+" with no tenant publishes an ACL", err, "42501")
	}
	for _, role := range []string{"nava_onboard", "nava_groupsync", "nava_relay", "nava_purge", "nava_audit_reader", "nava_erasure"} {
		for _, table := range aclTables {
			_, err := execIn(asRole(t, role), tenantA, "SELECT count(*) FROM "+table)
			wantSQLState(t, role+" reads "+table, err, "42501")
		}
	}
}

// Acceptance: two versions of one type's ACL are inserted, and an UPDATE or
// DELETE of either table is refused.
func TestP330Immutable(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedACL(t, super, tenantA)

	for i, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		version := 2 + i
		if _, err := execIn(conn, tenantA, insertACL, tenantA, tenantA, version, tenantA); err != nil {
			t.Fatalf("%s publishes ACL version %d: %v", role, version, err)
		}
		_, err := execIn(conn, tenantA, insertACL, tenantA, tenantA, version, tenantA)
		wantSQLState(t, role+" republishes an existing version", err, "23505")
		_, err = execIn(conn, tenantA, insertACL, tenantA, tenantA, 0, tenantA)
		wantSQLState(t, role+" publishes version 0", err, "23514")
		for _, q := range []string{
			"UPDATE acl_versions SET rules = '{}' WHERE tenant_id = $1",
			"DELETE FROM acl_versions WHERE tenant_id = $1",
			"UPDATE record_shares SET principal_kind = 'group' WHERE tenant_id = $1",
			"DELETE FROM record_shares WHERE tenant_id = $1",
		} {
			_, err := execIn(conn, tenantA, q, tenantA)
			wantSQLState(t, role+": "+q, err, "42501")
		}
	}
	if got := queryString(t, super, `SELECT string_agg(version::text, ',' ORDER BY version) FROM acl_versions
		WHERE tenant_id = $1 AND entity_definition_id = $1`, tenantA); got != "1,2,3" {
		t.Errorf("tenant A's ACL versions = %s, want 1,2,3", got)
	}
	// The evaluator's lookup: the newest version, by the primary key.
	if got := queryString(t, super, `SELECT version::text FROM acl_versions
		WHERE tenant_id = $1 AND entity_definition_id = $1 ORDER BY version DESC LIMIT 1`, tenantA); got != "3" {
		t.Errorf("newest ACL version = %s, want 3", got)
	}
}

// Acceptance: the composite foreign keys hold the tenant, though the policy
// passes the row: tenant A cannot share tenant B's record, publish an ACL for
// tenant B's type, or name tenant B's principal as publisher or granter.
func TestP330ForeignKeysRejectCrossTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedACL(t, super, tenantA, tenantB)
	// A tenant B record whose id tenant A does not have.
	other := "0b000000-0000-4000-8000-0000000000b7"
	exec(t, super, withRecordEntry("INSERT INTO records (tenant_id, id) VALUES ($1, $2)"), tenantB, other)

	app := asRole(t, "nava_app")
	_, err := execIn(app, tenantA, insertShare, tenantA, other, tenantA, "user", tenantA)
	wantSQLState(t, "tenant A shares tenant B's record", err, "23503")
	_, err = execIn(app, tenantA, insertACL, tenantA, tenantB, 2, tenantA)
	wantSQLState(t, "tenant A publishes an ACL for tenant B's type", err, "23503")
	_, err = execIn(app, tenantA, insertACL, tenantA, tenantA, 2, tenantB)
	wantSQLState(t, "tenant A names tenant B's principal as publisher", err, "23503")
	_, err = execIn(app, tenantA, insertShare, tenantA, pipelineRecord(tenantA), tenantB, "group", tenantB)
	wantSQLState(t, "tenant A names tenant B's principal as granter", err, "23503")

	// principal_kind is user or group; a user and a group with the same id
	// are two shares, and a repeat of either is refused.
	_, err = execIn(app, tenantA, insertShare, tenantA, pipelineRecord(tenantA), tenantA, "role", tenantA)
	wantSQLState(t, "a share with principal_kind role", err, "23514")
	if _, err := execIn(app, tenantA, insertShare, tenantA, pipelineRecord(tenantA), tenantA, "group", nil); err != nil {
		t.Errorf("sharing with a group whose id a user also has: %v", err)
	}
	_, err = execIn(app, tenantA, insertShare, tenantA, pipelineRecord(tenantA), tenantA, "user", tenantA)
	wantSQLState(t, "a repeat share", err, "23505")
}

// The index set is the two primary keys and the list join's
// record_shares_principal, each led by tenant_id.
func TestP330IndexSet(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	got := queryString(t, super, `SELECT string_agg(indexrelid::regclass::text || ' ' || pg_get_indexdef(indexrelid), E'\n' ORDER BY indexrelid::regclass::text)
		FROM pg_index WHERE indrelid IN ('acl_versions'::regclass, 'record_shares'::regclass)`)
	want := "acl_versions_pkey CREATE UNIQUE INDEX acl_versions_pkey ON public.acl_versions USING btree (tenant_id, entity_definition_id, version)\n" +
		"record_shares_pkey CREATE UNIQUE INDEX record_shares_pkey ON public.record_shares USING btree (tenant_id, record_id, principal_kind, principal_id)\n" +
		"record_shares_principal CREATE INDEX record_shares_principal ON public.record_shares USING btree (tenant_id, principal_id)"
	if got != want {
		t.Errorf("index set:\n%s\nwant:\n%s", got, want)
	}
}

// aclTablesMigration creates acl_versions and record_shares.
const aclTablesMigration = "20261108124249_acl_tables.sql"

// beforeACLTables is the pair of manifests for a run of the migrations in
// fsys: when fsys stops before aclTablesMigration, both tables are left out
// of the grants and audited manifests, which name tables the run has not
// created yet (the grant step fails on a missing table). Otherwise m and a
// are returned unchanged. Tests that migrate to an earlier version use it.
func beforeACLTables(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	if _, err := fs.Stat(fsys, aclTablesMigration); err == nil {
		return m, a
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		tables := make(map[string]map[string][]string, len(r.Tables))
		for tbl, privs := range r.Tables {
			if !slices.Contains(aclTables, tbl) {
				tables[tbl] = privs
			}
		}
		r.Tables = tables
		gm.Roles[name] = r
	}
	am := *a
	am.Audited = make(map[string]audited.Table, len(a.Audited))
	for name, e := range a.Audited {
		if !slices.Contains(aclTables, name) {
			am.Audited[name] = e
		}
	}
	return &gm, &am
}
