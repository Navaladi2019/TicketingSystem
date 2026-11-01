//go:build recordhistories

package interp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

// The history recorder (P3-23): `make workflow-record-histories`, a developer
// target, never a CI step. It runs nava.interpreter, behind the whole worker
// guard and the Propagator as cmd/nava-worker does, on a Temporal dev server
// (testsuite.StartDevServer, which downloads the temporal CLI once), drives
// every scenario in recordScenarios and writes each run's history to
// historiesDir as protojson.
//
// Committed histories are append-only (TestHistoriesAppendOnly), so the
// recorder never overwrites a file: it writes only the missing ones.
// NAVA_RECORD_OVERWRITE=1 overwrites, for a history that has not shipped to
// main yet.
//
// Environment:
//
//	NAVA_RECORD_OVERWRITE=1   overwrite existing history files
//	NAVA_TEMPORAL_CLI=<path>  use this temporal CLI instead of downloading one

// recordTaskQueue is the interpreter's queue (D-3-11).
const recordTaskQueue = "nava-tq"

// suggestCANAt is the history length past which the continue-as-new
// scenario's server suggests continue-as-new; the default (4096 events) is
// out of reach of a fixture run.
const suggestCANAt = 15

// recordScenario is one fixture run. Its histories are <name>.json, or
// <name>.run<N>.json for each run of a continue-as-new chain.
type recordScenario struct {
	name     string
	workflow uuid.UUID // the id's last part, fixed so a re-record keeps the id
	record   map[string]any
	// signals are sent right after the start: they wait on the signal
	// channel until the workflow reads it.
	signals []recordSignal
	can     bool // record on the server that suggests continue-as-new
}

type recordSignal struct {
	outbox uuid.UUID
	signal string // a key of the step-id map, "signal:<name>"; "" is an id the definition does not have
}

//nolint:gochecknoglobals // the fixed recording scenarios
var (
	recordVersion = uuid.MustParse("6a4f1c3e-2b7d-4e0a-9c55-0d3b8e1f7a21")
	recordRecord  = uuid.MustParse("3c9e7b12-5f4a-4d86-a1e0-7b2c4d6f8e93")
	unknownSignal = uuid.MustParse("9f0e1d2c-3b4a-4596-8778-695a4b3c2d1e")

	recordScenarios = []recordScenario{
		{
			// timer, condition (then), wait_signal on the signal path with a
			// foreign signal and a redelivery dropped, update_record, branch
			// (first case), end.
			name: "signal", workflow: uuid.MustParse("0b1c2d3e-4f50-4617-8829-3a4b5c6d7e81"),
			record: map[string]any{"priority": "high", "team": "a"},
			signals: []recordSignal{
				{outbox: uuid.MustParse("1a2b3c4d-5e6f-4708-9192-a3b4c5d6e7f1"), signal: ""},
				{outbox: uuid.MustParse("1a2b3c4d-5e6f-4708-9192-a3b4c5d6e7f2"), signal: "signal:approve"},
				{outbox: uuid.MustParse("1a2b3c4d-5e6f-4708-9192-a3b4c5d6e7f2"), signal: "signal:approve"},
			},
		},
		{
			// timer, condition (then), wait_signal on the timeout path,
			// update_record, branch (second case), end.
			name: "timeout", workflow: uuid.MustParse("0b1c2d3e-4f50-4617-8829-3a4b5c6d7e82"),
			record: map[string]any{"priority": "high", "team": "b"},
		},
		{
			// timer, condition (else), branch (default), update_record, end.
			name: "else-default", workflow: uuid.MustParse("0b1c2d3e-4f50-4617-8829-3a4b5c6d7e83"),
			record: map[string]any{"priority": "low"},
		},
		{
			// The signal path across continue-as-new: the signal arrives
			// during the timer, is carried to the next run as a pending
			// delivery and applied there.
			name: "continue-as-new", workflow: uuid.MustParse("0b1c2d3e-4f50-4617-8829-3a4b5c6d7e84"),
			record: map[string]any{"priority": "high", "team": "a"},
			signals: []recordSignal{
				{outbox: uuid.MustParse("1a2b3c4d-5e6f-4708-9192-a3b4c5d6e7f4"), signal: "signal:approve"},
			},
			can: true,
		},
	}
)

func TestRecordHistories(t *testing.T) {
	overwrite := os.Getenv("NAVA_RECORD_OVERWRITE") == "1"
	f := loadFixture(t, filepath.Join("testdata", "replay"))
	root := t.TempDir()
	if err := interptest.PutDefinition(root, tenantA, recordVersion, f.def, f.ids); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(historiesDir, 0o750); err != nil {
		t.Fatal(err)
	}

	// No sticky queue: its name is <host name>:<uuid>, which the history
	// records; with no cache the worker polls nava-tq only.
	worker.SetStickyWorkflowCacheSize(0)
	plain := startRecordServer(t, nil)
	can := startRecordServer(t, []string{"--dynamic-config-value", "limit.historyCount.suggestContinueAsNew=" + strconv.Itoa(suggestCANAt)})
	for _, s := range recordScenarios {
		c := plain
		if s.can {
			c = can
		}
		t.Run(s.name, func(t *testing.T) {
			if err := interptest.PutRecord(root, tenantA, recordRecord, s.record); err != nil {
				t.Fatal(err)
			}
			runs := recordScenarioRun(t, c, root, f, s)
			for i, h := range runs {
				name := s.name + ".json"
				if len(runs) > 1 || s.can {
					name = fmt.Sprintf("%s.run%d.json", s.name, i+1)
				}
				writeHistory(t, filepath.Join(historiesDir, name), h, overwrite)
			}
		})
	}
}

