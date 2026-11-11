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

	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
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
	// RecordA2 is tenant A's pipeline record (P3-09); the seed also repeats it
	// as the subject of one of tenant B's audit_log entries.
	RecordA2 = "0a000000-0000-4000-8000-0000000000a2"
	// RecordB2 is tenant B's pipeline record (P3-09), with audit rows.
	RecordB2 = "0b000000-0000-4000-8000-0000000000b2"
	// PrincipalB1 is a tenant B principal who is a member of a group.
	PrincipalB1 = "0b000000-0000-4000-8000-0000000003b1"
	// SharedDedupeKey and SharedIdempotencyKey exist in both tenants, so a
	// lookup by them alone would return both rows without isolation.
	SharedDedupeKey      = "shared-dedupe-key"
	SharedIdempotencyKey = "shared-idempotency-key"
	// SharedInviteEmail is invited by both tenants (pending_invite).
	SharedInviteEmail = "shared-invite@example.test"
	// EntityDefinitionB1 is tenant B's ticket entity type, which has two
	// versions; tenant A's ticket has one.
	EntityDefinitionB1 = "0b000000-0000-4000-8000-0000000006b1"
	// GroupB1 is a tenant B group that is not its admins group.
	GroupB1 = "0b000000-0000-4000-8000-0000000004b1"
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
	// StoredUnderA is for an HTTPCall that writes (P3-18, TICK-366; CTO
	// D-P10 conditions on TICK-366): the create refuses it with the 404 that
	// the same request to Unknown gets, byte for byte but for instance, and
	// writes nothing; or it answers 201 with the id of a new record
	// whose tenant_id is tenant A. Either way tenant B's record and audit row
	// counts are unchanged, and the body never holds tenant B's id. Everything
	// is read on the superuser connection, past row-level security: a read as
	// tenant A cannot see a row misfiled under tenant B, so it would pass
	// vacuously (TICK-363). The call carries its own Idempotency-Key, and a
	// replayed response fails: it never reached the handler.
	StoredUnderA
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
// name tenant B some other way. The response must carry only tenant A's data,
// or, for a write (StoredUnderA), refuse or store under tenant A only.
type HTTPCall struct {
	Method string
	Path   string // with the query string
	Header http.Header
	Body   string
	// Expect is zero for a read, which must get 200 with tenant A's data, so a
	// validator regression that starts refusing it cannot hide; or
	// StoredUnderA for a write.
	Expect Expect
	// Unknown is, for StoredUnderA, the same route naming a key no tenant
	// has. The only refusal that passes is its 404: any other refusal (a 403,
	// or a 422 from a handler that found tenant B's type) tells the caller
	// the key exists somewhere.
	Unknown string
	// NoRoute names the task that adds this route. Until then the call must
	// get 404 or 405, with no tenant B id in the body.
	NoRoute string
	// Token, when set, replaces the tenant A token with this shape (CI-5).
	// The call then passes only on 401: the shape must not resolve at all.
	Token *TokenShape
}

// Cluster is the suite's database, as a DB path sees it.
type Cluster interface {
	// Pool is a one-connection pool logged in as role, through PgBouncer for
	// roles on its login allow-list.
	Pool(role string) (*pgxpool.Pool, error)
	// PoolSize is Pool with room for conns connections, logged in directly:
	// the suite's PgBouncer gives a role one server connection, so a check
	// that holds one while another request needs its own cannot use it.
	PoolSize(role string, conns int32) (*pgxpool.Pool, error)
	// PooledSize is Pool with room for conns client connections, all through
	// PgBouncer's one server connection for the role: a check that holds a
	// server connection while another request needs one waits forever.
	PooledSize(role string, conns int32) (*pgxpool.Pool, error)
	// AllocatorPool is the record-number allocator pool, as nava-core opens
	// it (ADR-005 A3, A5, TICK-657): nava_app, checked by
	// tenancy.AllocPoolConfig, logged in directly so it never shares the
	// suite's one PgBouncer server connection with the creates that wait on
	// it.
	AllocatorPool() (*pgxpool.Pool, error)
	// AuditReader is the audit reader's own pool (read.OpenPool), logged in
	// as nava_audit_reader through PgBouncer with a DSN, as nava-core opens
	// it (ADR-004-A1 A1-5, A1-6).
	AuditReader() (*read.Pool, error)
	// Super is the bootstrap superuser on the platform database, which RLS
	// does not hold. A DB path that seeds rows with it deletes them before
	// it returns, so later checks see the seed unchanged.
	Super() (*pgx.Conn, error)
}

