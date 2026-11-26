//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// TICK-1070 (S31; ADR-010-A2 A2-9, PLAN-011-A2 D-11A2-2):
// entity_record_estimates holds N-hat_e, the live records of one entity type
// in one tenant. nava_app reads its own tenant's rows; only nava_worker, the
// refresh job's role, writes them. The job lists tenants through
// binding_lookup, with a grant on tenant_identity_binding.tenant_id only.

const (
	entityRecordEstimatesMigration = "20261121163600_entity_record_estimates.sql"
	bindingLookupWorkerMigration   = "20261121163700_binding_lookup_worker.sql"

	// upsertEstimate is the refresh job's write: tenant $1, type $2, count
	// $3, at $4.
	upsertEstimate = `INSERT INTO entity_record_estimates (tenant_id, entity_definition_id, live_records, refreshed_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id, entity_definition_id)
		DO UPDATE SET live_records = EXCLUDED.live_records, refreshed_at = EXCLUDED.refreshed_at`
	insertEstimate = `INSERT INTO entity_record_estimates (tenant_id, entity_definition_id, live_records, refreshed_at)
		VALUES ($1, $2, $3, now())`
)

// beforeEntityRecordEstimates leaves entity_record_estimates out of both
// manifests when fsys lacks its migration, so a run that stops at an earlier
// version applies only grants on tables it has created.
func beforeEntityRecordEstimates(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	if _, err := fs.Stat(fsys, entityRecordEstimatesMigration); err == nil {
		return m, a
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		r.Tables = maps.Clone(r.Tables)
		delete(r.Tables, "entity_record_estimates")
		gm.Roles[name] = r
	}
	am := *a
	am.Exempt = maps.Clone(a.Exempt)
	delete(am.Exempt, "entity_record_estimates")
	return &gm, &am
}

// seedEstimateTypes gives each tenant one entity type whose id is the
// tenant id, with its config entry.
func seedEstimateTypes(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	seedTenants(t, super, tenants, nil)
	for _, id := range tenants {
		exec(t, super, insertType, id, "ticket", "TCK")
	}
}

// The contract's columns, keys and CHECK; the primary key is the only index;
// FORCE RLS with one app.tenant_id policy per command and no DELETE policy.
func TestTICK1070Schema(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	got := queryString(t, super, `SELECT string_agg(format('%s %s %s', attname, format_type(atttypid, atttypmod),
			CASE WHEN attnotnull THEN 'not null' ELSE 'null' END), ', ' ORDER BY attnum)
		FROM pg_attribute WHERE attrelid = 'entity_record_estimates'::regclass AND attnum > 0 AND NOT attisdropped`)
	if want := "tenant_id uuid not null, entity_definition_id uuid not null, live_records bigint not null, refreshed_at timestamp with time zone not null"; got != want {
		t.Errorf("columns = %s, want %s", got, want)
	}
	for name, want := range map[string]string{
		"entity_record_estimates_pkey":               "PRIMARY KEY (tenant_id, entity_definition_id)",
		"entity_record_estimates_live_records_check": "CHECK ((live_records >= 0))",
		"entity_record_estimates_definition_fkey":    "FOREIGN KEY (tenant_id, entity_definition_id) REFERENCES entity_definitions(tenant_id, id)",
	} {
		if got := queryString(t, super, "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = $1", name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
		if !queryBool(t, super, "SELECT convalidated FROM pg_constraint WHERE conname = $1", name) {
			t.Errorf("%s is not validated", name)
		}
	}
	if got := queryString(t, super, "SELECT string_agg(indexrelid::regclass::text, ',') FROM pg_index WHERE indrelid = 'entity_record_estimates'::regclass"); got != "entity_record_estimates_pkey" {
		t.Errorf("indexes = %s, want entity_record_estimates_pkey only", got)
	}
	if !queryBool(t, super, "SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid = 'entity_record_estimates'::regclass") {
		t.Error("entity_record_estimates is not under ENABLE + FORCE RLS")
	}
	const pred = "(tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)"
	got = queryString(t, super, `SELECT string_agg(format('%s %s %s %s %s', polname, polcmd,
			(SELECT string_agg(rolname, ',' ORDER BY rolname) FROM pg_roles WHERE oid = ANY (polroles)),
			coalesce(pg_get_expr(polqual, polrelid), '-'), coalesce(pg_get_expr(polwithcheck, polrelid), '-')), E'\n' ORDER BY polname)
		FROM pg_policy WHERE polrelid = 'entity_record_estimates'::regclass`)
	want := strings.Join([]string{
		"entity_record_estimates_insert a nava_worker - " + pred,
		"entity_record_estimates_select r nava_app,nava_worker " + pred + " -",
		"entity_record_estimates_update w nava_worker " + pred + " " + pred,
	}, "\n")
	if got != want {
		t.Errorf("policies:\n%s\nwant:\n%s", got, want)
	}
	// Grants: nava_app reads; nava_worker reads, inserts and updates the two
	// value columns; no DELETE or TRUNCATE for anyone, and nothing for the
	// other runtime roles.
	got = queryString(t, super, `SELECT coalesce(string_agg(format('%s %s', grantee, privilege_type), ', ' ORDER BY grantee, privilege_type), '')
		FROM information_schema.role_table_grants
		WHERE table_name = 'entity_record_estimates' AND grantee <> 'nava_migrator'`)
	if want := "nava_app SELECT, nava_worker INSERT, nava_worker SELECT"; got != want {
		t.Errorf("table grants = %s, want %s", got, want)
	}
	got = queryString(t, super, `SELECT coalesce(string_agg(format('%s %s', grantee, column_name), ', ' ORDER BY grantee, column_name), '')
		FROM information_schema.column_privileges
		WHERE table_name = 'entity_record_estimates' AND privilege_type = 'UPDATE' AND grantee <> 'nava_migrator'`)
	if want := "nava_worker live_records, nava_worker refreshed_at"; got != want {
		t.Errorf("UPDATE column grants = %s, want %s", got, want)
	}
}

// Acceptance 1: with tenant A set, nava_app (and nava_worker) see only A's
// row of a same-shaped type both tenants hold; with no tenant, or an empty
// one, they see none.
func TestTICK1070TenantScoping(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedEstimateTypes(t, super, tenantA, tenantB)
	exec(t, super, insertEstimate, tenantA, tenantA, 10)
	exec(t, super, insertEstimate, tenantB, tenantB, 20)

	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, mode := range []string{"A", "empty", "unset"} {
			var own, other int
			err := inTenantTx(conn, "", func(tx pgx.Tx) error {
				if mode != "unset" {
					v := map[string]string{"A": tenantA, "empty": ""}[mode]
					if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", v); err != nil {
						return err
					}
				}
				return tx.QueryRow(context.Background(),
					"SELECT count(*) FILTER (WHERE tenant_id = $1), count(*) FILTER (WHERE tenant_id <> $1) FROM entity_record_estimates",
					tenantA).Scan(&own, &other)
			})
			if err != nil {
				t.Fatalf("%s reads with tenant %s: %v", role, mode, err)
			}
			wantOwn := 0
			if mode == "A" {
				wantOwn = 1
			}
			if own != wantOwn || other != 0 {
				t.Errorf("%s with tenant %s: own %d, other tenants %d; want %d, 0", role, mode, own, other, wantOwn)
			}
		}
		// B's row by its full key, under A.
		if n, err := execIn(conn, tenantA, "SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $1", tenantB); err != nil || n != 0 {
			t.Errorf("%s reads B's row by its key under A: %d rows, %v; want 0, nil", role, n, err)
		}
	}
}

