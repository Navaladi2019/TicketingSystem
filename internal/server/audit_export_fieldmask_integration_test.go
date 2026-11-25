//go:build integration

package server_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// A reader the record ACL hides cost and requester from must not read them,
// nor the org unit, in the export: the list redacts them (control), so the
// export must too (ADR-004 R7, CI-A12; TODO-S10-01).
//
// covers: exportAuditEntries 200
func TestExportAppliesTheCallersFieldMasks(t *testing.T) {
	x := newExportEnv(t)
	m := &maskEnv{etEnv: x.etEnv, desk: x.a.lookalikeGroup}
	if rec := m.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "xm-t", maskTypeBody("mjob", "MJOB")); rec.Code != http.StatusCreated {
		t.Fatalf("seed type = %d: %s", rec.Code, rec.Body)
	}
	rules := acltest.AllowUsers(m.a.admin.PrincipalID, m.a.user.PrincipalID, m.a.lookalike.PrincipalID)
	if err := tenancy.InTenantTx(t.Context(), m.pool, m.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "mjob", []authz.Rule{rules}, m.deskMasks())
		return err
	}); err != nil {
		t.Fatal(err)
	}
	unit := m.seedUnit(t, m.a.id, m.a.user.PrincipalID)
	r := m.mustCreateRec(t, "user-a", "mjob", "xm-1", requesterBody(m.a.user.PrincipalID))
	if rec := m.patch(t, "lookalike-a", "mjob", r.ID, `W/"1"`, `{"data": {"cost": 48613}}`); rec.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
	}
	x.sealAll(t)

	requester := `"requester":"` + m.a.user.PrincipalID.String() + `"`
	secrets := map[string]string{"cost": "48613", "org unit": unit.String(), "requester": requester}
	list, _ := m.auditGet(t, "admin-a", "?domain=record&subject_type=record&subject_id="+r.ID)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "redacted") {
		t.Fatalf("control: list = %d: %s", list.Code, list.Body)
	}
	compact := func(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, ": ", ":"), `\"`, `"`) }
	for _, q := range []string{"", "&format=csv"} {
		exp := x.export(t, "admin-a", "?domain=record&subject_id="+r.ID+q)
		if exp.Code != http.StatusOK {
			t.Fatalf("export = %d: %s", exp.Code, exp.Body)
		}
		for name, v := range secrets {
			if strings.Contains(compact(list.Body.String()), v) {
				t.Errorf("control: the list shows the masked %s", name)
			}
			if strings.Contains(strings.ReplaceAll(compact(exp.Body.String()), `""`, `"`), v) {
				t.Errorf("export%s shows the masked %s", q, name)
			}
		}
	}
}

// maskedExport is seedMaskedExport, then seals.
func maskedExport(t *testing.T, publish bool) (*exportEnv, *maskEnv, string, uuid.UUID) {
	t.Helper()
	x, m, rid, unit := seedMaskedExport(t, publish)
	x.sealAll(t)
	return x, m, rid, unit
}

// seedMaskedExport seeds a record (requester, org unit, one cost change) in
// tenant A, not yet sealed. With publish false the policy masks nothing.
func seedMaskedExport(t *testing.T, publish bool) (*exportEnv, *maskEnv, string, uuid.UUID) {
	t.Helper()
	x := newExportEnv(t)
	m := &maskEnv{etEnv: x.etEnv, desk: x.a.lookalikeGroup}
	if rec := m.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "xm-t", maskTypeBody("mjob", "MJOB")); rec.Code != http.StatusCreated {
		t.Fatalf("seed type = %d: %s", rec.Code, rec.Body)
	}
	var masks []authz.Mask
	if publish {
		masks = m.deskMasks()
	}
	rules := acltest.AllowUsers(m.a.admin.PrincipalID, m.a.user.PrincipalID, m.a.lookalike.PrincipalID)
	if err := tenancy.InTenantTx(t.Context(), m.pool, m.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "mjob", []authz.Rule{rules}, masks)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	unit := m.seedUnit(t, m.a.id, m.a.user.PrincipalID)
	r := m.mustCreateRec(t, "user-a", "mjob", "xm-1", requesterBody(m.a.user.PrincipalID))
	if rec := m.patch(t, "lookalike-a", "mjob", r.ID, `W/"1"`, `{"data": {"cost": 48613}}`); rec.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
	}
	return x, m, r.ID, unit
}

