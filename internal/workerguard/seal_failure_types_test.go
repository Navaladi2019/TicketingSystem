package workerguard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// The audit.seal failure types (TICK-585) reach the history as themselves,
// not as ErrTypeOpaque, and a sealer failure is not a guard deny: the run's
// failure names what went wrong, and the deny alert keeps meaning record data.
func TestSealFailureTypesPassUnchanged(t *testing.T) {
	id := idOf(tenantA)
	for _, tc := range []struct{ typ, msg string }{
		{ErrTypeAuditSealUnsealable, "seal: entry cannot be canonicalised"},
		{ErrTypeAuditSealHeadNotFound, "seal: audit chain head not found"},
	} {
		for _, local := range []bool{false, true} {
			name := map[bool]string{false: "activity", true: "local activity"}[local]
			t.Run(tc.typ+" "+name, func(t *testing.T) {
				h := newHarness(t, id)
				act := func(context.Context) error { return temporal.NewNonRetryableApplicationError(tc.msg, tc.typ, nil) }
				h.env.RegisterActivityWithOptions(act, activity.RegisterOptions{Name: "seal"})
				h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
					return runActivity(ctx, local, map[bool]any{false: "seal", true: act}[local])
				})
				var app *temporal.ApplicationError
				err := h.env.GetWorkflowError()
				if !errors.As(err, &app) || app.Type() != tc.typ || app.Message() != tc.msg || !app.NonRetryable() {
					t.Errorf("error %v, want the non-retryable %s error unchanged", err, tc.typ)
				}
				h.wantNoAlerts()
				for k, n := range h.decisions() {
					if !strings.HasSuffix(k, "/"+outcomeAllow) && n > 0 {
						t.Errorf("%s{%s} = %d, want no deny", DecisionsMetric, k, n)
					}
				}
			})
		}
	}
}
