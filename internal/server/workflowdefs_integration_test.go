//go:build integration

// P3-41 (TICK-710): the workflow-definition endpoints and the database
// DefinitionStore against a real PostgreSQL, through the real handler (authn,
// ratelimit, idempotency, strict server). Helpers are in
// entitytypes_integration_test.go and records_integration_test.go.

package server_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflowdefs"
)

// wfBody is a valid definition on entity whose update_record sets field. The
// bytes are indented, so a re-encoding would change them.
func wfBody(entity, field string) string {
	return fmt.Sprintf(`{
  "format": "nava.workflow/v1",
  "trigger": {"on": "record.created", "entity": %q},
  "start": "wait",
  "steps": [
    {"key": "wait", "type": "timer", "duration": "PT1H", "next": "mark"},
    {"key": "mark", "type": "update_record", "set": {%q: "x"}, "next": "done"},
    {"key": "done", "type": "end"}
  ]
}`, entity, field)
}

// wfBodyV2 adds the step notify between wait and mark.
func wfBodyV2(entity, field string) string {
	return fmt.Sprintf(`{
  "format": "nava.workflow/v1",
  "trigger": {"on": "record.created", "entity": %q},
  "start": "wait",
  "steps": [
    {"key": "wait", "type": "timer", "duration": "PT1H", "next": "notify"},
    {"key": "notify", "type": "update_record", "set": {%q: "n"}, "next": "mark"},
    {"key": "mark", "type": "update_record", "set": {%q: "x"}, "next": "done"},
    {"key": "done", "type": "end"}
  ]
}`, entity, field, field)
}

func wfBodyWithSignal() string {
	return `{"format": "nava.workflow/v1", "trigger": {"on": "record.created", "entity": "incident"}, "start": "gate",
  "steps": [{"key": "gate", "type": "wait_signal", "name": "go", "timeout": "PT1H", "on_timeout": "done", "next": "done"}, {"key": "done", "type": "end"}]}`
}

type wfVersion struct {
	Key        string               `json:"key"`
	Entity     string               `json:"entity"`
	Version    int                  `json:"version"`
	Current    bool                 `json:"current"`
	VersionID  uuid.UUID            `json:"version_id"`
	Body       json.RawMessage      `json:"body"`
	BodySHA256 string               `json:"body_sha256"`
	StepIDs    map[string]uuid.UUID `json:"step_ids"`
}

func wfParse(t *testing.T, b []byte) wfVersion {
	t.Helper()
	var v wfVersion
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("not a version: %v: %s", err, b)
	}
	return v
}

// newWFEnv is an etEnv with an "incident" entity type in both tenants.
func newWFEnv(t *testing.T) *etEnv {
	t.Helper()
	e := newETEnv(t)
	for _, tc := range []struct{ token, key string }{{"admin-a", "ka"}, {"admin-b", "kb"}} {
		if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", tc.token, tc.key, etBody("incident", "INC")); rec.Code != http.StatusCreated {
			t.Fatalf("create incident (%s) = %d: %s", tc.token, rec.Code, rec.Body)
		}
	}
	return e
}

func (e *etEnv) wfPublish(t *testing.T, token, key, idem, body string) (int, []byte) {
	t.Helper()
	rec := e.do(t, http.MethodPost, "/api/v1/workflow-definitions/"+key+"/versions", token, idem, body)
	return rec.Code, rec.Body.Bytes()
}

func mustPublishID(t *testing.T, e *etEnv, token, key, body string) uuid.UUID {
	t.Helper()
	code, b := e.wfPublish(t, token, key, "id-"+token+key, body)
	if code != http.StatusCreated {
		t.Fatalf("publish = %d: %s", code, b)
	}
	return wfParse(t, b).VersionID
}

