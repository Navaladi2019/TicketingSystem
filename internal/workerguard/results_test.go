package workerguard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// Results, side effects and failure text (TICK-245, Security triage rows 3 to
// 7): none may carry record data into the history.

// titled is record data in a result.
func titled() map[string]string { return map[string]string{"title": recordData} }

func refs() map[string]string { return map[string]string{"record_id": uuid.NewString()} }

// wantOpaque fails unless err is (or wraps) the withheld failure: fixed text,
// type ErrTypeOpaque, the given non-retryable flag, and no record data.
func wantOpaque(t *testing.T, err error, nonRetryable bool) {
	t.Helper()
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != ErrTypeOpaque {
		t.Fatalf("error %v, want the withheld %s failure", err, ErrTypeOpaque)
	}
	if app.Message() != opaqueMessage || app.HasDetails() || app.Unwrap() != nil {
		t.Errorf("withheld failure %q (details %v, cause %v), want only the fixed text", app.Message(), app.HasDetails(), app.Unwrap())
	}
	if app.NonRetryable() != nonRetryable {
		t.Errorf("non-retryable %v, want %v", app.NonRetryable(), nonRetryable)
	}
	if strings.Contains(err.Error(), "printer on fire") {
		t.Errorf("the error holds the record data: %v", err)
	}
}

// wantWithheldLogged fails unless the worker log has a withheld line for
// guard with the failure's shape, and no line holds the record data.
func (h *harness) wantWithheldLogged(guard, shape string) {
	h.t.Helper()
	var withheld bool
	for line := range strings.Lines(h.logs.String()) {
		withheld = withheld || (strings.Contains(line, "withheld a failure") &&
			strings.Contains(line, `"guard":"`+guard+`"`) && strings.Contains(line, shape))
	}
	if !withheld {
		h.t.Errorf("no %s withheld line with shape %q; log:\n%s", guard, shape, h.logs.String())
	}
	h.wantNoRecordData(nil)
}

func runActivity(ctx workflow.Context, local bool, act any) error {
	if local {
		ctx = workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
			StartToCloseTimeout: time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1},
		})
		return workflow.ExecuteLocalActivity(ctx, act).Get(ctx, nil)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	return workflow.ExecuteActivity(ctx, act).Get(ctx, nil)
}

func TestActivityResultGuard(t *testing.T) {
	id := idOf(tenantA)
	for _, local := range []bool{false, true} {
		name := map[bool]string{false: "activity", true: "local activity"}[local]
		t.Run(name+" result with record data fails", func(t *testing.T) {
			h := newHarness(t, id)
			act := func(context.Context) (map[string]string, error) { return titled(), nil }
			h.env.RegisterActivityWithOptions(act, activity.RegisterOptions{Name: "load"})
			h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
				return runActivity(ctx, local, map[bool]any{false: "load", true: act}[local])
			})
			err := h.env.GetWorkflowError()
			wantGuardError(t, err, guardActivityResult, outcomeRecordData)
			h.wantDenyAlert(guardActivityResult, outcomeRecordData, id)
			h.wantNoRecordData(err)
		})
		for rname, act := range map[string]func(context.Context) (any, error){
			"references": func(context.Context) (any, error) { return refs(), nil },
			"nil":        func(context.Context) (any, error) { return nil, nil },
			"a nil map":  func(context.Context) (any, error) { return map[string]string(nil), nil },
		} {
			t.Run(name+" result "+rname+" passes", func(t *testing.T) {
				h := newHarness(t, id)
				h.env.RegisterActivityWithOptions(act, activity.RegisterOptions{Name: "load"})
				h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
					return runActivity(ctx, local, map[bool]any{false: "load", true: act}[local])
				})
				if err := h.env.GetWorkflowError(); err != nil {
					t.Fatal(err)
				}
				h.wantNoAlerts()
			})
		}
	}
}

