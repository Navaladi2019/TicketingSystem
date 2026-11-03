//go:build integration

package dbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
)

// P5-11 (TICK-428): db/audited-tables.yaml, the generated writer grants and
// policies, and the audit_commit_check constraint trigger (ADR-004 decision
// 2, R12; CI-A2, CI-A3). The plan tests (TestP511CommitCheckPlan*) are
// spike S1 finding F1 with the TICK-460, Technical Architect and Finding 5
// additions.

// testTable is the one table this task attaches the trigger to, and only in
// tests: the real migrations attach it to none.
const testTable = "p511_audited"

// execer is a connection or a transaction.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// createTestTable creates testTable on q (the superuser, in the platform
// database): canonical RLS for every writer role of the manifest, INSERT,
// SELECT, DELETE and UPDATE of val (the one audited column) for each, and
// the commit-check trigger as P5-13b and P5-32 will attach it. On a
// connection it is dropped when the test ends; in a transaction the
// caller's rollback removes it.
func createTestTable(t *testing.T, q execer, writers []string) {
	t.Helper()
	roles := strings.Join(writers, ", ")
	pred := "tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid"
	for _, stmt := range []string{
		"CREATE TABLE " + testTable + " (tenant_id uuid NOT NULL, id uuid NOT NULL, val text, note text, PRIMARY KEY (tenant_id, id))",
		"ALTER TABLE " + testTable + " ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE " + testTable + " FORCE ROW LEVEL SECURITY",
		"CREATE POLICY p511_select ON " + testTable + " FOR SELECT TO " + roles + " USING (" + pred + ")",
		"CREATE POLICY p511_insert ON " + testTable + " FOR INSERT TO " + roles + " WITH CHECK (" + pred + ")",
		"CREATE POLICY p511_update ON " + testTable + " FOR UPDATE TO " + roles + " USING (" + pred + ") WITH CHECK (" + pred + ")",
		"CREATE POLICY p511_delete ON " + testTable + " FOR DELETE TO " + roles + " USING (" + pred + ")",
		"GRANT SELECT, INSERT, DELETE, UPDATE (val) ON " + testTable + " TO " + roles,
		"CREATE CONSTRAINT TRIGGER " + testTable + "_audit_commit_check AFTER INSERT OR DELETE OR UPDATE OF val ON " + testTable +
			" DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION audit_commit_check('record', 'id')",
	} {
		if _, err := q.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	if c, ok := q.(*pgx.Conn); ok {
		t.Cleanup(func() { _, _ = c.Exec(context.Background(), "DROP TABLE IF EXISTS "+testTable) })
	}
}

// testTableEntry is the manifest row for testTable.
func testTableEntry(writers []string) audited.Table {
	return audited.Table{WriterRoles: writers, AuditedColumns: []string{"val"}, SubjectType: "record", SubjectIDColumn: "id", Attached: true}
}

// insertEntryAs is insertEntry with a subject type: tenant $1, id $2,
// subject $3, version_after $4, subject_type $5.
const insertEntryAs = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after,
		actor_system, source, diff, diff_salt, diff_sha256)
		VALUES ($1, $2, 'record', 'update', $5, $3, $4, 'workflow', 'workflow',
		        '[{"op":"add","path":"/a","value":1}]', decode('00112233445566778899aabbccddeeff', 'hex'), sha256('d'))`

// CI-A2 mechanism on the test table: for every writer role in the manifest,
// a raw write without an entry fails at COMMIT, naming the trigger; with an
// entry for the same tenant, transaction and subject it commits. An entry
// for another subject or another subject type does not count, and an
// UPDATE that sets no audited column is not checked.
func TestP511CommitCheckTestTable(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX})
	writers := auditedManifest(t).Writers()
	createTestTable(t, super, writers)
	ctx := context.Background()

	for _, role := range writers {
		// nava_onboard reaches audit_log only for its own provisioning
		// tenant (the pinned policies); the others for any tenant they set.
		tenant := tenantA
		if role == audited.OnboardRole {
			tenant = tenantX
		}
		conn := asRole(t, role)
		row := uuid.NewString()
		write := func(what string, stmts ...func(tx pgx.Tx) error) error {
			t.Helper()
			return inTenantTx(conn, tenant, func(tx pgx.Tx) error {
				for _, s := range stmts {
					if err := s(tx); err != nil {
						return fmt.Errorf("%s: %w", what, err)
					}
				}
				return nil
			})
		}
		sql := func(q string, args ...any) func(tx pgx.Tx) error {
			return func(tx pgx.Tx) error { _, err := tx.Exec(ctx, q, args...); return err }
		}
		entry := func(subject, typ string, v int) func(tx pgx.Tx) error {
			return sql(insertEntryAs, tenant, uuid.NewString(), subject, v, typ)
		}
		ins := sql("INSERT INTO "+testTable+" (tenant_id, id, val) VALUES ($1, $2, 'a')", tenant, row)
		upd := sql("UPDATE "+testTable+" SET val = 'b' WHERE id = $1", row)
		del := sql("DELETE FROM "+testTable+" WHERE id = $1", row)

		missing := func(what string, err error) {
			t.Helper()
			code, constraint := pgErrDetail(err)
			if code != "23514" || constraint != testTable+"_audit_commit_check" {
				t.Errorf("%s %s: err = %v (constraint %q), want 23514 from %s_audit_commit_check", role, what, err, constraint, testTable)
			}
		}
		missing("insert without an entry", write("insert", ins))
		missing("insert with another subject's entry", write("insert", ins, entry(uuid.NewString(), "record", 1)))
		missing("insert with another subject type's entry", write("insert", ins, entry(row, "entity_definition", 1)))
		if err := write("insert with its entry", ins, entry(row, "record", 1)); err != nil {
			t.Fatalf("%s: %v", role, err)
		}
		missing("update of an audited column without an entry", write("update", upd))
		if err := write("update with its entry", upd, entry(row, "record", 2)); err != nil {
			t.Errorf("%s: %v", role, err)
		}
		// UPDATE OF val only: a write that sets no audited column is not
		// checked (the superuser here, as no writer may set note).
		if _, err := execIn(super, tenant, "UPDATE "+testTable+" SET note = 'n' WHERE id = $1", row); err != nil {
			t.Errorf("unaudited column update: %v", err)
		}
		missing("delete without an entry", write("delete", del))
		if err := write("delete with its entry", del, entry(row, "record", 3)); err != nil {
			t.Errorf("%s: %v", role, err)
		}
	}
}

// CI-A3: the real manifest passes on a migrated database (the migrator's
// gate runs it too), and each way the catalogue can depart from the
// manifest is reported. Every change is made in a rolled-back transaction.
func TestP511CatalogueCIA3(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	m, g := manifest(t), globalTables(t)
	ctx := context.Background()
	check := func(q pgx.Tx, a *audited.Manifest) []string {
		t.Helper()
		p, err := a.Check(ctx, q, m, g)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	if p, err := auditedManifest(t).Check(ctx, super, m, g); err != nil || len(p) != 0 {
		t.Fatalf("real manifest: problems %v, err %v", p, err)
	}
	// No table has the trigger: this task attaches it to none.
	if n := queryString(t, super, `SELECT count(*)::text FROM pg_trigger t JOIN pg_proc p ON p.oid = t.tgfoid
		WHERE p.proname = 'audit_commit_check'`); n != "0" {
		t.Errorf("%s audit_commit_check triggers after the real migrations, want 0", n)
	}

	writers := auditedManifest(t).Writers()
	withTestTable := func(mutate func(*audited.Table)) *audited.Manifest {
		a := auditedManifest(t)
		e := testTableEntry(writers)
		if mutate != nil {
			mutate(&e)
		}
		a.Audited[testTable] = e
		return a
	}
	const trig = "CREATE CONSTRAINT TRIGGER x_audit_commit_check AFTER INSERT OR DELETE OR UPDATE OF %s ON %s " +
		"%s FOR EACH ROW EXECUTE FUNCTION audit_commit_check(%s)"
	tests := []struct {
		name  string
		setup []string
		a     func() *audited.Manifest
		want  string // "" = no problem
	}{
		{"attached test table passes", nil, func() *audited.Manifest { return withTestTable(nil) }, ""},
		{"trigger on a table not attached",
			[]string{fmt.Sprintf(trig, "data", "records", "DEFERRABLE INITIALLY DEFERRED", "'record', 'id'")},
			func() *audited.Manifest { return withTestTable(nil) }, "but the table is not attached: true"},
		{"attached without a trigger", []string{"DROP TRIGGER " + testTable + "_audit_commit_check ON " + testTable},
			func() *audited.Manifest { return withTestTable(nil) }, "has 0 audit_commit_check triggers"},
		{"UPDATE OF other columns", []string{
			"DROP TRIGGER " + testTable + "_audit_commit_check ON " + testTable,
			fmt.Sprintf(trig, "val, note", testTable, "DEFERRABLE INITIALLY DEFERRED", "'record', 'id'"),
		}, func() *audited.Manifest { return withTestTable(nil) }, "want exactly the audited columns"},
		{"not deferred", []string{
			"DROP TRIGGER " + testTable + "_audit_commit_check ON " + testTable,
			fmt.Sprintf(trig, "val", testTable, "", "'record', 'id'"),
		}, func() *audited.Manifest { return withTestTable(nil) }, "DEFERRABLE INITIALLY DEFERRED"},
		{"wrong arguments", []string{
			"DROP TRIGGER " + testTable + "_audit_commit_check ON " + testTable,
			fmt.Sprintf(trig, "val", testTable, "DEFERRABLE INITIALLY DEFERRED", "'entity_definition', 'id'"),
		}, func() *audited.Manifest { return withTestTable(nil) }, "arguments"},
		{"disabled trigger", []string{"ALTER TABLE " + testTable + " DISABLE TRIGGER " + testTable + "_audit_commit_check"},
			func() *audited.Manifest { return withTestTable(nil) }, "is not enabled"},
		{"tenant table neither audited nor exempt", nil, func() *audited.Manifest { return auditedManifest(t) },
			"tenant table public.p511_audited is neither audited nor exempt"},
		{"stale manifest entry", nil, func() *audited.Manifest {
			a := withTestTable(nil)
			a.Exempt["no_such_table"] = audited.Exemption{Reason: "gone"}
			return a
		}, "lists no_such_table, which is not a tenant table"},
		{"writer_roles drift", []string{"GRANT INSERT ON " + testTable + " TO nava_groupsync"},
			func() *audited.Manifest { return withTestTable(nil) }, "but the catalogue lets"},
		{"updatable column not audited", []string{"GRANT UPDATE (note) ON " + testTable + " TO nava_app"},
			func() *audited.Manifest { return withTestTable(nil) }, "column note may be updated but is not in audited_columns"},
		{"writer cannot read a probe column", []string{"REVOKE SELECT (txid) ON audit_log FROM nava_app"},
			func() *audited.Manifest { return withTestTable(nil) }, "writer role nava_app cannot run the commit check or append: no SELECT (txid)"},
		{"writer has no SELECT policy", []string{"DROP POLICY audit_log_onboard_select ON audit_log"},
			func() *audited.Manifest { return withTestTable(nil) }, "writer role nava_onboard cannot run the commit check or append: no SELECT policy"},
		{"leftover migrator policy (R26.3)", []string{
			"CREATE POLICY p511_migrator ON tenants FOR SELECT TO nava_migrator USING (true)",
		}, func() *audited.Manifest { return withTestTable(nil) }, "policy public.tenants.p511_migrator is left over for nava_migrator"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := super.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			createTestTable(t, tx, writers)
			for _, q := range tc.setup {
				if _, err := tx.Exec(ctx, q); err != nil {
					t.Fatalf("%s: %v", q, err)
				}
			}
			p := check(tx, tc.a())
			joined := strings.Join(p, "\n")
			if tc.want == "" {
				if len(p) != 0 {
					t.Fatalf("problems:\n%s\nwant none", joined)
				}
				return
			}
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("problems:\n%s\nwant one containing %q", joined, tc.want)
			}
		})
	}
}

// The generator converges: a dropped generated policy and a drifted one are
// rebuilt by the next migrate run, a stray audit_log_writer_* policy is
// dropped, and a run after that changes nothing.
func TestP511GeneratedPoliciesConverge(t *testing.T) {
	m := migratedCluster(t)
	super := superNava(t)
	exec(t, super, "DROP POLICY audit_log_writer_insert ON audit_log")
	exec(t, super, "ALTER POLICY audit_log_onboard_select ON audit_log TO nava_onboard, nava_groupsync")
	exec(t, super, "CREATE POLICY audit_log_writer_extra ON audit_log FOR SELECT TO nava_relay USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)")
	res := migrateAll(t, m)
	if got, want := sorted(res.Policies.Created), []string{"audit_log_onboard_select", "audit_log_writer_insert"}; !slices.Equal(got, want) {
		t.Errorf("created %v, want %v", got, want)
	}
	if got, want := sorted(res.Policies.Dropped), []string{"audit_log_onboard_select", "audit_log_writer_extra"}; !slices.Equal(got, want) {
		t.Errorf("dropped %v, want %v", got, want)
	}
	if again := migrateAll(t, m); again.Policies.Changed() || again.Grants.Changed() {
		t.Errorf("third run changed policies %+v, grants %+v", again.Policies, again.Grants)
	}
}

// ---------------------------------------------------------------- plan tests

// plan is the part of an EXPLAIN (FORMAT JSON) the plan tests read.
type plan struct {
	Cost  float64 // the top node's total cost
	Scans []scan  // every scan node, in plan order
}

type scan struct {
	NodeType, Relation, Index, IndexCond string
	Cost                                 float64
}

func (s scan) probesKey() bool {
	return strings.Contains(s.IndexCond, "subject_id") && strings.Contains(s.IndexCond, "occurred_at")
}

// explainProbe runs the probe's EXPLAIN on conn. The probe text is the
// function's PERFORM, with its locals as $1..$5.
func explainProbe(t *testing.T, conn execer, tenant string) plan {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{
		"SET plan_cache_mode = force_generic_plan",
		"DEALLOCATE ALL",
		`PREPARE p511_probe(uuid, timestamptz, xid8, text, uuid) AS
		 SELECT 1 FROM audit_log
		  WHERE tenant_id = $1 AND occurred_at = $2 AND txid = $3 AND subject_type = $4 AND subject_id = $5
		  LIMIT 1`,
	} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	var raw string
	q := fmt.Sprintf("EXPLAIN (FORMAT JSON) EXECUTE p511_probe('%s', now(), pg_current_xact_id(), 'record', '%s')", tenant, uuid.NewString())
	if err := conn.QueryRow(ctx, q).Scan(&raw); err != nil {
		t.Fatalf("explain: %v", err)
	}
	var doc []struct {
		Plan map[string]any `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil || len(doc) != 1 {
		t.Fatalf("explain json %q: %v", raw, err)
	}
	var p plan
	p.Cost, _ = doc[0].Plan["Total Cost"].(float64)
	var walk func(n map[string]any)
	walk = func(n map[string]any) {
		nt, _ := n["Node Type"].(string)
		if strings.Contains(nt, "Scan") {
			s := scan{NodeType: nt}
			s.Relation, _ = n["Relation Name"].(string)
			s.Index, _ = n["Index Name"].(string)
			s.IndexCond, _ = n["Index Cond"].(string)
			s.Cost, _ = n["Total Cost"].(float64)
			p.Scans = append(p.Scans, s)
		}
		kids, _ := n["Plans"].([]any)
		for _, k := range kids {
			if m, ok := k.(map[string]any); ok {
				walk(m)
			}
		}
	}
	walk(doc[0].Plan)
	return p
}

