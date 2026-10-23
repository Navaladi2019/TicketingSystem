//go:build temporalstack

// Package temporalstack tests the real Temporal of the compose stack over mTLS:
// each platform identity dials with its own certificate (from `make dev-ca`) and
// a worker with the nava-worker certificate runs a workflow that the nava-relay
// certificate started and the nava-core certificate read back.
//
//	docker compose -f compose.yaml -f deploy/compose/compose.test.yaml up -d --build
//	go test -tags temporalstack -count=1 ./internal/temporalstack
package temporalstack

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const namespace = "nava"

func certDir() string {
	if d := os.Getenv("NAVA_TEST_CERT_DIR"); d != "" {
		return d
	}
	return filepath.Join("..", "..", "deploy", "compose", "certs")
}

func address() string {
	if a := os.Getenv("NAVA_TEST_TEMPORAL_ADDR"); a != "" {
		return a
	}
	return "127.0.0.1:17233"
}

// dial connects as the identity cn. The frontend certificate names `localhost`.
func dial(t *testing.T, cn string) client.Client {
	t.Helper()
	dir := certDir()
	pair, err := tls.LoadX509KeyPair(filepath.Join(dir, cn+".pem"), filepath.Join(dir, cn+".key"))
	if err != nil {
		t.Fatalf("%s: %v (run make dev-ca)", cn, err)
	}
	caPEM, err := os.ReadFile(filepath.Join(dir, "ca.pem"))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	c, err := client.Dial(client.Options{
		HostPort:  address(),
		Namespace: namespace,
		ConnectionOptions: client.ConnectionOptions{TLS: &tls.Config{
			Certificates: []tls.Certificate{pair}, RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS12,
		}},
	})
	if err != nil {
		t.Fatalf("%s: dial: %v", cn, err)
	}
	t.Cleanup(c.Close)
	return c
}

// The smoke workload is the D9 allow-list's only entry (ADR-001-A1): workflow
// type nava.smoke.Echo on task queue nava-tq.
const (
	echoType  = "nava.smoke.Echo"
	taskQueue = "nava-tq"
)

func Echo(ctx workflow.Context, in string) (string, error) { return "echo:" + in, nil }

func TestEachSDKClientDialsWithItsCertificateAndAWorkerStarts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Every identity completes the handshake and its own allowed call.
	for _, cn := range []string{"nava-relay", "nava-core", "nava-worker", "nava-operator", "temporal-system"} {
		c := dial(t, cn)
		if _, err := c.WorkflowService().GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{}); err != nil {
			t.Errorf("%s: GetSystemInfo: %v", cn, err)
		}
	}
	// The identity decides what is allowed: relay may not read the namespace.
	relay := dial(t, "nava-relay")
	_, err := relay.WorkflowService().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: namespace})
	var denied *serviceerror.PermissionDenied
	if !errors.As(err, &denied) {
		t.Errorf("nava-relay DescribeNamespace: got %v, want PermissionDenied", err)
	}
	op := dial(t, "nava-operator")
	if _, err := op.WorkflowService().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: namespace}); err != nil {
		t.Errorf("nava-operator DescribeNamespace: %v", err)
	}

	// A worker with the nava-worker certificate starts and runs a workflow that
	// nava-relay started; nava-core reads the result back.
	w := worker.New(dial(t, "nava-worker"), taskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(Echo, workflow.RegisterOptions{Name: echoType})
	if err := w.Start(); err != nil {
		t.Fatalf("worker start: %v", err)
	}
	t.Cleanup(w.Stop)

	// Workflow IDs carry the t:<tenant>:<rest> prefix the Authorizer requires;
	// without the tenant segment every workflow task completion is denied.
	id := fmt.Sprintf("t:t28:smoke-%d", time.Now().UnixNano())
	run, err := relay.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: id, TaskQueue: taskQueue}, echoType, "mtls")
	if err != nil {
		t.Fatalf("nava-relay ExecuteWorkflow: %v", err)
	}
	var out string
	core := dial(t, "nava-core")
	if err := core.GetWorkflow(ctx, run.GetID(), run.GetRunID()).Get(ctx, &out); err != nil || out != "echo:mtls" {
		t.Fatalf("nava-core read result %q, %v", out, err)
	}
}

