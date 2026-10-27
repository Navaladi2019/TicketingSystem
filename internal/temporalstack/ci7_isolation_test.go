//go:build temporalstack

package temporalstack

// CI-7 (ADR-001 Annex B, R1; PLAN-001 task T32b, audit TICK-93): Temporal
// histories are cross-tenant and unencrypted, so the workflow input, signal
// bodies, activity payloads and headers they record must hold references
// only, never record data, and headers only the tenant and trace context,
// never baggage. The activity, child, continue-as-new and update cases are
// TICK-216's: the worker guard refuses record-data arguments (argOK). The
// signal-external, signal-child, memo, result, side-effect and failure cases
// are TICK-245's. One named test per case, each named TestCI7_*;
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
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
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

// A start row with record data in its payload, written directly rather than
// through outbox.Enqueue (finding T32b-1), is refused by the database
// (outbox_payload_check, TICK-214 and TICK-240), so the relay never sees it and
// nothing reaches a history. The superuser skips RLS but not the CHECK.
func TestCI7_RelayRefusesRecordDataPayloadWrittenOutsideEnqueue(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	out, err := psqlSuperErr(t, tg.startSQL(`{"record_id":"`+tg.record.String()+`","title":"`+ci7Marker+`"}`))
	if err == nil || !strings.Contains(out, "outbox_payload_check") {
		t.Fatalf("direct INSERT with record data: %v\n%s\nwant a violation of outbox_payload_check", err, out)
	}
	if n := psqlSuper(t, "SELECT count(*) FROM outbox WHERE id = '"+tg.start.String()+"';"); n != "0" {
		t.Fatalf("%s outbox rows stored after the refused INSERT", n)
	}
	if _, err := s.core.DescribeWorkflowExecution(ctx, id, ""); err == nil {
		t.Fatalf("workflow %s exists; its history may hold %q", id, ci7Marker)
	} else if nf := (*serviceerror.NotFound)(nil); !errors.As(err, &nf) {
		t.Fatalf("describe %s: %v", id, err)
	}
}

// wantMarkerNotRecorded fails if any event of the run holds ci7Marker.
func wantMarkerNotRecorded(ctx context.Context, t *testing.T, s relayStack, id string) []*historypb.HistoryEvent {
	t.Helper()
	events := history(ctx, t, s, id, "")
	for _, ev := range events {
		b, err := proto.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), ci7Marker) {
			t.Fatalf("event %d %v records the record data %q", ev.GetEventId(), ev.GetEventType(), ci7Marker)
		}
	}
	return events
}

// wantGuardAlert fails unless the guard alerted guard/record_data for id:
// the guard refused the argument, not something else on the way.
func wantGuardAlert(t *testing.T, s relayStack, id, guard string) {
	t.Helper()
	if n := guardAlerts(s.logs, id, guard, "record_data"); n < 1 {
		t.Errorf("no %s/record_data guard alert for %s; log:\n%s", guard, id, s.logs.String())
	}
}

func countEvents(events []*historypb.HistoryEvent, et enumspb.EventType) int {
	n := 0
	for _, ev := range events {
		if ev.GetEventType() == et {
			n++
		}
	}
	return n
}

// An activity scheduled with record data in its argument must never be
// recorded: the history of the workflow must not hold the text.
func TestCI7_ActivityPayloadWithRecordDataNotRecorded(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	_, actSQL := tg.signalSQL(t, sigCI7Activity)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, tg.startSQL(tg.refs())+actSQL+doneSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.awaitStarted(ctx, t, id)
	// The result does not matter: a guard may fail the workflow instead.
	_ = s.core.GetWorkflow(ctx, id, "").Get(ctx, nil)

	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED); n != 1 {
		t.Errorf("%d activities scheduled, want 1: the workflow_ref close only", n)
	}
	wantGuardAlert(t, s, id, "activity_args")
}

