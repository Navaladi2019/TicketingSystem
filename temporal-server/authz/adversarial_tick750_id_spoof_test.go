package authz

import "testing"

// TICK-750 review: checkChildOnlyParent trusts the parent's workflow id, but no
// rule binds an id to its type. A tenant-A run of another type that holds the
// coordinator's id, or an id under a child-only type's prefix, would pass the
// parent rule. Each test below is the first step of that bypass and must deny.
// Same tenant only: checkTarget still pins every id to tenant A.

// A relay start of a non-coordinator type under the coordinator's id.
func TestRelayCannotStartOtherTypeAsCoordinatorID(t *testing.T) {
	requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution",
		interpreterStart("t:tenantA:field_index.coordinator", interpreterType), ReasonWorkflowType)
}

// An interpreter child of type nava.interpreter under the coordinator's id.
func TestInterpreterCannotStartChildAsCoordinatorID(t *testing.T) {
	requireCommandDeny(t, denyCase{name: "interpreter child as coordinator id",
		req:    respond(taskToken(t, wfA), interpreterChild("t:tenantA:field_index.coordinator", jsonString("tenantA"))),
		reason: ReasonWorkflowType, command: "StartChildWorkflowExecution", workflowID: wfA,
		target: "t:tenantA:field_index.coordinator", commandIdx: "0"})
}

// An interpreter child of type nava.interpreter under the removal prefix; it
// could then continue as field_index.remove.run.
func TestInterpreterCannotStartChildUnderRemovalPrefix(t *testing.T) {
	const spoof = "t:tenantA:field_index.remove.run:spoof"
	requireCommandDeny(t, denyCase{name: "interpreter child under removal prefix",
		req:    respond(taskToken(t, wfA), interpreterChild(spoof, jsonString("tenantA"))),
		reason: ReasonWorkflowType, command: "StartChildWorkflowExecution", workflowID: wfA,
		target: spoof, commandIdx: "0"})
}
