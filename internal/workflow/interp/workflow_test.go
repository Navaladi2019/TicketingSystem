package interp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

const (
	coolOff = 5 * time.Second
	day     = 24 * time.Hour
)

// v1 sets up fixture-v1 for tenant A with a record of the given priority.
func v1(t *testing.T, priority string) (*harness, fixture, uuid.UUID, uuid.UUID) {
	t.Helper()
	h := newHarness(t, tenantA)
	f := fixtureV1(t)
	version, record := uuid.New(), uuid.New()
	h.install(tenantA, version, f)
	h.putRecord(tenantA, record, map[string]any{"priority": priority})
	return h, f, version, record
}

// TestFixtureV1Branches: fixture-v1 completes on each branch, and the
// cool_off timer fires before the condition is evaluated.
func TestFixtureV1Branches(t *testing.T) {
	t.Run("else: low priority ends after the timer", func(t *testing.T) {
		h, f, version, record := v1(t, "low")
		h.start(version, record)
		h.wantCompleted()
		if got := h.timerDurations(); !slices.Equal(got, []time.Duration{coolOff}) {
			t.Errorf("timers %v, want [%v]", got, coolOff)
		}
		if h.elapsed() != coolOff {
			t.Errorf("elapsed %v, want %v: the timer fires", h.elapsed(), coolOff)
		}
		if c := h.calls(); len(c) != 0 {
			t.Errorf("writes %v, want none", c)
		}
		refs := h.nextRefs()
		want := []uuid.UUID{uuid.Nil, f.id(t, "is_high")}
		if len(refs) != len(want) {
			t.Fatalf("NextStep refs %+v, want steps %v", refs, want)
		}
		for i, r := range refs {
			if r.Step != want[i] || r.DefinitionVersionID != version || r.RecordID != record || r.Signal != nil {
				t.Errorf("NextStep %d: %+v, want step %s, no signal", i, r, want[i])
			}
		}
	})

	t.Run("then: approve applies mark_approved", func(t *testing.T) {
		h, f, version, record := v1(t, "high")
		h.signalAt(time.Hour, uuid.New(), f.id(t, def.SignalKey("approve")))
		h.start(version, record)
		h.wantCompleted()
		calls := h.calls()
		if len(calls) != 1 {
			t.Fatalf("writes %+v, want 1", calls)
		}
		wantKey := interp.IdempotencyKey(h.workflowID, f.id(t, "mark_approved"))
		c := calls[0]
		if c.Key != wantKey || c.Attempt != 1 || c.Tenant != tenantA || c.Record != record || c.Set["approved"] != true || !c.Applied {
			t.Errorf("write %+v, want key %s attempt 1, set approved", c, wantKey)
		}
		if got := h.timerDurations(); !slices.Equal(got, []time.Duration{coolOff, day}) {
			t.Errorf("timers %v, want [%v %v]", got, coolOff, day)
		}
		if h.elapsed() != time.Hour {
			t.Errorf("elapsed %v, want 1h (the signal)", h.elapsed())
		}
	})

	t.Run("on_timeout: no signal takes on_timeout", func(t *testing.T) {
		h, _, version, record := v1(t, "high")
		h.start(version, record)
		h.wantCompleted()
		if c := h.calls(); len(c) != 0 {
			t.Errorf("writes %v, want none", c)
		}
		if h.elapsed() != coolOff+day {
			t.Errorf("elapsed %v, want %v: cool_off then the 1 day timeout", h.elapsed(), coolOff+day)
		}
	})
}

// TestSignalAbsentPassesGuard: every NextStep before the signal carries no
// Signal, its encoding omits the key (never null), and the guard accepts
// it; the one after the signal carries it.
func TestSignalAbsentPassesGuard(t *testing.T) {
	h, f, version, record := v1(t, "high")
	sig := f.id(t, def.SignalKey("approve"))
	h.signalAt(time.Hour, uuid.New(), sig)
	h.start(version, record)
	h.wantCompleted()
	refs := h.nextRefs()
	if len(refs) != 4 {
		t.Fatalf("NextStep refs %+v, want 4", refs)
	}
	for i, r := range refs {
		withSignal := r.Signal != nil
		if want := i == 2; withSignal != want {
			t.Errorf("NextStep %d: Signal set %v, want %v", i, withSignal, want)
		}
		if withSignal && (*r.Signal != sig || r.Step != f.id(t, "mark_approved")) {
			t.Errorf("NextStep %d: %+v, want mark_approved with the approve signal", i, r)
		}
	}
	nulls := 0
	for _, d := range h.dc.recorded() {
		if strings.Contains(string(d), `"signal":null`) {
			nulls++
		}
	}
	if nulls != 0 {
		t.Errorf("%d recorded payloads carry a null signal", nulls)
	}
}

