//go:build integration

package dbtest

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// T09 (TICK-64): the platform tables of ADR-001 A.3 under FORCE RLS, and the
// 10-entry policy allow-list. Tenants A and B are active and fully seeded;
// X and Y are provisioning.
const (
	tenantA = "0190a000-0000-7000-8000-00000000000a"
	tenantB = "0190a000-0000-7000-8000-00000000000b"
	tenantX = "0190a000-0000-7000-8000-0000000000f1"
	tenantY = "0190a000-0000-7000-8000-0000000000f2"
)

// platformTables are the tables T09 creates.
func platformTables() []string {
	return []string{"idempotency_keys", "outbox", "records", "tenant_identity_binding", "tenants", "workflow_ref"}
}

// allowList is ADR-001 A.3: the only policies that do not test app.tenant_id,
// each with its command and roles.
func allowList() map[string]string {
	return map[string]string{
		"outbox_relay_select":       "SELECT TO nava_relay",
		"outbox_relay_update":       "UPDATE TO nava_relay",
		"binding_lookup":            "SELECT TO nava_app,nava_groupsync",
		"binding_onboard":           "INSERT TO nava_onboard",
		"idempotency_purge":         "DELETE TO nava_purge",
		"idempotency_purge_select":  "SELECT TO nava_purge",
		"outbox_purge":              "DELETE TO nava_purge",
		"outbox_purge_select":       "SELECT TO nava_purge",
		"workflow_ref_purge":        "DELETE TO nava_purge",
		"workflow_ref_purge_select": "SELECT TO nava_purge",
	}
}

// migratedPlatform is a fresh cluster with every migration and grant applied.
func migratedPlatform(t *testing.T) {
	t.Helper()
	m := freshCluster(t)
	migrateAll(t, m)
}

// superNava connects as the bootstrap superuser to the platform database. It
// bypasses RLS, so it seeds fixtures and checks what really landed.
func superNava(t *testing.T) *pgx.Conn {
	t.Helper()
	cfg := superConfig(t).Copy()
	cfg.Database = "nava"
	return connect(t, cfg)
}

func asRole(t *testing.T, role string) *pgx.Conn {
	t.Helper()
	return connect(t, roleConfig(t, role, "nava"))
}

