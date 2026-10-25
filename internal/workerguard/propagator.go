package workerguard

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type tenantKey struct{}

// ContextWithTenant returns ctx carrying tenant, which the Propagator writes
// as the nava-tenant header on a client call made with ctx (the relay's start
// and signal, nava-core's update). tenant must come from a platform-verified
// source (I2): outbox.tenant_id, or the request's resolved principal.
func ContextWithTenant(ctx context.Context, tenant uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantKey{}, tenant)
}

// TenantFromContext returns the tenant ContextWithTenant set. In an activity it
// is the tenant the activity guard verified.
func TenantFromContext(ctx context.Context) (uuid.UUID, bool) {
	t, ok := ctx.Value(tenantKey{}).(uuid.UUID)
	return t, ok && t != uuid.Nil
}

// Propagator is the nava-tenant ContextPropagator. It only writes the header:
//
//   - from workflow code (activities, children, signal-external, cancel-external
//     and continue-as-new) it writes the tenant of the workflow's own id, never
//     a value from the context, so a workflow cannot send another tenant's
//     header;
//   - from a client call it writes the tenant ContextWithTenant set, and
//     nothing when there is none.
//
// Reading is the Guard's job. The SDK panics the workflow task when
// ExtractToWorkflow returns an error (a poison loop), so the extract methods
// never fail and change nothing.
type Propagator struct{}

var _ workflow.ContextPropagator = Propagator{}

// Inject writes the context's tenant, if any.
func (Propagator) Inject(ctx context.Context, w workflow.HeaderWriter) error {
	tenant, ok := TenantFromContext(ctx)
	if !ok {
		return nil
	}
	p, err := tenantPayload(tenant)
	if err != nil {
		return fmt.Errorf("workerguard: encode tenant header: %w", err)
	}
	w.Set(TenantHeader, p)
	return nil
}

// InjectFromWorkflow writes the tenant of the workflow's own id. A workflow
// whose id has no tenant prefix has already failed its start guard; the error
// fails the command that asked for the header.
func (Propagator) InjectFromWorkflow(ctx workflow.Context, w workflow.HeaderWriter) error {
	tenant, err := TenantOf(workflow.GetInfo(ctx).WorkflowExecution.ID)
	if err != nil {
		return err
	}
	p, err := tenantPayload(tenant)
	if err != nil {
		return fmt.Errorf("workerguard: encode tenant header: %w", err)
	}
	w.Set(TenantHeader, p)
	return nil
}

// Extract changes nothing: the activity guard reads the header.
func (Propagator) Extract(ctx context.Context, _ workflow.HeaderReader) (context.Context, error) {
	return ctx, nil
}

// ExtractToWorkflow changes nothing: the workflow guards read the header.
func (Propagator) ExtractToWorkflow(ctx workflow.Context, _ workflow.HeaderReader) (workflow.Context, error) {
	return ctx, nil
}
