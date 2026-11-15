package server

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// P7-03: every metadata refusal maps to its registered code with the paths
// under /definition; an unknown one is an internal error, never a 422 that
// names the wrong rule.
func TestRefusalProblemMapsEachCode(t *testing.T) {
	t.Parallel()
	for code, want := range map[string]problem.Code{
		metadata.RefusalFilterableNotTopLevel:  problem.FilterableFieldNotTopLevel,
		metadata.RefusalFilterableTypeChange:   problem.FilterableFieldTypeChange,
		metadata.RefusalSearchableNotAvailable: problem.SearchableNotAvailable,
		"something_new":                        problem.InternalError,
	} {
		r := &metadata.RefusalError{Code: code, Paths: []string{"/schema/properties/a"}}
		pe := refusalProblem(r, "/definition", r)
		if pe.code != want {
			t.Errorf("%s: code = %s, want %s", code, pe.code, want)
		}
		if want != problem.InternalError && (len(pe.fields) != 1 || pe.fields[0].Pointer != "/definition/schema/properties/a") {
			t.Errorf("%s: fields = %v", code, pe.fields)
		}
		var got *metadata.RefusalError
		if !errors.As(error(pe), &got) {
			t.Errorf("%s: problem does not unwrap to the refusal", code)
		}
	}
}

func TestParseCreateRefusesSearchable(t *testing.T) {
	t.Parallel()
	body := `{"key":"incident","name":"T","number_prefix":"INC","definition":{"schema":` +
		`{"type":"object","additionalProperties":false,"properties":{"state":{"type":"string","enum":["new"],"x-nava-searchable":true}}},` +
		`"fixed_columns":{"state":"state"},"states":{"initial":"new","states":["new"],"transitions":[]}}}`
	_, perr := parseCreate(json.RawMessage(body))
	if perr == nil || perr.code != problem.SearchableNotAvailable {
		t.Fatalf("err = %v, want searchable_not_available", perr)
	}
	if len(perr.fields) != 1 || perr.fields[0].Pointer != "/definition/schema/properties/state/x-nava-searchable" {
		t.Fatalf("fields = %v", perr.fields)
	}
}
