package workerguard

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ErrTypeTenantGuard is the ApplicationError type of every guard failure. The
// message names the guard and the outcome only, never a header, payload or
// argument value.
const ErrTypeTenantGuard = "nava.TenantGuard"

// Guard is the worker interceptor that enforces ADR-001 A.2 "Guards in the
// worker". Install it on every worker, after the tracing interceptor (so a
// guard failure is on the span), with Propagator on the client:
//
//   - workflow start: a missing nava-tenant header, or one that differs from
//     the id's tenant, fails the workflow non-retryably;
//   - signal: from another tenant (or with no header) it is dropped, and the
//     workflow keeps running;
//   - update: from another tenant (or with no header) the validator rejects it;
//   - child, signal-external, cancel-external: a target in another namespace,
//     or whose id is not prefixed with this tenant (the SDK's default child id
//     is not), fails the call; children start only with ChildWorkflowID;
//   - continue-as-new: a header that is not this tenant, or a cron schedule,
//     fails the workflow;
//   - NavaTenantId: set on a child, continue-as-new or upsert to another
//     value, or unset by an upsert, fails the call;
//   - Nexus: always fails (the Authorizer denies Nexus commands);
//   - activity: a header that is missing or differs from its workflow id's
//     tenant fails it non-retryably; otherwise the tenant is on its context
//     for InTenantTx.
//
// Every deny adds to DecisionsMetric and logs an alert. No guard panics: a
// panic in a workflow task is a poison loop, so a guard that fails internally
// denies instead.
type Guard struct {
	interceptor.WorkerInterceptorBase
	rec *recorder
}

var _ interceptor.WorkerInterceptor = (*Guard)(nil)

// New returns the guard, logging denies to log and counting decisions on mp.
func New(log *slog.Logger, mp metric.MeterProvider) (*Guard, error) {
	if log == nil || mp == nil {
		return nil, errors.New("workerguard: nil logger or meter provider")
	}
	rec, err := newRecorder(log, mp)
	if err != nil {
		return nil, err
	}
	return &Guard{rec: rec}, nil
}

func guardError(guard, outcome string) error {
	return temporal.NewNonRetryableApplicationError("tenant guard denied "+guard+": "+outcome, ErrTypeTenantGuard, nil)
}

// check runs f and returns its outcome, or outcomeInternal if it panics.
func check(f func() string) (outcome string) {
	defer func() {
		if recover() != nil {
			outcome = outcomeInternal
		}
	}()
	return f()
}

// ownTenant is the tenant of the running workflow.
func ownTenant(ctx workflow.Context) (uuid.UUID, error) {
	return TenantOf(workflow.GetInfo(ctx).WorkflowExecution.ID)
}

func workflowAttrs(ctx workflow.Context) []slog.Attr {
	info := workflow.GetInfo(ctx)
	return []slog.Attr{
		slog.String("workflow_id", info.WorkflowExecution.ID),
		slog.String("run_id", info.WorkflowExecution.RunID),
	}
}

// record counts and alerts once per decision: not while the workflow replays.
func (r *recorder) record(ctx workflow.Context, guard, outcome string, attrs ...slog.Attr) {
	if workflow.IsReplaying(ctx) {
		return
	}
	if outcome == outcomeAllow {
		r.allow(guard)
		return
	}
	r.deny(guard, outcome, append(workflowAttrs(ctx), attrs...)...)
}

// --- workflow inbound -----------------------------------------------------------

// InterceptWorkflow implements interceptor.WorkerInterceptor.
func (g *Guard) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &workflowInbound{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, rec: g.rec}
}

type workflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	rec *recorder
}

func (w *workflowInbound) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return w.Next.Init(&workflowOutbound{
		WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{Next: outbound},
		rec:                             w.rec,
	})
}

// inboundHeader compares the header of the call on ctx (start, signal or
// update) with the workflow's tenant.
func inboundHeader(ctx workflow.Context) string {
	return check(func() string {
		tenant, err := ownTenant(ctx)
		if err != nil {
			return outcomeNoTenant
		}
		return headerOutcome(interceptor.WorkflowHeader(ctx), tenant)
	})
}

