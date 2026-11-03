//go:build integration

package dbtest

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// P5-11-E3b (TICK-509): the two commit-check budget rows of ADR-004-E3
// ("Budget (replaces the 42.2 µs row)"), measured in the CI integration job
// at the production partition count, each with a same-run reference. The
// verdict rule and its unit tests are in commit_check_verdict_test.go; the
// committed numbers are in docs/budgets.md.

const (
	// budgetPartitionCount is DEFAULT, the current month and 36 retained
	// months (ADR-004-E3).
	budgetPartitionCount = 38
	budgetRetainedMonths = 36
	// budgetRefFn is the same-run reference: audit_commit_check without
	// the plan_cache_mode clause.
	budgetRefFn = "p509_commit_check_ref"
	// budgetImportSubjects is the import row's transaction size.
	budgetImportSubjects = 10000
	// budgetTenant writes the single-subject transactions (in the young
	// fixture's statistics, as in the TA's budget.pgbench).
	budgetTenant = 4
)

// budgetNs are the single-subject run lengths, in the order
// commitCheckBudget's per-call arrays hold them.
var budgetNs = [3]int{200, 1000, 5000}

// budgetArm is one trigger function on its own table, on one fixture.
type budgetArm struct {
	name, table, fn string
}

var budgetArms = [2]budgetArm{
	{"clause", "p509_clause", "audit_commit_check"},
	{"ref", "p509_ref", budgetRefFn},
}

// TestCommitCheckBudget measures both rows and applies the verdict rule:
// over a ceiling with an out-of-range same-run ratio is the E3-R2 S3
// trigger; over a ceiling with in-range ratios is a runner problem, and the
// job is re-run. Neither ever moves a ceiling.
func TestCommitCheckBudget(t *testing.T) {
	start := time.Now()
	super := budgetFixture(t)
	t.Logf("fixture (%d partitions) built in %v", budgetPartitionCount, time.Since(start).Round(time.Millisecond))

	var b commitCheckBudget
	// Single subject: for each N, both arms, each on a fresh young fixture
	// in a fresh session, after pg_stat_reset().
	for i, n := range budgetNs {
		for _, arm := range budgetArms {
			us := budgetPerCall(t, super, arm, n)
			t.Logf("single subject, %s, N = %d: %.1f µs per call", arm.name, n, us)
			if arm.name == "clause" {
				b.PerCallClause[i] = us
			} else {
				b.PerCallRef[i] = us
			}
		}
	}
	// Import: 3 reps, the arms alternating, each on a fresh young fixture
	// in a fresh nava_worker session.
	var clauseRuns, refRuns []float64
	k := 0
	for rep := range 3 {
		for _, arm := range budgetArms {
			k++
			us := budgetImport(t, super, arm, k)
			t.Logf("import, %s, rep %d: %.1f µs per probe", arm.name, rep+1, us)
			if arm.name == "clause" {
				clauseRuns = append(clauseRuns, us)
			} else {
				refRuns = append(refRuns, us)
			}
		}
	}
	b.ImportClause, b.ImportRef = median(clauseRuns), median(refRuns)

	single, imp := b.singleLine(budgetPartitionCount), b.importLine(budgetPartitionCount, clauseRuns, refRuns)
	t.Log(single)
	t.Log(imp)
	t.Logf("COMMIT-CHECK-BUDGET wall_time=%v", time.Since(start).Round(time.Millisecond))
	for _, r := range []struct {
		line string
		v    budgetVerdict
	}{{single, b.singleVerdict()}, {imp, b.importVerdict()}} {
		switch r.v {
		case budgetS3:
			t.Errorf("E3-R2 S3 trigger: over the ceiling with a same-run ratio out of range; tell the Technical Lead on TICK-509, do not change the ceilings: %s", r.line)
		case budgetRerun:
			t.Errorf("over the ceiling with in-range same-run ratios: a runner problem, re-run the job (not S3): %s", r.line)
		}
	}
}

