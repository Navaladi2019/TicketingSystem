package identitystack

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestRealmFile asserts the ADR-001 A.1 settings on the import file itself, so
// a drift fails `go test` without a container.
func TestRealmFile(t *testing.T) {
	raw, err := os.ReadFile("../../deploy/compose/keycloak/import/realm-nava.json")
	if err != nil {
		t.Fatal(err)
	}
	var r struct {
		Realm                string   `json:"realm"`
		OrganizationsEnabled bool     `json:"organizationsEnabled"`
		AccessTokenLifespan  int      `json:"accessTokenLifespan"`
		DefaultScopes        []string `json:"defaultDefaultClientScopes"`
		Clients              []struct {
			ClientID  string            `json:"clientId"`
			Overrides map[string]string `json:"authenticationFlowBindingOverrides"`
		} `json:"clients"`
		ClientScopes []struct {
			Name            string `json:"name"`
			ProtocolMappers []struct {
				ProtocolMapper string            `json:"protocolMapper"`
				Config         map[string]string `json:"config"`
			} `json:"protocolMappers"`
		} `json:"clientScopes"`
		IdentityProviders []struct {
			Alias, FirstBrokerLoginFlowAlias string
			TrustEmail                       bool
		} `json:"identityProviders"`
		Flows []struct {
			ID         string `json:"id"`
			Alias      string `json:"alias"`
			Executions []struct {
				Authenticator       string `json:"authenticator"`
				AuthenticatorConfig string `json:"authenticatorConfig"`
				FlowAlias           string `json:"flowAlias"`
				Requirement         string `json:"requirement"`
				Priority            int    `json:"priority"`
			} `json:"authenticationExecutions"`
		} `json:"authenticationFlows"`
		AuthenticatorConfig []struct {
			Alias  string            `json:"alias"`
			Config map[string]string `json:"config"`
		} `json:"authenticatorConfig"`
		Organizations []struct {
			ID      string `json:"id"`
			Alias   string `json:"alias"`
			Domains []struct {
				Verified bool `json:"verified"`
			} `json:"domains"`
		} `json:"organizations"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	if r.Realm != "nava" || !r.OrganizationsEnabled {
		t.Errorf("realm=%q organizationsEnabled=%v", r.Realm, r.OrganizationsEnabled)
	}
	if r.AccessTokenLifespan <= 0 || r.AccessTokenLifespan > 300 {
		t.Errorf("accessTokenLifespan = %d, want 1..300", r.AccessTokenLifespan)
	}
	var org, idp bool
	for _, s := range r.ClientScopes {
		for _, m := range s.ProtocolMappers {
			c := m.Config
			switch m.ProtocolMapper {
			case "oidc-organization-membership-mapper":
				org = true
				if c["addOrganizationId"] != "true" || c["addOrganizationAttributes"] != "false" || c["multivalued"] != "true" || c["addOrganizationDomain"] != "false" || c["jsonType.label"] != "JSON" || c["access.token.claim"] != "true" {
					t.Errorf("organization mapper config = %v", c)
				}
			case "oidc-usersessionmodel-note-mapper":
				if c["claim.name"] == "idp" {
					idp = true
					if c["user.session.note"] != "identity_provider" || c["access.token.claim"] != "true" {
						t.Errorf("idp mapper config = %v", c)
					}
				}
			}
		}
	}
	if !org || !idp {
		t.Errorf("organization mapper present=%v, idp mapper present=%v", org, idp)
	}
	flows, flowIDs := map[string]bool{}, map[string]string{}
	for _, f := range r.Flows {
		flows[f.Alias] = true
		flowIDs[f.Alias] = f.ID
		if f.Alias != "nava-first-broker-login" {
			continue
		}
		want := []string{"idp-create-user-if-unique", "idp-add-organization-member"}
		if len(f.Executions) != len(want) {
			t.Fatalf("broker flow executions = %v, want exactly %v", f.Executions, want)
		}
		for i, e := range f.Executions {
			if e.Authenticator != want[i] || e.Requirement != "REQUIRED" {
				t.Errorf("execution %d = %+v, want %s REQUIRED", i, e, want[i])
			}
		}
	}
	if !flows["nava-first-broker-login"] || len(r.IdentityProviders) != 2 {
		t.Fatalf("flows=%v idps=%d", flows, len(r.IdentityProviders))
	}
	for _, p := range r.IdentityProviders {
		if p.TrustEmail || p.FirstBrokerLoginFlowAlias != "nava-first-broker-login" {
			t.Errorf("idp %s: trustEmail=%v flow=%q", p.Alias, p.TrustEmail, p.FirstBrokerLoginFlowAlias)
		}
	}
	if got := strings.Join(r.DefaultScopes, ","); got != "basic,roles,profile,email" {
		t.Errorf("defaultDefaultClientScopes = %q, want basic,roles,profile,email", got)
	}
	// Stable organization ids (the tenant UUIDs) for T15 seeds.
	wantOrg := map[string]string{"tenant-a": "00000000-0000-4000-8000-0000000000a1", "tenant-b": "00000000-0000-4000-8000-0000000000b1"}
	for _, o := range r.Organizations {
		if o.ID != wantOrg[o.Alias] {
			t.Errorf("organization %s id = %q, want %q", o.Alias, o.ID, wantOrg[o.Alias])
		}
	}
	// The operator audience is only for nava-operator holders.
	if !flows["nava-operator-direct-grant"] || flowIDs["nava-operator-direct-grant"] != "00000000-0000-4000-8000-0000000000f1" {
		t.Error("flow nava-operator-direct-grant missing")
	}
	assertOperatorFlow(t, r.Flows, r.AuthenticatorConfig)
	for _, c := range r.Clients {
		if c.ClientID == "nava-operator-cli" && c.Overrides["direct_grant"] != "00000000-0000-4000-8000-0000000000f1" {
			t.Errorf("nava-operator-cli direct_grant override = %q", c.Overrides["direct_grant"])
		}
	}
	for _, o := range r.Organizations {
		for _, d := range o.Domains {
			if !d.Verified {
				t.Errorf("organization %s has an unverified domain", o.Alias)
			}
		}
	}
}

// assertOperatorFlow pins the direct-grant flow of nava-operator-cli: Password,
// then a CONDITIONAL subflow that denies users without nava-operator. A broken
// flow would hand the operator audience to every tenant user.
func assertOperatorFlow(t *testing.T, flows []struct {
	ID         string `json:"id"`
	Alias      string `json:"alias"`
	Executions []struct {
		Authenticator       string `json:"authenticator"`
		AuthenticatorConfig string `json:"authenticatorConfig"`
		FlowAlias           string `json:"flowAlias"`
		Requirement         string `json:"requirement"`
		Priority            int    `json:"priority"`
	} `json:"authenticationExecutions"`
}, cfgs []struct {
	Alias  string            `json:"alias"`
	Config map[string]string `json:"config"`
}) {
	t.Helper()
	const sub = "nava-operator-direct-grant-role-check"
	for _, f := range flows {
		switch f.Alias {
		case "nava-operator-direct-grant":
			pw, subAt := -1, -1
			for _, e := range f.Executions {
				switch {
				case e.Authenticator == "direct-grant-validate-password":
					pw = e.Priority
					if e.Requirement != "REQUIRED" {
						t.Errorf("password execution requirement = %s, want REQUIRED", e.Requirement)
					}
				case e.FlowAlias == sub:
					subAt = e.Priority
					if e.Requirement != "CONDITIONAL" {
						t.Errorf("role-check subflow requirement = %s, want CONDITIONAL", e.Requirement)
					}
				}
			}
			if pw < 0 || subAt < 0 || subAt <= pw {
				t.Errorf("role-check subflow (priority %d) must come after Password (priority %d)", subAt, pw)
			}
		case sub:
			want := []string{"conditional-user-role", "deny-access-authenticator"}
			if len(f.Executions) != len(want) {
				t.Fatalf("role-check executions = %+v, want exactly %v", f.Executions, want)
			}
			for i, e := range f.Executions {
				if e.Authenticator != want[i] || e.Requirement != "REQUIRED" {
					t.Errorf("role-check execution %d = %s %s, want %s REQUIRED", i, e.Authenticator, e.Requirement, want[i])
				}
				if i > 0 && e.Priority <= f.Executions[i-1].Priority {
					t.Errorf("role-check execution %d is not ordered after %d", i, i-1)
				}
			}
			if f.Executions[0].AuthenticatorConfig != "nava-deny-non-operator" {
				t.Errorf("conditional-user-role config = %q, want nava-deny-non-operator", f.Executions[0].AuthenticatorConfig)
			}
		}
	}
	found := false
	for _, c := range cfgs {
		if c.Alias != "nava-deny-non-operator" {
			continue
		}
		found = true
		if len(c.Config) != 2 || c.Config["condUserRole"] != "nava-operator" || c.Config["negate"] != "true" {
			t.Errorf("nava-deny-non-operator = %v, want {condUserRole: nava-operator, negate: true}", c.Config)
		}
	}
	if !found {
		t.Error("authenticatorConfig nava-deny-non-operator missing")
	}
}