func (w *workflowInbound) ExecuteWorkflow(ctx workflow.Context, in *interceptor.ExecuteWorkflowInput) (any, error) {
	outcome := inboundHeader(ctx)
	w.rec.record(ctx, guardWorkflowStart, outcome)
	if outcome != outcomeAllow {
		return nil, guardError(guardWorkflowStart, outcome)
	}
	result, err := w.Next.ExecuteWorkflow(ctx, in)
	// The ContinueAsNewError is plain data: workflow code can change its
	// header after NewContinueAsNewError checked it. Check again what the
	// command will carry (TICK-65 security review, finding 2).
	var can *workflow.ContinueAsNewError
	if errors.As(err, &can) {
		outcome := check(func() string { return continueAsNewOutcome(can.Header, workflow.GetInfo(ctx)) })
		if outcome != outcomeAllow {
			w.rec.record(ctx, guardContinueAsNew, outcome)
			return nil, guardError(guardContinueAsNew, outcome)
		}
	}
	return result, err
}

// HandleSignal drops a signal whose header is not this tenant's. Returning nil
// keeps the workflow running: a wrong-tenant signal never fails its target.
func (w *workflowInbound) HandleSignal(ctx workflow.Context, in *interceptor.HandleSignalInput) error {
	outcome := inboundHeader(ctx)
	w.rec.record(ctx, guardSignal, outcome, slog.String("signal", in.SignalName))
	if outcome != outcomeAllow {
		return nil
	}
	return w.Next.HandleSignal(ctx, in)
}

// ValidateUpdate rejects an update whose header is not this tenant's. The SDK
// runs validators only when not replaying, so an accepted update was checked.
func (w *workflowInbound) ValidateUpdate(ctx workflow.Context, in *interceptor.UpdateInput) error {
	outcome := inboundHeader(ctx)
	w.rec.record(ctx, guardUpdate, outcome, slog.String("update", in.Name))
	if outcome != outcomeAllow {
		return guardError(guardUpdate, outcome)
	}
	return w.Next.ValidateUpdate(ctx, in)
}

// --- workflow outbound ----------------------------------------------------------

type workflowOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	rec *recorder
}

// sameNamespace reports whether ns names the workflow's own namespace. Empty
// means the workflow's own.
func sameNamespace(ctx workflow.Context, ns string) bool {
	return ns == "" || ns == workflow.GetInfo(ctx).Namespace
}

// targetOutcome checks a child, signal-external or cancel-external target.
func targetOutcome(ctx workflow.Context, namespace, workflowID string) string {
	tenant, err := ownTenant(ctx)
	if err != nil {
		return outcomeNoTenant
	}
	if !sameNamespace(ctx, namespace) {
		return outcomeNamespace
	}
	if target, err := TenantOf(workflowID); err != nil || target != tenant {
		return outcomeCrossTenant
	}
	return outcomeAllow
}

func (o *workflowOutbound) ExecuteChildWorkflow(ctx workflow.Context, childWorkflowType string, args ...any) workflow.ChildWorkflowFuture {
	opts := workflow.GetChildWorkflowOptions(ctx)
	outcome := check(func() string {
		//nolint:staticcheck // SA1019: deprecated, but still settable; the guard must see it to deny it.
		if out := targetOutcome(ctx, opts.Namespace, opts.WorkflowID); out != outcomeAllow {
			return out
		}
		if opts.CronSchedule != "" {
			return outcomeCron
		}
		tenant, _ := ownTenant(ctx)
		if !childSearchAttributesOK(opts, tenant) {
			return outcomeSearchAttribute
		}
		return outcomeAllow
	})
	o.rec.record(ctx, guardChild, outcome, slog.String("target_workflow_id", opts.WorkflowID))
	if outcome != outcomeAllow {
		return failedChild{failedFuture(ctx, guardError(guardChild, outcome))}
	}
	return o.Next.ExecuteChildWorkflow(ctx, childWorkflowType, args...)
}