type exportedEntry struct {
	Kind         string  `json:"kind"`
	Sealed       bool    `json:"sealed"`
	Diff         *string `json:"diff"`
	DiffSalt     *string `json:"diff_salt"`
	DiffRedacted bool    `json:"diff_redacted"`
}

// entriesWithDiff are the JSON lines export's entry lines that carry a diff.
func entriesWithDiff(t *testing.T, body string) []exportedEntry {
	t.Helper()
	var out []exportedEntry
	for _, l := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
		var e exportedEntry
		if err := json.Unmarshal([]byte(l), &e); err != nil {
			t.Fatalf("line %q: %v", l, err)
		}
		if e.Kind == "entry" && e.Diff != nil {
			out = append(out, e)
		}
	}
	return out
}

// exportedEntries reads the entry lines that carry a diff from a JSON lines
// or a CSV export, into one shape. A CSV file is read by column name.
func exportedEntries(t *testing.T, body string, asCSV bool) []exportedEntry {
	t.Helper()
	if !asCSV {
		return entriesWithDiff(t, body)
	}
	cr := csv.NewReader(strings.NewReader(body))
	cr.FieldsPerRecord = -1
	rows, err := cr.ReadAll()
	if err != nil {
		t.Fatalf("csv: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("csv: no rows")
	}
	col := map[string]int{}
	for i, name := range rows[0] {
		col[name] = i
	}
	for _, name := range []string{"kind", "sealed", "diff", "diff_salt", "diff_redacted"} {
		if _, ok := col[name]; !ok {
			t.Fatalf("csv: no column %q in %v", name, rows[0])
		}
	}
	var out []exportedEntry
	for _, row := range rows[1:] {
		if row[col["kind"]] != "entry" || row[col["diff"]] == "" {
			continue
		}
		e := exportedEntry{Kind: "entry", Sealed: row[col["sealed"]] == "true", DiffRedacted: row[col["diff_redacted"]] == "true"}
		d := row[col["diff"]]
		e.Diff = &d
		if sl := row[col["diff_salt"]]; sl != "" {
			e.DiffSalt = &sl
		}
		out = append(out, e)
	}
	return out
}

// R7: a masked entry comes out with the marker, no value and no diff salt, in
// both formats, before the entries are sealed (recent entries are read
// through the unsealed page) and after. A second environment whose policy
// masks nothing is the control: the same checks find cost, org unit and
// requester, with the salt, in its entries, so a wrong string format could
// not let the masked export pass.
//
// covers: exportAuditEntries 200
func TestExportMaskedEntriesCarryTheMarkerAndNoSalt(t *testing.T) {
	x, m, rid, unit := seedMaskedExport(t, true)
	cx, cm, crid, cunit := seedMaskedExport(t, false)
	needles := func(m *maskEnv, unit uuid.UUID) map[string]string {
		return map[string]string{"cost": "48613", "org unit": unit.String(), "requester": `"requester":"` + m.a.user.PrincipalID.String() + `"`}
	}
	diffs := func(entries []exportedEntry) string {
		var b strings.Builder
		for _, e := range entries {
			b.WriteString(strings.ReplaceAll(*e.Diff, ": ", ":"))
			b.WriteByte('\n')
		}
		return b.String()
	}
	run := func(t *testing.T, sealed bool) {
		for _, asCSV := range []bool{false, true} {
			q, name := "", "jsonl"
			if asCSV {
				q, name = "&format=csv", "csv"
			}
			t.Run(name, func(t *testing.T) {
				// Control: nothing masked, so every needle is found and the
				// diff has its salt.
				ctl := cx.export(t, "admin-a", "?domain=record&subject_id="+crid+q)
				if ctl.Code != http.StatusOK {
					t.Fatalf("control export = %d: %s", ctl.Code, ctl.Body)
				}
				cEntries := exportedEntries(t, ctl.Body.String(), asCSV)
				if len(cEntries) < 2 {
					t.Fatalf("control: entries with a diff = %d, want the create and the change", len(cEntries))
				}
				for i, e := range cEntries {
					// The salt is checked once sealed (TestExportWithoutMasksShowsEverything
					// does the same); an unsealed entry's salt is not this test's subject.
					if e.Sealed != sealed || e.DiffRedacted || (sealed && e.DiffSalt == nil) {
						t.Errorf("control entry %d: sealed=%v redacted=%v salt=%v, want sealed=%v, clear (with its salt once sealed)", i, e.Sealed, e.DiffRedacted, e.DiffSalt, sealed)
					}
				}
				for what, v := range needles(cm, cunit) {
					if !strings.Contains(diffs(cEntries), v) {
						t.Errorf("control: the unmasked export has no %s (%q)", what, v)
					}
				}

				exp := x.export(t, "admin-a", "?domain=record&subject_id="+rid+q)
				if exp.Code != http.StatusOK {
					t.Fatalf("export = %d: %s", exp.Code, exp.Body)
				}
				entries := exportedEntries(t, exp.Body.String(), asCSV)
				if len(entries) < 2 {
					t.Fatalf("entries with a diff = %d, want the create and the change", len(entries))
				}
				for i, e := range entries {
					if e.Sealed != sealed {
						t.Fatalf("control: entry %d sealed = %v, want %v", i, e.Sealed, sealed)
					}
					if !e.DiffRedacted || e.DiffSalt != nil || !strings.Contains(*e.Diff, `"redacted":"mask"`) {
						t.Errorf("entry %d: redacted=%v salt=%v diff=%s, want redacted, no salt, the mask marker", i, e.DiffRedacted, e.DiffSalt, *e.Diff)
					}
				}
				for what, v := range needles(m, unit) {
					if strings.Contains(diffs(entries), v) {
						t.Errorf("the masked export shows the %s (%q)", what, v)
					}
				}
			})
		}
	}
	t.Run("unsealed", func(t *testing.T) { run(t, false) })
	x.sealAll(t)
	cx.sealAll(t)
	t.Run("sealed", func(t *testing.T) { run(t, true) })
}

// A caller whose policy masks nothing still gets every value and the salt.
//
// covers: exportAuditEntries 200
func TestExportWithoutMasksShowsEverything(t *testing.T) {
	x, m, rid, unit := maskedExport(t, false)
	exp := x.export(t, "admin-a", "?domain=record&subject_id="+rid)
	if exp.Code != http.StatusOK {
		t.Fatalf("export = %d: %s", exp.Code, exp.Body)
	}
	body := strings.ReplaceAll(strings.ReplaceAll(exp.Body.String(), `\"`, `"`), ": ", ":")
	for name, v := range map[string]string{"cost": "48613", "org unit": unit.String(), "requester": `"requester":"` + m.a.user.PrincipalID.String() + `"`} {
		if !strings.Contains(body, v) {
			t.Errorf("an unmasked export lacks the %s", name)
		}
	}
	for i, e := range entriesWithDiff(t, exp.Body.String()) {
		if e.DiffRedacted || e.DiffSalt == nil {
			t.Errorf("entry %d: redacted=%v salt=%v, want clear with its salt", i, e.DiffRedacted, e.DiffSalt)
		}
	}
}

// R7, nava audit verify: a file with masked fields passes, and the masked
// entries are listed as not verifiable.
//
// covers: exportAuditEntries 200
func TestExportWithMaskedFieldsStillVerifies(t *testing.T) {
	x, _, rid, _ := maskedExport(t, true)
	exp := x.export(t, "admin-a", "?domain=record&subject_id="+rid)
	if exp.Code != http.StatusOK {
		t.Fatalf("export = %d: %s", exp.Code, exp.Body)
	}
	rep, err := verify.Verify(bytes.NewReader(exp.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Outcome != verify.OutcomePass {
		t.Fatalf("outcome %s, failure %+v\n%s", rep.Outcome, rep.Failure, exp.Body)
	}
	n := 0
	for _, o := range rep.Lines {
		if o == verify.LineDiffUnverified {
			n++
		}
	}
	if n == 0 {
		t.Errorf("no masked entry listed as diff_unverified: %v", rep.Lines)
	}
}

// One export per tenant at a time, so the overlap is across two tenants:
// A's caller masks cost, B's masks nothing, and B exports between two of A's
// pages.
//
// covers: exportAuditEntries 200
func TestOverlappingExportsKeepTheirOwnMasks(t *testing.T) {
	x, m, rid, _ := maskedExport(t, true)
	// More masked entries, so A has pages after the one B runs in.
	for _, ver := range []string{"2", "3", "4", "5", "6", "7"} {
		if rec := m.patch(t, "lookalike-a", "mjob", rid, `W/"`+ver+`"`, `{"data": {"cost": 48613}}`); rec.Code != http.StatusOK {
			t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
		}
	}
	if rec := m.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "xm-tb", maskTypeBody("mjob", "MJOB")); rec.Code != http.StatusCreated {
		t.Fatalf("seed type B = %d: %s", rec.Code, rec.Body)
	}
	rulesB := acltest.AllowUsers(m.b.admin.PrincipalID, m.b.user.PrincipalID)
	if err := tenancy.InTenantTx(t.Context(), m.pool, m.b.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "mjob", []authz.Rule{rulesB}, nil)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	rb := m.mustCreateRec(t, "user-b", "mjob", "xb-1", `{"short_description": "b", "cost": 77123}`)
	x.sealAll(t)

	// pages counts A's page transactions only: B's own run inside the hook is
	// not counted, so the control below can fail.
	var pages atomic.Int32
	var inB atomic.Bool
	var bBody string
	var bCode int
	x.onPage = func() {
		// Call 1 and 2 are the header's reads; call 3 is A's first data
		// page; 4 is the start of the second.
		if inB.Load() {
			return
		}
		if pages.Add(1) != 4 {
			return
		}
		inB.Store(true)
		defer inB.Store(false)
		rec := x.export(t, "admin-b", "")
		bCode, bBody = rec.Code, rec.Body.String()
	}
	a := x.export(t, "admin-a", "")
	x.onPage = nil
	if pages.Load() < 5 {
		t.Fatalf("control: A's export ran %d page transactions, want B between two of them, with more to follow", pages.Load())
	}
	if bCode == 0 {
		t.Fatal("control: B's export did not run")
	}
	if a.Code != http.StatusOK || bCode != http.StatusOK {
		t.Fatalf("A = %d, B = %d: %.300s", a.Code, bCode, bBody)
	}
	if strings.Contains(a.Body.String(), "48613") {
		t.Error("A's file shows the masked cost")
	}
	if !strings.Contains(strings.ReplaceAll(bBody, `\"`, `"`), "77123") {
		t.Error("B's file lacks its own cost")
	}
	if strings.Contains(bBody, x.a.id.String()) || strings.Contains(a.Body.String(), x.b.id.String()) || strings.Contains(a.Body.String(), rb.ID) {
		t.Error("a file holds the other tenant's entries")
	}
}

// A mask read that fails ends the request before the lease and the entry.
// The REVOKE cases fail the SELECT itself (acl_versions for ReadMasks,
// entity_definition_versions for addOrgUnitMasks); that aborts the
// transaction, so a dropped error would still end in 500 at export.Begin.
// The other cases leave the transaction usable and fail the decode of what
// was read: only these catch a dropped error. As the superuser in the test's
// own database, acl_versions.masks becomes an object (a list is expected, so its CHECK is dropped) and
// fixed_columns gets a value that is no string (its CHECK allows any object).
//
// covers: exportAuditEntries 500
func TestExportFailsClosedWhenAMaskReadFails(t *testing.T) {
	revoke := func(table string) func(*testing.T, *exportEnv) {
		return func(t *testing.T, x *exportEnv) {
			if _, err := x.super.Exec(t.Context(), "REVOKE SELECT ON "+table+" FROM nava_app"); err != nil {
				t.Fatal(err)
			}
		}
	}
	for name, breakIt := range map[string]func(*testing.T, *exportEnv){
		"ReadMasks, select refused":       revoke("acl_versions"),
		"addOrgUnitMasks, select refused": revoke("entity_definition_versions"),
		"ReadMasks, masks do not decode": func(t *testing.T, x *exportEnv) {
			// The table's CHECK refuses a masks value of the wrong shape; drop it in this database.
			if _, err := x.super.Exec(t.Context(), `ALTER TABLE acl_versions DROP CONSTRAINT acl_versions_masks_check`); err != nil {
				t.Fatal(err)
			}
			if _, err := x.super.Exec(t.Context(), `UPDATE acl_versions SET masks = '{"not":"a list"}'::jsonb WHERE tenant_id = $1`, x.a.id); err != nil {
				t.Fatal(err)
			}
		},
		"addOrgUnitMasks, fixed_columns do not decode": func(t *testing.T, x *exportEnv) {
			if _, err := x.super.Exec(t.Context(), `UPDATE entity_definition_versions SET fixed_columns = '{"requester": 5}'::jsonb WHERE tenant_id = $1`, x.a.id); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			x, _, rid, _ := maskedExport(t, true)
			lease := func() (l *time.Time) {
				if err := x.super.QueryRow(t.Context(), "SELECT export_lease_until FROM audit_chain_head WHERE tenant_id = $1", x.a.id).Scan(&l); err != nil {
					t.Fatal(err)
				}
				return l
			}
			exports := func() (n int) {
				if err := x.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'export'", x.a.id).Scan(&n); err != nil {
					t.Fatal(err)
				}
				return n
			}
			before, nBefore := lease(), exports()
			breakIt(t, x)
			for _, q := range []string{"", "&format=csv"} {
				rec := x.export(t, "admin-a", "?domain=record&subject_id="+rid+q)
				if rec.Code != http.StatusInternalServerError || patchProblemCode(t, rec) != "internal_error" {
					t.Fatalf("export%s = %d: %.300s, want 500 internal_error", q, rec.Code, rec.Body)
				}
				if strings.Contains(rec.Header().Get("Content-Type"), "ndjson") || strings.Contains(rec.Body.String(), `"kind":"header"`) {
					t.Errorf("export%s wrote file bytes: %.300s", q, rec.Body)
				}
			}
			after := lease()
			if (before == nil) != (after == nil) || (before != nil && !before.Equal(*after)) {
				t.Errorf("export_lease_until changed: %v -> %v", before, after)
			}
			if n := exports(); n != nBefore {
				t.Errorf("audit/export entries = %d, want %d", n, nBefore)
			}
		})
	}
}

// The gate comes before the mask read: without audit.export the answer is 403
// even though the mask read could not succeed.
//
// covers: exportAuditEntries 403
func TestExportGateComesBeforeTheMaskRead(t *testing.T) {
	x, _, rid, _ := maskedExport(t, true)
	if _, err := x.super.Exec(t.Context(), "REVOKE SELECT ON acl_versions FROM nava_app"); err != nil {
		t.Fatal(err)
	}
	if rec := x.export(t, "user-a", "?domain=record&subject_id="+rid); rec.Code != http.StatusForbidden {
		t.Errorf("export without audit.export = %d: %.300s, want 403", rec.Code, rec.Body)
	}
}