// A child started with record data in its argument must never be recorded:
// no child is initiated, and the parent's history does not hold the text.
func TestCI7_ChildPayloadWithRecordDataNotRecorded(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	_, childSQL := tg.signalSQL(t, sigCI7Child)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, tg.startSQL(tg.refs())+childSQL+doneSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.result(ctx, t, id)

	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED); n != 0 {
		t.Errorf("%d children initiated, want 0", n)
	}
	wantGuardAlert(t, s, id, "child")
}

// A continue-as-new with record data in its input (here in RunInput.Refs,
// the one place a workflow may carry its own state) must never be recorded:
// the workflow fails instead of continuing.
func TestCI7_ContinueAsNewPayloadWithRecordDataNotRecorded(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	_, rollSQL := tg.signalSQL(t, sigCI7Rollover)
	psqlSuper(t, tg.startSQL(tg.refs())+rollSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.awaitStarted(ctx, t, id)
	err := s.core.GetWorkflow(ctx, id, "").Get(ctx, nil)
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeTenantGuard {
		t.Fatalf("result: %v; want the guard's failure", err)
	}

	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW); n != 0 {
		t.Errorf("%d continue-as-new events, want 0", n)
	}
	if n := countEvents(events, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED); n != 1 {
		t.Errorf("%d failed events, want 1", n)
	}
	wantGuardAlert(t, s, id, "continue_as_new")
}

// An update from nava-core, with the right tenant, whose argument holds
// record data is rejected in the validator and never recorded. A
// references-only update on the same workflow is accepted.
func TestCI7_UpdatePayloadWithRecordDataNotRecorded(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	psqlSuper(t, tg.startSQL(tg.refs()))
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.awaitStarted(ctx, t, id)
	own := workerguard.ContextWithTenant(ctx, tg.tenant)
	update := func(arg map[string]string) error {
		t.Helper()
		// SDK v1.49 keeps a rejection on the handle (see workerguard_test.go).
		h, err := s.core.UpdateWorkflow(own, client.UpdateWorkflowOptions{
			WorkflowID: id, UpdateName: updCI7, Args: []any{arg}, WaitForStage: client.WorkflowUpdateStageCompleted,
		})
		if err != nil {
			return err
		}
		return h.Get(ctx, nil)
	}

	err := update(map[string]string{"title": ci7Marker})
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeTenantGuard ||
		!strings.Contains(app.Error(), "tenant guard denied update: record_data") {
		t.Fatalf("record-data update: %v; want the validator's rejection", err)
	}
	if strings.Contains(err.Error(), ci7Marker) {
		t.Errorf("the rejection holds the record data: %v", err)
	}
	if err := update(map[string]string{"record_id": tg.record.String()}); err != nil {
		t.Fatalf("references-only update: %v", err)
	}
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, doneSQL)
	s.result(ctx, t, id)

	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED); n != 1 {
		t.Errorf("%d updates accepted, want 1: the references-only one", n)
	}
	wantGuardAlert(t, s, id, "update")
}

// probeChildID is the id of the one child the probe initiated, read from
// its history; "" if none.
func probeChildID(events []*historypb.HistoryEvent) string {
	for _, ev := range events {
		if a := ev.GetStartChildWorkflowExecutionInitiatedEventAttributes(); a != nil {
			return a.GetWorkflowId()
		}
	}
	return ""
}

// runCI7Signal sends the probe signal kind, which starts a references-only
// child and signals it record data, then closes the probe. The record data
// must be in neither history: not the sender's, where
// SignalExternalWorkflowExecutionInitiated would record it before the
// receiver's guard ran, nor the child's.
func runCI7Signal(t *testing.T, kind string) {
	t.Helper()
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	_, sigSQL := tg.signalSQL(t, kind)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, tg.startSQL(tg.refs())+sigSQL+doneSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.result(ctx, t, id)

	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED); n != 1 {
		t.Fatalf("%d children initiated, want 1: the signal's references-only target", n)
	}
	if n := countEvents(events, enumspb.EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED); n != 0 {
		t.Errorf("%d signals initiated, want 0", n)
	}
	child := probeChildID(events)
	s.terminateAtEnd(t, child)
	childEvents := wantMarkerNotRecorded(ctx, t, s, child)
	if n := countEvents(childEvents, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED); n != 0 {
		t.Errorf("child %s was signaled %d times, want 0", child, n)
	}
	wantGuardAlert(t, s, id, "signal_external")
}

