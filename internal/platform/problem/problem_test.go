package problem_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
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
	if n := len(problem.Codes()); n != 35 {
		t.Errorf("%d Go codes, want 35; update this test with the registry", n)
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

func TestWriteErrorReports5xxToTheLogOnly(t *testing.T) {
	t.Parallel()
	reg, _ := loadReal(t)
	req := httptest.NewRequestWithContext(requestid.NewContext(t.Context(), "r1"), http.MethodGet, "/x", nil)
	cause := errors.New("db password=hunter2 failed")

	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	rec := httptest.NewRecorder()
	reg.WriteError(rec, req, log, problem.InternalError, cause)
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "hunter2") {
		t.Fatalf("got %d %s: the body must not carry the cause", rec.Code, rec.Body)
	}
	if !strings.Contains(logs.String(), "hunter2") {
		t.Fatalf("the cause must reach the log: %s", logs.String())
	}

	// A registered 4xx is not reported.
	logs.Reset()
	rec = httptest.NewRecorder()
	reg.WriteError(rec, req, log, problem.RateLimited, cause)
	if rec.Code != 429 || logs.Len() != 0 {
		t.Fatalf("got %d, logs %q: a 4xx must not be logged as an error", rec.Code, logs.String())
	}

	// An unregistered code is reported and answered as internal_error.
	logs.Reset()
	rec = httptest.NewRecorder()
	reg.WriteError(rec, req, log, "no_such_code", cause)
	if rec.Code != 500 || logs.Len() == 0 {
		t.Fatalf("got %d, logs %q", rec.Code, logs.String())
	}
}
