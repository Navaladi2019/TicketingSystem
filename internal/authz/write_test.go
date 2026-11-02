package authz_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

func TestTenantMemberAuthorizer(t *testing.T) {
	a := authz.TenantMemberAuthorizer{}
	ctx := context.Background()
	tenant := uuid.New()
	if err := a.CanWrite(ctx, identity.Principal{TenantID: tenant, PrincipalID: uuid.New()}, "incident", nil, nil); err != nil {
		t.Fatalf("tenant principal: %v", err)
	}
	if err := a.CanWrite(ctx, identity.Principal{TenantID: tenant}, "incident", nil, nil); err != nil {
		t.Fatalf("system writer: %v", err)
	}
	if err := a.CanWrite(ctx, identity.Principal{PrincipalID: uuid.New()}, "incident", nil, nil); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("no tenant: %v, want ErrForbidden", err)
	}
}