// covers: publishWorkflowDefinitionVersion 201
// covers: getWorkflowDefinition 200
func TestWorkflowDefinitionPublishKeepsStepIDsAndLeavesV1Untouched(t *testing.T) {
	e := newWFEnv(t)
	body1 := wfBody("incident", "short_description")
	code, b := e.wfPublish(t, "admin-a", "triage", "p1", body1)
	if code != http.StatusCreated {
		t.Fatalf("publish v1 = %d: %s", code, b)
	}
	v1 := wfParse(t, b)
	if v1.Version != 1 || !v1.Current || v1.Entity != "incident" || len(v1.StepIDs) != 3 {
		t.Fatalf("v1 = %+v", v1)
	}
	if string(v1.Body) != body1 || v1.BodySHA256 != etSHA(body1) {
		t.Errorf("v1 body is not the exact request bytes (sha %s)", v1.BodySHA256)
	}
	snapshot := func() string {
		var s string
		if err := e.super.QueryRow(t.Context(), `SELECT v.body::text || v.body_sha256::text || v.step_ids::text || v.id::text || v.published_at::text
			FROM workflow_definition_versions v WHERE v.tenant_id = $1 AND v.version = 1`, e.a.id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := snapshot()

	code, b = e.wfPublish(t, "admin-a", "triage", "p2", wfBodyV2("incident", "short_description"))
	if code != http.StatusCreated {
		t.Fatalf("publish v2 = %d: %s", code, b)
	}
	v2 := wfParse(t, b)
	if v2.Version != 2 || !v2.Current || v2.VersionID == v1.VersionID {
		t.Fatalf("v2 = %+v", v2)
	}
	for _, k := range []string{"wait", "mark", "done"} {
		if v2.StepIDs[k] != v1.StepIDs[k] {
			t.Errorf("step %q changed id: %s -> %s", k, v1.StepIDs[k], v2.StepIDs[k])
		}
	}
	for _, id := range v1.StepIDs {
		if v2.StepIDs["notify"] == id {
			t.Error("the new step reused an old id")
		}
	}
	if v2.StepIDs["notify"] == uuid.Nil {
		t.Error("the new step has no id")
	}
	if after := snapshot(); after != before {
		t.Errorf("v1's row changed:\nbefore %s\nafter  %s", before, after)
	}

	rec := e.do(t, http.MethodGet, "/api/v1/workflow-definitions/triage", "admin-a", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get = %d: %s", rec.Code, rec.Body)
	}
	got := wfParse(t, rec.Body.Bytes())
	if got.Version != 2 || !got.Current || got.VersionID != v2.VersionID || got.StepIDs["notify"] != v2.StepIDs["notify"] {
		t.Errorf("get = %+v, want v2", got)
	}
	var cur int
	if err := e.super.QueryRow(t.Context(), `SELECT current_version FROM workflow_definitions WHERE tenant_id = $1 AND key = 'triage'`, e.a.id).Scan(&cur); err != nil || cur != 2 {
		t.Errorf("current_version = %d, %v, want 2", cur, err)
	}
}

// covers: publishWorkflowDefinitionVersion 422
func TestWorkflowDefinitionInvalidBodiesAre422AndNameTheStep(t *testing.T) {
	e := newWFEnv(t)
	valid := wfBody("incident", "short_description")
	for i, tc := range []struct {
		name, body, pointer string
		absent              string // a value the problem must not echo
	}{
		{"a step id", strings.Replace(valid, `"key": "wait", `, `"key": "wait", "id": "00000000-0000-4000-8000-0000000000aa", `, 1), "/steps/wait/id", "0000000000aa"},
		{"a signal id", strings.Replace(wfBodyWithSignal(), `"name": "go"`, `"name": "go", "id": "00000000-0000-4000-8000-0000000000bb"`, 1), "/steps/gate/id", "0000000000bb"},
		{"a top-level step_ids", strings.Replace(valid, `"start"`, `"step_ids": {"wait": "00000000-0000-4000-8000-0000000000cc"}, "start"`, 1), "", "0000000000cc"},
		{"an unknown entity", wfBody("no_such_type", "short_description"), "/trigger/entity", "no_such_type"},
		{"a set on a field the schema lacks", wfBody("incident", "no_such_field"), "/steps/mark/set/no_such_field", ""},
		{"a field reference as a duration", strings.Replace(valid, `"PT1H"`, `"/secret_duration"`, 1), "/steps/wait/duration", "secret_duration"},
		{"an empty object", `{}`, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, b := e.wfPublish(t, "admin-a", "bad_one", fmt.Sprintf("bad-%d", i), tc.body)
			if code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d: %s", code, b)
			}
			var p etProblem
			if err := json.Unmarshal(b, &p); err != nil || p.Code != problem.WorkflowDefinitionInvalid {
				t.Fatalf("problem = %+v, %v: %s", p, err, b)
			}
			if tc.pointer != "" && !slices.Contains(pointers(p), tc.pointer) {
				t.Errorf("pointers %v lack %q", pointers(p), tc.pointer)
			}
			if tc.absent != "" && strings.Contains(string(b), tc.absent) {
				t.Errorf("the problem echoes %q: %s", tc.absent, b)
			}
		})
	}
	var n int
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM workflow_definitions WHERE tenant_id = $1`, e.a.id).Scan(&n); err != nil || n != 0 {
		t.Errorf("%d definitions after only invalid publishes, %v", n, err)
	}
	// A malformed key is a 422 on publish and a 404 on read.
	if code, _ := e.wfPublish(t, "admin-a", "Bad-Key", "k1", valid); code != http.StatusUnprocessableEntity {
		t.Errorf("publish with a malformed key = %d, want 422", code)
	}
	if rec := e.do(t, http.MethodGet, "/api/v1/workflow-definitions/Bad-Key", "admin-a", "", ""); rec.Code != http.StatusNotFound {
		t.Errorf("get with a malformed key = %d, want 404", rec.Code)
	}
}

// A body that is not JSON never reaches the handler: the contract validator
// answers 400 validation_failed at /body, not the 422 of a definition that
// parses but is not publishable.
// covers: publishWorkflowDefinitionVersion 400
func TestWorkflowDefinitionBodyThatIsNotJSONIs400(t *testing.T) {
	e := newWFEnv(t)
	code, b := e.wfPublish(t, "admin-a", "bad_one", "notjson-1", `{"format":`)
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", code, b)
	}
	var p etProblem
	if err := json.Unmarshal(b, &p); err != nil || p.Code != problem.ValidationFailed {
		t.Fatalf("problem = %+v, %v: %s", p, err, b)
	}
	if !slices.Contains(pointers(p), "/body") {
		t.Errorf("pointers %v lack /body", pointers(p))
	}
}

// A definition belongs to one trigger entity for life.
// covers: publishWorkflowDefinitionVersion 422
func TestWorkflowDefinitionCannotChangeItsTriggerEntity(t *testing.T) {
	e := newWFEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "kt", etBody("task", "TSK")); rec.Code != http.StatusCreated {
		t.Fatalf("create task = %d: %s", rec.Code, rec.Body)
	}
	if code, b := e.wfPublish(t, "admin-a", "triage", "c1", wfBody("incident", "short_description")); code != http.StatusCreated {
		t.Fatalf("v1 = %d: %s", code, b)
	}
	code, b := e.wfPublish(t, "admin-a", "triage", "c2", wfBody("task", "short_description"))
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("v2 on another entity = %d: %s", code, b)
	}
}

// covers: publishWorkflowDefinitionVersion 403
// covers: getWorkflowDefinition 403
func TestWorkflowDefinitionNonAdminIs403(t *testing.T) {
	e := newWFEnv(t)
	if code, b := e.wfPublish(t, "admin-a", "triage", "n0", wfBody("incident", "short_description")); code != http.StatusCreated {
		t.Fatalf("seed = %d: %s", code, b)
	}
	for _, token := range []string{"user-a", "lookalike-a"} {
		code, b := e.wfPublish(t, token, "triage", "n-"+token, wfBody("incident", "short_description"))
		if code != http.StatusForbidden {
			t.Errorf("%s publish = %d: %s", token, code, b)
		}
		if rec := e.do(t, http.MethodGet, "/api/v1/workflow-definitions/triage", token, "", ""); rec.Code != http.StatusForbidden {
			t.Errorf("%s get = %d: %s", token, rec.Code, rec.Body)
		}
	}
	var n int
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM workflow_definition_versions WHERE tenant_id = $1`, e.a.id).Scan(&n); err != nil || n != 1 {
		t.Errorf("%d versions, %v, want the 1 the admin published", n, err)
	}
}

