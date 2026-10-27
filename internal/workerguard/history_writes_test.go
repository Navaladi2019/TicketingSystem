package workerguard

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// The remaining ways workflow code writes to its own history (TICK-245):
// signal-external and signal-child arguments, and memos.

// signalWaiter starts a waitForSignal child, signals it with arg through
// SignalExternalWorkflow or, if viaChild, the child future, and returns what
// the child returns.
func signalWaiter(ctx workflow.Context, viaChild bool, arg any) (string, error) {
	id, err := ChildWorkflowID(ctx, "waiter")
	if err != nil {
		return "", err
	}
	fut := workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: id}), waitForSignal)
	if err := fut.GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		return "", err
	}
	if viaChild {
		err = fut.SignalChildWorkflow(ctx, "ping", arg).Get(ctx, nil)
	} else {
		err = workflow.SignalExternalWorkflow(ctx, id, "", "ping", arg).Get(ctx, nil)
	}
	if err != nil {
		return "", err
	}
	var out string
	err = fut.Get(ctx, &out)
	return out, err
}

func TestSignalArgsGuard(t *testing.T) {
	id := idOf(tenantA)
	for _, via := range []struct {
		name     string
		viaChild bool
	}{{"signal-external", false}, {"signal-child", true}} {
		for _, tc := range []struct {
			name string
			arg  any
		}{
			{"title", map[string]string{"title": recordData}},
			{"string", recordData},
			{"reference next to text", map[string]string{"record_id": uuid.NewString(), "note": recordData}},
		} {
			t.Run(via.name+" with record data ("+tc.name+") fails before it is sent", func(t *testing.T) {
				h := newHarness(t, id)
				h.env.RegisterWorkflow(waitForSignal)
				// A closure: a Delivery argument would decode as a map.
				h.env.ExecuteWorkflow(func(ctx workflow.Context) (string, error) { return signalWaiter(ctx, via.viaChild, tc.arg) })
				err := h.env.GetWorkflowError()
				wantGuardError(t, err, guardSignalExternal, outcomeRecordData)
				h.wantDenyAlert(guardSignalExternal, outcomeRecordData, id)
				h.wantNoRecordData(err)
			})
		}
		for _, tc := range []struct {
			name string
			arg  any
		}{
			{"references", map[string]string{"record_id": uuid.NewString()}},
			{"a Delivery", validDelivery(t)},
			{"nil", nil},
		} {
			t.Run(via.name+" with "+tc.name+" is delivered", func(t *testing.T) {
				h := newHarness(t, id)
				h.env.RegisterWorkflow(waitForSignal)
				// A closure: a Delivery argument would decode as a map.
				h.env.ExecuteWorkflow(func(ctx workflow.Context) (string, error) { return signalWaiter(ctx, via.viaChild, tc.arg) })
				var out string
				if err := h.env.GetWorkflowResult(&out); err != nil || out != "ping" {
					t.Fatalf("result %q, %v; want the signal delivered", out, err)
				}
				h.wantNoAlerts()
				if d := h.decisions()[guardSignalExternal+"/"+outcomeAllow]; d != 1 {
					t.Errorf("%s allow count %d, want 1", guardSignalExternal, d)
				}
			})
		}
	}
	// The target is still checked first: a cross-tenant signal with a valid
	// argument is cross_tenant, not allowed.
	t.Run("cross-tenant target with references", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
			return workflow.SignalExternalWorkflow(ctx, idOf(tenantB), "", "ping", map[string]string{"record_id": uuid.NewString()}).Get(ctx, nil)
		})
		wantGuardError(t, h.env.GetWorkflowError(), guardSignalExternal, outcomeCrossTenant)
	})
}

