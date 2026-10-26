//go:build temporalstack

package temporalstack

// CI-7 (ADR-001 Annex B, R1; PLAN-001 task T32b, audit TICK-93): Temporal
// histories are cross-tenant and unencrypted, so the workflow input, signal
// bodies, activity payloads and headers they record must hold references
// only, never record data, and headers only the tenant and trace context,
// never baggage. One named test per case, each named TestCI7_*;
// tools/ci/ci6report lists them.

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/baggage"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The producer's schema (outbox.ValidatePayload, used by outbox.Enqueue)
// rejects every record-data shape: free text, numbers, nested objects, a
// reference field next to a text field, and anything that is not an object.
// Only UUIDs and arrays of UUIDs pass.
func TestCI7_ProducerSchemaRejectsRecordDataFields(t *testing.T) {
	ref := uuid.NewString()
	for _, bad := range []string{
		`{"title":"` + ci7Marker + `"}`,
		`{"record_id":"` + ref + `","note":"free text"}`,
		`{"amount":42}`,
		`{"approved":true}`,
		`{"customer":{"email":"alice@tenant-a.test"}}`,
		`{"watchers":["` + ref + `","bob@tenant-a.test"]}`,
		`{"record_id":"` + strings.ToUpper(ref) + ` "}`,
		`{"record_id":null}`,
		`["` + ref + `"]`,
		`"` + ref + `"`,
		`null`,
		`not json`,
	} {
		if err := outbox.ValidatePayload(json.RawMessage(bad)); !errors.Is(err, outbox.ErrInvalidEntry) {
			t.Errorf("payload %s accepted (%v), want ErrInvalidEntry", bad, err)
		}
	}
	for _, ok := range []string{`{}`, `{"record_id":"` + ref + `"}`, `{"watchers":["` + ref + `"],"record_id":"` + ref + `"}`} {
		if err := outbox.ValidatePayload(json.RawMessage(ok)); err != nil {
			t.Errorf("references-only payload %s rejected: %v", ok, err)
		}
	}
}

// deliveryField reports whether k is a key of outbox.Delivery's JSON shape:
// references and counters.
func deliveryField(k string) bool {
	switch k {
	case "outbox_id", "kind_version", "causation_id", "causation_depth", "payload":
		return true
	}
	return false
}

// What the relay records in a history, the input of the start and the body
// of every signal, is an outbox.Delivery with nothing but references: its Go
// type has only UUID, integer and payload fields, and the recorded JSON of a
// relay-started workflow has only those keys, with a payload that passes the
// producer's schema.
func TestCI7_RelayDeliveryInHistoryIsReferencesOnly(t *testing.T) {
	allowed := map[reflect.Type]bool{
		reflect.TypeFor[uuid.UUID](): true, reflect.TypeFor[*uuid.UUID](): true,
		reflect.TypeFor[int16](): true, reflect.TypeFor[json.RawMessage](): true,
	}
	dt := reflect.TypeFor[outbox.Delivery]()
	for i := range dt.NumField() {
		f := dt.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if !allowed[f.Type] || !deliveryField(name) {
			t.Errorf("outbox.Delivery field %s (%s, json %q) is not a reference, counter or checked payload", f.Name, f.Type, name)
		}
	}

	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	payload := `{"record_id":"` + tg.record.String() + `","watchers":["` + uuid.NewString() + `"]}`
	_, noteSQL := tg.signalSQL(t, sigNote)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, tg.startSQL(payload)+noteSQL+doneSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.result(ctx, t, id)

	var deliveries int
	for _, ev := range history(ctx, t, s, id, "") {
		var ps []*commonpb.Payload
		switch {
		case ev.GetWorkflowExecutionStartedEventAttributes() != nil:
			ps = ev.GetWorkflowExecutionStartedEventAttributes().GetInput().GetPayloads()
		case ev.GetWorkflowExecutionSignaledEventAttributes() != nil:
			ps = ev.GetWorkflowExecutionSignaledEventAttributes().GetInput().GetPayloads()
		default:
			continue
		}
		for _, p := range ps {
			deliveries++
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(p.GetData(), &fields); err != nil {
				t.Fatalf("%v input is not a JSON object: %v", ev.GetEventType(), err)
			}
			for k := range fields {
				if !deliveryField(k) {
					t.Errorf("%v input has field %q, not an outbox.Delivery reference", ev.GetEventType(), k)
				}
			}
			if err := outbox.ValidatePayload(fields["payload"]); err != nil {
				t.Errorf("%v payload %s: %v", ev.GetEventType(), fields["payload"], err)
			}
		}
	}
	if deliveries != 3 {
		t.Errorf("%d deliveries in the history, want 3 (start, note, done)", deliveries)
	}
}

