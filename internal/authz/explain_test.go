package authz_test

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

func TestExplainAllowNamesTheGrantingRule(t *testing.T) {
	t.Parallel()
	g := uuid.New()
	pol := &authz.Policy{Version: 4, Rules: []authz.Rule{
		{ID: "nobody", Ops: []string{authz.OpRead}, Who: authz.Subject{Groups: []uuid.UUID{uuid.New()}}},
		{ID: "write-only", Ops: []string{authz.OpWrite}, Who: authz.Subject{Groups: []uuid.UUID{g}}},
		{ID: "readers", Ops: []string{authz.OpRead}, Who: authz.Subject{Groups: []uuid.UUID{g}}},
		{ID: "later", Ops: []string{authz.OpRead}, Who: authz.Subject{Groups: []uuid.UUID{g}}},
	}}
	target := identity.Principal{PrincipalID: uuid.New(), Groups: []uuid.UUID{g}}
	ex := authz.Explain(pol, identity.Principal{}, target, authz.Row{ID: uuid.New()})
	if !ex.Allow || ex.RuleID != "readers" || ex.Version != 4 {
		t.Fatalf("explanation = %+v, want allow by the first rule that grants read, version 4", ex)
	}
}

func TestExplainDenyHasNoRule(t *testing.T) {
	t.Parallel()
	pol := &authz.Policy{Version: 2, Rules: []authz.Rule{
		{ID: "r", Ops: []string{authz.OpRead}, Who: authz.Subject{Groups: []uuid.UUID{uuid.New()}}},
	}}
	ex := authz.Explain(pol, identity.Principal{}, identity.Principal{PrincipalID: uuid.New()}, authz.Row{})
	if ex.Allow || ex.RuleID != "" || ex.Version != 2 || ex.MaskedFields == nil {
		t.Fatalf("explanation = %+v, want deny with no rule, version 2, and a non-nil (empty) masked list", ex)
	}
}

func TestExplainWithNoPolicyDeniesAtVersionZero(t *testing.T) {
	t.Parallel()
	ex := authz.Explain(nil, identity.Principal{}, identity.Principal{PrincipalID: uuid.New()}, authz.Row{})
	if ex.Allow || ex.Version != 0 || ex.MaskedFields == nil || len(ex.MaskedFields) != 0 {
		t.Fatalf("explanation = %+v, want fail-closed deny at version 0", ex)
	}
}

func TestExplainMatchesTheEnforcementOnState(t *testing.T) {
	t.Parallel()
	g := uuid.New()
	pol := &authz.Policy{Version: 1, Rules: []authz.Rule{{
		ID: "open-only", Ops: []string{authz.OpRead}, Who: authz.Subject{Groups: []uuid.UUID{g}},
		Where: []authz.Cond{{Column: "state", Op: authz.CondEq, Values: []string{"open"}}},
	}}}
	target := identity.Principal{PrincipalID: uuid.New(), Groups: []uuid.UUID{g}}
	for state, want := range map[string]bool{"open": true, "closed": false} {
		row := authz.Row{State: state}
		_, enforced := authz.Allows(pol, target, authz.OpRead, row)
		if ex := authz.Explain(pol, identity.Principal{}, target, row); ex.Allow != want || ex.Allow != enforced {
			t.Errorf("state %s: explain %v, enforcement %v, want %v", state, ex.Allow, enforced, want)
		}
	}
}

// The explainer must not tell an admin that a field exists when the admin
// cannot read it itself.
func TestExplainNeverListsAFieldTheAdminCannotRead(t *testing.T) {
	t.Parallel()
	admins, hr, desk := uuid.New(), uuid.New(), uuid.New()
	pol := &authz.Policy{Version: 1, Masks: []authz.Mask{
		{Field: "salary", Read: authz.Subject{Groups: []uuid.UUID{hr}}, Write: authz.Subject{Groups: []uuid.UUID{hr}}},
		{Field: "notes", Read: authz.Subject{Groups: []uuid.UUID{admins}}, Write: authz.Subject{Groups: []uuid.UUID{admins}}},
		{Field: "open", Read: authz.Subject{Groups: []uuid.UUID{admins, desk}}},
	}}
	admin := identity.Principal{PrincipalID: uuid.New(), Groups: []uuid.UUID{admins}}
	target := identity.Principal{PrincipalID: uuid.New(), Groups: []uuid.UUID{desk}}
	ex := authz.Explain(pol, admin, target, authz.Row{})
	// The target is hidden salary and notes. The admin can read notes but not
	// salary, so only notes is listed; open is readable by the target.
	if want := []string{"notes"}; !slices.Equal(ex.MaskedFields, want) {
		t.Fatalf("masked fields = %v, want %v (salary must not appear)", ex.MaskedFields, want)
	}
}

func TestExplainListsMaskedFieldsSorted(t *testing.T) {
	t.Parallel()
	g := uuid.New()
	pol := &authz.Policy{Version: 1, Masks: []authz.Mask{
		{Field: "zeta", Read: authz.Subject{Groups: []uuid.UUID{g}}},
		{Field: "alpha", Read: authz.Subject{Groups: []uuid.UUID{g}}},
	}}
	admin := identity.Principal{PrincipalID: uuid.New(), Groups: []uuid.UUID{g}}
	ex := authz.Explain(pol, admin, identity.Principal{PrincipalID: uuid.New()}, authz.Row{})
	if want := []string{"alpha", "zeta"}; !slices.Equal(ex.MaskedFields, want) {
		t.Fatalf("masked fields = %v, want %v", ex.MaskedFields, want)
	}
}