// loadEntries inserts fixture rows into audit_log as the superuser, with
// user triggers and foreign keys off (session_replication_role = replica),
// so occurred_at can lie in the past and every row gets its own txid, as
// many transactions over time would leave them. from is a query of
// (tenant_id uuid, occurred_at timestamptz) rows.
func loadEntries(t *testing.T, q execer, from string) {
	t.Helper()
	for _, stmt := range []string{
		"SET session_replication_role = replica",
		`INSERT INTO audit_log (tenant_id, id, occurred_at, txid, domain, action, subject_type, subject_id, actor_system, source)
		 SELECT f.tenant_id, gen_random_uuid(), f.occurred_at, (1000000 + row_number() OVER ())::text::xid8,
		        'record', 'update', 'record', gen_random_uuid(), 'workflow', 'workflow'
		 FROM (` + from + `) f(tenant_id, occurred_at)`,
		"SET session_replication_role = DEFAULT",
	} {
		if _, err := q.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

// fixtureTenant is the uuid of fixture tenant n, as SQL over an integer
// expression; fixtureTenantID is the same in Go.
func fixtureTenant(n string) string {
	return "('0190b000-0000-7000-8000-' || lpad(to_hex(" + n + "), 12, '0'))::uuid"
}

func fixtureTenantID(n int) string { return fmt.Sprintf("0190b000-0000-7000-8000-%012x", n) }

// lastMonth spreads rows over the month before the current one, so they sit
// in DEFAULT and a range partition for the current month can still be made.
const lastMonth = "date_trunc('month', now()) - interval '1 second' * "

// thisMonth spreads n rows between the start of the current month and now.
func thisMonth(g, n string) string {
	return "date_trunc('month', now()) + (now() - date_trunc('month', now())) * (" + g + "::float8 / (" + n + " + 1))"
}

// youngPartition creates the current month's range partition as Epic 30
// must (n_distinct = 1 on occurred_at, RLS forced), with autovacuum off so
// nothing analyses it behind the test's back.
func youngPartition(t *testing.T, q execer) string {
	t.Helper()
	const name = "audit_log_p511_young"
	for _, stmt := range []string{
		"CREATE TABLE " + name + " PARTITION OF audit_log FOR VALUES FROM (date_trunc('month', now())) " +
			"TO (date_trunc('month', now()) + interval '1 month') WITH (autovacuum_enabled = false)",
		"ALTER TABLE " + name + " ALTER COLUMN occurred_at SET (n_distinct = 1)",
		"ALTER TABLE " + name + " ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE " + name + " FORCE ROW LEVEL SECURITY",
	} {
		if _, err := q.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	return name
}

// keyProbes reports whether p scans something and every scan probes
// subject_id and occurred_at in its Index Cond: the R2 key or the R28
// subject index (the index name is not pinned).
func keyProbes(p plan) bool {
	if len(p.Scans) == 0 {
		return false
	}
	for _, s := range p.Scans {
		if !s.probesKey() {
			return false
		}
	}
	return true
}

func scansRel(p plan, rel string) bool {
	return slices.ContainsFunc(p.Scans, func(s scan) bool { return s.Relation == rel })
}

// requireKeyProbe fails unless keyProbes(p) and a scan is on rel.
func requireKeyProbe(t *testing.T, what string, p plan, rel string) {
	t.Helper()
	if !keyProbes(p) || !scansRel(p, rel) {
		t.Errorf("%s: plan %+v, want every scan to have subject_id and occurred_at in Index Cond, and one on %s", what, p.Scans, rel)
	}
}

// dropKeys removes the R2 key and the R28 subject index inside tx, so the
// probe's next-best path is exposed.
func dropKeys(t *testing.T, tx pgx.Tx) {
	t.Helper()
	for _, stmt := range []string{
		"ALTER TABLE audit_log DROP CONSTRAINT audit_log_commit_key",
		"DROP INDEX audit_log_subject",
	} {
		if _, err := tx.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

// denseFixture is Finding 5's dense case: 50 tenants with 200 rows each,
// last month, so in the DEFAULT partition (>= 100 rows per tenant per
// partition).
const denseFixture = "SELECT " + "('0190b000-0000-7000-8000-' || lpad(to_hex(t), 12, '0'))::uuid, " +
	lastMonth + "(t * 1000 + r) FROM generate_series(1, 50) t, generate_series(1, 200) r"

// TICK-460 case (a), the Technical Architect's cost assertion with its
// negative control, and Finding 5 item 1, on the dense fixture, analysed.
// Through the parent, the generic plan probes subject_id and occurred_at in
// the DEFAULT partition; with the R2 key and the subject index dropped, the
// (tenant_id, occurred_at)-prefix path costs at least 10x more; with the
// n_distinct override RESET, that ratio is under 2x (S1: 214.98 against
// 8.46, and 8.46 against 8.46).
func TestP511CommitCheckPlanDefaultPartition(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	ctx := context.Background()
	loadEntries(t, super, denseFixture)
	exec(t, super, "ANALYZE audit_log")
	tenant := fixtureTenantID(1)

	ratio := func(what string, prep func(tx pgx.Tx)) (key, next plan) {
		t.Helper()
		tx, err := super.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if prep != nil {
			prep(tx)
		}
		key = explainProbe(t, tx, tenant)
		dropKeys(t, tx)
		next = explainProbe(t, tx, tenant)
		t.Logf("%s: key path %.2f %+v; next-best path %.2f %+v; ratio %.1fx", what, key.Cost, key.Scans, next.Cost, next.Scans, next.Cost/key.Cost)
		return key, next
	}

	key, next := ratio("override on", nil)
	requireKeyProbe(t, "DEFAULT partition, override on", key, "audit_log_default")
	if keyProbes(next) {
		t.Errorf("with the R2 key and subject index dropped the plan still probes subject_id: %+v", next.Scans)
	}
	if r := next.Cost / key.Cost; r < 10 {
		t.Errorf("override on: next-best path %.2f, key path %.2f: ratio %.1fx, want >= 10x", next.Cost, key.Cost, r)
	}

	key, next = ratio("override RESET (negative control)", func(tx pgx.Tx) {
		for _, stmt := range []string{
			"ALTER TABLE audit_log ALTER COLUMN occurred_at RESET (n_distinct)",
			"ALTER TABLE audit_log_default ALTER COLUMN occurred_at RESET (n_distinct)",
			"ANALYZE audit_log",
		} {
			if _, err := tx.Exec(ctx, stmt); err != nil {
				t.Fatalf("%s: %v", stmt, err)
			}
		}
	})
	if r := next.Cost / key.Cost; r >= 2 {
		t.Errorf("negative control: with the override RESET the ratio is %.1fx (%.2f / %.2f), want < 2x, or the cost assertion proves nothing",
			r, next.Cost, key.Cost)
	}
}

// TICK-460 case (b): a range partition created and never analysed that
// holds now(), the state at the start of every month. Through the parent,
// pruned on now(), the generic plan's scan of that partition probes
// subject_id and occurred_at.
func TestP511CommitCheckPlanNeverAnalysedPartition(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	loadEntries(t, super, denseFixture)
	exec(t, super, "ANALYZE audit_log")
	young := youngPartition(t, super)
	loadEntries(t, super, "SELECT "+fixtureTenant("1 + g % 10")+", "+thisMonth("g", "1000")+" FROM generate_series(1, 1000) g")
	if got := queryString(t, super, `SELECT c.reltuples::text || ',' || coalesce(s.last_analyze::text, '') || coalesce(s.last_autoanalyze::text, '')
		FROM pg_class c JOIN pg_stat_user_tables s ON s.relid = c.oid WHERE c.relname = $1`, young); got != "-1," {
		t.Fatalf("young partition (reltuples, last analyse) = %q, want never analysed (-1,)", got)
	}
	p := explainProbe(t, super, fixtureTenantID(1))
	t.Logf("never-analysed partition: %.2f %+v", p.Cost, p.Scans)
	requireKeyProbe(t, "never-analysed range partition", p, young)
}

// Finding 5 item 2: on the sparse fixtures, analysed, the generic plan
// probes subject_id and occurred_at after each of 10 fresh ANALYZE samples,
// each read in a new session: (a) 10,000 tenants with 1 row each, (b) a
// young range partition with 60 rows.
func TestP511CommitCheckPlanSparse(t *testing.T) {
	samples := func(t *testing.T, super *pgx.Conn, rel string) {
		t.Helper()
		good := 0
		for i := range 10 {
			exec(t, super, "ANALYZE audit_log")
			cfg := superConfig(t).Copy()
			cfg.Database = "nava"
			conn, err := pgx.ConnectConfig(context.Background(), cfg) // a new session per sample
			if err != nil {
				t.Fatal(err)
			}
			p := explainProbe(t, conn, fixtureTenantID(1))
			_ = conn.Close(context.Background())
			if keyProbes(p) && scansRel(p, rel) {
				good++
			} else {
				t.Logf("sample %d: %.2f %+v", i+1, p.Cost, p.Scans)
			}
		}
		t.Logf("%d of 10 fresh ANALYZE samples probe subject_id and occurred_at", good)
		if good != 10 {
			t.Errorf("%d of 10 fresh ANALYZE samples probe subject_id and occurred_at, want 10 of 10", good)
		}
	}
	t.Run("10000 tenants 1 row each", func(t *testing.T) {
		migratedPlatform(t)
		super := superNava(t)
		loadEntries(t, super, "SELECT "+fixtureTenant("t")+", "+lastMonth+"t FROM generate_series(1, 10000) t")
		samples(t, super, "audit_log_default")
	})
	t.Run("young partition 60 rows", func(t *testing.T) {
		migratedPlatform(t)
		super := superNava(t)
		young := youngPartition(t, super)
		loadEntries(t, super, "SELECT "+fixtureTenant("1 + g % 20")+", "+thisMonth("g", "60")+" FROM generate_series(1, 60) g")
		samples(t, super, young)
	})
}

// importCommit times the COMMIT of one transaction that, as nava_worker
// under plan_cache_mode = force_generic_plan, writes n rows to the test
// table and their n audit entries for tenant: the commit-check triggers all
// run at that COMMIT. tenant must exist and be absent from audit_log's
// statistics.
func importCommit(t *testing.T, tenant string, n int) time.Duration {
	t.Helper()
	ctx := context.Background()
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
		{"SET LOCAL plan_cache_mode = force_generic_plan", nil},
		{"SELECT set_config('app.tenant_id', $1, true)", []any{tenant}},
		{`INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
		  SELECT $1::uuid, gen_random_uuid(), 'record', 'create', 'record', i.id, 1, 'workflow', 'workflow' FROM (` +
			ids + `) i`, []any{tenant, n}},
		{"INSERT INTO " + testTable + " (tenant_id, id, val) SELECT $1::uuid, i.id, 'v' FROM (" +
			ids + ") i", []any{tenant, n}},
	} {
		if _, err := tx.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatalf("%s: %v", q.sql, err)
		}
	}
	start := time.Now()
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit of %d subjects: %v", n, err)
	}
	return time.Since(start)
}

// Finding 5 item 3: on each sparse fixture, the commit check of an import
// into a tenant absent from the statistics is linear: t(4N)/t(N) < 8 for
// N = 1,000 (an O(n^2) check gives about 16). Each timing is the fastest of
// three imports, each into a new tenant.
func TestP511CommitCheckLinearity(t *testing.T) {
	const n = 1000
	run := func(t *testing.T, super *pgx.Conn) {
		t.Helper()
		createTestTable(t, super, auditedManifest(t).Writers())
		exec(t, super, "ANALYZE audit_log")
		next := 0
		best := func(size int) time.Duration {
			var b time.Duration
			for range 3 {
				next++
				tenant := fmt.Sprintf("0190c000-0000-7000-8000-%012x", next)
				exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
				d := importCommit(t, tenant, size)
				if b == 0 || d < b {
					b = d
				}
			}
			return b
		}
		t1, t4 := best(n), best(4*n)
		ratio := float64(t4) / float64(t1)
		t.Logf("commit of %d subjects %v, of %d subjects %v: ratio %.2f", n, t1, 4*n, t4, ratio)
		if ratio >= 8 {
			t.Errorf("t(%d)/t(%d) = %.2f (%v / %v), want < 8: the commit check is not linear", 4*n, n, ratio, t4, t1)
		}
	}
	t.Run("10000 tenants 1 row each", func(t *testing.T) {
		migratedPlatform(t)
		super := superNava(t)
		exec(t, super, "ALTER TABLE audit_log_default SET (autovacuum_enabled = false)")
		loadEntries(t, super, "SELECT "+fixtureTenant("t")+", "+lastMonth+"t FROM generate_series(1, 10000) t")
		run(t, super)
	})
	t.Run("young partition 60 rows", func(t *testing.T) {
		migratedPlatform(t)
		super := superNava(t)
		youngPartition(t, super)
		loadEntries(t, super, "SELECT "+fixtureTenant("1 + g % 20")+", "+thisMonth("g", "60")+" FROM generate_series(1, 60) g")
		run(t, super)
	})
}