// Every header recorded in a history, on the start, signals, the activity,
// continue-as-new and the new run, carries only nava-tenant and the trace
// context (_tracer-data with traceparent and tracestate), never baggage:
// not even when the signalling client's context holds baggage.
func TestCI7_RecordedHeadersCarryOnlyTenantAndTraceContext(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	n1, n1SQL := tg.signalSQL(t, sigNote)
	psqlSuper(t, tg.startSQL(tg.refs())+n1SQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	if r := awaitSettled(t, n1); !r.dispatched {
		t.Fatalf("note row %+v, want dispatched", r)
	}
	firstRun := s.awaitStarted(ctx, t, id)

	// A platform client whose context carries baggage (record data, as an
	// upstream service might add) signals the workflow with its own tenant.
	member, err := baggage.NewMemberRaw("ci7-title", ci7Marker)
	if err != nil {
		t.Fatal(err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatal(err)
	}
	bctx := workerguard.ContextWithTenant(baggage.ContextWithBaggage(ctx, bag), tg.tenant)
	if err := s.relay.SignalWorkflow(bctx, id, "", sigNote, outbox.Delivery{OutboxID: uuid.New()}); err != nil {
		t.Fatalf("signal with baggage: %v", err)
	}

	// The rollover goes alone: a done buffered in the same workflow task
	// would be dropped by the continue-as-new, so wait for the new run first.
	roll, rollSQL := tg.signalSQL(t, sigRollover)
	psqlSuper(t, rollSQL)
	if r := awaitSettled(t, roll); !r.dispatched {
		t.Fatalf("rollover row %+v, want dispatched", r)
	}
	deadline := time.Now().Add(60 * time.Second)
	for s.awaitStarted(ctx, t, id) == firstRun {
		if time.Now().After(deadline) {
			t.Fatalf("no new run of %s after continue-as-new within 60 s", id)
		}
		time.Sleep(200 * time.Millisecond)
	}
	_, n2SQL := tg.signalSQL(t, sigNote)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, n2SQL+doneSQL)
	s.result(ctx, t, id)

	var tenantHeaders int
	for _, run := range []string{firstRun, ""} {
		for _, ev := range history(ctx, t, s, id, run) {
			b, err := proto.Marshal(ev)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), "ci7-title") || strings.Contains(string(b), ci7Marker) {
				t.Errorf("event %d %v records the client's baggage", ev.GetEventId(), ev.GetEventType())
			}
			for _, h := range headersIn(ev.ProtoReflect()) {
				for k, p := range h.GetFields() {
					switch k {
					case "nava-tenant":
						tenantHeaders++
					case "_tracer-data":
						var tc map[string]string
						if err := json.Unmarshal(p.GetData(), &tc); err != nil {
							t.Errorf("event %d %v _tracer-data is not a string map: %v", ev.GetEventId(), ev.GetEventType(), err)
						}
						for tk := range tc {
							if tk != "traceparent" && tk != "tracestate" {
								t.Errorf("event %d %v _tracer-data carries %q, want traceparent and tracestate only", ev.GetEventId(), ev.GetEventType(), tk)
							}
						}
					default:
						t.Errorf("event %d %v header %q, want nava-tenant and _tracer-data only", ev.GetEventId(), ev.GetEventType(), k)
					}
				}
			}
		}
	}
	// Start, three signals and continue-as-new in the first run; at least the
	// new run's start, a signal and the final activity in the second.
	if tenantHeaders < 8 {
		t.Errorf("%d recorded nava-tenant headers, want at least 8: the walk found too few headers", tenantHeaders)
	}
}

// history is every event of the run (the latest when run is ""), read by
// nava-core.
func history(ctx context.Context, t *testing.T, s relayStack, id, run string) []*historypb.HistoryEvent {
	t.Helper()
	hctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var out []*historypb.HistoryEvent
	it := s.core.GetWorkflowHistory(hctx, id, run, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for it.HasNext() {
		ev, err := it.Next()
		if err != nil {
			t.Fatalf("history of %s %s: %v", id, run, err)
		}
		out = append(out, ev)
	}
	return out
}

// headersIn returns every temporal.api.common.v1.Header anywhere in m.
func headersIn(m protoreflect.Message) []*commonpb.Header {
	var out []*commonpb.Header
	headerName := (&commonpb.Header{}).ProtoReflect().Descriptor().FullName()
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() != protoreflect.MessageKind {
			return true
		}
		visit := func(sub protoreflect.Message) {
			if sub.Descriptor().FullName() == headerName {
				if h, ok := sub.Interface().(*commonpb.Header); ok {
					out = append(out, h)
				}
				return
			}
			out = append(out, headersIn(sub)...)
		}
		switch {
		case fd.IsList():
			for i := range v.List().Len() {
				visit(v.List().Get(i).Message())
			}
		case fd.IsMap():
			if fd.MapValue().Kind() == protoreflect.MessageKind {
				v.Map().Range(func(_ protoreflect.MapKey, mv protoreflect.Value) bool { visit(mv.Message()); return true })
			}
		default:
			visit(v.Message())
		}
		return true
	})
	return out
}