// Server is the real nava-core API handler the HTTP paths run against, as an
// API path sees it. Its resolver is the Keycloak adapter trusting the
// suite's test realm, whose organization OrgA is bound to tenant A, reading
// bindings and principals as nava_app through PgBouncer; OperatorSubject is
// on its operator allow-list.
type Server interface {
	// Mint signs an access token from the test realm, valid for five minutes.
	Mint(tok Token) string
	// Do sends an /api/v1 request with token as its bearer and header as
	// extra request headers (nil for none), and returns the status and body.
	Do(ctx context.Context, method, path, token, body string, header http.Header) (int, string)
	// ResolveOperator is the adapter's operator verdict on token, the check
	// the operator routes authenticate with: nil for an allow-listed
	// operator.
	ResolveOperator(ctx context.Context, token string) error
}

// OperatorSubject is the operator the test realm's adapter allow-lists.
const OperatorSubject = "op-isolation"

// Token describes a test-realm access token.
type Token struct {
	// Subject is sub, principals.keycloak_user_id.
	Subject string
	// NoOrg leaves the organization claim out; otherwise the token names the
	// organization bound to tenant A.
	NoOrg bool
	// Audiences is aud; empty means nava-api.
	Audiences []string
	// RealmRoles are realm_access.roles.
	RealmRoles []string
}

// DefaultStackTarget runs a Stack path whose StackTarget is empty.
const DefaultStackTarget = "make stack-test-identity"

// Path is one registered route to tenant data. Exactly one of SQL, HTTP,
// Check, DB, API and Stack is set.
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
	// API is a check that drives the database and the real API handler
	// together; it returns an error when isolation does not hold or its
	// positive control fails.
	API func(context.Context, Cluster, Server) error
	// Stack names the test in internal/identitystack that proves the path
	// against the real Keycloak (make stack-test-identity, the CI stack
	// job). The suite lists it and TestStackPathsExist fails when the test
	// is gone, so the registration cannot go stale.
	Stack string
	// StackTarget is the make target that runs Stack, when it is not
	// make stack-test-identity (DefaultStackTarget); the report names it.
	StackTarget string
	// Pending names the task that makes the path runnable. A pending path is
	// listed in the report and not run.
	Pending string
}

