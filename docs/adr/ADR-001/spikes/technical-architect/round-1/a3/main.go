// Spike A3: does the nava-tenant header reach every hop (start, signal,
// signal-with-start, activity, local activity, child, continue-as-new,
// workflow retry)? Raw interceptor header vs ContextPropagator.
// Throwaway spike code.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	rawKey  = "nava-tenant-raw" // written by raw interceptors only
	propKey = "nava-tenant"     // written by ContextPropagator only
	tq      = "spike-tq"
	ns      = "nava"
)

type ctxKey struct{}

var dc = converter.GetDefaultDataConverter()

func enc(s string) *commonpb.Payload { p, _ := dc.ToPayload(s); return p }
func dec(p *commonpb.Payload) string {
	if p == nil {
		return "<absent>"
	}
	var s string
	_ = dc.FromPayload(p, &s)
	return s
}

func obs(where, wfid, raw, prop string) {
	fmt.Printf("OBS %-28s wf=%-22s raw=%-9s prop=%s\n", where, wfid, raw, prop)
}

// ---------- ContextPropagator ----------
type prop struct{}

func (prop) Inject(ctx context.Context, w workflow.HeaderWriter) error {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		w.Set(propKey, enc(v))
	}
	return nil
}
func (prop) InjectFromWorkflow(ctx workflow.Context, w workflow.HeaderWriter) error {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		w.Set(propKey, enc(v))
	}
	return nil
}
func (prop) Extract(ctx context.Context, r workflow.HeaderReader) (context.Context, error) {
	if p, ok := r.Get(propKey); ok {
		return context.WithValue(ctx, ctxKey{}, dec(p)), nil
	}
	return ctx, nil
}
func (prop) ExtractToWorkflow(ctx workflow.Context, r workflow.HeaderReader) (workflow.Context, error) {
	if p, ok := r.Get(propKey); ok {
		return workflow.WithValue(ctx, ctxKey{}, dec(p)), nil
	}
	return ctx, nil
}

func propVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "<absent>"
}

// ---------- client interceptor (raw header) ----------
type cliI struct{ interceptor.ClientInterceptorBase }

func (c *cliI) InterceptClient(next interceptor.ClientOutboundInterceptor) interceptor.ClientOutboundInterceptor {
	o := &cliO{}
	o.Next = next
	return o
}

type cliO struct{ interceptor.ClientOutboundInterceptorBase }

func setRaw(ctx context.Context, where string) {
	h := interceptor.Header(ctx)
	v, _ := ctx.Value(ctxKey{}).(string)
	if h == nil {
		fmt.Printf("CLI %-28s interceptor.Header(ctx)==nil -> raw header CANNOT be written\n", where)
		return
	}
	if v != "" {
		h[rawKey] = enc(v)
	}
}

func (o *cliO) ExecuteWorkflow(ctx context.Context, in *interceptor.ClientExecuteWorkflowInput) (client.WorkflowRun, error) {
	setRaw(ctx, "client.ExecuteWorkflow")
	return o.Next.ExecuteWorkflow(ctx, in)
}
func (o *cliO) SignalWorkflow(ctx context.Context, in *interceptor.ClientSignalWorkflowInput) error {
	setRaw(ctx, "client.SignalWorkflow")
	return o.Next.SignalWorkflow(ctx, in)
}
func (o *cliO) SignalWithStartWorkflow(ctx context.Context, in *interceptor.ClientSignalWithStartWorkflowInput) (client.WorkflowRun, error) {
	setRaw(ctx, "client.SignalWithStart")
	return o.Next.SignalWithStartWorkflow(ctx, in)
}

// ---------- worker interceptors ----------
type wkI struct{ interceptor.WorkerInterceptorBase }

func (w *wkI) InterceptActivity(ctx context.Context, next interceptor.ActivityInboundInterceptor) interceptor.ActivityInboundInterceptor {
	a := &actIn{}
	a.Next = next
	return a
}

type actIn struct{ interceptor.ActivityInboundInterceptorBase }

func (a *actIn) ExecuteActivity(ctx context.Context, in *interceptor.ExecuteActivityInput) (any, error) {
	raw := "<nil-map>"
	if h := interceptor.Header(ctx); h != nil {
		raw = dec(h[rawKey])
	}
	info := activity.GetInfo(ctx)
	kind := "activity"
	if info.IsLocalActivity {
		kind = "localActivity"
	}
	obs("in:"+kind, info.WorkflowExecution.ID, raw, propVal(ctx.Value(ctxKey{})))
	return a.Next.ExecuteActivity(ctx, in)
}

