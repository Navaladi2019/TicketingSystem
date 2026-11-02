package authz

import (
	"context"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// Authorizer decides whether a principal may write a record of an entity
// type, given the record's payload before (nil on a create) and after the
// write (PLAN-003, the S9 shape; S10 widens it and keeps CanWrite). It returns
// ErrForbidden to refuse. The record write pipeline calls it before any
// statement that changes data.
type Authorizer interface {
	CanWrite(ctx context.Context, p identity.Principal, entity string, before, after map[string]any) error
}

// TenantMemberAuthorizer is the interim S9 Authorizer (P3-10): any principal of
// the tenant may write any record of it, and so may a system writer, which
// carries the tenant but no principal id. The pipeline has already checked
// that p.TenantID is the tenant of its transaction, and that a user write
// names a principal. S10 deletes it (P3-31).
type TenantMemberAuthorizer struct{}

var _ Authorizer = TenantMemberAuthorizer{}

// CanWrite allows every write for a principal that names a tenant.
func (TenantMemberAuthorizer) CanWrite(_ context.Context, p identity.Principal, _ string, _, _ map[string]any) error {
	if p.TenantID == uuid.Nil {
		return ErrForbidden
	}
	return nil
}
