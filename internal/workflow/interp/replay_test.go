package interp_test

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// The replay gate (P3-23, Epic 09 AC2, PLAN-003 D-3-17): every history
// recorded under historiesDir by `make workflow-record-histories` replays
// against today's nava.interpreter, behind the worker guard and the
// Propagator as cmd/nava-worker runs it. A change that alters the commands a
// recorded run produced fails here with a non-determinism error. An
// incompatible change ships behind workflow.GetVersion and adds a new history
// (docs/quality/ci-pipeline.md, "Changing nava.interpreter").
//
// It is an ordinary unit test: the replayer needs no server.

// historiesDir holds the recorded histories, one protojson file per run.
var historiesDir = filepath.Join("..", "testdata", "histories") //nolint:gochecknoglobals // a fixed path

// wantHistories is the named set the gate requires. Together the histories
// exercise every step type (timer, condition both ways, branch on both cases
// and the default, wait_signal, update_record, end), the signal path (with a
// foreign signal and a redelivery dropped), the timeout path and a
// continue-as-new that carries a pending signal into the next run. A new
// history is added here and recorded; none is ever removed.
//
//nolint:gochecknoglobals // the fixed set
var wantHistories = []string{
	"continue-as-new.run1.json",
	"continue-as-new.run2.json",
	"continue-as-new.run3.json",
	"continue-as-new.run4.json",
	"continue-as-new.run5.json",
	"else-default.json",
	"signal.json",
	"timeout.json",
}

// history is one recorded run.
type history struct {
	name       string
	workflowID string
	h          *historypb.History
}

// checkHistorySet fails unless dir holds every name in want. A file not in
// want is reported too: a history nobody lists would be replayed without
// being required, and its deletion would go unnoticed.
func checkHistorySet(dir string, want []string) ([]string, error) {
	got, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(got))
	for _, p := range got {
		names = append(names, filepath.Base(p))
	}
	var problems []string
	if len(names) < len(want) {
		problems = append(problems, "the directory holds "+strconv.Itoa(len(names))+" histories, want at least "+strconv.Itoa(len(want)))
	}
	for _, w := range want {
		if !slices.Contains(names, w) {
			problems = append(problems, w+" is missing (histories are append-only: restore it)")
		}
	}
	for _, n := range names {
		if !slices.Contains(want, n) {
			problems = append(problems, n+" is not in wantHistories (list it so the gate requires it)")
		}
	}
	if len(problems) > 0 {
		return nil, errors.New("histories in " + dir + ":\n" + strings.Join(problems, "\n"))
	}
	return names, nil
}

// loadHistories reads every history in wantHistories.
func loadHistories(t *testing.T) []history {
	t.Helper()
	names, err := checkHistorySet(historiesDir, wantHistories)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]history, 0, len(names))
	for _, n := range names {
		out = append(out, readHistory(t, filepath.Join(historiesDir, n)))
	}
	return out
}

func readHistory(t *testing.T, path string) history {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := client.HistoryFromJSON(bytes.NewReader(b), client.HistoryJSONOptions{})
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if len(h.GetEvents()) == 0 {
		t.Fatalf("%s: no events", path)
	}
	id := h.GetEvents()[0].GetWorkflowExecutionStartedEventAttributes().GetWorkflowId()
	if id == "" {
		t.Fatalf("%s: the first event is not a start with a workflow id", path)
	}
	return history{name: filepath.Base(path), workflowID: id, h: h}
}

// replay replays h against nava.interpreter behind the worker guard, with
// extra worker interceptors in front of it (a test's deliberate change).
func replay(t *testing.T, h history, extra ...interceptor.WorkerInterceptor) error {
	t.Helper()
	g, err := workerguard.New(slog.New(slog.DiscardHandler), sdkmetric.NewMeterProvider())
	if err != nil {
		t.Fatal(err)
	}
	r, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{
		Interceptors:       append(slices.Clone(extra), g),
		ContextPropagators: []workflow.ContextPropagator{workerguard.Propagator{}},
		DataConverter:      workerguard.DataConverter(),
	})
	if err != nil {
		t.Fatal(err)
	}
	interp.RegisterWorkflow(r)
	return r.ReplayWorkflowHistoryWithOptions(nil, h.h, worker.ReplayWorkflowHistoryOptions{
		OriginalExecution: workflow.Execution{ID: h.workflowID, RunID: "replay"},
	})
}

// TestReplayRecordedHistories is the gate.
func TestReplayRecordedHistories(t *testing.T) {
	for _, h := range loadHistories(t) {
		t.Run(h.name, func(t *testing.T) {
			if err := replay(t, h); err != nil {
				t.Fatalf("%s no longer replays: %v\n"+
					"A change to nava.interpreter altered the commands a recorded run produced. "+
					"Put it behind workflow.GetVersion and record a new history "+
					"(docs/quality/ci-pipeline.md, \"Changing nava.interpreter\").", h.name, err)
			}
		})
	}
}

