package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestRunShutsDownTelemetryOnCancel(t *testing.T) {
	env := map[string]string{"NAVA_HTTP_ADDR": "127.0.0.1:0"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	ctx, cancel := context.WithCancel(t.Context())
	var out syncBuffer
	done := make(chan error, 1)
	go func() { done <- run(ctx, lookup, &out) }()

	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(out.String(), "nava-core listening") {
		if time.Now().After(deadline) {
			t.Fatalf("server did not start: %s", out.String())
		}
		time.Sleep(10 * time.Millisecond)
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
	logs := out.String()
	if !strings.Contains(logs, `"msg":"telemetry shut down"`) {
		t.Errorf("telemetry.Shutdown was not called during shutdown:\n%s", logs)
	}
	if !strings.Contains(logs, `"trace_export":false`) {
		t.Errorf("trace export is on with no endpoint:\n%s", logs)
	}
}

func TestRunShutsDownGracefullyWhenServingFails(t *testing.T) {
	env := map[string]string{"NAVA_HTTP_ADDR": "127.0.0.1:99999"} // invalid port: ListenAndServe fails
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	var out syncBuffer
	done := make(chan error, 1)
	go func() { done <- run(t.Context(), lookup, &out) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("run returned nil, want the listen error")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("run did not return after serving failed")
	}
	logs := out.String()
	for _, msg := range []string{"nava-core shutting down", "telemetry shut down"} {
		if !strings.Contains(logs, `"msg":"`+msg+`"`) {
			t.Errorf("no %q line; the shutdown path was skipped:\n%s", msg, logs)
		}
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