// budgetFixture is a migrated platform database with audit_log at 38
// partitions: DEFAULT, 36 retained monthly partitions of 200 rows each
// (the Technical Architect's builder, rfc-010/recheck-38-partitions.md),
// and the young partition budgetYoung rebuilds. It also creates the
// reference function and one table per arm with its constraint trigger.
func budgetFixture(t *testing.T) *pgx.Conn {
	t.Helper()
	migratedPlatform(t)
	super := superNava(t)
	retainedMonths(t, super, budgetRetainedMonths)
	exec(t, super, withoutClause(t, commitCheckDef(t, super), budgetRefFn))
	for _, a := range budgetArms {
		exec(t, super, "CREATE TABLE "+a.table+" (tenant_id uuid NOT NULL, id uuid NOT NULL, val text, PRIMARY KEY (tenant_id, id))")
		exec(t, super, "GRANT SELECT, INSERT ON "+a.table+" TO nava_worker")
		exec(t, super, "CREATE CONSTRAINT TRIGGER "+a.table+"_cc AFTER INSERT ON "+a.table+
			" DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION "+a.fn+"('record', 'id')")
	}
	exec(t, super, "INSERT INTO tenants (id, state) SELECT "+fixtureTenant("g")+", 'active' FROM generate_series(1, 50) g ON CONFLICT DO NOTHING")
	budgetYoung(t, super)
	if got := queryString(t, super, "SELECT count(*)::text FROM pg_inherits WHERE inhparent = 'audit_log'::regclass"); got != fmt.Sprint(budgetPartitionCount) {
		t.Fatalf("audit_log has %s partitions, want %d", got, budgetPartitionCount)
	}
	return super
}

// retainedMonths adds n monthly partitions below the current month, 200
// rows each over 50 fixture tenants, as Epic 30 must create them
// (n_distinct = 1 on occurred_at, RLS forced).
func retainedMonths(t *testing.T, super *pgx.Conn, n int) {
	t.Helper()
	exec(t, super, `DO $$
DECLARE m int; lo timestamptz; p text;
BEGIN
  FOR m IN 1..`+fmt.Sprint(n)+` LOOP
    lo := date_trunc('month', now()) - make_interval(months => m);
    p := 'audit_log_p509_m' || m;
    EXECUTE format('CREATE TABLE %I PARTITION OF audit_log FOR VALUES FROM (%L) TO (%L)', p, lo, lo + interval '1 month');
    EXECUTE format('ALTER TABLE %I ALTER COLUMN occurred_at SET (n_distinct = 1)', p);
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', p);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', p);
  END LOOP;
END $$`)
	for _, stmt := range []string{
		"SET session_replication_role = replica",
		`INSERT INTO audit_log (tenant_id, id, occurred_at, txid, domain, action, subject_type, subject_id, actor_system, source)
		 SELECT ` + fixtureTenant("1 + g % 50") + `, gen_random_uuid(),
		        date_trunc('month', now()) - make_interval(months => m) + g * interval '3 hours', (1000000 + m * 1000 + g)::text::xid8,
		        'record', 'update', 'record', gen_random_uuid(), 'workflow', 'workflow'
		 FROM generate_series(1, ` + fmt.Sprint(n) + `) m, generate_series(1, 200) g`,
		"SET session_replication_role = DEFAULT",
	} {
		exec(t, super, stmt)
	}
}

// budgetYoung rebuilds the young partition (the TA's young.sql): the arm
// tables emptied, the current month's partition recreated with 60 rows over
// 20 tenants and autovacuum off, and audit_log analysed, so the young
// partition is 2 pages.
func budgetYoung(t *testing.T, super *pgx.Conn) {
	t.Helper()
	exec(t, super, "TRUNCATE "+budgetArms[0].table+", "+budgetArms[1].table)
	exec(t, super, "DROP TABLE IF EXISTS audit_log_p511_young")
	young := youngPartition(t, super)
	loadEntries(t, super, "SELECT "+fixtureTenant("1 + g % 20")+", "+thisMonth("g", "60")+" FROM generate_series(1, 60) g")
	exec(t, super, "ANALYZE audit_log")
	if got := queryString(t, super, "SELECT relpages::text FROM pg_class WHERE relname = $1", young); got != "2" {
		t.Fatalf("young partition has %s pages after ANALYZE, want 2", got)
	}
}

