package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
)

func record(ctx context.Context, t *testing.T) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	log := slog.New(logging.NewContextHandler(slog.NewJSONHandler(&buf, nil))).With(slog.String("component", "test"))
	log.InfoContext(ctx, "line")
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("not JSON: %q", buf.String())
	}
	return m
}

func TestContextHandlerAddsCorrelationIDs(t *testing.T) {
	t.Parallel()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36},
		SpanID:  trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7},
	})
	ctx := requestid.NewContext(trace.ContextWithSpanContext(t.Context(), sc), "req-1")

	m := record(ctx, t)
	want := map[string]string{
		"trace_id":   "4bf92f3577b34da6a3ce929d0e0e4736",
		"span_id":    "00f067aa0ba902b7",
		"request_id": "req-1",
		"component":  "test",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %s", k, m[k], v)
		}
	}
}

func TestContextHandlerOutsideARequestAddsNothing(t *testing.T) {
	t.Parallel()
	m := record(t.Context(), t)
	for _, k := range []string{"trace_id", "span_id", "request_id"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s set outside a request: %v", k, m[k])
		}
	}
}
