//go:build recordhistories

package seal_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The audit.seal history recorder (P5-34b): `make seal-record-histories`, a
// developer target, never a CI step. It runs audit.seal behind the whole
// worker guard and the Propagator, as cmd/nava-worker does, on a Temporal dev
// server (testsuite.StartDevServer downloads the temporal CLI once;
// NAVA_TEMPORAL_CLI=<path> uses one you have), with stand-in A1 and A2 under
// the real names: a history records an activity's name and result, never the
// database it ran against. It writes only missing files;
// NAVA_RECORD_OVERWRITE=1 overwrites one main does not have yet.

//nolint:gochecknoglobals // the fixed recording tenant
var recordTenant = uuid.MustParse("5d1c9a7e-3b2f-4e60-8a14-7c9d0e2f4b31")

// sealScenario is one recorded run: what A1 and A2 do.
type sealScenario struct {
	name string
	run  uuid.UUID // o, fixed so a re-record keeps the workflow id
	a1   func(context.Context) (seal.Rescheduled, error)
	a2   func(context.Context) error
}

func nextOutbox(o uuid.UUID) *uuid.UUID {
	n := uuid.NewSHA1(o, []byte("next"))
	return &n
}

func recordScenarios() []sealScenario {
	ok := func(context.Context) error { return nil }
	live := func(o uuid.UUID) func(context.Context) (seal.Rescheduled, error) {
		return func(context.Context) (seal.Rescheduled, error) {
			return seal.Rescheduled{NextOutboxID: nextOutbox(o)}, nil
		}
	}
	stale := func(context.Context) (seal.Rescheduled, error) { return seal.Rescheduled{}, nil }
	runs := []uuid.UUID{
		uuid.MustParse("01a1b3c0-0000-7000-8000-000000000001"),
		uuid.MustParse("01a1b3c0-0000-7000-8000-000000000002"),
		uuid.MustParse("01a1b3c0-0000-7000-8000-000000000003"),
		uuid.MustParse("01a1b3c0-0000-7000-8000-000000000004"),
	}
	return []sealScenario{
		// A live lineage: A1 commits o′, A2 seals.
		{name: "next-run", run: runs[0], a1: live(runs[0]), a2: ok},
		// A stale lineage: A1 returns no o′, A2 still runs.
		{name: "stale", run: runs[1], a1: stale, a2: ok},
		// A1 fails after its commit and is retried; the retry returns no o′
		// and A2 runs (TICK-440 review).
		{name: "a1-retried", run: runs[2], a1: func(ctx context.Context) (seal.Rescheduled, error) {
			if activity.GetInfo(ctx).Attempt == 1 {
				return seal.Rescheduled{}, errors.New("worker lost after commit")
			}
			return seal.Rescheduled{}, nil
		}, a2: ok},
		// A poison batch: A2 fails non-retryably and the run fails, after A1
		// committed o′.
		{name: "poison", run: runs[3], a1: live(runs[3]), a2: func(context.Context) error {
			return temporal.NewNonRetryableApplicationError("seal: entry cannot be canonicalised", seal.ErrTypeUnsealable, nil)
		}},
	}
}

func TestRecordHistories(t *testing.T) {
	overwrite := os.Getenv("NAVA_RECORD_OVERWRITE") == "1"
	if err := os.MkdirAll(historiesDir, 0o750); err != nil {
		t.Fatal(err)
	}
	// No sticky queue: its name has the host name, which the history keeps.
	worker.SetStickyWorkflowCacheSize(0)
	c := startRecordServer(t)
	for _, s := range recordScenarios() {
		t.Run(s.name, func(t *testing.T) {
			writeHistory(t, filepath.Join(historiesDir, s.name+".json"), recordRun(t, c, s), overwrite)
		})
	}
}

func startRecordServer(t *testing.T) client.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	opts := testsuite.DevServerOptions{
		ClientOptions: &client.Options{
			Namespace:          "default",
			Identity:           "nava-record-histories",
			ContextPropagators: []workflow.ContextPropagator{workerguard.Propagator{}},
			DataConverter:      workerguard.DataConverter(),
		},
		LogLevel: "error",
		Stdout:   io.Discard,
		Stderr:   io.Discard,
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

// recordRun runs s on a fresh guarded worker, started as the relay starts a
// seal row, and returns its history.
func recordRun(t *testing.T, c client.Client, s sealScenario) *historypb.History {
	t.Helper()
	g, err := workerguard.New(slog.New(slog.DiscardHandler), sdkmetric.NewMeterProvider())
	if err != nil {
		t.Fatal(err)
	}
	w := worker.New(c, "nava-tq", workerguard.WorkerOptions(worker.Options{}, g))
	seal.RegisterWorkflow(w)
	w.RegisterActivityWithOptions(s.a1, activity.RegisterOptions{Name: seal.RescheduleActivity})
	w.RegisterActivityWithOptions(s.a2, activity.RegisterOptions{Name: seal.SealActivity})
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	ctx, cancel := context.WithTimeout(workerguard.ContextWithTenant(context.Background(), recordTenant), 2*time.Minute)
	defer cancel()
	id := "t:" + recordTenant.String() + ":" + outbox.SealKind + ":" + s.run.String()
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: id, TaskQueue: "nava-tq", WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}, seal.WorkflowType, outbox.Delivery{OutboxID: s.run, KindVersion: 1, Payload: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	_ = run.Get(ctx, nil) // the poison run fails; its history is what is kept
	it := c.GetWorkflowHistory(ctx, id, run.GetRunID(), false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
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
		t.Logf("%s exists; kept", path)
		return
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
