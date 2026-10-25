package workerguard

import (
	"errors"
	"testing"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Adversarial (TICK-65 security review of PR #73): a typed search attribute
// key is name + type, so NavaTenantId set or unset through a key of another
// type (Text, KeywordList) is a different map entry to GetKeyword, yet the SDK
// encodes it under the same name on the command. The Authorizer checks the
// encoded field by name and denies it; the worker guard must deny it first,
// or the workflow task retries at the server until someone acts.

func TestAdversarialChildNavaTenantIdOtherKeyType(t *testing.T) {
	other := tenantB.String()
	for name, sa := range map[string]temporal.SearchAttributes{
		"text key":         temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyString(TenantSearchAttribute).ValueSet(other)),
		"keyword list key": temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeywordList(TenantSearchAttribute).ValueSet([]string{other})),
	} {
		if childSearchAttributesOK(workflow.ChildWorkflowOptions{TypedSearchAttributes: sa}, tenantA) {
			t.Errorf("%s: child NavaTenantId=%s of tenant B allowed for a tenant A workflow", name, other)
		}
	}
}

func TestAdversarialUpsertNavaTenantIdOtherKeyType(t *testing.T) {
	other := tenantB.String()
	for name, u := range map[string]temporal.SearchAttributeUpdate{
		"text key set":           temporal.NewSearchAttributeKeyString(TenantSearchAttribute).ValueSet(other),
		"keyword list key set":   temporal.NewSearchAttributeKeyKeywordList(TenantSearchAttribute).ValueSet([]string{other}),
		"text key unset":         temporal.NewSearchAttributeKeyString(TenantSearchAttribute).ValueUnset(),
		"keyword list key unset": temporal.NewSearchAttributeKeyKeywordList(TenantSearchAttribute).ValueUnset(),
	} {
		if typedUpsertOK([]temporal.SearchAttributeUpdate{u}, tenantA) {
			t.Errorf("%s: upsert of NavaTenantId through another key type allowed", name)
		}
	}
}

// Adversarial: ContinueAsNewError has exported fields and Header is a
// pointer, so workflow code can take the error NewContinueAsNewError already
// checked and rewrite its header to tenant B before returning it. The guard
// must still fail the workflow instead of letting the command reach the
// Authorizer.
func TestAdversarialContinueAsNewHeaderRewrittenAfterGuard(t *testing.T) {
	id := idOf(tenantA)
	h := newHarness(t, id)
	h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
		err := workflow.NewContinueAsNewError(ctx, "x")
		var can *workflow.ContinueAsNewError
		if !errors.As(err, &can) {
			return err
		}
		p, perr := tenantPayload(tenantB)
		if perr != nil {
			return perr
		}
		can.Header.Fields[TenantHeader] = p
		return can
	})
	wantGuardError(t, h.env.GetWorkflowError(), guardContinueAsNew, outcomeMismatch)
}
