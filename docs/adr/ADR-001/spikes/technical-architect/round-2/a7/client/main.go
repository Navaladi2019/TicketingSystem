// Spike A7 client: SDK v1.49.0 clients holding each certificate identity.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const ns = "nava"

func must(err error) {
	if err != nil {
		panic(err)
	}
}

type kp struct{ pem, kpem []byte }

func load(n string) *kp {
	p, err := os.ReadFile("/certs/" + n + ".pem")
	must(err)
	k, err := os.ReadFile("/certs/" + n + ".key")
	must(err)
	return &kp{p, k}
}

// ---------------- workflows (deliberately unguarded: simulates a compromised worker) ----------------
func Child(ctx workflow.Context) (string, error) { return workflow.GetInfo(ctx).WorkflowExecution.ID, nil }

func Parent(ctx workflow.Context, childID string) (string, error) {
	co := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: childID, WorkflowTaskTimeout: 5 * time.Second})
	var out string
	err := workflow.ExecuteChildWorkflow(co, Child).Get(ctx, &out)
	return out, err
}

func SignalOther(ctx workflow.Context, target string) error {
	return workflow.SignalExternalWorkflow(ctx, target, "", "sig", "x").Get(ctx, nil)
}

func Wait(ctx workflow.Context) error {
	workflow.GetSignalChannel(ctx, "sig").Receive(ctx, nil)
	return nil
}