// TestSignalAppliedOnce: a redelivered signal (same outbox id) is applied
// once.
func TestSignalAppliedOnce(t *testing.T) {
	t.Run("fixture-v1: approve delivered twice writes once", func(t *testing.T) {
		h, f, version, record := v1(t, "high")
		sig, outboxID := f.id(t, def.SignalKey("approve")), uuid.New()
		h.signalAt(time.Hour, outboxID, sig)
		h.signalAt(time.Hour+time.Minute, outboxID, sig)
		h.start(version, record)
		h.wantCompleted()
		key := interp.IdempotencyKey(h.workflowID, f.id(t, "mark_approved"))
		if n, err := h.writer.Applications(key); err != nil || n != 1 {
			t.Errorf("applications %d (%v), want 1", n, err)
		}
		if c := h.calls(); len(c) != 1 {
			t.Errorf("writes %+v, want 1", c)
		}
	})

	t.Run("two waits: the redelivery does not satisfy the second", func(t *testing.T) {
		h := newHarness(t, tenantA)
		f := loadFixture(t, "testdata/twowaits")
		version, record := uuid.New(), uuid.New()
		h.install(tenantA, version, f)
		h.putRecord(tenantA, record, map[string]any{})
		sig, first := f.id(t, def.SignalKey("approve")), uuid.New()
		h.signalAt(time.Hour, first, sig)
		h.signalAt(time.Hour, first, sig) // the relay's redelivery
		h.signalAt(3*time.Hour, uuid.New(), sig)
		h.start(version, record)
		h.wantCompleted()
		if h.elapsed() != 3*time.Hour {
			t.Errorf("elapsed %v, want 3h: the second wait ends at the second outbox id, not the redelivery", h.elapsed())
		}
		if c := h.calls(); len(c) != 1 || c[0].Step != f.id(t, "apply") {
			t.Errorf("writes %+v, want apply once", c)
		}
	})

	t.Run("another signal id is not applied", func(t *testing.T) {
		h, _, version, record := v1(t, "high")
		h.signalAt(time.Hour, uuid.New(), uuid.New())
		h.start(version, record)
		h.wantCompleted()
		if c := h.calls(); len(c) != 0 {
			t.Errorf("writes %+v, want none: the signal was not approve", c)
		}
		if h.elapsed() != coolOff+day {
			t.Errorf("elapsed %v, want the timeout", h.elapsed())
		}
	})
}

// TestBranchFirstMatchingCaseWins: cases in array order, the first that
// holds wins (both hold for "high"), default when none holds.
func TestBranchFirstMatchingCaseWins(t *testing.T) {
	f := loadFixture(t, "testdata/branch")
	for priority, lane := range map[string]string{"high": "first_lane", "urgent": "first_lane", "low": "fallback_lane"} {
		t.Run(priority, func(t *testing.T) {
			h := newHarness(t, tenantA)
			version, record := uuid.New(), uuid.New()
			h.install(tenantA, version, f)
			h.putRecord(tenantA, record, map[string]any{"priority": priority})
			h.start(version, record)
			h.wantCompleted()
			c := h.calls()
			if len(c) != 1 || c[0].Step != f.id(t, lane) {
				t.Errorf("writes %+v, want one write of %s", c, lane)
			}
		})
	}
}

// TestContinueAsNewResumesAtSameStep: with the suggestion on from the start,
// the run continues as new after the first NextStep, carrying that step, and
// the next run resumes there and completes.
func TestContinueAsNewResumesAtSameStep(t *testing.T) {
	h, f, version, record := v1(t, "low")
	h.env.SetContinueAsNewSuggested(true)
	h.start(version, record)
	in := h.continued()
	var refs struct {
		Step uuid.UUID `json:"step"`
	}
	if err := json.Unmarshal(in.Refs, &refs); err != nil {
		t.Fatal(err)
	}
	if refs.Step != f.id(t, "cool_off") {
		t.Fatalf("continued at %s, want cool_off %s", refs.Step, f.id(t, "cool_off"))
	}
	if got := h.timerDurations(); len(got) != 0 {
		t.Errorf("timers %v before continue-as-new, want none: it happens before the wait starts", got)
	}
	h.wantNoDenies()

	next := newHarness(t, tenantA, sameStore(h))
	next.env.ExecuteWorkflow(interp.WorkflowType, in)
	next.wantCompleted()
	if r := next.nextRefs(); len(r) == 0 || r[0].Step != f.id(t, "cool_off") {
		t.Errorf("next run's first NextStep %+v, want cool_off", r)
	}
	if got := next.timerDurations(); !slices.Equal(got, []time.Duration{coolOff}) {
		t.Errorf("next run's timers %v, want [%v]", got, coolOff)
	}
}