func (w *wkI) InterceptWorkflow(ctx workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	i := &wfIn{}
	i.Next = next
	return i
}

type wfIn struct{ interceptor.WorkflowInboundInterceptorBase }

func (i *wfIn) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	o := &wfOut{}
	o.Next = outbound
	return i.Next.Init(o)
}

// tenant guard as RFC §4.3 describes: header must exist and equal the id prefix.
func guard(wfid, tenant string) error {
	parts := strings.SplitN(wfid, ":", 3)
	if len(parts) < 3 || parts[0] != "t" || parts[1] != tenant {
		return fmt.Errorf("tenant guard: header %q does not match workflow id %q", tenant, wfid)
	}
	return nil
}

func (i *wfIn) ExecuteWorkflow(ctx workflow.Context, in *interceptor.ExecuteWorkflowInput) (any, error) {
	info := workflow.GetInfo(ctx)
	raw := dec(interceptor.WorkflowHeader(ctx)[rawKey])
	if !workflow.IsReplaying(ctx) {
		obs(fmt.Sprintf("in:ExecuteWorkflow(attempt=%d)", info.Attempt), info.WorkflowExecution.ID, raw, propVal(ctx.Value(ctxKey{})))
	}
	if strings.Contains(info.WorkflowExecution.ID, "nocopy") {
		return i.Next.ExecuteWorkflow(ctx, in) // observe-only run: no guard
	}
	if err := guard(info.WorkflowExecution.ID, raw); err != nil {
		if strings.Contains(info.WorkflowExecution.ID, "panic") {
			panic(err) // fails the workflow TASK
		}
		return nil, temporal.NewNonRetryableApplicationError(err.Error(), "TenantGuard", nil)
	}
	return i.Next.ExecuteWorkflow(ctx, in)
}

func (i *wfIn) HandleSignal(ctx workflow.Context, in *interceptor.HandleSignalInput) error {
	raw := "<nil-map>"
	if h := interceptor.WorkflowHeader(ctx); h != nil {
		raw = dec(h[rawKey])
	}
	if !workflow.IsReplaying(ctx) {
		obs("in:HandleSignal", workflow.GetInfo(ctx).WorkflowExecution.ID, raw, propVal(ctx.Value(ctxKey{})))
	}
	return i.Next.HandleSignal(ctx, in)
}

type wfOut struct{ interceptor.WorkflowOutboundInterceptorBase }

// RFC: "child or activity calls inherit header via interceptor". Copy the raw
// header only when the workflow id contains ":copy" so we see both behaviours.
func copyRaw(ctx workflow.Context, where string) {
	id := workflow.GetInfo(ctx).WorkflowExecution.ID
	h := interceptor.WorkflowHeader(ctx)
	if h == nil {
		fmt.Printf("OUT %-28s WorkflowHeader(ctx)==nil\n", where)
		return
	}
	if strings.Contains(id, ":copy") {
		h[rawKey] = enc(strings.SplitN(id, ":", 3)[1])
	}
}

func (o *wfOut) ExecuteActivity(ctx workflow.Context, t string, args ...any) workflow.Future {
	copyRaw(ctx, "out:ExecuteActivity")
	return o.Next.ExecuteActivity(ctx, t, args...)
}
func (o *wfOut) ExecuteLocalActivity(ctx workflow.Context, t string, args ...any) workflow.Future {
	copyRaw(ctx, "out:ExecuteLocalActivity")
	return o.Next.ExecuteLocalActivity(ctx, t, args...)
}
func (o *wfOut) ExecuteChildWorkflow(ctx workflow.Context, t string, args ...any) workflow.ChildWorkflowFuture {
	copyRaw(ctx, "out:ExecuteChildWorkflow")
	return o.Next.ExecuteChildWorkflow(ctx, t, args...)
}
func (o *wfOut) NewContinueAsNewError(ctx workflow.Context, wfn any, args ...any) error {
	copyRaw(ctx, "out:NewContinueAsNewError")
	return o.Next.NewContinueAsNewError(ctx, wfn, args...)
}
func (o *wfOut) SignalChildWorkflow(ctx workflow.Context, wid, name string, arg any) workflow.Future {
	return o.Next.SignalChildWorkflow(ctx, wid, name, arg)
}

// ---------- workflows / activities ----------
func Act(ctx context.Context) error { return nil }

func Child(ctx workflow.Context) error { return nil }

