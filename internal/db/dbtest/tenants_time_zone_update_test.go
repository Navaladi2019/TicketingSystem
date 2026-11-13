//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// TICK-795 (for TICK-672, ADR-005 §3.6): nava_app sets its own tenant's
// time_zone and nothing else on tenants.

const tenantsTimeZoneUpdateBefore = 20261113105959

// Acceptance: the policy is FOR UPDATE TO nava_app with the canonical
// predicate on id in USING and WITH CHECK, and nava_app may update
// time_zone and no other tenants column.
func TestTick795PolicyAndGrants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	const canon = "(id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)"
	got := queryString(t, super, `SELECT cmd || '|' || array_to_string(roles, ',') || '|' || coalesce(qual, '') || '|' || coalesce(with_check, '')
		FROM pg_policies WHERE schemaname = 'public' AND tablename = 'tenants' AND policyname = 'tenants_time_zone_update'`)
	if want := "UPDATE|nava_app|" + canon + "|" + canon; got != want {
		t.Errorf("tenants_time_zone_update = %q, want %q", got, want)
	}
	for priv, want := range map[string]string{"UPDATE": "time_zone", "SELECT": "id,state,time_zone"} {
		got := queryString(t, super, `SELECT coalesce(string_agg(attname, ',' ORDER BY attname), '') FROM pg_attribute
			WHERE attrelid = 'tenants'::regclass AND attnum > 0 AND NOT attisdropped
			  AND has_column_privilege('nava_app', attrelid, attnum, $1)`, priv)
		if got != want {
			t.Errorf("nava_app %s columns of tenants = %q, want %q", priv, got, want)
		}
	}
	for _, priv := range []string{"INSERT", "DELETE", "TRUNCATE"} {
		if queryBool(t, super, "SELECT has_table_privilege('nava_app', 'public.tenants', $1)", priv) {
			t.Errorf("nava_app has %s on tenants", priv)
		}
	}
}

// Acceptance: nava_app in tenant A sets and reads A's time_zone, never B's;
// with no tenant or an empty one it updates nothing; it cannot touch state
// or the id; the shape CHECK still refuses a bad name; no other runtime role
// updates tenants.
func TestTick795TenantIsolation(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	app := asRole(t, "nava_app")

	const set = "UPDATE tenants SET time_zone = 'Asia/Kolkata' WHERE id IN ($1, $2)"
	if n, err := execIn(app, tenantA, set, tenantA, tenantB); err != nil || n != 1 {
		t.Errorf("nava_app in A sets A's and B's time_zone: %d rows, %v; want 1, nil", n, err)
	}
	if got := queryString(t, super, "SELECT string_agg(time_zone, ',' ORDER BY id) FROM tenants WHERE id IN ($1, $2)",
		tenantA, tenantB); got != "Asia/Kolkata,UTC" {
		t.Errorf("time_zone of A,B = %s, want Asia/Kolkata,UTC", got)
	}
	// nava_app reads the value back through tenants_tenant.
	var tz string
	if err := inTenantTx(app, tenantA, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT time_zone FROM tenants WHERE id = $1", tenantA).Scan(&tz)
	}); err != nil || tz != "Asia/Kolkata" {
		t.Errorf("nava_app in A reads its time_zone: %q, %v; want Asia/Kolkata", tz, err)
	}
	// No tenant, then an empty one: both update nothing.
	for _, empty := range []bool{false, true} {
		var n int64
		err := inTenantTx(app, "", func(tx pgx.Tx) error {
			if empty {
				if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', '', true)"); err != nil {
					return err
				}
			}
			tag, err := tx.Exec(context.Background(), "UPDATE tenants SET time_zone = 'Etc/GMT+5' WHERE id IN ($1, $2)", tenantA, tenantB)
			n = tag.RowsAffected()
			return err
		})
		if err != nil || n != 0 {
			t.Errorf("nava_app with empty tenant %v: %d rows, %v; want 0, nil", empty, n, err)
		}
	}

	_, err := execIn(app, tenantA, "UPDATE tenants SET state = 'provisioning' WHERE id = $1", tenantA)
	wantSQLState(t, "nava_app updates its tenant's state", err, "42501")
	_, err = execIn(app, tenantA, "UPDATE tenants SET id = $2 WHERE id = $1", tenantA, tenantB)
	wantSQLState(t, "nava_app rewrites its tenant's id", err, "42501")
	_, err = execIn(app, tenantA, "DELETE FROM tenants WHERE id = $1", tenantA)
	wantSQLState(t, "nava_app deletes its tenant", err, "42501")
	_, err = execIn(app, tenantA, "UPDATE tenants SET time_zone = '+05:30' WHERE id = $1", tenantA)
	if code, con := pgErrDetail(err); code != "23514" || con != "tenants_time_zone_check" {
		t.Errorf("nava_app sets a bad time_zone: %v (constraint %q), want 23514 on tenants_time_zone_check", err, con)
	}

	for _, role := range []string{"nava_worker", "nava_onboard", "nava_groupsync", "nava_relay", "nava_purge", "nava_audit_reader", "nava_erasure"} {
		_, err := execIn(asRole(t, role), tenantA, "UPDATE tenants SET time_zone = 'UTC' WHERE id = $1", tenantA)
		wantSQLState(t, role+" sets time_zone", err, "42501")
	}
}

// Acceptance: the migration goes down (the policy is gone and nava_app's
// update matches no row) and up again.
func TestTick795Migration(t *testing.T) {
	m := migratedCluster(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	app := asRole(t, "nava_app")
	const set = "UPDATE tenants SET time_zone = 'Asia/Kolkata' WHERE id = $1"
	const policies = "SELECT count(*)::text FROM pg_policies WHERE tablename = 'tenants' AND policyname = 'tenants_time_zone_update'"

	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.DownTo(ctx, tenantsTimeZoneUpdateBefore); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := queryString(t, super, policies); got != "0" {
		t.Errorf("after down: %s tenants_time_zone_update policies, want 0", got)
	}
	if n, err := execIn(app, tenantA, set, tenantA); err != nil || n != 0 {
		t.Errorf("after down: nava_app sets its time_zone: %d rows, %v; want 0, nil (no UPDATE policy)", n, err)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := queryString(t, super, policies); got != "1" {
		t.Errorf("after up: %s tenants_time_zone_update policies, want 1", got)
	}
	if n, err := execIn(app, tenantA, set, tenantA); err != nil || n != 1 {
		t.Errorf("after up: nava_app sets its time_zone: %d rows, %v; want 1, nil", n, err)
	}
}
