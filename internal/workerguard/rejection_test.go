package workerguard

import (
	"errors"
	"strings"
	"testing"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// P3-22a: the interpreter's failures keep their text in the history only as
// Rejected writes it.

func allRejections() []Rejection {
	return []Rejection{RejectNotFound, RejectInput, RejectStep, RejectDirective, RejectValidation, RejectTransition, RejectForbidden}
}

func TestRejectedIsRecordedAsIs(t *testing.T) {
	for _, r := range allRejections() {
		err := Rejected(r)
		var app *temporal.ApplicationError
		if !errors.As(err, &app) || app.Type() != ErrTypeWorkflowRejected || !app.NonRetryable() || app.HasDetails() {
			t.Fatalf("%s: %v, want a non-retryable %s with no details", r, err, ErrTypeWorkflowRejected)
		}
		if changed, out := recordable(err); changed || !errors.Is(out, err) {
			t.Errorf("%s: the guard changed it to %v", r, out)
		}
	}
}

func TestRejectedUnknownIsOpaque(t *testing.T) {
	err := Rejected(Rejection("title: " + recordData))
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != ErrTypeOpaque || strings.Contains(err.Error(), recordData) {
		t.Errorf("%v, want nava.Opaque with fixed text", err)
	}
}

// TestRejectedTypeWithOtherTextIsWithheld: the type alone is not enough; a
// failure of that type built outside Rejected, with data in its message,
// details or cause, is withheld.
func TestRejectedTypeWithOtherTextIsWithheld(t *testing.T) {
	for name, err := range map[string]error{
		"other message":     temporal.NewNonRetryableApplicationError("workflow rejected: "+recordData, ErrTypeWorkflowRejected, nil),
		"prefix only":       temporal.NewApplicationError("workflow rejected: ", ErrTypeWorkflowRejected),
		"known text, cause": temporal.NewNonRetryableApplicationError("workflow rejected: "+string(RejectNotFound), ErrTypeWorkflowRejected, errors.New(recordData)),
		"known text, details": temporal.NewNonRetryableApplicationError("workflow rejected: "+string(RejectNotFound), ErrTypeWorkflowRejected, nil,
			map[string]string{"title": recordData}),
	} {
		changed, out := recordable(err)
		if !changed || strings.Contains(out.Error(), recordData) {
			t.Errorf("%s: recorded as %v", name, out)
		}
	}
}

// TestRejectedFailsTheWorkflow: through the whole guard, a workflow that
// returns a Rejection fails with it unchanged and no deny.
func TestRejectedFailsTheWorkflow(t *testing.T) {
	h := newHarness(t, idOf(tenantA))
	h.env.ExecuteWorkflow(func(workflow.Context) error { return Rejected(RejectDirective) })
	var app *temporal.ApplicationError
	if err := h.env.GetWorkflowError(); !errors.As(err, &app) || app.Type() != ErrTypeWorkflowRejected ||
		app.Message() != "workflow rejected: "+string(RejectDirective) {
		t.Fatalf("workflow error %v, want the rejection", err)
	}
	h.wantNoAlerts()
}
