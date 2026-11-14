//go:build spike

// Package s1 is the Go half of ADR-007 spike S1 (RFC-017 §7 S1): every access
// token the browser driver (s1.mjs) got from the scratch Keycloak is fed to
// keycloak.Adapter.Resolve exactly as it is on main. Not in CI.
//
//	sh run-keycloak.sh && node s1.mjs && go test -tags spike -count=1 -v .
//
// Run it within the tokens' 300 s lifetime.
package s1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

const issuer = "http://localhost:18181/realms/spike-s1"

// The scratch realm's fixed ids (realm/spike-s1.json).
const (
	orgA     = "00000000-0000-4000-8000-0000000005a1"
	orgB     = "00000000-0000-4000-8000-0000000005b1"
	subOne   = "0f000000-0000-4000-8000-000000000001"
	subMulti = "0f000000-0000-4000-8000-000000000002"
)

// store stands in for pgstore: the spike tests the token shape against
// Resolve, not the data-access path. Both bindings list the brokering IdP
// alias "upstream", so a user brokered by it resolves in either tenant (see
// the evidence page, "Fixture choices").
type store struct {
	tenantA, tenantB uuid.UUID
	// principals holds one row per (tenant, sub), as group sync writes them.
	principals map[uuid.UUID]map[string]uuid.UUID
}

func newStore() store {
	a := uuid.MustParse("5a000000-0000-4000-8000-00000000000a")
	b := uuid.MustParse("5b000000-0000-4000-8000-00000000000b")
	return store{tenantA: a, tenantB: b, principals: map[uuid.UUID]map[string]uuid.UUID{
		a: {subOne: uuid.MustParse("a0000000-0000-4000-8000-000000000001"), subMulti: uuid.MustParse("a0000000-0000-4000-8000-000000000002")},
		b: {subMulti: uuid.MustParse("b0000000-0000-4000-8000-000000000002")},
	}}
}

func (s store) Binding(_ context.Context, iss, org string) (identity.Binding, error) {
	if iss != issuer {
		return identity.Binding{}, identity.ErrNoBinding
	}
	switch org {
	case orgA:
		return identity.Binding{TenantID: s.tenantA, IdPAliases: []string{"upstream"}, TenantState: identity.TenantStateActive}, nil
	case orgB:
		return identity.Binding{TenantID: s.tenantB, IdPAliases: []string{"upstream"}, TenantState: identity.TenantStateActive}, nil
	}
	return identity.Binding{}, identity.ErrNoBinding
}

func (s store) Principal(_ context.Context, tenant uuid.UUID, sub string) (identity.PrincipalRecord, error) {
	id, ok := s.principals[tenant][sub]
	if !ok {
		return identity.PrincipalRecord{}, identity.ErrNoPrincipal
	}
	return identity.PrincipalRecord{ID: id, Enabled: true, LinksKnown: true, FederatedIdPs: []string{"upstream"}}, nil
}

type captured struct {
	Engine string `json:"engine"`
	User   string `json:"user"`
	Step   string `json:"step"`
	Scope  string `json:"scope"`
	Expect string `json:"expect"` // "A", "B" or "401"
	Token  string `json:"token"`
}

type outcome struct {
	Engine, User, Step, Scope, Expect, Got string
	Pass                                   bool
}

func TestS1ResolveUnchanged(t *testing.T) {
	raw, err := os.ReadFile("out/tokens.json")
	if err != nil {
		t.Fatalf("run node s1.mjs first: %v", err)
	}
	var toks []captured
	if err := json.Unmarshal(raw, &toks); err != nil {
		t.Fatal(err)
	}
	if len(toks) == 0 {
		t.Fatal("no tokens captured")
	}

	st := newStore()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	a, err := keycloak.New(t.Context(), keycloak.Config{Issuers: []string{issuer}, Store: st}, out)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close(context.Background()) })

	want := map[string]uuid.UUID{"A": st.tenantA, "B": st.tenantB}
	var results []outcome
	for _, c := range toks {
		o := outcome{Engine: c.Engine, User: c.User, Step: c.Step, Scope: c.Scope, Expect: c.Expect}
		p, err := a.Resolve(t.Context(), c.Token)
		switch {
		case errors.Is(err, identity.ErrTenantUnresolved), errors.Is(err, identity.ErrInvalidToken):
			o.Got = "401: " + err.Error()
			o.Pass = c.Expect == "401"
		case err != nil:
			t.Fatalf("%s %s %q: store error %v", c.Engine, c.User, c.Scope, err)
		default:
			o.Got = fmt.Sprintf("tenant %s principal %s idp %s", p.TenantID, p.PrincipalID, p.IdPAlias)
			o.Pass = want[c.Expect] == p.TenantID && st.principals[p.TenantID][subOf(c.User)] == p.PrincipalID
		}
		if !o.Pass {
			t.Errorf("%s | %s | %s | scope %q: want %s, got %s", c.Engine, c.User, c.Step, c.Scope, c.Expect, o.Got)
		} else {
			t.Logf("ok %s | %s | %s | scope %q: %s", c.Engine, c.User, c.Step, c.Scope, o.Got)
		}
		results = append(results, o)
	}
	b, _ := json.MarshalIndent(results, "", " ")
	if err := os.WriteFile("out/resolve.json", b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func subOf(user string) string {
	if user == "one@org-a.test" {
		return subOne
	}
	return subMulti
}
