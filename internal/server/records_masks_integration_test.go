//go:build integration

// P3-33 (TICK-618): field masks through the real handler, on real
// PostgreSQL. A field masked for the caller cannot be written (422
// field_not_writable, nothing written) and is never in a read: GET, the 201
// of a create, the 200 of a PATCH, history (before, after, changed[]) and
// the audit timeline's diff. The same reads by a caller the mask admits
// contain it.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// maskSchema is recSchema plus a data property (cost), a data property with
// a default (triage, so a create stores it without the caller sending it)
// and a unique one (serial).
var maskSchema = strings.Replace(recSchema, `"requester": {"type": "string", "format": "uuid"}`,
	`"requester": {"type": "string", "format": "uuid"},
    "cost": {"type": "number"},
    "triage": {"type": "string", "default": "pending"},
    "serial": {"type": "string", "x-nava-unique": true}`, 1)

func maskTypeBody(key, prefix string) string {
	return fmt.Sprintf(`{"key": %q, "name": "Type %s", "number_prefix": %q, "definition": {"schema": %s,
  "fixed_columns": {"state": "state", "group": "assignment_group_id", "assignee": "assignee_id", "requester": "requester_id"},
  "states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}}`,
		key, key, prefix, maskSchema)
}

// maskEnv: entity type "job" in tenant A. Every seeded principal may read and
// write every job. cost, triage and assignee (a fixed column) are readable
// and writable by the desk group only (lookalike-a, "the agent"); requester
// is masked with a row-dependent subject only, so it is hidden from everyone
// (A2). user-a is "the requester": in no group.
type maskEnv struct {
	*etEnv
	desk uuid.UUID
}

func newMaskEnv(t *testing.T) *maskEnv {
	t.Helper()
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "mk-a", maskTypeBody("job", "JOB")); rec.Code != http.StatusCreated {
		t.Fatalf("seed type = %d: %s", rec.Code, rec.Body)
	}
	m := &maskEnv{etEnv: e, desk: e.a.lookalikeGroup}
	m.publishMasks(t, m.deskMasks())
	return m
}

func (m *maskEnv) deskMasks() []authz.Mask {
	desk := authz.Subject{Groups: []uuid.UUID{m.desk}}
	return []authz.Mask{
		{Field: "cost", Read: desk, Write: desk},
		{Field: "triage", Read: desk, Write: desk},
		{Field: "assignee", Read: desk, Write: desk},
		{Field: "requester", Read: authz.Subject{Requester: true}, Write: authz.Subject{Requester: true, Users: []uuid.UUID{m.a.user.PrincipalID}}},
	}
}

func (m *maskEnv) publishMasks(t *testing.T, masks []authz.Mask) {
	t.Helper()
	rules := []authz.Rule{acltest.AllowUsers(m.a.admin.PrincipalID, m.a.user.PrincipalID, m.a.lookalike.PrincipalID)}
	if err := tenancy.InTenantTx(t.Context(), m.pool, m.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "job", rules, masks)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func (m *maskEnv) tenantAudits(t *testing.T) int {
	t.Helper()
	var n int
	if err := m.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND domain = 'record'", m.a.id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

type maskProblem struct {
	Code   string `json:"code"`
	Errors []struct {
		Pointer string `json:"pointer"`
		Rule    string `json:"rule"`
	} `json:"errors"`
}

func wantNotWritable(t *testing.T, name string, rec *httptest.ResponseRecorder, pointers ...string) {
	t.Helper()
	var p maskProblem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("%s: not a problem: %s", name, rec.Body)
	}
	var got []string
	for _, f := range p.Errors {
		got = append(got, f.Pointer)
	}
	if rec.Code != http.StatusUnprocessableEntity || p.Code != "field_not_writable" || !slices.Equal(got, pointers) {
		t.Fatalf("%s = %d %s, want 422 field_not_writable at %v", name, rec.Code, rec.Body, pointers)
	}
}

func recordData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var b recBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("not a record: %s", rec.Body)
	}
	return b.Data
}