// A signal-external with record data in its argument must never be recorded,
// in the sender's history or the target's (TICK-245).
func TestCI7_SignalExternalPayloadWithRecordDataNotRecorded(t *testing.T) {
	runCI7Signal(t, sigCI7SignalExternal)
}

// A signal-child with record data in its argument must never be recorded,
// in the parent's history or the child's (TICK-245).
func TestCI7_SignalChildPayloadWithRecordDataNotRecorded(t *testing.T) {
	runCI7Signal(t, sigCI7SignalChild)
}

// A child started with record data in its memo must never be recorded: no
// child is initiated (TICK-245).
func TestCI7_ChildMemoWithRecordDataNotRecorded(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	_, memoSQL := tg.signalSQL(t, sigCI7ChildMemo)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, tg.startSQL(tg.refs())+memoSQL+doneSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.result(ctx, t, id)

	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED); n != 0 {
		t.Errorf("%d children initiated, want 0", n)
	}
	wantGuardAlert(t, s, id, "child")
}

// An UpsertMemo with record data must never be recorded: no
// WorkflowPropertiesModified event, and the memo stays empty (TICK-245).
func TestCI7_UpsertMemoWithRecordDataNotRecorded(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	_, memoSQL := tg.signalSQL(t, sigCI7UpsertMemo)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, tg.startSQL(tg.refs())+memoSQL+doneSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.result(ctx, t, id)

	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_WORKFLOW_PROPERTIES_MODIFIED); n != 0 {
		t.Errorf("%d memo upserts recorded, want 0", n)
	}
	desc, err := s.core.DescribeWorkflowExecution(ctx, id, "")
	if err != nil {
		t.Fatal(err)
	}
	if f := desc.GetWorkflowExecutionInfo().GetMemo().GetFields(); len(f) != 0 {
		t.Errorf("memo has %d fields, want none", len(f))
	}
	wantGuardAlert(t, s, id, "memo")
}

// runCI7Probe sends the probe signal kind, then done, and returns the stack,
// the workflow id and how the run ended.
func runCI7Probe(t *testing.T, kind string) (context.Context, relayStack, string, error) {
	t.Helper()
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	_, kindSQL := tg.signalSQL(t, kind)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, tg.startSQL(tg.refs())+kindSQL+doneSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.awaitStarted(ctx, t, id)
	return ctx, s, id, s.core.GetWorkflow(ctx, id, "").Get(ctx, nil)
}

// wantWithheld fails unless the guard logged a withheld line for guard and
// id, and no guard log line holds the record data: the log gets the
// failure's shape, never its text.
func wantWithheld(t *testing.T, s relayStack, id, guard string) {
	t.Helper()
	var n int
	for line := range strings.Lines(s.logs.String()) {
		if strings.Contains(line, "withheld a failure") && strings.Contains(line, `"guard":"`+guard+`"`) && strings.Contains(line, id) {
			n++
		}
	}
	if n < 1 {
		t.Errorf("no %s withheld line for %s; log:\n%s", guard, id, s.logs.String())
	}
	if strings.Contains(s.logs.String(), ci7Marker) {
		t.Errorf("the guard log holds the record data")
	}
}

// wantOpaque fails unless err is the withheld failure, with no record data.
func wantOpaque(t *testing.T, err error) {
	t.Helper()
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeOpaque {
		t.Fatalf("error %v, want the withheld %s failure", err, workerguard.ErrTypeOpaque)
	}
	if strings.Contains(err.Error(), ci7Marker) {
		t.Errorf("the failure holds the record data: %v", err)
	}
}

