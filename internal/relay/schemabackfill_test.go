package relay

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/schemabackfill"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// TICK-920 (P10-03, ADR-009 D2, contract C3): a schema.backfill row is a
// start of t:<tenant>:schema.backfill:<entity_definition_id>:<target_version>
// on schema-backfill, the id built from the row's tenant, the payload's
// definition and target_seq.

const defAID = "0192bbbb-0000-7000-8000-0000000000d1"

// schemaBackfillRow is a claimed schema.backfill start row, as
// outbox.EnqueueSchemaBackfill and the rollout migration write it, for
// target version seq.
func schemaBackfillRow(seq int64) Claimed {
	id := uuid.Must(uuid.NewV7())
	return Claimed{ID: id, TenantID: uuid.MustParse(tenantAID), Kind: outbox.SchemaBackfillKind, KindVersion: 1,
		TargetKey: outbox.SchemaBackfillKind + ":" + id.String(), TargetSeq: seq, Attempts: 1,
		Payload: json.RawMessage(`{"entity_definition_id":"` + defAID + `"}`)}
}

func TestSchemaBackfillIsAStartOfTheJob(t *testing.T) {
	h := newHarness(t)
	c := schemaBackfillRow(3)
	h.dispatch(t.Context(), c)

	if len(h.fe.starts) != 1 || len(h.fe.sws) != 0 || len(h.fe.signals) != 0 {
		t.Fatalf("%d starts, %d signal-with-starts, %d signals; want one start", len(h.fe.starts), len(h.fe.sws), len(h.fe.signals))
	}
	req := h.fe.starts[0]
	if want := "t:" + tenantAID + ":schema.backfill:" + defAID + ":3"; req.GetWorkflowId() != want {
		t.Errorf("workflow id %q, want %q", req.GetWorkflowId(), want)
	}
	if req.GetWorkflowType().GetName() != schemabackfill.WorkflowType || req.GetTaskQueue().GetName() != schemabackfill.TaskQueue {
		t.Errorf("type %q queue %q, want %s on %s", req.GetWorkflowType().GetName(), req.GetTaskQueue().GetName(),
			schemabackfill.WorkflowType, schemabackfill.TaskQueue)
	}
	if req.GetWorkflowIdReusePolicy() != enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE {
		t.Errorf("reuse policy %v, want REJECT_DUPLICATE", req.GetWorkflowIdReusePolicy())
	}
	if req.GetRetryPolicy() != nil || req.GetCronSchedule() != "" {
		t.Error("a relay start sets no retry policy or cron (TICK-573)")
	}
	if got, _ := headerString(t, req.GetHeader(), workerguard.TenantHeader); got != tenantAID {
		t.Errorf("nava-tenant header %q, want %s", got, tenantAID)
	}
	var in outbox.Delivery
	if err := json.Unmarshal(req.GetInput().GetPayloads()[0].GetData(), &in); err != nil || in.OutboxID != c.ID ||
		string(in.Payload) != `{"entity_definition_id":"`+defAID+`"}` {
		t.Errorf("input %s (%v), want the row's Delivery with no tenant", req.GetInput().GetPayloads()[0].GetData(), err)
	}
	if got := h.db.only(t); got.op != "dispatched" || got.id != c.ID {
		t.Errorf("settled %+v, want dispatched", got)
	}
}

// CI-11 (relay half): the same row delivered twice gives one workflow
// execution; the second start's AlreadyStarted acknowledges the row.
func TestSchemaBackfillDeliveredTwiceStartsOneJob(t *testing.T) {
	h := newHarness(t)
	c := schemaBackfillRow(2)
	h.dispatch(t.Context(), c)
	h.dispatch(t.Context(), c)
	if executions, _, _ := h.fe.counts(); executions != 1 {
		t.Fatalf("%d executions, want 1", executions)
	}
	for _, s := range h.db.calls {
		if s.op != "dispatched" {
			t.Errorf("settled %+v, want dispatched both times", s)
		}
	}
}

// A row the job could not use, or that names a tenant in its payload, is
// marked dead without a Temporal call (C3: the relay rejects a payload that
// carries a tenant_id).
func TestSchemaBackfillMalformedRowIsDead(t *testing.T) {
	for name, mutate := range map[string]func(*Claimed){
		"version 0":         func(c *Claimed) { c.TargetSeq = 0 },
		"version too large": func(c *Claimed) { c.TargetSeq = 1 << 31 },
		"nil definition": func(c *Claimed) {
			c.Payload = json.RawMessage(`{"entity_definition_id":"00000000-0000-0000-0000-000000000000"}`)
		},
		"upper-case definition": func(c *Claimed) {
			c.Payload = json.RawMessage(`{"entity_definition_id":"0192BBBB-0000-7000-8000-0000000000D1"}`)
		},
		"payload names a tenant": func(c *Claimed) {
			c.Payload = json.RawMessage(`{"entity_definition_id":"` + defAID + `","tenant_id":"0192aaaa-0000-7000-8000-00000000000b"}`)
		},
		"no definition": func(c *Claimed) { c.Payload = json.RawMessage(`{}`) },
		"not a start": func(c *Claimed) {
			c.TargetKey = outbox.SchemaBackfillKind + ":" + uuid.NewString()
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			c := schemaBackfillRow(1)
			mutate(&c)
			h.dispatch(t.Context(), c)
			if len(h.fe.starts) != 0 || len(h.fe.signals) != 0 {
				t.Fatal("a malformed row reached Temporal")
			}
			if got := h.db.only(t); got.op != "dead" {
				t.Errorf("settled %+v, want dead", got)
			}
		})
	}
}
