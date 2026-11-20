package apihttp_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// TestStrictBodyRefused: a body with a duplicate member name or invalid
// UTF-8 is answered 422 with the route's own code before the validator or
// the handler reads it (TICK-964), and the response echoes none of it.
func TestStrictBodyRefused(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	tests := []struct {
		name, body, pointer, rule string
	}{
		{"duplicate", `{"name":"` + secret + `","name":"abc"}`, "/body/name", "duplicate"},
		{"duplicate spelled with an escape", strings.ReplaceAll(`{"name":"abc","%u006eame":"abc"}`, "%u", `\u`), "/body/name", "duplicate"},
		{"duplicate at nested depth", `{"name":"abc","x":{"y":[{"z":1,"z":"` + secret + `"}]}}`, "/body/x/y/0/z", "duplicate"},
		{"lone surrogate", `{"name":"\ud800` + secret + `"}`, "/body/name", "invalid_utf8"},
		{"overlong UTF-8", "{\"name\":\"\xc0\xaf" + secret + "\"}", "/body/name", "invalid_utf8"},
		{"invalid UTF-8 in a name", "{\"name\":\"abc\",\"" + secret + "\xff\":1}", "/body", "invalid_utf8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := e.do(t, "POST", "/api/v1/items", tt.body)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422: %s", rec.Code, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); ct != problem.ContentType {
				t.Fatalf("Content-Type = %q, want %q", ct, problem.ContentType)
			}
			if strings.Contains(rec.Body.String(), secret) {
				t.Fatalf("body echoes the input: %s", rec.Body)
			}
			e.assertProblemSchema(t, rec.Body.Bytes())
			var p problem.Body
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			if p.Code != problem.RecordInvalid {
				t.Errorf("code = %q, want %q", p.Code, problem.RecordInvalid)
			}
			if len(p.Errors) != 1 || p.Errors[0].Pointer != tt.pointer || p.Errors[0].Rule != tt.rule {
				t.Errorf("errors = %+v, want %s (%s)", p.Errors, tt.pointer, tt.rule)
			}
		})
	}
}

// TestStrictBodyPassesTheBodyOn: a clean body reaches the validator and the
// handler unchanged, and a syntax error stays the validator's 400.
func TestStrictBodyPassesTheBodyOn(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if rec := e.do(t, "POST", "/api/v1/items", `{"name":"abc"}`); rec.Code != http.StatusCreated {
		t.Errorf("clean body = %d: %s", rec.Code, rec.Body)
	}
	if rec := e.do(t, "POST", "/api/v1/items", `{"name":"abc"`); rec.Code != http.StatusBadRequest {
		t.Errorf("truncated body = %d, want 400: %s", rec.Code, rec.Body)
	}

	var got string
	a := apihttp.New(&openapi3.T{}, e.reg, slog.New(slog.DiscardHandler), "")
	h := a.StrictBody(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
	}))
	body := `{"name":"abc","x":[1,2]}`
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), "POST", "/", strings.NewReader(body)))
	if got != body {
		t.Errorf("next read %q, want %q", got, body)
	}
}

// TestStrictBodyUnmappedRoute: a route with no 422 code configured still
// refuses the body, as validation_failed, never passes it on.
func TestStrictBodyUnmappedRoute(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	a := apihttp.New(&openapi3.T{}, e.reg, slog.New(slog.DiscardHandler), "")
	h := a.StrictBody(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("handler ran")
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), "POST", "/", strings.NewReader(`{"a":1,"a":2}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body)
	}
	var p problem.Body
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != problem.ValidationFailed || len(p.Errors) != 1 || p.Errors[0].Rule != "duplicate" {
		t.Errorf("problem = %+v, want validation_failed with rule duplicate", p)
	}
}

// countingReader serves n zero-free filler bytes and counts what was read.
type countingReader struct{ left, read int }

func (c *countingReader) Read(p []byte) (int, error) {
	if c.left == 0 {
		return 0, io.EOF
	}
	n := min(len(p), c.left)
	for i := range p[:n] {
		p[i] = ' '
	}
	c.left -= n
	c.read += n
	return n, nil
}

// TestStrictBodyBounded: a body is read no further than the idempotency
// middleware's limit, on a GET as on a POST, and an oversized body is refused.
func TestStrictBodyBounded(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	const huge = 64 << 20
	for _, tt := range []struct{ method, target string }{
		{"GET", "/api/v1/items/" + goodID},
		{"POST", "/api/v1/items"},
	} {
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()
			body := &countingReader{left: huge}
			req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.target, body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.handler.ServeHTTP(rec, req)
			if limit := idempotency.MaxBodySize; body.read > limit+(64<<10) {
				t.Errorf("read %d bytes, want at most about %d", body.read, limit)
			}
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body)
			}
		})
	}
}