// TestContinueAsNewWaitsForTheWaitToEnd: the suggestion turns on during a
// wait_signal wait; the run keeps waiting, the timeout is not restarted, and
// it continues as new only after the wait ends, at the step that follows.
func TestContinueAsNewWaitsForTheWaitToEnd(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		h, f, version, record := v1(t, "high")
		h.env.RegisterDelayedCallback(func() { h.env.SetContinueAsNewSuggested(true) }, time.Hour)
		h.start(version, record)
		in := h.continued()
		if got := h.timerDurations(); !slices.Equal(got, []time.Duration{coolOff, day}) {
			t.Errorf("timers %v, want [%v %v]: one timeout, not restarted", got, coolOff, day)
		}
		if h.elapsed() != coolOff+day {
			t.Errorf("continued at %v, want %v: after the timeout", h.elapsed(), coolOff+day)
		}
		var refs struct {
			Step uuid.UUID `json:"step"`
		}
		if err := json.Unmarshal(in.Refs, &refs); err != nil || refs.Step != f.id(t, "done") {
			t.Errorf("continued at %s (%v), want done", refs.Step, err)
		}
	})

	t.Run("signal, and a pending delivery carried", func(t *testing.T) {
		h := newHarness(t, tenantA)
		f := loadFixture(t, "testdata/twowaits")
		version, record := uuid.New(), uuid.New()
		h.install(tenantA, version, f)
		h.putRecord(tenantA, record, map[string]any{})
		sig, second := f.id(t, def.SignalKey("approve")), uuid.New()
		h.env.RegisterDelayedCallback(func() { h.env.SetContinueAsNewSuggested(true) }, time.Hour)
		h.signalAt(2*time.Hour, uuid.New(), sig)
		h.signalAt(2*time.Hour, second, sig)
		h.start(version, record)
		in := h.continued()
		if h.elapsed() != 2*time.Hour {
			t.Errorf("continued at %v, want 2h: when the first wait ended", h.elapsed())
		}
		if c := h.calls(); len(c) != 0 {
			t.Errorf("writes %+v before continue-as-new, want none", c)
		}
		if in.Seen.Len() != 1 {
			t.Errorf("seen %d ids, want the applied one", in.Seen.Len())
		}

		next := newHarness(t, tenantA, sameStore(h))
		next.env.SetStartTime(t0.Add(2 * time.Hour))
		next.env.ExecuteWorkflow(interp.WorkflowType, in)
		next.wantCompleted()
		if got := next.timerDurations(); len(got) != 0 {
			t.Errorf("next run's timers %v, want none: the carried delivery satisfies the second wait", got)
		}
		if c := next.calls(); len(c) != 1 || c[0].Step != f.id(t, "apply") {
			t.Errorf("next run's writes %+v, want apply once", c)
		}
	})
}

// literals is every field name and literal of a definition: step keys,
// signal names, the trigger entity, condition fields (each pointer token)
// and values, and update_record set keys and values.
func literals(t *testing.T, raw []byte) []string {
	t.Helper()
	var d struct {
		Trigger struct {
			Entity string `json:"entity"`
		} `json:"trigger"`
		Steps []map[string]json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	out := []string{d.Trigger.Entity}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			for tok := range strings.SplitSeq(strings.TrimPrefix(x, "/"), "/") {
				out = append(out, tok)
			}
		case bool, float64:
			out = append(out, fmt.Sprint(x))
		case []any:
			for _, e := range x {
				walk(e)
			}
		case map[string]any:
			for k, e := range x {
				out = append(out, k)
				walk(e)
			}
		}
	}
	for _, s := range d.Steps {
		for name, v := range s {
			if name == "type" || name == "duration" || name == "timeout" || name == "op" {
				continue // schema words and durations, not author data
			}
			var x any
			if err := json.Unmarshal(v, &x); err != nil {
				t.Fatal(err)
			}
			walk(x)
		}
	}
	return out
}

