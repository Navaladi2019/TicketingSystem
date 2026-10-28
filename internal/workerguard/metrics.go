package workerguard

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Guard names, the `guard` attribute of DecisionsMetric.
const (
	guardWorkflowStart   = "workflow_start"
	guardSignal          = "signal"
	guardUpdate          = "update"
	guardActivity        = "activity"
	guardChild           = "child"
	guardSignalExternal  = "signal_external"
	guardCancelExternal  = "cancel_external"
	guardContinueAsNew   = "continue_as_new"
	guardSearchAttribute = "search_attribute"
	guardNexus           = "nexus"
	// guardMemo checks workflow.UpsertMemo; a child's memo is guardChild's.
	guardMemo = "memo"
	// Results, side effects and failure text (TICK-245, results.go).
	guardActivityResult  = "activity_result"
	guardActivityFailure = "activity_failure"
	guardWorkflowResult  = "workflow_result"
	guardWorkflowFailure = "workflow_failure"
	guardUpdateResult    = "update_result"
	guardUpdateFailure   = "update_failure"
	guardSideEffect      = "side_effect"
	// guardActivityArgs and guardLocalActivityArgs check what a workflow
	// schedules; guardActivity checks what an activity worker receives.
	guardActivityArgs      = "activity_args"
	guardLocalActivityArgs = "local_activity_args"
)

// Outcomes, the `outcome` attribute of DecisionsMetric. Every outcome but
// outcomeAllow is a deny.
const (
	outcomeAllow = "allow"
	// outcomeMissing: no nava-tenant header, or not a json/plain string.
	outcomeMissing = "missing"
	// outcomeMismatch: the header names another tenant.
	outcomeMismatch = "mismatch"
	// outcomeNoTenant: the workflow's own id has no t:<tenant>: prefix.
	outcomeNoTenant = "no_tenant"
	// outcomeCrossTenant: a target id with another tenant's prefix, or none
	// (the SDK default child id <runID>_<n> is one).
	outcomeCrossTenant = "cross_tenant"
	// outcomeNamespace: a target in another namespace.
	outcomeNamespace = "namespace"
	// outcomeSearchAttribute: NavaTenantId set to another value, or unset.
	outcomeSearchAttribute = "search_attribute"
	// outcomeCron: a cron schedule (no cron in M1, D22).
	outcomeCron = "cron"
	// outcomeNotAllowed: a command the platform never sends (Nexus).
	outcomeNotAllowed = "not_allowed"
	// outcomeRecordData: an argument that is not references only (CI-7).
	outcomeRecordData = "record_data"
	// outcomeInternal: the guard itself failed; it denies.
	outcomeInternal = "internal"
)

// DecisionsMetric counts guard decisions. Its only attributes are `guard` and
// `outcome` (TICK-93 audit): never a tenant or a workflow id, which go in the
// log line instead.
const DecisionsMetric = "nava.worker.guard.decisions"

// recorder emits the metric and, for a deny, the alert log line. Workflow
// guards call it only when the workflow is not replaying, so a deny is
// counted and alerted once.
type recorder struct {
	log       *slog.Logger
	decisions metric.Int64Counter
}

func newRecorder(log *slog.Logger, mp metric.MeterProvider) (*recorder, error) {
	c, err := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/workerguard").Int64Counter(
		DecisionsMetric,
		metric.WithDescription("Worker tenant guard decisions, by guard and outcome."),
		metric.WithUnit("{decision}"),
	)
	if err != nil {
		return nil, err
	}
	return &recorder{log: log, decisions: c}, nil
}

func (r *recorder) count(guard, outcome string) {
	r.decisions.Add(context.Background(), 1, metric.WithAttributeSet(attribute.NewSet(
		attribute.String("guard", guard),
		attribute.String("outcome", outcome),
	)))
}

// deny counts the deny and logs it at error level with alert=true: each deny
// is a tenant-boundary event someone must look at. attrs are ids only
// (workflow id, run id, signal or update name, target id), never a header,
// payload or argument value.
func (r *recorder) deny(guard, outcome string, attrs ...slog.Attr) {
	// Neither the metric nor the log may turn a deny into a panic.
	defer func() { _ = recover() }()
	r.count(guard, outcome)
	base := []slog.Attr{
		slog.Bool("alert", true),
		slog.String("guard", guard),
		slog.String("outcome", outcome),
	}
	r.log.LogAttrs(context.Background(), slog.LevelError, "worker tenant guard denied", append(base, attrs...)...)
}

func (r *recorder) allow(guard string) {
	defer func() { _ = recover() }()
	r.count(guard, outcomeAllow)
}
