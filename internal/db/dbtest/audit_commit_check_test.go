//go:build integration

package dbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
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
	// Exactly one table has the trigger: records (P5-13c, TICK-530). P5-32
	// attaches it to the entity-definition tables.
	if got := queryString(t, super, `SELECT coalesce(string_agg(t.tgrelid::regclass::text || ':' || t.tgname || ':' || t.tgenabled::text, ','), '')
		FROM pg_trigger t JOIN pg_proc p ON p.oid = t.tgfoid WHERE p.proname = 'audit_commit_check'`); got != "records:records_audit_commit_check:O" {
		t.Errorf("audit_commit_check triggers after the real migrations: %q, want records:records_audit_commit_check:O", got)
	}

	def := commitCheckDef(t, super)
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
			[]string{fmt.Sprintf(trig, "current_version, updated_at", "entity_definitions", "DEFERRABLE INITIALLY DEFERRED", "'entity_definition', 'id'")},
			func() *audited.Manifest { return withTestTable(nil) }, "but the table is not attached: true"},
		{"records attached without its trigger", []string{"DROP TRIGGER records_audit_commit_check ON records"},
			func() *audited.Manifest { return withTestTable(nil) }, "has 0 audit_commit_check triggers"},
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
		// TICK-452 (PR #410 carry-over): the row trigger never sees a
		// TRUNCATE, so a runtime role must not hold it on an audited table.
		{"runtime role holds TRUNCATE", []string{"GRANT TRUNCATE ON " + testTable + " TO nava_worker"},
			func() *audited.Manifest { return withTestTable(nil) }, "runtime role nava_worker holds TRUNCATE"},
		{"runtime role holds TRUNCATE on records", []string{"GRANT TRUNCATE ON records TO nava_app"},
			func() *audited.Manifest { return withTestTable(nil) }, "audited table records: runtime role nava_app holds TRUNCATE"},
		{"updatable column not audited", []string{"GRANT UPDATE (note) ON " + testTable + " TO nava_app"},
			func() *audited.Manifest { return withTestTable(nil) }, "column note may be updated but is not in audited_columns"},
		{"writer cannot read a probe column", []string{"REVOKE SELECT (txid) ON audit_log FROM nava_app"},
			func() *audited.Manifest { return withTestTable(nil) }, "writer role nava_app cannot run the commit check or append: no SELECT (txid)"},
		{"writer has no SELECT policy", []string{"DROP POLICY audit_log_onboard_select ON audit_log"},
			func() *audited.Manifest { return withTestTable(nil) }, "writer role nava_onboard cannot run the commit check or append: no SELECT policy"},
		// ADR-004-E3 decision 3: the clause is in pg_proc.proconfig; a copy
		// without it, an ALTER ... RESET, or a CREATE OR REPLACE without the
		// SET (which resets proconfig) is reported.
		{"CREATE OR REPLACE without the plan_cache_mode clause", []string{withoutClause(t, def, "public.audit_commit_check")},
			func() *audited.Manifest { return withTestTable(nil) }, "function audit_commit_check() has no SET plan_cache_mode=force_custom_plan"},
		{"plan_cache_mode clause RESET", []string{"ALTER FUNCTION audit_commit_check() RESET plan_cache_mode"},
			func() *audited.Manifest { return withTestTable(nil) }, "function audit_commit_check() has no SET plan_cache_mode=force_custom_plan"},
		{"copy without the clause in another schema", []string{"CREATE SCHEMA p511_copy", withoutClause(t, def, "p511_copy.audit_commit_check")},
			func() *audited.Manifest { return withTestTable(nil) }, "function p511_copy.audit_commit_check() has no SET plan_cache_mode=force_custom_plan"},
		// CI-E5-3 (ADR-004-E5 R29): an FK action on an audited table fails
		// CI-A3, on the scratch table and on records alike; NO ACTION and
		// RESTRICT pass (records' own FK is RESTRICT after TICK-538).
		{"R29: ON DELETE SET NULL (col) on an audited table", []string{
			"ALTER TABLE " + testTable + " ADD COLUMN grp uuid, ADD CONSTRAINT p511_grp_fkey FOREIGN KEY (tenant_id, grp) REFERENCES groups (tenant_id, id) ON DELETE SET NULL (grp)",
		}, func() *audited.Manifest { return withTestTable(nil) },
			"audited table p511_audited: foreign key p511_grp_fkey has ON DELETE SET NULL ON UPDATE NO ACTION; R29"},
		{"R29: ON UPDATE CASCADE on an audited table", []string{
			"ALTER TABLE " + testTable + " ADD COLUMN grp uuid, ADD CONSTRAINT p511_grp_fkey FOREIGN KEY (tenant_id, grp) REFERENCES groups (tenant_id, id) ON UPDATE CASCADE",
		}, func() *audited.Manifest { return withTestTable(nil) },
			"audited table p511_audited: foreign key p511_grp_fkey has ON DELETE NO ACTION ON UPDATE CASCADE; R29"},
		{"R29: NO ACTION and RESTRICT pass", []string{
			"ALTER TABLE " + testTable + " ADD COLUMN grp uuid, ADD COLUMN grp2 uuid," +
				" ADD CONSTRAINT p511_grp_fkey FOREIGN KEY (tenant_id, grp) REFERENCES groups (tenant_id, id)," +
				" ADD CONSTRAINT p511_grp2_fkey FOREIGN KEY (tenant_id, grp2) REFERENCES groups (tenant_id, id) ON DELETE RESTRICT ON UPDATE RESTRICT",
		}, func() *audited.Manifest { return withTestTable(nil) }, ""},
		{"R29: records back on ON DELETE SET NULL", []string{
			"ALTER TABLE records DROP CONSTRAINT records_assignment_group_fkey, ADD CONSTRAINT records_assignment_group_fkey" +
				" FOREIGN KEY (tenant_id, assignment_group_id) REFERENCES groups (tenant_id, id) ON DELETE SET NULL (assignment_group_id) NOT VALID",
		}, func() *audited.Manifest { return withTestTable(nil) },
			"audited table records: foreign key records_assignment_group_fkey has ON DELETE SET NULL ON UPDATE NO ACTION; R29"},
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
	Conds                                []string // Index Cond, Recheck Cond and Filter, as present
}