// TestHistorySetRequired: the gate fails on a missing named history, an
// empty directory and an unlisted file; an empty glob is never green.
func TestHistorySetRequired(t *testing.T) {
	want := []string{"a.json", "b.json"}
	put := func(dir string, names ...string) {
		for _, n := range names {
			if err := os.WriteFile(filepath.Join(dir, n), []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	cases := []struct {
		name  string
		files []string
		wants []string // substrings of the error; nil means no error
	}{
		{name: "complete", files: []string{"a.json", "b.json"}},
		{name: "empty", wants: []string{"holds 0 histories, want at least 2", "a.json is missing", "b.json is missing"}},
		{name: "one missing", files: []string{"a.json", "c.json"}, wants: []string{"b.json is missing", "c.json is not in wantHistories"}},
		{name: "unlisted", files: []string{"a.json", "b.json", "c.json"}, wants: []string{"c.json is not in wantHistories"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			put(dir, c.files...)
			_, err := checkHistorySet(dir, want)
			if c.wants == nil {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want an error containing %q, got none", c.wants)
			}
			for _, w := range c.wants {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
		})
	}
	t.Run("the committed set", func(t *testing.T) {
		if len(wantHistories) == 0 {
			t.Fatal("wantHistories is empty")
		}
	})
}

// extraActivityBeforeTimer is a deliberate non-deterministic change to the
// interpreter: one more NextStep before every timer. With versioned set, the
// change is behind workflow.GetVersion, as an incompatible change must ship.
type extraActivityBeforeTimer struct {
	interceptor.WorkerInterceptorBase
	versioned bool
}

// replayGateChangeID is the GetVersion change id of the deliberate change.
const replayGateChangeID = "replay-gate-extra-activity"

func (e *extraActivityBeforeTimer) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &extraInbound{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, versioned: e.versioned}
}

type extraInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	versioned bool
}

func (i *extraInbound) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return i.Next.Init(&extraOutbound{WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{Next: outbound}, versioned: i.versioned})
}

type extraOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	versioned bool
}

func (o *extraOutbound) NewTimer(ctx workflow.Context, d time.Duration) workflow.Future {
	extra := true
	if o.versioned {
		extra = workflow.GetVersion(ctx, replayGateChangeID, workflow.DefaultVersion, 1) == 1
	}
	if extra {
		actx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		// workflow.ExecuteActivity, not o.ExecuteActivity: it gives the call
		// the header context the outbound chain needs, as workflow code does.
		_ = workflow.ExecuteActivity(actx, interp.NextStepActivity, interp.StepRef{}).Get(actx, nil)
	}
	return o.Next.NewTimer(ctx, d)
}

// wantNonDeterminism fails unless err is the replayer's non-determinism error.
func wantNonDeterminism(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s replayed green with an extra activity before the timer; the gate would miss a breaking change", name)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "nondetermin") {
		t.Fatalf("%s: error %v, want a non-determinism error", name, err)
	}
}

// TestReplayCatchesNonDeterminism: an extra activity before the timer, a
// change in command order, fails replay of every recorded history that
// starts a timer (a timer step, or a wait_signal's timeout).
func TestReplayCatchesNonDeterminism(t *testing.T) {
	timed := 0
	t.Cleanup(func() {
		if timed == 0 {
			t.Error("no recorded history starts a timer; the check proved nothing")
		}
	})
	for _, h := range loadHistories(t) {
		if !hasTimer(h.h) {
			continue
		}
		timed++
		t.Run(h.name, func(t *testing.T) {
			wantNonDeterminism(t, h.name, replay(t, h, &extraActivityBeforeTimer{}))
		})
	}
}

// TestReplayGetVersionKeepsHistoriesGreen: the same change behind
// workflow.GetVersion replays every recorded history green, because a run
// recorded before it takes DefaultVersion.
func TestReplayGetVersionKeepsHistoriesGreen(t *testing.T) {
	for _, h := range loadHistories(t) {
		t.Run(h.name, func(t *testing.T) {
			if err := replay(t, h, &extraActivityBeforeTimer{versioned: true}); err != nil {
				t.Fatalf("%s: the GetVersion-wrapped change failed replay: %v", h.name, err)
			}
		})
	}
}

// hasTimer reports whether the run starts a timer.
func hasTimer(h *historypb.History) bool {
	for _, e := range h.GetEvents() {
		if e.GetTimerStartedEventAttributes() != nil {
			return true
		}
	}
	return false
}
