//go:build integration

// TICK-964: every tenant write refuses a body with a duplicate member name or
// invalid UTF-8 with a 422 and its own code, through the real handler against
// a real PostgreSQL, and writes nothing. The bodies are the TICK-510 refusal
// bodies (refusals_integration_test.go), which succeed as sent, with one
// offending member added.

package server_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// covers: createEntityType 422
// covers: publishEntityTypeVersion 422
// covers: createRecord 422
// covers: patchRecord 422
// covers: putEntityTypeAccess 422
// covers: publishWorkflowDefinitionVersion 422
func TestEveryTenantWriteRefusesNonStrictJSONAndWritesNothing(t *testing.T) {
	e := newRecEnv(t)
	r := e.mustCreateRec(t, "user-a", "ticket", "seed", `{"short_description": "seed"}`)
	codes := map[string]problem.Code{
		"createEntityType":                 problem.InvalidDefinition,
		"publishEntityTypeVersion":         problem.InvalidDefinition,
		"createRecord":                     problem.RecordInvalid,
		"patchRecord":                      problem.RecordInvalid,
		"putEntityTypeAccess":              problem.InvalidAccessPolicy,
		"publishWorkflowDefinitionVersion": problem.WorkflowDefinitionInvalid,
	}
	// Each variant goes in front of the body's first member.
	bs := `\`
	variants := []struct {
		name, member, pointer, rule string
	}{
		{"duplicate", `"zz":{"k":1,"k":2},`, "/body/zz/k", "duplicate"},
		{"lone surrogate", `"zz":"` + bs + `ud800",`, "/body/zz", "invalid_utf8"},
		{"overlong UTF-8", "\"zz\":\"\xc0\xaf\",", "/body/zz", "invalid_utf8"},
	}
	seen := 0
	before := withoutIdempotencyKeys(e.rowCounts(t))
	for _, op := range refusalOps(e, r.ID) {
		code, ok := codes[op.id]
		if !op.write {
			continue
		}
		if !ok {
			t.Errorf("write %s has no expected code in this test", op.id)
			continue
		}
		seen++
		for _, v := range variants {
			bad := op
			bad.body = strings.Replace(op.body, "{", "{"+v.member, 1)
			rec := e.send(t, bad, "admin-a", "strict-"+op.id+"-"+v.name)
			p := wantProblem(t, rec, http.StatusUnprocessableEntity, code)
			if len(p.Errors) != 1 || p.Errors[0].Pointer != v.pointer || p.Errors[0].Rule != v.rule {
				t.Errorf("%s %s: errors = %+v, want %s (%s)", op.id, v.name, p.Errors, v.pointer, v.rule)
			}
		}
	}
	if seen != len(codes) {
		t.Errorf("checked %d writes, want %d", seen, len(codes))
	}
	if after := withoutIdempotencyKeys(e.rowCounts(t)); !slices.Equal(before, after) {
		t.Errorf("refused requests changed rows:\nbefore %v\nafter  %v", before, after)
	}
}
