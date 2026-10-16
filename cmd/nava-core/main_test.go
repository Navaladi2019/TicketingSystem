package main

import (
	"bytes"
	"context"
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

func TestRunShutsDownTelemetryOnCancel(t *testing.T) {
	t.Parallel()
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
	if !strings.Contains(logs, `"msg":"telemetry shut down","ok":true`) {
		t.Errorf("telemetry.Shutdown was not called during shutdown:\n%s", logs)
	}
	if !strings.Contains(logs, `"trace_export":false`) {
		t.Errorf("trace export is on with no endpoint:\n%s", logs)
	}
}
