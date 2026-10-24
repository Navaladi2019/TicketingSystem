package workerguard

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestTenantOf(t *testing.T) {
	for _, tc := range []struct {
		id   string
		want uuid.UUID
	}{
		{"t:" + tenantA.String() + ":test:x", tenantA},
		{"t:" + tenantA.String() + ":k", tenantA},
		{"", uuid.Nil},
		{"t:", uuid.Nil},
		{"t:" + tenantA.String(), uuid.Nil},       // no rest
		{"t:" + tenantA.String() + ":", uuid.Nil}, // empty rest
		{"T:" + tenantA.String() + ":x", uuid.Nil},
		{"t:" + strings.ToUpper(tenantA.String()) + ":x", uuid.Nil}, // not canonical
		{"t:{" + tenantA.String() + "}:x", uuid.Nil},
		{"t:" + uuid.Nil.String() + ":x", uuid.Nil},
		{"t:A:x", uuid.Nil},
		{"0192f3a0-run_1", uuid.Nil}, // the SDK default child id shape
	} {
		got, err := TenantOf(tc.id)
		if got != tc.want || (err == nil) != (tc.want != uuid.Nil) {
			t.Errorf("TenantOf(%q) = %v, %v; want %v", tc.id, got, err, tc.want)
		}
	}
}

func TestPayloadString(t *testing.T) {
	good := stringPayload(t, "x")
	if s, ok := payloadString(good); !ok || s != "x" {
		t.Errorf("json/plain string: %q %v", s, ok)
	}
	list, _ := converter.GetDefaultDataConverter().ToPayload([]string{"x"})
	raw, _ := converter.GetDefaultDataConverter().ToPayload([]byte("x"))
	codec := &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("binary/encrypted")}, Data: good.GetData()}
	for name, p := range map[string]*commonpb.Payload{"nil": nil, "list": list, "binary/plain": raw, "codec": codec} {
		if _, ok := payloadString(p); ok {
			t.Errorf("%s decoded as a string", name)
		}
	}
}

// check turns a panic into outcomeInternal, a deny.
func TestCheckNeverPanics(t *testing.T) {
	if got := check(func() string { panic("boom") }); got != outcomeInternal {
		t.Errorf("check(panic) = %q, want %q", got, outcomeInternal)
	}
}

// --- workflow start -------------------------------------------------------------

func echo(ctx workflow.Context, in string) (string, error) { return "echo:" + in, nil }

func TestStartGuard(t *testing.T) {
	id := idOf(tenantA)
	t.Run("allow", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.ExecuteWorkflow(echo, "x")
		var out string
		if err := h.env.GetWorkflowResult(&out); err != nil || out != "echo:x" {
			t.Fatalf("result %q, %v", out, err)
		}
		h.wantNoAlerts()
		if got := h.decisions()[guardWorkflowStart+"/"+outcomeAllow]; got != 1 {
			t.Errorf("allow count %d, want 1", got)
		}
	})
	for _, tc := range []struct {
		name, outcome string
		id            string
		header        *commonpb.Header
	}{
		{"missing", outcomeMissing, id, &commonpb.Header{}},
		{"mismatch", outcomeMismatch, id, headerFor(t, tenantB)},
		{"not json/plain", outcomeMissing, id, &commonpb.Header{Fields: map[string]*commonpb.Payload{
			TenantHeader: {Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte(tenantA.String())},
		}}},
		{"id without tenant prefix", outcomeNoTenant, "plain-id", headerFor(t, tenantA)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, tc.id)
			h.env.SetHeader(tc.header)
			ran := false
			h.env.ExecuteWorkflow(func(ctx workflow.Context) error { ran = true; return nil })
			if ran {
				t.Error("the workflow function ran")
			}
			wantGuardError(t, h.env.GetWorkflowError(), guardWorkflowStart, tc.outcome)
			h.wantDenyAlert(guardWorkflowStart, tc.outcome, tc.id)
		})
	}
}

// --- signals --------------------------------------------------------------------

// collect applies "add" signals until it gets "done", and returns them.
func collect(ctx workflow.Context) ([]string, error) {
	var got []string
	add := workflow.GetSignalChannel(ctx, "add")
	done := workflow.GetSignalChannel(ctx, "done")
	for finished := false; !finished; {
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(add, func(c workflow.ReceiveChannel, _ bool) {
			var s string
			c.Receive(ctx, &s)
			got = append(got, s)
		})
		sel.AddReceive(done, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); finished = true })
		sel.Select(ctx)
	}
	return got, nil
}

