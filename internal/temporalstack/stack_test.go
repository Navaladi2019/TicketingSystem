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

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
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
	queue := fmt.Sprintf("t28-%d", time.Now().UnixNano())
	w := worker.New(dial(t, "nava-worker"), queue, worker.Options{})
	w.RegisterWorkflow(Echo)
	if err := w.Start(); err != nil {
		t.Fatalf("worker start: %v", err)
	}
	t.Cleanup(w.Stop)

	// Workflow IDs carry the t:<tenant>:<rest> prefix the Authorizer requires;
	// without the tenant segment every workflow task completion is denied.
	run, err := relay.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "t:t28:" + queue, TaskQueue: queue}, Echo, "mtls")
	if err != nil {
		t.Fatalf("nava-relay ExecuteWorkflow: %v", err)
	}
	var out string
	core := dial(t, "nava-core")
	if err := core.GetWorkflow(ctx, run.GetID(), run.GetRunID()).Get(ctx, &out); err != nil || out != "echo:mtls" {
		t.Fatalf("nava-core read result %q, %v", out, err)
	}
}
