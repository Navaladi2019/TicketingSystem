package relay

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/grpc"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// frontend is an in-process Temporal frontend for the dispatcher's tests. It
// answers the real SDK client over gRPC with the server semantics the relay
// depends on: one execution per workflow id (REJECT_DUPLICATE answers
// AlreadyStarted, open or closed), and NotFound for a signal to a missing or
// completed workflow. Errors and delays can be injected per call.
type frontend struct {
	workflowservice.UnimplementedWorkflowServiceServer

	mu         sync.Mutex
	executions map[string]*execution // by workflow id
	starts     []*workflowservice.StartWorkflowExecutionRequest
	signals    []*workflowservice.SignalWorkflowExecutionRequest
	// fail, when set, answers a call before the semantics above; a nil
	// return carries on.
	fail  func(method string, workflowID string) error
	delay func(method string) time.Duration
}

type execution struct {
	runID     string
	kind      string
	completed bool
	// signals are the outbox ids of the signals received, in order.
	signals []uuid.UUID
}

func (f *frontend) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{}, nil
}

func (f *frontend) inject(ctx context.Context, method, workflowID string) error {
	f.mu.Lock()
	fail, delay := f.fail, f.delay
	f.mu.Unlock()
	if delay != nil {
		select {
		case <-time.After(delay(method)):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if fail != nil {
		return fail(method, workflowID)
	}
	return nil
}

// grpcErr carries err's status details over the wire, as the real frontend
// does, so the SDK rebuilds the same serviceerror type.
func grpcErr(err error) error {
	if err == nil {
		return nil
	}
	return serviceerror.ToStatus(err).Err()
}

func (f *frontend) StartWorkflowExecution(ctx context.Context, req *workflowservice.StartWorkflowExecutionRequest) (*workflowservice.StartWorkflowExecutionResponse, error) {
	resp, err := f.start(ctx, req)
	return resp, grpcErr(err)
}

func (f *frontend) SignalWorkflowExecution(ctx context.Context, req *workflowservice.SignalWorkflowExecutionRequest) (*workflowservice.SignalWorkflowExecutionResponse, error) {
	resp, err := f.signal(ctx, req)
	return resp, grpcErr(err)
}

func (f *frontend) start(ctx context.Context, req *workflowservice.StartWorkflowExecutionRequest) (*workflowservice.StartWorkflowExecutionResponse, error) {
	f.mu.Lock()
	f.starts = append(f.starts, req)
	f.mu.Unlock()
	if err := f.inject(ctx, "start", req.GetWorkflowId()); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if e, ok := f.executions[req.GetWorkflowId()]; ok {
		return nil, serviceerror.NewWorkflowExecutionAlreadyStarted("workflow execution already started", req.GetRequestId(), e.runID)
	}
	e := &execution{runID: uuid.NewString(), kind: req.GetWorkflowType().GetName()}
	f.executions[req.GetWorkflowId()] = e
	return &workflowservice.StartWorkflowExecutionResponse{RunId: e.runID, Started: true}, nil
}

func (f *frontend) signal(ctx context.Context, req *workflowservice.SignalWorkflowExecutionRequest) (*workflowservice.SignalWorkflowExecutionResponse, error) {
	id := req.GetWorkflowExecution().GetWorkflowId()
	f.mu.Lock()
	f.signals = append(f.signals, req)
	f.mu.Unlock()
	if err := f.inject(ctx, "signal", id); err != nil {
		return nil, err
	}
	var d outbox.Delivery
	if err := converter.GetDefaultDataConverter().FromPayloads(req.GetInput(), &d); err != nil {
		return nil, serviceerror.NewInvalidArgument("signal input is not a Delivery")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.executions[id]
	if !ok {
		return nil, serviceerror.NewNotFound("workflow not found for ID: " + id)
	}
	if e.completed {
		return nil, serviceerror.NewNotFound("workflow execution already completed")
	}
	e.signals = append(e.signals, d.OutboxID)
	return &workflowservice.SignalWorkflowExecutionResponse{}, nil
}

// addExecution records an open (or completed) execution of id.
func (f *frontend) addExecution(id string, completed bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.executions[id] = &execution{runID: uuid.NewString(), completed: completed}
}

func (f *frontend) execution(id string) (execution, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.executions[id]
	if !ok {
		return execution{}, false
	}
	return execution{runID: e.runID, kind: e.kind, completed: e.completed, signals: append([]uuid.UUID(nil), e.signals...)}, true
}

func (f *frontend) counts() (executions, starts, signals int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.executions), len(f.starts), len(f.signals)
}

func (f *frontend) lastStart() *workflowservice.StartWorkflowExecutionRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.starts) == 0 {
		return nil
	}
	return f.starts[len(f.starts)-1]
}

func (f *frontend) setFail(fn func(method, workflowID string) error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail = fn
}

// startFrontend serves a frontend on a loopback port until the test ends and
// returns it with its address.
func startFrontend(t *testing.T) (*frontend, string) {
	t.Helper()
	lis, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &frontend{executions: map[string]*execution{}}
	s := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(s, f)
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(s.Stop)
	return f, lis.Addr().String()
}

// dialFrontend is the relay's production client setup (telemetry.NewTemporal
// and workerguard.ConfigureClient) pointed at address, on tel.
func dialFrontend(t *testing.T, address string, tel *telemetry.Telemetry) client.Client {
	t.Helper()
	tt, err := tel.NewTemporal(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	o := client.Options{HostPort: address, Namespace: "nava"}
	workerguard.ConfigureClient(&o, tt)
	c, err := client.Dial(o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

// newTelemetry is a telemetry.Telemetry that exports nothing.
func newTelemetry(t *testing.T, opts ...telemetry.Option) *telemetry.Telemetry {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	return tel
}

// headerString decodes a json/plain string header field.
func headerString(t *testing.T, h *commonpb.Header, key string) (string, bool) {
	t.Helper()
	p := h.GetFields()[key]
	if p == nil {
		return "", false
	}
	var s string
	if err := converter.GetDefaultDataConverter().FromPayload(p, &s); err != nil {
		t.Fatalf("header %s: %v", key, err)
	}
	return s, true
}

// reusePolicyOK reports the start options A.2 requires.
func reusePolicyOK(req *workflowservice.StartWorkflowExecutionRequest) bool {
	return req.GetWorkflowIdReusePolicy() == enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE &&
		req.GetTaskQueue().GetName() == TaskQueue
}