// Coverage is how much of each ADR test id this suite proves (PLAN-001
// "Annex B coverage"). The report prints it above each group.
func Coverage() map[string]string {
	return map[string]string{
		"CI-2":           "full: generic checks on every tenant table; forged-tenant HTTP paths (T17); the forged write on POST /api/v1/records/incident, StoredUnderA (P3-18)",
		"CI-9":           "full: nava_relay, nava_purge, nava_onboard and nava_backup reach (T12); nava_backup through T08's pg_hba cases",
		"CI-8":           "full: the record routes' byte-identical 404 on GET, PATCH and history, stale If-Match 412, and (e)/(f) (TICK-87); cursor cases from T23; idempotency cases from T22; the masked-field filter 400 (d) and the S10 attack rows on the record ACL, masks, admin routes and explainer (P3-37)",
		"CI-10":          "partial → P3-42 (S11b): (g) signal ping-pong needs the interpreter; (f) the start key is per tenant, here; (a)–(e) are TestCI10_* in internal/records/pipeline and the outbox depth limit TestCI10_* in internal/outbox, both against real PostgreSQL in make test-integration",
		"T26":            "full: deprovisioning within D-P9's 10 minutes and the Epic 03 identity boundary (TICK-82); cases 1, 2, 6 and 8 run here, cases 3, 4, 5 and 7 need the real Keycloak and run in make stack-test-identity",
		"CI-A4":          "full (ADR-004, P5-19, P5-42): audit_log through the parent, its DEFAULT partition, audit_chain_head and record_write_keys as every runtime role, with no tenant and with onboarding's pinned policies; GET /api/v1/audit/entries and record history for tenant B's admin, auditor and agent. export, stubs, reseed and erasure (P5-42, TICK-447): TestCIA4* in internal/server (make test-integration), tenant B against tenant A through the real handler, sealer, verifier and nava_erasure",
		"CI-A5":          "full (ADR-004, P5-19, P5-42): table bytes of audit_log and entity_definition_versions after record and S8 canaries, the controls uncompressed and under 1,900 bytes stored; export bytes in TestCIA5ExportBytesCarryNoCanary in internal/server (P5-42, make test-integration)",
		caseDiffDenied:   "full (ADR-004-A1, TICK-558): every runtime role but nava_audit_reader gets 42501 for every listed way of reading audit_log.diff and diff_salt, through PgBouncer and nava_erasure directly; the reader cannot write audit_log or read records",
		caseReaderScoped: "full (ADR-004-A1, TICK-558): nava_audit_reader through read.Reader and as raw SQL reads only the set tenant's rows, none with no tenant, and never txid",
		caseReaderMasks:  "full (ADR-004-A1, TICK-558): GET /api/v1/audit/entries on the reader pool masks an x-nava-secret property for an auditor with no mask permission; red without nava_audit_reader in entity_definition_versions_select",
		"CI-A10":         "full (ADR-004, P5-19): the nava-core and nava-worker pools from compose.yaml log in as nava_app and nava_worker and rerun the P5-10b matrix; nava_onboard and nava_groupsync are TestP510b* in internal/db/dbtest",
		"I5":             "partial → P3-34 (S10): the record ACL's SQL list predicate (authz.CompileRead) and Go check (authz.Allows) agree for random policies, principals and records under RLS, with pair-matched shares and mutant compilers that must be caught (P3-32, TestI5Agreement); data filters and the list route wait for P3-34",
		"CI-5":           "full in make identity-attacks (real Keycloak, the CI stack job, ci5-report artifact); here every case is registered: token shapes through the API handler (401 for the stated reason), the duplicate binding on this database, Keycloak-only cases as Stack paths on TestCI5",
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

// forgedWrite is a StoredUnderA create of tenant B's only entity key, with
// its own Idempotency-Key (keys under iso-ci2- are removed after the call).
func forgedWrite(key, body string, header http.Header) HTTPCall {
	h := http.Header{"Idempotency-Key": {key}}
	for k, vs := range header {
		h[k] = vs
	}
	return HTTPCall{Method: http.MethodPost, Path: "/api/v1/records/incident", Header: h, Body: body, Expect: StoredUnderA,
		Unknown: "/api/v1/records/iso_et_none"}
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
		// T22a (TICK-212): the takeover lease nava_app may move forward.
		sql("idempotency_keys: extend tenant B's lease", "nava_app", Denied,
			"UPDATE idempotency_keys SET lease_until = now() + interval '1 hour' WHERE tenant_id = $1", TenantB),
		sql("idempotency_keys: reset tenant B's lease so a takeover can run", "nava_app", Denied,
			"UPDATE idempotency_keys SET lease_until = now() - interval '1 hour' WHERE tenant_id = $1", TenantB),
		// ADR-002 AC5: the /me statement with the wrong tenant predicate. Run
		// as tenant A, row-level security hides tenant B's membership and
		// group rows, so the join finds nothing.
		sql("groups: list tenant B principal's group paths with tenant B's predicate", "nava_app", Denied,
			"SELECT g.id, g.path FROM group_members gm JOIN groups g ON g.tenant_id = gm.tenant_id AND g.id = gm.group_id WHERE gm.tenant_id = $1 AND gm.principal_id = $2 ORDER BY g.id",
			TenantB, PrincipalB1),
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
		// pending_invite (T252a, TICK-254): both tenants invited the same
		// email, so a lookup by email alone would find both rows.
		sql("pending_invite: onboarding looks an invited email up in every tenant", "nava_onboard", OwnRowsOnly,
			"SELECT tenant_id FROM pending_invite WHERE email = $1", SharedInviteEmail),
		sql("pending_invite: onboarding invites for tenant B", "nava_onboard", Denied,
			"INSERT INTO pending_invite (tenant_id, email, expires_at) VALUES ($1, 'iso-new@example.test', now() + interval '1 day')", TenantB),
		sql("pending_invite: onboarding re-invites tenant B's invitee", "nava_onboard", Denied,
			`INSERT INTO pending_invite (tenant_id, email, expires_at) VALUES ($1, $2, now() + interval '30 days')
			 ON CONFLICT (tenant_id, email) DO UPDATE SET expires_at = EXCLUDED.expires_at WHERE pending_invite.consumed_at IS NULL`,
			TenantB, SharedInviteEmail),
		sql("pending_invite: onboarding extends tenant B's invites", "nava_onboard", Denied,
			"UPDATE pending_invite SET expires_at = now() + interval '30 days' WHERE tenant_id = $1", TenantB),
		sql("pending_invite: group sync reads tenant B's invite by its full key", "nava_groupsync", OwnRowsOnly,
			"SELECT tenant_id FROM pending_invite WHERE tenant_id = $1 AND email = $2", TenantB, SharedInviteEmail),
		sql("pending_invite: group sync consumes tenant B's invite", "nava_groupsync", Denied,
			`UPDATE pending_invite SET consumed_at = now()
			 WHERE tenant_id = $1 AND email = $2 AND consumed_at IS NULL AND expires_at > now() RETURNING tenant_id`,
			TenantB, SharedInviteEmail),

		// Entity types (P3-07, TICK-345). The seed gives both tenants the key
		// ticket and prefix TCK, and only tenant B the key incident, so a
		// lookup by key or prefix alone would find tenant B's rows.
		sql("entity_definitions: read tenant B's only key", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM entity_definitions WHERE key = 'incident'"),
		sql("entity_definitions: look a shared key and prefix up in every tenant", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM entity_definitions WHERE key = 'ticket' OR number_prefix = 'TCK'"),
		sql("entity_definitions: read tenant B's row by its full key", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM entity_definitions WHERE tenant_id = $1 AND id = $2", TenantB, EntityDefinitionB1),
		sql("entity_definitions: insert a type for tenant B", "nava_app", Denied,
			"INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version) VALUES ($1, $2, 'iso_new', 'Iso', 'ISO', 1)",
			TenantB, NewID),
		sql("entity_definitions: onboarding seeds a type for tenant B", "nava_onboard", Denied,
			"INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version) VALUES ($1, $2, 'iso_new', 'Iso', 'ISO', 1)",
			TenantB, NewID),
		// TICK-401: the rest of onboarding's ownProvisioning policies, each
		// aimed at tenant B. Every statement is valid for tenant B (the path
		// matches groups_path_check; the Keycloak id and the version are new;
		// the version names tenant B's principal, see the nava_app publish
		// row), so only the policy can deny it: a policy that stops testing
		// the tenant lets the row through and the path fails.
		sql("groups: onboarding inserts a group for tenant B", "nava_onboard", Denied,
			"INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, 'kc-iso-new', $3)",
			TenantB, NewID, "/t/"+TenantB+"/iso"),
		sql("groups: onboarding reads tenant B's admins group by its Keycloak id", "nava_onboard", Denied,
			"SELECT tenant_id FROM groups WHERE tenant_id = $1 AND keycloak_group_id = 'kc-group-b2'", TenantB),
		sql("tenant_admin_group: onboarding reads tenant B's admins row", "nava_onboard", Denied,
			"SELECT tenant_id FROM tenant_admin_group WHERE tenant_id = $1", TenantB),
		sql("entity_definition_versions: onboarding seeds a version for tenant B", "nava_onboard", Denied,
			`INSERT INTO entity_definition_versions (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
			 VALUES ($1, $2, $3, 3, '{}'::json, sha256(convert_to('{}', 'UTF8')), '{}', '{}', 'compatible', $4)`,
			TenantB, NewID, EntityDefinitionB1, PrincipalB1),
		sql("entity_definitions: move tenant B's current versions", "nava_app", Denied,
			"UPDATE entity_definitions SET current_version = 1, updated_at = now() WHERE tenant_id = $1", TenantB),
		sql("entity_definition_versions: read tenant B's versions by definition id", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM entity_definition_versions WHERE definition_id = $1", EntityDefinitionB1),
		sql("entity_definition_versions: read tenant B's versions by their full key", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM entity_definition_versions WHERE tenant_id = $1 AND definition_id = $2", TenantB, EntityDefinitionB1),
		// published_by names tenant B's principal: a version above 1 must
		// name one (entity_definition_versions_published_by_check), and
		// without it that CHECK, not the policy, would refuse the row.
		sql("entity_definition_versions: publish a version for tenant B", "nava_app", Denied,
			`INSERT INTO entity_definition_versions (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
			 VALUES ($1, $2, $3, 3, '{}'::json, sha256(convert_to('{}', 'UTF8')), '{}', '{}', 'compatible', $4)`,
			TenantB, NewID, EntityDefinitionB1, PrincipalB1),
		// The composite foreign key holds the tenant: tenant A cannot hang a
		// version off tenant B's definition. published_by names tenant A's
		// principal, so the CHECK passes and the foreign key (23503), not
		// entity_definition_versions_published_by_check (23514), refuses it.
		sql("entity_definition_versions: hang tenant A's version off tenant B's definition", "nava_app", Denied,
			`INSERT INTO entity_definition_versions (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
			 VALUES ($1, $2, $3, 3, '{}'::json, sha256(convert_to('{}', 'UTF8')), '{}', '{}', 'compatible', $4)`,
			TenantA, NewID, EntityDefinitionB1, principalA1),
		sql("entity_definition_versions: rewrite tenant B's versions", "nava_app", Denied,
			"UPDATE entity_definition_versions SET compat = 'breaking' WHERE tenant_id = $1", TenantB),
		sql("tenant_admin_group: read tenant B's admins group", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM tenant_admin_group WHERE tenant_id = $1", TenantB),
		sql("tenant_admin_group: name an admins group for tenant B", "nava_onboard", Denied,
			"INSERT INTO tenant_admin_group (tenant_id, group_id) VALUES ($1, $2)", TenantB, GroupB1),
		sql("tenant_admin_group: point tenant B's admins at another group", "nava_app", Denied,
			"UPDATE tenant_admin_group SET group_id = $2 WHERE tenant_id = $1", TenantB, GroupB1),
		sql("tenant_auditor_group: read tenant B's auditors group", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM tenant_auditor_group WHERE tenant_id = $1", TenantB),
		sql("tenant_auditor_group: onboarding reads tenant B's auditors row", "nava_onboard", Denied,
			"SELECT tenant_id FROM tenant_auditor_group WHERE tenant_id = $1", TenantB),
		sql("tenant_auditor_group: name an auditors group for tenant B", "nava_onboard", Denied,
			"INSERT INTO tenant_auditor_group (tenant_id, group_id) VALUES ($1, $2)", TenantB, GroupB1),
		sql("tenant_auditor_group: point tenant B's auditors at another group", "nava_app", Denied,
			"UPDATE tenant_auditor_group SET group_id = $2 WHERE tenant_id = $1", TenantB, GroupB1),

		// Records pipeline (P3-09, TICK-353). Both tenants hold the record
		// number TCK0000001 and the audit key shared-audit-key, so a lookup by
		// either alone would find tenant B's rows.
		sql("records: look a record number up in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM records WHERE number = 'TCK0000001'"),
		// The composite foreign key holds the tenant: tenant A cannot assign
		// its record to tenant B's group, or name B's principal.
		sql("records: name tenant B's group from tenant A", "nava_app", Denied,
			"INSERT INTO records (tenant_id, id, assignment_group_id) VALUES ($1, $2, $3)", TenantA, NewID, GroupB1),
		sql("records: move tenant A's records to tenant B's group", "nava_worker", Denied,
			"UPDATE records SET assignment_group_id = $2 WHERE tenant_id = $1", TenantA, GroupB1),
		sql("records: name tenant B's principal as requester from tenant A", "nava_app", Denied,
			"INSERT INTO records (tenant_id, id, requester_id) VALUES ($1, $2, $3)", TenantA, NewID, PrincipalB1),
		// audit_log took over from audit_log_v0 in P5-17 (TICK-435): the same
		// five cases, filtered on subject_id. The seed repeats subject
		// RecordA2 in both tenants, so a lookup by subject alone would find
		// both tenants' rows.
		sql("audit_log: read tenant B's audit rows by subject id", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM audit_log WHERE subject_id = $1", RecordB2),
		sql("audit_log: look a subject up in every tenant", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM audit_log WHERE subject_id = $1", RecordA2),
		sql("audit_log: insert an audit row for tenant B", "nava_app", Denied,
			"INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source) VALUES ($1, $2, 'record', 'update', 'record', $3, 'user', 'user')",
			TenantB, NewID, RecordB2),
		sql("audit_log: worker inserts a row with tenant B's tenant_id (WITH CHECK)", "nava_worker", Denied,
			"INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source) VALUES ($1, $2, 'record', 'update', 'record', $3, 'workflow', 'workflow')",
			TenantB, NewID, RecordB2),
		sql("audit_log: rewrite tenant B's audit rows", "nava_app", Denied,
			"UPDATE audit_log SET action = 'forged' WHERE tenant_id = $1", TenantB),
		// On the 3-column key of ADR-005 §4.2 (R1's upsert, P6-02 [Sec C4]),
		// scope '' as every create writes until S18.
		sql("record_counters: read tenant B's counters by type and scope", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM record_counters WHERE entity_definition_id = $1 AND scope_key = ''", EntityDefinitionB1),
		sql("record_counters: take a number from tenant B's counter", "nava_app", Denied,
			`INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES ($1, $2, '', 2)
			 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1`,
			TenantB, EntityDefinitionB1),
		sql("record_counters: start a counter in tenant A for tenant B's type", "nava_worker", Denied,
			"INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES ($1, $2, '', 2)", TenantA, EntityDefinitionB1),
		sql("record_counters: move tenant B's counters", "nava_worker", Denied,
			"UPDATE record_counters SET next = next + 1 WHERE tenant_id = $1", TenantB),

		// Forged tenant: a valid token for tenant A names tenant B in the
		// payload, a header, the query or baggage. The tenant comes only from
		// the token's binding (ADR-001 I2).
		forged("forged tenant in the query string", HTTPCall{Method: http.MethodGet, Path: "/api/v1/me?tenant_id=" + TenantB}),
		forged("forged tenant in a header", HTTPCall{Method: http.MethodGet, Path: "/api/v1/me",
			Header: http.Header{"X-Tenant-Id": {TenantB}, "X-Nava-Tenant": {TenantB}}}),
		// The forged write (P3-18, TICK-366; CTO D-P10 condition 2 on TICK-72).
		// incident is tenant B's only key (testdata/seed.sql), so a handler
		// that honoured the forged tenant would find B's type and file the
		// record under B; under tenant A's own binding it is a 404.
		forged("forged tenant_id in the body of POST /api/v1/records/incident", forgedWrite("iso-ci2-body",
			`{"tenant_id":"`+TenantB+`","data":{}}`, nil)),
		forged("forged data.tenant_id in POST /api/v1/records/incident", forgedWrite("iso-ci2-data",
			`{"data":{"tenant_id":"`+TenantB+`"}}`, nil)),
		forged("forged X-Tenant-Id header on POST /api/v1/records/incident", forgedWrite("iso-ci2-header",
			`{"data":{}}`, http.Header{"X-Tenant-Id": {TenantB}})),
		forged("forged tenant in the body of GET /api/v1/me", HTTPCall{Method: http.MethodGet, Path: "/api/v1/me",
			Body: `{"tenant_id":"` + TenantB + `"}`}),
		// TICK-93: baggage is never read; the edge propagator drops it.
		forged("forged tenant in baggage", HTTPCall{Method: http.MethodGet, Path: "/api/v1/me",
			Header: http.Header{"Baggage": {"tenant_id=" + TenantB + ",nava.tenant_id=" + TenantB}}}),
	}
	paths = append(paths, cursorPaths()...)
	paths = append(paths, idempotencyPaths()...)
	paths = append(paths, ci5Paths()...)
	paths = append(paths, rolePaths()...)
	paths = append(paths, entityTypePaths()...)
	paths = append(paths, recordPaths()...)
	paths = append(paths, recordCI8Paths()...)
	paths = append(paths, cascadePaths()...)
	paths = append(paths, auditPaths()...)
	paths = append(paths, auditReaderPaths()...)
	paths = append(paths, aclPaths()...)
	paths = append(paths, aclAttackPaths()...)
	return append(paths, boundaryPaths()...)
}