func hasNone(t *testing.T, name string, data map[string]any, fields ...string) {
	t.Helper()
	for _, f := range fields {
		if _, ok := data[f]; ok {
			t.Errorf("%s has masked field %q: %v", name, f, data)
		}
	}
}

// covers: createRecord 201 422
// covers: patchRecord 200 422
// covers: getRecord 200
func TestMaskedFieldIsNeitherWrittenNorRead(t *testing.T) {
	m := newMaskEnv(t)
	masked := []string{"cost", "triage", "assignee", "requester"}
	records, audits := m.recordCount(t, "a"), m.tenantAudits(t)

	// The requester cannot write a masked field: data, fixed column, or a
	// field with a default. Nothing is written.
	wantNotWritable(t, "create with cost", m.createRec(t, "user-a", "job", "mk-c1", `{"short_description": "s", "cost": 3}`), "/data/cost")
	wantNotWritable(t, "create with assignee and triage",
		m.createRec(t, "user-a", "job", "mk-c2", fmt.Sprintf(`{"short_description": "s", "assignee": %q, "triage": "pending"}`, m.a.lookalike.PrincipalID)),
		"/data/assignee", "/data/triage")
	if got := m.recordCount(t, "a"); got != records {
		t.Fatalf("records = %d, want %d: a refused create wrote", got, records)
	}
	if got := m.tenantAudits(t); got != audits {
		t.Fatalf("audit entries = %d, want %d: a refused create wrote", got, audits)
	}

	// Without masked fields the create succeeds; the server's default for
	// triage is not the caller's write. The 201 hides every masked field.
	rec := m.createRec(t, "user-a", "job", "mk-c3", `{"short_description": "s"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	hasNone(t, "the requester's 201", recordData(t, rec), masked...)
	var r recBody
	_ = json.Unmarshal(rec.Body.Bytes(), &r)

	// A PATCH that sets or changes a masked field is refused; one that does
	// not succeeds, and its 200 hides the masked fields.
	wantNotWritable(t, "patch cost", m.patch(t, "user-a", "job", r.ID, `W/"1"`, `{"data": {"cost": 9}}`), "/data/cost")
	wantNotWritable(t, "patch removes triage", m.patch(t, "user-a", "job", r.ID, `W/"1"`, `{"data": {"triage": null}}`), "/data/triage")
	if got := m.auditCount(t, r.ID); got != 1 {
		t.Fatalf("audit entries of the record = %d, want 1 after the refusals", got)
	}
	rec = m.patch(t, "user-a", "job", r.ID, `W/"1"`, `{"data": {"short_description": "t"}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch of an open field = %d: %s", rec.Code, rec.Body)
	}
	hasNone(t, "the requester's PATCH 200", recordData(t, rec), masked...)

	// The agent in Write can write them, and sees them on its 200.
	agentPatch := fmt.Sprintf(`{"data": {"cost": 9, "assignee": %q}}`, m.a.lookalike.PrincipalID)
	rec = m.patch(t, "lookalike-a", "job", r.ID, `W/"2"`, agentPatch)
	if rec.Code != http.StatusOK {
		t.Fatalf("the agent's patch = %d: %s", rec.Code, rec.Body)
	}
	if d := recordData(t, rec); d["cost"] != float64(9) || d["triage"] != "pending" || d["assignee"] != m.a.lookalike.PrincipalID.String() {
		t.Fatalf("the agent's PATCH 200 data = %v, want cost, triage and assignee", d)
	}
	hasNone(t, "the agent's PATCH 200", recordData(t, rec), "requester")

	// GET: hidden from the requester and the admin, shown to the agent.
	for _, token := range []string{"user-a", "admin-a"} {
		rec = m.do(t, http.MethodGet, "/api/v1/records/job/"+r.ID, token, "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s GET = %d: %s", token, rec.Code, rec.Body)
		}
		hasNone(t, token+" GET", recordData(t, rec), masked...)
	}
	rec = m.do(t, http.MethodGet, "/api/v1/records/job/"+r.ID, "lookalike-a", "", "")
	if d := recordData(t, rec); d["cost"] != float64(9) || d["triage"] != "pending" || d["assignee"] == nil {
		t.Fatalf("the agent's GET data = %v, want the desk fields", d)
	}

	// A2: requester is masked with Requester only, so its own requester does
	// not see it either.
	rec = m.createRec(t, "user-a", "job", "mk-c4", fmt.Sprintf(`{"short_description": "mine", "requester": %q}`, m.a.user.PrincipalID))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create with requester (writable by user-a by Users) = %d: %s", rec.Code, rec.Body)
	}
	hasNone(t, "the requester's own record", recordData(t, rec), "requester")
}

