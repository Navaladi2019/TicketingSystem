//go:build integration

// TICK-1115 (ADR-010 D8): with the org_unit_id constraint in place, the API
// still answers a definition with that property from Compile, and writes
// nothing. Helpers are in entitytypes_integration_test.go.
//
// TICK-1125: the operator's repair, then the API over the repaired type. The
// repair's SQL has its own tests, on the release before S31 and through the
// real upgrade, in internal/db/dbtest.

package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/db/repairtest"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// covers: createEntityType 422
// covers: publishEntityTypeVersion 422
func TestTICK1115OrgUnitIDPropertyIsInvalidDefinition(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	const pointer = "/definition/schema/properties/org_unit_id"
	reserved := strings.Replace(etSchema, `"properties": {`, `"properties": {"org_unit_id": {"type": "string"},`, 1)
	for _, c := range []struct{ name, path, body string }{
		{"publish", etVersionsPath, fmt.Sprintf(`{"definition": {"schema": %s, %s}}`, reserved, etDefTail)},
		{"create", "/api/v1/entity-types", fmt.Sprintf(`{"key": "request", "name": "Request", "number_prefix": "REQ", "definition": {"schema": %s, %s}}`,
			reserved, etDefTail)},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := wantProblem(t, e.do(t, http.MethodPost, c.path, "admin-a", "k-1115-"+c.name, c.body), http.StatusUnprocessableEntity, problem.InvalidDefinition)
			if got := pointers(p); len(got) != 1 || got[0] != pointer {
				t.Errorf("pointers = %v, want [%s]", got, pointer)
			}
		})
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions after the refusals, want 1", n)
	}
	var types int
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", e.a.id).Scan(&types); err != nil || types != 1 {
		t.Errorf("%d types (err %v), want 1", types, err)
	}
}

// ouStandIn is the name the API accepts; ouStore then turns it into
// org_unit_id in the stored rows, as a release before S31 stored them.
const ouStandIn = "xrg_unit_id"

const ouSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["state"],
  "properties": {
    "state": {"type": "string", "enum": ["new", "closed"], "default": "new"},
    "xrg_unit_id": {"type": "string", "x-nava-unique": true}
  }
}`

const ouMappedSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["state"],
  "properties": {
    "state": {"type": "string", "enum": ["new", "closed"], "default": "new"},
    "xrg_unit_id": {"type": "string", "format": "uuid"}
  }
}`

const ouDefinitionTail = `"states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}`

func ouTypeBody(key, prefix, schema, fixed string) string {
	return fmt.Sprintf(`{"key": %q, "name": %q, "number_prefix": %q, "definition": {"schema": %s, "fixed_columns": %s, `, key, key, prefix, schema, fixed) +
		ouDefinitionTail + "}"
}

func (e *etEnv) superExec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// ouStore drops the guard and renames the stand-in in every stored row.
func (e *etEnv) ouStore(t *testing.T) {
	t.Helper()
	e.superExec(t, "ALTER TABLE entity_definition_versions DROP CONSTRAINT entity_definition_versions_org_unit_id_reserved")
	e.superExec(t, "SET session_replication_role = replica")
	e.superExec(t, `UPDATE entity_definition_versions SET schema = replace(schema::text, $1::text, 'org_unit_id')::json,
		schema_sha256 = sha256(convert_to(replace(schema::text, $1::text, 'org_unit_id'), 'UTF8')),
		fixed_columns = replace(fixed_columns::text, $1::text, 'org_unit_id')::jsonb`, ouStandIn)
	e.superExec(t, "UPDATE records SET data = replace(data::text, $1::text, 'org_unit_id')::jsonb", ouStandIn)
	e.superExec(t, "ALTER TABLE audit_log DISABLE TRIGGER audit_log_guard_update")
	e.superExec(t, `UPDATE audit_log SET diff = replace(diff::text, $1::text, 'org_unit_id')::json,
		diff_sha256 = sha256(diff_salt || convert_to(replace(diff::text, $1::text, 'org_unit_id'), 'UTF8')) WHERE diff IS NOT NULL`, ouStandIn)
	e.superExec(t, "ALTER TABLE audit_log ENABLE TRIGGER audit_log_guard_update")
	e.superExec(t, "SET session_replication_role = DEFAULT")
}