// startRecordServer starts a dev server with extra arguments and a worker on
// it, and returns its client.
func startRecordServer(t *testing.T, extra []string) client.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	opts := testsuite.DevServerOptions{
		ClientOptions: &client.Options{
			Namespace: "default",
			// A fixed identity: the default is <pid>@<host name>, which would
			// put the recording host into a committed file.
			Identity:           "nava-record-histories",
			ContextPropagators: []workflow.ContextPropagator{workerguard.Propagator{}},
			DataConverter:      workerguard.DataConverter(),
		},
		ExtraArgs: extra,
		LogLevel:  "error",
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}
	if p := os.Getenv("NAVA_TEMPORAL_CLI"); p != "" {
		opts.ExistingPath = p
	}
	srv, err := testsuite.StartDevServer(ctx, opts)
	if err != nil {
		t.Fatalf("start the Temporal dev server: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop() })
	return srv.Client()
}

// recordScenarioRun runs s on a fresh worker and returns the history of every
// run of its execution, in order.
func recordScenarioRun(t *testing.T, c client.Client, root string, f fixture, s recordScenario) []*historypb.History {
	t.Helper()
	g, err := workerguard.New(slog.New(slog.DiscardHandler), sdkmetric.NewMeterProvider())
	if err != nil {
		t.Fatal(err)
	}
	w := worker.New(c, recordTaskQueue, workerguard.WorkerOptions(worker.Options{}, g))
	interp.RegisterWorkflow(w)
	(&interp.Activities{
		Definitions: interptest.FixtureStore{Root: root},
		Records:     interptest.FileRecords{Root: root},
		Writer:      &interptest.FileWriter{Root: root},
	}).Register(w)
	// CloseWorkflowRef's real activity needs the database; the history holds
	// only its name and no payload either way.
	w.RegisterActivityWithOptions(func(context.Context) error { return nil },
		activity.RegisterOptions{Name: workerguard.CloseWorkflowRefActivity})
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	ctx, cancel := context.WithTimeout(workerguard.ContextWithTenant(context.Background(), tenantA), 2*time.Minute)
	defer cancel()
	id := "t:" + tenantA.String() + ":interp:" + s.workflow.String()
	start := startDelivery(t, uuid.NewSHA1(s.workflow, []byte("start")))
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: id, TaskQueue: recordTaskQueue}, interp.WorkflowType, start)
	if err != nil {
		t.Fatal(err)
	}
	// Get follows the continue-as-new chain and moves GetRunID with it, so
	// keep the first run's id now.
	firstRun := run.GetRunID()
	for _, sig := range s.signals {
		signal := unknownSignal
		if sig.signal != "" {
			signal = f.id(t, sig.signal)
		}
		b, err := json.Marshal(map[string]uuid.UUID{"signal": signal})
		if err != nil {
			t.Fatal(err)
		}
		if err := c.SignalWorkflow(ctx, id, "", interp.SignalName, outbox.Delivery{OutboxID: sig.outbox, KindVersion: 1, Payload: b}); err != nil {
			t.Fatal(err)
		}
	}
	if err := run.Get(ctx, nil); err != nil {
		t.Fatalf("run: %v", err)
	}

	var out []*historypb.History
	for runID := firstRun; runID != ""; {
		h := fetchHistory(ctx, t, c, id, runID)
		out = append(out, h)
		runID = ""
		last := h.GetEvents()[len(h.GetEvents())-1]
		switch last.GetEventType() {
		case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW:
			runID = last.GetWorkflowExecutionContinuedAsNewEventAttributes().GetNewExecutionRunId()
		case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
		default:
			t.Fatalf("run %d ended with %v, want completed or continued-as-new", len(out), last.GetEventType())
		}
	}
	if s.can && len(out) < 2 {
		t.Fatalf("%s: %d run, want a continue-as-new; lower suggestCANAt", s.name, len(out))
	}
	if !s.can && len(out) != 1 {
		t.Fatalf("%s: %d runs, want 1", s.name, len(out))
	}
	return out
}

func startDelivery(t *testing.T, outboxID uuid.UUID) outbox.Delivery {
	t.Helper()
	b, err := json.Marshal(map[string]uuid.UUID{"definition_version_id": recordVersion, "record_id": recordRecord})
	if err != nil {
		t.Fatal(err)
	}
	return outbox.Delivery{OutboxID: outboxID, KindVersion: 1, Payload: b}
}

func fetchHistory(ctx context.Context, t *testing.T, c client.Client, id, runID string) *historypb.History {
	t.Helper()
	it := c.GetWorkflowHistory(ctx, id, runID, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	h := &historypb.History{}
	for it.HasNext() {
		e, err := it.Next()
		if err != nil {
			t.Fatal(err)
		}
		h.Events = append(h.Events, e)
	}
	return h
}

func writeHistory(t *testing.T, path string, h *historypb.History, overwrite bool) {
	t.Helper()
	if _, err := os.Stat(path); err == nil && !overwrite {
		t.Logf("%s exists; kept (append-only; NAVA_RECORD_OVERWRITE=1 overwrites one not on main yet)", path)
		return
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	b, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%d events)", path, len(h.GetEvents()))
}
