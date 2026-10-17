package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// syncBuffer is a bytes.Buffer safe for the server goroutine and the test.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// The run tests are not parallel: run calls telemetry.RouteErrors, which sets
// OTel's process-wide error handler, so parallel runs would route each
// other's OTel errors into the wrong log buffer.

// freeAddr returns a loopback address with a port that was free a moment ago.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

// get returns the status of GET url, retrying while the listener comes up.
func get(t *testing.T, url string) int {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return resp.StatusCode
		}
		if time.Now().After(deadline) {
			t.Fatalf("GET %s: %v", url, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// assertClosed fails if anything still accepts connections on addr.
func assertClosed(t *testing.T, name, addr string) {
	t.Helper()
	conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(t.Context(), "tcp", addr)
	if err == nil {
		_ = conn.Close()
		t.Errorf("the %s listener %s still accepts connections after run returned", name, addr)
	}
}

// logLines returns the msg of each JSON log line, prefixed with its listener
// attribute when it has one ("public: nava-core shutting down").
func logLines(t *testing.T, out string) []string {
	t.Helper()
	var lines []string
	for l := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		var rec struct{ Msg, Listener string }
		if err := json.Unmarshal([]byte(l), &rec); err != nil {
			t.Fatalf("log line is not JSON: %q", l)
		}
		if rec.Listener != "" {
			rec.Msg = rec.Listener + ": " + rec.Msg
		}
		lines = append(lines, rec.Msg)
	}
	return lines
}

// runWith returns run bound to env instead of the process environment.
func runWith(env map[string]string) func(context.Context, *syncBuffer) error {
	var lookup config.LookupFunc = func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	return func(ctx context.Context, out *syncBuffer) error { return run(ctx, lookup, out) }
}

// TestRunSignalDrainsBothServers starts run for real: /metrics answers only on
// the admin listener, and a signal shuts down both listeners before
// telemetry is flushed.
func TestRunSignalDrainsBothServers(t *testing.T) {
	public, admin := freeAddr(t), freeAddr(t)
	start := runWith(map[string]string{"NAVA_HTTP_ADDR": public, "NAVA_ADMIN_ADDR": admin})

	ctx, cancel := context.WithCancel(t.Context())
	var out syncBuffer
	done := make(chan error, 1)
	go func() { done <- start(ctx, &out) }()

	if code := get(t, "http://"+admin+"/metrics"); code != http.StatusOK {
		t.Errorf("admin /metrics = %d, want 200", code)
	}
	for _, p := range []string{"/healthz", "/readyz"} {
		if code := get(t, "http://"+public+p); code != http.StatusNotFound {
			t.Errorf("public %s = %d, want 404", p, code)
		}
	}
	// No database is configured, so the admin probes fail closed.
	if code := get(t, "http://"+admin+"/healthz"); code != http.StatusServiceUnavailable {
		t.Errorf("admin /healthz with no database = %d, want 503", code)
	}
	if code := get(t, "http://"+admin+"/readyz"); code != http.StatusServiceUnavailable {
		t.Errorf("admin /readyz with no database = %d, want 503", code)
	}
	if code := get(t, "http://"+public+"/metrics"); code != http.StatusNotFound {
		t.Errorf("public /metrics = %d, want 404", code)
	}
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("run did not return after cancel")
	}
	assertClosed(t, "public", public)
	assertClosed(t, "admin", admin)

	logs := out.String()
	lines := logLines(t, logs)
	flushed := slices.Index(lines, "telemetry shut down")
	if flushed < 0 {
		t.Fatalf("telemetry.Shutdown was not called during shutdown:\n%s", logs)
	}
	for _, want := range []string{"public: nava-core shutting down", "admin: nava-core shutting down"} {
		if i := slices.Index(lines, want); i < 0 || i > flushed {
			t.Errorf("no %q line before the telemetry flush:\n%s", want, logs)
		}
	}
	if !strings.Contains(logs, `"trace_export":false`) {
		t.Errorf("trace export is on with no endpoint:\n%s", logs)
	}
}

// TestRunListenFailureShutsDownBoth occupies one listener's port, so its
// ListenAndServe fails: run must drain the other listener too, flush
// telemetry and return the listen error.
func TestRunListenFailureShutsDownBoth(t *testing.T) {
	for _, failing := range []string{"public", "admin"} {
		t.Run(failing, func(t *testing.T) {
			taken, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = taken.Close() }()
			healthy := freeAddr(t)
			addrs := map[string]string{"public": healthy, "admin": healthy}
			addrs[failing] = taken.Addr().String()
			other := "admin"
			if failing == "admin" {
				other = "public"
			}
			start := runWith(map[string]string{"NAVA_HTTP_ADDR": addrs["public"], "NAVA_ADMIN_ADDR": addrs["admin"]})

			var out syncBuffer
			done := make(chan error, 1)
			go func() { done <- start(t.Context(), &out) }()

			select {
			case err := <-done:
				if err == nil {
					t.Fatal("run returned nil, want the listen error")
				}
			case <-time.After(20 * time.Second):
				t.Fatal("run did not return after a listener failed: the other one kept serving")
			}
			assertClosed(t, other, healthy)

			logs := out.String()
			lines := logLines(t, logs)
			for _, want := range []string{
				failing + ": nava-core serve failed",
				failing + ": nava-core shutting down",
				other + ": nava-core shutting down",
				"telemetry shut down",
			} {
				if !slices.Contains(lines, want) {
					t.Errorf("no %q line; the shutdown path was skipped:\n%s", want, logs)
				}
			}
		})
	}
}

func TestServePanicDrainsTheServer(t *testing.T) {
	t.Parallel()
	var out syncBuffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	srv := &http.Server{Addr: "127.0.0.1:0", ReadHeaderTimeout: time.Second}

	err := serve(t.Context(), log, srv, func() error { panic("boom") }, time.Second)
	if err == nil || !strings.Contains(err.Error(), "panic: boom") {
		t.Fatalf("serve = %v, want the recovered panic", err)
	}
	logs := out.String()
	if !strings.Contains(logs, `"level":"ERROR","msg":"nava-core serve failed"`) {
		t.Errorf("the panic was not logged at ERROR by the process logger:\n%s", logs)
	}
	if !strings.Contains(logs, `"msg":"nava-core shutting down"`) {
		t.Errorf("no shutdown line:\n%s", logs)
	}
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("ListenAndServe after serve = %v, want ErrServerClosed: Shutdown was skipped", err)
	}
}

func TestRecoveredTurnsPanicIntoError(t *testing.T) {
	t.Parallel()
	err := recovered(func() error { panic("boom") })
	if err == nil {
		t.Fatal("recovered returned nil for a panic")
	}
	if msg := err.Error(); !strings.Contains(msg, "panic: boom") || !strings.Contains(msg, "main_test.go") {
		t.Errorf("error = %q, want the panic value and a stack reaching the panic", msg)
	}

	want := errors.New("plain")
	if got := recovered(func() error { return want }); !errors.Is(got, want) {
		t.Errorf("recovered = %v, want fn's own error passed through", got)
	}
}