// Tenant B can neither read nor change tenant A's definition: a read is the
// same 404 as a missing key, byte for byte, and B's publish of the same key
// makes B's own definition (a 409 or a 403 would be a cross-tenant oracle).
// covers: getWorkflowDefinition 404
// covers: publishWorkflowDefinitionVersion 201
func TestWorkflowDefinitionIsolation(t *testing.T) {
	e := newWFEnv(t)
	bodyA := wfBody("incident", "short_description")
	if code, b := e.wfPublish(t, "admin-a", "triage", "i1", bodyA); code != http.StatusCreated {
		t.Fatalf("A publish = %d: %s", code, b)
	}
	foreign := e.do(t, http.MethodGet, "/api/v1/workflow-definitions/triage", "admin-b", "", "")
	missing := e.do(t, http.MethodGet, "/api/v1/workflow-definitions/no_such_flow", "admin-b", "", "")
	if foreign.Code != http.StatusNotFound || missing.Code != http.StatusNotFound {
		t.Fatalf("foreign = %d, missing = %d, want 404 and 404: %s", foreign.Code, missing.Code, foreign.Body)
	}
	if foreign.Body.String() != missing.Body.String() {
		t.Errorf("foreign 404 differs from the missing-key 404:\n%s\n%s", foreign.Body, missing.Body)
	}
	if strings.Contains(foreign.Body.String(), "instance") {
		t.Errorf("the 404 carries an instance: %s", foreign.Body)
	}

	bodyB := wfBodyV2("incident", "short_description")
	code, b := e.wfPublish(t, "admin-b", "triage", "i2", bodyB)
	if code != http.StatusCreated {
		t.Fatalf("B publish = %d: %s", code, b)
	}
	if v := wfParse(t, b); v.Version != 1 {
		t.Errorf("B's publish of A's key made version %d, want its own version 1", v.Version)
	}
	for _, tc := range []struct{ token, body string }{{"admin-a", bodyA}, {"admin-b", bodyB}} {
		rec := e.do(t, http.MethodGet, "/api/v1/workflow-definitions/triage", tc.token, "", "")
		if v := wfParse(t, rec.Body.Bytes()); v.Version != 1 || string(v.Body) != tc.body {
			t.Errorf("%s reads %+v, want its own version 1", tc.token, v)
		}
	}
}

