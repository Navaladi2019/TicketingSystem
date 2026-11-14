//go:build integration

// P3-36g (TICK-660): numbering_mode in the entity-type admin API, against a
// real PostgreSQL. Helpers are in entitytypes_integration_test.go.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func etModeCreateBody(mode string) string {
	b := etBody("incident", "INC")
	return strings.Replace(b, `"number_prefix": "INC",`, fmt.Sprintf(`"number_prefix": "INC", "numbering_mode": %s,`, mode), 1)
}

func etModeOf(t *testing.T, e *etEnv, token string) string {
	t.Helper()
	rec := e.do(t, http.MethodGet, "/api/v1/entity-types/incident", token, "", "")
	var v struct {
		Mode string `json:"numbering_mode"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("get = %d %v: %s", rec.Code, err, rec.Body)
	}
	return v.Mode
}

func etStoredMode(t *testing.T, e *etEnv) string {
	t.Helper()
	var m string
	if err := e.super.QueryRow(t.Context(), "SELECT numbering_mode FROM entity_definitions WHERE tenant_id = $1 AND key = 'incident'", e.a.id).Scan(&m); err != nil {
		t.Fatal(err)
	}
	return m
}

// covers: createEntityType 201
// covers: getEntityType 200
// covers: listEntityTypes 200
func TestNumberingModeDefaultsToGapsAndIsListed(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	if got := etModeOf(t, e, "user-a"); got != "gaps" {
		t.Errorf("default mode = %q, want gaps", got)
	}
	rec := e.do(t, http.MethodGet, "/api/v1/entity-types", "user-a", "", "")
	if !strings.Contains(rec.Body.String(), `"numbering_mode":"gaps"`) {
		t.Errorf("list lacks numbering_mode: %s", rec.Body)
	}
}

// covers: createEntityType 201 422
func TestNumberingModeSetOnCreateAndInvalidIs422(t *testing.T) {
	e := newETEnv(t)
	for i, bad := range []string{`"sometimes"`, `""`, `5`, `null`} {
		rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", fmt.Sprintf("k-bad-%d", i), etModeCreateBody(bad))
		p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidDefinition)
		if got := pointers(p); len(got) != 1 || got[0] != "/numbering_mode" {
			t.Errorf("%s: pointers = %v, want [/numbering_mode]", bad, got)
		}
	}
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-ok", etModeCreateBody(`"gapless"`)); rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	if got, stored := etModeOf(t, e, "user-a"), etStoredMode(t, e); got != "gapless" || stored != "gapless" {
		t.Errorf("mode = %q, stored %q, want gapless", got, stored)
	}
}

// covers: publishEntityTypeVersion 201 422
func TestNumberingModeChangesOnUpdateWithoutAVersion(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-m1", `{"numbering_mode":"gapless"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"numbering_mode":"gapless"`) || !strings.Contains(rec.Body.String(), `"version":1`) {
		t.Fatalf("mode-only = %d: %s", rec.Code, rec.Body)
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("versions = %d, want 1: a mode-only change publishes none", n)
	}
	if got := etStoredMode(t, e); got != "gapless" {
		t.Errorf("stored = %q", got)
	}
	// Audited like other type edits: one config entry for the change.
	var n int
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND domain = 'config' AND subject_type = 'entity_definition' AND diff::text LIKE '%numbering_mode%'", e.a.id).Scan(&n); err != nil || n < 1 {
		t.Errorf("audit entries for numbering_mode = %d (err %v), want >= 1", n, err)
	}
	// With a definition: the mode is set and version 2 publishes.
	body := strings.Replace(etPublishBody(t, `"description": "a <b>&amp; tag"`, `"description": "changed"`), `{"definition"`, `{"numbering_mode":"gaps","definition"`, 1)
	rec = e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-m2", body)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"version":2`) || !strings.Contains(rec.Body.String(), `"numbering_mode":"gaps"`) {
		t.Fatalf("mode+definition = %d: %s", rec.Code, rec.Body)
	}
	// Invalid values write nothing.
	for i, bad := range []string{`"sometimes"`, `""`, `5`} {
		rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", fmt.Sprintf("k-bad-%d", i), `{"numbering_mode":`+bad+`}`)
		p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidDefinition)
		if got := pointers(p); len(got) != 1 || got[0] != "/numbering_mode" {
			t.Errorf("%s: pointers = %v", bad, got)
		}
	}
	if got := etStoredMode(t, e); got != "gaps" {
		t.Errorf("stored after bad values = %q, want gaps", got)
	}
}

// covers: publishEntityTypeVersion 422
func TestNumberingModeWithABreakingDefinitionChangesNothing(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	auditCount := func() int {
		var n int
		if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE tenant_id = $1", e.a.id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := auditCount()
	body := strings.Replace(etPublishBody(t, `"default": "new"`, `"default": "closed"`), `{"definition"`, `{"numbering_mode":"gapless","definition"`, 1)
	wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-brk", body), http.StatusUnprocessableEntity, problem.SchemaBreakingChange)
	if got := etStoredMode(t, e); got != "gaps" {
		t.Errorf("stored mode = %q, want gaps: the refused request rolls back", got)
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("versions = %d, want 1", n)
	}
	if after := auditCount(); after != before {
		t.Errorf("audit entries = %d, want %d", after, before)
	}
}

// covers: createEntityType 403
// covers: publishEntityTypeVersion 403 404
func TestNumberingModeRefusedForNonAdminsAndOtherTenants(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	wantProblem(t, e.do(t, http.MethodPost, "/api/v1/entity-types", "user-a", "k-u-create", etModeCreateBody(`"gapless"`)), http.StatusForbidden, problem.Forbidden)
	wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "user-a", "k-u-pub", `{"numbering_mode":"gapless"}`), http.StatusForbidden, problem.Forbidden)
	// Tenant B's admin cannot see or change tenant A's type: the same 404 as an unknown key.
	rec := e.do(t, http.MethodPost, etVersionsPath, "admin-b", "k-b-pub", `{"numbering_mode":"gapless"}`)
	unknown := e.do(t, http.MethodPost, "/api/v1/entity-types/no_such_type/versions", "admin-b", "k-b-unknown", `{"numbering_mode":"gapless"}`)
	if rec.Code != http.StatusNotFound || rec.Body.String() != unknown.Body.String() {
		t.Errorf("cross-tenant = %d %s, want the unknown-key 404", rec.Code, rec.Body)
	}
	if got := etStoredMode(t, e); got != "gaps" {
		t.Errorf("stored = %q, want gaps", got)
	}
}

// covers: publishEntityTypeVersion 201 412
func TestModeOnlyPublishHonoursIfMatch(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	// Publish version 2 so that W/"1" is stale and W/"2" is current.
	body := etPublishBody(t, `"description": "a <b>&amp; tag"`, `"description": "changed"`)
	if rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-v2", body); rec.Code != http.StatusCreated {
		t.Fatalf("publish v2 = %d: %s", rec.Code, rec.Body)
	}
	const modeBody = `{"numbering_mode":"gapless"}`

	// Stale If-Match: 412, and the stored mode is unchanged.
	rec := e.doIf(t, http.MethodPost, etVersionsPath, "admin-a", "k-stale", `W/"1"`, modeBody)
	wantProblem(t, rec, http.StatusPreconditionFailed, problem.VersionConflict)
	if got := etStoredMode(t, e); got != "gaps" {
		t.Fatalf("stored mode after a stale If-Match = %q, want gaps", got)
	}

	// Current If-Match: the mode changes.
	rec = e.doIf(t, http.MethodPost, etVersionsPath, "admin-a", "k-cur", `W/"2"`, modeBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("current If-Match = %d: %s", rec.Code, rec.Body)
	}
	if got := etStoredMode(t, e); got != "gapless" {
		t.Errorf("stored mode = %q, want gapless", got)
	}

	// No If-Match: still optional.
	rec = e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-none", `{"numbering_mode":"gaps"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("no If-Match = %d: %s", rec.Code, rec.Body)
	}
	if got := etStoredMode(t, e); got != "gaps" {
		t.Errorf("stored mode = %q, want gaps", got)
	}
}
