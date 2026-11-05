package authz

// TICK-573 (ADR-004 R18, ADR-001-A1 D9a): a relay start may not ask the server
// to create runs on its own. A CronSchedule makes the server start a new run
// on every tick, and a RetryPolicy makes it start a new run after each failure.
// Neither run is a relay start, so the outbox never fenced it and D22 never
// tracked it (R16: every sealer run is a relay start). The relay sets neither
// (internal/relay/dispatch.go startOptions), so only a relay bug or a misused
// nava-relay certificate reaches these denies. commands.go already denies a
// cron schedule on continue-as-new (erratum E1).

import (
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// startPaths returns a relay start and a signal-with-start of wt under wfID,
// each passed through set before it is returned.
func startPaths(wfID, wt string,
	set func(cron *string, retry **commonpb.RetryPolicy),
) map[string]any {
	start := interpreterStart(wfID, wt)
	set(&start.CronSchedule, &start.RetryPolicy)
	sws := auditSealSignalWithStart(wfID, wt)
	set(&sws.CronSchedule, &sws.RetryPolicy)
	return map[string]any{
		"StartWorkflowExecution":           start,
		"SignalWithStartWorkflowExecution": sws,
	}
}

func platformStarts() []struct{ wt, id string } {
	return []struct{ wt, id string }{{auditSealType, sealRunA}, {interpreterType, "t:tenantA:ticket:1"}}
}

// Negative: a cron schedule on any platform type, on either start path.
func TestRelayStartCronScheduleDenied(t *testing.T) {
	for _, p := range platformStarts() {
		for _, cron := range []string{"* * * * *", "@every 1m", " "} {
			for name, req := range startPaths(p.id, p.wt, func(c *string, _ **commonpb.RetryPolicy) { *c = cron }) {
				t.Run(p.wt+"/"+name+"/"+cron, func(t *testing.T) {
					requireWorkloadDeny(t, Relay, wf+name, req, ReasonStartCron)
				})
			}
		}
	}
}

// Negative: a workflow retry policy on any platform type, on either start
// path, including an empty one (the server fills in its defaults, so an empty
// policy still retries) and one with MaximumAttempts 1.
func TestRelayStartRetryPolicyDenied(t *testing.T) {
	for _, p := range platformStarts() {
		for rpName, rp := range map[string]*commonpb.RetryPolicy{
			"empty":       {},
			"unlimited":   {InitialInterval: durationpb.New(1e9)},
			"one attempt": {MaximumAttempts: 1},
		} {
			for name, req := range startPaths(p.id, p.wt, func(_ *string, r **commonpb.RetryPolicy) { *r = rp }) {
				t.Run(p.wt+"/"+name+"/"+rpName, func(t *testing.T) {
					requireWorkloadDeny(t, Relay, wf+name, req, ReasonStartRetryPolicy)
				})
			}
		}
	}
}

// Positive: the relay's own starts (no cron, no retry policy) still pass, on
// both paths, for both platform types.
func TestRelayStartWithoutScheduleAllowed(t *testing.T) {
	for _, p := range platformStarts() {
		for name, req := range startPaths(p.id, p.wt, func(*string, **commonpb.RetryPolicy) {}) {
			t.Run(p.wt+"/"+name, func(t *testing.T) {
				requireWorkloadAllow(t, Relay, wf+name, req)
			})
		}
	}
}

// The new rules sit after the type, queue and id rules, so a start that fails
// one of those still reports it, not the cron reason.
func TestRelayStartCronAfterWorkloadRules(t *testing.T) {
	req := interpreterStart(sealRunA, "steal.me")
	req.CronSchedule = "* * * * *"
	requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", req, ReasonWorkflowType)
	sws := auditSealSignalWithStart("t:tenantA", auditSealType) // no trailing ':', so no tenant
	sws.RetryPolicy = &commonpb.RetryPolicy{}
	requireWorkloadDeny(t, Relay, wf+"SignalWithStartWorkflowExecution", sws, ReasonNoTenantPrefix)
}
