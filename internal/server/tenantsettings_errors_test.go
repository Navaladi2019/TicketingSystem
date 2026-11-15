package server

import (
	"errors"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenantsettings"
)

// TICK-896: the caller's own tenant row missing is a server fault (500
// internal_error), never a 404 a client could read as "no such tenant";
// forbidden stays 403, and an unknown error passes through unchanged.
func TestTenantSettingsErrorMapping(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		err  error
		code problem.Code
	}{
		{authz.ErrForbidden, problem.Forbidden},
		{tenantsettings.ErrNotFound, problem.InternalError},
	} {
		var pe *problemError
		if got := tenantSettingsError(tc.err); !errors.As(got, &pe) || pe.code != tc.code || !errors.Is(got, tc.err) {
			t.Errorf("tenantSettingsError(%v) = %v, want the %s problem", tc.err, got, tc.code)
		}
	}
	other := errors.New("boom")
	var pe *problemError
	if got := tenantSettingsError(other); !errors.Is(got, other) || errors.As(got, &pe) {
		t.Errorf("tenantSettingsError(unknown) = %v, want it unchanged", got)
	}
}
