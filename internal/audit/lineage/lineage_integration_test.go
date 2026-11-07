//go:build integration

// P5-34f (TICK-590, ADR-004 R17) against a real PostgreSQL, migrated with the
// real migrations and reached as the real nava_purge and nava_app roles under
// RLS. make test-integration (scripts/ci/test-integration.sh) sets
// NAVA_TEST_PG_URL to the shared cluster's superuser; this package drops and
// recreates the manifest's databases and roles there, so it runs with the
// other shared-cluster packages one at a time (-p 1).

package lineage_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/lineage"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

func testPassword(role string) (string, bool) { return "test-pw-" + role, true }

type itEnv struct {
	super *pgx.Conn // the platform database as superuser: seeding
	purge *pgxpool.Pool
	app   *pgxpool.Pool
}

//nolint:gochecknoglobals // one migrated cluster per package run
var (
	envOnce sync.Once
	envErr  error
	shared  itEnv
)

func migrated(t *testing.T) itEnv {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	envOnce.Do(func() { shared, envErr = setup(url) })
	if envErr != nil {
		t.Fatal(envErr)
	}
	// Each test owns every head: the gauge counts across tenants.
	mustExec(t, shared.super, "DELETE FROM audit_chain_head")
	return shared
}

func setup(url string) (itEnv, error) {
	ctx := context.Background()
	super, err := pgx.ParseConfig(url)
	if err != nil {
		return itEnv{}, err
	}
	m, am, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		return itEnv{}, err
	}
	conn, err := pgx.ConnectConfig(ctx, super)
	if err != nil {
		return itEnv{}, err
	}
	for _, d := range m.DatabaseNames() {
		if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{d}.Sanitize()+" WITH (FORCE)"); err != nil {
			return itEnv{}, err
		}
	}
	for _, r := range m.RoleNames() {
		if _, err := conn.Exec(ctx, "DROP ROLE IF EXISTS "+pgx.Identifier{r}.Sanitize()); err != nil {
			return itEnv{}, err
		}
	}
	_ = conn.Close(ctx)
	log := slog.New(slog.DiscardHandler)
	if _, err := roles.Run(ctx, super, m, testPassword, log); err != nil {
		return itEnv{}, err
	}
	g, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		return itEnv{}, err
	}
	owner := super.Copy()
	owner.User = m.Owner()
	owner.Password, _ = testPassword(m.Owner())
	owner.Database = m.PlatformDatabase
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, am, g, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		return itEnv{}, err
	}
	platformSuper := super.Copy()
	platformSuper.Database = m.PlatformDatabase
	var e itEnv
	if e.super, err = pgx.ConnectConfig(ctx, platformSuper); err != nil {
		return itEnv{}, err
	}
	pool := func(role string) (*pgxpool.Pool, error) {
		cfg, err := pgxpool.ParseConfig("")
		if err != nil {
			return nil, err
		}
		cfg.ConnConfig = super.Copy()
		cfg.ConnConfig.User = role
		cfg.ConnConfig.Password, _ = testPassword(role)
		cfg.ConnConfig.Database = m.PlatformDatabase
		cfg.MaxConns = 4
		return pgxpool.NewWithConfig(ctx, cfg)
	}
	if e.purge, err = pool("nava_purge"); err != nil {
		return itEnv{}, err
	}
	if e.app, err = pool("nava_app"); err != nil {
		return itEnv{}, err
	}
	return e, nil
}

func mustExec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// seedHead seeds, as the superuser, an active tenant whose head's next run
// was due ago before now (nil: the head has no lineage), and returns it.
func seedHead(t *testing.T, e itEnv, ago *time.Duration) uuid.UUID {
	t.Helper()
	tenant := uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	if ago == nil {
		mustExec(t, e.super, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, $2)", tenant, uuid.Must(uuid.NewV7()))
		return tenant
	}
	mustExec(t, e.super, `INSERT INTO audit_chain_head (tenant_id, id, next_seal_outbox_id, next_seal_at)
		VALUES ($1, $2, $3, now() - make_interval(secs => $4))`,
		tenant, uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), ago.Seconds())
	return tenant
}

func dur(d time.Duration) *time.Duration { return &d }

// Past and inside are 30 s either side of R17's 10 min: a mutation of the
// interval by more than that, in either reader, fails these tests.
//
//nolint:gochecknoglobals // test fixtures
var (
	past   = dur(lineage.DeadAfter + 30*time.Second)
	inside = dur(lineage.DeadAfter - 30*time.Second)
)

func newMonitor(t *testing.T, e itEnv) *lineage.Monitor {
	t.Helper()
	m, err := lineage.NewMonitor(e.purge, sdkmetric.NewMeterProvider(), slog.New(slog.DiscardHandler), 0)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func gauge(t *testing.T, m *lineage.Monitor) int64 {
	t.Helper()
	if err := m.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	n, ok := m.Value()
	if !ok {
		t.Fatal("gauge withdrawn after a successful read")
	}
	return n
}

// AC: one head past the threshold reads 1; one inside it reads 0.
func TestGaugeThreshold(t *testing.T) {
	t.Run("past reads 1", func(t *testing.T) {
		e := migrated(t)
		dead := seedHead(t, e, past)
		if n := gauge(t, newMonitor(t, e)); n != 1 {
			t.Fatalf("gauge = %d, want 1", n)
		}
		got, err := lineage.DeadLineages(context.Background(), e.purge)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].TenantID != dead {
			t.Fatalf("DeadLineages = %v, want only %s", got, dead)
		}
	})
	t.Run("inside reads 0", func(t *testing.T) {
		e := migrated(t)
		seedHead(t, e, inside)
		seedHead(t, e, nil) // no lineage yet: not dead
		if n := gauge(t, newMonitor(t, e)); n != 0 {
			t.Fatalf("gauge = %d, want 0", n)
		}
	})
	t.Run("counts every tenant", func(t *testing.T) {
		e := migrated(t)
		seedHead(t, e, past)
		seedHead(t, e, past)
		seedHead(t, e, inside)
		if n := gauge(t, newMonitor(t, e)); n != 2 {
			t.Fatalf("gauge = %d, want 2", n)
		}
	})
}