func (s scan) probesKey() bool {
	return strings.Contains(s.IndexCond, "subject_id") && strings.Contains(s.IndexCond, "occurred_at")
}

// explainRole is the role the plan-shape EXPLAIN runs as: a writer of
// records, so its grants and the audit_log SELECT policy (RLS) shape the
// plan exactly as they do in production. The superuser bypasses RLS, and
// its plan would not be the one a writer runs (TICK-428 review, carried to
// TICK-452).
const explainRole = "nava_worker"

// explainProbe runs the probe's EXPLAIN on conn as explainRole, with
// app.tenant_id set to tenant. The probe text is the function's PERFORM,
// with its locals as $1..$5. The role and the setting are reset before it
// returns, so the caller is the superuser again.
func explainProbe(t *testing.T, conn execer, tenant string) plan {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{
		"SET plan_cache_mode = force_generic_plan",
		"DEALLOCATE ALL",
		"SET ROLE " + explainRole,
		"SELECT set_config('app.tenant_id', '" + tenant + "', false)",
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
	err := conn.QueryRow(ctx, q).Scan(&raw)
	var who string
	if err == nil {
		// Simple protocol: DEALLOCATE ALL above would drop a cached statement.
		err = conn.QueryRow(ctx, "SELECT current_user::text", pgx.QueryExecModeSimpleProtocol).Scan(&who)
	}
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	if who != explainRole {
		t.Fatalf("explain ran as %s, want %s", who, explainRole)
	}
	for _, q := range []string{"DEALLOCATE p511_probe", "RESET app.tenant_id", "RESET ROLE"} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	var doc []struct {
		Plan map[string]any `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil || len(doc) != 1 {
		t.Fatalf("explain json %q: %v", raw, err)
	}
	return parsePlan(doc[0].Plan)
}

// parsePlan reads a FORMAT JSON plan tree: EXPLAIN's or auto_explain's.
func parsePlan(top map[string]any) plan {
	var p plan
	p.Cost, _ = top["Total Cost"].(float64)
	var walk func(n map[string]any)
	walk = func(n map[string]any) {
		nt, _ := n["Node Type"].(string)
		if strings.Contains(nt, "Scan") {
			s := scan{NodeType: nt}
			s.Relation, _ = n["Relation Name"].(string)
			s.Index, _ = n["Index Name"].(string)
			s.IndexCond, _ = n["Index Cond"].(string)
			s.Cost, _ = n["Total Cost"].(float64)
			for _, k := range []string{"Index Cond", "Recheck Cond", "Filter"} {
				if c, ok := n[k].(string); ok {
					s.Conds = append(s.Conds, c)
				}
			}
			p.Scans = append(p.Scans, s)
		}
		kids, _ := n["Plans"].([]any)
		for _, k := range kids {
			if m, ok := k.(map[string]any); ok {
				walk(m)
			}
		}
	}
	walk(top)
	return p
}

// paramRef matches a plan parameter ($1): a generic plan's bound value.
var paramRef = regexp.MustCompile(`\$[0-9]+`)

// boundAsConstants reports whether no scan condition of p names a
// parameter: the plan was built for the call's values (a custom plan).
func boundAsConstants(p plan) bool {
	for _, s := range p.Scans {
		for _, c := range s.Conds {
			if paramRef.MatchString(c) {
				return false
			}
		}
	}
	return true
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
	// Item 2 (b'), ADR-004-E3: on the young partition the generic plan is a
	// Seq Scan (0 of 10 on PostgreSQL 16.10), which is why the function is
	// planned per call. So this case reads the plans that really run at
	// COMMIT, through auto_explain, not a PREPAREd generic plan.
	t.Run("young_partition_60_rows", func(t *testing.T) {
		good := 0
		for i := range 10 {
			if commitPlansProbeKey(t, i+1) {
				good++
			}
		}
		t.Logf("%d of 10 fresh ANALYZE samples: every COMMIT-time probe plan of a 1,000-subject import probes subject_id and occurred_at, with constants", good)
		if good != 10 {
			t.Errorf("%d of 10 fresh ANALYZE samples, want 10 of 10", good)
		}
	})
}

// youngFixture is the young-partition fixture on a fresh platform
// database: the current month's partition with 60 rows over 20 fixture
// tenants, the test table with the commit-check trigger, and audit_log
// analysed, so the partition is 2 pages.
func youngFixture(t *testing.T) (*pgx.Conn, string) {
	t.Helper()
	migratedPlatform(t)
	super := superNava(t)
	young := youngPartition(t, super)
	loadEntries(t, super, "SELECT "+fixtureTenant("1 + g % 20")+", "+thisMonth("g", "60")+" FROM generate_series(1, 60) g")
	createTestTable(t, super, auditedManifest(t).Writers())
	exec(t, super, "ANALYZE audit_log")
	if got := queryString(t, super, "SELECT relpages::text FROM pg_class WHERE relname = $1", young); got != "2" {
		t.Fatalf("young partition has %s pages after ANALYZE, want 2", got)
	}
	return super, young
}

// newTenant inserts an active tenant, absent from audit_log's statistics.
func newTenant(t *testing.T, super *pgx.Conn, prefix string, n int) string {
	t.Helper()
	id := fmt.Sprintf("%s-0000-7000-8000-%012x", prefix, n)
	exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", id)
	return id
}

// probeText marks the commit check's probe in auto_explain's Query Text:
// only that statement reads the v_txid local.
const probeText = "txid = v_txid"

// commitPlansProbeKey is one sample of item 2 (b'): on a fresh young
// fixture (a fresh ANALYZE), a nava_worker session with auto_explain
// (log_nested_statements on, log_min_duration 0, notices to the client)
// and plan_cache_mode = force_generic_plan writes 1 subject, then imports
// 1,000 subjects in one transaction. Any plan passes for the 1-subject
// write; every probe plan of the import's COMMIT must have subject_id and
// occurred_at in Index Cond and its bound values as constants. It reports
// whether the sample passed, logging why not.
func commitPlansProbeKey(t *testing.T, sample int) bool {
	t.Helper()
	ctx := context.Background()
	super, young := youngFixture(t)
	var notices []string
	cfg := superConfig(t).Copy()
	cfg.Database = "nava"
	cfg.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) { notices = append(notices, n.Message) }
	conn := connect(t, cfg)
	for _, q := range []string{
		"LOAD 'auto_explain'",
		"SET auto_explain.log_min_duration = 0",
		"SET auto_explain.log_nested_statements = on",
		"SET auto_explain.log_level = notice",
		"SET auto_explain.log_format = json",
		"SET plan_cache_mode = force_generic_plan",
		"SET ROLE nava_worker", // the writer's own grants and policies
	} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	importCommitOn(t, conn, newTenant(t, super, "0190e000", 1), 1, "")
	notices = nil
	importCommitOn(t, conn, newTenant(t, super, "0190e000", 2), 1000, "")
	plans := probePlans(t, notices)
	if len(plans) != 1000 {
		t.Errorf("sample %d: auto_explain logged %d probe plans at the COMMIT of 1,000 subjects, want 1000", sample, len(plans))
		return false
	}
	for j, p := range plans {
		if !keyProbes(p) || !scansRel(p, young) || !boundAsConstants(p) {
			t.Logf("sample %d: probe %d of 1000: %+v, want every scan to probe subject_id and occurred_at, one on %s, and no $n", sample, j+1, p.Scans, young)
			return false
		}
	}
	return true
}

// probePlans parses auto_explain's JSON notices and keeps the probe's.
func probePlans(t *testing.T, notices []string) []plan {
	t.Helper()
	var out []plan
	for _, n := range notices {
		i := strings.Index(n, "{")
		if i < 0 || !strings.Contains(n, probeText) {
			continue
		}
		var doc struct {
			QueryText string         `json:"Query Text"`
			Plan      map[string]any `json:"Plan"`
		}
		if err := json.Unmarshal([]byte(n[i:]), &doc); err != nil {
			t.Fatalf("auto_explain notice %q: %v", n, err)
		}
		if strings.Contains(doc.QueryText, probeText) {
			out = append(out, parsePlan(doc.Plan))
		}
	}
	return out
}

// importCommit times the COMMIT of one transaction that, as nava_worker in
// a new session under plan_cache_mode = force_generic_plan, writes n rows to
// the test table and their n audit entries for tenant: the commit-check
// triggers all run at that COMMIT, so the probe's plan is built then. tenant
// must exist and be absent from audit_log's statistics.
func importCommit(t *testing.T, tenant string, n int) time.Duration {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.ConnectConfig(ctx, roleConfig(t, "nava_worker", "nava"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	return importCommitOn(t, conn, tenant, n, "force_generic_plan")
}

// importCommitOn is importCommit in the nava_worker session conn, which
// keeps whatever plan the commit check cached in it before, with
// plan_cache_mode set to mode for the transaction ("" leaves the session's).
func importCommitOn(t *testing.T, conn *pgx.Conn, tenant string, n int, mode string) time.Duration {
	t.Helper()
	ctx := context.Background()
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
		{"SELECT set_config('plan_cache_mode', coalesce(nullif($1, ''), current_setting('plan_cache_mode')), true)", []any{mode}},
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
// N = 1,000 (an O(n^2) check gives about 16). For each size the timing is
// the median of linearityRuns fresh fixtures, each import into a new tenant,
// as TestP511CommitCheckPooledSession does (TICK-560): one fixture whose
// 1,000-subject baseline lands fast on a shared runner (78 ms against the
// usual 180 ms) no longer decides the ratio on its own.
func TestP511CommitCheckLinearity(t *testing.T) {
	for _, fx := range []struct {
		name string
		load func(t *testing.T, super *pgx.Conn)
	}{
		{"10000 tenants 1 row each", func(t *testing.T, super *pgx.Conn) {
			exec(t, super, "ALTER TABLE audit_log_default SET (autovacuum_enabled = false)")
			loadEntries(t, super, "SELECT "+fixtureTenant("t")+", "+lastMonth+"t FROM generate_series(1, 10000) t")
		}},
		{"young partition 60 rows", func(t *testing.T, super *pgx.Conn) {
			youngPartition(t, super)
			loadEntries(t, super, "SELECT "+fixtureTenant("1 + g % 20")+", "+thisMonth("g", "60")+" FROM generate_series(1, 60) g")
		}},
	} {
		t.Run(fx.name, func(t *testing.T) {
			t1, t4 := linearityMedians(t, fx.load)
			ratio := float64(t4) / float64(t1)
			if ratio >= 8 {
				t.Errorf("median t(%d)/median t(%d) = %.2f (%v / %v), want < 8: the commit check is not linear", 4*linearityN, linearityN, ratio, t4, t1)
			}
		})
	}
}

// linearityN and linearityRuns are item 3's N and its number of fresh
// fixtures per size.
const linearityN, linearityRuns = 1000, 3

// linearityMedians builds linearityRuns fresh fixtures with load; on each it
// times the COMMIT of an import of N and then of 4N subjects, each into a new
// tenant absent from the statistics. It returns the median t(N) and the
// median t(4N), logging every sample.
func linearityMedians(t *testing.T, load func(t *testing.T, super *pgx.Conn)) (time.Duration, time.Duration) {
	t.Helper()
	t1s := make([]time.Duration, 0, linearityRuns)
	t4s := make([]time.Duration, 0, linearityRuns)
	for r := range linearityRuns {
		migratedPlatform(t)
		super := superNava(t)
		load(t, super)
		createTestTable(t, super, auditedManifest(t).Writers())
		exec(t, super, "ANALYZE audit_log")
		t1 := importCommit(t, newTenant(t, super, "0190c000", 1), linearityN)
		t4 := importCommit(t, newTenant(t, super, "0190c000", 2), 4*linearityN)
		t.Logf("fixture %d: commit of %d subjects %v, of %d subjects %v: ratio %.2f", r+1, linearityN, t1, 4*linearityN, t4, float64(t4)/float64(t1))
		t1s, t4s = append(t1s, t1), append(t4s, t4)
	}
	slices.Sort(t1s)
	slices.Sort(t4s)
	t1, t4 := t1s[linearityRuns/2], t4s[linearityRuns/2]
	t.Logf("median over %d fresh fixtures: t(%d) %v, t(%d) %v: ratio %.2f", linearityRuns, linearityN, t1, 4*linearityN, t4, float64(t4)/float64(t1))
	return t1, t4
}

// Finding 5 item 3 (c), ADR-004-E3: the pooled-session case. In each
// session mode (auto, force_generic_plan) one nava_worker session runs 10
// warm-up 1-subject writes while the young partition is 2 pages, so a
// generic plan cached then would be a Seq Scan, then imports N = 1,000 and
// N = 4,000 subjects, each into a tenant absent from the statistics, with no
// ANALYZE in between. With the clause the median t(4N)/t(N) of 3 runs on
// fresh fixtures is < 8 in both modes (measured 3.1 to 5.4). The negative
// control runs the same script against a copy of the function without the
// clause; its median must be >= 8 (measured 12.3 to 20.0), or this test can
// no longer catch the O(n^2) path.
func TestP511CommitCheckPooledSession(t *testing.T) {
	const n, runs = 1000, 3
	for _, mode := range []string{"auto", "force_generic_plan"} {
		t.Run(mode, func(t *testing.T) {
			for _, arm := range []struct {
				name     string
				noClause bool
			}{{"with_clause", false}, {"without_clause_negative_control", true}} {
				t.Run(arm.name, func(t *testing.T) {
					ratios := make([]float64, 0, runs)
					for r := range runs {
						ratios = append(ratios, pooledRatio(t, mode, n, arm.noClause, r+1))
					}
					sort.Float64s(ratios)
					median := ratios[runs/2]
					t.Logf("%s, %s: t(%d)/t(%d) over %d fresh fixtures %.2f, median %.2f", mode, arm.name, 4*n, n, runs, ratios, median)
					if !arm.noClause && median >= 8 {
						t.Errorf("median t(%d)/t(%d) = %.2f, want < 8: the commit check is not linear in a pooled session", 4*n, n, median)
					}
					if arm.noClause && median < 8 {
						t.Errorf("negative control: without the clause the median t(%d)/t(%d) = %.2f, want >= 8, or this test cannot catch the O(n^2) path", 4*n, n, median)
					}
				})
			}
		})
	}
}

// pooledRatio is one run of item 3 (c) on a fresh young fixture: it returns
// t(4n)/t(n) in one nava_worker session under plan_cache_mode = mode. With
// noClause the test table's trigger calls a copy of audit_commit_check
// without SET plan_cache_mode.
func pooledRatio(t *testing.T, mode string, n int, noClause bool, run int) float64 {
	t.Helper()
	super, _ := youngFixture(t)
	if noClause {
		exec(t, super, withoutClause(t, commitCheckDef(t, super), noClauseCopy))
		exec(t, super, "DROP TRIGGER "+testTable+"_audit_commit_check ON "+testTable)
		exec(t, super, "CREATE CONSTRAINT TRIGGER "+testTable+"_audit_commit_check AFTER INSERT OR DELETE OR UPDATE OF val ON "+
			testTable+" DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION "+noClauseCopy+"('record', 'id')")
	}
	conn := connect(t, roleConfig(t, "nava_worker", "nava"))
	exec(t, conn, "SELECT set_config('plan_cache_mode', $1, false)", mode)
	next := 0
	tenant := func() string { next++; return newTenant(t, super, "0190d000", next) }
	for range 10 { // warm-up while the partition is 2 pages
		importCommitOn(t, conn, tenant(), 1, "")
	}
	t1 := importCommitOn(t, conn, tenant(), n, "")
	t4 := importCommitOn(t, conn, tenant(), 4*n, "")
	ratio := float64(t4) / float64(t1)
	t.Logf("run %d: commit of %d subjects %v, of %d subjects %v: ratio %.2f", run, n, t1, 4*n, t4, ratio)
	return ratio
}

// noClauseCopy names the negative controls' copy of the function.
const noClauseCopy = "p511_commit_check_no_clause"

// commitCheckDef is pg_get_functiondef of public.audit_commit_check().
func commitCheckDef(t *testing.T, q execer) string {
	t.Helper()
	var def string
	if err := q.QueryRow(context.Background(), "SELECT pg_get_functiondef('public.audit_commit_check()'::regprocedure)").Scan(&def); err != nil {
		t.Fatal(err)
	}
	return def
}

// planCacheLine is the clause's line in pg_get_functiondef.
var planCacheLine = regexp.MustCompile(`(?m)^ SET plan_cache_mode TO '?force_custom_plan'?\n`)

// withoutClause turns def into a CREATE OR REPLACE of function name with
// the same body and no SET plan_cache_mode; it fails if def has no clause.
func withoutClause(t *testing.T, def, name string) string {
	t.Helper()
	if !planCacheLine.MatchString(def) {
		t.Fatalf("audit_commit_check has no plan_cache_mode clause:\n%s", def)
	}
	const head = "FUNCTION public.audit_commit_check()"
	if !strings.Contains(def, head) {
		t.Fatalf("pg_get_functiondef does not name %s:\n%s", head, def)
	}
	return strings.Replace(planCacheLine.ReplaceAllString(def, ""), head, "FUNCTION "+name+"()", 1)
}

// probeQuery is the commit check's probe as def holds it: from PERFORM to
// the next semicolon.
func probeQuery(def string) string {
	i := strings.Index(def, "PERFORM")
	if i < 0 {
		return ""
	}
	if j := strings.Index(def[i:], ";"); j >= 0 {
		return def[i : i+j]
	}
	return def[i:]
}

// probeProblems is ADR-004-E3 decision 2 on a function definition: the
// probe exists and binds plpgsql locals only, never a NEW or OLD column.
func probeProblems(def string) []string {
	q := probeQuery(def)
	if q == "" {
		return []string{"no PERFORM probe"}
	}
	var out []string
	for _, rec := range []string{"NEW.", "OLD."} {
		if strings.Contains(strings.ToUpper(q), rec) {
			out = append(out, "probe references "+rec+"<col>: "+q)
		}
	}
	return out
}

// ADR-004-E3 decision 2: the deployed probe binds plpgsql locals only. The
// negative control is a copy whose probe reads NEW.id, which must be
// reported.
func TestP511CommitCheckProbeLocalsOnly(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	ctx := context.Background()
	def := commitCheckDef(t, super)
	if p := probeProblems(def); len(p) != 0 {
		t.Errorf("audit_commit_check: %v\n%s", p, def)
	}
	for _, local := range []string{"v_tenant", "v_at", "v_txid", "v_type", "v_id"} {
		if !strings.Contains(probeQuery(def), local) {
			t.Errorf("probe does not bind the local %s: %s", local, probeQuery(def))
		}
	}

	tx, err := super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	bad := strings.Replace(withoutClause(t, def, noClauseCopy), "subject_id = v_id", "subject_id = NEW.id", 1)
	if bad == withoutClause(t, def, noClauseCopy) {
		t.Fatalf("negative control: no \"subject_id = v_id\" in the probe to replace:\n%s", def)
	}
	if _, err := tx.Exec(ctx, bad); err != nil {
		t.Fatalf("create the NEW. copy: %v", err)
	}
	var badDef string
	if err := tx.QueryRow(ctx, "SELECT pg_get_functiondef($1::regprocedure)", noClauseCopy+"()").Scan(&badDef); err != nil {
		t.Fatal(err)
	}
	if p := probeProblems(badDef); len(p) == 0 {
		t.Errorf("negative control: a probe that reads NEW.id is not reported:\n%s", badDef)
	}
}