// TestHistoryHoldsNoDefinitionLiteral: everything a fixture-v1 run records
// (every encoded argument, result and input) holds no field name or literal
// of the definition, nor the record's value.
func TestHistoryHoldsNoDefinitionLiteral(t *testing.T) {
	h, f, version, record := v1(t, "high")
	h.signalAt(time.Hour, uuid.New(), f.id(t, def.SignalKey("approve")))
	h.start(version, record)
	h.wantCompleted()
	text := h.recordedText()
	lits := literals(t, f.def)
	for _, want := range []string{"approved", "true", "high", "priority", "approve", "ticket", "mark_approved"} {
		if !slices.Contains(lits, want) {
			t.Fatalf("literal scan misses %q: %v", want, lits)
		}
	}
	for _, lit := range lits {
		if lit != "" && strings.Contains(text, lit) {
			t.Errorf("recorded history holds the definition literal %q", lit)
		}
	}
	if len(h.dc.recorded()) < 8 {
		t.Errorf("only %d payloads recorded; the scan saw too little", len(h.dc.recorded()))
	}
}

// markerReader fails NextStep with the record's field value in the error
// text, as an adapter that quotes a bad condition operand might.
type markerReader struct{ interptest.FileRecords }

func (m markerReader) Read(ctx context.Context, tenant, record uuid.UUID) (map[string]any, error) {
	r, err := m.FileRecords.Read(ctx, tenant, record)
	if err != nil {
		return nil, err
	}
	return nil, temporal.NewNonRetryableApplicationError(fmt.Sprintf("condition /priority: cannot compare %v", r["priority"]), "adapter.Bad", nil)
}

// TestNextStepFailureWithheld (Security): a NextStep failure whose text holds
// a record value leaves no trace of it in the history; it fails the run as
// nava.Opaque.
func TestNextStepFailureWithheld(t *testing.T) {
	const marker = "MARKER-7f3c-record-value"
	h := newHarness(t, tenantA, withRecords(func(root string) interp.RecordReader {
		return markerReader{interptest.FileRecords{Root: root}}
	}))
	f := fixtureV1(t)
	version, record := uuid.New(), uuid.New()
	h.install(tenantA, version, f)
	h.putRecord(tenantA, record, map[string]any{"priority": marker})
	h.start(version, record)
	err := h.env.GetWorkflowError()
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeOpaque {
		t.Fatalf("error %v, want nava.Opaque", err)
	}
	if text := h.recordedText(); strings.Contains(text, marker) {
		t.Errorf("the marker reached the history:\n%s", text)
	}
	if strings.Contains(h.logs.String(), marker) {
		t.Error("the marker reached the guard log")
	}
}

// TestCrossTenantIsNotFound: tenant A's run cannot read tenant B's
// definition version or record; the tenant is the header's, never input.
func TestCrossTenantIsNotFound(t *testing.T) {
	f := fixtureV1(t)
	t.Run("B's definition version", func(t *testing.T) {
		h := newHarness(t, tenantA)
		version, record := uuid.New(), uuid.New()
		h.install(tenantB, version, f)
		h.putRecord(tenantA, record, map[string]any{"priority": "high"})
		h.start(version, record)
		wantRejected(t, h.env.GetWorkflowError(), workerguard.RejectNotFound)
	})
	t.Run("B's record", func(t *testing.T) {
		h := newHarness(t, tenantA)
		version, record := uuid.New(), uuid.New()
		h.install(tenantA, version, f)
		h.putRecord(tenantB, record, map[string]any{"priority": "high"})
		h.start(version, record)
		wantRejected(t, h.env.GetWorkflowError(), workerguard.RejectNotFound)
		if c := h.calls(); len(c) != 0 {
			t.Errorf("writes %+v, want none", c)
		}
	})
	t.Run("B's step id", func(t *testing.T) {
		h := newHarness(t, tenantA)
		version, record := uuid.New(), uuid.New()
		h.install(tenantA, version, f)
		h.putRecord(tenantA, record, map[string]any{"priority": "high"})
		b, err := json.Marshal(map[string]uuid.UUID{"step": uuid.New()})
		if err != nil {
			t.Fatal(err)
		}
		in := workerguard.RunInput{Delivery: startInput(t, version, record), Refs: b}
		h.env.ExecuteWorkflow(interp.WorkflowType, in)
		wantRejected(t, h.env.GetWorkflowError(), workerguard.RejectNotFound)
	})
}

