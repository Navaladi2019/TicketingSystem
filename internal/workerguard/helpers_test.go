package workerguard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

//nolint:gochecknoglobals // fixed test tenants
var (
	tenantA = uuid.MustParse("0192f3a0-0000-7000-8000-00000000000a")
	tenantB = uuid.MustParse("0192f3a0-0000-7000-8000-00000000000b")
)

// idOf is a relay-shaped workflow id in tenant: t:<tenant>:<kind>:<outbox id>.
func idOf(tenant uuid.UUID) string {
	return "t:" + tenant.String() + ":test:" + uuid.NewString()
}

// headerFor is the nava-tenant header a correct client sends for tenant.
func headerFor(t *testing.T, tenant uuid.UUID) *commonpb.Header {
	t.Helper()
	return &commonpb.Header{Fields: map[string]*commonpb.Payload{TenantHeader: stringPayload(t, tenant.String())}}
}

func stringPayload(t *testing.T, s string) *commonpb.Payload {
	t.Helper()
	p, err := converter.GetDefaultDataConverter().ToPayload(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// override sets the nava-tenant header a call arrives with, standing in for
// the client that sent it: the test environment sends signals and updates
// with no header, and activities get the Propagator's. A nil payload removes
// the header.
type override struct {
	set bool
	p   *commonpb.Payload
}

// transport is a test-only worker interceptor placed before the Guard. It
// rewrites the header of the inbound calls whose override is set.
type transport struct {
	interceptor.WorkerInterceptorBase
	mu                       sync.Mutex
	signal, update, activity override
	// seen holds the nava-tenant header each workflow start and activity
	// arrived with, by workflow id or "activity:<type>", as the command
	// carried it.
	seen map[string]*commonpb.Payload
}

func (tr *transport) record(key string, fields map[string]*commonpb.Payload) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.seen == nil {
		tr.seen = map[string]*commonpb.Payload{}
	}
	tr.seen[key] = fields[TenantHeader]
}

func (tr *transport) header(key string) *commonpb.Payload {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.seen[key]
}

func (tr *transport) setSignal(p *commonpb.Payload) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.signal = override{set: true, p: p}
}

func (tr *transport) setUpdate(p *commonpb.Payload) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.update = override{set: true, p: p}
}

func (tr *transport) setActivity(p *commonpb.Payload) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.activity = override{set: true, p: p}
}

func (tr *transport) get(which *override) override {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return *which
}

func apply(fields map[string]*commonpb.Payload, o override) {
	if !o.set || fields == nil {
		return
	}
	if o.p == nil {
		delete(fields, TenantHeader)
		return
	}
	fields[TenantHeader] = o.p
}

func (tr *transport) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &transportWorkflow{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, tr: tr}
}

func (tr *transport) InterceptActivity(_ context.Context, next interceptor.ActivityInboundInterceptor) interceptor.ActivityInboundInterceptor {
	return &transportActivity{ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next}, tr: tr}
}

type transportWorkflow struct {
	interceptor.WorkflowInboundInterceptorBase
	tr *transport
}

func (w *transportWorkflow) ExecuteWorkflow(ctx workflow.Context, in *interceptor.ExecuteWorkflowInput) (any, error) {
	w.tr.record(workflow.GetInfo(ctx).WorkflowExecution.ID, interceptor.WorkflowHeader(ctx))
	return w.Next.ExecuteWorkflow(ctx, in)
}

func (w *transportWorkflow) HandleSignal(ctx workflow.Context, in *interceptor.HandleSignalInput) error {
	apply(interceptor.WorkflowHeader(ctx), w.tr.get(&w.tr.signal))
	return w.Next.HandleSignal(ctx, in)
}

func (w *transportWorkflow) ValidateUpdate(ctx workflow.Context, in *interceptor.UpdateInput) error {
	apply(interceptor.WorkflowHeader(ctx), w.tr.get(&w.tr.update))
	return w.Next.ValidateUpdate(ctx, in)
}

type transportActivity struct {
	interceptor.ActivityInboundInterceptorBase
	tr *transport
}

func (a *transportActivity) ExecuteActivity(ctx context.Context, in *interceptor.ExecuteActivityInput) (any, error) {
	a.tr.record("activity:"+activity.GetInfo(ctx).ActivityType.Name, interceptor.Header(ctx))
	apply(interceptor.Header(ctx), a.tr.get(&a.tr.activity))
	return a.Next.ExecuteActivity(ctx, in)
}

// testDeadlockDetectionTimeout replaces the SDK's 1 s default in the test
// environment only (TICK-184). The detector measures wall-clock time between
// yields, so on a loaded CI host a workflow task that needs well under a
// millisecond of CPU can still be descheduled for over 1 s and fail with
// TMPRL1101. Ten seconds still catches a workflow that really never yields.
// Production workers keep the SDK default.
const testDeadlockDetectionTimeout = 10 * time.Second

