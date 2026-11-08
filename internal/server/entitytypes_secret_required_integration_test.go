//go:build integration

// P5-E5-6 (TICK-539): S8 create and publish refuse a definition that requires
// a secret-marked property (ADR-004-E5 D9, definition_secret_required).

package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// etSecretSchema is etSchema with a marked "api_token" property and extra
// root keywords spliced in before "properties".
func etSecretSchema(t *testing.T, extra string) string {
	t.Helper()
	s := strings.Replace(etSchema, `"note": {"type": "string", "description": "a <b>&amp; tag"}`,
		`"note": {"type": "string", "description": "a <b>&amp; tag"},
    "api_token": {"type": "string", "x-nava-secret": true}`, 1)
	if extra != "" {
		s = strings.Replace(s, `"properties": {`, extra+`, "properties": {`, 1)
	}
	return s
}

func etSecretPublish(schema string) string {
	return fmt.Sprintf(`{"definition": {"schema": %s, %s}}`, schema, etDefTail)
}

func TestPublishRefusesRequiredSecret(t *testing.T) {
	cases := []struct {
		name, schema, pointer string
	}{
		{"top-level required",
			strings.Replace(etSecretSchema(t, ""), `"required": ["short_description", "state"]`, `"required": ["short_description", "state", "api_token"]`, 1),
			"/definition/schema/required/2"},
		{"allOf", etSecretSchema(t, `"allOf": [{"required": ["api_token"]}]`), "/definition/schema/allOf/0/required/0"},
		{"if/then", etSecretSchema(t, `"if": {"required": ["note"]}, "then": {"required": ["api_token"]}`), "/definition/schema/then/required/0"},
		{"dependentRequired", etSecretSchema(t, `"dependentRequired": {"note": ["api_token"]}`), "/definition/schema/dependentRequired/note/0"},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newETEnv(t)
			etCreateIncident(t, e)
			v1 := e.do(t, http.MethodGet, etVersionsPath+"/1", "user-a", "", "").Body.String()
			rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", fmt.Sprintf("k-sr-%d", i), etSecretPublish(tc.schema))
			p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.DefinitionSecretRequired)
			if got := pointers(p); !slices.Equal(got, []string{tc.pointer}) {
				t.Errorf("pointers = %v, want %v", got, tc.pointer)
			}
			if n := etVersionCount(t, e); n != 1 {
				t.Errorf("%d versions, want 1", n)
			}
			if rows := etAuditRows(t, e); len(rows) != 1 {
				t.Errorf("%d audit entries, want 1", len(rows))
			}
			if cur := e.do(t, http.MethodGet, etVersionsPath+"/1", "user-a", "", "").Body.String(); cur != v1 {
				t.Errorf("version 1 changed")
			}
		})
	}
}

func TestPublishAllowsUnrequiredMarker(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-ok-1", etSecretPublish(etSecretSchema(t, "")))
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish = %d: %s", rec.Code, rec.Body)
	}
	if n := etVersionCount(t, e); n != 2 {
		t.Errorf("%d versions, want 2", n)
	}
}

func TestCreateRefusesRequiredSecret(t *testing.T) {
	e := newETEnv(t)
	schema := etSecretSchema(t, `"allOf": [{"required": ["api_token"]}]`)
	body := fmt.Sprintf(`{"key": "incident", "name": "T", "number_prefix": "INC", "definition": {"schema": %s, %s}}`, schema, etDefTail)
	p := wantProblem(t, e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-cr-1", body), http.StatusUnprocessableEntity, problem.DefinitionSecretRequired)
	if got := pointers(p); !slices.Equal(got, []string{"/definition/schema/allOf/0/required/0"}) {
		t.Errorf("pointers = %v", got)
	}
	if n := etVersionCount(t, e); n != 0 {
		t.Errorf("%d versions after a refused create, want 0", n)
	}
	var defs int
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", e.a.id).Scan(&defs); err != nil || defs != 0 {
		t.Errorf("definitions = %d, err %v; want 0", defs, err)
	}
	if rows := etAuditRows(t, e); len(rows) != 0 {
		t.Errorf("%d audit entries, want 0", len(rows))
	}
}

func TestSecretRequiredReplay(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)

	ok := etSecretPublish(etSecretSchema(t, ""))
	first := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-rp-ok", ok)
	again := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-rp-ok", ok)
	if first.Code != http.StatusCreated || again.Code != http.StatusCreated || first.Body.String() != again.Body.String() {
		t.Fatalf("replay = %d / %d, want the same 201", first.Code, again.Code)
	}
	if n := etVersionCount(t, e); n != 2 {
		t.Errorf("%d versions, want 2", n)
	}

	bad := etSecretPublish(etSecretSchema(t, `"allOf": [{"required": ["api_token"]}]`))
	for range 2 {
		wantProblem(t, e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-rp-bad", bad), http.StatusUnprocessableEntity, problem.DefinitionSecretRequired)
	}
	if n := etVersionCount(t, e); n != 2 {
		t.Errorf("%d versions after refused replays, want 2", n)
	}
}

func TestSecretRequiredCrossTenant404Unchanged(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	bad := etSecretPublish(etSecretSchema(t, `"allOf": [{"required": ["api_token"]}]`))
	foreign := e.do(t, http.MethodPost, etVersionsPath, "admin-b", "k-x-1", bad)
	unknown := e.do(t, http.MethodPost, "/api/v1/entity-types/nosuchtype/versions", "admin-a", "k-x-2", bad)
	wantProblem(t, foreign, http.StatusNotFound, problem.NotFound)
	wantProblem(t, unknown, http.StatusNotFound, problem.NotFound)
	if foreign.Body.String() != unknown.Body.String() {
		t.Errorf("404 bodies differ: %s vs %s", foreign.Body, unknown.Body)
	}
}
