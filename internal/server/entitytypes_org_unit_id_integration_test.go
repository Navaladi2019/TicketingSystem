//go:build integration

// TICK-1115 (ADR-010 D8): with the org_unit_id constraint in place, the API
// still answers a definition with that property from Compile, and writes
// nothing. Helpers are in entitytypes_integration_test.go.

package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
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
