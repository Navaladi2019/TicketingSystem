package interp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

//nolint:gochecknoglobals // fixed test tenants and start time
var (
	tenantA = uuid.MustParse("0192f3a0-0000-7000-8000-00000000000a")
	tenantB = uuid.MustParse("0192f3a0-0000-7000-8000-00000000000b")
	t0      = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
)

// testDeadlockDetectionTimeout: as in internal/workerguard (TICK-184).
const testDeadlockDetectionTimeout = 10 * time.Second

// fixture is a definition file and its committed step-id map.
type fixture struct {
	def, ids []byte
	stepIDs  map[string]uuid.UUID
}

func loadFixture(t *testing.T, path string) fixture {
	t.Helper()
	d, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	ids, err := os.ReadFile(path + ".ids.json")
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{def: d, ids: ids}
	if err := json.Unmarshal(ids, &f.stepIDs); err != nil {
		t.Fatal(err)
	}
	return f
}

// fixtureV1 is P3-20's fixture-v1, read from the def package's testdata.
func fixtureV1(t *testing.T) fixture {
	t.Helper()
	return loadFixture(t, filepath.Join("..", "def", "testdata", "fixture-v1"))
}

func (f fixture) id(t *testing.T, key string) uuid.UUID {
	t.Helper()
	id, ok := f.stepIDs[key]
	if !ok {
		t.Fatalf("fixture has no id for %q", key)
	}
	return id
}

// recordingDC is the default converter, keeping every payload it encodes:
// everything the history records goes through it (arguments, results,
// signal and continue-as-new inputs).
type recordingDC struct {
	converter.DataConverter
	mu   sync.Mutex
	data [][]byte
}

func (r *recordingDC) keep(p *commonpb.Payload) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = append(r.data, bytes.Clone(p.GetData()))
}

func (r *recordingDC) ToPayload(v any) (*commonpb.Payload, error) {
	p, err := r.DataConverter.ToPayload(v)
	if err == nil {
		r.keep(p)
	}
	return p, err
}

func (r *recordingDC) ToPayloads(v ...any) (*commonpb.Payloads, error) {
	ps, err := r.DataConverter.ToPayloads(v...)
	if err == nil {
		for _, p := range ps.GetPayloads() {
			r.keep(p)
		}
	}
	return ps, err
}

func (r *recordingDC) recorded() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]byte(nil), r.data...)
}

// signalHeader stands in for the relay's client: it puts the tenant's
// nava-tenant header on every signal, which the test environment sends
// without one (the guard drops a signal with no header).
type signalHeader struct {
	interceptor.WorkerInterceptorBase
	tenant uuid.UUID
}

func (s *signalHeader) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &signalHeaderInbound{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, tenant: s.tenant}
}

type signalHeaderInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	tenant uuid.UUID
}

func (s *signalHeaderInbound) HandleSignal(ctx workflow.Context, in *interceptor.HandleSignalInput) error {
	if fields := interceptor.WorkflowHeader(ctx); fields != nil {
		p, err := converter.GetDefaultDataConverter().ToPayload(s.tenant.String())
		if err != nil {
			return err
		}
		fields[workerguard.TenantHeader] = p
	}
	return s.Next.HandleSignal(ctx, in)
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// scheduled is one activity the workflow scheduled, with the options it
// scheduled it with.
type scheduled struct {
	activity string
	options  workflow.ActivityOptions
}

// optionsRecorder records the ActivityOptions of every activity the workflow
// schedules, as the workflow code set them. The SDK test environment fills in
// its own retry default (10 attempts) when none is set, so only the options
// the workflow passed show whether it set ActivityRetryPolicy.
type optionsRecorder struct {
	interceptor.WorkerInterceptorBase
	h *harness
}

func (o *optionsRecorder) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &optionsRecorderInbound{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, h: o.h}
}

type optionsRecorderInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	h *harness
}

func (i *optionsRecorderInbound) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return i.Next.Init(&optionsRecorderOutbound{WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{Next: outbound}, h: i.h})
}

type optionsRecorderOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	h *harness
}

func (o *optionsRecorderOutbound) ExecuteActivity(ctx workflow.Context, activityType string, args ...any) workflow.Future {
	o.h.mu.Lock()
	o.h.opts = append(o.h.opts, scheduled{activity: activityType, options: workflow.GetActivityOptions(ctx)})
	o.h.mu.Unlock()
	return o.Next.ExecuteActivity(ctx, activityType, args...)
}