// ouGuard adds the constraint back with the migration's own statements: they
// fail while a refused version is stored.
func (e *etEnv) ouGuard(t *testing.T) error {
	t.Helper()
	b, err := fs.ReadFile(db.Migrations(), "20261120235900_entity_versions_org_unit_id_reserved.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, up, _ := strings.Cut(string(b), "-- +goose Up")
	up, _, _ = strings.Cut(up, "-- +goose Down")
	_, err = e.super.Exec(t.Context(), up)
	return err
}

// ouRepair runs the documented repair of one of tenant A's types.
func (e *etEnv) ouRepair(t *testing.T, key string) {
	t.Helper()
	var typ uuid.UUID
	if err := e.super.QueryRow(t.Context(), "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2", e.a.id, key).Scan(&typ); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../docs/data/entity-versions-org-unit-id-reserved.md")
	if err != nil {
		t.Fatal(err)
	}
	markdown := strings.ReplaceAll(string(b), "\r\n", "\n")
	one, err := repairtest.Block(markdown, repairtest.PasteOne)
	if err != nil {
		t.Fatal(err)
	}
	if one, err = repairtest.WithInputs(one, repairtest.Inputs{
		TenantID: e.a.id.String(), TypeID: typ.String(),
		NewName: "old_unit_id", SourceRef: "CHG-1125/run-1", RecordLimit: "100000",
	}); err != nil {
		t.Fatal(err)
	}
	two, err := repairtest.Block(markdown, repairtest.PasteTwo)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.ConnectConfig(t.Context(), e.super.Config())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(t.Context())) }()
	for _, sql := range []string{one, two} {
		out, err := repairtest.Paste(t.Context(), conn, sql, false)
		if err != nil || out.Err != nil {
			t.Fatalf("repair of %s: %v, %v", key, out.Err, err)
		}
	}
}

