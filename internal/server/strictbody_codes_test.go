package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// TestStrictBodyRefusalsCoverTheContract: every operation of the contract
// with a request body has its strict-JSON refusal (TICK-964). Each refusal
// answers a registered 422 that the operation declares, with the one field
// error it is given, and no entry names an operation that has no body.
func TestStrictBodyRefusalsCoverTheContract(t *testing.T) {
	t.Parallel()
	data, err := api.Files.ReadFile(api.SpecFile)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := apihttp.LoadSpec(data)
	if err != nil {
		t.Fatal(err)
	}
	errData, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(errData)
	if err != nil {
		t.Fatal(err)
	}
	refusals := strictBodyRefusals(reg)
	fe := problem.FieldError{Pointer: "/body/a", Rule: "duplicate"}
	var withBody []string
	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			if op.RequestBody == nil {
				continue
			}
			pattern := method + " " + baseURL + path
			withBody = append(withBody, pattern)
			refuse, ok := refusals[pattern]
			if !ok {
				t.Errorf("%s (%s) has a request body but no strict-JSON refusal", pattern, op.OperationID)
				continue
			}
			if op.Responses.Status(http.StatusUnprocessableEntity) == nil {
				t.Errorf("%s (%s) declares no 422 response", pattern, op.OperationID)
			}
			rec := httptest.NewRecorder()
			refuse(rec, httptest.NewRequestWithContext(t.Context(), method, path, nil), fe)
			var p problem.Body
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatalf("%s: body is not a problem: %v", pattern, err)
			}
			e, registered := reg.Lookup(p.Code)
			if rec.Code != http.StatusUnprocessableEntity || !registered || e.Status != http.StatusUnprocessableEntity {
				t.Errorf("%s: refusal answered %d %q, want a registered 422", pattern, rec.Code, p.Code)
			}
			if len(p.Errors) != 1 || p.Errors[0] != fe {
				t.Errorf("%s: errors = %+v, want [%+v]", pattern, p.Errors, fe)
			}
		}
	}
	for pattern := range refusals {
		if !slices.Contains(withBody, pattern) {
			t.Errorf("strict-JSON refusal for %s, which has no request body in the contract", pattern)
		}
	}
	if len(withBody) == 0 {
		t.Fatal("the contract has no operation with a request body")
	}
}
