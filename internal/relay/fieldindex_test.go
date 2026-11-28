package relay

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// TICK-745 (PLAN-007 D-7-12): a field_index.backfill row is a
// SignalWithStart of the tenant's coordinator, never a start of the backfill.

const fieldAID = "0192bbbb-0000-7000-8000-0000000000f1"

// backfillRow is a claimed field_index.backfill start row, as
// outbox.EnqueueFieldIndexBackfill writes it, for requested_version seq.
func backfillRow(seq int64) Claimed {
	id := uuid.Must(uuid.NewV7())
	return Claimed{ID: id, TenantID: uuid.MustParse(tenantAID), Kind: outbox.FieldIndexBackfillKind, KindVersion: 1,
		TargetKey: outbox.FieldIndexBackfillKind + ":" + id.String(), TargetSeq: seq, Attempts: 1,
		Payload: json.RawMessage(`{"tenant_id":"` + tenantAID + `","field_id":"` + fieldAID + `"}`)}
}

func TestFieldIndexBackfillIsSignalWithStartOfTheCoordinator(t *testing.T) {
	h := newHarness(t)
	c := backfillRow(3)
	h.dispatch(t.Context(), c)

	if _, starts, signals := h.fe.counts(); starts != 0 || signals != 0 {
		t.Fatalf("%d starts and %d signals, want only a signal-with-start", starts, signals)
	}
	if len(h.fe.sws) != 1 {
		t.Fatalf("%d signal-with-starts, want 1", len(h.fe.sws))
	}
	req := h.fe.sws[0]
	if want := "t:" + tenantAID + ":field_index.coordinator"; req.GetWorkflowId() != want {
		t.Errorf("workflow id %q, want %q", req.GetWorkflowId(), want)
	}
	if req.GetWorkflowType().GetName() != fieldindex.CoordinatorWorkflowType || req.GetSignalName() != fieldindex.SignalEnqueue ||
		req.GetTaskQueue().GetName() != TaskQueue {
		t.Errorf("type %q signal %q queue %q", req.GetWorkflowType().GetName(), req.GetSignalName(), req.GetTaskQueue().GetName())
	}
	// The coordinator completes when idle, so its id must start again.
	if req.GetWorkflowIdReusePolicy() != enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE {
		t.Errorf("reuse policy %v, want ALLOW_DUPLICATE", req.GetWorkflowIdReusePolicy())
	}
	if req.GetRetryPolicy() != nil || req.GetCronSchedule() != "" {
		t.Error("a relay start sets no retry policy or cron (TICK-573)")
	}
	if got, _ := headerString(t, req.GetHeader(), workerguard.TenantHeader); got != tenantAID {
		t.Errorf("nava-tenant header %q, want %s", got, tenantAID)
	}
	var item fieldindex.EnqueueItem
	if err := json.Unmarshal(req.GetSignalInput().GetPayloads()[0].GetData(), &item); err != nil {
		t.Fatal(err)
	}
	want := fieldindex.EnqueueItem{Op: fieldindex.OpBackfill, FieldID: fieldAID, RequestedVersion: 3, OutboxID: c.ID.String()}
	if item != want {
		t.Errorf("signal %+v, want %+v", item, want)
	}
	var in workerguard.RunInput
	if err := json.Unmarshal(req.GetInput().GetPayloads()[0].GetData(), &in); err != nil || in.OutboxID != c.ID {
		t.Errorf("input %s (%v), want the row's Delivery", req.GetInput().GetPayloads()[0].GetData(), err)
	}
	if got := h.db.only(t); got.op != "dispatched" || got.id != c.ID {
		t.Errorf("settled %+v, want dispatched", got)
	}
	if n, _ := h.counter(t, "outbox_relay_dispatches", map[string]string{"op": "signal_with_start", "outcome": "dispatched"}); n != 1 {
		t.Errorf("dispatches{signal_with_start,dispatched} = %d, want 1", n)
	}
}