// inTenantTx runs fn in one transaction with app.tenant_id set
// transaction-locally, as the InTenantTx helper does, and commits if fn
// succeeds. tenant "" sets nothing.
func inTenantTx(conn *pgx.Conn, tenant string, fn func(tx pgx.Tx) error) error {
	ctx := context.Background()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if tenant != "" {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenant); err != nil {
			return err
		}
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// execIn runs one statement in a tenant transaction and returns rows affected.
func execIn(conn *pgx.Conn, tenant, sql string, args ...any) (int64, error) {
	var n int64
	err := inTenantTx(conn, tenant, func(tx pgx.Tx) error {
		tag, err := tx.Exec(context.Background(), sql, args...)
		n = tag.RowsAffected()
		return err
	})
	return n, err
}

// seedTenants writes, as the superuser, active tenants with one row in every
// tenant table each, and provisioning tenants with only their tenants row.
func seedTenants(t *testing.T, super *pgx.Conn, active, provisioning []string) {
	t.Helper()
	for _, id := range provisioning {
		exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'provisioning')", id)
	}
	for i, id := range active {
		exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", id)
		exec(t, super, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc/realms/nava', $2)",
			id, fmt.Sprintf("org-%d", i))
		exec(t, super, "INSERT INTO records (tenant_id, id) VALUES ($1, $1)", id)
		exec(t, super, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq)
			VALUES (gen_random_uuid(), $1::uuid, 'workflow.start', 'seed', 'rec:' || $1::text, 1)`, id)
		exec(t, super, "INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ($1, gen_random_uuid(), $1, 'approval')", id)
		exec(t, super, `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at)
			VALUES ($1, $1, 'seed', '\x00', now() + interval '1 day')`, id)
	}
}

func wantSQLState(t *testing.T, what string, err error, code string) {
	t.Helper()
	if sqlState(err) != code {
		t.Errorf("%s: err = %v, want SQLSTATE %s", what, err, code)
	}
}

// Every T09 table is under ENABLE and FORCE ROW LEVEL SECURITY, and so is
// every other table in schema public but goose's version table.
func TestT09EveryTableForcesRLS(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	got := queryString(t, super, `
		SELECT coalesce(string_agg(relname, ',' ORDER BY relname), '') FROM pg_class
		WHERE relnamespace = 'public'::regnamespace AND relkind IN ('r', 'p')
		  AND relrowsecurity AND relforcerowsecurity`)
	if got != strings.Join(platformTables(), ",") {
		t.Fatalf("tables with ENABLE + FORCE RLS = %s, want %s", got, strings.Join(platformTables(), ","))
	}
	if bad := queryString(t, super, `
		SELECT coalesce(string_agg(relname, ','), '') FROM pg_class
		WHERE relnamespace = 'public'::regnamespace AND relkind IN ('r', 'p')
		  AND NOT (relrowsecurity AND relforcerowsecurity) AND relname <> 'goose_db_version'`); bad != "" {
		t.Fatalf("tables without ENABLE + FORCE RLS: %s", bad)
	}
}

// Acceptance: tenant A inserting a dedupe_key that B already uses succeeds;
// the same key twice in one tenant does not.
func TestT09DedupeKeyIsPerTenant(t *testing.T) {
	migratedPlatform(t)
	seedTenants(t, superNava(t), []string{tenantA, tenantB}, nil)
	app := asRole(t, "nava_app")
	insert := `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq)
		VALUES (gen_random_uuid(), $1, 'workflow.signal', 'order-42', 'rec:x', 2) RETURNING id`
	if _, err := execIn(app, tenantB, insert, tenantB); err != nil {
		t.Fatalf("B inserts order-42: %v", err)
	}
	if _, err := execIn(app, tenantA, insert, tenantA); err != nil {
		t.Fatalf("A inserts the dedupe_key B already uses: %v", err)
	}
	_, err := execIn(app, tenantA, insert, tenantA)
	wantSQLState(t, "A inserts order-42 twice", err, "23505")
	// The producer's ON CONFLICT DO NOTHING path.
	if n, err := execIn(app, tenantA, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq)
		VALUES (gen_random_uuid(), $1, 'workflow.signal', 'order-42', 'rec:x', 3)
		ON CONFLICT (tenant_id, dedupe_key) DO NOTHING`, tenantA); err != nil || n != 0 {
		t.Fatalf("ON CONFLICT DO NOTHING: %d rows, %v; want 0, nil", n, err)
	}
}

