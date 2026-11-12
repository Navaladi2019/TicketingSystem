package db

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

// TICK-715 (docs/data/fk-validate-force-rls.md): under FORCE ROW LEVEL
// SECURITY the table owner's foreign-key check is vacuous. When nava_migrator
// owns both tables, PostgreSQL checks an FK with one anti-join query
// (RI_Initial_Check) that runs under both tables' RLS; nava_migrator has no
// policy, so the query sees no row and the key is marked valid over any
// orphan. That covers ALTER TABLE ... VALIDATE CONSTRAINT and an FK added
// without NOT VALID to a table that already holds rows. A CHECK is validated
// by a heap scan and is not affected. An FK added in the ALTER TABLE that
// also adds one of its columns with no default is exempt: that column is NULL
// in every existing row, and MATCH SIMPLE checks no row with a NULL key
// column (20261101190000, 20261108184658).
//
// This guard runs in `task test` with no database. Every such statement on a
// FORCE-RLS table must run, in a transactional migration, while a
// "FOR SELECT TO nava_migrator USING (true)" policy exists on each FORCE-RLS
// table of the pair, and every such policy must be dropped in the same file
// (the P3-40 pattern, 20261111123340 on the TICK-337 story branch).

// migratorRole owns every platform table (db/grants.yaml owner).
const migratorRole = "nava_migrator"

// fkValidateAllowed are the statements that broke the rule before the guard
// existed. Applied migrations are never edited; each entry names why it is
// safe or the migration that re-checks it for real. An entry that no longer
// matches a finding fails the test.
func fkValidateAllowed() map[string]string {
	return map[string]string{
		"20261104013500_groups_tombstone_validate.sql group_members_group_live_fkey": "re-checked by 20261112100000_group_members_group_live_fkey_recheck.sql",
		"20261104043100_groups_contract_validate.sql records_assignment_group_fkey":  "re-checked by 20261112100100_records_assignment_group_fkey_recheck.sql",
	}
}