// budgetPerCall is one single-subject run: on a fresh young fixture, after
// pg_stat_reset(), one new session (one pooled backend) with
// track_functions = pl runs n transactions, each as nava_worker writing one
// audit entry and one row of arm's table (the TA's budget.pgbench). It
// returns the arm function's self_time / calls in µs.
func budgetPerCall(t *testing.T, super *pgx.Conn, arm budgetArm, n int) float64 {
	t.Helper()
	ctx := context.Background()
	budgetYoung(t, super)
	cfg := superConfig(t).Copy()
	cfg.Database = "nava"
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	exec(t, conn, "SET track_functions = 'pl'")
	exec(t, super, "SELECT pg_stat_reset()")
	tenant := fixtureTenantID(budgetTenant)
	for range n {
		sid := uuid.NewString()
		// One simple-protocol round trip per transaction, like pgbench's
		// default; the values are literals the test made.
		txn := "BEGIN; SET LOCAL ROLE nava_worker; SELECT set_config('app.tenant_id', '" + tenant + "', true); " +
			"INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source) " +
			"VALUES ('" + tenant + "', gen_random_uuid(), 'record', 'create', 'record', '" + sid + "', 1, 'workflow', 'workflow'); " +
			"INSERT INTO " + arm.table + " VALUES ('" + tenant + "', '" + sid + "', 'v'); COMMIT"
		if _, err := conn.Exec(ctx, txn); err != nil {
			t.Fatalf("%s, N = %d: %v", arm.name, n, err)
		}
	}
	// Flush this backend's function counters, then read them.
	exec(t, conn, "SELECT pg_stat_force_next_flush()")
	exec(t, conn, "SELECT 1")
	exec(t, conn, "SELECT pg_stat_clear_snapshot()")
	var calls int64
	var selfMs float64
	if err := conn.QueryRow(ctx, "SELECT calls, self_time FROM pg_stat_user_functions WHERE schemaname = 'public' AND funcname = $1", arm.fn).
		Scan(&calls, &selfMs); err != nil {
		t.Fatalf("%s, N = %d: pg_stat_user_functions: %v", arm.name, n, err)
	}
	if calls != int64(n) {
		t.Fatalf("%s, N = %d: %d calls in pg_stat_user_functions, want %d", arm.name, n, calls, n)
	}
	return selfMs * 1000 / float64(calls)
}

// budgetImport is one import run: on a fresh young fixture, a new
// nava_worker session (plan_cache_mode left at the default) writes 10,000
// subjects for a new tenant, absent from the statistics, and their entries
// in one transaction. It returns COMMIT time / 10,000 in µs.
func budgetImport(t *testing.T, super *pgx.Conn, arm budgetArm, k int) float64 {
	t.Helper()
	ctx := context.Background()
	budgetYoung(t, super)
	tenant := newTenant(t, super, "0190e000", k)
	conn, err := pgx.ConnectConfig(ctx, roleConfig(t, "nava_worker", "nava"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ids := "SELECT md5($1::text || ':' || g)::uuid AS id FROM generate_series(1, $2::int) g"
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{"SELECT set_config('app.tenant_id', $1, true)", []any{tenant}},
		{`INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
		  SELECT $1::uuid, gen_random_uuid(), 'record', 'create', 'record', i.id, 1, 'workflow', 'workflow' FROM (` + ids + `) i`,
			[]any{tenant, budgetImportSubjects}},
		{"INSERT INTO " + arm.table + " (tenant_id, id, val) SELECT $1::uuid, i.id, 'v' FROM (" + ids + ") i",
			[]any{tenant, budgetImportSubjects}},
	} {
		if _, err := tx.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatalf("%s: %v", q.sql, err)
		}
	}
	begin := time.Now()
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("%s: commit of %d subjects: %v", arm.name, budgetImportSubjects, err)
	}
	return float64(time.Since(begin).Microseconds()) / budgetImportSubjects
}

func median(xs []float64) float64 {
	s := slices.Clone(xs)
	slices.Sort(s)
	return s[len(s)/2]
}
