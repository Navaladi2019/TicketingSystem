//go:build integration

// P3-10 against a real PostgreSQL, migrated with the real migrations and
// reached as the real nava_app role, under RLS. make test-integration
// (scripts/ci/test-integration.sh) sets NAVA_TEST_PG_URL to the shared
// cluster's superuser; this package drops and recreates the manifest's
// databases and roles there, so it runs with the other shared-cluster
// packages one at a time (-p 1).

package pipeline_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const entityKey = "incident"

func testPassword(role string) (string, bool) { return "test-pw-" + role, true }

func superConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	dsn := os.Getenv("NAVA_TEST_PG_URL")
	if dsn == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

type env struct {
	super *pgx.Conn // the platform database as superuser, for seeding and reading back
	app   *pgxpool.Pool
	// alloc is the allocator pool newPipeline's TxRunner uses, where a create
	// takes its number (D-3-9 as amended 2026-11-10): nava_app, separate from
	// app as nava-core's is (TICK-657). Nil: the runner uses app.
	alloc    *pgxpool.Pool
	platform string
}

// migrated recreates the manifest's roles and databases, runs the real
// migrations, and opens a nava_app pool on the platform database.
func migrated(t *testing.T) env {
	t.Helper()
	ctx := context.Background()
	m, am, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		t.Fatal(err)
	}
	super, err := pgx.ConnectConfig(ctx, superConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range m.DatabaseNames() {
		mustExec(t, super, "DROP DATABASE IF EXISTS "+pgx.Identifier{d}.Sanitize()+" WITH (FORCE)")
	}
	for _, r := range m.RoleNames() {
		mustExec(t, super, "DROP ROLE IF EXISTS "+pgx.Identifier{r}.Sanitize())
	}
	_ = super.Close(ctx)
	log := slog.New(slog.DiscardHandler)
	if _, err := roles.Run(ctx, superConfig(t), m, testPassword, log); err != nil {
		t.Fatalf("db-roles: %v", err)
	}
	g, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		t.Fatal(err)
	}
	owner := superConfig(t).Copy()
	owner.User, owner.Database = m.Owner(), m.PlatformDatabase
	owner.Password, _ = testPassword(m.Owner())
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, am, g, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	platformSuper := superConfig(t).Copy()
	platformSuper.Database = m.PlatformDatabase
	conn, err := pgx.ConnectConfig(ctx, platformSuper)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	dsn := appDSN(t, m.PlatformDatabase)
	return env{super: conn, app: appPool(t, dsn), alloc: allocPool(t, dsn), platform: m.PlatformDatabase}
}

// appDSN is the nava_app connection string on the platform database.
func appDSN(t *testing.T, database string) string {
	t.Helper()
	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := testPassword("nava_app")
	u.User = url.UserPassword("nava_app", pw)
	u.Path = "/" + database
	return u.String()
}

// serverPoolConfig parses dsn into the pool config nava-core builds for
// NAVA_DATABASE_URL (cmd/nava-core/identity.go): QueryExecModeExec, so
// statements stay unnamed behind PgBouncer's transaction mode, and a 5s
// connect timeout. In that mode no parameter has a declared type and pgx
// picks the wire format from the Go type alone, so a []byte bound to a
// json or jsonb parameter goes as bytea and PostgreSQL refuses it (22P02,
// TICK-411, TICK-471). Every nava_app pool in this package is built from it;
// a test on pgx's default mode would hide that class of bug. Keep it in step
// with nava-core's pool.
func serverPoolConfig(t *testing.T, dsn string) *pgxpool.Config {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	return cfg
}

// allocPool is a nava_app allocator pool on dsn, as nava-core opens it
// (cmd/nava-core/allocator.go): the server's protocol mode, allocatorMaxConns
// wide, and every connection's role checked (tenancy.AllocPoolConfig).
func allocPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	cfg := serverPoolConfig(t, dsn)
	cfg.MaxConns = allocatorMaxConns
	if err := tenancy.AllocPoolConfig(cfg); err != nil {
		t.Fatal(err)
	}
	return newPool(t, cfg)
}