// A local activity that returns record data must never have it recorded: the
// result is replaced by the guard error (TICK-245, triage row 3).
func TestCI7_ActivityResultWithRecordDataNotRecorded(t *testing.T) {
	ctx, s, id, err := runCI7Probe(t, sigCI7ActivityResult)
	if err != nil {
		t.Fatalf("result: %v", err)
	}
	wantMarkerNotRecorded(ctx, t, s, id)
	wantGuardAlert(t, s, id, "activity_result")
}

// A local activity that fails with record data in its message must never
// have it recorded: the failure is withheld, and the log has its shape only
// (TICK-245, triage row 7).
func TestCI7_ActivityFailureWithRecordDataNotRecorded(t *testing.T) {
	ctx, s, id, err := runCI7Probe(t, sigCI7ActivityFailure)
	if err != nil {
		t.Fatalf("result: %v", err)
	}
	wantMarkerNotRecorded(ctx, t, s, id)
	wantGuardAlert(t, s, id, "activity_failure")
	wantWithheld(t, s, id, "activity_failure")
}

// A SideEffect whose value is record data must never have it recorded: the
// marker holds the guard's sentinel instead (TICK-245, triage row 6).
func TestCI7_SideEffectWithRecordDataNotRecorded(t *testing.T) {
	ctx, s, id, err := runCI7Probe(t, sigCI7SideEffect)
	if err != nil {
		t.Fatalf("result: %v", err)
	}
	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_MARKER_RECORDED); n != 1 {
		t.Errorf("%d markers recorded, want 1: the side effect's, holding the sentinel", n)
	}
	wantGuardAlert(t, s, id, "side_effect")
}

// A workflow that returns record data must never have it recorded: it fails
// with the guard error instead (TICK-245, triage row 4).
func TestCI7_WorkflowResultWithRecordDataNotRecorded(t *testing.T) {
	ctx, s, id, err := runCI7Probe(t, sigCI7Result)
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeTenantGuard ||
		!strings.Contains(app.Error(), "tenant guard denied workflow_result: record_data") {
		t.Fatalf("result: %v; want the guard's failure", err)
	}
	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED); n != 0 {
		t.Errorf("%d completed events, want 0", n)
	}
	wantGuardAlert(t, s, id, "workflow_result")
}

// A workflow that fails with record data in its message must never have it
// recorded: the failure is withheld (TICK-245, triage row 7).
func TestCI7_WorkflowFailureWithRecordDataNotRecorded(t *testing.T) {
	ctx, s, id, err := runCI7Probe(t, sigCI7Failure)
	wantOpaque(t, err)
	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED); n != 1 {
		t.Errorf("%d failed events, want 1", n)
	}
	wantGuardAlert(t, s, id, "workflow_failure")
	wantWithheld(t, s, id, "workflow_failure")
}

// An update that returns record data, or fails with it, must never have it
// recorded: the result is replaced by the guard error, the failure withheld
// (TICK-245, triage rows 5 and 7).
func TestCI7_UpdateResultWithRecordDataNotRecorded(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	psqlSuper(t, tg.startSQL(tg.refs()))
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	s.awaitStarted(ctx, t, id)
	own := workerguard.ContextWithTenant(ctx, tg.tenant)
	update := func(name string) error {
		t.Helper()
		h, err := s.core.UpdateWorkflow(own, client.UpdateWorkflowOptions{
			WorkflowID: id, UpdateName: name, WaitForStage: client.WorkflowUpdateStageCompleted,
		})
		if err != nil {
			return err
		}
		return h.Get(ctx, nil)
	}

	err := update(updCI7Result)
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeTenantGuard ||
		!strings.Contains(app.Error(), "tenant guard denied update_result: record_data") {
		t.Fatalf("record-data update result: %v; want the guard error", err)
	}
	wantOpaque(t, update(updCI7Failure))
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, doneSQL)
	s.result(ctx, t, id)

	events := wantMarkerNotRecorded(ctx, t, s, id)
	if n := countEvents(events, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED); n != 2 {
		t.Errorf("%d updates completed, want 2", n)
	}
	wantGuardAlert(t, s, id, "update_result")
	wantGuardAlert(t, s, id, "update_failure")
	wantWithheld(t, s, id, "update_failure")
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
