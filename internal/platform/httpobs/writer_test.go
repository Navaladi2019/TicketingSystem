package httpobs

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutePath(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"GET /items/{id}": "/items/{id}",
		"/plain":          "/plain",
		"HOST":            "HOST", // no slash: returned as is
		"":                "",
	} {
		if got := routePath(in); got != want {
			t.Errorf("routePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResponseStatus(t *testing.T) {
	t.Parallel()
	var r response
	if r.status() != http.StatusOK {
		t.Fatalf("uncommitted status = %d, want 200", r.status())
	}
	r.commit(http.StatusTeapot)
	r.commit(http.StatusOK) // later commits are ignored
	if r.status() != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", r.status())
	}
}

// serveThrough runs h behind serve on a real server, so the writer has the
// optional interfaces (ReaderFrom, Flusher, Hijacker) httpsnoop must keep.
func serveThrough(t *testing.T, log *slog.Logger, h http.HandlerFunc) (*httptest.Server, func() response) {
	t.Helper()
	var got response
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		got = serve(log, h, w, r)
	}))
	t.Cleanup(srv.Close)
	// The result is read only after the handler has returned.
	return srv, func() response {
		<-done
		return got
	}
}

func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, string(b)
}

func TestWrapRecordsEveryWritePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		h    http.HandlerFunc
		code int
		body string
	}{
		{"WriteString", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "s") }, 200, "s"},
		{"ReadFrom", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.(io.ReaderFrom).ReadFrom(strings.NewReader("r"))
		}, 200, "r"},
		{"Flush", func(w http.ResponseWriter, _ *http.Request) { w.(http.Flusher).Flush() }, 200, ""},
		{"FlushError", func(w http.ResponseWriter, _ *http.Request) {
			_ = http.NewResponseController(w).Flush()
		}, 200, ""},
		{"informational then final", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusEarlyHints) // 1xx commits nothing
			w.WriteHeader(http.StatusAccepted)
		}, 202, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv, wait := serveThrough(t, slog.New(slog.DiscardHandler), tt.h)
			res, body := get(t, srv.URL)
			defer res.Body.Close()
			if res.StatusCode != tt.code || body != tt.body {
				t.Fatalf("got %d %q, want %d %q", res.StatusCode, body, tt.code, tt.body)
			}
			if got := wait(); got.status() != tt.code || !got.committed {
				t.Fatalf("observed %+v, want committed %d", got, tt.code)
			}
		})
	}
}

func TestWrapRecordsHijack(t *testing.T) {
	t.Parallel()
	srv, wait := serveThrough(t, slog.New(slog.DiscardHandler), func(w http.ResponseWriter, _ *http.Request) {
		conn, _, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = conn.Close()
	})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	if res, err := http.DefaultClient.Do(req); err == nil {
		_ = res.Body.Close()
	} // the connection was closed with no response: an error is expected
	if !wait().committed {
		t.Fatal("a hijacked response must count as committed")
	}
}

func TestPanicBecomesInternalErrorAndKeepsRequestID(t *testing.T) {
	t.Parallel()
	for name, v := range map[string]any{"error": errors.New("boom"), "string": "boom"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			srv, wait := serveThrough(t, slog.New(slog.NewJSONHandler(&logs, nil)), func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Set-Cookie", "a=b")
				panic(v)
			})
			res, body := get(t, srv.URL)
			defer res.Body.Close()
			if res.StatusCode != 500 || !strings.Contains(body, "internal_error") {
				t.Fatalf("got %d %s, want the internal_error problem", res.StatusCode, body)
			}
			if res.Header.Get("Set-Cookie") != "" {
				t.Fatal("headers set before the panic must be dropped")
			}
			if got := wait(); got.status() != 500 || !strings.Contains(logs.String(), "http handler panic") {
				t.Fatalf("status %d, logs %s", got.status(), logs.String())
			}
		})
	}
}

func TestPanicAfterCommitAbortsTheConnection(t *testing.T) {
	t.Parallel()
	srv, _ := serveThrough(t, slog.New(slog.DiscardHandler), func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		panic("late")
	})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return // aborted before the headers arrived: also a dropped connection
	}
	defer res.Body.Close()
	if _, err := io.ReadAll(res.Body); err == nil {
		t.Fatal("a partial response must not look complete: want a read error")
	}
}

func TestAbortHandlerPassesThrough(t *testing.T) {
	t.Parallel()
	defer func() {
		if v := recover(); v != http.ErrAbortHandler { //nolint:errorlint // identity, as net/http
			t.Fatalf("recovered %v, want http.ErrAbortHandler", v)
		}
	}()
	serve(slog.New(slog.DiscardHandler), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}), httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
}
