package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

func validEntry() Entry {
	return Entry{
		Kind: "ticket.approval", KindVersion: 1, DedupeKey: "d1", RecordID: uuid.New(),
		Target: TargetStart{Kind: "ticket.approval"},
	}
}

// Every case is rejected before Enqueue touches tx, so a nil tx proves
// nothing was written.
func TestEnqueueRejectsBeforeAnyWrite(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Entry)
		want   error
	}{
		{"CI-10 depth beyond limit", func(e *Entry) { e.Causation = Causation{OriginID: uuid.New(), Depth: 9} }, ErrCascadeTooDeep},
		{"negative depth", func(e *Entry) { e.Causation.Depth = -1 }, ErrInvalidEntry},
		{"depth with no origin", func(e *Entry) { e.Causation.Depth = 1 }, ErrInvalidEntry},
		{"empty kind", func(e *Entry) { e.Kind = ""; e.Target = TargetStart{} }, ErrInvalidEntry},
		{"free-text kind", func(e *Entry) { e.Kind = "Jane Doe"; e.Target = TargetStart{Kind: "Jane Doe"} }, ErrInvalidEntry},
		{"kind with a colon", func(e *Entry) { e.Kind = "a:b"; e.Target = TargetStart{Kind: "a:b"} }, ErrInvalidEntry},
		{"kind version 0", func(e *Entry) { e.KindVersion = 0 }, ErrInvalidEntry},
		{"empty dedupe key", func(e *Entry) { e.DedupeKey = "" }, ErrInvalidEntry},
		{"long dedupe key", func(e *Entry) { e.DedupeKey = strings.Repeat("k", 201) }, ErrInvalidEntry},
		{"no record", func(e *Entry) { e.RecordID = uuid.Nil }, ErrInvalidEntry},
		{"no target", func(e *Entry) { e.Target = nil }, ErrInvalidEntry},
		{"start kind differs", func(e *Entry) { e.Target = TargetStart{Kind: "other"} }, ErrInvalidEntry},
		{"signal name differs", func(e *Entry) { e.Target = TargetSignal{Kind: "ticket.approval", SignalName: "approve"} }, ErrInvalidEntry},
		{"signal kind free text", func(e *Entry) { e.Kind = "approve"; e.Target = TargetSignal{Kind: "A B", SignalName: "approve"} }, ErrInvalidEntry},
		{"record data in payload", func(e *Entry) { e.Payload = json.RawMessage(`{"title":"printer on fire"}`) }, ErrInvalidEntry},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := validEntry()
			c.mutate(&e)
			id, err := Enqueue(context.Background(), nil, e)
			if !errors.Is(err, c.want) || id != uuid.Nil {
				t.Fatalf("Enqueue = %v, %v; want %v", id, err, c.want)
			}
		})
	}
}

func TestDepthAtLimitPassesValidation(t *testing.T) {
	e := validEntry()
	e.Causation = Causation{OriginID: uuid.New(), Depth: 8}
	if _, err := validate(e); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePayload(t *testing.T) {
	id := uuid.NewString()
	for _, ok := range []string{`{}`, `{"ticket_id":"` + id + `"}`, `{"ids":["` + id + `","` + id + `"]}`, `{"ids":[]}`} {
		if err := ValidatePayload(json.RawMessage(ok)); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{
		`null`, `[]`, `"x"`, `{"n":1}`, `{"b":true}`, `{"s":"text"}`, `{"o":{"id":"` + id + `"}}`,
		`{"ids":["` + id + `","x"]}`, `{"z":null}`, `{`,
		// Keys (TICK-240).
		`{"printer on fire":"` + id + `"}`, `{"Ticket_id":"` + id + `"}`, `{"ticket.id":"` + id + `"}`,
		`{"1d":"` + id + `"}`, `{"":"` + id + `"}`, `{"` + strings.Repeat("a", 33) + `":"` + id + `"}`,
		// Non-canonical values (TICK-240).
		`{"ticket_id":"` + strings.ToUpper(id) + `"}`, `{"ticket_id":"urn:uuid:` + id + `"}`,
		`{"ticket_id":"{` + id + `}"}`, `{"ticket_id":"` + strings.ReplaceAll(id, "-", "") + `"}`,
		`{"ids":["` + id + `","` + strings.ToUpper(id) + `"]}`,
	} {
		if err := ValidatePayload(json.RawMessage(bad)); !errors.Is(err, ErrInvalidEntry) {
			t.Errorf("%s: %v, want ErrInvalidEntry", bad, err)
		}
	}
	if err := ValidatePayload(json.RawMessage(`{"` + strings.Repeat("a", 32) + `":"` + id + `"}`)); err != nil {
		t.Errorf("32-character key: %v", err)
	}
}

// A refusal never quotes the payload, not even the key (TICK-240).
func TestValidatePayloadErrorOmitsPayload(t *testing.T) {
	const marker = "printer on fire alice"
	for _, p := range []string{
		`{"` + marker + `":"` + uuid.NewString() + `"}`,
		`{"note":"` + marker + `"}`,
		`{"ids":["` + marker + `"]}`,
	} {
		err := ValidatePayload(json.RawMessage(p))
		if err == nil {
			t.Fatalf("%s accepted", p)
		}
		if strings.Contains(err.Error(), "printer") {
			t.Errorf("error %q quotes the payload", err)
		}
	}
}

func TestTraceparentOnlyTraceContext(t *testing.T) {
	if tp := traceparent(context.Background()); tp != nil {
		t.Fatalf("no span: %q, want nil", *tp)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3},
		SpanID:     trace.SpanID{4, 5, 6},
		TraceFlags: trace.FlagsSampled,
		TraceState: mustTraceState(t, "vendor=secret"),
	})
	member, err := baggage.NewMember("user", "jane@example.com")
	if err != nil {
		t.Fatal(err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatal(err)
	}
	ctx := baggage.ContextWithBaggage(trace.ContextWithSpanContext(context.Background(), sc), bag)
	tp := traceparent(ctx)
	want := "00-01020300000000000000000000000000-0405060000000000-01"
	if tp == nil || *tp != want {
		t.Fatalf("traceparent = %v, want %s (no tracestate, no baggage)", tp, want)
	}
}

func mustTraceState(t *testing.T, s string) trace.TraceState {
	t.Helper()
	ts, err := trace.ParseTraceState(s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}