func TestSignalGuard(t *testing.T) {
	id := idOf(tenantA)
	h := newHarness(t, id)
	send := func(p *commonpb.Payload, name, arg string) {
		h.tr.setSignal(p)
		h.env.SignalWorkflow(name, arg)
	}
	h.env.RegisterDelayedCallback(func() { send(stringPayload(t, tenantA.String()), "add", "own") }, time.Second)
	h.env.RegisterDelayedCallback(func() { send(stringPayload(t, tenantB.String()), "add", "other-tenant") }, 2*time.Second)
	h.env.RegisterDelayedCallback(func() { send(nil, "add", "no-header") }, 3*time.Second)
	// A wrong-tenant "done" must not end the workflow either.
	h.env.RegisterDelayedCallback(func() { send(stringPayload(t, tenantB.String()), "done", "") }, 4*time.Second)
	h.env.RegisterDelayedCallback(func() { send(stringPayload(t, tenantA.String()), "add", "own-2") }, 5*time.Second)
	h.env.RegisterDelayedCallback(func() { send(stringPayload(t, tenantA.String()), "done", "") }, 6*time.Second)
	h.env.ExecuteWorkflow(collect)

	var got []string
	if err := h.env.GetWorkflowResult(&got); err != nil {
		t.Fatalf("the workflow failed: %v", err)
	}
	if strings.Join(got, ",") != "own,own-2" {
		t.Errorf("applied %v, want [own own-2]: wrong-tenant signals must be dropped", got)
	}
	d := h.decisions()
	if d[guardSignal+"/"+outcomeMismatch] != 2 || d[guardSignal+"/"+outcomeMissing] != 1 || d[guardSignal+"/"+outcomeAllow] != 3 {
		t.Errorf("signal decisions %v, want 2 mismatch, 1 missing, 3 allow", d)
	}
	var n int
	for _, a := range h.alerts() {
		if a["guard"] == guardSignal {
			n++
			if a["workflow_id"] != id || a["signal"] == nil {
				t.Errorf("alert %v lacks the workflow id or signal name", a)
			}
		}
	}
	if n != 3 {
		t.Errorf("%d signal alerts, want 3", n)
	}
}

// --- updates --------------------------------------------------------------------

func withUpdate(ctx workflow.Context) (int, error) {
	n := 0
	if err := workflow.SetUpdateHandler(ctx, "inc", func(ctx workflow.Context) (int, error) { n++; return n, nil }); err != nil {
		return 0, err
	}
	if err := workflow.Await(ctx, func() bool { return n >= 1 }); err != nil {
		return 0, err
	}
	// Leave time for the rejected updates below to arrive.
	if err := workflow.Sleep(ctx, time.Hour); err != nil {
		return 0, err
	}
	return n, nil
}

type updateResult struct {
	rejected error
	accepted bool
}

func TestUpdateGuard(t *testing.T) {
	id := idOf(tenantA)
	h := newHarness(t, id)
	results := map[string]*updateResult{}
	update := func(name string, p *commonpb.Payload) {
		r := &updateResult{}
		results[name] = r
		h.tr.setUpdate(p)
		h.env.UpdateWorkflow("inc", name, &testsuite.TestUpdateCallback{
			OnReject:   func(err error) { r.rejected = err },
			OnAccept:   func() { r.accepted = true },
			OnComplete: func(any, error) {},
		})
	}
	h.env.RegisterDelayedCallback(func() { update("other-tenant", stringPayload(t, tenantB.String())) }, time.Second)
	h.env.RegisterDelayedCallback(func() { update("no-header", nil) }, 2*time.Second)
	h.env.RegisterDelayedCallback(func() { update("own", stringPayload(t, tenantA.String())) }, 3*time.Second)
	h.env.ExecuteWorkflow(withUpdate)

	var n int
	if err := h.env.GetWorkflowResult(&n); err != nil || n != 1 {
		t.Fatalf("result %d, %v; want 1 applied update", n, err)
	}
	wantGuardError(t, results["other-tenant"].rejected, guardUpdate, outcomeMismatch)
	wantGuardError(t, results["no-header"].rejected, guardUpdate, outcomeMissing)
	if r := results["own"]; r.rejected != nil || !r.accepted {
		t.Errorf("own-tenant update: rejected %v, accepted %v", r.rejected, r.accepted)
	}
	h.wantDenyAlert(guardUpdate, outcomeMismatch, id)
	h.wantDenyAlert(guardUpdate, outcomeMissing, id)
}