// scheduledOptions returns every scheduled activity with its options.
func (h *harness) scheduledOptions() []scheduled {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]scheduled(nil), h.opts...)
}

// harness is one test environment for the interpreter: the whole worker
// guard, the Propagator, the file-backed fixtures under root, a recording
// converter, and listeners for NextStep refs and timers.
type harness struct {
	t          *testing.T
	env        *testsuite.TestWorkflowEnvironment
	root       string
	tenant     uuid.UUID
	workflowID string
	writer     *interptest.FileWriter
	dc         *recordingDC
	logs       *syncBuffer

	mu     sync.Mutex
	opts   []scheduled
	refs   []interp.StepRef
	timers []time.Duration
	closes int
}

type harnessConfig struct {
	root       string
	workflowID string
	records    func(root string) interp.RecordReader
	noGuard    bool
}

type harnessOption func(*harnessConfig)

// sameStore reuses another harness's fixture directory and workflow id, as
// the next run of a continue-as-new does.
func sameStore(h *harness) harnessOption {
	return func(c *harnessConfig) { c.root, c.workflowID = h.root, h.workflowID }
}

// withRecords replaces the file-backed RecordReader over the harness root.
func withRecords(f func(root string) interp.RecordReader) harnessOption {
	return func(c *harnessConfig) { c.records = f }
}

func withoutGuard() harnessOption { return func(c *harnessConfig) { c.noGuard = true } }

func newHarness(t *testing.T, tenant uuid.UUID, opts ...harnessOption) *harness {
	t.Helper()
	cfg := harnessConfig{root: t.TempDir(), workflowID: "t:" + tenant.String() + ":interp:" + uuid.NewString()}
	for _, o := range opts {
		o(&cfg)
	}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	h := &harness{
		t: t, env: env, root: cfg.root, tenant: tenant, workflowID: cfg.workflowID,
		writer: &interptest.FileWriter{Root: cfg.root},
		dc:     &recordingDC{DataConverter: converter.GetDefaultDataConverter()},
		logs:   &syncBuffer{},
	}
	opts2 := worker.Options{DeadlockDetectionTimeout: testDeadlockDetectionTimeout}
	opts2.Interceptors = []interceptor.WorkerInterceptor{&optionsRecorder{h: h}}
	if !cfg.noGuard {
		g, err := workerguard.New(slog.New(slog.NewJSONHandler(h.logs, nil)), sdkmetric.NewMeterProvider())
		if err != nil {
			t.Fatal(err)
		}
		opts2.Interceptors = append(opts2.Interceptors, &signalHeader{tenant: tenant})
		opts2 = workerguard.WorkerOptions(opts2, g)
	}
	env.SetWorkerOptions(opts2)
	env.SetContextPropagators([]workflow.ContextPropagator{workerguard.Propagator{}})
	env.SetDataConverter(h.dc)
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: cfg.workflowID})
	p, err := converter.GetDefaultDataConverter().ToPayload(tenant.String())
	if err != nil {
		t.Fatal(err)
	}
	env.SetHeader(&commonpb.Header{Fields: map[string]*commonpb.Payload{workerguard.TenantHeader: p}})
	env.SetStartTime(t0)

	acts := &interp.Activities{
		Definitions: interptest.FixtureStore{Root: cfg.root},
		Records:     interptest.FileRecords{Root: cfg.root},
		Writer:      h.writer,
	}
	if cfg.records != nil {
		acts.Records = cfg.records(cfg.root)
	}
	interp.RegisterWorkflow(env)
	acts.Register(env)
	env.RegisterActivityWithOptions(func(context.Context) error {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.closes++
		return nil
	}, activity.RegisterOptions{Name: workerguard.CloseWorkflowRefActivity})
	env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, args converter.EncodedValues) {
		if info.ActivityType.Name != interp.NextStepActivity {
			return
		}
		var ref interp.StepRef
		if err := args.Get(&ref); err != nil {
			t.Errorf("NextStep args: %v", err)
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		h.refs = append(h.refs, ref)
	})
	env.SetOnTimerScheduledListener(func(_ string, d time.Duration) {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.timers = append(h.timers, d)
	})
	return h
}

// install puts f under tenant as version, and the record with data.
func (h *harness) install(tenant, version uuid.UUID, f fixture) {
	h.t.Helper()
	if err := interptest.PutDefinition(h.root, tenant, version, f.def, f.ids); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) putRecord(tenant, record uuid.UUID, data map[string]any) {
	h.t.Helper()
	if err := interptest.PutRecord(h.root, tenant, record, data); err != nil {
		h.t.Fatal(err)
	}
}

