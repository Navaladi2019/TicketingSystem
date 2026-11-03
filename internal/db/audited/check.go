package audited

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// Check is CI-A3 (ADR-004; R26.3). Against the database q is connected to,
// it reports, sorted and each prefixed "CI-A3: ":
//
//   - a tenant table (a partition counts as its parent) that is neither
//     audited nor exempt, and a manifest entry that is not a tenant table;
//   - an audited table whose writer_roles differ from the roles the
//     catalogue lets INSERT, UPDATE or DELETE it, a column some role may
//     UPDATE that audited_columns leaves out, or a missing audited, subject
//     or tenant column;
//   - an audit_commit_check trigger on a table not marked attached: true, an
//     attached table without exactly one, and one that is not an enabled
//     AFTER INSERT OR DELETE [OR UPDATE OF exactly audited_columns]
//     DEFERRABLE INITIALLY DEFERRED row constraint trigger called with
//     (subject_type, subject_id_column);
//   - an audit_commit_check function whose pg_proc.proconfig lacks
//     plan_cache_mode=force_custom_plan (ADR-004-E3 decision 3; a CREATE OR
//     REPLACE FUNCTION without the SET resets proconfig), or no such
//     function at all;
//   - a writer role that cannot run the check or append: SELECT on a probe
//     column, INSERT on an R10 column, or an audit_log SELECT or INSERT
//     policy for it, missing;
//   - any policy left over for the owner (nava_migrator), R26.3.
//
// The policies' predicates are the catalogue gate's business
// (catalog.PolicyProblems), which pins the generated ones too. An error
// means the check could not run.
func (a *Manifest) Check(ctx context.Context, q catalog.Querier, m *grants.Manifest, g *catalog.GlobalTables) ([]string, error) {
	var problems []string
	bad := func(format string, args ...any) { problems = append(problems, "CI-A3: "+fmt.Sprintf(format, args...)) }

	tables, err := catalog.TenantTables(ctx, q, g)
	if err != nil {
		return nil, err
	}
	partitions, err := strs(ctx, q, `SELECT n.nspname || '.' || c.relname FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace WHERE c.relispartition`)
	if err != nil {
		return nil, err
	}
	listed := map[string]string{} // qualified name -> manifest key
	for name := range a.Audited {
		listed[qualify(name)] = name
	}
	for name := range a.Exempt {
		listed[qualify(name)] = name
	}
	tenant := map[string]bool{}
	for _, t := range tables {
		full := t.Schema + "." + t.Name
		if slices.Contains(partitions, full) {
			continue
		}
		tenant[full] = true
		if _, ok := listed[full]; !ok {
			bad("tenant table %s is neither audited nor exempt in db/audited-tables.yaml", full)
		}
	}
	for full, name := range listed {
		if !tenant[full] {
			bad("db/audited-tables.yaml lists %s, which is not a tenant table", name)
		}
	}

	runtime := []string{}
	for _, r := range m.RoleNames() {
		if r != m.Owner() {
			runtime = append(runtime, r)
		}
	}
	for _, name := range a.TableNames() {
		p, err := a.checkTable(ctx, q, name, runtime)
		if err != nil {
			return nil, err
		}
		for _, s := range p {
			bad("%s", s)
		}
	}
	p, err := a.checkTriggers(ctx, q)
	if err != nil {
		return nil, err
	}
	for _, s := range p {
		bad("%s", s)
	}
	p, err = checkFunctionConfig(ctx, q)
	if err != nil {
		return nil, err
	}
	for _, s := range p {
		bad("%s", s)
	}
	p, err = a.checkWriters(ctx, q)
	if err != nil {
		return nil, err
	}
	for _, s := range p {
		bad("%s", s)
	}

	left, err := strs(ctx, q, `SELECT schemaname || '.' || tablename || '.' || policyname FROM pg_catalog.pg_policies
		WHERE $1 = ANY (roles) AND schemaname !~ '^pg_' AND schemaname <> 'information_schema'`, m.Owner())
	if err != nil {
		return nil, err
	}
	for _, l := range left {
		bad("policy %s is left over for %s (R26.3)", l, m.Owner())
	}
	slices.Sort(problems)
	return slices.Compact(problems), nil
}