// --- activities -----------------------------------------------------------------

// seenTenant is an activity that returns the tenant on its context.
func seenTenant(ctx context.Context) (string, error) {
	tenant, ok := TenantFromContext(ctx)
	if !ok {
		return "", errors.New("no tenant on the activity context")
	}
	return tenant.String(), nil
}

func callSeenTenant(ctx workflow.Context) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 5},
	})
	var out string
	err := workflow.ExecuteActivity(ctx, seenTenant).Get(ctx, &out)
	return out, err
}

func TestActivityGuard(t *testing.T) {
	id := idOf(tenantA)
	t.Run("allow puts the tenant on the context", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.RegisterActivity(seenTenant)
		h.env.ExecuteWorkflow(callSeenTenant)
		var out string
		if err := h.env.GetWorkflowResult(&out); err != nil || out != tenantA.String() {
			t.Fatalf("result %q, %v; want %s", out, err, tenantA)
		}
		h.wantNoAlerts()
	})
	for _, tc := range []struct {
		name, outcome string
		header        *commonpb.Payload
	}{
		// TICK-84 finding 1: a header naming another tenant fails, not only a missing one.
		{"other tenant", outcomeMismatch, stringPayload(t, tenantB.String())},
		{"missing", outcomeMissing, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, id)
			attempts := 0
			h.env.RegisterActivityWithOptions(func(ctx context.Context) (string, error) {
				attempts++
				return seenTenant(ctx)
			}, activity.RegisterOptions{Name: "seenTenant"})
			h.tr.setActivity(tc.header)
			h.env.ExecuteWorkflow(callSeenTenant)
			err := h.env.GetWorkflowError()
			wantGuardError(t, err, guardActivity, tc.outcome)
			if attempts != 0 {
				t.Errorf("the activity body ran %d times; the guard must stop it", attempts)
			}
			if got := h.decisions()[guardActivity+"/"+tc.outcome]; got != 1 {
				t.Errorf("%d %s decisions, want 1 (non-retryable: one attempt)", got, tc.outcome)
			}
			var found bool
			for _, a := range h.alerts() {
				found = found || (a["guard"] == guardActivity && a["workflow_id"] == id && a["activity_type"] == "seenTenant")
			}
			if !found {
				t.Errorf("no activity alert with the workflow id and type: %s", h.logs.String())
			}
		})
	}
}

// --- the propagator's outbound header -------------------------------------------

func noop(context.Context) error { return nil }

// headersParent runs an activity and a child, so the transport sees the
// header each command carried.
func headersParent(ctx workflow.Context) (string, error) {
	actx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
	if err := workflow.ExecuteActivity(actx, noop).Get(ctx, nil); err != nil {
		return "", err
	}
	id, err := ChildWorkflowID(ctx, "child")
	if err != nil {
		return "", err
	}
	cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: id})
	return id, workflow.ExecuteChildWorkflow(cctx, child).Get(ctx, nil)
}

func TestPropagatorHeaderOnActivityAndChild(t *testing.T) {
	h := newHarness(t, idOf(tenantA))
	h.env.RegisterActivity(noop)
	h.env.RegisterWorkflow(child)
	h.env.ExecuteWorkflow(headersParent)
	var childID string
	if err := h.env.GetWorkflowResult(&childID); err != nil {
		t.Fatal(err)
	}
	for what, p := range map[string]*commonpb.Payload{
		"activity": h.tr.header("activity:noop"),
		"child":    h.tr.header(childID),
	} {
		if enc := string(p.GetMetadata()["encoding"]); enc != "json/plain" {
			t.Errorf("%s header encoding %q, want json/plain", what, enc)
		}
		var got string
		if err := converter.GetDefaultDataConverter().FromPayload(p, &got); err != nil || got != tenantA.String() {
			t.Errorf("%s header decodes to %q, %v; want %s", what, got, err, tenantA)
		}
	}
	h.wantNoAlerts()
}

func continueOnce(ctx workflow.Context, again bool) error {
	if again {
		return workflow.NewContinueAsNewError(ctx, continueOnce, false)
	}
	return nil
}

