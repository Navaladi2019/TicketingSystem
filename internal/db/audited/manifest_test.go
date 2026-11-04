package audited_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

func grantManifest(t *testing.T) *grants.Manifest {
	t.Helper()
	m, err := grants.Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// The real manifest parses, and the effective grant manifest gives every
// writer exactly the R12 columns on audit_log, keeping nava_worker's sealer
// SELECT and giving nothing to a role that writes no audited table.
func TestLoadRealManifest(t *testing.T) {
	m, a, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := a.Writers(), []string{"nava_app", "nava_onboard", "nava_worker"}; !slices.Equal(got, want) {
		t.Fatalf("writers = %v, want %v", got, want)
	}
	for _, w := range a.Writers() {
		log := m.Roles[w].Tables[audited.LogTable]
		if !sameSet(log["INSERT"], audited.InsertColumns()) {
			t.Errorf("%s INSERT on audit_log = %v, want the R10 columns", w, log["INSERT"])
		}
		for _, c := range audited.SelectColumns() {
			if !slices.Contains(log["SELECT"], c) {
				t.Errorf("%s SELECT on audit_log lacks %s", w, c)
			}
		}
	}
	// nava_app is also the reader (P5-17, R8): db/grants.yaml lists its read
	// columns, diff and diff_salt included, and the merge adds the commit-check
	// columns to them. nava_onboard, a writer only, gets exactly the latter.
	got := m.Roles["nava_app"].Tables[audited.LogTable]["SELECT"]
	for _, c := range append(audited.SelectColumns(), "diff", "diff_salt") {
		if !slices.Contains(got, c) {
			t.Errorf("nava_app SELECT on audit_log lacks %s: %v", c, got)
		}
	}
	if got := m.Roles["nava_onboard"].Tables[audited.LogTable]["SELECT"]; !sameSet(got, audited.SelectColumns()) {
		t.Errorf("nava_onboard SELECT on audit_log = %v, want exactly the commit-check columns", got)
	}
	if got := m.Roles["nava_onboard"].Tables[audited.LogTable]; len(got) != 2 {
		t.Errorf("nava_onboard on audit_log = %v, want INSERT and SELECT only", got)
	}
	raw := grantManifest(t)
	if w := raw.Roles["nava_worker"].Tables[audited.LogTable]; len(m.Roles["nava_worker"].Tables[audited.LogTable]["SELECT"]) != len(w["SELECT"]) {
		t.Errorf("nava_worker's sealer SELECT changed in the merge")
	}
	for _, r := range []string{"nava_groupsync", "nava_relay", "nava_purge", "nava_erasure"} {
		if !reflect.DeepEqual(m.Roles[r], raw.Roles[r]) {
			t.Errorf("%s writes no audited table, but the merge changed its grants", r)
		}
	}
	// db/grants.yaml itself names no writer's R10 INSERT: it is generated.
	for _, w := range a.Writers() {
		if cols := raw.Roles[w].Tables[audited.LogTable]["INSERT"]; len(cols) > 0 {
			t.Errorf("db/grants.yaml lists %s INSERT on audit_log; it is generated from db/audited-tables.yaml", w)
		}
	}
}

// Merging leaves its input alone and is idempotent.
func TestMergeGrantsIdempotent(t *testing.T) {
	raw := grantManifest(t)
	before := grantManifest(t)
	a, err := audited.Parse(db.AuditedTables(), raw)
	if err != nil {
		t.Fatal(err)
	}
	once, err := a.MergeGrants(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(raw, before) {
		t.Fatal("MergeGrants modified its input")
	}
	twice, err := a.MergeGrants(once)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(once, twice) {
		t.Fatal("merging twice differs from merging once")
	}
}

// A table-level grant in db/grants.yaml stays table-level.
func TestMergeGrantsKeepsTableLevel(t *testing.T) {
	raw := grantManifest(t)
	r := raw.Roles["nava_app"]
	tables := map[string]map[string][]string{}
	for k, v := range r.Tables {
		tables[k] = v
	}
	tables[audited.LogTable] = map[string][]string{"SELECT": {}}
	r.Tables = tables
	raw.Roles["nava_app"] = r
	a, err := audited.Parse(db.AuditedTables(), raw)
	if err != nil {
		t.Fatal(err)
	}
	m, err := a.MergeGrants(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Roles["nava_app"].Tables[audited.LogTable]["SELECT"]; len(got) != 0 {
		t.Errorf("table-level SELECT became %v", got)
	}
}

func TestParseRejects(t *testing.T) {
	const good = `
audited:
  records:
    writer_roles: [nava_app]
    audited_columns: [data]
    subject_type: record
    subject_id_column: id
    attached: false
exempt:
  outbox: { reason: "transport" }
`
	if _, err := audited.Parse([]byte(good), grantManifest(t)); err != nil {
		t.Fatalf("good manifest: %v", err)
	}
	for name, tc := range map[string]struct{ yaml, want string }{
		"empty reason": {strings.Replace(good, `"transport"`, `"  "`, 1), "reason is required"},
		"unknown role": {strings.Replace(good, "[nava_app]", "[nava_nobody]", 1), "not a role in db/grants.yaml"},
		"owner writer": {strings.Replace(good, "[nava_app]", "[nava_migrator]", 1), "is the owner"},
		"no writers":   {strings.Replace(good, "[nava_app]", "[]", 1), "writer_roles is empty"},
		"repeated":     {strings.Replace(good, "[nava_app]", "[nava_app, nava_app]", 1), "writer_roles repeat"},
		"subject type": {strings.Replace(good, "subject_type: record", "subject_type: ticket", 1), "not registered"},
		"bad column":   {strings.Replace(good, "[data]", "[Data]", 1), "not a lower-case identifier"},
		"both lists":   {strings.Replace(good, "outbox:", "records:", 1), "also listed under exempt"},
		"the log":      {strings.Replace(good, "  records:", "  audit_log:", 1), "cannot audit itself"},
		"unknown key":  {strings.Replace(good, "attached: false", "attached: false\n    attach: true", 1), "attach"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := audited.Parse([]byte(tc.yaml), grantManifest(t))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

// The onboard pair is exactly the catalogue gate's pinned form, and the
// canonical pair names every other writer.
func TestPolicies(t *testing.T) {
	_, a, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]catalog.Policy{}
	for _, p := range a.Policies() {
		got[p.Name] = p.Policy
	}
	canonical := catalog.TenantPred("tenant_id")
	pinned := catalog.PinnedTenantPolicies()
	want := map[string]catalog.Policy{
		"audit_log_writer_insert":  {Cmd: "INSERT", Roles: "nava_app,nava_worker", Check: canonical},
		"audit_log_writer_select":  {Cmd: "SELECT", Roles: "nava_app,nava_worker", Qual: canonical},
		"audit_log_onboard_insert": pinned["public.audit_log.audit_log_onboard_insert"],
		"audit_log_onboard_select": pinned["public.audit_log.audit_log_onboard_select"],
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("policies = %v\nwant %v", got, want)
	}
}

func sameSet(a, b []string) bool {
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}