func TestActivityFailureWithheld(t *testing.T) {
	id := idOf(tenantA)
	for _, tc := range []struct {
		name         string
		fail         func() error
		nonRetryable bool
		shape        string
	}{
		{"plain error", func() error { return fmt.Errorf("ticket %q not found", recordData) }, false, "*errors.errorString"},
		{"wrapped guard error", func() error {
			return fmt.Errorf("%s: %w", recordData, guardError(guardActivity, outcomeMismatch))
		}, false, "*fmt.wrapError / *internal.ApplicationError[nava.TenantGuard]"},
		{"application error of another type", func() error {
			return temporal.NewNonRetryableApplicationError(recordData, "billing.Declined", nil)
		}, true, "*internal.ApplicationError[billing.Declined]"},
		{"allowed type with a cause", func() error {
			return temporal.NewNonRetryableApplicationError("workflow_ref: closed", ErrTypeWorkflowRef, errors.New(recordData))
		}, true, "*internal.ApplicationError[nava.WorkflowRefNotFound] / *errors.errorString"},
		{"panic", func() error { panic("ticket " + recordData) }, false, "workerguard.activityPanic"},
	} {
		for _, local := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s local=%v", tc.name, local), func(t *testing.T) {
				h := newHarness(t, id)
				act := func(context.Context) error { return tc.fail() }
				h.env.RegisterActivityWithOptions(act, activity.RegisterOptions{Name: "fail"})
				h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
					return runActivity(ctx, local, map[bool]any{false: "fail", true: act}[local])
				})
				err := h.env.GetWorkflowError()
				wantOpaque(t, err, tc.nonRetryable)
				h.wantDenyAlert(guardActivityFailure, outcomeRecordData, id)
				h.wantWithheldLogged(guardActivityFailure, tc.shape)
				if strings.Contains(err.Error(), "goroutine") {
					t.Errorf("the failure holds a stack: %v", err)
				}
			})
		}
	}
	t.Run("allowed type passes unchanged", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.RegisterActivityWithOptions(func(context.Context) error {
			return temporal.NewNonRetryableApplicationError("workflow_ref: no row for this workflow", ErrTypeWorkflowRef, nil)
		}, activity.RegisterOptions{Name: "close"})
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error { return runActivity(ctx, false, "close") })
		var app *temporal.ApplicationError
		if err := h.env.GetWorkflowError(); !errors.As(err, &app) || app.Type() != ErrTypeWorkflowRef ||
			app.Message() != "workflow_ref: no row for this workflow" {
			t.Fatalf("error %v, want the %s error unchanged", err, ErrTypeWorkflowRef)
		}
		h.wantNoAlerts()
	})
	t.Run("allowed type keeps only reference details", func(t *testing.T) {
		ref := refs()
		changed, out := recordable(temporal.NewApplicationErrorWithOptions("tenant guard denied x: y", ErrTypeTenantGuard,
			temporal.ApplicationErrorOptions{NonRetryable: true, Details: []any{ref, titled()}}))
		var app *temporal.ApplicationError
		if !changed || !errors.As(out, &app) || app.Type() != ErrTypeTenantGuard || !app.NonRetryable() {
			t.Fatalf("recordable: %v, %v; want the guard error with a detail dropped", out, changed)
		}
		var got map[string]string
		var extra any
		if err := app.Details(&got, &extra); err == nil || got != nil {
			// Two details read means the record-data one survived.
			t.Errorf("details after the check: %v, %v", got, extra)
		}
		if err := app.Details(&got); err != nil || got["record_id"] != ref["record_id"] {
			t.Errorf("reference detail %v, %v; want it kept", got, err)
		}
	})
}

func TestRecordablePassesSDKErrors(t *testing.T) {
	for name, err := range map[string]error{
		"canceled":        temporal.NewCanceledError(),
		"timeout":         temporal.NewTimeoutError(0, nil),
		"result pending":  activity.ErrResultPending,
		"context":         context.Canceled,
		"continue-as-new": &workflow.ContinueAsNewError{},
		"guard error":     guardError(guardChild, outcomeCrossTenant),
	} {
		if changed, out := recordable(err); changed || out != err { //nolint:errorlint // identity is the point
			t.Errorf("%s: recordable changed it to %v", name, out)
		}
	}
	if changed, out := recordable(nil); changed || out != nil {
		t.Errorf("nil: %v, %v", out, changed)
	}
}

