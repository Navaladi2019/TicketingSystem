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
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
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
	super    *pgx.Conn // the platform database as superuser, for seeding and reading back
	app      *pgxpool.Pool
	platform string
}

// migrated recreates the manifest's roles and databases, runs the real
// migrations, and opens a nava_app pool on the platform database.
func migrated(t *testing.T) env {
	t.Helper()
	ctx := context.Background()
	m, err := grants.Parse(db.Grants())
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
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, g, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	platformSuper := superConfig(t).Copy()
	platformSuper.Database = m.PlatformDatabase
	conn, err := pgx.ConnectConfig(ctx, platformSuper)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return env{super: conn, app: appPool(t, appDSN(t, m.PlatformDatabase)), platform: m.PlatformDatabase}
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

func appPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	p, err := pgxpool.NewWithConfig(context.Background(), cfg)
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
			_, err := entitytypes.Store{}.Create(context.Background(), tx, f.id, entitytypes.NewInput{
				Name: d.Key, PublishedBy: f.principal.PrincipalID, Definition: d,
			})
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
// many audit rows the record has at that point.
type countingSink struct {
	mu     sync.Mutex
	events []pipeline.Event
	audits []int
	err    error
}

func (s *countingSink) RecordChanged(ctx context.Context, tx pgx.Tx, ev pipeline.Event) error {
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE record_id = $1`, ev.RecordID).Scan(&n); err != nil {
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
	return pipeline.New(authz.TenantMemberAuthorizer{}, entitytypes.Store{}, sink, pipeline.PoolRunner{Pool: e.app}, pipeline.SystemClock{})
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