// harness is a test environment with the Propagator, the transport and the
// Guard, a metric reader for DecisionsMetric and a buffer for the alert log.
type harness struct {
	t      *testing.T
	env    *testsuite.TestWorkflowEnvironment
	tr     *transport
	reader *sdkmetric.ManualReader
	logs   *syncBuffer
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

type harnessOption func(*harnessConfig)

type harnessConfig struct {
	noPropagator bool
	before       []interceptor.WorkerInterceptor
}

func withoutPropagator() harnessOption { return func(c *harnessConfig) { c.noPropagator = true } }

// withOuter adds interceptors before the transport and the Guard (tracing).
func withOuter(i ...interceptor.WorkerInterceptor) harnessOption {
	return func(c *harnessConfig) { c.before = append(c.before, i...) }
}

// newHarness starts a workflow id in tenant with a correct start header.
func newHarness(t *testing.T, workflowID string, opts ...harnessOption) *harness {
	t.Helper()
	var cfg harnessConfig
	for _, o := range opts {
		o(&cfg)
	}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	reader := sdkmetric.NewManualReader()
	logs := &syncBuffer{}
	g, err := New(slog.New(slog.NewJSONHandler(logs, nil)), sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	tr := &transport{}
	interceptors := append(append([]interceptor.WorkerInterceptor(nil), cfg.before...), tr)
	env.SetWorkerOptions(WorkerOptions(worker.Options{
		Interceptors:             interceptors,
		DeadlockDetectionTimeout: testDeadlockDetectionTimeout,
	}, g))
	if !cfg.noPropagator {
		env.SetContextPropagators([]workflow.ContextPropagator{Propagator{}})
	}
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: workflowID})
	if tenant, err := TenantOf(workflowID); err == nil {
		env.SetHeader(headerFor(t, tenant))
	}
	return &harness{t: t, env: env, tr: tr, reader: reader, logs: logs}
}

// decisions returns DecisionsMetric by "guard/outcome", and fails if a data
// point has any attribute but guard and outcome.
func (h *harness) decisions() map[string]int64 {
	h.t.Helper()
	var rm metricdata.ResourceMetrics
	if err := h.reader.Collect(context.Background(), &rm); err != nil {
		h.t.Fatal(err)
	}
	out := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != DecisionsMetric {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				h.t.Fatalf("%s is %T, want an int64 sum", m.Name, m.Data)
			}
			for _, dp := range sum.DataPoints {
				var guard, outcome string
				for _, kv := range dp.Attributes.ToSlice() {
					switch kv.Key {
					case "guard":
						guard = kv.Value.AsString()
					case "outcome":
						outcome = kv.Value.AsString()
					default:
						h.t.Errorf("%s has attribute %q; only guard and outcome are allowed", DecisionsMetric, kv.Key)
					}
				}
				out[guard+"/"+outcome] += dp.Value
			}
		}
	}
	return out
}

// alerts returns the guard's alert log lines.
func (h *harness) alerts() []map[string]any {
	h.t.Helper()
	var out []map[string]any
	for line := range strings.Lines(h.logs.String()) {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			h.t.Fatalf("log line %q: %v", line, err)
		}
		if rec["alert"] == true {
			out = append(out, rec)
		}
	}
	return out
}

// wantDenyAlert fails unless exactly one alert names guard and outcome, with
// the workflow id.
func (h *harness) wantDenyAlert(guard, outcome, workflowID string) {
	h.t.Helper()
	var n int
	for _, a := range h.alerts() {
		if a["guard"] == guard && a["outcome"] == outcome {
			n++
			if a["workflow_id"] != workflowID {
				h.t.Errorf("alert %v: workflow_id = %v, want %s", a, a["workflow_id"], workflowID)
			}
			if a["level"] != "ERROR" {
				h.t.Errorf("alert %v: level %v, want ERROR", a, a["level"])
			}
		}
	}
	if n != 1 {
		h.t.Errorf("%d alerts for %s/%s, want 1; log:\n%s", n, guard, outcome, h.logs.String())
	}
	if got := h.decisions()[guard+"/"+outcome]; got != 1 {
		h.t.Errorf("%s{guard=%s,outcome=%s} = %d, want 1", DecisionsMetric, guard, outcome, got)
	}
}

// wantNoAlerts fails if any deny was logged.
func (h *harness) wantNoAlerts() {
	h.t.Helper()
	if a := h.alerts(); len(a) > 0 {
		h.t.Errorf("unexpected alerts: %v", a)
	}
}

// wantGuardError fails unless err is (or wraps) the guard's non-retryable
// ApplicationError for guard and outcome.
func wantGuardError(t *testing.T, err error, guard, outcome string) {
	t.Helper()
	var app *temporal.ApplicationError
	if !errors.As(err, &app) {
		t.Fatalf("error %v, want an ApplicationError", err)
	}
	if app.Type() != ErrTypeTenantGuard || !app.NonRetryable() {
		t.Errorf("error type %q non-retryable %v, want %q non-retryable", app.Type(), app.NonRetryable(), ErrTypeTenantGuard)
	}
	if want := "tenant guard denied " + guard + ": " + outcome; !strings.Contains(app.Error(), want) {
		t.Errorf("error %q, want it to contain %q", app.Error(), want)
	}
}