func (o *workflowOutbound) SignalExternalWorkflow(ctx workflow.Context, workflowID, runID, signalName string, arg any) workflow.Future {
	ns := workflow.GetChildWorkflowOptions(ctx).Namespace //nolint:staticcheck // SA1019: see ExecuteChildWorkflow
	outcome := check(func() string { return targetOutcome(ctx, ns, workflowID) })
	o.rec.record(ctx, guardSignalExternal, outcome, slog.String("target_workflow_id", workflowID), slog.String("signal", signalName))
	if outcome != outcomeAllow {
		return failedFuture(ctx, guardError(guardSignalExternal, outcome))
	}
	return o.Next.SignalExternalWorkflow(ctx, workflowID, runID, signalName, arg)
}

// SignalChildWorkflow signals a child whose id ExecuteChildWorkflow already
// checked; the check here costs nothing and keeps the rule in one place.
func (o *workflowOutbound) SignalChildWorkflow(ctx workflow.Context, workflowID, signalName string, arg any) workflow.Future {
	outcome := check(func() string { return targetOutcome(ctx, "", workflowID) })
	o.rec.record(ctx, guardSignalExternal, outcome, slog.String("target_workflow_id", workflowID), slog.String("signal", signalName))
	if outcome != outcomeAllow {
		return failedFuture(ctx, guardError(guardSignalExternal, outcome))
	}
	return o.Next.SignalChildWorkflow(ctx, workflowID, signalName, arg)
}

func (o *workflowOutbound) RequestCancelExternalWorkflow(ctx workflow.Context, workflowID, runID string) workflow.Future {
	ns := workflow.GetChildWorkflowOptions(ctx).Namespace //nolint:staticcheck // SA1019: see ExecuteChildWorkflow
	outcome := check(func() string { return targetOutcome(ctx, ns, workflowID) })
	o.rec.record(ctx, guardCancelExternal, outcome, slog.String("target_workflow_id", workflowID))
	if outcome != outcomeAllow {
		return failedFuture(ctx, guardError(guardCancelExternal, outcome))
	}
	return o.Next.RequestCancelExternalWorkflow(ctx, workflowID, runID)
}

// NewContinueAsNewError checks what the command will carry: the header the
// Propagator wrote, and the search attributes the SDK copies from the start.
// On a deny the workflow fails with the guard error instead of continuing.
func (o *workflowOutbound) NewContinueAsNewError(ctx workflow.Context, wfn any, args ...any) error {
	err := o.Next.NewContinueAsNewError(ctx, wfn, args...)
	var can *workflow.ContinueAsNewError
	if !errors.As(err, &can) {
		return err
	}
	outcome := check(func() string { return continueAsNewOutcome(can.Header, workflow.GetInfo(ctx)) })
	o.rec.record(ctx, guardContinueAsNew, outcome)
	if outcome != outcomeAllow {
		return guardError(guardContinueAsNew, outcome)
	}
	return err
}

// continueAsNewOutcome checks a continue-as-new command of the run described
// by info: the header the Propagator wrote, no cron schedule (the SDK copies a
// cron-started run's schedule onto the command, which the Authorizer denies;
// no cron in M1), and the NavaTenantId the SDK copies from the start.
func continueAsNewOutcome(header *commonpb.Header, info *workflow.Info) string {
	tenant, err := TenantOf(info.WorkflowExecution.ID)
	if err != nil {
		return outcomeNoTenant
	}
	if out := headerOutcome(header.GetFields(), tenant); out != outcomeAllow {
		return out
	}
	if info.CronSchedule != "" {
		return outcomeCron
	}
	// The encoded form is what the SDK copies onto the command.
	if !indexedTenantOK(info.SearchAttributes, tenant) { //nolint:staticcheck // SA1019
		return outcomeSearchAttribute
	}
	return outcomeAllow
}

