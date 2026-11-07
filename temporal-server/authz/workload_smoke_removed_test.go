package authz

// TICK-374 (P3-27): the M1 smoke workflow types are off the D9 allow-list.
// nava.interpreter is the only platform workflow type, the smoke types are
// denied like any other unlisted type, and the child and continue-as-new
// allow cases run over every entry, so a future entry is covered without
// anyone choosing it.
//
// This file is the one Go file that may name the old types: the leftover
// check (tools/ci/nosmoke) exempts it by path.

import (
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
)

// The removed types: the stack smoke's nava.smoke.Echo and the relay demo's
// nava.smoke.relay.
var removedSmokeTypes = []string{"nava.smoke.Echo", "nava.smoke.relay"}

func TestSmokeTypesDenied(t *testing.T) {
	for _, wt := range removedSmokeTypes {
		if inSet(platformWorkflowTypes(), wt) {
			t.Fatalf("%s is still in platformWorkflowTypes", wt)
		}
		t.Run(wt, func(t *testing.T) { requireTypeDenied(t, wt) })
	}
}

// Every allow-listed type passes the relay's start and signal-with-start, and
// a worker's child and continue-as-new unless it is start-only (TICK-438),
// with the tenantA prefix and header. A start-only type is denied as a child
// or continue-as-new.
func TestChildAndContinueAsNewAllowEveryPlatformType(t *testing.T) {
	types := sortedKeys(platformWorkflowTypes())
	if len(types) == 0 {
		t.Fatal("platformWorkflowTypes is empty")
	}
	tok := taskToken(t, wfA)
	for _, wt := range types {
		t.Run(wt, func(t *testing.T) {
			requireWorkloadAllow(t, Relay, wf+"StartWorkflowExecution", interpreterStart("t:tenantA:wf:1", wt))
			requireWorkloadAllow(t, Relay, wf+"SignalWithStartWorkflowExecution", &workflowservice.SignalWithStartWorkflowExecutionRequest{
				Namespace: Namespace, WorkflowId: "t:tenantA:wf:1", SignalName: "event",
				WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: interpreterQueue()})
			child := interpreterChild("t:tenantA:wf:1:child", jsonString("tenantA"))
			child.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = wt
			can := interpreterContinueAsNew(jsonString("tenantA"))
			can.GetContinueAsNewWorkflowExecutionCommandAttributes().WorkflowType.Name = wt
			if inSet(startOnlyWorkflowTypes(), wt) {
				requireCommandDeny(t, denyCase{name: wt, req: respond(tok, child), reason: ReasonWorkflowType,
					command: "StartChildWorkflowExecution", workflowID: wfA, target: "t:tenantA:wf:1:child", commandIdx: "0"})
				requireCommandDeny(t, denyCase{name: wt, req: respond(tok, can), reason: ReasonWorkflowType,
					command: "ContinueAsNewWorkflowExecution", workflowID: wfA, commandIdx: "0"})
				return
			}
			requireCommandAllow(t, respond(tok, child))
			requireCommandAllow(t, respond(tok, can))
		})
	}
}
