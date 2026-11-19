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
// evaluation (ADR-010 D5b, R7). The Prometheus exporter adds the _total
// suffix: authz_policy_invalid_total{entity_type}. It carries no tenant
// (TICK-93): the log line does not either, and an operator finds the tenant
// by the entity type and the version.
const PolicyInvalidMetric = "authz.policy.invalid"

// invalidCounter is the registered counter; nil records nothing. Allows and
// CompileRead are pure functions called from many places, so the counter is
// registered once per process (RegisterMetrics) rather than passed in.
//
//nolint:gochecknoglobals // one process-wide counter, set once at start (RegisterMetrics)
var invalidCounter atomic.Pointer[metric.Int64Counter]

// RegisterMetrics registers the authorizer's metrics on mp. A process that
// evaluates policies calls it once at start; until it does, nothing is
// counted (the log line is still written).
func RegisterMetrics(mp metric.MeterProvider) error {
	c, err := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/authz").Int64Counter(PolicyInvalidMetric,
		metric.WithDescription("Access policies voided by the planner because a node is invalid (unresolved ref, cycle, cap, unavailable kind), once per evaluation (ADR-010 D5b)."))
	if err != nil {
		return fmt.Errorf("authz: %s: %w", PolicyInvalidMetric, err)
	}
	invalidCounter.Store(&c)
	return nil
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
