package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func TestParseFormatLink(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	ok := func(raw string) entitytypes.NumberFormatChange {
		t.Helper()
		c, good := parseFormatLink(json.RawMessage(raw))
		if !good {
			t.Fatalf("%s refused", raw)
		}
		return c
	}
	if c := ok(`null`); !c.Set || c.Ref != nil {
		t.Errorf("null = %+v, want Set with no Ref", c)
	}
	if c := ok(`{"id":"` + id.String() + `","version":3}`); !c.Set || c.Ref == nil || c.Ref.ID != id || c.Ref.Version != 3 {
		t.Errorf("link = %+v", c)
	}
	for _, raw := range []string{
		`"x"`, `1`, `[]`, `{}`, `{"id":"` + id.String() + `"}`, `{"version":1}`,
		`{"id":"nope","version":1}`, `{"id":"` + id.String() + `","version":0}`,
		`{"id":"` + id.String() + `","version":-1}`, `{"id":"` + id.String() + `","version":1.5}`,
		`{"id":"` + id.String() + `","version":1,"extra":true}`, `{"id":null,"version":1}`,
	} {
		if c, good := parseFormatLink(json.RawMessage(raw)); good || c.Set {
			t.Errorf("%s accepted as %+v", raw, c)
		}
	}
}

func TestParsePublishNumberFormat(t *testing.T) {
	t.Parallel()
	def := `"definition":{"schema":{"type":"object"}}`
	id := uuid.New().String()

	pr, perr := parsePublish(json.RawMessage(`{` + def + `}`))
	if perr != nil || pr.format.Set {
		t.Fatalf("no field: %+v %v", pr.format, perr)
	}
	pr, perr = parsePublish(json.RawMessage(`{` + def + `,"number_format":null}`))
	if perr != nil || !pr.format.Set || pr.format.Ref != nil {
		t.Fatalf("null: %+v %v", pr.format, perr)
	}
	pr, perr = parsePublish(json.RawMessage(`{` + def + `,"number_format":{"id":"` + id + `","version":2}}`))
	if perr != nil || pr.format.Ref == nil || pr.format.Ref.Version != 2 {
		t.Fatalf("link: %+v %v", pr.format, perr)
	}
	for name, body := range map[string]string{
		"bad shape":      `{` + def + `,"number_format":{"id":"x"}}`,
		"without a body": `{"number_format":{"id":"` + id + `","version":1}}`,
	} {
		_, perr := parsePublish(json.RawMessage(body))
		if perr == nil || perr.code != problem.InvalidDefinition {
			t.Errorf("%s: %+v", name, perr)
		}
	}
}

func TestFormatLinkProblemGivesOneBodyForNotFound(t *testing.T) {
	t.Parallel()
	a := formatLinkProblem(entitytypes.ErrFormatNotFound)
	b := formatLinkProblem(errors.New("wrapped: " + entitytypes.ErrFormatNotFound.Error()))
	if a.code != problem.NumberFormatNotFound || b.code != a.code || len(a.fields) != 1 || len(b.fields) != 1 || a.fields[0] != b.fields[0] {
		t.Errorf("not found: %+v / %+v", a, b)
	}
	o := formatLinkProblem(&entitytypes.OverlapError{Template: "{prefix}{seq:7}"})
	if o.code != problem.NumberFormatOverlap || len(o.fields) != 1 || o.fields[0].Pointer != "/number_format" || o.fields[0].Rule != "overlap:{prefix}{seq:7}" {
		t.Errorf("overlap: %+v", o)
	}
	if !strings.Contains((&entitytypes.OverlapError{Template: "T"}).Error(), `"T"`) {
		t.Error("OverlapError does not name the template")
	}
}

func TestRepublishRefusalMapsEachFailureToItsCode(t *testing.T) {
	t.Parallel()
	var logs strings.Builder
	h := &handlers{log: slog.New(slog.NewTextHandler(&logs, nil))}
	p := identity.Principal{TenantID: uuid.New()}
	for _, tc := range []struct {
		name string
		err  error
		code problem.Code
	}{
		{"forbidden", authz.ErrForbidden, problem.Forbidden},
		{"overlap", &entitytypes.OverlapError{Template: "T"}, problem.NumberFormatOverlap},
		{"format gone", entitytypes.ErrFormatNotFound, problem.NumberFormatNotFound},
		{"type moved", &entitytypes.ConflictError{Current: 3}, problem.VersionConflict},
		{"type deleted", entitytypes.ErrNotFound, problem.VersionConflict},
		{"unexpected", errors.New("db down: secret detail"), problem.InternalError},
	} {
		got := h.refusal(t.Context(), p, republishResult{EntityType: "tx"}, tc.err)
		if got.Status != "refused" || got.Code != string(tc.code) || got.EntityType != "tx" || got.Version != nil {
			t.Errorf("%s: %+v, want refused %s", tc.name, got, tc.code)
		}
	}
	if !strings.Contains(logs.String(), "db down") {
		t.Error("an unexpected failure is not logged")
	}
	b, _ := json.Marshal(h.refusal(t.Context(), p, republishResult{EntityType: "tx"}, errors.New("db down: secret detail")))
	if strings.Contains(string(b), "secret") {
		t.Errorf("the report leaks the failure: %s", b)
	}
}