func (o *workflowOutbound) UpsertSearchAttributes(ctx workflow.Context, attributes map[string]any) error {
	outcome := check(func() string {
		v, set := attributes[TenantSearchAttribute]
		if !set {
			return outcomeAllow
		}
		tenant, err := ownTenant(ctx)
		if err != nil {
			return outcomeNoTenant
		}
		if s, ok := v.(string); !ok || s != tenant.String() {
			return outcomeSearchAttribute
		}
		return outcomeAllow
	})
	o.rec.record(ctx, guardSearchAttribute, outcome)
	if outcome != outcomeAllow {
		return guardError(guardSearchAttribute, outcome)
	}
	return o.Next.UpsertSearchAttributes(ctx, attributes)
}

func (o *workflowOutbound) UpsertTypedSearchAttributes(ctx workflow.Context, attributes ...temporal.SearchAttributeUpdate) error {
	outcome := check(func() string {
		tenant, err := ownTenant(ctx)
		if err != nil {
			return outcomeNoTenant
		}
		if !typedUpsertOK(attributes, tenant) {
			return outcomeSearchAttribute
		}
		return outcomeAllow
	})
	o.rec.record(ctx, guardSearchAttribute, outcome)
	if outcome != outcomeAllow {
		return guardError(guardSearchAttribute, outcome)
	}
	return o.Next.UpsertTypedSearchAttributes(ctx, attributes...)
}

// ExecuteNexusOperation always fails: the platform has no Nexus endpoints and
// the Authorizer denies every Nexus command.
func (o *workflowOutbound) ExecuteNexusOperation(ctx workflow.Context, _ interceptor.ExecuteNexusOperationInput) workflow.NexusOperationFuture {
	o.rec.record(ctx, guardNexus, outcomeNotAllowed)
	return failedNexus{failedFuture(ctx, guardError(guardNexus, outcomeNotAllowed))}
}

// --- search attributes ----------------------------------------------------------

// tenantSearchAttributeKey is the typed key of NavaTenantId.
func tenantSearchAttributeKey() temporal.SearchAttributeKeyKeyword {
	return temporal.NewSearchAttributeKeyKeyword(TenantSearchAttribute)
}

// childSearchAttributesOK: NavaTenantId, if a child sets it, is the tenant,
// through a Keyword key. A typed key is name plus type, so GetKeyword misses a
// Text or KeywordList key of the same name, which the SDK still encodes as
// NavaTenantId; any such key is denied (TICK-65 security review, finding 1).
func childSearchAttributesOK(opts workflow.ChildWorkflowOptions, tenant uuid.UUID) bool {
	//nolint:staticcheck // SA1019: the untyped map is deprecated, but a workflow can still set it.
	if v, set := opts.SearchAttributes[TenantSearchAttribute]; set {
		if s, ok := v.(string); !ok || s != tenant.String() {
			return false
		}
	}
	for key := range opts.TypedSearchAttributes.GetUntypedValues() {
		if key.GetName() == TenantSearchAttribute && key.GetValueType() != enumspb.INDEXED_VALUE_TYPE_KEYWORD {
			return false
		}
	}
	if s, set := opts.TypedSearchAttributes.GetKeyword(tenantSearchAttributeKey()); set && s != tenant.String() {
		return false
	}
	return true
}

// untouched marks NavaTenantId as not changed by an upsert. It is not a UUID,
// so no update can produce it.
const untouched = "\x00untouched"

// typedUpsertOK applies updates over a NavaTenantId of `untouched`: an update
// that unsets it removes it, one that sets it changes it. It must stay
// untouched or become the tenant. NavaTenantId keys of every other type are
// seeded too, twice with different values, and must come out unchanged: an
// update through any of them, set or unset, is denied (finding 1).
func typedUpsertOK(updates []temporal.SearchAttributeUpdate, tenant uuid.UUID) bool {
	key := tenantSearchAttributeKey()
	all := append([]temporal.SearchAttributeUpdate{key.ValueSet(untouched)}, updates...)
	got, set := temporal.NewSearchAttributes(all...).GetKeyword(key)
	if !set || (got != untouched && got != tenant.String()) {
		return false
	}
	for pass := range 2 {
		seeds, values := otherTypeSeeds(pass)
		after := temporal.NewSearchAttributes(append(seeds, updates...)...).GetUntypedValues()
		for k, want := range values {
			if v, ok := after[k]; !ok || !reflect.DeepEqual(v, want) {
				return false
			}
		}
	}
	return true
}