func Parent(ctx workflow.Context, iteration int) error {
	ao := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second})
	if err := workflow.ExecuteActivity(ao, Act).Get(ctx, nil); err != nil {
		return err
	}
	lo := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{StartToCloseTimeout: 10 * time.Second})
	if err := workflow.ExecuteLocalActivity(lo, Act).Get(ctx, nil); err != nil {
		return err
	}
	id := workflow.GetInfo(ctx).WorkflowExecution.ID
	co := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: id + ":child"})
	if err := workflow.ExecuteChildWorkflow(co, Child).Get(ctx, nil); err != nil {
		return err
	}
	if iteration == 0 {
		var s string
		workflow.GetSignalChannel(ctx, "sig").Receive(ctx, &s)
		return workflow.NewContinueAsNewError(ctx, Parent, 1)
	}
	return nil
}

func RetryWF(ctx workflow.Context) error {
	if workflow.GetInfo(ctx).Attempt == 1 {
		return errors.New("fail first attempt")
	}
	return nil
}

func Upsert(ctx workflow.Context) error {
	return workflow.UpsertTypedSearchAttributes(ctx, saKey.ValueSet("B"))
}

func Wait(ctx workflow.Context) error {
	var s string
	workflow.GetSignalChannel(ctx, "sig").Receive(ctx, &s)
	return nil
}

var saKey = temporal.NewSearchAttributeKeyKeyword("NavaTenantId")