func (a *Manifest) checkTable(ctx context.Context, q catalog.Querier, name string, runtime []string) ([]string, error) {
	t := a.Audited[name]
	full := qualify(name)
	var out []string
	cols := map[string]string{} // column -> type
	rows, err := q.Query(ctx, `SELECT a.attname::text, pg_catalog.format_type(a.atttypid, a.atttypmod)
		FROM pg_catalog.pg_attribute a WHERE a.attrelid = pg_catalog.to_regclass($1) AND a.attnum > 0 AND NOT a.attisdropped`, full)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c, typ string
		if err := rows.Scan(&c, &typ); err != nil {
			return nil, err
		}
		cols[c] = typ
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return []string{fmt.Sprintf("audited table %s does not exist", name)}, nil
	}
	if cols["tenant_id"] != "uuid" {
		out = append(out, fmt.Sprintf("audited table %s has no uuid tenant_id", name))
	}
	if cols[t.SubjectIDColumn] != "uuid" {
		out = append(out, fmt.Sprintf("audited table %s: subject_id_column %s is not a uuid column", name, t.SubjectIDColumn))
	}
	for _, c := range t.AuditedColumns {
		if _, ok := cols[c]; !ok {
			out = append(out, fmt.Sprintf("audited table %s: audited column %s does not exist", name, c))
		}
	}
	writers, err := strs(ctx, q, `SELECT r FROM unnest($2::text[]) r
		WHERE pg_catalog.has_any_column_privilege(r, $1, 'INSERT')
		   OR pg_catalog.has_any_column_privilege(r, $1, 'UPDATE')
		   OR pg_catalog.has_table_privilege(r, $1, 'DELETE')
		ORDER BY 1`, full, runtime)
	if err != nil {
		return nil, err
	}
	declared := slices.Sorted(slices.Values(t.WriterRoles))
	if !slices.Equal(writers, declared) {
		out = append(out, fmt.Sprintf("audited table %s: writer_roles %v, but the catalogue lets %v write it", name, declared, writers))
	}
	updatable, err := strs(ctx, q, `SELECT DISTINCT a.attname::text
		FROM pg_catalog.pg_attribute a CROSS JOIN unnest($2::text[]) r
		WHERE a.attrelid = pg_catalog.to_regclass($1) AND a.attnum > 0 AND NOT a.attisdropped
		  AND pg_catalog.has_column_privilege(r, a.attrelid, a.attnum, 'UPDATE')
		ORDER BY 1`, full, runtime)
	if err != nil {
		return nil, err
	}
	for _, c := range updatable {
		if !slices.Contains(t.AuditedColumns, c) {
			out = append(out, fmt.Sprintf("audited table %s: column %s may be updated but is not in audited_columns", name, c))
		}
	}
	return out, nil
}

// trigger is one audit_commit_check trigger as the catalogue holds it.
type trigger struct {
	table, name              string
	tgtype                   int16
	deferrable, initDeferred bool
	constraint               bool
	enabled                  string
	columns, args            []string
}

// pg_trigger.tgtype bits (src/include/catalog/pg_trigger.h).
const (
	tgRow      = 1 << 0
	tgBefore   = 1 << 1
	tgInsert   = 1 << 2
	tgDelete   = 1 << 3
	tgUpdate   = 1 << 4
	tgTruncate = 1 << 5
	tgInstead  = 1 << 6
)

