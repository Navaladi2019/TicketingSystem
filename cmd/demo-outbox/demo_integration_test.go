//go:build integration

// The S7 demo against a real PostgreSQL, migrated with the real migrations
// and reached as nava_app under RLS (TICK-411): the seed applies twice, and
// the demo writes its record through the pipeline and its rows at versions
// 1, 2 and 3. make test-integration (scripts/ci/test-integration.sh) sets
// NAVA_TEST_PG_URL to the shared cluster's superuser; this package drops and
// recreates the manifest's databases and roles there, so it runs with the
// other shared-cluster packages one at a time (-p 1).

package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
)

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

// migrated recreates the manifest's roles and databases, runs the real
// migrations, and returns a superuser connection to the platform database and
// the nava_app connection string the demo takes.
func migrated(t *testing.T) (*pgx.Conn, string) {
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

	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := testPassword("nava_app")
	u.User = url.UserPassword("nava_app", pw)
	u.Path = "/" + m.PlatformDatabase
	return conn, u.String()
}

func mustExec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// seed is what scripts/dev/demo-outbox.sh runs as the superuser.
func seed(t *testing.T, super *pgx.Conn) {
	t.Helper()
	mustExec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT DO NOTHING", tenantA)
	s, err := seedSQL()
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, super, s)
}

// demo runs the command once and returns its "name: value" lines.
func demo(t *testing.T, dsn, key string) map[string]string {
	t.Helper()
	env := map[string]string{"NAVA_DATABASE_URL": dsn, "DEDUPE_KEY": key}
	var out bytes.Buffer
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	if err := run(context.Background(), lookup, &out); err != nil {
		t.Fatalf("demo-outbox: %v", err)
	}
	got := map[string]string{}
	for line := range strings.Lines(out.String()) {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), ": "); ok {
			got[k] = v
		}
	}
	return got
}

func TestDemoWritesThroughThePipeline(t *testing.T) {
	super, dsn := migrated(t)
	ctx := context.Background()
	seed(t, super)
	seed(t, super) // idempotent

	first := demo(t, dsn, "it-key")
	second := demo(t, dsn, "it-key")
	for k, v := range first {
		if second[k] != v {
			t.Errorf("rerun printed %s %q, the first run %q", k, second[k], v)
		}
	}
	record, err := uuid.Parse(first["record"])
	if err != nil {
		t.Fatalf("record %q: %v", first["record"], err)
	}

	// The pipeline wrote the record: three writes, one audit row each, and
	// the rerun replayed them (D-3-13) and wrote nothing.
	var (
		version int64
		number  string
		audits  int
	)
	if err := super.QueryRow(ctx, `SELECT r.version, r.number,
		(SELECT count(*) FROM audit_log_v0 a WHERE a.tenant_id = r.tenant_id AND a.record_id = r.id)
		FROM records r WHERE r.tenant_id = $1 AND r.id = $2`, tenantA, record).Scan(&version, &number, &audits); err != nil {
		t.Fatal(err)
	}
	if version != 3 || audits != 3 || !strings.HasPrefix(number, entityPrefix) {
		t.Errorf("record version %d, %d audit rows, number %q; want 3, 3, %s-...", version, audits, number, entityPrefix)
	}

	// One row per write, at the version that write gave the record, and the
	// signals queued behind the start.
	for _, c := range []struct {
		line string
		seq  int64
	}{{"start row", 1}, {"signal row 1", 2}, {"signal row 2", 3}} {
		var (
			seq       int64
			targetKey string
		)
		if err := super.QueryRow(ctx, `SELECT target_seq, target_key FROM outbox WHERE tenant_id = $1 AND id = $2`,
			tenantA, first[c.line]).Scan(&seq, &targetKey); err != nil {
			t.Fatalf("%s %s: %v", c.line, first[c.line], err)
		}
		if seq != c.seq || targetKey != workflowType+":"+first["start row"] {
			t.Errorf("%s: target_seq %d, target_key %s; want %d, %s:%s", c.line, seq, targetKey, c.seq, workflowType, first["start row"])
		}
	}
	var rows, claims int
	if err := super.QueryRow(ctx, `SELECT (SELECT count(*) FROM outbox WHERE tenant_id = $1),
		(SELECT count(*) FROM cascade_start WHERE tenant_id = $1)`, tenantA).Scan(&rows, &claims); err != nil {
		t.Fatal(err)
	}
	if rows != 3 || claims != 1 {
		t.Errorf("%d outbox rows and %d start keys after two runs, want 3 and 1", rows, claims)
	}

	// Another key is another record and another run.
	other := demo(t, dsn, "it-other")
	if other["record"] == first["record"] || other["start row"] == first["start row"] {
		t.Errorf("a new key reused the record or the start: %v", other)
	}
}