// Acceptance: as nava_app with tenant A, rows of B are invisible in every
// table it can read (nava_worker too). The one exception is
// tenant_identity_binding, which binding_lookup opens on purpose (A.1 step 4).
// With no tenant, or an empty one, every tenant-scoped read gets no rows.
func TestT09TenantRowsInvisible(t *testing.T) {
	migratedPlatform(t)
	seedTenants(t, superNava(t), []string{tenantA, tenantB}, nil)

	// table -> the tenant column, for each role's readable tables.
	readable := map[string]map[string]string{
		"nava_app":    {"tenants": "id", "records": "tenant_id", "outbox": "tenant_id", "workflow_ref": "tenant_id", "idempotency_keys": "tenant_id"},
		"nava_worker": {"records": "tenant_id", "outbox": "tenant_id", "workflow_ref": "tenant_id"},
	}
	for role, tables := range readable {
		conn := asRole(t, role)
		for table, col := range tables {
			// "A", then an empty value and no value at all: both are no tenant.
			for _, mode := range []string{"A", "empty", "unset"} {
				var own, other int
				err := inTenantTx(conn, "", func(tx pgx.Tx) error {
					if mode != "unset" {
						v := map[string]string{"A": tenantA, "empty": ""}[mode]
						if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", v); err != nil {
							return err
						}
					}
					return tx.QueryRow(context.Background(), fmt.Sprintf(
						"SELECT count(*) FILTER (WHERE %[1]s = $1), count(*) FILTER (WHERE %[1]s <> $1) FROM %[2]s", col, table),
						tenantA).Scan(&own, &other)
				})
				if err != nil {
					t.Fatalf("%s reads %s (tenant %s): %v", role, table, mode, err)
				}
				wantOwn := 0
				if mode == "A" {
					wantOwn = 1
				}
				if own != wantOwn || other != 0 {
					t.Errorf("%s reads %s with tenant %s: own %d, other tenants %d; want %d, 0", role, table, mode, own, other, wantOwn)
				}
			}
		}
	}

	app := asRole(t, "nava_app")
	var n int
	if err := inTenantTx(app, tenantA, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT count(*) FROM tenant_identity_binding WHERE tenant_id = $1", tenantB).Scan(&n)
	}); err != nil || n != 1 {
		t.Fatalf("binding_lookup: %d rows of B, %v; want 1 (the allow-listed lookup)", n, err)
	}

	// B's rows cannot be written from A either: WITH CHECK refuses, and an
	// UPDATE finds nothing.
	for _, w := range []struct{ name, sql string }{
		{"records", "INSERT INTO records (tenant_id, id) VALUES ($1, gen_random_uuid())"},
		{"outbox", "INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq) VALUES (gen_random_uuid(), $1, 'k', 'd-b', 't', 1)"},
		{"workflow_ref", "INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ($1, gen_random_uuid(), $1, 'approval')"},
		{"idempotency_keys", `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at) VALUES ($1, $1, 'k', '\x00', now())`},
	} {
		_, err := execIn(app, tenantA, w.sql, tenantB)
		wantSQLState(t, "A writes a B row into "+w.name, err, "42501")
	}
	if n, err := execIn(app, tenantA, "UPDATE records SET version = version + 1 WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
		t.Errorf("A updates B's record: %d rows, %v; want 0", n, err)
	}
	if n, err := execIn(app, tenantA, "UPDATE idempotency_keys SET state = 'completed', response_status = 200 WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
		t.Errorf("A updates B's idempotency key: %d rows, %v; want 0", n, err)
	}
}

// Foreign keys include tenant_id: a workflow_ref of A cannot point at B's record.
func TestT09ForeignKeyIncludesTenant(t *testing.T) {
	migratedPlatform(t)
	seedTenants(t, superNava(t), []string{tenantA, tenantB}, nil)
	worker := asRole(t, "nava_worker")
	// Records are seeded with id = tenant id, so B's record id is tenantB.
	_, err := execIn(worker, tenantA, "INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ($1, gen_random_uuid(), $2, 'approval')", tenantA, tenantB)
	wantSQLState(t, "A's workflow_ref to B's record", err, "23503")
	if _, err := execIn(worker, tenantA, "INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ($1, gen_random_uuid(), $1, 'approval')", tenantA); err != nil {
		t.Fatalf("A's workflow_ref to its own record: %v", err)
	}
	if n, err := execIn(worker, tenantA, "UPDATE workflow_ref SET closed_at = now() WHERE tenant_id = $1", tenantA); err != nil || n != 2 {
		t.Fatalf("worker closes A's workflow_refs: %d rows, %v; want 2", n, err)
	}
}