func (a *Manifest) checkTriggers(ctx context.Context, q catalog.Querier) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT n.nspname || '.' || c.relname, t.tgname::text, t.tgtype, t.tgdeferrable, t.tginitdeferred,
		       t.tgconstraint <> 0, t.tgenabled::text,
		       ARRAY(SELECT a.attname::text FROM unnest(t.tgattr::int2[]) u(attnum)
		             JOIN pg_catalog.pg_attribute a ON a.attrelid = t.tgrelid AND a.attnum = u.attnum ORDER BY 1),
		       pg_catalog.string_to_array(pg_catalog.encode(t.tgargs, 'escape'), '\000')
		FROM pg_catalog.pg_trigger t
		JOIN pg_catalog.pg_proc p ON p.oid = t.tgfoid
		JOIN pg_catalog.pg_class c ON c.oid = t.tgrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE p.proname = $1 AND NOT t.tgisinternal
		ORDER BY 1, 2`, TriggerFunction)
	if err != nil {
		return nil, err
	}
	trigs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (trigger, error) {
		var tr trigger
		err := r.Scan(&tr.table, &tr.name, &tr.tgtype, &tr.deferrable, &tr.initDeferred, &tr.constraint, &tr.enabled, &tr.columns, &tr.args)
		// tgargs ends with a NUL, so the split ends with an empty element.
		if n := len(tr.args); n > 0 && tr.args[n-1] == "" {
			tr.args = tr.args[:n-1]
		}
		return tr, err
	})
	if err != nil {
		return nil, err
	}
	byTable := map[string][]trigger{}
	for _, tr := range trigs {
		byTable[tr.table] = append(byTable[tr.table], tr)
	}
	var out []string
	attached := map[string]bool{}
	for _, name := range a.TableNames() {
		t := a.Audited[name]
		full := qualify(name)
		if !t.Attached {
			continue
		}
		attached[full] = true
		got := byTable[full]
		if len(got) != 1 {
			out = append(out, fmt.Sprintf("%s is attached: true but has %d %s triggers, want 1", name, len(got), TriggerFunction))
			continue
		}
		out = append(out, checkTrigger(name, t, got[0])...)
	}
	for table, ts := range byTable {
		if attached[table] {
			continue
		}
		for _, tr := range ts {
			out = append(out, fmt.Sprintf("trigger %s on %s calls %s, but the table is not attached: true in db/audited-tables.yaml", tr.name, table, TriggerFunction))
		}
	}
	return out, nil
}

func checkTrigger(name string, t Table, tr trigger) []string {
	var out []string
	bad := func(format string, args ...any) {
		out = append(out, fmt.Sprintf("trigger %s on %s: ", tr.name, name)+fmt.Sprintf(format, args...))
	}
	want := int16(tgRow | tgInsert | tgDelete)
	if len(t.AuditedColumns) > 0 {
		want |= tgUpdate
	}
	if tr.tgtype != want {
		bad("events %s, want %s", events(tr.tgtype), events(want))
	}
	if !tr.constraint || !tr.deferrable || !tr.initDeferred {
		bad("must be a CONSTRAINT TRIGGER, DEFERRABLE INITIALLY DEFERRED")
	}
	if tr.enabled != "O" && tr.enabled != "A" {
		bad("is not enabled (tgenabled %q)", tr.enabled)
	}
	cols := slices.Sorted(slices.Values(t.AuditedColumns))
	if !slices.Equal(tr.columns, cols) {
		bad("UPDATE OF %v, want exactly the audited columns %v", tr.columns, cols)
	}
	if wantArgs := []string{t.SubjectType, t.SubjectIDColumn}; !slices.Equal(tr.args, wantArgs) {
		bad("arguments %q, want %q", tr.args, wantArgs)
	}
	return out
}

func events(tgtype int16) string {
	var e []string
	if tgtype&tgBefore != 0 {
		e = append(e, "BEFORE")
	} else if tgtype&tgInstead != 0 {
		e = append(e, "INSTEAD OF")
	} else {
		e = append(e, "AFTER")
	}
	for _, b := range []struct {
		bit  int16
		name string
	}{{tgInsert, "INSERT"}, {tgDelete, "DELETE"}, {tgUpdate, "UPDATE"}, {tgTruncate, "TRUNCATE"}} {
		if tgtype&b.bit != 0 {
			e = append(e, b.name)
		}
	}
	if tgtype&tgRow != 0 {
		e = append(e, "FOR EACH ROW")
	} else {
		e = append(e, "FOR EACH STATEMENT")
	}
	return strings.Join(e, " ")
}

// PlanCacheSetting is the proconfig entry every audit_commit_check function
// must carry (ADR-004-E3 decision 1): plpgsql then plans the probe on every
// call, with its locals as constants, instead of reusing a generic plan
// cached while the partition was young.
const PlanCacheSetting = "plan_cache_mode=force_custom_plan"

// checkFunctionConfig reports every function named TriggerFunction, in any
// schema, whose proconfig lacks PlanCacheSetting, and reports it missing
// when there is none: checkTriggers matches triggers by function name, so a
// copy in another schema could carry the trigger and must carry the clause
// too.
func checkFunctionConfig(ctx context.Context, q catalog.Querier) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT p.oid::regprocedure::text, NOT (coalesce(p.proconfig, '{}') @> ARRAY[$2::text])
		FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		WHERE p.proname = $1 AND n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
		ORDER BY 1`, TriggerFunction, PlanCacheSetting)
	if err != nil {
		return nil, err
	}
	type fn struct {
		name    string
		missing bool
	}
	fns, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (fn, error) {
		var f fn
		return f, r.Scan(&f.name, &f.missing)
	})
	if err != nil {
		return nil, err
	}
	if len(fns) == 0 {
		return []string{fmt.Sprintf("function %s() does not exist", TriggerFunction)}, nil
	}
	var out []string
	for _, f := range fns {
		if f.missing {
			out = append(out, fmt.Sprintf("function %s has no SET %s in pg_proc.proconfig (ADR-004-E3)", f.name, PlanCacheSetting))
		}
	}
	return out, nil
}

func (a *Manifest) checkWriters(ctx context.Context, q catalog.Querier) ([]string, error) {
	var out []string
	log := "public." + LogTable
	for _, w := range a.Writers() {
		missing, err := strs(ctx, q, `
			SELECT 'SELECT (' || c || ')' FROM unnest($3::text[]) c WHERE NOT pg_catalog.has_column_privilege($1, $2, c, 'SELECT')
			UNION ALL
			SELECT 'INSERT (' || c || ')' FROM unnest($4::text[]) c WHERE NOT pg_catalog.has_column_privilege($1, $2, c, 'INSERT')
			UNION ALL
			SELECT want_cmd || ' policy' FROM unnest(ARRAY['SELECT', 'INSERT']) want_cmd
			WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_policies p
			                  WHERE p.schemaname = 'public' AND p.tablename = $5 AND p.cmd = want_cmd AND $1 = ANY (p.roles))
			ORDER BY 1`, w, log, ProbeColumns(), InsertColumns(), LogTable)
		if err != nil {
			return nil, err
		}
		for _, m := range missing {
			out = append(out, fmt.Sprintf("writer role %s cannot run the commit check or append: no %s on %s", w, m, log))
		}
	}
	return out, nil
}

func qualify(name string) string {
	s, t, err := grants.SplitTable(name)
	if err != nil {
		return name
	}
	return s + "." + t
}

func strs(ctx context.Context, q catalog.Querier, sql string, args ...any) ([]string, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
