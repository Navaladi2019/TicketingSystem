package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/tenantlimits"
)

// WithOperatorLimits turns PUT /api/v1/operator/tenants/{tenant}/limits on:
// p is the nava_onboard pool the change is written on (tenant_limits' only
// writer, AR-TICK-916-01), in the target tenant's
// transaction (TICK-751). Without it the route answers 501.
func WithOperatorLimits(p *pgxpool.Pool) Option { return func(o *options) { o.limitsPool = p } }

// SetTenantLimits serves PUT /api/v1/operator/tenants/{tenant}/limits (S22,
// TICK-751, ADR-006 D12). Only an operator reaches it: operatorAuth refuses a
// tenant token with 403 before this runs. The tenant is the path's, set as
// app.tenant_id for the one transaction that reads and writes its row and the
// audit entry; nothing in the body names a tenant.
func (h *handlers) SetTenantLimits(ctx context.Context, req api.SetTenantLimitsRequestObject) (api.SetTenantLimitsResponseObject, error) {
	call, ok := ctx.Value(operatorCtxKey{}).(operatorCall)
	if !ok || call.operator.PrincipalID == "" {
		return nil, errNoPrincipal
	}
	if h.limitsPool == nil {
		return nil, notImplemented()
	}
	if req.Body == nil {
		return nil, &problemError{problem.ValidationFailed, []problem.FieldError{{Pointer: "/body", Rule: "required"}}, errors.New("no body")}
	}
	tenant := req.Tenant
	if tenant == uuid.Nil {
		return nil, &problemError{problem.NotFound, nil, errors.New("nil tenant")}
	}
	var got tenantlimits.Limits
	err := tenancy.InTenantTx(ctx, h.limitsPool, tenant, func(tx pgx.Tx) error {
		var err error
		got, err = tenantlimits.SetPerType(ctx, tx, tenant, req.Body.FilterableFieldsPerType, req.Body.Reason, call.operator.PrincipalID)
		return err
	})
	if err != nil {
		return nil, tenantLimitsError(err)
	}
	h.log.InfoContext(ctx, "tenant limits set", slog.String("tenant_id", tenant.String()),
		slog.Int("filterable_fields_per_type", got.FilterableFieldsPerType))
	return api.SetTenantLimits200JSONResponse(api.TenantLimits{FilterableFieldsPerType: got.FilterableFieldsPerType}), nil
}

// tenantLimitsError maps a tenantlimits error to its problem; the value and
// the reason are never echoed.
func tenantLimitsError(err error) error {
	switch {
	case errors.Is(err, tenantlimits.ErrNotFound):
		return &problemError{problem.NotFound, nil, err}
	case errors.Is(err, tenantlimits.ErrOutOfRange):
		return &problemError{problem.ValidationFailed, []problem.FieldError{invalid("/filterable_fields_per_type", "range")}, err}
	case errors.Is(err, tenantlimits.ErrNoReason):
		return &problemError{problem.ValidationFailed, []problem.FieldError{invalid("/reason", "required")}, err}
	case errors.Is(err, tenantlimits.ErrNoOperator):
		return errNoPrincipal
	}
	return err
}