// Acceptance 2: nava_app may not write; nava_worker's INSERT and upsert
// succeed under its own tenant and are refused by RLS for another tenant's
// row. Nobody deletes, and the key columns are never updated.
func TestTICK1070WriteRole(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedEstimateTypes(t, super, tenantA, tenantB)
	now := time.Now()

	app := asRole(t, "nava_app")
	_, err := execIn(app, tenantA, insertEstimate, tenantA, tenantA, 1)
	wantSQLState(t, "nava_app inserts", err, "42501")
	_, err = execIn(app, tenantA, upsertEstimate, tenantA, tenantA, 1, now)
	wantSQLState(t, "nava_app upserts", err, "42501")
	_, err = execIn(app, tenantA, "UPDATE entity_record_estimates SET live_records = 0 WHERE tenant_id = $1", tenantA)
	wantSQLState(t, "nava_app updates", err, "42501")

	worker := asRole(t, "nava_worker")
	if n, err := execIn(worker, tenantA, insertEstimate, tenantA, tenantA, 5); err != nil || n != 1 {
		t.Fatalf("nava_worker inserts A's row: %d, %v; want 1, nil", n, err)
	}
	if n, err := execIn(worker, tenantA, upsertEstimate, tenantA, tenantA, 6, now); err != nil || n != 1 {
		t.Errorf("nava_worker upserts A's row: %d, %v; want 1, nil", n, err)
	}
	// B's row under A: the INSERT's WITH CHECK refuses it before the FK or
	// the conflict arbiter is reached.
	_, err = execIn(worker, tenantA, insertEstimate, tenantB, tenantB, 5)
	wantSQLState(t, "nava_worker inserts B's row under A", err, "42501")
	_, err = execIn(worker, tenantA, upsertEstimate, tenantB, tenantB, 5, now)
	wantSQLState(t, "nava_worker upserts B's row under A", err, "42501")
	// With no tenant set, nothing passes the WITH CHECK either.
	_, err = execIn(worker, "", upsertEstimate, tenantA, tenantA, 5, now)
	wantSQLState(t, "nava_worker upserts with no tenant", err, "42501")

	// B's existing row cannot be reached from A: the UPDATE finds nothing,
	// and the upsert's INSERT is refused even when B's row would conflict.
	exec(t, super, insertEstimate, tenantB, tenantB, 50)
	if n, err := execIn(worker, tenantA, "UPDATE entity_record_estimates SET live_records = 0 WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
		t.Errorf("nava_worker updates B's row under A: %d, %v; want 0, nil", n, err)
	}
	_, err = execIn(worker, tenantA, upsertEstimate, tenantB, tenantB, 0, now)
	wantSQLState(t, "nava_worker upserts over B's existing row under A", err, "42501")

	_, err = execIn(worker, tenantA, "DELETE FROM entity_record_estimates WHERE tenant_id = $1", tenantA)
	wantSQLState(t, "nava_worker deletes", err, "42501")
	_, err = execIn(worker, tenantA, "UPDATE entity_record_estimates SET entity_definition_id = entity_definition_id WHERE tenant_id = $1", tenantA)
	wantSQLState(t, "nava_worker updates the key", err, "42501")
	for _, role := range []string{"nava_groupsync", "nava_relay", "nava_purge"} {
		_, err := execIn(asRole(t, role), tenantA, "SELECT 1 FROM entity_record_estimates")
		wantSQLState(t, role+" reads", err, "42501")
	}

	if got := queryString(t, super, "SELECT live_records::text FROM entity_record_estimates WHERE tenant_id = $1", tenantB); got != "50" {
		t.Errorf("B's live_records = %s after A's writes, want 50", got)
	}
}