// Tests 1, 5, 9, 10 and 22 of TICK-1125.
//
// covers: getEntityType 200
// covers: publishEntityTypeVersion 201
// covers: createRecord 201 409
// covers: getRecord 200
// covers: patchRecord 200 412
// covers: getRecordHistory 200
// covers: listAuditEntries 200
// covers: exportAuditEntries 200
func TestTICK1125APIAfterTheRepair(t *testing.T) {
	x := newExportEnv(t)
	for i, body := range []string{
		ouTypeBody("ticket", "TKT", ouSchema, `{"state": "state"}`),
		ouTypeBody("mapped", "MAP", ouMappedSchema, `{"state": "state", "xrg_unit_id": "requester_id"}`),
	} {
		if rec := x.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", fmt.Sprintf("ou-type-%d", i), body); rec.Code != http.StatusCreated {
			t.Fatalf("create type = %d: %s", rec.Code, rec.Body)
		}
	}
	const value = "unit-value-1125"
	r := x.mustCreateRec(t, "user-a", "ticket", "ou-1", `{"xrg_unit_id": "`+value+`"}`)
	big := x.mustCreateRec(t, "user-a", "ticket", "ou-big", `{"xrg_unit_id": "`+strings.Repeat("x", 70000)+`"}`)
	mapped := x.mustCreateRec(t, "user-a", "mapped", "ou-mapped", `{"xrg_unit_id": "`+x.a.user.PrincipalID.String()+`"}`)
	x.ouStore(t)
	if err := x.ouGuard(t); err == nil {
		t.Fatal("control: the guard accepts the refused versions")
	}
	x.sealAll(t)

	x.ouRepair(t, "ticket")
	x.ouRepair(t, "mapped")
	if err := x.ouGuard(t); err != nil {
		t.Fatalf("the guard after the repair: %v", err)
	}

	// 10: the type is read.
	rec := x.do(t, http.MethodGet, "/api/v1/entity-types/ticket", "user-a", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"old_unit_id"`) || strings.Contains(rec.Body.String(), `"org_unit_id"`) {
		t.Fatalf("read type = %d: %s", rec.Code, rec.Body)
	}
	// 10: its records are read.
	rec = x.do(t, http.MethodGet, "/api/v1/records/ticket/"+r.ID, "user-a", "", "")
	if d := recordData(t, rec); rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` || d["old_unit_id"] != value || d["org_unit_id"] != nil {
		t.Fatalf("read record = %d %s: %s", rec.Code, rec.Header().Get("ETag"), rec.Body)
	}
	rec = x.do(t, http.MethodGet, "/api/v1/records/mapped/"+mapped.ID, "user-a", "", "")
	if d := recordData(t, rec); d["old_unit_id"] != x.a.user.PrincipalID.String() {
		t.Errorf("read the mapped record: %s", rec.Body)
	}
	// 22: the ETag from before the repair conflicts.
	if rec := x.patch(t, "user-a", "ticket", r.ID, `W/"1"`, `{"data": {"state": "closed"}}`); rec.Code != http.StatusPreconditionFailed {
		t.Errorf("patch with the ETag from before = %d: %s", rec.Code, rec.Body)
	}
	// 10: records are updated and created.
	if rec := x.patch(t, "user-a", "ticket", r.ID, `W/"2"`, `{"data": {"old_unit_id": "`+value+`-b"}}`); rec.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
	}
	if rec := x.patch(t, "user-a", "mapped", mapped.ID, `W/"2"`, `{"data": {"old_unit_id": "`+x.a.admin.PrincipalID.String()+`"}}`); rec.Code != http.StatusOK {
		t.Errorf("patch the mapped record = %d: %s", rec.Code, rec.Body)
	}
	x.mustCreateRec(t, "user-a", "ticket", "ou-2", `{"old_unit_id": "fresh"}`)
	// 22: the property is still unique.
	if rec := x.createRec(t, "user-a", "ticket", "ou-3", `{"old_unit_id": "`+value+`-b"}`); rec.Code != http.StatusConflict ||
		patchProblemCode(t, rec) != "unique_violation" || !strings.Contains(rec.Body.String(), "/data/old_unit_id") {
		t.Errorf("a duplicate = %d: %s", rec.Code, rec.Body)
	}
	// 10: the next version is published.
	next := strings.Replace(strings.Replace(ouSchema, ouStandIn, "old_unit_id", 1), `"properties": {`, `"properties": {"note": {"type": "string"},`, 1)
	rec = x.do(t, http.MethodPost, "/api/v1/entity-types/ticket/versions", "admin-a", "ou-publish",
		`{"definition": {"schema": `+next+`, "fixed_columns": {"state": "state"}, `+ouDefinitionTail+`}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"version":2`) {
		t.Fatalf("publish = %d: %s", rec.Code, rec.Body)
	}
	// 9: the history replays across the repair and the later update.
	rec = x.do(t, http.MethodGet, "/api/v1/records/ticket/"+r.ID+"/history", "user-a", "", "")
	var history struct {
		Items []struct {
			Before, After map[string]any
			Changed       []string
		}
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &history); err != nil || rec.Code != http.StatusOK || len(history.Items) != 3 {
		t.Fatalf("history = %d: %s", rec.Code, rec.Body)
	}
	if h := history.Items; h[0].Before["old_unit_id"] != value || h[0].After["old_unit_id"] != value+"-b" || h[1].After["old_unit_id"] != value ||
		!slices.Equal(h[1].Changed, []string{"/data/old_unit_id"}) {
		t.Errorf("history: %s", rec.Body)
	}

	// 1: a mask on the new name hides the value in the repair's own entry.
	repairEntry := func() (diff string, redacted bool) {
		t.Helper()
		rec, page := x.auditGet(t, "admin-a", "?domain=record&subject_type=record&subject_id="+r.ID)
		if rec.Code != http.StatusOK || len(page.Items) != 3 {
			t.Fatalf("audit list = %d, %d items", rec.Code, len(page.Items))
		}
		it := page.Items[1]
		return string(it.Diff), it.DiffRedacted
	}
	if diff, redacted := repairEntry(); redacted || !strings.Contains(diff, value) || !strings.Contains(diff, `"/org_unit_id"`) {
		t.Fatalf("control: the repair's entry without a mask: %s", diff)
	}
	if err := tenancy.InTenantTx(t.Context(), x.pool, x.a.id, func(tx pgx.Tx) error {
		only := authz.Subject{Users: []uuid.UUID{x.a.user.PrincipalID}}
		_, err := acltest.PublishPolicy(t.Context(), tx, "ticket", []authz.Rule{acltest.AllowUsers(x.a.admin.PrincipalID, x.a.user.PrincipalID)},
			[]authz.Mask{{Field: "old_unit_id", Read: only, Write: only}})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if diff, redacted := repairEntry(); !redacted || strings.Contains(diff, value) {
		t.Errorf("the repair's entry under a mask on the new name: redacted %v, %s", redacted, diff)
	}

	// 5: the sealer and the verifier pass over a truncated entry and a
	// fixed_columns entry.
	var truncated, mappedEntries int
	if err := x.super.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE diff_truncated AND subject_id = $2),
		count(*) FILTER (WHERE domain = 'config' AND version_after IS NULL AND diff::text LIKE '%/fixed_columns/old_unit_id%')
		FROM audit_log WHERE tenant_id = $1 AND actor_system = 'operator'`, x.a.id, big.ID).Scan(&truncated, &mappedEntries); err != nil || truncated != 1 || mappedEntries != 1 {
		t.Fatalf("control: %d truncated and %d fixed_columns entries of the repair (%v)", truncated, mappedEntries, err)
	}
	x.sealAll(t)
	var unsealed int
	if err := x.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NULL", x.a.id).Scan(&unsealed); err != nil || unsealed != 0 {
		t.Errorf("%d entries not sealed (%v)", unsealed, err)
	}
	rec = x.export(t, "admin-a", "")
	if rep := report(t, rec.Body.Bytes()); rec.Code != http.StatusOK || rep.Outcome != verify.OutcomePass {
		t.Errorf("verify = %d %s, %+v", rec.Code, rep.Outcome, rep.Failure)
	}
}
