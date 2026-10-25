package workerguard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// payloadMarker stands for a payload value: it arrives in a header and ends
// up in an error message.
const payloadMarker = "payload-value-5b1e9c"

// spanText is everything an exported span says: name, attributes, events,
// links and status.
func spanText(s tracetest.SpanStub) string {
	var b strings.Builder
	b.WriteString(s.Name + "\n" + s.Status.Description + "\n")
	for _, kv := range s.Attributes {
		b.WriteString(string(kv.Key) + "=" + kv.Value.String() + "\n")
	}
	for _, ev := range s.Events {
		b.WriteString(ev.Name + "\n")
		for _, kv := range ev.Attributes {
			b.WriteString(string(kv.Key) + "=" + kv.Value.String() + "\n")
		}
	}
	for _, l := range s.Links {
		for _, kv := range l.Attributes {
			b.WriteString(string(kv.Key) + "=" + kv.Value.String() + "\n")
		}
	}
	return b.String()
}

func activityThenFail(ctx workflow.Context) error {
	actx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
	if err := workflow.ExecuteActivity(actx, noop).Get(ctx, nil); err != nil {
		// Workflow code that quotes a payload value next to the guard failure.
		return fmt.Errorf("activity for %s: %w", payloadMarker, err)
	}
	return nil
}

// TICK-94: a guard failure whose error carries a payload value produces an
// exported span with no payload value in its attributes, events or status.
// The worker's spans come from the shared Temporal constructor, so they pass
// the same export allow-list as every other span.
func TestGuardFailureSpanCarriesNoPayloadValue(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), config.Telemetry{},
		telemetry.WithSpanExporter(exp), telemetry.WithSampler(sdktrace.AlwaysSample()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	// raw sees the spans before the allow-list, to prove the value was there.
	raw := tracetest.NewInMemoryExporter()
	tel.TracerProvider.(*sdktrace.TracerProvider).RegisterSpanProcessor(sdktrace.NewSimpleSpanProcessor(raw))
	tt, err := tel.NewTemporal(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}

	id := idOf(tenantA)
	h := newHarness(t, id, withOuter(tt.Tracing))
	h.env.RegisterActivity(noop)
	// The activity arrives with a header whose value is the marker.
	h.tr.setActivity(stringPayload(t, payloadMarker))
	h.env.ExecuteWorkflow(activityThenFail)
	err = h.env.GetWorkflowError()
	var actErr *temporal.ActivityError
	if !errors.As(err, &actErr) {
		t.Fatalf("workflow error %v, want an activity failure", err)
	}
	wantGuardError(t, actErr.Unwrap(), guardActivity, outcomeMismatch)
	if !strings.Contains(err.Error(), payloadMarker) {
		t.Fatalf("workflow error %q does not carry the marker; the test proves nothing", err)
	}
	if strings.Contains(h.logs.String(), payloadMarker) {
		t.Errorf("the guard logged the header value:\n%s", h.logs.String())
	}

	var rawHasMarker bool
	for _, s := range raw.GetSpans() {
		rawHasMarker = rawHasMarker || strings.Contains(spanText(s), payloadMarker)
	}
	if !rawHasMarker {
		t.Fatal("no unfiltered span carries the marker; the test proves nothing")
	}

	spans := exp.GetSpans()
	var failed int
	for _, s := range spans {
		if s.Status.Code == codes.Error {
			failed++
		}
		if text := spanText(s); strings.Contains(text, payloadMarker) {
			t.Errorf("exported span %q carries the payload value:\n%s", s.Name, text)
		}
	}
	if failed == 0 {
		t.Errorf("no exported span has an Error status; spans: %d", len(spans))
	}
	for _, s := range spans {
		for _, kv := range s.Attributes {
			if kv.Key == "temporalWorkflowID" && kv.Value.AsString() != id {
				t.Errorf("span %q workflow id %q, want %q", s.Name, kv.Value.AsString(), id)
			}
		}
	}
}
