//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// TICK-868 (20261115100000_tenants_select_worker.sql): nava_worker reads
// tenants (id, time_zone) under tenants_tenant, so a worker create on a
// dated number format gets its tenant's zone (ADR-005 §3.6, A7).

// tenantsSelectWorkerBefore is the version just before the TICK-868
// migration, so DownTo rolls back only that one.
const tenantsSelectWorkerBefore = 20261115090100

// workerZoneSQL is records/store's scopeZoneSQL subquery with no format zone.
const workerZoneSQL = `SELECT coalesce((SELECT t.time_zone FROM tenants t WHERE t.id = $1), '<none>')`

// Acceptance: tenants_tenant is FOR SELECT TO nava_app, nava_worker with
// the canonical predicate on id; nava_worker may SELECT id and time_zone
// and nothing else on tenants.
func TestTICK868PolicyAndGrants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	const canon = "(id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)"
	got := queryString(t, super, `SELECT cmd || '|' || array_to_string(roles, ',') || '|' || coalesce(qual, '') || '|' || coalesce(with_check, '')
		FROM pg_policies WHERE schemaname = 'public' AND tablename = 'tenants' AND policyname = 'tenants_tenant'`)
	if want := "SELECT|nava_app,nava_worker|" + canon + "|"; got != want {
		t.Errorf("tenants_tenant = %q, want %q", got, want)
	}
	for priv, want := range map[string]string{"SELECT": "id,time_zone", "INSERT": "", "UPDATE": "", "REFERENCES": ""} {
		got := queryString(t, super, `SELECT coalesce(string_agg(attname, ',' ORDER BY attname), '') FROM pg_attribute
			WHERE attrelid = 'tenants'::regclass AND attnum > 0 AND NOT attisdropped
			  AND has_column_privilege('nava_worker', attrelid, attnum, $1)`, priv)
		if got != want {
			t.Errorf("nava_worker %s columns of tenants = %q, want %q", priv, got, want)
		}
	}
	for _, priv := range []string{"DELETE", "TRUNCATE"} {
		if queryBool(t, super, "SELECT has_table_privilege('nava_worker', 'public.tenants', $1)", priv) {
			t.Errorf("nava_worker has %s on tenants", priv)
		}
	}
}

// Acceptance (cross-tenant): a worker in tenant A reads A's time_zone and
// never B's, by key or by scan; with no tenant or an empty one it reads no
// row; it reads no other column and writes nothing.
func TestTICK868WorkerTenantZoneStaysInTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	exec(t, super, "UPDATE tenants SET time_zone = 'Asia/Kolkata' WHERE id = $1", tenantA)
	exec(t, super, "UPDATE tenants SET time_zone = 'America/New_York' WHERE id = $1", tenantB)
	worker := asRole(t, "nava_worker")
	zone := func(ctxTenant string, empty bool, of string) string {
		t.Helper()
		var tz string
		if err := inTenantTx(worker, ctxTenant, func(tx pgx.Tx) error {
			if empty {
				if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', '', true)"); err != nil {
					return err
				}
			}
			return tx.QueryRow(context.Background(), workerZoneSQL, of).Scan(&tz)
		}); err != nil {
			t.Fatalf("nava_worker reads %s's zone from tenant %q: %v", of, ctxTenant, err)
		}
		return tz
	}
	if got := zone(tenantA, false, tenantA); got != "Asia/Kolkata" {
		t.Errorf("from A, A's zone = %q, want Asia/Kolkata", got)
	}
	if got := zone(tenantA, false, tenantB); got != "<none>" {
		t.Errorf("from A, B's zone = %q, want no row", got)
	}
	if got := zone(tenantB, false, tenantB); got != "America/New_York" {
		t.Errorf("from B, B's zone = %q, want America/New_York", got)
	}
	for _, empty := range []bool{false, true} {
		if got := zone("", empty, tenantA); got != "<none>" {
			t.Errorf("no tenant (empty %v), A's zone = %q, want no row", empty, got)
		}
	}
	var n int
	if err := inTenantTx(worker, tenantA, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT count(*) FROM tenants WHERE time_zone = 'America/New_York' OR id <> $1", tenantA).Scan(&n)
	}); err != nil || n != 0 {
		t.Errorf("from A, other tenants by scan: %d rows, %v; want 0, nil", n, err)
	}
	for _, col := range []string{"state", "onboarding_request_hash", "created_at", "*"} {
		_, err := execIn(worker, tenantA, "SELECT "+col+" FROM tenants")
		wantSQLState(t, "nava_worker SELECT "+col, err, "42501")
	}
	for name, stmt := range map[string]string{
		"INSERT": "INSERT INTO tenants (id, state) VALUES (gen_random_uuid(), 'provisioning') RETURNING $1::uuid",
		"UPDATE": "UPDATE tenants SET time_zone = 'UTC' WHERE id = $1",
		"DELETE": "DELETE FROM tenants WHERE id = $1",
	} {
		_, err := execIn(worker, tenantA, stmt, tenantA)
		wantSQLState(t, "nava_worker "+name, err, "42501")
	}
}

// Acceptance: the migration goes down (the worker reads no zone) and up
// again (it reads its own).
func TestTICK868Migration(t *testing.T) {
	m := migratedCluster(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	worker := asRole(t, "nava_worker")
	read := func() string {
		t.Helper()
		var tz string
		if err := inTenantTx(worker, tenantA, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), workerZoneSQL, tenantA).Scan(&tz)
		}); err != nil {
			t.Fatalf("nava_worker reads its zone: %v", err)
		}
		return tz
	}
	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.DownTo(ctx, tenantsSelectWorkerBefore); err != nil {
		t.Fatalf("down: %v", err)
	}
	// The column grant stays (grants.yaml is applied by nava-db migrate, not
	// by goose); without the policy FORCE RLS still hides every row.
	if got := read(); got != "<none>" {
		t.Errorf("after down: worker reads zone %q, want no row", got)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := read(); got != "UTC" {
		t.Errorf("after up: worker reads zone %q, want UTC", got)
	}
}