// newPool opens a pool on cfg, closed when the test ends.
func newPool(t *testing.T, cfg *pgxpool.Config) *pgxpool.Pool {
	t.Helper()
	p, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

// allocatorMaxConns is nava-core's allocator pool width (allocatorMaxConns in
// cmd/nava-core/allocator.go).
const allocatorMaxConns = 4

// appPool is a nava_app pool in the server's protocol mode (serverPoolConfig).
func appPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	p, err := pgxpool.NewWithConfig(context.Background(), serverPoolConfig(t, dsn))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func mustExec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func count(t *testing.T, conn *pgx.Conn, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// schemaV1 is the incident type at version 1. status is the state property
// and requester a fixed column; priority is required with a default, title
// required without one.
const schemaV1 = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["title", "priority"],
  "properties": {
    "title": {"type": "string", "minLength": 1},
    "status": {"type": "string", "enum": ["open", "closed"]},
    "priority": {"type": "string", "enum": ["low", "high"], "default": "low"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "requester": {"type": "string", "format": "uuid"},
    "summary": {"type": "string"}
  }
}`

// schemaV2 adds channel, with a default.
const schemaV2 = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["title", "priority"],
  "properties": {
    "title": {"type": "string", "minLength": 1},
    "status": {"type": "string", "enum": ["open", "closed"]},
    "priority": {"type": "string", "enum": ["low", "high"], "default": "low"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "requester": {"type": "string", "format": "uuid"},
    "summary": {"type": "string"},
    "channel": {"type": "string", "default": "email"}
  }
}`

func definition(schema string) metadata.Definition {
	return metadata.Definition{
		Key:          entityKey,
		Schema:       json.RawMessage(schema),
		NumberPrefix: "INC",
		FixedColumns: map[string]string{"status": metadata.ColumnState, "requester": metadata.ColumnRequesterID},
		States: metadata.StateModel{
			Initial:     "open",
			States:      []string{"open", "closed"},
			Transitions: []metadata.Transition{{From: "open", To: "closed"}},
		},
	}
}

// tenantFixture is one tenant with a principal and the incident type at v1.
type tenantFixture struct {
	id        uuid.UUID
	principal identity.Principal
}

func (f tenantFixture) system() identity.Principal { return identity.Principal{TenantID: f.id} }

// workflowDefinition creates, as the superuser, one workflow definition with
// a new id in each tenant of fs, on its entity type entityKey, and returns
// the id: cascade_start's definition_id references workflow_definitions
// (P3-40), so every claim names a definition of its own tenant.
func workflowDefinition(t *testing.T, e env, fs ...tenantFixture) uuid.UUID {
	t.Helper()
	id := uuid.New()
	for _, f := range fs {
		// The audit entry is the one the commit check wants (P3-41b); Publish
		// is not used because the fixture needs a bare definition with no version.
		mustExec(t, e.super, `WITH w AS (INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version, created_by)
				SELECT $1, $2, $3, d.id, 1, $4
				FROM entity_definitions d WHERE d.tenant_id = $1 AND d.key = $5 RETURNING tenant_id, id)
			INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source, content_sha256)
			SELECT tenant_id, gen_random_uuid(), 'config', 'create', 'workflow_definition', id, 'system', 'system', sha256(''::bytea) FROM w`,
			f.id, id, "wf_"+strings.ReplaceAll(id.String(), "-", ""), f.principal.PrincipalID, entityKey)
	}
	return id
}

func seedTenant(t *testing.T, e env) tenantFixture {
	t.Helper()
	return seedTenantWith(t, e, definition(schemaV1))
}

// seedTenantWith is one tenant with a principal and the given entity types,
// each at version 1.
func seedTenantWith(t *testing.T, e env, defs ...metadata.Definition) tenantFixture {
	t.Helper()
	f := tenantFixture{id: uuid.New()}
	f.principal = identity.Principal{TenantID: f.id, PrincipalID: uuid.New()}
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", f.id)
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, $3, true, now())`, f.id, f.principal.PrincipalID, "kc-"+f.principal.PrincipalID.String())
	for _, d := range defs {
		inTx(t, e, f.id, func(tx pgx.Tx) error {
			if _, err := (entitytypes.Store{}).Create(context.Background(), tx, f.id, entitytypes.NewInput{
				Name: d.Key, PublishedBy: f.principal.PrincipalID, Definition: d,
			}); err != nil {
				return err
			}
			// Authorization fails closed: the fixture's principal may read and
			// write every record of the type (P3-31).
			_, err := acltest.Publish(context.Background(), tx, d.Key, acltest.AllowUsers(f.principal.PrincipalID))
			return err
		})
	}
	return f
}

func publishV2(t *testing.T, e env, f tenantFixture) {
	t.Helper()
	inTx(t, e, f.id, func(tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Publish(context.Background(), tx, f.id, entitytypes.PublishInput{
			Key: entityKey, PublishedBy: f.principal.PrincipalID, Definition: definition(schemaV2),
		})
		return err
	})
}

func inTx(t *testing.T, e env, tenant uuid.UUID, fn func(pgx.Tx) error) {
	t.Helper()
	if err := tenancy.InTenantTx(context.Background(), e.app, tenant, fn); err != nil {
		t.Fatal(err)
	}
}

// countingSink records every call and, inside the write's transaction, how
// many audit entries the record has at that point (read as nava_app, through
// the commit-check columns and the writer SELECT policy).
type countingSink struct {
	mu     sync.Mutex
	events []pipeline.Event
	audits []int
	err    error
}

func (s *countingSink) RecordChanged(ctx context.Context, tx pgx.Tx, ev pipeline.Event) error {
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE subject_type = 'record' AND subject_id = $1`, ev.RecordID).Scan(&n); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	s.audits = append(s.audits, n)
	return s.err
}

