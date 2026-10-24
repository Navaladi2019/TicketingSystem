package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
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

// logLines runs the same calls through a logger and returns its stdout.
func logLines(t *testing.T, build func(*slog.Logger) *slog.Logger) string {
	t.Helper()
	var buf bytes.Buffer
	log := build(logging.New(&buf, config.Log{Level: slog.LevelInfo, Format: config.FormatJSON}))
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2},
	})
	ctx := requestid.NewContext(trace.ContextWithSpanContext(t.Context(), sc), "req-1")
	log.InfoContext(ctx, "http request", slog.String("route", "/x"), slog.Group("g", slog.Int("n", 1)))
	log.With(slog.String("listener", "public")).WarnContext(ctx, "second")
	return timeRE.ReplaceAllString(buf.String(), "")
}

// timeRE strips the wall-clock field, the one part that differs run to run.
var timeRE = regexp.MustCompile(`"time":"[^"]*",`)

func TestWithExportNilLeavesLoggerUnchanged(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := logging.New(&buf, config.Log{Level: slog.LevelInfo, Format: config.FormatJSON})
	if got := logging.WithExport(base, nil); got != base {
		t.Fatal("WithExport(base, nil) must return base itself")
	}
}

// TestStdoutUnchangedByExport: with no export handler (endpoint unset) stdout
// is byte-for-byte what it was before the bridge existed. With one, stdout is
// still identical: the bridge adds a destination, it does not touch stdout.
func TestStdoutUnchangedByExport(t *testing.T) {
	t.Parallel()
	want := logLines(t, func(l *slog.Logger) *slog.Logger { return l })
	if !strings.Contains(want, `"trace_id":"01000000000000000000000000000000"`) {
		t.Fatalf("baseline lacks trace_id: %s", want)
	}
	if got := logLines(t, func(l *slog.Logger) *slog.Logger { return logging.WithExport(l, nil) }); got != want {
		t.Errorf("export off changed stdout:\n got %q\nwant %q", got, want)
	}
	exp := &captureHandler{}
	if got := logLines(t, func(l *slog.Logger) *slog.Logger { return logging.WithExport(l, exp) }); got != want {
		t.Errorf("export on changed stdout:\n got %q\nwant %q", got, want)
	}
	if len(exp.records) != 2 {
		t.Fatalf("exported %d records, want 2", len(exp.records))
	}
	// The exported branch gets the request id but not duplicate trace attrs.
	got := map[string]string{}
	exp.records[0].Attrs(func(a slog.Attr) bool { got[a.Key] = a.Value.String(); return true })
	if got["request_id"] != "req-1" || got["route"] != "/x" {
		t.Errorf("exported attrs = %v", got)
	}
	if _, ok := got["trace_id"]; ok {
		t.Error("exported record carries trace_id as an attribute; the bridge sets it as a field")
	}
}

type captureHandler struct {
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }
