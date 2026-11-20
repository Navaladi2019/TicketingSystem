package authz

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// PolicyInvalidMetric counts the policies the planner voided, once per
// evaluation (ADR-010 D5b, R7). An evaluation is one plan: one Allows, one
// CompileRead, one ReadBranches or one ReadPredicate call. The list read path
// uses ReadPredicate, so a list request counts each voided policy once. A
// write (ACL.CanWrite) evaluates the before and the after row separately, so
// an update counts a voided write policy twice and a create once. The Prometheus exporter adds the _total
// suffix: authz_policy_invalid_total{entity_type}. It carries no tenant
// (TICK-93): the log line does not either, and an operator finds the tenant
// by the entity type and the version.
const PolicyInvalidMetric = "authz.policy.invalid"

// ReadBranchesFallbackMetric counts the ReadBranches and ReadPredicate calls
// whose plan has more branches than the SP-2 threshold (ReadBranchThreshold), so
// the list falls back to CompileRead's OR form (ADR-010 D6). One call counts
// once. Prometheus: authz_read_branches_fallback_total{entity_type}. No tenant,
// as for authz_policy_invalid_total. It should stay at zero: the D6 check at
// planning confirms no published document is above the threshold
// (ReadBranchBound), so a rise means a document published since needs a look.
const ReadBranchesFallbackMetric = "authz.read_branches.fallback"

// PolicyRefMissingMetric counts the workflow policy tests that named no policy
// (ADR-010 D9): the record's entity type is not the reference's, or the type
// has no published policy of that name. It denies, is audited, and should stay
// at zero. Prometheus: authz_policy_ref_missing_total{entity_type}, the label
// the record's own entity type, never a name from a request or a tenant.
const PolicyRefMissingMetric = "authz.policy_ref.missing"

// refMissingCounter is the registered PolicyRefMissingMetric counter; nil
// records nothing.
//
//nolint:gochecknoglobals // one process-wide counter, set once at start (RegisterMetrics)
var refMissingCounter atomic.Pointer[metric.Int64Counter]

// invalidCounter is the registered counter; nil records nothing. Allows and
// CompileRead are pure functions called from many places, so the counter is
// registered once per process (RegisterMetrics) rather than passed in.
//
//nolint:gochecknoglobals // one process-wide counter, set once at start (RegisterMetrics)
var invalidCounter atomic.Pointer[metric.Int64Counter]

// fallbackCounter is the registered ReadBranchesFallbackMetric counter; nil
// records nothing.
//
//nolint:gochecknoglobals // one process-wide counter, set once at start (RegisterMetrics)
var fallbackCounter atomic.Pointer[metric.Int64Counter]

// RegisterMetrics registers the authorizer's metrics on mp. A process that
// evaluates policies calls it once at start; until it does, nothing is
// counted (the log line is still written).
func RegisterMetrics(mp metric.MeterProvider) error {
	meter := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/authz")
	c, err := meter.Int64Counter(PolicyInvalidMetric,
		metric.WithDescription("Access policies voided by the planner because a node is invalid (unresolved ref, cycle, cap, unavailable kind), once per evaluation (ADR-010 D5b)."))
	if err != nil {
		return fmt.Errorf("authz: %s: %w", PolicyInvalidMetric, err)
	}
	f, err := meter.Int64Counter(ReadBranchesFallbackMetric,
		metric.WithDescription("List reads whose access predicate has more branches than the SP-2 threshold and so use the OR form (ADR-010 D6)."))
	if err != nil {
		return fmt.Errorf("authz: %s: %w", ReadBranchesFallbackMetric, err)
	}
	m, err := meter.Int64Counter(PolicyRefMissingMetric,
		metric.WithDescription("Workflow policy tests that named a missing entity type or policy and so denied (ADR-010 D9)."))
	if err != nil {
		return fmt.Errorf("authz: %s: %w", PolicyRefMissingMetric, err)
	}
	refMissingCounter.Store(&m)
	invalidCounter.Store(&c)
	fallbackCounter.Store(&f)
	return nil
}

// countVoided records one evaluation of each voided policy in planned.
func countVoided(pol *Policy, planned []plannedPolicy) {
	for _, pp := range planned {
		if pp.err != nil {
			countInvalid(pol, pp)
		}
	}
}

// countInvalid records one evaluation of a voided policy.
func countInvalid(pol *Policy, pp plannedPolicy) {
	var rule string
	var pe *PlanError
	if errors.As(pp.err, &pe) {
		rule = pe.Rule
	}
	// Allows and CompileRead take no context; the request's is not at hand.
	ctx := context.Background()
	slog.WarnContext(ctx, "authz: access policy voided, it grants nothing",
		slog.String("entity_type", pol.EntityType), slog.Int("version", pol.Version),
		slog.String("policy", pp.name), slog.String("problem", rule))
	if c := invalidCounter.Load(); c != nil {
		(*c).Add(ctx, 1, metric.WithAttributes(attribute.String("entity_type", pol.EntityType)))
	}
}

// countReadFallback records one read that fell back to CompileRead because its
// plan is above the SP-2 total-branch threshold.
func countReadFallback(pol *Policy) {
	if c := fallbackCounter.Load(); c != nil {
		(*c).Add(context.Background(), 1, metric.WithAttributes(attribute.String("entity_type", pol.EntityType)))
	}
}

// countRefMissing records one workflow policy test that named no policy.
func countRefMissing(ctx context.Context, entityType string) {
	if c := refMissingCounter.Load(); c != nil {
		(*c).Add(ctx, 1, metric.WithAttributes(attribute.String("entity_type", entityType)))
	}
}
