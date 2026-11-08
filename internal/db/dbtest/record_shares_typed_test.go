//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// P3-30a (TICK-625, closes AR-S10-01): record_shares names its principal by
// user_id (-> principals) or group_id (-> groups), exactly one, the one
// principal_kind says; principal_id is generated from them.

// beforeRecordSharesTyped is the version just before
// 20261108184658_record_shares_typed_principals.sql.
const beforeRecordSharesTyped = 20261108184657

// groupOnly and userOnly are tenant A ids that only a group or only a user
// has; seedACL's tenant-id principal is both.
func groupOnly(tenant string) string { return tenant[:len(tenant)-3] + "a01" }
func userOnly(tenant string) string  { return tenant[:len(tenant)-3] + "a02" }

// Acceptance: a group id as a user, a user id as a group, and an id the
// tenant does not have are refused (23503), as is another tenant's user or
// group; a kind that does not match the set column, both set, or neither, is
// refused; principal_id cannot be written. A user share and a group share are
// accepted, and principal_id is the id each names.
func TestTICK625SharePrincipalIsTyped(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedACL(t, super, tenantA, tenantB)
	exec(t, super, insertShareGroup, tenantA, groupOnly(tenantA))
	exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, 'kc-user-only', true, now())`, tenantA, userOnly(tenantA))
	const missing = "0190a000-0000-7000-8000-0000000dead0"
	rec := pipelineRecord(tenantA)

	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, c := range []struct {
			what         string
			user, group  any
			kind, sqlErr string
		}{
			{"a group id as a user", groupOnly(tenantA), nil, "user", "23503"},
			{"a user id as a group", nil, userOnly(tenantA), "group", "23503"},
			{"an id no user of the tenant has", missing, nil, "user", "23503"},
			{"an id no group of the tenant has", nil, missing, "group", "23503"},
			{"tenant B's user", tenantB, nil, "user", "23503"},
			{"tenant B's group", nil, tenantB, "group", "23503"},
			{"a user id under kind group", userOnly(tenantA), nil, "group", "23514"},
			{"a group id under kind user", nil, groupOnly(tenantA), "user", "23514"},
			{"both a user and a group", userOnly(tenantA), groupOnly(tenantA), "user", "23514"},
			{"neither a user nor a group", nil, nil, "user", "23502"},
		} {
			_, err := execIn(conn, tenantA, insertShare, tenantA, rec, c.user, c.group, c.kind, tenantA)
			wantSQLState(t, role+" shares with "+c.what, err, c.sqlErr)
		}
		_, err := execIn(conn, tenantA, `INSERT INTO record_shares (tenant_id, record_id, principal_id, principal_kind)
			VALUES ($1, $2, $3, 'user')`, tenantA, rec, userOnly(tenantA))
		wantSQLState(t, role+" writes principal_id", err, "428C9")
	}

	app := asRole(t, "nava_app")
	if _, err := execIn(app, tenantA, insertShare, tenantA, rec, userOnly(tenantA), nil, "user", tenantA); err != nil {
		t.Errorf("a user share: %v", err)
	}
	if _, err := execIn(app, tenantA, insertShare, tenantA, rec, nil, groupOnly(tenantA), "group", tenantA); err != nil {
		t.Errorf("a group share: %v", err)
	}
	got := queryString(t, super, `SELECT string_agg(principal_kind || ' ' || principal_id::text, ',' ORDER BY principal_kind, principal_id)
		FROM record_shares WHERE tenant_id = $1 AND principal_id IN ($2, $3)`, tenantA, userOnly(tenantA), groupOnly(tenantA))
	if want := "group " + groupOnly(tenantA) + ",user " + userOnly(tenantA); got != want {
		t.Errorf("principal_id of the two shares = %s, want %s", got, want)
	}
}

// Acceptance: the migration goes down and up. Down keeps a share's
// principal_id as a written column and restores the kind check; up again on a
// table that holds an old share fails and changes nothing; once the share is
// gone it applies, and the index set is the same as before.
func TestTICK625MigrationDownUp(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedACL(t, super, tenantA)

	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.DownTo(ctx, beforeRecordSharesTyped); err != nil {
		t.Fatalf("down: %v", err)
	}
	columns := `SELECT string_agg(attname || CASE WHEN attgenerated <> '' THEN ' generated' ELSE '' END, ',' ORDER BY attname)
		FROM pg_attribute WHERE attrelid = 'record_shares'::regclass AND attnum > 0 AND NOT attisdropped`
	if got := queryString(t, super, columns); got != "granted_at,granted_by,principal_id,principal_kind,record_id,tenant_id" {
		t.Errorf("after down: columns %s", got)
	}
	if got := queryString(t, super, `SELECT principal_id::text FROM record_shares WHERE tenant_id = $1`, tenantA); got != tenantA {
		t.Errorf("after down: principal_id = %s, want %s", got, tenantA)
	}
	_, err = super.Exec(ctx, `INSERT INTO record_shares (tenant_id, record_id, principal_id, principal_kind) VALUES ($1, $2, $3, 'role')`,
		tenantA, pipelineRecord(tenantA), userOnly(tenantA))
	wantSQLState(t, "after down: a share with principal_kind role", err, "23514")

	if _, err := p.Up(ctx); err == nil {
		t.Fatal("up again with an old share: want an error")
	}
	if got := queryString(t, super, columns); got != "granted_at,granted_by,principal_id,principal_kind,record_id,tenant_id" {
		t.Errorf("after the failed up: columns %s, want them unchanged", got)
	}

	exec(t, super, "DELETE FROM record_shares")
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := queryString(t, super, columns); got != "granted_at,granted_by,group_id,principal_id generated,principal_kind,record_id,tenant_id,user_id" {
		t.Errorf("after up again: columns %s", got)
	}
	got := queryString(t, super, `SELECT string_agg(conname || ' ' || pg_get_constraintdef(oid), E'\n' ORDER BY conname)
		FROM pg_constraint WHERE conrelid = 'record_shares'::regclass`)
	want := "record_shares_granted_by_fkey FOREIGN KEY (tenant_id, granted_by) REFERENCES principals(tenant_id, id)\n" +
		"record_shares_group_fkey FOREIGN KEY (tenant_id, group_id) REFERENCES groups(tenant_id, id)\n" +
		"record_shares_pkey PRIMARY KEY (tenant_id, record_id, principal_kind, principal_id)\n" +
		"record_shares_principal_check CHECK ((((principal_kind = 'user'::text) AND (user_id IS NOT NULL) AND (group_id IS NULL)) OR ((principal_kind = 'group'::text) AND (group_id IS NOT NULL) AND (user_id IS NULL))))\n" +
		"record_shares_record_fkey FOREIGN KEY (tenant_id, record_id) REFERENCES records(tenant_id, id)\n" +
		"record_shares_user_fkey FOREIGN KEY (tenant_id, user_id) REFERENCES principals(tenant_id, id)"
	if got != want {
		t.Errorf("constraints:\n%s\nwant:\n%s", got, want)
	}
}
