//go:build integration

// The workflow_ref activity against a real PostgreSQL, migrated with the real
// migrations and reached as the real nava_worker role, under RLS.
// make test-integration (scripts/ci/test-integration.sh) sets
// NAVA_TEST_PG_URL to the shared cluster's superuser; this package drops and
// recreates the manifest's databases and roles there, so it runs with the
// other shared-cluster packages one at a time (-p 1).

package workerguard

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	platformdb "github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

func testPassword(role string) (string, bool) { return "test-pw-" + role, true }

func superConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// migrated recreates the manifest's roles and databases, runs the real
// migrations, and returns a superuser connection to the platform database
// (for seeding and reading back) and a nava_worker pool on it.
func migrated(t *testing.T) (*pgx.Conn, *pgxpool.Pool) {
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
	owner := roleConfig(t, m.Owner(), m.PlatformDatabase)
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

	tel, err := telemetry.New(ctx, config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	poolCfg, err := pgxpool.ParseConfig(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	poolCfg.ConnConfig.User = "nava_worker"
	poolCfg.ConnConfig.Password, _ = testPassword("nava_worker")
	poolCfg.ConnConfig.Database = m.PlatformDatabase
	pool, err := platformdb.NewPool(ctx, poolCfg, tel)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return conn, pool
}

func roleConfig(t *testing.T, role, database string) *pgx.ConnConfig {
	t.Helper()
	cfg := superConfig(t).Copy()
	cfg.User = role
	cfg.Password, _ = testPassword(role)
	cfg.Database = database
	return cfg
}

func mustExec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// seedRef inserts, as the superuser, a tenant, a record and an open
// workflow_ref for a start row, and returns the relay's workflow id for it.
func seedRef(t *testing.T, conn *pgx.Conn, tenant uuid.UUID, kind string) (workflowID string, outboxID uuid.UUID) {
	t.Helper()
	outboxID, record := uuid.New(), uuid.New()
	mustExec(t, conn, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT DO NOTHING", tenant)
	mustExec(t, conn, "INSERT INTO records (tenant_id, id) VALUES ($1, $2)", tenant, record)
	mustExec(t, conn, "INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ($1, $2, $3, $4)",
		tenant, outboxID, record, kind)
	return "t:" + tenant.String() + ":" + kind + ":" + outboxID.String(), outboxID
}

func closedAt(t *testing.T, conn *pgx.Conn, tenant, outboxID uuid.UUID) *time.Time {
	t.Helper()
	var at *time.Time
	if err := conn.QueryRow(context.Background(),
		"SELECT closed_at FROM workflow_ref WHERE tenant_id = $1 AND outbox_id = $2", tenant, outboxID).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

// relayWorkflow is a relay-started workflow: it does its work, then closes
// its workflow_ref as its final activity.
func relayWorkflow(ctx workflow.Context) error {
	if err := workflow.Sleep(ctx, time.Minute); err != nil {
		return err
	}
	return CloseWorkflowRef(ctx)
}

// A completed workflow has closed_at set on its workflow_ref row, written by
// nava_worker under RLS with the header tenant; a retry keeps the first time.
func TestCompletedWorkflowClosesItsWorkflowRef(t *testing.T) {
	conn, pool := migrated(t)
	id, outboxID := seedRef(t, conn, tenantA, "approval")
	// Another tenant's open row, which must stay open.
	_, otherOutbox := seedRef(t, conn, tenantB, "approval")

	h := newHarness(t, id)
	(&Activities{Pool: pool}).Register(h.env)
	h.env.ExecuteWorkflow(relayWorkflow)
	if err := h.env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow: %v", err)
	}
	first := closedAt(t, conn, tenantA, outboxID)
	if first == nil {
		t.Fatal("closed_at is NULL after the workflow completed")
	}
	if at := closedAt(t, conn, tenantB, otherOutbox); at != nil {
		t.Errorf("tenant B's workflow_ref was closed: %v", at)
	}

	// A retried close (same workflow id) keeps the first closed_at.
	h2 := newHarness(t, id)
	(&Activities{Pool: pool}).Register(h2.env)
	h2.env.ExecuteWorkflow(relayWorkflow)
	if err := h2.env.GetWorkflowError(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if again := closedAt(t, conn, tenantA, outboxID); again == nil || !again.Equal(*first) {
		t.Errorf("closed_at changed on retry: %v -> %v", first, again)
	}
}

// A tenant-A workflow whose id names tenant B's outbox row finds nothing:
// RLS hides B's row, the activity fails non-retryably, and B's row stays open.
func TestWorkflowRefCloseIsTenantScoped(t *testing.T) {
	conn, pool := migrated(t)
	_, bOutbox := seedRef(t, conn, tenantB, "approval")
	id := "t:" + tenantA.String() + ":approval:" + bOutbox.String()

	h := newHarness(t, id)
	(&Activities{Pool: pool}).Register(h.env)
	h.env.ExecuteWorkflow(relayWorkflow)
	var app *temporal.ApplicationError
	if err := h.env.GetWorkflowError(); !errors.As(err, &app) || app.Type() != ErrTypeWorkflowRef || !app.NonRetryable() {
		t.Fatalf("error %v, want a non-retryable %s", err, ErrTypeWorkflowRef)
	}
	if at := closedAt(t, conn, tenantB, bOutbox); at != nil {
		t.Errorf("tenant B's workflow_ref was closed by a tenant A workflow: %v", at)
	}
}

// An activity whose header names another tenant never reaches the database.
func TestWorkflowRefCloseWithWrongHeaderTouchesNothing(t *testing.T) {
	conn, pool := migrated(t)
	id, outboxID := seedRef(t, conn, tenantA, "approval")

	h := newHarness(t, id)
	(&Activities{Pool: pool}).Register(h.env)
	h.tr.setActivity(stringPayload(t, tenantB.String()))
	h.env.ExecuteWorkflow(relayWorkflow)
	var actErr *temporal.ActivityError
	if err := h.env.GetWorkflowError(); !errors.As(err, &actErr) {
		t.Fatalf("error %v, want an activity failure", err)
	}
	wantGuardError(t, actErr.Unwrap(), guardActivity, outcomeMismatch)
	if at := closedAt(t, conn, tenantA, outboxID); at != nil {
		t.Errorf("closed_at set by an activity that failed its guard: %v", at)
	}
}

// InTenantTx refuses to open a transaction without a guard-verified tenant.
func TestInTenantTxNeedsVerifiedTenant(t *testing.T) {
	_, pool := migrated(t)
	called := false
	err := InTenantTx(context.Background(), pool, func(pgx.Tx) error { called = true; return nil })
	if err == nil || called {
		t.Fatalf("InTenantTx without a tenant: err %v, fn called %v", err, called)
	}
}
