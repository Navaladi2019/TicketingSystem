// Package isolation is the tenant isolation suite (PLAN-001 T11, ADR-001
// Annex B CI-2). `make isolation` builds a fresh platform database (db-roles,
// every migration), seeds tenants A and B, and then:
//
//   - runs the generic CI-2 checks over every tenant table, discovered
//     fail-closed by catalog.TenantTables, as every runtime role and as the
//     owner: with no tenant set, with app.tenant_id empty, and after a prior
//     SET LOCAL of tenant A, both directly and through PgBouncer in
//     transaction mode, each SELECT, UPDATE and DELETE the role is granted
//     must reach 0 rows or fail; with tenant A set it must reach none of
//     tenant B's rows. A new table needs no registration, only seed rows;
//   - runs every registered Path below as its role, scoped to tenant A with
//     tenancy.InTenantTx, aimed at tenant B's data; a DB path drives the
//     database itself, for the CI-9 role reach checks that need seeded rows
//     or a positive control;
//   - writes a report, one line per role × table, grouped by ADR test id, so
//     partial coverage (CI-8, CI-10) stays visible.
//
// Later epics register their named paths in Paths. Editing this file needs
// CTO sign-off (docs/delivery/review-paths.yaml); so does removing a test of
// this package or adding a skip to one.
package isolation

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The seeded tenants (testdata/seed.sql). Every registered path runs as
// tenant A and aims at tenant B.
const (
	TenantA = "0a000000-0000-4000-8000-00000000000a"
	TenantB = "0b000000-0000-4000-8000-00000000000b"

	// RecordA1 is one of tenant A's records.
	RecordA1 = "0a000000-0000-4000-8000-0000000000a1"
	// RecordB1 is one of tenant B's records.
	RecordB1 = "0b000000-0000-4000-8000-0000000000b1"
	// SharedDedupeKey and SharedIdempotencyKey exist in both tenants, so a
	// lookup by them alone would return both rows without isolation.
	SharedDedupeKey      = "shared-dedupe-key"
	SharedIdempotencyKey = "shared-idempotency-key"
	// NewID is an id no seeded row uses, for inserts.
	NewID = "0c000000-0000-4000-8000-0000000000c1"
)

// Expect is what a registered SQL call must do.
type Expect int

const (
	// OwnRowsOnly means the query succeeds and the first column of every row it
	// returns is tenant A's id. Zero rows pass.
	OwnRowsOnly Expect = iota + 1
	// Denied means the statement reaches 0 rows or is refused: SQLSTATE
	// 42501 (a grant or a row-level security policy) or class 23 (an
	// integrity constraint, such as a foreign key that holds the tenant). Any
	// other error fails the path, so a typo cannot pass as a denial; aim an
	// UPDATE or DELETE where an integrity error cannot hide a reached row.
	Denied
)

// SQLCall is a statement run as the path's role inside
// tenancy.InTenantTx(TenantA), through PgBouncer for roles on its login
// allow-list, and rolled back.
type SQLCall struct {
	Query  string
	Args   []any
	Expect Expect
}

// HTTPCall is a request made with a valid token for tenant A that tries to
// name tenant B some other way. The response must be a 4xx or carry only
// tenant A's data.
type HTTPCall struct {
	Method string
	Path   string // with the query string
	Header http.Header
	Body   string
	// NoRoute names the task that adds this route. Until then the call must
	// get 404 or 405; without it, the call must get 200 with tenant A's data,
	// so a validator regression that starts refusing it cannot hide.
	NoRoute string
}

// Cluster is the suite's database, as a DB path sees it.
type Cluster interface {
	// Pool is a one-connection pool logged in as role, through PgBouncer for
	// roles on its login allow-list.
	Pool(role string) (*pgxpool.Pool, error)
	// Super is the bootstrap superuser on the platform database, which RLS
	// does not hold. A DB path that seeds rows with it deletes them before
	// it returns, so later checks see the seed unchanged.
	Super() (*pgx.Conn, error)
}

// Path is one registered route to tenant data. Exactly one of SQL, HTTP,
// Check and DB is set.
type Path struct {
	Name string
	// Case is the ADR-001 Annex B test id the path proves ("CI-2", "CI-8",
	// ...); it must be a key of Coverage.
	Case string
	// Role is the database role a SQL call runs as.
	Role string
	SQL  *SQLCall
	HTTP *HTTPCall
	// Check is an in-process check; it returns an error when isolation does
	// not hold.
	Check func(context.Context) error
	// DB is a check against the suite's database; it returns an error when
	// the role reaches more than ADR-001 A.3 gives it, or when its positive
	// control fails (so a broken check cannot pass as a denial).
	DB func(context.Context, Cluster) error
	// Pending names the task that makes the path runnable. A pending path is
	// listed in the report and not run.
	Pending string
}

// Coverage is how much of each ADR test id this suite proves (PLAN-001
// "Annex B coverage"). The report prints it above each group.
func Coverage() map[string]string {
	return map[string]string{
		"CI-2":  "full: generic checks on every tenant table; forged-tenant HTTP paths (T17)",
		"CI-9":  "full: nava_relay, nava_purge, nava_onboard and nava_backup reach (T12); nava_backup through T08's pg_hba cases",
		"CI-8":  "partial → TICK-87 (byte-identical 404, stale If-Match 412, masked-field filter 400); cursor cases from T23",
		"CI-10": "partial → TICK-88; no path registered yet (T29 depth limit is a unit test)",
	}
}