func main() {
	addr := os.Getenv("TEMPORAL_ADDR")
	caPEM, err := os.ReadFile("/certs/ca.pem")
	must(err)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	dial := func(id *kp) (client.Client, error) {
		cert, _ := tls.X509KeyPair(id.pem, id.kpem)
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		return client.DialContext(ctx, client.Options{HostPort: addr, Namespace: ns,
			ConnectionOptions: client.ConnectionOptions{TLS: &tls.Config{RootCAs: pool, ServerName: "localhost",
				Certificates: []tls.Certificate{cert}}}})
	}
	ids := map[string]*kp{}
	for _, cn := range []string{"nava-relay", "nava-core", "nava-worker", "nava-unknown"} {
		ids[cn] = load(cn)
	}
	rogue := load("rogue") // right CN, wrong CA

	fmt.Println("\n== T1 handshake: right CN, rogue CA")
	if c, err := dial(rogue); err != nil {
		fmt.Println("T1 dial error:", firstLine(err))
	} else {
		_, err = c.WorkflowService().GetSystemInfo(context.Background(), &workflowservice.GetSystemInfoRequest{})
		fmt.Println("T1 dial OK?! GetSystemInfo err:", firstLine(err))
		c.Close()
	}

	clients := map[string]client.Client{}
	fmt.Println("\n== T2 dial per identity (SDK Dial calls GetSystemInfo)")
	for _, cn := range []string{"nava-relay", "nava-core", "nava-worker", "nava-unknown"} {
		c, err := dial(ids[cn])
		fmt.Printf("T2 dial %-13s err=%v\n", cn, firstLine(err))
		if err == nil {
			clients[cn] = c
			defer c.Close()
		}
	}

	tq := "a7"
	if wc, ok := clients["nava-worker"]; ok {
		w := worker.New(wc, tq, worker.Options{})
		w.RegisterWorkflow(Parent)
		w.RegisterWorkflow(Child)
		w.RegisterWorkflow(SignalOther)
		w.RegisterWorkflow(Wait)
		err := w.Start()
		fmt.Println("T3 worker.Start with nava-worker cert:", firstLine(err))
		defer w.Stop()
	}

	fmt.Println("\n== T4 API matrix per identity")
	for _, cn := range []string{"nava-relay", "nava-core", "nava-worker", "nava-unknown"} {
		c := clients[cn]
		if c == nil {
			fmt.Printf("T4 %-13s (no client)\n", cn)
			continue
		}
		id := fmt.Sprintf("t:A:matrix-%s", cn)
		_, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{ID: id, TaskQueue: tq}, Wait)
		fmt.Printf("T4 %-13s Start           -> %s\n", cn, verdict(err))
		err = c.SignalWorkflow(context.Background(), id, "", "sig", nil)
		fmt.Printf("T4 %-13s Signal          -> %s\n", cn, verdict(err))
		err = c.CancelWorkflow(context.Background(), id, "")
		fmt.Printf("T4 %-13s Cancel          -> %s\n", cn, verdict(err))
		_, err = c.DescribeWorkflowExecution(context.Background(), id, "")
		fmt.Printf("T4 %-13s Describe        -> %s\n", cn, verdict(err))
		err = c.TerminateWorkflow(context.Background(), id, "", "x")
		fmt.Printf("T4 %-13s Terminate       -> %s\n", cn, verdict(err))
		_, err = c.WorkflowService().ListWorkflowExecutions(context.Background(), &workflowservice.ListWorkflowExecutionsRequest{Namespace: ns})
		fmt.Printf("T4 %-13s ListWorkflows   -> %s\n", cn, verdict(err))
	}

	relay, core := clients["nava-relay"], clients["nava-core"]
	if relay != nil {
		fmt.Println("\n== T5 relay starts A's workflow; the worker cert runs it (end-to-end with the RFC list)")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		r, err := relay.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "t:A:ok", TaskQueue: tq}, Parent, "t:A:ok:child")
		fmt.Println("T5 start:", verdict(err))
		if err == nil && core != nil {
			var out string
			// relay cannot read results (no GetWorkflowExecutionHistory); core can
			err = core.GetWorkflow(ctx, "t:A:ok", r.GetRunID()).Get(ctx, &out)
			fmt.Printf("T5 result via core: out=%q err=%s\n", out, verdict(err))
		}
		cancel()

		fmt.Println("\n== T6 worker-side cross-tenant child start (workflow code without the outbound guard)")
		ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
		r, err = relay.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "t:A:evil", TaskQueue: tq}, Parent, "t:B:evil-child")
		fmt.Println("T6 start:", verdict(err))
		if err == nil && core != nil {
			var out string
			err = core.GetWorkflow(ctx, "t:A:evil", r.GetRunID()).Get(ctx, &out)
			fmt.Printf("T6 parent result: out=%q err=%s\n", out, verdict(err))
			d, derr := core.DescribeWorkflowExecution(context.Background(), "t:B:evil-child", "")
			if derr == nil {
				fmt.Printf("T6 t:B:evil-child EXISTS status=%s\n", d.WorkflowExecutionInfo.Status)
			} else {
				fmt.Printf("T6 t:B:evil-child describe: %s\n", verdict(derr))
			}
		}
		cancel()

		fmt.Println("\n== T7 worker-side cross-tenant signal-external")
		ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
		_, err = relay.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "t:B:victim", TaskQueue: tq}, Wait)
		fmt.Println("T7 victim start:", verdict(err))
		r, err = relay.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "t:A:signaller", TaskQueue: tq}, SignalOther, "t:B:victim")
		if err == nil && core != nil {
			err = core.GetWorkflow(ctx, "t:A:signaller", r.GetRunID()).Get(ctx, nil)
			fmt.Printf("T7 signaller result: %s\n", verdict(err))
			d, _ := core.DescribeWorkflowExecution(context.Background(), "t:B:victim", "")
			if d != nil {
				fmt.Printf("T7 t:B:victim status=%s (Completed means the cross-tenant signal landed)\n", d.WorkflowExecutionInfo.Status)
			}
		}
		cancel()
	}

}

func verdict(err error) string {
	if err == nil {
		return "OK"
	}
	var pd *serviceerror.PermissionDenied
	if errors.As(err, &pd) {
		return "PermissionDenied"
	}
	return "error: " + firstLine(err)
}

func firstLine(err error) string {
	if err == nil {
		return "<nil>"
	}
	s := err.Error()
	if i := strings.IndexByte(s, '\n'); i > 0 {
		s = s[:i]
	}
	if len(s) > 180 {
		s = s[:180]
	}
	return s
}
