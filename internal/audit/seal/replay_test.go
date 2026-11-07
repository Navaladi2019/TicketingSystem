package seal_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The audit.seal replay gate (P5-34b): every history recorded under
// historiesDir by `make seal-record-histories` replays against today's
// audit.seal, behind the worker guard and the Propagator as cmd/nava-worker
// runs it. A change to the commands a recorded run produced fails here with
// a non-determinism error; ship it behind workflow.GetVersion and record a
// new history (docs/quality/ci-pipeline.md, "Changing nava.interpreter",
// applies to audit.seal the same way). Histories are added, never removed.

var historiesDir = filepath.Join("testdata", "histories") //nolint:gochecknoglobals // a fixed path

// wantHistories is the set the gate requires: a live lineage, a stale one,
// an A1 retried after its commit, and a poison A2.
//
//nolint:gochecknoglobals // the fixed set
var wantHistories = []string{"a1-retried.json", "next-run.json", "poison.json", "stale.json"}

type sealHistory struct {
	name, workflowID string
	h                *historypb.History
}

func loadHistories(t *testing.T) []sealHistory {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(historiesDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range paths {
		names = append(names, filepath.Base(p))
	}
	slices.Sort(names)
	if !slices.Equal(names, wantHistories) {
		t.Fatalf("histories in %s are %v, want exactly %v (append-only: list a new one in wantHistories)", historiesDir, names, wantHistories)
	}
	out := make([]sealHistory, 0, len(names))
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(historiesDir, n))
		if err != nil {
			t.Fatal(err)
		}
		h, err := client.HistoryFromJSON(bytes.NewReader(b), client.HistoryJSONOptions{})
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		first := h.GetEvents()[0].GetWorkflowExecutionStartedEventAttributes()
		if first.GetWorkflowType().GetName() != seal.WorkflowType || first.GetTaskQueue().GetName() != "nava-tq" {
			t.Fatalf("%s: a %q run on %q, want audit.seal on nava-tq", n, first.GetWorkflowType().GetName(), first.GetTaskQueue().GetName())
		}
		out = append(out, sealHistory{name: n, workflowID: first.GetWorkflowId(), h: h})
	}
	return out
}

func replay(t *testing.T, h sealHistory) error {
	t.Helper()
	g, err := workerguard.New(slog.New(slog.DiscardHandler), sdkmetric.NewMeterProvider())
	if err != nil {
		t.Fatal(err)
	}
	r, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{
		Interceptors:       []interceptor.WorkerInterceptor{g},
		ContextPropagators: []workflow.ContextPropagator{workerguard.Propagator{}},
		DataConverter:      workerguard.DataConverter(),
	})
	if err != nil {
		t.Fatal(err)
	}
	seal.RegisterWorkflow(r)
	return r.ReplayWorkflowHistoryWithOptions(nil, h.h, worker.ReplayWorkflowHistoryOptions{
		OriginalExecution: workflow.Execution{ID: h.workflowID, RunID: "replay"},
	})
}

// TestReplayRecordedHistories is the gate.
func TestReplayRecordedHistories(t *testing.T) {
	for _, h := range loadHistories(t) {
		t.Run(h.name, func(t *testing.T) {
			if err := replay(t, h); err != nil {
				t.Fatalf("%s no longer replays: %v\nA change to audit.seal altered the commands a recorded run "+
					"produced. Put it behind workflow.GetVersion and record a new history.", h.name, err)
			}
		})
	}
}

// TestRecordedHistoriesShape: in every recorded run A1 is scheduled before
// A2, A2 is scheduled whatever A1 returned, and no run starts a child or
// continues as new (TICK-438 / TICK-572): the history, not just the code,
// shows the sealer reschedules only through the outbox.
func TestRecordedHistoriesShape(t *testing.T) {
	for _, h := range loadHistories(t) {
		t.Run(h.name, func(t *testing.T) {
			var scheduled []string
			for _, e := range h.h.GetEvents() {
				switch e.GetEventType() {
				case enumspb.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED,
					enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW:
					t.Errorf("event %d is %v", e.GetEventId(), e.GetEventType())
				case enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
					scheduled = append(scheduled, e.GetActivityTaskScheduledEventAttributes().GetActivityType().GetName())
				}
			}
			if want := []string{seal.RescheduleActivity, seal.SealActivity}; !slices.Equal(scheduled, want) {
				t.Errorf("scheduled %v, want %v", scheduled, want)
			}
			if !strings.HasPrefix(h.workflowID, "t:") {
				t.Errorf("workflow id %q has no tenant prefix", h.workflowID)
			}
		})
	}
}