// A mask on the assignment group's fixed-column property also hides the
// top-level assignment_group of the body.
func TestMaskedGroupPropertyHidesAssignmentGroup(t *testing.T) {
	m := newMaskEnv(t)
	desk := authz.Subject{Groups: []uuid.UUID{m.desk}}
	m.publishMasks(t, append(m.deskMasks(), authz.Mask{Field: "group", Read: desk, Write: desk}))
	r := m.mustCreateRec(t, "lookalike-a", "job", "mk-g1", fmt.Sprintf(`{"short_description": "s", "group": %q}`, m.desk))
	var b struct {
		AssignmentGroup *struct{ ID string } `json:"assignment_group"`
		Data            map[string]any       `json:"data"`
	}
	rec := m.do(t, http.MethodGet, "/api/v1/records/job/"+r.ID, "user-a", "", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil || b.AssignmentGroup != nil || b.Data["group"] != nil {
		t.Fatalf("the requester's GET = %s, want no group anywhere", rec.Body)
	}
	rec = m.do(t, http.MethodGet, "/api/v1/records/job/"+r.ID, "lookalike-a", "", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil || b.AssignmentGroup == nil || b.Data["group"] != m.desk.String() {
		t.Fatalf("the agent's GET = %s, want the group", rec.Body)
	}
}

type historyBody struct {
	Items []struct {
		Changed []string        `json:"changed"`
		Before  *map[string]any `json:"before"`
		After   map[string]any  `json:"after"`
	} `json:"items"`
}

func (m *maskEnv) history(t *testing.T, token, id string) historyBody {
	t.Helper()
	rec := m.do(t, http.MethodGet, "/api/v1/records/job/"+id+"/history", token, "", "")
	var h historyBody
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &h) != nil {
		t.Fatalf("%s history = %d: %s", token, rec.Code, rec.Body)
	}
	return h
}

// covers: getRecordHistory 200
func TestHistoryOfAMaskedOnlyChange(t *testing.T) {
	m := newMaskEnv(t)
	r := m.mustCreateRec(t, "lookalike-a", "job", "mk-h1", `{"short_description": "s", "cost": 1}`)
	if rec := m.patch(t, "lookalike-a", "job", r.ID, `W/"1"`, `{"data": {"cost": 2}}`); rec.Code != http.StatusOK {
		t.Fatalf("the agent's patch = %d: %s", rec.Code, rec.Body)
	}

	h := m.history(t, "user-a", r.ID)
	if len(h.Items) != 2 {
		t.Fatalf("the requester's history has %d entries, want 2 (a masked-only change still appears)", len(h.Items))
	}
	top := h.Items[0] // newest first: the cost-only change
	if len(top.Changed) != 0 || top.Before == nil {
		t.Fatalf("masked-only change: changed %v before %v, want empty changed[] and a before", top.Changed, top.Before)
	}
	for i, it := range h.Items {
		hasNone(t, fmt.Sprintf("history[%d].after", i), it.After, "cost", "triage")
		if it.Before != nil {
			hasNone(t, fmt.Sprintf("history[%d].before", i), *it.Before, "cost", "triage")
		}
		for _, c := range it.Changed {
			if c == "/data/cost" || c == "/data/triage" {
				t.Errorf("history[%d].changed names masked %s", i, c)
			}
		}
	}

	h = m.history(t, "lookalike-a", r.ID)
	if top := h.Items[0]; !slices.Equal(top.Changed, []string{"/data/cost"}) || top.After["cost"] != float64(2) || (*top.Before)["cost"] != float64(1) {
		t.Fatalf("the agent's history top = %+v, want cost 1 -> 2", top)
	}
}

// PR #523 finding 4: the audit timeline is not ACL-filtered, but a field the
// caller's masks hide is masked in every record entry's diff, as a secret
// is, and the salt is withheld. A policy that lets the reader see the field
// returns the diff as stored.
//
// covers: listAuditEntries 200
func TestAuditEntriesMaskFieldsHiddenFromTheReader(t *testing.T) {
	m := newMaskEnv(t)
	r := m.mustCreateRec(t, "lookalike-a", "job", "mk-a1", `{"short_description": "s", "cost": 41}`)

	_, page := m.auditGet(t, "admin-a", "?domain=record&subject_type=record&subject_id="+r.ID)
	if len(page.Items) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(page.Items))
	}
	it := page.Items[0]
	if strings.Contains(string(it.Diff), "41") || !strings.Contains(string(it.Diff), `"redacted":"mask"`) || !it.DiffRedacted || it.DiffSalt != "" {
		t.Fatalf("admin's entry: diff %s redacted %v salt %q, want cost masked, redacted, no salt", it.Diff, it.DiffRedacted, it.DiffSalt)
	}

	// Let the admins read every desk field: the stored diff comes back.
	both := authz.Subject{Groups: []uuid.UUID{m.desk, m.a.adminGroup}}
	m.publishMasks(t, []authz.Mask{{Field: "cost", Read: both, Write: both}, {Field: "triage", Read: both, Write: both}})
	_, page = m.auditGet(t, "admin-a", "?domain=record&subject_type=record&subject_id="+r.ID)
	if it := page.Items[0]; !strings.Contains(string(it.Diff), "41") || it.DiffRedacted || it.DiffSalt == "" {
		t.Fatalf("allowed reader's entry: diff %s redacted %v salt %q, want the stored diff with its salt", it.Diff, it.DiffRedacted, it.DiffSalt)
	}
}