// Exemption is a role, table and command whose cross-tenant reach is by
// design: an ADR-001 A.3 allow-listed policy that does not test
// app.tenant_id. The generic checks report its rows as ALLOWLISTED instead of
// failing. Each must match catalog.AllowList (TestExemptionsAreAllowListed).
//
// The key is role × table × command, not the policy's predicate: any row that
// role reaches through that command on that table is ALLOWLISTED. That is safe
// only because migrate.Run runs catalog.Check, which pins each allow-listed
// policy's exact USING and WITH CHECK expressions (catalog.PolicyProblems),
// before the harness seeds; a migration that widens a predicate fails the
// migrator, so the suite never runs against it.
type Exemption struct {
	Role, Table, Command, Policy string
}

// Exemptions are the allow-listed policies that reach seeded rows. The purge
// policies are not here: they reach only expired rows and the seed has none,
// so a purge policy that stops testing expiry fails the suite.
func Exemptions() []Exemption {
	return []Exemption{
		{"nava_relay", "public.outbox", "SELECT", "outbox_relay_select"},
		{"nava_relay", "public.outbox", "UPDATE", "outbox_relay_update"},
		{"nava_app", "public.tenant_identity_binding", "SELECT", "binding_lookup"},
		{"nava_groupsync", "public.tenant_identity_binding", "SELECT", "binding_lookup"},
	}
}

// Paths is the registry of named paths.
func Paths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: name, Case: "CI-2", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	forged := func(name string, c HTTPCall) Path {
		return Path{Name: name, Case: "CI-2", HTTP: &c}
	}
	paths := []Path{
		// IDOR by id: the id alone must not find tenant B's row.
		sql("records: read tenant B's record by id", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM records WHERE id = $1", RecordB1),
		sql("records: read tenant B's record by its full key", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM records WHERE tenant_id = $1 AND id = $2", TenantB, RecordB1),
		sql("records: insert a row for tenant B", "nava_app", Denied,
			"INSERT INTO records (tenant_id, id) VALUES ($1, $2)", TenantB, NewID),
		sql("records: bump tenant B's versions", "nava_worker", Denied,
			"UPDATE records SET version = version + 1 WHERE tenant_id = $1", TenantB),
		sql("outbox: look a dedupe key up in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM outbox WHERE dedupe_key = $1", SharedDedupeKey),
		sql("outbox: insert an event for tenant B", "nava_worker", Denied,
			"INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq) VALUES ($1, $2, 'k', 'iso-new', 't', 1)",
			NewID, TenantB),
		// The composite foreign key holds the tenant: tenant A cannot point
		// at tenant B's record.
		sql("workflow_ref: reference tenant B's record from tenant A", "nava_worker", Denied,
			"INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ($1, $2, $3, 'k')",
			TenantA, NewID, RecordB1),
		sql("workflow_ref: close tenant B's workflows", "nava_worker", Denied,
			"UPDATE workflow_ref SET closed_at = now() WHERE tenant_id = $1", TenantB),
		sql("idempotency_keys: look a key up in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM idempotency_keys WHERE key = $1", SharedIdempotencyKey),
		sql("idempotency_keys: insert a key for tenant B", "nava_app", Denied,
			"INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at) VALUES ($1, $2, 'iso-new', decode('00', 'hex'), now() + interval '1 day')",
			TenantB, NewID),
		sql("idempotency_keys: overwrite tenant B's stored response", "nava_app", Denied,
			"UPDATE idempotency_keys SET response_body = 'x' WHERE tenant_id = $1", TenantB),
		sql("tenants: read tenant B", "nava_app", OwnRowsOnly,
			"SELECT id FROM tenants WHERE id = $1", TenantB),
		sql("tenants: onboarding creates a tenant other than its own", "nava_onboard", Denied,
			"INSERT INTO tenants (id, state) VALUES ($1, 'provisioning')", NewID),
		sql("tenants: onboarding changes tenant B's state", "nava_onboard", Denied,
			"UPDATE tenants SET state = 'active' WHERE id = $1", TenantB),
		sql("tenant_identity_binding: onboarding binds tenant B", "nava_onboard", Denied,
			"INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://idp.example/realms/iso', 'org-iso')", TenantB),
		sql("tenant_identity_binding: onboarding rebinds its own active tenant", "nava_onboard", Denied,
			"INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://idp.example/realms/iso', 'org-iso')", TenantA),

		// Forged tenant: a valid token for tenant A names tenant B in the
		// payload, a header, the query or baggage. The tenant comes only from
		// the token's binding (ADR-001 I2). The payload path has no route yet
		// (PLAN-002 records), so it proves only that a 4xx keeps B out.
		forged("forged tenant in the query string", HTTPCall{Method: http.MethodGet, Path: "/api/v1/me?tenant_id=" + TenantB}),
		forged("forged tenant in a header", HTTPCall{Method: http.MethodGet, Path: "/api/v1/me",
			Header: http.Header{"X-Tenant-Id": {TenantB}, "X-Nava-Tenant": {TenantB}}}),
		forged("forged tenant in the payload", HTTPCall{Method: http.MethodPost, Path: "/api/v1/records",
			Body: `{"tenant_id":"` + TenantB + `"}`, NoRoute: "PLAN-002 records"}),
		forged("forged tenant in the body of GET /api/v1/me", HTTPCall{Method: http.MethodGet, Path: "/api/v1/me",
			Body: `{"tenant_id":"` + TenantB + `"}`}),
		// TICK-93: baggage is never read; the edge propagator drops it.
		forged("forged tenant in baggage", HTTPCall{Method: http.MethodGet, Path: "/api/v1/me",
			Header: http.Header{"Baggage": {"tenant_id=" + TenantB + ",nava.tenant_id=" + TenantB}}}),
	}
	paths = append(paths, cursorPaths()...)
	return append(paths, rolePaths()...)
}
