package workerguard

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// childKind is the shape of a child's kind: lower case, no colon, so the id
// t:<tenant>:<kind>:<uuid> splits one way.
var childKind = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,62}$`)

// ChildWorkflowID returns a new child workflow id t:<tenant>:<kind>:<uuid> in
// the running workflow's tenant. It is the only way workflow code names a
// child: the SDK's default child id <runID>_<n> has no tenant prefix, so the
// guard (and then the Authorizer) rejects a child started without one.
//
//	id, err := workerguard.ChildWorkflowID(ctx, "approval")
//	cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: id})
//
// The uuid is drawn in a SideEffect, so a replay gets the same id.
func ChildWorkflowID(ctx workflow.Context, kind string) (string, error) {
	if !childKind.MatchString(kind) {
		return "", errors.New("workerguard: child kind must match " + childKind.String())
	}
	tenant, err := ownTenant(ctx)
	if err != nil {
		return "", err
	}
	var id uuid.UUID
	if err := workflow.SideEffect(ctx, func(workflow.Context) any { return uuid.New() }).Get(&id); err != nil {
		return "", fmt.Errorf("workerguard: child id: %w", err)
	}
	return workflowIDPrefix + tenant.String() + ":" + kind + ":" + id.String(), nil
}