// Fifty admins publish to one new key at once. The versions are 1..50: no gap,
// no duplicate, none refused, and the key's current_version is 50.
// covers: publishWorkflowDefinitionVersion 201
func TestWorkflowDefinitionFiftyConcurrentPublishesNumberGaplessly(t *testing.T) {
	const n = 50
	e := newWFEnv(t)
	type result struct {
		code int
		body []byte
	}
	results := make([]result, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rec := e.do(t, http.MethodPost, "/api/v1/workflow-definitions/burst/versions", "admin-a", fmt.Sprintf("burst-%d", i), wfBody("incident", "short_description"))
			results[i] = result{rec.Code, rec.Body.Bytes()}
		}()
	}
	close(start)
	wg.Wait()
	seen := map[int]bool{}
	for i, r := range results {
		if r.code != http.StatusCreated {
			t.Errorf("publish %d = %d: %s", i, r.code, r.body)
			continue
		}
		v := wfParse(t, r.body)
		if seen[v.Version] {
			t.Errorf("version %d was handed out twice", v.Version)
		}
		seen[v.Version] = true
	}
	for v := 1; v <= n; v++ {
		if !seen[v] {
			t.Errorf("version %d was never handed out", v)
		}
	}
	rows, err := e.super.Query(t.Context(), `SELECT v.version FROM workflow_definition_versions v JOIN workflow_definitions d
		ON d.tenant_id = v.tenant_id AND d.id = v.definition_id WHERE d.tenant_id = $1 AND d.key = 'burst' ORDER BY v.version`, e.a.id)
	if err != nil {
		t.Fatal(err)
	}
	var stored []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		stored = append(stored, v)
	}
	rows.Close()
	if len(stored) != n || stored[0] != 1 || stored[n-1] != n {
		t.Errorf("stored versions %v, want 1..%d", stored, n)
	}
	var cur, defs int
	if err := e.super.QueryRow(t.Context(), `SELECT max(current_version), count(*) FROM workflow_definitions WHERE tenant_id = $1 AND key = 'burst'`, e.a.id).Scan(&cur, &defs); err != nil || cur != n || defs != 1 {
		t.Errorf("current_version %d over %d definitions, %v, want %d over 1", cur, defs, err, n)
	}
}