// CI-H (RFC-002 §8, ADR-001-A1 D9): the live replay of S1 against our image. A
// nava-relay start of Temporal's batch workflow on the system worker's queue,
// aimed at another tenant's workflow, is denied and the victim keeps running;
// the same for SignalWithStart. A nava-worker STICKY poll named after the
// system queue is denied while a system workflow task is pending there.
func TestCIHReservedSystemWorkloadDenied(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	relay, core, system := dial(t, "nava-relay"), dial(t, "nava-core"), dial(t, "temporal-system")
	nanos := time.Now().UnixNano()

	// The victim is tenant B's workflow. A start delay keeps it running with no
	// worker involved, so only a terminate could close it.
	victimID := fmt.Sprintf("t:B:victim-%d", nanos)
	victim, err := relay.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: victimID, TaskQueue: taskQueue, StartDelay: time.Hour,
	}, echoType, "victim")
	if err != nil {
		t.Fatalf("start victim: %v", err)
	}
	t.Cleanup(func() {
		_ = system.TerminateWorkflow(context.Background(), victimID, victim.GetRunID(), "CI-H cleanup")
	})

	// BatchParams of the system batch workflow: terminate the victim in `nava`.
	input, err := converter.GetDefaultDataConverter().ToPayloads(map[string]any{
		"BatchType": "terminate", "Reason": "CI-H", "Namespace": namespace,
		"Executions": []map[string]string{{"WorkflowId": victimID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	batchType := &commonpb.WorkflowType{Name: "temporal-sys-batch-workflow"}
	sysQueue := &taskqueuepb.TaskQueue{Name: "temporal-sys-per-ns-tq", Kind: enumspb.TASK_QUEUE_KIND_NORMAL}
	attackID := fmt.Sprintf("t:A:attack-%d", nanos)
	var denied *serviceerror.PermissionDenied

	_, err = relay.WorkflowService().StartWorkflowExecution(ctx, &workflowservice.StartWorkflowExecutionRequest{
		Namespace: namespace, WorkflowId: attackID, WorkflowType: batchType, TaskQueue: sysQueue,
		Input: input, RequestId: fmt.Sprintf("cih-start-%d", nanos),
	})
	if !errors.As(err, &denied) {
		t.Errorf("nava-relay StartWorkflowExecution of temporal-sys-batch-workflow: got %v, want PermissionDenied", err)
	}
	_, err = relay.WorkflowService().SignalWithStartWorkflowExecution(ctx, &workflowservice.SignalWithStartWorkflowExecutionRequest{
		Namespace: namespace, WorkflowId: attackID, WorkflowType: batchType, TaskQueue: sysQueue,
		Input: input, SignalName: "cih", RequestId: fmt.Sprintf("cih-sws-%d", nanos),
	})
	if !errors.As(err, &denied) {
		t.Errorf("nava-relay SignalWithStartWorkflowExecution of temporal-sys-batch-workflow: got %v, want PermissionDenied", err)
	}

	// Give a batch run, had one started, time to act; then the victim must
	// still be running and no attack workflow may exist.
	time.Sleep(5 * time.Second)
	desc, err := core.DescribeWorkflowExecution(ctx, victimID, victim.GetRunID())
	if err != nil {
		t.Fatalf("describe victim: %v", err)
	}
	if st := desc.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		t.Errorf("victim %s status %v, want RUNNING", victimID, st)
	}
	var notFound *serviceerror.NotFound
	if _, err := system.DescribeWorkflowExecution(ctx, attackID, ""); !errors.As(err, &notFound) {
		t.Errorf("attack workflow %s: got %v, want NotFound", attackID, err)
	}

	// A system workflow task pending on the system queue: temporal-system (the
	// server's own identity, exempt from D9) starts a workflow there.
	pendingID := fmt.Sprintf("t:cih:pending-%d", nanos)
	pending, err := system.WorkflowService().StartWorkflowExecution(ctx, &workflowservice.StartWorkflowExecutionRequest{
		Namespace: namespace, WorkflowId: pendingID, WorkflowType: &commonpb.WorkflowType{Name: "nava.cih.Pending"},
		TaskQueue: sysQueue, RequestId: fmt.Sprintf("cih-pending-%d", nanos),
	})
	if err != nil {
		t.Fatalf("temporal-system start on temporal-sys-per-ns-tq: %v", err)
	}
	t.Cleanup(func() {
		_ = system.TerminateWorkflow(context.Background(), pendingID, pending.GetRunId(), "CI-H cleanup")
	})
	deadline := time.Now().Add(15 * time.Second)
	for {
		d, err := system.DescribeWorkflowExecution(ctx, pendingID, pending.GetRunId())
		if err == nil && d.GetPendingWorkflowTask() != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no workflow task pending on temporal-sys-per-ns-tq for %s (err %v)", pendingID, err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	pollCtx, pollCancel := context.WithTimeout(ctx, 30*time.Second)
	defer pollCancel()
	attacker := dial(t, "nava-worker")
	_, err = attacker.WorkflowService().PollWorkflowTaskQueue(pollCtx, &workflowservice.PollWorkflowTaskQueueRequest{
		Namespace: namespace, Identity: "cih-attacker",
		TaskQueue: &taskqueuepb.TaskQueue{Name: "temporal-sys-per-ns-tq", Kind: enumspb.TASK_QUEUE_KIND_STICKY, NormalName: taskQueue},
	})
	if !errors.As(err, &denied) {
		t.Errorf("nava-worker STICKY poll on temporal-sys-per-ns-tq: got %v, want PermissionDenied", err)
	}
}
