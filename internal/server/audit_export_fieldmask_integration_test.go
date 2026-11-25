//go:build integration

package server_test

import (
	"bytes"
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

// maskedExport seeds a record (requester, org unit, one cost change) in
// tenant A and seals. With publish false the policy masks nothing.
func maskedExport(t *testing.T, publish bool) (*exportEnv, *maskEnv, string, uuid.UUID) {
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
	x.sealAll(t)
	return x, m, r.ID, unit
}

type exportedEntry struct {
	Kind         string  `json:"kind"`
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

// R7: a masked entry comes out with the marker, no value and no diff salt.
//
// covers: exportAuditEntries 200
func TestExportMaskedEntriesCarryTheMarkerAndNoSalt(t *testing.T) {
	x, _, rid, _ := maskedExport(t, true)
	exp := x.export(t, "admin-a", "?domain=record&subject_id="+rid)
	if exp.Code != http.StatusOK {
		t.Fatalf("export = %d: %s", exp.Code, exp.Body)
	}
	entries := entriesWithDiff(t, exp.Body.String())
	if len(entries) < 2 {
		t.Fatalf("entries with a diff = %d, want the create and the change", len(entries))
	}
	for i, e := range entries {
		if !e.DiffRedacted || e.DiffSalt != nil || !strings.Contains(*e.Diff, `"redacted":"mask"`) {
			t.Errorf("entry %d: redacted=%v salt=%v diff=%s, want redacted, no salt, the mask marker", i, e.DiffRedacted, e.DiffSalt, *e.Diff)
		}
	}
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
	for name, v := range map[string]string{"cost": "48613", "org unit": unit.String(), "requester": m.a.user.PrincipalID.String()} {
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
	for _, ver := range []string{"2", "3", "4"} {
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

	var pages atomic.Int32
	var bBody string
	var bCode int
	x.onPage = func() {
		if pages.Add(1) != 2 {
			return
		}
		rec := x.export(t, "admin-b", "?domain=record")
		bCode, bBody = rec.Code, rec.Body.String()
	}
	a := x.export(t, "admin-a", "?domain=record")
	x.onPage = nil
	if pages.Load() < 2 {
		t.Fatal("control: A's export had fewer than two pages")
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
// Each table is the one a single call reads: acl_versions for ReadMasks,
// entity_definition_versions for addOrgUnitMasks.
//
// covers: exportAuditEntries 500
func TestExportFailsClosedWhenAMaskReadFails(t *testing.T) {
	for name, table := range map[string]string{
		"ReadMasks":       "acl_versions",
		"addOrgUnitMasks": "entity_definition_versions",
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
			if _, err := x.super.Exec(t.Context(), "REVOKE SELECT ON "+table+" FROM nava_app"); err != nil {
				t.Fatal(err)
			}
			for _, q := range []string{"", "&format=csv"} {
				rec := x.export(t, "admin-a", "?domain=record&subject_id="+rid+q)
				if rec.Code != http.StatusInternalServerError || patchProblemCode(t, rec) != "internal-error" {
					t.Fatalf("export%s = %d: %.300s, want 500 internal-error", q, rec.Code, rec.Body)
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