// SDK cancel and timeout errors have public constructors that take details
// and a cause, so workflow or activity code can put record data in them; the
// guard keeps their type and drops the data (TICK-245 security review,
// finding 1).
func TestRecordableChecksCancelAndTimeoutContent(t *testing.T) {
	t.Run("canceled with record-data details", func(t *testing.T) {
		changed, out := recordable(temporal.NewCanceledError(titled()))
		var c *temporal.CanceledError
		if !changed || !errors.As(out, &c) || c.HasDetails() {
			t.Fatalf("recordable: %v, %v; want a canceled error with no details", changed, out)
		}
	})
	t.Run("canceled keeps reference details", func(t *testing.T) {
		ref := refs()
		changed, out := recordable(temporal.NewCanceledError(ref, titled()))
		var c *temporal.CanceledError
		var got map[string]string
		if !changed || !errors.As(out, &c) || c.Details(&got) != nil || got["record_id"] != ref["record_id"] {
			t.Fatalf("recordable: %v, %v (%v); want the reference detail kept", changed, out, got)
		}
		var extra any
		if c.Details(&got, &extra) == nil {
			t.Errorf("a second detail survived: %v", extra)
		}
	})
	// Finding 1b: NewCanceledErrorWithOptions sets a message and a cause too.
	for name, opts := range map[string]temporal.CanceledErrorOptions{
		"canceled with a record-data message": {Message: recordData},
		"canceled with a record-data cause":   {Cause: fmt.Errorf("ticket %s", recordData)},
		"canceled with both and details":      {Message: recordData, Cause: errors.New(recordData), Details: []any{titled()}},
	} {
		t.Run(name, func(t *testing.T) {
			changed, out := recordable(temporal.NewCanceledErrorWithOptions(opts))
			var c *temporal.CanceledError
			if !changed || !errors.As(out, &c) || c.HasDetails() || c.Error() != canceledMessage {
				t.Fatalf("recordable: %v, %v; want the SDK's plain canceled error", changed, out)
			}
			for e := out; e != nil; e = errors.Unwrap(e) {
				if strings.Contains(e.Error(), "printer on fire") {
					t.Errorf("the chain holds the record data: %v", e)
				}
			}
		})
	}
	t.Run("the SDK's plain canceled error passes", func(t *testing.T) {
		in := temporal.NewCanceledErrorWithOptions(temporal.CanceledErrorOptions{Details: []any{refs()}})
		if changed, out := recordable(in); changed || out != in { //nolint:errorlint // identity is the point
			t.Errorf("recordable changed it to %v", out)
		}
	})
	t.Run("timeout with a record-data cause", func(t *testing.T) {
		changed, out := recordable(temporal.NewTimeoutError(enumspb.TIMEOUT_TYPE_START_TO_CLOSE, fmt.Errorf("ticket %s", recordData)))
		var to *temporal.TimeoutError
		if !changed || !errors.As(out, &to) || to.TimeoutType() != enumspb.TIMEOUT_TYPE_START_TO_CLOSE {
			t.Fatalf("recordable: %v, %v; want a start-to-close timeout", changed, out)
		}
		if strings.Contains(out.Error(), "printer on fire") {
			t.Errorf("the timeout holds the record data: %v", out)
		}
		var app *temporal.ApplicationError
		if !errors.As(out, &app) || app.Type() != ErrTypeOpaque {
			t.Errorf("cause %v, want it withheld", errors.Unwrap(out))
		}
	})
	t.Run("heartbeat timeout with record-data details", func(t *testing.T) {
		changed, out := recordable(temporal.NewHeartbeatTimeoutError(recordData))
		var to *temporal.TimeoutError
		if !changed || !errors.As(out, &to) || to.TimeoutType() != enumspb.TIMEOUT_TYPE_HEARTBEAT || to.HasLastHeartbeatDetails() {
			t.Fatalf("recordable: %v, %v; want a heartbeat timeout with no details", changed, out)
		}
	})
	t.Run("a workflow failing with canceled record data alerts", func(t *testing.T) {
		id := idOf(tenantA)
		h := newHarness(t, id)
		h.env.ExecuteWorkflow(func(workflow.Context) error { return temporal.NewCanceledError(titled()) })
		err := h.env.GetWorkflowError()
		var c *temporal.CanceledError
		if !errors.As(err, &c) || c.HasDetails() {
			t.Fatalf("workflow error %v, want a canceled error with no details", err)
		}
		h.wantDenyAlert(guardWorkflowFailure, outcomeRecordData, id)
		h.wantWithheldLogged(guardWorkflowFailure, "*internal.CanceledError")
	})
}