// Acceptance 3: a negative count is refused by the CHECK, and a row for an
// unknown type, or another tenant's type, by the FK.
func TestTICK1070Constraints(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedEstimateTypes(t, super, tenantA, tenantB)
	worker := asRole(t, "nava_worker")

	_, err := execIn(worker, tenantA, insertEstimate, tenantA, tenantA, -1)
	wantSQLState(t, "live_records = -1", err, "23514")
	_, err = execIn(worker, tenantA, insertEstimate, tenantA, uuid.NewString(), 1)
	wantSQLState(t, "an unknown entity_definition_id", err, "23503")
	// tenant B's type id under tenant A's tenant_id: the FK carries the tenant.
	_, err = execIn(worker, tenantA, insertEstimate, tenantA, tenantB, 1)
	wantSQLState(t, "tenant B's entity type under tenant A", err, "23503")
}

// Acceptance 4: the same upsert twice leaves one row, with the second run's
// count and refreshed_at.
func TestTICK1070IdempotentUpsert(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedEstimateTypes(t, super, tenantA)
	worker := asRole(t, "nava_worker")

	first := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	second := first.Add(time.Hour)
	for _, run := range []struct {
		at time.Time
		n  int
	}{{first, 7}, {second, 9}} {
		if n, err := execIn(worker, tenantA, upsertEstimate, tenantA, tenantA, run.n, run.at); err != nil || n != 1 {
			t.Fatalf("upsert at %s: %d, %v; want 1, nil", run.at, n, err)
		}
	}
	got := queryString(t, super, `SELECT format('%s %s %s', count(*), max(live_records), max(refreshed_at) = $2)
		FROM entity_record_estimates WHERE tenant_id = $1`, tenantA, second)
	if got != "1 9 t" {
		t.Errorf("after two upserts: %s, want one row with 9 and the second refreshed_at", got)
	}
}

// Acceptance 5: nava_worker reads tenant_identity_binding.tenant_id for every
// tenant (binding_lookup), and no other column.
func TestTICK1070WorkerReadsBindingTenantOnly(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	worker := asRole(t, "nava_worker")

	// The refresh job's tenant list: no tenant set, every bound tenant.
	var got []string
	if err := inTenantTx(worker, "", func(tx pgx.Tx) error {
		rows, err := tx.Query(context.Background(), "SELECT DISTINCT tenant_id::text FROM tenant_identity_binding ORDER BY 1")
		if err != nil {
			return err
		}
		got, err = pgx.CollectRows(rows, pgx.RowTo[string])
		return err
	}); err != nil {
		t.Fatalf("nava_worker lists tenants: %v", err)
	}
	if want := []string{tenantA, tenantB}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("tenants = %v, want %v", got, want)
	}
	for _, col := range []string{"issuer", "org_id", "idp_aliases", "created_at", "*"} {
		_, err := execIn(worker, "", "SELECT "+col+" FROM tenant_identity_binding")
		wantSQLState(t, "nava_worker reads "+col, err, "42501")
	}
	got2 := queryString(t, super, `SELECT coalesce(string_agg(format('%s %s', privilege_type, column_name), ', ' ORDER BY privilege_type, column_name), '')
		FROM information_schema.column_privileges
		WHERE table_name = 'tenant_identity_binding' AND grantee = 'nava_worker'`)
	if got2 != "SELECT tenant_id" {
		t.Errorf("nava_worker's column privileges on tenant_identity_binding = %s, want SELECT tenant_id", got2)
	}
	if queryBool(t, super, `SELECT EXISTS (SELECT 1 FROM information_schema.role_table_grants
		WHERE table_name = 'tenant_identity_binding' AND grantee = 'nava_worker')`) {
		t.Error("nava_worker holds a table-level grant on tenant_identity_binding")
	}
	// It writes nothing there.
	_, err := execIn(worker, tenantA, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'x', 'y')", tenantA)
	wantSQLState(t, "nava_worker binds a tenant", err, "42501")
}
