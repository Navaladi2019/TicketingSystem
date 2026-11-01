package interp_test

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// errAdapterDown is a retryable adapter fault: not a port error, so the
// activities return it as is and the guard withholds it, retryable.
var errAdapterDown = errors.New("adapter: connection refused")

// countingReader fails every Read with errAdapterDown and counts the calls.
type countingReader struct{ n *atomic.Int32 }

func (c countingReader) Read(context.Context, uuid.UUID, uuid.UUID) (map[string]any, error) {
	c.n.Add(1)
	return nil, errAdapterDown
}

// TestActivityRetryPolicyIsBounded: both activities carry an explicit policy
// with a maximum interval and a maximum attempt count, never the SDK default
// that retries forever (TICK-384 review).
func TestActivityRetryPolicyIsBounded(t *testing.T) {
	p := interp.ActivityRetryPolicy()
	if p.MaximumAttempts <= 0 || p.MaximumInterval <= 0 || p.InitialInterval <= 0 {
		t.Fatalf("policy %+v: want MaximumAttempts, MaximumInterval and InitialInterval set", p)
	}
	if p.MaximumInterval < p.InitialInterval {
		t.Errorf("MaximumInterval %v below InitialInterval %v", p.MaximumInterval, p.InitialInterval)
	}
}

// wantRetryPolicy: every NextStep and ApplyStep the run scheduled carried
// exactly ActivityRetryPolicy, and the run scheduled want of them. It reads
// the options the workflow set, not the SDK test environment's effective
// policy, whose default without one is also 10 attempts.
func wantRetryPolicy(t *testing.T, h *harness, want ...string) {
	t.Helper()
	policy := interp.ActivityRetryPolicy()
	seen := map[string]bool{}
	for _, s := range h.scheduledOptions() {
		if s.activity != interp.NextStepActivity && s.activity != interp.ApplyStepActivity {
			continue
		}
		seen[s.activity] = true
		if !reflect.DeepEqual(s.options.RetryPolicy, policy) {
			t.Errorf("%s scheduled with RetryPolicy %+v, want %+v", s.activity, s.options.RetryPolicy, policy)
		}
	}
	for _, a := range want {
		if !seen[a] {
			t.Errorf("%s was never scheduled", a)
		}
	}
}

// wantOpaque: the run failed with the guard's withheld (retryable) failure,
// after the retries ran out, not a Rejection.
func wantOpaque(t *testing.T, err error) {
	t.Helper()
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeOpaque {
		t.Fatalf("error %v, want the withheld %s failure", err, workerguard.ErrTypeOpaque)
	}
}

// TestPersistentNextStepFaultFailsTheRun: a NextStep adapter fault that never
// clears fails the run after ActivityMaxAttempts attempts.
func TestPersistentNextStepFaultFailsTheRun(t *testing.T) {
	var n atomic.Int32
	h := newHarness(t, tenantA, withRecords(func(string) interp.RecordReader { return countingReader{&n} }))
	version, record := uuid.New(), uuid.New()
	h.install(tenantA, version, fixtureV1(t))
	h.start(version, record)
	if !h.env.IsWorkflowCompleted() {
		t.Fatal("the run did not end")
	}
	wantOpaque(t, h.env.GetWorkflowError())
	wantRetryPolicy(t, h, interp.NextStepActivity)
	if got := n.Load(); got != interp.ActivityMaxAttempts {
		t.Errorf("NextStep read the record %d times, want %d", got, interp.ActivityMaxAttempts)
	}
}

// TestPersistentApplyStepFaultFailsTheRun: an ApplyStep writer fault that
// never clears fails the run after ActivityMaxAttempts attempts, each
// recorded with its attempt number, and applies nothing.
func TestPersistentApplyStepFaultFailsTheRun(t *testing.T) {
	h, f, version, record := v1(t, "high")
	h.writer.Fail = errAdapterDown
	h.signalAt(time.Hour, uuid.New(), f.id(t, def.SignalKey("approve")))
	h.start(version, record)
	if !h.env.IsWorkflowCompleted() {
		t.Fatal("the run did not end")
	}
	wantOpaque(t, h.env.GetWorkflowError())
	wantRetryPolicy(t, h, interp.NextStepActivity, interp.ApplyStepActivity)
	calls := h.calls()
	if len(calls) != interp.ActivityMaxAttempts {
		t.Fatalf("writes %d, want %d", len(calls), interp.ActivityMaxAttempts)
	}
	for i, c := range calls {
		if c.Attempt != int32(i+1) || c.Applied {
			t.Errorf("write %d: %+v, want attempt %d, not applied", i, c, i+1)
		}
	}
}