// PR #523 finding 1: a unique value held by a record the caller cannot read
// is a 409 that names the property only, never the other record's id or
// number. (That some record holds the value is accepted under D-3-22.)
//
// covers: createRecord 409
func TestUniqueViolationDoesNotIdentifyAnUnreadableRecord(t *testing.T) {
	m := newMaskEnv(t)
	// The admin reads and writes every job; the user only its own.
	rules := []authz.Rule{
		acltest.AllowUsers(m.a.admin.PrincipalID),
		{ID: "own", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{m.a.user.PrincipalID}},
			Where: []authz.Cond{{Column: "requester_id", Op: authz.CondEq, Caller: authz.CallerSelf}}},
	}
	if err := tenancy.InTenantTx(t.Context(), m.pool, m.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "job", rules, nil)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	other := m.mustCreateRec(t, "admin-a", "job", "mk-u1", `{"short_description": "s", "serial": "SN-1"}`)
	if rec := m.do(t, http.MethodGet, "/api/v1/records/job/"+other.ID, "user-a", "", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("control: the user's GET of the admin's record = %d, want 404", rec.Code)
	}

	rec := m.createRec(t, "user-a", "job", "mk-u2", fmt.Sprintf(`{"short_description": "s", "serial": "SN-1", "requester": %q}`, m.a.user.PrincipalID))
	var p maskProblem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || rec.Code != http.StatusConflict || p.Code != "unique_violation" ||
		len(p.Errors) != 1 || p.Errors[0].Pointer != "/data/serial" {
		t.Fatalf("create with a taken serial = %d %s, want 409 unique_violation at /data/serial only", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	for _, leak := range []string{other.ID, other.Number, strings.ReplaceAll(other.ID, "-", "")} {
		if strings.Contains(body, leak) {
			t.Fatalf("409 body names the unreadable record (%s): %s", leak, body)
		}
	}
	for _, h := range []string{"Location", "ETag"} {
		if v := rec.Header().Get(h); v != "" {
			t.Fatalf("409 carries %s %q", h, v)
		}
	}
}

func (m *maskEnv) tenantRows(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := m.super.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE tenant_id = $1", m.a.id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TICK-633 B1: a PATCH is no oracle for a field the caller cannot read. Its
// current value, a wrong guess and null all get the same 422
// field_not_writable; a masked state gets it too, never illegal_transition,
// whatever the hidden current state. Nothing is written in any case.
//
// covers: patchRecord 422
func TestPatchIsNoOracleForAMaskedField(t *testing.T) {
	m := newMaskEnv(t)
	desk := authz.Subject{Groups: []uuid.UUID{m.desk}}
	m.publishMasks(t, append(m.deskMasks(), authz.Mask{Field: "state", Read: desk, Write: desk}))
	r := m.mustCreateRec(t, "lookalike-a", "job", "mk-o1", `{"short_description": "s", "cost": 500}`)
	// The agent closes it: closed -> new is not a transition the model has.
	if rec := m.patch(t, "lookalike-a", "job", r.ID, `W/"1"`, `{"state": "closed"}`); rec.Code != http.StatusOK {
		t.Fatalf("the agent's close = %d: %s", rec.Code, rec.Body)
	}
	records, audits, outbox := m.recordCount(t, "a"), m.tenantAudits(t), m.tenantRows(t, "outbox")

	for name, c := range map[string]struct{ body, pointer string }{
		"cost, its stored value":             {`{"data": {"cost": 500}}`, "/data/cost"},
		"cost, a wrong guess":                {`{"data": {"cost": 7}}`, "/data/cost"},
		"cost, null":                         {`{"data": {"cost": null}}`, "/data/cost"},
		"triage, null on its stored default": {`{"data": {"triage": null}}`, "/data/triage"},
		"state, its stored value":            {`{"state": "closed"}`, "/data/state"},
		"state, an illegal transition":       {`{"state": "new"}`, "/data/state"},
		"state, inside data":                 {`{"data": {"state": "new"}}`, "/data/state"},
	} {
		wantNotWritable(t, name, m.patch(t, "user-a", "job", r.ID, `W/"2"`, c.body), c.pointer)
	}
	if got := m.recordCount(t, "a"); got != records {
		t.Errorf("records = %d, want %d", got, records)
	}
	if got := m.tenantAudits(t); got != audits {
		t.Errorf("audit entries = %d, want %d: a refused PATCH wrote", got, audits)
	}
	if got := m.tenantRows(t, "outbox"); got != outbox {
		t.Errorf("outbox rows = %d, want %d: a refused PATCH wrote", got, outbox)
	}
	rec := m.do(t, http.MethodGet, "/api/v1/records/job/"+r.ID, "lookalike-a", "", "")
	if d := recordData(t, rec); rec.Header().Get("ETag") != `W/"2"` || d["cost"] != float64(500) || d["state"] != "closed" {
		t.Fatalf("record after the refusals: ETag %q data %v, want version 2, cost 500, closed", rec.Header().Get("ETag"), d)
	}
	// The requester can still patch what it can read, and the body it was
	// given (no masked field) goes back unchanged.
	if rec := m.patch(t, "user-a", "job", r.ID, `W/"2"`, `{"data": {"short_description": "s"}}`); rec.Code != http.StatusOK {
		t.Fatalf("the requester's patch of an open field = %d: %s", rec.Code, rec.Body)
	}
}
