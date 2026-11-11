package server

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/reseed"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// ReseedAuditChain serves POST /api/v1/audit/chain:reseed (ADR-004 R17,
// P5-35). One nava_app transaction: the audit.export gate first, then the
// shared R17 check under the head lock (reseed.Do). The tenant is the
// token's; the request carries none. A tenant with no head is 404, not 409,
// so the call cannot probe other tenants' heads.
func (h *handlers) ReseedAuditChain(ctx context.Context, _ api.ReseedAuditChainRequestObject) (api.ReseedAuditChainResponseObject, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.PrincipalID == uuid.Nil {
		return nil, errNoPrincipal
	}
	if h.pool == nil {
		return nil, notImplemented()
	}
	var res reseed.Result
	err := tenancy.InTenantTx(ctx, h.pool, p.TenantID, func(tx pgx.Tx) error {
		if err := authz.RequireAuditPermission(ctx, tx, p, authz.PermAuditExport); err != nil {
			return err
		}
		var err error
		res, err = reseed.Do(ctx, tx, p.TenantID, p.PrincipalID)
		return err
	})
	switch {
	case errors.Is(err, authz.ErrForbidden):
		return nil, &problemError{problem.Forbidden, nil, err}
	case errors.Is(err, reseed.ErrChainAlive):
		return nil, &problemError{problem.ChainAlive, nil, err}
	case errors.Is(err, reseed.ErrHeadNotFound):
		return nil, &problemError{problem.NotFound, nil, err}
	case err != nil:
		return nil, err
	}
	return api.ReseedAuditChain200JSONResponse{ReseedId: res.ReseedID, SealOutboxId: res.OutboxID}, nil
}
