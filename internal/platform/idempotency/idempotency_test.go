package idempotency_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func TestRequestHash(t *testing.T) {
	t.Parallel()
	h := func(m, p, b string) string { return string(idempotency.RequestHash(m, p, []byte(b))) }
	same := [][2]string{
		{`{"a":1,"b":[1,2]}`, ` { "b" : [1, 2], "a": 1 } `},
		{``, "  \n"},
	}
	for _, c := range same {
		if h("POST", "/x", c[0]) != h("post", "/x", c[1]) {
			t.Errorf("%q and %q should hash alike", c[0], c[1])
		}
	}
	differ := []struct{ name, m, p, b string }{
		{"method", "PUT", "/x", `{"a":1}`},
		{"path", "POST", "/y", `{"a":1}`},
		{"query", "POST", "/x?q=1", `{"a":1}`},
		{"value", "POST", "/x", `{"a":2}`},
		{"number text", "POST", "/x", `{"a":1.0}`},
		{"array order", "POST", "/x", `[2,1]`},
		{"raw body", "POST", "/x", `not json`},
		{"empty", "POST", "/x", ``},
	}
	base := h("POST", "/x", `{"a":1}`)
	seen := map[string]string{base: "base"}
	for _, c := range differ {
		got := h(c.m, c.p, c.b)
		if prev, dup := seen[got]; dup {
			t.Errorf("%s hashes like %s", c.name, prev)
		}
		seen[got] = c.name
	}
	if h("POST", "/x", `{"a":1} {"a":2}`) == h("POST", "/x", `{"a":2}`) {
		t.Error("trailing data must not be dropped")
	}
	if h("POST", "/x", `"a"`) == h("POST", "/x", `a`) {
		t.Error("a JSON string and raw text must differ")
	}
}

func newMW(t *testing.T) (http.Handler, *int) {
	t.Helper()
	// A lazy pool that is never dialled: these cases end before the database.
	pool, err := pgxpool.New(t.Context(), "postgres://nobody@127.0.0.1:1/none")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	mw, err := idempotency.New(pool, reg, slog.New(slog.DiscardHandler), noop.NewMeterProvider())
	if err != nil {
		t.Fatal(err)
	}
	runs := new(int)
	return mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*runs++
		w.WriteHeader(http.StatusNoContent)
	})), runs
}

func call(h http.Handler, method string, hdr http.Header, body string, withPrincipal bool) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, "/api/v1/things", strings.NewReader(body))
	for k, v := range hdr {
		req.Header[k] = v
	}
	if withPrincipal {
		req = req.WithContext(identity.NewContext(req.Context(), identity.Principal{TenantID: uuid.New(), PrincipalID: uuid.New()}))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestWritesNeedAValidKey(t *testing.T) {
	t.Parallel()
	h, runs := newMW(t)
	long := strings.Repeat("k", 256)
	for name, hdr := range map[string]http.Header{
		"missing":       {},
		"empty":         {idempotency.Header: {""}},
		"two":           {idempotency.Header: {"a", "b"}},
		"too long":      {idempotency.Header: {long}},
		"has a space":   {idempotency.Header: {"a b"}},
		"non-printable": {idempotency.Header: {"a\x01"}},
	} {
		for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			rec := call(h, m, hdr, `{}`, true)
			var b problem.Body
			if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
				t.Fatalf("%s %s: %v", name, m, err)
			}
			if rec.Code != http.StatusBadRequest || b.Code != problem.ValidationFailed ||
				len(b.Errors) != 1 || b.Errors[0].Pointer != "/header/Idempotency-Key" ||
				rec.Header().Get("Content-Type") != problem.ContentType {
				t.Errorf("%s %s = %d %s, want 400 validation_failed on the header", name, m, rec.Code, rec.Body)
			}
		}
	}
	if *runs != 0 {
		t.Errorf("handler ran %d times for rejected requests", *runs)
	}
}

func TestReadsPassThrough(t *testing.T) {
	t.Parallel()
	h, runs := newMW(t)
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		if rec := call(h, m, nil, "", true); rec.Code != http.StatusNoContent {
			t.Errorf("%s = %d, want the handler's 204", m, rec.Code)
		}
	}
	if *runs != 3 {
		t.Errorf("handler ran %d times, want 3", *runs)
	}
}

func TestWriteWithoutPrincipalIs500(t *testing.T) {
	t.Parallel()
	h, runs := newMW(t)
	rec := call(h, http.MethodPost, http.Header{idempotency.Header: {"k"}}, `{}`, false)
	if rec.Code != http.StatusInternalServerError || *runs != 0 {
		t.Errorf("= %d, runs %d; want 500 and no handler run", rec.Code, *runs)
	}
}

func TestOversizedBodyIsRejected(t *testing.T) {
	t.Parallel()
	h, runs := newMW(t)
	rec := call(h, http.MethodPost, http.Header{idempotency.Header: {"k"}}, strings.Repeat("x", 1<<20+1), true)
	if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("max_size")) || *runs != 0 {
		t.Errorf("= %d %s, runs %d; want 400 max_size", rec.Code, rec.Body, *runs)
	}
}

func TestNewRequiresDependencies(t *testing.T) {
	t.Parallel()
	if _, err := idempotency.New(nil, nil, nil, nil); err == nil {
		t.Error("New accepted nil dependencies")
	}
}

func TestContentTypeFor(t *testing.T) {
	t.Parallel()
	for status, want := range map[int]string{200: "application/json", 201: "application/json", 399: "application/json", 400: problem.ContentType, 409: problem.ContentType, 422: problem.ContentType} {
		if got := idempotency.ContentTypeFor(status); got != want {
			t.Errorf("ContentTypeFor(%d) = %q, want %q", status, got, want)
		}
	}
}