func (s *countingSink) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

func newPipeline(e env, sink pipeline.EventSink) *pipeline.Pipeline {
	if sink == nil {
		sink = pipeline.NoopSink{}
	}
	return pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, sink, e.runner(), pipeline.SystemClock{})
}

// noLatencyBudget is the allocation timeout of the tests that check what a
// create does, not how fast it allocates: a loaded CI host must not fail them
// on the production 200 ms (ADR-005 A5; TICK-778, where 50 concurrent creates
// through PgBouncer timed out on a cold pool).
// TestAllocationTimeoutIsRetryableAndMeasured sets its own, short timeout,
// and the starvation controls run on the production default.
const noLatencyBudget = 5 * time.Second

// runner is the pipeline's TxRunner: a PoolRunner on the allocator pool, or
// on app when e has none, with the noLatencyBudget allocation timeout.
func (e env) runner() pipeline.PoolRunner {
	if e.alloc != nil {
		return pipeline.PoolRunner{Pool: e.alloc, AllocTimeout: noLatencyBudget}
	}
	return pipeline.PoolRunner{Pool: e.app, AllocTimeout: noLatencyBudget}
}

// write runs one Write in its own committed tenant transaction.
func write(ctx context.Context, e env, p *pipeline.Pipeline, tenant uuid.UUID, cmd pipeline.Command) (pipeline.Result, error) {
	var res pipeline.Result
	err := tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error {
		var err error
		res, err = p.Write(ctx, tx, cmd)
		return err
	})
	return res, err
}

// workflowCausation is a workflow write's causation: one deeper than the
// start that caused it (ADR-001 A5-7), never depth 0.
func workflowCausation() outbox.Causation { return outbox.Causation{OriginID: uuid.New(), Depth: 1} }

func createCmd(f tenantFixture, data map[string]any) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpCreate, EntityKey: entityKey, Data: data, Actor: f.principal, Source: pipeline.SourceUser}
}

func updateCmd(f tenantFixture, id uuid.UUID, version int64, patch map[string]any) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpUpdate, EntityKey: entityKey, RecordID: id, ExpectedVersion: version,
		Data: patch, Actor: f.principal, Source: pipeline.SourceUser}
}

func mustCreate(t *testing.T, e env, p *pipeline.Pipeline, f tenantFixture, data map[string]any) pipeline.Result {
	t.Helper()
	res, err := write(context.Background(), e, p, f.id, createCmd(f, data))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return res
}

// storedData returns the record's data column and fixed columns as the
// superuser sees them.
func storedData(t *testing.T, e env, id uuid.UUID) (data map[string]any, state string, requester *uuid.UUID, version int64, schemaVersion int) {
	t.Helper()
	var raw []byte
	if err := e.super.QueryRow(context.Background(),
		`SELECT data, state, requester_id, version, schema_version FROM records WHERE id = $1`, id).
		Scan(&raw, &state, &requester, &version, &schemaVersion); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	return data, state, requester, version, schemaVersion
}

// waitForLockWait polls until some nava_app backend waits on a lock: the
// second of two racing writes is then blocked behind the first.
func waitForLockWait(t *testing.T, e env) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for count(t, e.super, `SELECT count(*) FROM pg_stat_activity WHERE usename = 'nava_app' AND wait_event_type = 'Lock'`) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the second write never blocked on the first")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// seedEntrySQL is the audit entry of a records row a fixture writes outside
// the pipeline: tenant $1, entry id $2, record $3, version_after $4. The
// records commit-check trigger (P5-13b) refuses the fixture's COMMIT without it.
const seedEntrySQL = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
	VALUES ($1, $2, 'record', 'create', 'record', $3, $4, 'system', 'system')`

// seedRecord runs sql (an INSERT of record id in tenant) as the superuser,
// in one transaction with the record's audit entry, as every records write
// must now be (CI-A2).
func seedRecord(t *testing.T, e env, tenant, id uuid.UUID, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	err := pgx.BeginFunc(ctx, e.super, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, seedEntrySQL, tenant, uuid.Must(uuid.NewV7()), id, 1)
		return err
	})
	if err != nil {
		t.Fatalf("seed record %s: %v", id, err)
	}
}

// entries counts the tenant's audit_log entries with the given action.
func entries(t *testing.T, e env, tenant uuid.UUID, action string) int {
	t.Helper()
	return count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = $2`, tenant, action)
}

// recordEntries counts the record's entries (subject the record).
func recordEntries(t *testing.T, e env, record uuid.UUID) int {
	t.Helper()
	return count(t, e.super, `SELECT count(*) FROM audit_log WHERE subject_type = 'record' AND subject_id = $1`, record)
}
