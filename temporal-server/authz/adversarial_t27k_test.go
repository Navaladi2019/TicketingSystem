package authz

// Security review of T27k (PR #102): D9h-failures must hold on every Failure a
// worker writes to history, including the ones carried inside a protocol
// message's google.protobuf.Any body (workflow update Response and Rejection).
// At PR #102 head 92fb820 the CI-C walker stopped at Any (`messages[].body.value`
// was classed "not an id or queue"), so neither the golden nor the Authorizer
// saw these failures.

import (
	"testing"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/authorization"

	failurepb "go.temporal.io/api/failure/v1"
	protocolpb "go.temporal.io/api/protocol/v1"
	updatepb "go.temporal.io/api/update/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func updateMessage(t *testing.T, body proto.Message) *protocolpb.Message {
	t.Helper()
	a, err := anypb.New(body)
	if err != nil {
		t.Fatal(err)
	}
	return &protocolpb.Message{Id: "m1/accept", ProtocolInstanceId: "u1", Body: a}
}

// A forged child failure inside an update outcome or rejection is denied, as it
// is in a FailWorkflow command; a real prefixed one passes.
func TestAdversarialUpdateMessageChildFailure(t *testing.T) {
	tok := taskToken(t, wfA)
	bodies := map[string]func(*failurepb.Failure) proto.Message{
		"Response": func(f *failurepb.Failure) proto.Message {
			return &updatepb.Response{Meta: &updatepb.Meta{UpdateId: "u1"},
				Outcome: &updatepb.Outcome{Value: &updatepb.Outcome_Failure{Failure: f}}}
		},
		"Rejection": func(f *failurepb.Failure) proto.Message {
			return &updatepb.Rejection{RejectedRequestMessageId: "u1/request", Failure: f}
		},
	}
	for name, body := range bodies {
		with := func(f *failurepb.Failure) *workflowservice.RespondWorkflowTaskCompletedRequest {
			req := respond(tok)
			req.Messages = []*protocolpb.Message{updateMessage(t, body(f))}
			return req
		}
		t.Run(name+" real child", func(t *testing.T) {
			requireCommandAllow(t, with(childFailure("t:tenantA:child", 1)))
		})
		for forgedName, forged := range forgedIDs {
			t.Run(name+" "+forgedName, func(t *testing.T) {
				if decision, _, logs := authorizeWorker(t, with(childFailure(forged, 2))); decision != authorization.DecisionDeny {
					t.Errorf("update %s with child failure id %q: decision = %v, want deny (log %v)", name, forged, decision, logs)
				}
			})
		}
	}
}
