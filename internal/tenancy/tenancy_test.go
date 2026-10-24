package tenancy_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TestNoTenant: uuid.Nil is refused before any transaction. The pool is nil,
// so any attempt to begin one would panic.
func TestNoTenant(t *testing.T) {
	t.Parallel()
	called := false
	err := tenancy.InTenantTx(t.Context(), nil, uuid.Nil, func(pgx.Tx) error {
		called = true
		return nil
	})
	if !errors.Is(err, tenancy.ErrNoTenant) {
		t.Errorf("err = %v, want ErrNoTenant", err)
	}
	if called {
		t.Error("fn ran without a tenant")
	}
}