// Acceptance: a duplicate (issuer, org_id) binding is rejected, even for
// another tenant.
func TestT09DuplicateBindingRejected(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX})
	onboard := asRole(t, "nava_onboard")
	// tenantA holds (https://kc/realms/nava, org-0).
	_, err := execIn(onboard, tenantX, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc/realms/nava', 'org-0')", tenantX)
	wantSQLState(t, "X binds A's (issuer, org_id)", err, "23505")
	if _, err := execIn(onboard, tenantX, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc/realms/nava', 'org-x')", tenantX); err != nil {
		t.Fatalf("X binds a new org: %v", err)
	}
	_, err = execIn(onboard, tenantX, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES ($1, 'https://kc/realms/nava', 'org-x', '{saml}')", tenantX)
	wantSQLState(t, "X binds its org twice", err, "23505")
}

// Acceptance: nava_onboard inserting a binding for an active tenant is denied,
// its own included.
func TestT09OnboardCannotBindActiveTenant(t *testing.T) {
	migratedPlatform(t)
	seedTenants(t, superNava(t), []string{tenantA}, nil)
	onboard := asRole(t, "nava_onboard")
	_, err := execIn(onboard, tenantA, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc/realms/nava', 'org-new')", tenantA)
	wantSQLState(t, "binding for active tenant A", err, "42501")
	_, err = execIn(onboard, "", "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc/realms/nava', 'org-new')", tenantA)
	wantSQLState(t, "binding with no tenant set", err, "42501")
}

// Acceptance: nava_onboard moves its own tenant provisioning -> active; cannot
// move an active tenant back to provisioning; cannot update another tenant.
func TestT09OnboardActivation(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantY})
	onboard := asRole(t, "nava_onboard")

	// Onboarding creates its tenant inside InTenantTx(newTenantID).
	if _, err := execIn(onboard, tenantX, "INSERT INTO tenants (id, state, onboarding_request_hash) VALUES ($1, 'provisioning', '\\x01')", tenantX); err != nil {
		t.Fatalf("create X: %v", err)
	}
	_, err := execIn(onboard, tenantX, "INSERT INTO tenants (id, state) VALUES ($1, 'provisioning')", "0190a000-0000-7000-8000-0000000000f3")
	wantSQLState(t, "create a tenant other than app.tenant_id", err, "42501")
	_, err = execIn(onboard, "0190a000-0000-7000-8000-0000000000f4", "INSERT INTO tenants (id, state) VALUES ($1, 'active')", "0190a000-0000-7000-8000-0000000000f4")
	wantSQLState(t, "create a tenant already active", err, "42501")

	state := func(id string) string {
		return queryString(t, super, "SELECT state FROM tenants WHERE id = $1", id)
	}
	if n, err := execIn(onboard, tenantX, "UPDATE tenants SET state = 'active' WHERE id = $1", tenantX); err != nil || n != 1 || state(tenantX) != "active" {
		t.Fatalf("activate X: %d rows, %v, state %s; want 1, nil, active", n, err, state(tenantX))
	}
	if n, err := execIn(onboard, tenantX, "UPDATE tenants SET state = 'provisioning' WHERE id = $1", tenantX); err != nil || n != 0 || state(tenantX) != "active" {
		t.Errorf("move X back: %d rows, %v, state %s; want 0, nil, active", n, err, state(tenantX))
	}
	if n, err := execIn(onboard, tenantX, "UPDATE tenants SET state = 'active' WHERE id = $1", tenantY); err != nil || n != 0 || state(tenantY) != "provisioning" {
		t.Errorf("activate Y from X: %d rows, %v, state %s; want 0, nil, provisioning", n, err, state(tenantY))
	}
	// Only state is updatable.
	_, err = execIn(onboard, tenantY, "UPDATE tenants SET onboarding_request_hash = '\\x02' WHERE id = $1", tenantY)
	wantSQLState(t, "update onboarding_request_hash", err, "42501")
}

// Acceptance: with X and Y both provisioning, nava_onboard in InTenantTx(X)
// cannot bind Y: binding_onboard's subquery sees tenants only through
// tenants_onboard_select, which shows X alone.
func TestT09OnboardCannotBindOtherProvisioningTenant(t *testing.T) {
	migratedPlatform(t)
	seedTenants(t, superNava(t), nil, []string{tenantX, tenantY})
	onboard := asRole(t, "nava_onboard")
	_, err := execIn(onboard, tenantX, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc/realms/nava', 'org-y')", tenantY)
	wantSQLState(t, "bind Y from InTenantTx(X)", err, "42501")
	if _, err := execIn(onboard, tenantY, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc/realms/nava', 'org-y')", tenantY); err != nil {
		t.Fatalf("bind Y from InTenantTx(Y): %v", err)
	}
}

// Acceptance: exactly 10 policies do not reference app.tenant_id, and they
// are the A.3 allow-list, each FOR its one command and TO its named roles.
// No policy anywhere is TO PUBLIC.
func TestT09PolicyAllowList(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	rows, err := super.Query(context.Background(), `
		SELECT tablename, policyname, cmd, roles::text[],
		       coalesce(qual, '') || ' ' || coalesce(with_check, '') LIKE '%app.tenant_id%'
		FROM pg_policies WHERE schemaname = 'public' ORDER BY policyname`)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	total := 0
	for rows.Next() {
		var table, name, cmd string
		var roles []string
		var tested bool
		if err := rows.Scan(&table, &name, &cmd, &roles, &tested); err != nil {
			t.Fatal(err)
		}
		total++
		if slices.Contains(roles, "public") {
			t.Errorf("policy %s on %s is TO PUBLIC", name, table)
		}
		if !tested {
			slices.Sort(roles)
			got[name] = cmd + " TO " + strings.Join(roles, ",")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := allowList()
	if len(got) != 10 || len(want) != 10 {
		t.Errorf("%d policies do not test app.tenant_id (of %d), want 10: %v", len(got), total, got)
	}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("allow-listed %s = %q, want %q", name, got[name], w)
		}
	}
	for name, g := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("policy %s (%s) does not test app.tenant_id and is not on the allow-list", name, g)
		}
	}
}