func TestChildMemoGuard(t *testing.T) {
	id := idOf(tenantA)
	for _, tc := range []struct {
		name string
		memo map[string]any
	}{
		{"record data", map[string]any{"title": recordData}},
		{"a reference", map[string]any{"record_id": uuid.NewString()}},
	} {
		t.Run(tc.name+" in the memo fails before the child starts", func(t *testing.T) {
			h := newHarness(t, id)
			ran := false
			h.env.RegisterWorkflowWithOptions(func(workflow.Context) error { ran = true; return nil },
				workflow.RegisterOptions{Name: "child"})
			h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
				cid, err := ChildWorkflowID(ctx, "child")
				if err != nil {
					return err
				}
				cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: cid, Memo: tc.memo})
				return workflow.ExecuteChildWorkflow(cctx, "child").Get(ctx, nil)
			})
			err := h.env.GetWorkflowError()
			wantGuardError(t, err, guardChild, outcomeRecordData)
			if ran {
				t.Error("the child ran")
			}
			h.wantDenyAlert(guardChild, outcomeRecordData, id)
			h.wantNoRecordData(err)
		})
	}
	t.Run("an empty memo starts it", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.RegisterWorkflow(child)
		h.env.ExecuteWorkflow(func(ctx workflow.Context) (string, error) {
			return startChild(ctx, true, workflow.ChildWorkflowOptions{Memo: map[string]any{}})
		})
		if err := h.env.GetWorkflowError(); err != nil {
			t.Fatal(err)
		}
		h.wantNoAlerts()
	})
}

func TestUpsertMemoGuard(t *testing.T) {
	id := idOf(tenantA)
	for _, tc := range []struct {
		name string
		memo map[string]any
	}{
		{"record data", map[string]any{"title": recordData}},
		{"a reference", map[string]any{"record_id": uuid.NewString()}},
		{"a removal", map[string]any{"title": nil}},
	} {
		t.Run(tc.name+" fails", func(t *testing.T) {
			h := newHarness(t, id)
			h.env.ExecuteWorkflow(func(ctx workflow.Context) error { return workflow.UpsertMemo(ctx, tc.memo) })
			err := h.env.GetWorkflowError()
			wantGuardError(t, err, guardMemo, outcomeRecordData)
			h.wantDenyAlert(guardMemo, outcomeRecordData, id)
			h.wantNoRecordData(err)
		})
	}
	// An empty upsert records nothing: the guard passes it on, and the SDK
	// refuses it.
	t.Run("an empty upsert is the SDK's error, not a deny", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error { return workflow.UpsertMemo(ctx, map[string]any{}) })
		if err := h.env.GetWorkflowError(); err == nil || strings.Contains(err.Error(), ErrTypeTenantGuard) {
			t.Fatalf("error %v, want the SDK's empty-memo error", err)
		}
		h.wantNoAlerts()
	})
}

// recordingConverter is a converter that is not the guard's: it would record
// a different encoding from the one argOK checked.
type recordingConverter struct{ converter.DataConverter }

// Every platform client, so every worker, encodes with the converter argOK
// checks with (TICK-245): ConfigureClient replaces any other, and the bytes
// it records are the bytes the guard checked.
func TestConfigureClientUsesTheGuardsDataConverter(t *testing.T) {
	for _, tc := range []struct {
		name string
		dc   converter.DataConverter
	}{
		{"unset", nil},
		{"another converter", recordingConverter{converter.NewCompositeDataConverter(converter.NewJSONPayloadConverter())}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := client.Options{DataConverter: tc.dc}
			ConfigureClient(&o, &telemetry.Temporal{})
			if o.DataConverter != DataConverter() {
				t.Fatalf("client converter %T, want workerguard.DataConverter", o.DataConverter)
			}
			arg := validRunInput(t)
			recorded, err := o.DataConverter.ToPayload(arg)
			if err != nil {
				t.Fatal(err)
			}
			checked, err := DataConverter().ToPayload(arg)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(recorded.GetData(), checked.GetData()) {
				t.Errorf("recorded %s, checked %s", recorded.GetData(), checked.GetData())
			}
		})
	}
}