// Replay race: the same row delivered twice reaches one coordinator
// execution; the coordinator drops the second signal on its outbox id.
func TestFieldIndexBackfillDeliveredTwiceStartsOneCoordinator(t *testing.T) {
	h := newHarness(t)
	c := backfillRow(1)
	h.dispatch(t.Context(), c)
	h.dispatch(t.Context(), c)
	if executions, _, _ := h.fe.counts(); executions != 1 {
		t.Fatalf("%d executions, want 1", executions)
	}
	if len(h.fe.sws) != 2 {
		t.Fatalf("%d signal-with-starts, want 2", len(h.fe.sws))
	}
}

// Two fields of one tenant reach the same coordinator: neither is dropped
// as WorkflowExecutionAlreadyStarted (the D-7-12 failure of a plain start).
func TestTwoFieldsShareTheCoordinator(t *testing.T) {
	h := newHarness(t)
	a := backfillRow(1)
	b := backfillRow(1)
	b.Payload = json.RawMessage(`{"tenant_id":"` + tenantAID + `","field_id":"0192bbbb-0000-7000-8000-0000000000f2"}`)
	h.dispatch(t.Context(), a)
	h.dispatch(t.Context(), b)
	if executions, _, _ := h.fe.counts(); executions != 1 || len(h.fe.sws) != 2 {
		t.Fatalf("%d executions and %d signal-with-starts, want 1 and 2", executions, len(h.fe.sws))
	}
	for _, c := range h.db.calls {
		if c.op != "dispatched" {
			t.Errorf("settled %+v, want dispatched", c)
		}
	}
}

// A row whose field id or version the coordinator could not use is marked
// dead without a Temporal call.
func TestFieldIndexBackfillMalformedRowIsDead(t *testing.T) {
	for name, mutate := range map[string]func(*Claimed){
		"version 0":         func(c *Claimed) { c.TargetSeq = 0 },
		"version too large": func(c *Claimed) { c.TargetSeq = 32768 },
		"nil field": func(c *Claimed) {
			c.Payload = json.RawMessage(`{"tenant_id":"` + tenantAID + `","field_id":"00000000-0000-0000-0000-000000000000"}`)
		},
		// TICK-748: the payload names another tenant than the row.
		"payload tenant differs": func(c *Claimed) {
			c.Payload = json.RawMessage(`{"tenant_id":"0192aaaa-0000-7000-8000-00000000000b","field_id":"` + fieldAID + `"}`)
		},
		"payload tenant upper case": func(c *Claimed) {
			c.Payload = json.RawMessage(`{"tenant_id":"0192AAAA-0000-7000-8000-00000000000A","field_id":"` + fieldAID + `"}`)
		},
		"extra key": func(c *Claimed) {
			c.Payload = json.RawMessage(`{"tenant_id":"` + tenantAID + `","field_id":"` + fieldAID + `","x":"` + fieldAID + `"}`)
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			c := backfillRow(1)
			mutate(&c)
			h.dispatch(t.Context(), c)
			if len(h.fe.sws) != 0 {
				t.Fatal("a malformed row reached Temporal")
			}
			if got := h.db.only(t); got.op != "dead" || got.cause != CauseInvalidPayload {
				t.Errorf("settled %+v, want dead with %q", got, CauseInvalidPayload)
			}
		})
	}
}

// A failed signal-with-start is retried, never marked dead, even NotFound.
func TestFieldIndexBackfillFailureBacksOff(t *testing.T) {
	h := newHarness(t)
	h.fe.setFail(func(method, _ string) error {
		if method == "signal_with_start" {
			return serviceerror.NewNotFound("namespace not found")
		}
		return nil
	})
	c := backfillRow(1)
	h.dispatch(t.Context(), c)
	if got := h.db.only(t); got.op != "backoff" {
		t.Errorf("settled %+v, want backoff", got)
	}
}
