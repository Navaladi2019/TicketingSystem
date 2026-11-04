package pgstore_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
)

// deadStore is a Store on a pool whose database does not exist: every call
// must come back as a wrapped error, never a zero value that reads as "no
// binding", "no principal" or "domain not owned".
func deadStore(t *testing.T) *pgstore.Store {
	t.Helper()
	cfg, err := pgxpool.ParseConfig("postgres://nava_app:x@127.0.0.1:1/nava?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pgstore.New(pool)
}

func TestStoreDatabaseErrorsAreNotAbsence(t *testing.T) {
	t.Parallel()
	s := deadStore(t)
	tenant := uuid.New()

	if _, err := s.Binding(t.Context(), "https://idp/realms/nava", "org"); err == nil ||
		errors.Is(err, identity.ErrNoBinding) {
		t.Errorf("Binding: err = %v, want a database error", err)
	}
	if _, err := s.Principal(t.Context(), tenant, "sub"); err == nil ||
		errors.Is(err, identity.ErrNoPrincipal) {
		t.Errorf("Principal: err = %v, want a database error", err)
	}
	if owns, err := s.TenantOwnsEmailDomain(t.Context(), tenant, "Example.TEST"); err == nil || owns {
		t.Errorf("TenantOwnsEmailDomain: owns = %v, err = %v, want false and an error", owns, err)
	}
	if paths, err := s.GroupPaths(t.Context(), tenant, uuid.New()); err == nil || paths != nil {
		t.Errorf("GroupPaths: paths = %v, err = %v, want nil and an error", paths, err)
	}
}
