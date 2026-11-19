package server

import (
	"net/http"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// TestStrictBodyCodesCoverTheContract: every operation of the contract with
// a request body has its strict-JSON 422 code (TICK-964), the code is a
// registered 422 the operation declares, and no entry names an operation
// that has no body.
func TestStrictBodyCodesCoverTheContract(t *testing.T) {
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
	codes := strictBodyCodes()
	var withBody []string
	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			if op.RequestBody == nil {
				continue
			}
			pattern := method + " " + baseURL + path
			withBody = append(withBody, pattern)
			code, ok := codes[pattern]
			if !ok {
				t.Errorf("%s (%s) has a request body but no strict-JSON code", pattern, op.OperationID)
				continue
			}
			if e, ok := reg.Lookup(code); !ok || e.Status != http.StatusUnprocessableEntity {
				t.Errorf("%s: code %q is not a registered 422", pattern, code)
			}
			if op.Responses.Status(http.StatusUnprocessableEntity) == nil {
				t.Errorf("%s (%s) declares no 422 response", pattern, op.OperationID)
			}
		}
	}
	for pattern := range codes {
		if !slices.Contains(withBody, pattern) {
			t.Errorf("strict-JSON code for %s, which has no request body in the contract", pattern)
		}
	}
	if len(withBody) == 0 {
		t.Fatal("the contract has no operation with a request body")
	}
}