// startInput is the relay's start delivery for version and record.
func startInput(t *testing.T, version, record uuid.UUID) outbox.Delivery {
	t.Helper()
	b, err := json.Marshal(map[string]uuid.UUID{"definition_version_id": version, "record_id": record})
	if err != nil {
		t.Fatal(err)
	}
	return outbox.Delivery{OutboxID: uuid.New(), KindVersion: 1, Payload: b}
}

// start runs the workflow to completion with the relay's start input.
func (h *harness) start(version, record uuid.UUID) {
	h.env.ExecuteWorkflow(interp.WorkflowType, startInput(h.t, version, record))
}

// signalAt delivers the relay's signal for signal id with outboxID after d.
func (h *harness) signalAt(d time.Duration, outboxID, signal uuid.UUID) {
	b, err := json.Marshal(map[string]uuid.UUID{"signal": signal})
	if err != nil {
		h.t.Fatal(err)
	}
	del := outbox.Delivery{OutboxID: outboxID, KindVersion: 1, Payload: b}
	h.env.RegisterDelayedCallback(func() { h.env.SignalWorkflow(interp.SignalName, del) }, d)
}

func (h *harness) nextRefs() []interp.StepRef {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]interp.StepRef(nil), h.refs...)
}

func (h *harness) timerDurations() []time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]time.Duration(nil), h.timers...)
}

func (h *harness) closeCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closes
}

func (h *harness) calls() []interptest.Call {
	h.t.Helper()
	c, err := h.writer.Calls()
	if err != nil {
		h.t.Fatal(err)
	}
	return c
}

func (h *harness) elapsed() time.Duration { return h.env.Now().Sub(t0) }

func (h *harness) wantCompleted() {
	h.t.Helper()
	if !h.env.IsWorkflowCompleted() {
		h.t.Fatal("workflow did not complete")
	}
	if err := h.env.GetWorkflowError(); err != nil {
		h.t.Fatalf("workflow error: %v", err)
	}
	if n := h.closeCount(); n != 1 {
		h.t.Errorf("CloseWorkflowRef ran %d times, want 1", n)
	}
	h.wantNoDenies()
}

// wantNoDenies fails if the guard logged a deny alert.
func (h *harness) wantNoDenies() {
	h.t.Helper()
	for line := range strings.Lines(h.logs.String()) {
		if strings.Contains(line, `"alert":true`) {
			h.t.Errorf("guard alert: %s", line)
		}
	}
}

// continued returns the input the run continued as new with.
func (h *harness) continued() workerguard.RunInput {
	h.t.Helper()
	err := h.env.GetWorkflowError()
	var can *workflow.ContinueAsNewError
	if !errors.As(err, &can) {
		h.t.Fatalf("workflow error %v, want continue-as-new", err)
	}
	var in workerguard.RunInput
	if err := converter.GetDefaultDataConverter().FromPayloads(can.Input, &in); err != nil {
		h.t.Fatal(err)
	}
	return in
}

// wantRejected fails unless err is (or wraps) the non-retryable rejection r.
func wantRejected(t *testing.T, err error, r workerguard.Rejection) {
	t.Helper()
	var app *temporal.ApplicationError
	if !errors.As(err, &app) {
		t.Fatalf("error %v, want an ApplicationError", err)
	}
	if app.Type() != workerguard.ErrTypeWorkflowRejected || !app.NonRetryable() {
		t.Errorf("error type %q non-retryable %v, want %q non-retryable", app.Type(), app.NonRetryable(), workerguard.ErrTypeWorkflowRejected)
	}
	if want := "workflow rejected: " + string(r); app.Message() != want {
		t.Errorf("message %q, want %q", app.Message(), want)
	}
}

// recordedText is every payload the run encoded plus its failure, as the
// history would hold them.
func (h *harness) recordedText() string {
	h.t.Helper()
	var b strings.Builder
	for _, d := range h.dc.recorded() {
		b.Write(d)
		b.WriteByte('\n')
	}
	if err := h.env.GetWorkflowError(); err != nil {
		f := temporal.GetDefaultFailureConverter().ErrorToFailure(err)
		j, mErr := protojson.Marshal(f)
		if mErr != nil {
			h.t.Fatal(mErr)
		}
		b.Write(j)
	}
	return b.String()
}