// otherTypeSeeds returns an update setting a NavaTenantId key of each
// non-Keyword type, and the value each holds. Pass 0 and pass 1 use different
// values, so an update that sets a key to its seed is still seen.
func otherTypeSeeds(pass int) ([]temporal.SearchAttributeUpdate, map[temporal.SearchAttributeKey]any) {
	text := temporal.NewSearchAttributeKeyString(TenantSearchAttribute)
	list := temporal.NewSearchAttributeKeyKeywordList(TenantSearchAttribute)
	i64 := temporal.NewSearchAttributeKeyInt64(TenantSearchAttribute)
	f64 := temporal.NewSearchAttributeKeyFloat64(TenantSearchAttribute)
	flag := temporal.NewSearchAttributeKeyBool(TenantSearchAttribute)
	at := temporal.NewSearchAttributeKeyTime(TenantSearchAttribute)
	sv, n, b := untouched+strconv.Itoa(pass), int64(pass), pass == 1
	tv := time.Unix(int64(pass), 0).UTC()
	return []temporal.SearchAttributeUpdate{
			text.ValueSet(sv), list.ValueSet([]string{sv}), i64.ValueSet(n),
			f64.ValueSet(float64(n)), flag.ValueSet(b), at.ValueSet(tv),
		}, map[temporal.SearchAttributeKey]any{
			text: sv, list: []string{sv}, i64: n, f64: float64(n), flag: b, at: tv,
		}
}

// indexedTenantOK: NavaTenantId in the encoded attributes, if present, is the
// tenant as a json/plain string.
func indexedTenantOK(sa *commonpb.SearchAttributes, tenant uuid.UUID) bool {
	p, set := sa.GetIndexedFields()[TenantSearchAttribute]
	if !set {
		return true
	}
	got, ok := payloadString(p)
	return ok && got == tenant.String()
}

// --- failed futures -------------------------------------------------------------

func failedFuture(ctx workflow.Context, err error) workflow.Future {
	f, s := workflow.NewFuture(ctx)
	s.SetError(err)
	return f
}

// failedChild is a child that never started: every future holds the error.
type failedChild struct{ workflow.Future }

func (c failedChild) GetChildWorkflowExecution() workflow.Future { return c.Future }

func (c failedChild) SignalChildWorkflow(workflow.Context, string, any) workflow.Future {
	return c.Future
}

type failedNexus struct{ workflow.Future }

func (n failedNexus) GetNexusOperationExecution() workflow.Future { return n.Future }

// --- activity inbound -----------------------------------------------------------

// InterceptActivity implements interceptor.WorkerInterceptor.
func (g *Guard) InterceptActivity(_ context.Context, next interceptor.ActivityInboundInterceptor) interceptor.ActivityInboundInterceptor {
	return &activityInbound{ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next}, rec: g.rec}
}

type activityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	rec *recorder
}

// ExecuteActivity compares the activity's header with its workflow id's
// tenant. Present is not enough: a header naming another tenant fails too
// (TICK-84 security finding 1). On success the tenant is on the context, for
// InTenantTx.
func (a *activityInbound) ExecuteActivity(ctx context.Context, in *interceptor.ExecuteActivityInput) (any, error) {
	var (
		tenant uuid.UUID
		info   activity.Info
	)
	outcome := check(func() string {
		info = activity.GetInfo(ctx)
		t, err := TenantOf(info.WorkflowExecution.ID)
		if err != nil {
			return outcomeNoTenant
		}
		tenant = t
		return headerOutcome(interceptor.Header(ctx), t)
	})
	if outcome != outcomeAllow {
		a.rec.deny(guardActivity, outcome,
			slog.String("workflow_id", info.WorkflowExecution.ID),
			slog.String("run_id", info.WorkflowExecution.RunID),
			slog.String("activity_type", info.ActivityType.Name))
		return nil, guardError(guardActivity, outcome)
	}
	a.rec.allow(guardActivity)
	return a.Next.ExecuteActivity(ContextWithTenant(ctx, tenant), in)
}
