package workerguard

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// TestChildWorkflowIDIsV7: the random part of a child workflow id is a UUIDv7
// (TICK-963).
func TestChildWorkflowIDIsV7(t *testing.T) {
	h := newHarness(t, idOf(tenantA))
	var got string
	h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
		var err error
		got, err = ChildWorkflowID(ctx, "child")
		return err
	})
	if err := h.env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	last := got[strings.LastIndex(got, ":")+1:]
	id, err := uuid.Parse(last)
	if err != nil {
		t.Fatalf("child id %q: %v", got, err)
	}
	if id.Version() != 7 {
		t.Fatalf("child id %s is version %d, want 7", id, id.Version())
	}
}