func TestPropagatorHeaderOnContinueAsNew(t *testing.T) {
	h := newHarness(t, idOf(tenantA))
	h.env.ExecuteWorkflow(continueOnce, true)
	var can *workflow.ContinueAsNewError
	if err := h.env.GetWorkflowError(); !errors.As(err, &can) {
		t.Fatalf("error %v, want continue-as-new", err)
	}
	p := can.Header.GetFields()[TenantHeader]
	if enc := string(p.GetMetadata()["encoding"]); enc != "json/plain" {
		t.Errorf("continue-as-new header encoding %q, want json/plain", enc)
	}
	var got string
	if err := converter.GetDefaultDataConverter().FromPayload(p, &got); err != nil || got != tenantA.String() {
		t.Errorf("continue-as-new header decodes to %q, %v; want %s", got, err, tenantA)
	}
	if d := h.decisions()[guardContinueAsNew+"/"+outcomeAllow]; d != 1 {
		t.Errorf("continue-as-new allow count %d, want 1", d)
	}
}

// Without the Propagator the continue-as-new carries no header; the guard
// fails the workflow instead of sending a command the Authorizer denies.
func TestContinueAsNewGuardMissingHeader(t *testing.T) {
	id := idOf(tenantA)
	h := newHarness(t, id, withoutPropagator())
	h.env.ExecuteWorkflow(continueOnce, true)
	wantGuardError(t, h.env.GetWorkflowError(), guardContinueAsNew, outcomeMissing)
	h.wantDenyAlert(guardContinueAsNew, outcomeMissing, id)
}

// The SDK copies the start's search attributes onto continue-as-new; another
// tenant's NavaTenantId there would be denied at the Authorizer, so the guard
// fails the workflow instead. The test environment drops start search
// attributes, so this checks the predicate the guard applies to them.
func TestContinueAsNewSearchAttributeCheck(t *testing.T) {
	sa := func(p *commonpb.Payload) *commonpb.SearchAttributes {
		return &commonpb.SearchAttributes{IndexedFields: map[string]*commonpb.Payload{TenantSearchAttribute: p}}
	}
	for name, tc := range map[string]struct {
		sa   *commonpb.SearchAttributes
		want bool
	}{
		"none":           {nil, true},
		"other keys":     {&commonpb.SearchAttributes{IndexedFields: map[string]*commonpb.Payload{"X": stringPayload(t, "y")}}, true},
		"own tenant":     {sa(stringPayload(t, tenantA.String())), true},
		"another tenant": {sa(stringPayload(t, tenantB.String())), false},
		"not a string":   {sa(&commonpb.Payload{Data: []byte(tenantA.String())}), false},
	} {
		if got := indexedTenantOK(tc.sa, tenantA); got != tc.want {
			t.Errorf("%s: indexedTenantOK = %v, want %v", name, got, tc.want)
		}
	}
}

// --- children -------------------------------------------------------------------

func child(ctx workflow.Context) (string, error) {
	return workflow.GetInfo(ctx).WorkflowExecution.ID, nil
}

// startChild starts child with opts, after setting its id with the helper if
// useHelper, and returns the child's own id.
func startChild(ctx workflow.Context, useHelper bool, opts workflow.ChildWorkflowOptions) (string, error) {
	if useHelper {
		id, err := ChildWorkflowID(ctx, "approval")
		if err != nil {
			return "", err
		}
		opts.WorkflowID = id
	}
	var out string
	err := workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, opts), child).Get(ctx, &out)
	return out, err
}