func withTenant(t string) context.Context {
	return context.WithValue(context.Background(), ctxKey{}, t)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	addr := os.Getenv("TEMPORAL_ADDR")
	c, err := client.Dial(client.Options{HostPort: addr, Namespace: ns,
		Interceptors:       []interceptor.ClientInterceptor{&cliI{}},
		ContextPropagators: []workflow.ContextPropagator{prop{}}})
	must(err)
	defer c.Close()

	// search attribute on SQL (PostgreSQL) visibility
	_, err = c.OperatorService().AddSearchAttributes(context.Background(), &operatorservice.AddSearchAttributesRequest{
		Namespace: ns, SearchAttributes: map[string]enums.IndexedValueType{"NavaTenantId": enums.INDEXED_VALUE_TYPE_KEYWORD}})
	fmt.Println("AddSearchAttributes NavaTenantId Keyword:", err)

	w := worker.New(c, tq, worker.Options{Interceptors: []interceptor.WorkerInterceptor{&wkI{}}})
	w.RegisterWorkflow(Parent)
	w.RegisterWorkflow(Child)
	w.RegisterWorkflow(RetryWF)
	w.RegisterWorkflow(Upsert)
	w.RegisterWorkflow(Wait)
	w.RegisterActivity(Act)
	must(w.Start())
	defer w.Stop()

	run := fmt.Sprint(time.Now().Unix())
	sa := temporal.NewSearchAttributes(saKey.ValueSet("A"))
	opts := func(id string) client.StartWorkflowOptions {
		return client.StartWorkflowOptions{ID: id, TaskQueue: tq, TypedSearchAttributes: sa}
	}

	// T1: propagation, with and without the raw copy in the outbound interceptor
	for _, mode := range []string{"copy", "nocopy"} {
		id := fmt.Sprintf("t:A:%s-%s", mode, run)
		fmt.Printf("\n== T1 %s\n", id)
		r, err := c.ExecuteWorkflow(withTenant("A"), opts(id), Parent, 0)
		must(err)
		time.Sleep(3 * time.Second)
		fmt.Println("signal:", c.SignalWorkflow(withTenant("A"), id, "", "sig", "x"))
		err = r.Get(context.Background(), nil)
		fmt.Println("T1 result:", err)
	}

	fmt.Println("\n== T1b signal carrying a DIFFERENT tenant (B) to A's workflow")
	idw := "t:A:wait-" + run
	rw, err := c.ExecuteWorkflow(withTenant("A"), opts(idw), Wait)
	must(err)
	time.Sleep(2 * time.Second)
	must(c.SignalWorkflow(withTenant("B"), idw, "", "sig", "x"))
	fmt.Println("T1b result:", rw.Get(context.Background(), nil))

	fmt.Println("\n== T1c SignalWithStart")
	idsws := "t:A:sws-" + run
	rs, err := c.SignalWithStartWorkflow(withTenant("A"), idsws, "sig", "x", opts(idsws), Wait)
	must(err)
	fmt.Println("T1c result:", rs.Get(context.Background(), nil))

	fmt.Println("\n== T2 workflow retry (attempt 2 header)")
	o := opts("t:A:retry-" + run)
	o.RetryPolicy = &temporal.RetryPolicy{MaximumAttempts: 2, InitialInterval: time.Second}
	rr, err := c.ExecuteWorkflow(withTenant("A"), o, RetryWF)
	must(err)
	fmt.Println("T2 result:", rr.Get(context.Background(), nil))

	fmt.Println("\n== T3 missing header: guard returns error vs panics")
	for _, id := range []string{"t:A:nohdr-err-" + run, "t:A:nohdr-panic-" + run} {
		_, err := c.ExecuteWorkflow(context.Background(), opts(id), Wait)
		must(err)
	}
	fmt.Println("T3 forged-but-consistent: tenant B header with id t:B:... from the same client")
	rf, err := c.ExecuteWorkflow(withTenant("B"), opts("t:B:forged-"+run), RetryWF)
	must(err)
	_ = rf
	time.Sleep(10 * time.Second)
	for _, id := range []string{"t:A:nohdr-err-" + run, "t:A:nohdr-panic-" + run, "t:B:forged-" + run} {
		d, err := c.DescribeWorkflowExecution(context.Background(), id, "")
		must(err)
		attempts := int32(0)
		if pwt := d.GetPendingWorkflowTask(); pwt != nil {
			attempts = pwt.GetAttempt()
		}
		fmt.Printf("T3 %-28s status=%s pendingWFTaskAttempt=%d\n", id, d.WorkflowExecutionInfo.Status, attempts)
	}
	_ = c.TerminateWorkflow(context.Background(), "t:A:nohdr-panic-"+run, "", "cleanup")

	fmt.Println("\n== T4 duplicate start (outbox redelivery)")
	idd := "t:A:dup-" + run
	r1, err := c.ExecuteWorkflow(withTenant("A"), opts(idd), Wait)
	must(err)
	r2, err := c.ExecuteWorkflow(withTenant("A"), opts(idd), Wait)
	fmt.Printf("T4 running, default opts: err=%v sameRun=%v\n", err, err == nil && r1.GetRunID() == r2.GetRunID())
	o4 := opts(idd)
	o4.WorkflowExecutionErrorWhenAlreadyStarted = true
	_, err = c.ExecuteWorkflow(withTenant("A"), o4, Wait)
	fmt.Printf("T4 running, ErrorWhenAlreadyStarted: err=%v\n", err)
	must(c.SignalWorkflow(withTenant("A"), idd, "", "sig", "x"))
	must(r1.Get(context.Background(), nil))
	r3, err := c.ExecuteWorkflow(withTenant("A"), opts(idd), Wait)
	fmt.Printf("T4 after completion, default reuse policy: err=%v newRun=%v\n", err, err == nil && r3 != nil && r3.GetRunID() != r1.GetRunID())
	if err == nil {
		_ = c.TerminateWorkflow(context.Background(), idd, "", "cleanup")
	}
	o5 := opts(idd)
	o5.WorkflowIDReusePolicy = enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE
	r5, err := c.ExecuteWorkflow(withTenant("A"), o5, Wait)
	fmt.Printf("T4 after completion, REJECT_DUPLICATE (ErrorWhenAlreadyStarted=false): err=%v returnedRun==r3:%v\n", err, err == nil && r5.GetRunID() == r3.GetRunID())
	o5.WorkflowExecutionErrorWhenAlreadyStarted = true
	_, err = c.ExecuteWorkflow(withTenant("A"), o5, Wait)
	fmt.Printf("T4 after completion, REJECT_DUPLICATE + ErrorWhenAlreadyStarted: err=%v\n", err)
	d5, _ := c.DescribeWorkflowExecution(context.Background(), idd, "")
	fmt.Printf("T4 latest run status=%s isR3=%v\n", d5.WorkflowExecutionInfo.Status, d5.WorkflowExecutionInfo.Execution.RunId == r3.GetRunID())

	fmt.Println("\n== T5 search attribute: visibility query + mutability")
	ru, err := c.ExecuteWorkflow(withTenant("A"), opts("t:A:upsert-"+run), Upsert)
	must(err)
	must(ru.Get(context.Background(), nil))
	time.Sleep(3 * time.Second)
	for _, q := range []string{"NavaTenantId = 'A'", "NavaTenantId = 'B'"} {
		resp, err := c.ListWorkflow(context.Background(), &workflowservice.ListWorkflowExecutionsRequest{Namespace: ns, Query: q + " AND WorkflowId STARTS_WITH 't:'", PageSize: 1000})
		must(err)
		ids := []string{}
		for _, e := range resp.Executions {
			if strings.HasSuffix(e.Execution.WorkflowId, run) || strings.Contains(e.Execution.WorkflowId, run) {
				ids = append(ids, e.Execution.WorkflowId)
			}
		}
		fmt.Printf("T5 %s -> %d executions from this run; upsert wf present: %v\n", q, len(ids), contains(ids, "t:A:upsert-"+run))
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