// The reseed gate and the gauge agree on every head: the alert fires exactly
// when a reseed becomes allowed.
func TestReseedGateAgreesWithGauge(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	heads := map[uuid.UUID]bool{
		seedHead(t, e, past):   true,
		seedHead(t, e, inside): false,
		seedHead(t, e, nil):    false,
	}
	got, err := lineage.DeadLineages(ctx, e.purge)
	if err != nil {
		t.Fatal(err)
	}
	for tenant, want := range heads {
		var allowed bool
		err := tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error {
			var err error
			allowed, err = lineage.ReseedAllowed(ctx, tx, tenant)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		inGauge := slices.ContainsFunc(got, func(d lineage.Dead) bool { return d.TenantID == tenant })
		if allowed != want || inGauge != want {
			t.Errorf("tenant %s: reseed allowed %v, in gauge %v, want both %v", tenant, allowed, inGauge, want)
		}
	}
}

// nava_app sees only its own tenant's head, so another tenant's dead head is
// "not found" to the gate, never allowed.
func TestReseedGateIsTenantScoped(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	other := seedHead(t, e, past)
	own := seedHead(t, e, inside)
	err := tenancy.InTenantTx(ctx, e.app, own, func(tx pgx.Tx) error {
		_, err := lineage.ReseedAllowed(ctx, tx, other)
		return err
	})
	if !errors.Is(err, lineage.ErrHeadNotFound) {
		t.Fatalf("gate on another tenant's head: %v, want ErrHeadNotFound", err)
	}
}

func wantSQLState(t *testing.T, err error, code, what string) {
	t.Helper()
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != code {
		t.Fatalf("%s: %v, want SQLSTATE %s", what, err, code)
	}
}

// Security: nava_purge reads only dead heads, only tenant_id and
// next_seal_at, and writes nothing.
func TestPurgeRoleReadsOnlyTheCount(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	seedHead(t, e, past)
	alive := seedHead(t, e, inside)
	var n int64
	if err := e.purge.QueryRow(ctx, "SELECT count(*) FROM audit_chain_head").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("nava_purge sees %d heads with no WHERE, want only the 1 dead one", n)
	}
	if err := e.purge.QueryRow(ctx, "SELECT count(*) FROM audit_chain_head WHERE tenant_id = $1", alive).Scan(&n); err != nil || n != 0 {
		t.Fatalf("nava_purge sees the alive head: %d, %v", n, err)
	}
	for _, col := range []string{"entry_hash", "seq", "version", "next_seal_outbox_id", "export_lease_until", "id"} {
		_, err := e.purge.Exec(ctx, "SELECT "+col+" FROM audit_chain_head")
		wantSQLState(t, err, "42501", "SELECT "+col)
	}
	_, err := e.purge.Exec(ctx, "UPDATE audit_chain_head SET next_seal_at = now()")
	wantSQLState(t, err, "42501", "UPDATE")
	_, err = e.purge.Exec(ctx, "DELETE FROM audit_chain_head")
	wantSQLState(t, err, "42501", "DELETE")
}

// The policy, the shared Predicate and PolicyQual are one predicate: the
// migration's policy deparses to PolicyQual, and so does Predicate.
func TestPolicyIsThePredicate(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	var qual string
	if err := e.super.QueryRow(ctx, `SELECT qual FROM pg_policies
		WHERE schemaname = 'public' AND tablename = 'audit_chain_head' AND policyname = 'audit_chain_head_purge_select'`).Scan(&qual); err != nil {
		t.Fatal(err)
	}
	if qual != lineage.PolicyQual {
		t.Fatalf("policy qual = %q, want lineage.PolicyQual %q", qual, lineage.PolicyQual)
	}
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "CREATE TEMP TABLE r17 (next_seal_at timestamptz) ON COMMIT DROP"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "CREATE POLICY r17 ON r17 USING ("+lineage.Predicate+")"); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, "SELECT qual FROM pg_policies WHERE tablename = 'r17' AND policyname = 'r17'").Scan(&qual); err != nil {
		t.Fatal(err)
	}
	if qual != lineage.PolicyQual {
		t.Fatalf("Predicate deparses to %q, want PolicyQual %q", qual, lineage.PolicyQual)
	}
}

// AC: promtool test rules, AuditDeadLineage silent at 0 and firing at 1, with
// the pinned prom/prometheus image (docs/pins.md). Docker is the integration
// job's, as for the PostgreSQL container.
func TestAlertRules(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pins, err := os.ReadFile(filepath.Join(root, "docs", "pins.md"))
	if err != nil {
		t.Fatal(err)
	}
	tag := regexp.MustCompile("(?m)^\\| `prom/prometheus` \\| ([^ |]+) \\|").FindSubmatch(pins)
	if tag == nil {
		t.Fatal("no prom/prometheus row in docs/pins.md")
	}
	dir := filepath.Join(root, "deploy", "compose", "prometheus")
	cmd := exec.CommandContext(context.Background(), "docker", "run", "--rm", "--network", "none",
		"--entrypoint", "promtool", "-v", dir+":/p:ro", "-w", "/p/tests",
		"prom/prometheus:"+string(tag[1]), "test", "rules", "audit_test.yml")
	cmd.Env = append(os.Environ(), "MSYS_NO_PATHCONV=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("promtool test rules: %v\n%s", err, out)
	}
}