func TestMigrationsFKValidateUnderForceRLS(t *testing.T) {
	findings, err := checkFKValidateForceRLS(Migrations())
	if err != nil {
		t.Fatal(err)
	}
	allowed := fkValidateAllowed()
	var problems []string
	seen := map[string]bool{}
	for _, f := range findings {
		if _, ok := allowed[f.key()]; ok {
			seen[f.key()] = true
			continue
		}
		problems = append(problems, f.String())
	}
	for key, why := range allowed {
		if !seen[key] {
			problems = append(problems, fmt.Sprintf("fkValidateAllowed entry %q matches no finding: remove it", key))
		}
		if file, ok := strings.CutPrefix(why, "re-checked by "); ok {
			if _, err := fs.Stat(Migrations(), file); err != nil {
				problems = append(problems, fmt.Sprintf("fkValidateAllowed entry %q: re-check migration %s: %v", key, file, err))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		t.Fatalf("foreign keys checked under FORCE RLS without the migrator-policy pattern "+
			"(docs/data/fk-validate-force-rls.md):\n%s", strings.Join(problems, "\n"))
	}
}

// The acceptance fixture: a bare VALIDATE of an FK between FORCE-RLS tables.
func TestFKValidateGuardCatchesBareValidateFixture(t *testing.T) {
	findings, err := checkFKValidateForceRLS(os.DirFS("testdata/fk_validate_force_rls"))
	if err != nil {
		t.Fatal(err)
	}
	got := findingStrings(findings)
	want := []string{
		"00002_validate.sql: child VALIDATE CONSTRAINT child_parent_fkey: no SELECT TO nava_migrator USING (true) policy on child, parent in effect",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("findings =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestFKValidateGuardCases(t *testing.T) {
	const tables = `-- +goose Up
CREATE TABLE parent (id uuid PRIMARY KEY);
CREATE TABLE child (id uuid PRIMARY KEY, parent_id uuid);
ALTER TABLE parent ENABLE ROW LEVEL SECURITY;
ALTER TABLE parent FORCE ROW LEVEL SECURITY;
ALTER TABLE child ENABLE ROW LEVEL SECURITY;
ALTER TABLE child FORCE ROW LEVEL SECURITY;
ALTER TABLE child ADD CONSTRAINT child_parent_fkey FOREIGN KEY (parent_id) REFERENCES parent (id) NOT VALID,
    ADD CONSTRAINT child_id_check CHECK (id IS NOT NULL) NOT VALID;
`
	const policies = `CREATE POLICY child_migrator_validate ON child FOR SELECT TO nava_migrator USING (true);
CREATE POLICY parent_migrator_validate ON parent FOR SELECT TO nava_migrator USING (true);
`
	const drops = `DROP POLICY parent_migrator_validate ON parent;
DROP POLICY child_migrator_validate ON child;
`
	const validate = "ALTER TABLE child VALIDATE CONSTRAINT child_parent_fkey;\n"
	cases := []struct {
		name   string
		first  string // replaces tables when set
		second string
		want   []string
	}{
		{name: "pattern", second: "-- +goose Up\nSET LOCAL lock_timeout = '5s';\n" + policies + validate + drops},
		{name: "pattern with drop and re-add", second: "-- +goose Up\n" + policies + `ALTER TABLE child
    DROP CONSTRAINT child_parent_fkey,
    ADD CONSTRAINT child_parent_fkey FOREIGN KEY (parent_id) REFERENCES parent (id) NOT VALID;
` + validate + drops},
		{name: "check constraint", second: "-- +goose Up\nALTER TABLE child VALIDATE CONSTRAINT child_id_check;\n"},
		{name: "commented out", second: "-- +goose Up\n-- ALTER TABLE child VALIDATE CONSTRAINT child_parent_fkey;\nSELECT 1;\n"},
		{name: "not forced", first: strings.ReplaceAll(tables, "ALTER TABLE child FORCE ROW LEVEL SECURITY;\n", "") +
			"ALTER TABLE parent NO FORCE ROW LEVEL SECURITY;\n", second: "-- +goose Up\n" + validate},
		{name: "policy on the referencing table only", second: "-- +goose Up\n" +
			"CREATE POLICY child_migrator_validate ON child FOR SELECT TO nava_migrator USING (true);\n" + validate +
			"DROP POLICY child_migrator_validate ON child;\n",
			want: []string{"00002_x.sql: child VALIDATE CONSTRAINT child_parent_fkey: no SELECT TO nava_migrator USING (true) policy on parent in effect"}},
		{name: "policy dropped before the validate", second: "-- +goose Up\n" + policies + drops + validate,
			want: []string{"00002_x.sql: child VALIDATE CONSTRAINT child_parent_fkey: no SELECT TO nava_migrator USING (true) policy on child, parent in effect"}},
		{name: "policy for another role", second: "-- +goose Up\n" + strings.ReplaceAll(policies, "nava_migrator", "nava_app") + validate +
			strings.ReplaceAll(drops, "nava_migrator", "nava_app"),
			want: []string{"00002_x.sql: child VALIDATE CONSTRAINT child_parent_fkey: no SELECT TO nava_migrator USING (true) policy on child, parent in effect"}},
		{name: "policy outlives the migration", second: "-- +goose Up\n" + policies + validate,
			want: []string{
				"00002_x.sql: policy child_migrator_validate on child is not dropped in the same migration",
				"00002_x.sql: policy parent_migrator_validate on parent is not dropped in the same migration",
			}},
		{name: "no transaction", second: "-- +goose NO TRANSACTION\n-- +goose Up\n" + policies + validate + drops,
			want: []string{"00002_x.sql: -- +goose NO TRANSACTION: the migrator policies would be visible to every session"}},
		{name: "validated add on an existing table", second: "-- +goose Up\nALTER TABLE child DROP CONSTRAINT child_parent_fkey;\n" +
			"ALTER TABLE child ADD CONSTRAINT child_parent_fkey FOREIGN KEY (parent_id) REFERENCES parent (id);\n",
			want: []string{"00002_x.sql: child ADD CONSTRAINT child_parent_fkey FOREIGN KEY without NOT VALID: no SELECT TO nava_migrator USING (true) policy on child, parent in effect"}},
		{name: "validated add with a new column", second: "-- +goose Up\nALTER TABLE child DROP CONSTRAINT child_parent_fkey;\n" +
			"ALTER TABLE child ADD COLUMN other_id uuid,\n    ADD CONSTRAINT child_parent_fkey FOREIGN KEY (other_id, parent_id) REFERENCES parent (id, id);\n"},
		{name: "validated add with a new column that has a default", second: "-- +goose Up\nALTER TABLE child DROP CONSTRAINT child_parent_fkey;\n" +
			"ALTER TABLE child ADD COLUMN other_id uuid NOT NULL DEFAULT gen_random_uuid(),\n    ADD CONSTRAINT child_parent_fkey FOREIGN KEY (other_id, parent_id) REFERENCES parent (id, id);\n",
			want: []string{"00002_x.sql: child ADD CONSTRAINT child_parent_fkey FOREIGN KEY without NOT VALID: no SELECT TO nava_migrator USING (true) policy on child, parent in effect"}},
		{name: "validated add on a table created in the same file", first: strings.ReplaceAll(tables, " NOT VALID,", ","), second: "-- +goose Up\nSELECT 1;\n"},
		{name: "down section ignored", second: "-- +goose Up\nSELECT 1;\n-- +goose Down\n" + validate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			first := tables
			if c.first != "" {
				first = c.first
			}
			findings, err := checkFKValidateForceRLS(fstest.MapFS{
				"00001_tables.sql": {Data: []byte(first)},
				"00002_x.sql":      {Data: []byte(c.second)},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := findingStrings(findings); !slices.Equal(got, c.want) {
				t.Fatalf("findings =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
		})
	}
}

func findingStrings(fs []fkFinding) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.String())
	}
	return out
}

// fkFinding is one statement or file that breaks the rule. constraint is ""
// for a file-level finding.
type fkFinding struct {
	file, constraint, msg string
}

func (f fkFinding) key() string    { return f.file + " " + f.constraint }
func (f fkFinding) String() string { return f.file + ": " + f.msg }

var (
	reCreateTable = regexp.MustCompile(`^create table (?:if not exists )?(?:\w+\.)?(\w+)`)
	reAlterTable  = regexp.MustCompile(`^alter table (?:if exists )?(?:only )?(?:\w+\.)?(\w+) (.*)$`)
	reFKClause    = regexp.MustCompile(`^add constraint (\w+) foreign key \(([^)]*)\) references (?:\w+\.)?(\w+)`)
	reAddColumn   = regexp.MustCompile(`^add (?:column )?(?:if not exists )?(\w+) `)
	reInlineFK    = regexp.MustCompile(`constraint (\w+) foreign key \([^)]*\) references (?:\w+\.)?(\w+)`)
	reDropCon     = regexp.MustCompile(`^drop constraint (?:if exists )?(\w+)`)
	reValidate    = regexp.MustCompile(`^validate constraint (\w+)$`)
	reCreatePol   = regexp.MustCompile(`^create policy (\w+) on (?:\w+\.)?(\w+) (?:as permissive )?for select to ([\w, ]+?) using \(true\)$`)
	reDropPol     = regexp.MustCompile(`^drop policy (?:if exists )?(\w+) on (?:\w+\.)?(\w+)$`)
	reNoTx        = regexp.MustCompile(`(?mi)^-- \+goose no transaction\s*$`)
	reUpDown      = regexp.MustCompile(`(?mi)^-- \+goose (up|down)\b.*$`)
	reSpace       = regexp.MustCompile(`\s+`)
)

// checkFKValidateForceRLS replays the Up sections of fsys's migrations in
// name order, tracking which tables are FORCE-RLS and which constraints are
// foreign keys, and reports every FK check that would run as the owner
// without the migrator-policy pattern.
func checkFKValidateForceRLS(fsys fs.FS) ([]fkFinding, error) {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	forced := map[string]bool{}
	type fk struct{ table, ref string }
	fks := map[string]fk{}
	var findings []fkFinding
	for _, name := range names {
		raw, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		body := upSection(string(raw))
		created := map[string]bool{}    // tables created in this file: empty, nothing to check
		policies := map[string]string{} // live migrator SELECT policy name -> table
		usedPolicies := false
		covered := func(tables ...string) []string {
			var missing []string
			for _, tb := range tables {
				if !forced[tb] || slices.Contains(missing, tb) {
					continue
				}
				found := false
				for _, on := range policies {
					found = found || on == tb
				}
				if !found {
					missing = append(missing, tb)
				}
			}
			return missing
		}
		report := func(table, constraint, what string, missing []string) {
			findings = append(findings, fkFinding{name, constraint, fmt.Sprintf(
				"%s %s: no SELECT TO %s USING (true) policy on %s in effect", table, what, migratorRole, strings.Join(missing, ", "))})
		}
		for _, stmt := range splitStatements(body) {
			if m := reCreateTable.FindStringSubmatch(stmt); m != nil {
				created[m[1]] = true
				for _, c := range reInlineFK.FindAllStringSubmatch(stmt, -1) {
					fks[c[1]] = fk{m[1], c[2]}
				}
				continue
			}
			if m := reCreatePol.FindStringSubmatch(stmt); m != nil {
				for _, r := range strings.Split(m[3], ",") {
					if strings.TrimSpace(r) == migratorRole {
						policies[m[1]] = m[2]
						usedPolicies = true
					}
				}
				continue
			}
			if m := reDropPol.FindStringSubmatch(stmt); m != nil {
				if policies[m[1]] == m[2] {
					delete(policies, m[1])
				}
				continue
			}
			m := reAlterTable.FindStringSubmatch(stmt)
			if m == nil {
				continue
			}
			table := m[1]
			clauses := splitTopLevel(m[2], ',')
			nullCols := map[string]bool{} // columns this statement adds with no default: NULL in every row
			for _, clause := range clauses {
				if c := reAddColumn.FindStringSubmatch(clause); c != nil && c[1] != "constraint" &&
					!strings.Contains(clause, " default ") && !strings.Contains(clause, " generated ") {
					nullCols[c[1]] = true
				}
			}
			for _, clause := range clauses {
				switch {
				case clause == "force row level security":
					forced[table] = true
				case clause == "no force row level security":
					forced[table] = false
				case reDropCon.MatchString(clause):
					delete(fks, reDropCon.FindStringSubmatch(clause)[1])
				case reFKClause.MatchString(clause):
					c := reFKClause.FindStringSubmatch(clause)
					fks[c[1]] = fk{table, c[3]}
					if strings.HasSuffix(clause, " not valid") || created[table] ||
						slices.ContainsFunc(strings.Split(c[2], ","), func(col string) bool { return nullCols[strings.TrimSpace(col)] }) {
						continue
					}
					if missing := covered(table, c[3]); len(missing) > 0 {
						report(table, c[1], "ADD CONSTRAINT "+c[1]+" FOREIGN KEY without NOT VALID", missing)
					}
				case reValidate.MatchString(clause):
					con := reValidate.FindStringSubmatch(clause)[1]
					k, ok := fks[con]
					if !ok {
						continue // a CHECK: validated by a heap scan, RLS does not apply
					}
					if missing := covered(k.table, k.ref); len(missing) > 0 {
						report(table, con, "VALIDATE CONSTRAINT "+con, missing)
					}
				}
			}
		}
		for pol, on := range policies {
			findings = append(findings, fkFinding{name, "", fmt.Sprintf("policy %s on %s is not dropped in the same migration", pol, on)})
		}
		if usedPolicies && reNoTx.MatchString(string(raw)) {
			findings = append(findings, fkFinding{name, "", "-- +goose NO TRANSACTION: the migrator policies would be visible to every session"})
		}
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].file != findings[j].file {
			return findings[i].file < findings[j].file
		}
		return findings[i].msg < findings[j].msg
	})
	return findings, nil
}

// upSection is the text between "-- +goose Up" and "-- +goose Down" (or the
// end), with line comments removed.
func upSection(s string) string {
	var out []string
	up := false
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if m := reUpDown.FindStringSubmatch(line); m != nil {
			up = strings.EqualFold(m[1], "up")
			continue
		}
		if !up {
			continue
		}
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// splitStatements splits on top-level semicolons, skipping quoted and
// dollar-quoted text, and returns each statement lower-cased with its
// whitespace collapsed. Statements inside a dollar-quoted body (a DO block or
// a function) are not statements of the migration and are not returned
// separately.
func splitStatements(s string) []string {
	var out []string
	for _, st := range splitTopLevel(s, ';') {
		st = strings.TrimSpace(reSpace.ReplaceAllString(strings.ToLower(st), " "))
		if st != "" {
			out = append(out, st)
		}
	}
	return out
}

var reDollarTag = regexp.MustCompile(`^\$\w*\$`)

// splitTopLevel splits s on sep outside parentheses, single quotes and
// dollar quotes, trimming each part.
func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\'':
			for i++; i < len(s) && s[i] != '\''; i++ {
			}
		case c == '$':
			if tag := reDollarTag.FindString(s[i:]); tag != "" {
				end := strings.Index(s[i+len(tag):], tag)
				if end < 0 {
					i = len(s)
				} else {
					i += len(tag) + end + len(tag) - 1
				}
			}
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == sep && depth == 0:
			parts = append(parts, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	return append(parts, strings.TrimSpace(s[start:]))
}