// The allow-listed relay and purge policies reach only what A.3 gives them.
func TestT09RelayAndPurgeScope(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	ctx := context.Background()

	relay := asRole(t, "nava_relay")
	var n int
	if err := relay.QueryRow(ctx, "SELECT count(DISTINCT tenant_id) FROM outbox WHERE dispatched_at IS NULL").Scan(&n); err != nil || n != 2 {
		t.Fatalf("relay polls pending outbox: %d tenants, %v; want 2", n, err)
	}
	for _, q := range []string{"SELECT dedupe_key FROM outbox", "SELECT last_error FROM outbox", "SELECT id FROM records", "SELECT outbox_id FROM workflow_ref"} {
		_, err := relay.Exec(ctx, q)
		wantSQLState(t, "relay: "+q, err, "42501")
	}
	if tag, err := relay.Exec(ctx, "UPDATE outbox SET dispatched_at = now(), attempts = attempts + 1 WHERE tenant_id = $1", tenantA); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("relay dispatches A's row: %v, %v", tag, err)
	}
	if tag, err := relay.Exec(ctx, "UPDATE outbox SET dead_at = now() WHERE tenant_id = $1", tenantA); err != nil || tag.RowsAffected() != 0 {
		t.Errorf("relay updates a dispatched row: %v, %v; want 0 rows", tag, err)
	}
	_, err := relay.Exec(ctx, "UPDATE outbox SET tenant_id = $1", tenantB)
	wantSQLState(t, "relay rewrites tenant_id", err, "42501")

	// Age A's rows past every expiry; B's stay fresh.
	exec(t, super, "UPDATE outbox SET dispatched_at = now() - interval '8 days' WHERE tenant_id = $1", tenantA)
	exec(t, super, "UPDATE workflow_ref SET closed_at = now() - interval '31 days' WHERE tenant_id = $1", tenantA)
	exec(t, super, "UPDATE workflow_ref SET closed_at = now() WHERE tenant_id = $1", tenantB)
	exec(t, super, "UPDATE idempotency_keys SET expires_at = now() - interval '1 second' WHERE tenant_id = $1", tenantA)

	purge := asRole(t, "nava_purge")
	for _, table := range []string{"outbox", "workflow_ref", "idempotency_keys"} {
		if err := purge.QueryRow(ctx, "SELECT count(*) FILTER (WHERE tenant_id = $1) FROM "+table, tenantB).Scan(&n); err != nil || n != 0 {
			t.Errorf("purge sees %d unexpired %s rows, %v; want 0", n, table, err)
		}
		tag, err := purge.Exec(ctx, "DELETE FROM "+table)
		if err != nil || tag.RowsAffected() != 1 {
			t.Errorf("purge deletes %s: %v, %v; want 1 row (A's)", table, tag, err)
		}
		if left := queryString(t, super, "SELECT string_agg(tenant_id::text, ',') FROM "+table); left != tenantB {
			t.Errorf("%s after purge holds %s, want only B", table, left)
		}
	}
	_, err = purge.Exec(ctx, "SELECT payload FROM outbox")
	wantSQLState(t, "purge reads payload", err, "42501")
}

// Writes under FORCE RLS fail closed without a tenant: no tenant, no insert.
func TestT09NoTenantNoWrite(t *testing.T) {
	migratedPlatform(t)
	seedTenants(t, superNava(t), []string{tenantA}, nil)
	app := asRole(t, "nava_app")
	_, err := execIn(app, "", "INSERT INTO records (tenant_id, id) VALUES ($1, gen_random_uuid())", tenantA)
	wantSQLState(t, "insert with no tenant", err, "42501")
}
