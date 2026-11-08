package audited

import (
	"reflect"
	"strings"
	"testing"
)

// events spells every trigger type CI-A3 can report, so a wrong trigger is
// named in the problem as PostgreSQL would declare it.
func TestEvents(t *testing.T) {
	for tgtype, want := range map[int16]string{
		tgRow | tgInsert | tgDelete | tgUpdate:      "AFTER INSERT DELETE UPDATE FOR EACH ROW",
		tgBefore | tgRow | tgInsert:                 "BEFORE INSERT FOR EACH ROW",
		tgInstead | tgRow | tgUpdate:                "INSTEAD OF UPDATE FOR EACH ROW",
		tgTruncate:                                  "AFTER TRUNCATE FOR EACH STATEMENT",
		tgInsert | tgDelete | tgUpdate | tgTruncate: "AFTER INSERT DELETE UPDATE TRUNCATE FOR EACH STATEMENT",
	} {
		if got := events(tgtype); got != want {
			t.Errorf("events(%d) = %q, want %q", tgtype, got, want)
		}
	}
}

func TestFKAction(t *testing.T) {
	for code, want := range map[string]string{
		"a": "NO ACTION", "r": "RESTRICT", "c": "CASCADE", "n": "SET NULL", "d": "SET DEFAULT", "x": "x",
	} {
		if got := fkAction(code); got != want {
			t.Errorf("fkAction(%q) = %q, want %q", code, got, want)
		}
	}
}

// A trigger that departs from the manifest in each way CI-A3 checks is
// reported once per departure; the expected trigger reports nothing.
func TestCheckTrigger(t *testing.T) {
	table := Table{AuditedColumns: []string{"data", "state"}, SubjectType: "record", SubjectIDColumn: "id"}
	good := trigger{
		table: "public.records", name: "records_audit_commit_check",
		tgtype: tgRow | tgInsert | tgDelete | tgUpdate, deferrable: true, initDeferred: true, constraint: true,
		enabled: "O", columns: []string{"data", "state"}, args: []string{"record", "id"},
	}
	if p := checkTrigger("records", table, good); len(p) != 0 {
		t.Fatalf("the expected trigger: %v", p)
	}
	always := good
	always.enabled = "A"
	if p := checkTrigger("records", table, always); len(p) != 0 {
		t.Fatalf("an ENABLE ALWAYS trigger: %v", p)
	}
	// With no audited columns the trigger must not fire on UPDATE.
	insertOnly := good
	insertOnly.tgtype, insertOnly.columns = tgRow|tgInsert|tgDelete, nil
	if p := checkTrigger("records", Table{SubjectType: "record", SubjectIDColumn: "id"}, insertOnly); len(p) != 0 {
		t.Fatalf("an insert/delete trigger for a table with no audited columns: %v", p)
	}

	bad := good
	bad.tgtype = tgBefore | tgRow | tgInsert
	bad.deferrable = false
	bad.enabled = "D"
	bad.columns = []string{"data"}
	bad.args = []string{"record"}
	p := checkTrigger("records", table, bad)
	wants := []string{
		"events BEFORE INSERT FOR EACH ROW, want AFTER INSERT DELETE UPDATE FOR EACH ROW",
		"must be a CONSTRAINT TRIGGER, DEFERRABLE INITIALLY DEFERRED",
		`is not enabled (tgenabled "D")`,
		"UPDATE OF [data], want exactly the audited columns [data state]",
		`arguments ["record"], want ["record" "id"]`,
	}
	if len(p) != len(wants) {
		t.Fatalf("problems %q, want %d", p, len(wants))
	}
	for i, w := range wants {
		if !strings.HasPrefix(p[i], "trigger records_audit_commit_check on records: ") || !strings.HasSuffix(p[i], w) {
			t.Errorf("problem %d = %q, want it to end with %q", i, p[i], w)
		}
	}
}

// qualify adds the public schema, and leaves a name it cannot split alone
// (Validate has already reported it).
func TestQualify(t *testing.T) {
	got := []string{qualify("records"), qualify("public.records"), qualify("a.b.c")}
	if want := []string{"public.records", "public.records", "a.b.c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("qualify = %q, want %q", got, want)
	}
}