func TestWorkflowResultGuard(t *testing.T) {
	id := idOf(tenantA)
	t.Run("record data fails the workflow", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.ExecuteWorkflow(func(workflow.Context) (map[string]string, error) { return titled(), nil })
		err := h.env.GetWorkflowError()
		wantGuardError(t, err, guardWorkflowResult, outcomeRecordData)
		h.wantDenyAlert(guardWorkflowResult, outcomeRecordData, id)
		h.wantNoRecordData(err)
	})
	for name, result := range map[string]any{"references": refs(), "a Delivery": validDelivery(t), "nil": nil} {
		t.Run(name+" completes", func(t *testing.T) {
			h := newHarness(t, id)
			h.env.ExecuteWorkflow(func(workflow.Context) (any, error) { return result, nil })
			if err := h.env.GetWorkflowError(); err != nil {
				t.Fatal(err)
			}
			h.wantNoAlerts()
		})
	}
}

func TestWorkflowFailureWithheld(t *testing.T) {
	id := idOf(tenantA)
	t.Run("free text is withheld", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.ExecuteWorkflow(func(workflow.Context) error { return fmt.Errorf("ticket %q stuck", recordData) })
		wantOpaque(t, h.env.GetWorkflowError(), false)
		h.wantDenyAlert(guardWorkflowFailure, outcomeRecordData, id)
		h.wantWithheldLogged(guardWorkflowFailure, "*errors.errorString")
	})
	t.Run("the guard's own error passes", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.ExecuteWorkflow(func(workflow.Context) error { return guardError(guardChild, outcomeCrossTenant) })
		wantGuardError(t, h.env.GetWorkflowError(), guardChild, outcomeCrossTenant)
		h.wantNoAlerts()
	})
}

// withHandler serves update "u" with handler for an hour.
func withHandler(handler any) func(workflow.Context) error {
	return func(ctx workflow.Context) error {
		if err := workflow.SetUpdateHandler(ctx, "u", handler); err != nil {
			return err
		}
		return workflow.Sleep(ctx, time.Hour)
	}
}

func TestUpdateResultAndFailureGuard(t *testing.T) {
	id := idOf(tenantA)
	for _, tc := range []struct {
		name    string
		handler any
		check   func(t *testing.T, h *harness, err error)
	}{
		{"record-data result", func(workflow.Context) (map[string]string, error) { return titled(), nil },
			func(t *testing.T, h *harness, err error) {
				wantGuardError(t, err, guardUpdateResult, outcomeRecordData)
				h.wantDenyAlert(guardUpdateResult, outcomeRecordData, id)
			}},
		{"record-data failure", func(workflow.Context) error { return errors.New(recordData) },
			func(t *testing.T, h *harness, err error) {
				wantOpaque(t, err, false)
				h.wantDenyAlert(guardUpdateFailure, outcomeRecordData, id)
				h.wantWithheldLogged(guardUpdateFailure, "*errors.errorString")
			}},
		{"references result", func(workflow.Context) (map[string]string, error) { return refs(), nil },
			func(t *testing.T, h *harness, err error) {
				if err != nil {
					t.Fatal(err)
				}
				h.wantNoAlerts()
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, id)
			h.tr.setUpdate(stringPayload(t, tenantA.String()))
			var completed bool
			var updateErr error
			h.env.RegisterDelayedCallback(func() {
				h.env.UpdateWorkflow("u", "1", &testsuite.TestUpdateCallback{
					OnReject:   func(err error) { t.Errorf("rejected: %v", err) },
					OnAccept:   func() {},
					OnComplete: func(_ any, err error) { completed, updateErr = true, err },
				})
			}, time.Second)
			h.env.ExecuteWorkflow(withHandler(tc.handler))
			if err := h.env.GetWorkflowError(); err != nil {
				t.Fatalf("workflow: %v", err)
			}
			if !completed {
				t.Fatal("the update did not complete")
			}
			tc.check(t, h, updateErr)
			if updateErr != nil && strings.Contains(updateErr.Error(), "printer on fire") {
				t.Errorf("the update failure holds the record data: %v", updateErr)
			}
		})
	}
}