func TestChildGuard(t *testing.T) {
	id := idOf(tenantA)
	t.Run("helper id starts", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.RegisterWorkflow(child)
		h.env.ExecuteWorkflow(startChild, true, workflow.ChildWorkflowOptions{})
		var out string
		if err := h.env.GetWorkflowResult(&out); err != nil {
			t.Fatalf("child with the helper id: %v", err)
		}
		prefix := "t:" + tenantA.String() + ":approval:"
		if !strings.HasPrefix(out, prefix) {
			t.Errorf("child id %q, want prefix %q", out, prefix)
		}
		if _, err := uuid.Parse(strings.TrimPrefix(out, prefix)); err != nil {
			t.Errorf("child id %q does not end in a uuid", out)
		}
		h.wantNoAlerts()
		if d := h.decisions(); d[guardChild+"/"+outcomeAllow] != 1 || d[guardWorkflowStart+"/"+outcomeAllow] != 2 {
			t.Errorf("decisions %v, want the child allowed and both starts allowed", d)
		}
	})
	for _, tc := range []struct {
		name, outcome string
		opts          workflow.ChildWorkflowOptions
		useHelper     bool
	}{
		{"SDK default id", outcomeCrossTenant, workflow.ChildWorkflowOptions{}, false},
		{"other tenant", outcomeCrossTenant, workflow.ChildWorkflowOptions{WorkflowID: idOf(tenantB)}, false},
		{"no prefix", outcomeCrossTenant, workflow.ChildWorkflowOptions{WorkflowID: "child-1"}, false},
		{"other namespace", outcomeNamespace, workflow.ChildWorkflowOptions{Namespace: "other"}, true}, //nolint:staticcheck // SA1019: still settable
		{"cron", outcomeCron, workflow.ChildWorkflowOptions{CronSchedule: "* * * * *"}, true},
		{"NavaTenantId of another tenant", outcomeSearchAttribute, workflow.ChildWorkflowOptions{
			TypedSearchAttributes: temporal.NewSearchAttributes(tenantSearchAttributeKey().ValueSet(tenantB.String())),
		}, true},
		{"untyped NavaTenantId of another tenant", outcomeSearchAttribute, workflow.ChildWorkflowOptions{
			SearchAttributes: map[string]any{TenantSearchAttribute: tenantB.String()}, //nolint:staticcheck // SA1019: still settable
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, id)
			ran := false
			h.env.RegisterWorkflowWithOptions(func(ctx workflow.Context) (string, error) { ran = true; return "", nil },
				workflow.RegisterOptions{Name: "child"})
			// A closure: options do not survive a workflow argument's encoding.
			h.env.ExecuteWorkflow(func(ctx workflow.Context) (string, error) { return startChild(ctx, tc.useHelper, tc.opts) })
			wantGuardError(t, h.env.GetWorkflowError(), guardChild, tc.outcome)
			if ran {
				t.Error("the child ran")
			}
			h.wantDenyAlert(guardChild, tc.outcome, id)
		})
	}
	t.Run("NavaTenantId of its own tenant", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.RegisterWorkflow(child)
		h.env.ExecuteWorkflow(func(ctx workflow.Context) (string, error) {
			return startChild(ctx, true, workflow.ChildWorkflowOptions{
				TypedSearchAttributes: temporal.NewSearchAttributes(tenantSearchAttributeKey().ValueSet(tenantA.String())),
			})
		})
		if err := h.env.GetWorkflowError(); err != nil {
			t.Fatal(err)
		}
		h.wantNoAlerts()
	})
}

func TestChildWorkflowIDRejectsBadKind(t *testing.T) {
	h := newHarness(t, idOf(tenantA))
	h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
		for _, kind := range []string{"", "a:b", "Upper", "-x", strings.Repeat("a", 64)} {
			if _, err := ChildWorkflowID(ctx, kind); err == nil {
				return errors.New("kind " + kind + " accepted")
			}
		}
		return nil
	})
	if err := h.env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
}

// --- signal-external and cancel-external ----------------------------------------

// waitForSignal blocks until "ping" or cancellation, and reports which.
func waitForSignal(ctx workflow.Context) (string, error) {
	var got string
	sel := workflow.NewSelector(ctx)
	sel.AddReceive(workflow.GetSignalChannel(ctx, "ping"), func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, &got) })
	sel.AddReceive(ctx.Done(), func(workflow.ReceiveChannel, bool) { got = "cancelled" })
	sel.Select(ctx)
	return got, nil
}

// external starts a waiting child, then signals (or cancels) target, or the
// child itself if target is empty, and returns the child's result.
func external(ctx workflow.Context, cancel bool, target, namespace string) (string, error) {
	id, err := ChildWorkflowID(ctx, "waiter")
	if err != nil {
		return "", err
	}
	cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: id})
	fut := workflow.ExecuteChildWorkflow(cctx, waitForSignal)
	if err := fut.GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		return "", err
	}
	if target == "" {
		target = id
	}
	octx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{Namespace: namespace}) //nolint:staticcheck // SA1019: still settable
	if cancel {
		err = workflow.RequestCancelExternalWorkflow(octx, target, "").Get(ctx, nil)
	} else {
		err = workflow.SignalExternalWorkflow(octx, target, "", "ping", "pong").Get(ctx, nil)
	}
	if err != nil {
		return "", err
	}
	var out string
	err = fut.Get(ctx, &out)
	if cancel && temporal.IsCanceledError(err) {
		return "cancelled", nil
	}
	return out, err
}

