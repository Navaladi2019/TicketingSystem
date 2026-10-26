package problem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
)

func loadReal(t *testing.T) (*problem.Registry, []byte) {
	t.Helper()
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatalf("api/errors.yaml: %v", err)
	}
	return reg, data
}

// TestRegistryMatchesGoCodes: LoadRegistry fails on a Go constant missing from
// the file; this checks the other direction, a file row no constant names.
func TestRegistryMatchesGoCodes(t *testing.T) {
	t.Parallel()
	reg, _ := loadReal(t)
	known := map[problem.Code]bool{}
	for _, c := range problem.Codes() {
		known[c] = true
		if _, ok := reg.Lookup(c); !ok {
			t.Errorf("code %q is not in api/errors.yaml", c)
		}
	}
	for _, c := range []problem.Code{"tenant_unresolved", "invalid_token", "idempotency_key_reuse",
		"invalid_cursor", "rate_limited", "not_found", "validation_failed"} {
		if !known[c] {
			t.Errorf("PLAN-001 first code %q has no Go constant", c)
		}
	}
	if n := len(problem.Codes()); n != 14 {
		t.Errorf("%d Go codes, want 14; update this test with the registry", n)
	}
}

func TestInternalMatchesRegistry(t *testing.T) {
	t.Parallel()
	reg, _ := loadReal(t)
	e, _ := reg.Lookup(problem.InternalError)
	rec := httptest.NewRecorder()
	problem.Internal(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	var b problem.Body
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if rec.Code != e.Status || b.Title != e.Title || b.Detail != e.Detail || b.Code != e.Code {
		t.Errorf("Internal() = %+v, registry says %+v", b, e)
	}
}

func TestLoadRegistryRejectsBadFiles(t *testing.T) {
	t.Parallel()
	_, real := loadReal(t)
	tests := []struct{ name, data string }{
		{"unknown key", strings.Replace(string(real), "title:", "titel:", 1)},
		{"second document", string(real) + "---\ncodes: []\n"},
		{"duplicate code", string(real) + "  - {code: not_found, status: 404, title: t, detail: d}\n"},
		{"missing Go code", "codes:\n  - {code: not_found, status: 404, title: t, detail: d}\n"},
		{"bad status", strings.Replace(string(real), "status: 401", "status: 200", 1)},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := problem.LoadRegistry([]byte(tt.data)); err == nil {
				t.Error("LoadRegistry accepted it")
			}
		})
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()
	reg, _ := loadReal(t)
	req := httptest.NewRequestWithContext(requestid.NewContext(t.Context(), "abc123"), http.MethodGet, "/x?q=SECRET", nil)

	rec := httptest.NewRecorder()
	reg.Write(rec, req, problem.RateLimited)
	var b problem.Body
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 429 || b.Type != "urn:nava:problem:rate_limited" || b.Instance != "urn:nava:request:abc123" ||
		rec.Header().Get("Content-Type") != problem.ContentType {
		t.Errorf("got %d %+v %v", rec.Code, b, rec.Header())
	}
	if strings.Contains(rec.Body.String(), "SECRET") {
		t.Error("body echoes the query")
	}

	// An unregistered code is a programming error: it degrades to internal_error.
	rec = httptest.NewRecorder()
	reg.Write(rec, req, "no_such_code")
	if rec.Code != 500 || !strings.Contains(rec.Body.String(), `"code":"internal_error"`) {
		t.Errorf("unregistered code answered %d %s", rec.Code, rec.Body)
	}
}