// TestBusinessRejectionFailsTheRun: a validation rejection from the writer
// is non-retryable with fixed text; ApplyStep runs once and the run fails.
func TestBusinessRejectionFailsTheRun(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want workerguard.Rejection
	}{
		"validation": {fmt.Errorf("%w: field title MARKER", interp.ErrValidation), workerguard.RejectValidation},
		"transition": {fmt.Errorf("%w: open -> MARKER", interp.ErrIllegalTransition), workerguard.RejectTransition},
		"forbidden":  {interp.ErrForbidden, workerguard.RejectForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			h, f, version, record := v1(t, "high")
			h.writer.Fail = tc.err
			h.signalAt(time.Hour, uuid.New(), f.id(t, def.SignalKey("approve")))
			h.start(version, record)
			err := h.env.GetWorkflowError()
			wantRejected(t, err, tc.want)
			if c := h.calls(); len(c) != 1 || c[0].Attempt != 1 {
				t.Errorf("writes %+v, want one attempt", c)
			}
			if strings.Contains(h.recordedText(), "MARKER") {
				t.Error("the writer's text reached the history")
			}
		})
	}
}

// TestKindDefaultFailsWithoutPanic: a Directive the switch does not know (or
// with a missing edge) fails the run with the fixed rejection; the guard is
// off here, since it would refuse the Directive first.
func TestKindDefaultFailsWithoutPanic(t *testing.T) {
	step := uuid.New()
	for name, d := range map[string]workerguard.Directive{
		"unknown kind":     {Step: step, Kind: "close_ticket"},
		"timer, no next":   {Step: step, Kind: workerguard.KindTimer, WaitSeconds: 1},
		"update, no next":  {Step: step, Kind: workerguard.KindUpdateRecord},
		"wait, no signal":  {Step: step, Kind: workerguard.KindWaitSignal, WaitSeconds: 1, Next: &step, OnTimeout: &step},
		"wait, no timeout": {Step: step, Kind: workerguard.KindWaitSignal, WaitSeconds: 1, Signal: &step, Next: &step},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, tenantA, withoutGuard())
			d := d
			h.env.RegisterActivityWithOptions(func(context.Context, interp.StepRef) (workerguard.Directive, error) { return d, nil },
				activity.RegisterOptions{Name: interp.NextStepActivity})
			h.env.RegisterActivityWithOptions(func(context.Context, interp.StepRef) error { return nil },
				activity.RegisterOptions{Name: interp.ApplyStepActivity})
			h.start(uuid.New(), uuid.New())
			wantRejected(t, h.env.GetWorkflowError(), workerguard.RejectDirective)
		})
	}
}

// TestStartInputRefused: a start payload that is not exactly the two UUIDs
// fails the run with the fixed rejection.
func TestStartInputRefused(t *testing.T) {
	for name, p := range map[string]string{
		"missing record":       `{"definition_version_id":"` + uuid.NewString() + `"}`,
		"extra definition_id":  `{"definition_version_id":"` + uuid.NewString() + `","record_id":"` + uuid.NewString() + `","definition_id":"` + uuid.NewString() + `"}`,
		"nil version":          `{"definition_version_id":"` + uuid.Nil.String() + `","record_id":"` + uuid.NewString() + `"}`,
		"empty object payload": `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, tenantA)
			in := startInput(t, uuid.New(), uuid.New())
			in.Payload = json.RawMessage(p)
			h.env.ExecuteWorkflow(interp.WorkflowType, in)
			wantRejected(t, h.env.GetWorkflowError(), workerguard.RejectInput)
		})
	}
}

// TestWaitBoundsMatchTheGuard: the definition format's wait bounds are the
// Directive's (TICK-376 review); def stays Temporal-free, so the check is
// here.
func TestWaitBoundsMatchTheGuard(t *testing.T) {
	if def.MinWaitSeconds != workerguard.MinWaitSeconds || def.MaxWaitSeconds != workerguard.MaxWaitSeconds {
		t.Errorf("def wait bounds %d..%d, Directive %d..%d", def.MinWaitSeconds, def.MaxWaitSeconds, workerguard.MinWaitSeconds, workerguard.MaxWaitSeconds)
	}
}