func TestSignalAndCancelExternalGuard(t *testing.T) {
	id := idOf(tenantA)
	for _, tc := range []struct {
		name   string
		cancel bool
		guard  string
		want   string
	}{
		{"signal", false, guardSignalExternal, "pong"},
		{"cancel", true, guardCancelExternal, "cancelled"},
	} {
		t.Run(tc.name+" same tenant", func(t *testing.T) {
			h := newHarness(t, id)
			h.env.RegisterWorkflow(waitForSignal)
			h.env.ExecuteWorkflow(external, tc.cancel, "", "")
			var out string
			if err := h.env.GetWorkflowResult(&out); err != nil || out != tc.want {
				t.Fatalf("result %q, %v; want %q", out, err, tc.want)
			}
			h.wantNoAlerts()
		})
		for _, deny := range []struct {
			name, target, namespace, outcome string
		}{
			{"other tenant", idOf(tenantB), "", outcomeCrossTenant},
			{"no prefix", "someone", "", outcomeCrossTenant},
			{"other namespace", "", "other", outcomeNamespace},
		} {
			t.Run(tc.name+" "+deny.name, func(t *testing.T) {
				h := newHarness(t, id)
				h.env.RegisterWorkflow(waitForSignal)
				h.env.ExecuteWorkflow(external, tc.cancel, deny.target, deny.namespace)
				wantGuardError(t, h.env.GetWorkflowError(), tc.guard, deny.outcome)
				h.wantDenyAlert(tc.guard, deny.outcome, id)
			})
		}
	}
}

// --- search attribute upserts ---------------------------------------------------

func upsert(ctx workflow.Context, typed bool, value *string, otherKey bool) error {
	if typed {
		var u temporal.SearchAttributeUpdate
		switch {
		case otherKey:
			u = temporal.NewSearchAttributeKeyKeyword("CustomKeywordField").ValueSet("x")
		case value == nil:
			u = tenantSearchAttributeKey().ValueUnset()
		default:
			u = tenantSearchAttributeKey().ValueSet(*value)
		}
		return workflow.UpsertTypedSearchAttributes(ctx, u)
	}
	m := map[string]any{TenantSearchAttribute: nil}
	switch {
	case otherKey:
		m = map[string]any{"CustomKeywordField": "x"}
	case value != nil:
		m[TenantSearchAttribute] = *value
	}
	//nolint:staticcheck // SA1019: the untyped form is deprecated, but a workflow can still call it.
	return workflow.UpsertSearchAttributes(ctx, m)
}

func TestUpsertSearchAttributeGuard(t *testing.T) {
	id := idOf(tenantA)
	own, other := tenantA.String(), tenantB.String()
	for _, typed := range []bool{true, false} {
		for _, tc := range []struct {
			name     string
			value    *string
			otherKey bool
			allow    bool
		}{
			{"own tenant", &own, false, true},
			{"another key", nil, true, true},
			{"another tenant", &other, false, false},
			{"unset", nil, false, false},
		} {
			name := tc.name
			if typed {
				name += " typed"
			}
			t.Run(name, func(t *testing.T) {
				h := newHarness(t, id)
				h.env.ExecuteWorkflow(upsert, typed, tc.value, tc.otherKey)
				err := h.env.GetWorkflowError()
				if tc.allow {
					if err != nil {
						t.Fatal(err)
					}
					h.wantNoAlerts()
					return
				}
				wantGuardError(t, err, guardSearchAttribute, outcomeSearchAttribute)
				h.wantDenyAlert(guardSearchAttribute, outcomeSearchAttribute, id)
			})
		}
	}
}

// --- Nexus ----------------------------------------------------------------------

func TestNexusDenied(t *testing.T) {
	id := idOf(tenantA)
	h := newHarness(t, id)
	h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
		c := workflow.NewNexusClient("endpoint", "service")
		return c.ExecuteOperation(ctx, "op", nil, workflow.NexusOperationOptions{}).Get(ctx, nil)
	})
	wantGuardError(t, h.env.GetWorkflowError(), guardNexus, outcomeNotAllowed)
	h.wantDenyAlert(guardNexus, outcomeNotAllowed, id)
}

func TestNewRejectsNil(t *testing.T) {
	if _, err := New(nil, nil); err == nil {
		t.Error("New(nil, nil) succeeded")
	}
}