// DefinitionStore.Load reads under the header tenant: another tenant's
// version id is ErrNotFound, so is an id that does not exist, and so is a
// tenant argument that is not the header's.
func TestWorkflowDefinitionStoreLoadIsTenantScoped(t *testing.T) {
	e := newWFEnv(t)
	idA := mustPublishID(t, e, "admin-a", "triage", wfBody("incident", "short_description"))
	idB := mustPublishID(t, e, "admin-b", "triage", wfBodyV2("incident", "short_description"))

	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("nava_worker", etPassword("nava_worker"))
	u.Path = "/" + e.pool.Config().ConnConfig.Database
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		t.Fatal(err)
	}
	worker, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(worker.Close)
	store := workflowdefs.Reader{Pool: worker}

	v, err := store.Load(workerguard.ContextWithTenant(t.Context(), e.a.id), e.a.id, idA)
	if err != nil {
		t.Fatalf("tenant A loads its own version: %v", err)
	}
	if v.Definition == nil || len(v.StepIDs) != 3 {
		t.Errorf("loaded %+v", v)
	}
	if _, ok := v.Definition.Step("mark"); !ok {
		t.Error("the loaded definition lacks its step")
	}
	if v, err = store.Load(workerguard.ContextWithTenant(t.Context(), e.b.id), e.b.id, idB); err != nil || len(v.StepIDs) != 4 {
		t.Errorf("tenant B loads its own version: %v %+v", err, v)
	}

	for _, tc := range []struct {
		name            string
		ctxTenant       uuid.UUID
		tenant, version uuid.UUID
	}{
		{"B asks for A's version id", e.b.id, e.b.id, idA},
		{"A asks for B's version id", e.a.id, e.a.id, idB},
		{"B's header with A as the argument", e.b.id, e.a.id, idA},
		{"A's header with B as the argument", e.a.id, e.b.id, idA},
		{"an id that does not exist", e.a.id, e.a.id, uuid.New()},
	} {
		if _, err := store.Load(workerguard.ContextWithTenant(t.Context(), tc.ctxTenant), tc.tenant, tc.version); !errors.Is(err, interp.ErrNotFound) {
			t.Errorf("%s: err = %v, want interp.ErrNotFound", tc.name, err)
		}
	}
	// No verified tenant on the context: nothing is read.
	if _, err := store.Load(t.Context(), e.a.id, idA); !errors.Is(err, interp.ErrNotFound) {
		t.Errorf("no header tenant: err = %v, want interp.ErrNotFound", err)
	}
}

// covers: publishWorkflowDefinitionVersion 400 409
func TestWorkflowDefinitionPublishNeedsAnIdempotencyKeyAndRefusesItsReuse(t *testing.T) {
	e := newWFEnv(t)
	body := wfBody("incident", "short_description")
	if code, b := e.wfPublish(t, "admin-a", "triage", "", body); code != http.StatusBadRequest {
		t.Errorf("no Idempotency-Key = %d, want 400: %s", code, b)
	}
	if code, b := e.wfPublish(t, "admin-a", "triage", "same", body); code != http.StatusCreated {
		t.Fatalf("first = %d: %s", code, b)
	}
	if code, b := e.wfPublish(t, "admin-a", "triage", "same", wfBodyV2("incident", "short_description")); code != http.StatusConflict {
		t.Errorf("the key with another body = %d, want 409: %s", code, b)
	}
	var n int
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM workflow_definition_versions WHERE tenant_id = $1`, e.a.id).Scan(&n); err != nil || n != 1 {
		t.Errorf("%d versions, %v, want 1", n, err)
	}
}