func TestSideEffectValueOK(t *testing.T) {
	at := time.Date(2026, 10, 27, 9, 0, 0, 5, time.UTC)
	var nilPtr *uuid.UUID
	for _, v := range []any{nil, uuid.New(), uuid.NewString(), at, 42, 0.5, true, refs(), validDelivery(t), nilPtr} {
		if !sideEffectValueOK(v) {
			t.Errorf("%T %v denied, want allowed", v, v)
		}
	}
	for _, v := range []any{recordData, titled(), strings.ToUpper(uuid.NewString()), "{" + uuid.NewString() + "}",
		[]string{uuid.NewString()}, []byte(`"` + recordData + `"`), map[string]any{"n": 1}} {
		if sideEffectValueOK(v) {
			t.Errorf("%T %v allowed, want denied", v, v)
		}
	}
}

func TestSideEffectGuard(t *testing.T) {
	id := idOf(tenantA)
	t.Run("record data is not recorded and Get fails", func(t *testing.T) {
		h := newHarness(t, id)
		var getErr error
		var has bool
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
			v := workflow.SideEffect(ctx, func(workflow.Context) any { return titled() })
			has = v.HasValue()
			var out map[string]string
			getErr = v.Get(&out)
			return nil
		})
		if err := h.env.GetWorkflowError(); err != nil {
			t.Fatal(err)
		}
		if has {
			t.Error("HasValue on a denied side effect")
		}
		wantGuardError(t, getErr, guardSideEffect, outcomeRecordData)
		h.wantDenyAlert(guardSideEffect, outcomeRecordData, id)
		h.wantNoRecordData(getErr)
	})
	t.Run("a UUID, a time and references pass", func(t *testing.T) {
		h := newHarness(t, id)
		want := uuid.New()
		var got uuid.UUID
		var gotRefs map[string]string
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
			if err := workflow.SideEffect(ctx, func(workflow.Context) any { return want }).Get(&got); err != nil {
				return err
			}
			var at time.Time
			if err := workflow.SideEffect(ctx, func(workflow.Context) any { return workflow.Now(ctx) }).Get(&at); err != nil {
				return err
			}
			return workflow.SideEffectWithOptions(ctx, workflow.SideEffectOptions{},
				func(workflow.Context) any { return map[string]string{"record_id": want.String()} }).Get(&gotRefs)
		})
		if err := h.env.GetWorkflowError(); err != nil {
			t.Fatal(err)
		}
		if got != want || gotRefs["record_id"] != want.String() {
			t.Errorf("side effects %v, %v; want %v", got, gotRefs, want)
		}
		h.wantNoAlerts()
	})
	t.Run("MutableSideEffect is not allowed and never runs f", func(t *testing.T) {
		h := newHarness(t, id)
		ran := false
		var getErr error
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
			v := workflow.MutableSideEffect(ctx, "k", func(workflow.Context) any { ran = true; return 1 },
				func(a, b any) bool { return a == b })
			var n int
			getErr = v.Get(&n)
			return nil
		})
		if err := h.env.GetWorkflowError(); err != nil {
			t.Fatal(err)
		}
		if ran {
			t.Error("f ran")
		}
		wantGuardError(t, getErr, guardSideEffect, outcomeNotAllowed)
		h.wantDenyAlert(guardSideEffect, outcomeNotAllowed, id)
	})
}
