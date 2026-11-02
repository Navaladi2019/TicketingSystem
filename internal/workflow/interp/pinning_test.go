package interp_test

// P3-24 (TICK-351, PLAN-003 D-3-12): a run is pinned to the definition
// version its start payload names. Publishing v2 while a v1 run waits on its
// timer never changes what that run does, across continue-as-new too; a new
// run started on v2 uses v2.
//
// testdata/pinned-v1 and pinned-v2 differ only in the value of the
// update_record step after the 5 s timer ("stage": "v1" / "v2"), and share
// one step-id map (def.AssignStepIDs keeps an id by key), so the write the
// run makes is the one thing that tells a pinned run from an unpinned one.

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

// sharedRoot reuses another harness's fixture directory under a new workflow
// id: another run of the same tenant against the same store.
func sharedRoot(h *harness) harnessOption {
	return func(c *harnessConfig) { c.root = h.root }
}

// pinned installs pinned-v1 for tenant A with a record, starts nothing, and
// returns the harness, v1's fixture, v2's fixture, both version ids and the
// record.
func pinned(t *testing.T) (h *harness, f1, f2 fixture, v1ID, v2ID, record uuid.UUID) {
	t.Helper()
	h = newHarness(t, tenantA)
	f1, f2 = loadFixture(t, "testdata/pinned-v1"), loadFixture(t, "testdata/pinned-v2")
	v1ID, v2ID, record = uuid.New(), uuid.New(), uuid.New()
	h.install(tenantA, v1ID, f1)
	h.putRecord(tenantA, record, map[string]any{})
	return h, f1, f2, v1ID, v2ID, record
}

// publishDuringTimer installs v2 one second into the run, while v1's 5 s
// cool_off timer is pending, and reports whether it ran.
func (h *harness) publishDuringTimer(v2ID uuid.UUID, f2 fixture, then func()) *bool {
	done := new(bool)
	h.env.RegisterDelayedCallback(func() {
		h.install(tenantA, v2ID, f2)
		*done = true
		if then != nil {
			then()
		}
	}, time.Second)
	return done
}

// wantOneStamp fails unless h's run wrote stamp once, with stage want, under
// its own key. Runs sharing a root share calls.jsonl, so it reads only the
// calls whose key holds h's workflow id.
func (h *harness) wantOneStamp(f fixture, want string) {
	h.t.Helper()
	var calls []interptest.Call
	for _, c := range h.calls() {
		if strings.HasPrefix(c.Key, "wf:"+h.workflowID+":") {
			calls = append(calls, c)
		}
	}
	if len(calls) != 1 {
		h.t.Fatalf("writes %+v, want 1", calls)
	}
	c := calls[0]
	if got := c.Set["stage"]; got != want {
		h.t.Errorf("stamp wrote stage %v, want %q", got, want)
	}
	if key := interp.IdempotencyKey(h.workflowID, f.id(h.t, "stamp")); c.Key != key || !c.Applied {
		h.t.Errorf("write %+v, want key %s applied", c, key)
	}
}

// wantPinned fails unless every NextStep of h's run named version.
func (h *harness) wantPinned(version uuid.UUID) {
	h.t.Helper()
	refs := h.nextRefs()
	if len(refs) == 0 {
		h.t.Fatal("no NextStep ran")
	}
	for i, r := range refs {
		if r.DefinitionVersionID != version {
			h.t.Errorf("NextStep %d named version %s, want the pinned %s", i, r.DefinitionVersionID, version)
		}
	}
}

// TestInFlightRunKeepsItsVersion: v2 is published while a v1 run waits on
// its timer. The v1 run completes with v1's update value; a new run uses v2.
func TestInFlightRunKeepsItsVersion(t *testing.T) {
	h, f1, f2, v1ID, v2ID, record := pinned(t)
	published := h.publishDuringTimer(v2ID, f2, nil)
	h.start(v1ID, record)
	h.wantCompleted()
	if !*published {
		t.Fatal("v2 was never published: the run ended before its timer fired")
	}
	if h.elapsed() != coolOff {
		t.Errorf("elapsed %v, want %v: v2 was published during the timer, before stamp", h.elapsed(), coolOff)
	}
	h.wantOneStamp(f1, "v1")
	h.wantPinned(v1ID)

	next := newHarness(t, tenantA, sharedRoot(h))
	next.start(v2ID, record)
	next.wantCompleted()
	next.wantOneStamp(f2, "v2")
	next.wantPinned(v2ID)
}

// TestPinnedAcrossContinueAsNew: v2 is published and the server suggests
// continue-as-new while the v1 run waits on its timer. The run continues as
// new at stamp, still on v1 (the start payload is carried, never re-read),
// and the next run writes v1's value.
func TestPinnedAcrossContinueAsNew(t *testing.T) {
	h, f1, f2, v1ID, v2ID, record := pinned(t)
	published := h.publishDuringTimer(v2ID, f2, func() { h.env.SetContinueAsNewSuggested(true) })
	h.start(v1ID, record)
	in := h.continued()
	if !*published {
		t.Fatal("v2 was never published")
	}
	if c := h.calls(); len(c) != 0 {
		t.Errorf("writes %+v before continue-as-new, want none: it happens between steps", c)
	}
	var refs struct {
		Step uuid.UUID `json:"step"`
	}
	if err := json.Unmarshal(in.Refs, &refs); err != nil || refs.Step != f1.id(t, "stamp") {
		t.Fatalf("continued at %s (%v), want stamp", refs.Step, err)
	}
	h.wantPinned(v1ID)

	next := newHarness(t, tenantA, sameStore(h))
	next.env.SetStartTime(t0.Add(coolOff))
	next.env.ExecuteWorkflow(interp.WorkflowType, in)
	next.wantCompleted()
	next.wantOneStamp(f1, "v1")
	next.wantPinned(v1ID)
}

// TestApplyOnceFixture: the crash case's fixture (testdata/applyonce, used by
// the interpreter stack case) parses, writes apply once and ends, so a stack
// failure is about the crash, not the fixture.
func TestApplyOnceFixture(t *testing.T) {
	h := newHarness(t, tenantA)
	f := loadFixture(t, "testdata/applyonce")
	version, record := uuid.New(), uuid.New()
	h.install(tenantA, version, f)
	h.putRecord(tenantA, record, map[string]any{})
	h.start(version, record)
	h.wantCompleted()
	calls := h.calls()
	if len(calls) != 1 || calls[0].Step != f.id(t, "apply") || calls[0].Set["applied"] != true || !calls[0].Applied {
		t.Errorf("writes %+v, want apply once", calls)
	}
	if got := h.timerDurations(); len(got) != 0 {
		t.Errorf("timers %v, want none: the crash case kills on the first step", got)
	}
}
