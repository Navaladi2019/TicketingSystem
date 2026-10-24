package identitystack

import (
	"encoding/json"
	"os"
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
		Realm                string `json:"realm"`
		OrganizationsEnabled bool   `json:"organizationsEnabled"`
		AccessTokenLifespan  int    `json:"accessTokenLifespan"`
		ClientScopes         []struct {
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
			Alias      string `json:"alias"`
			Executions []struct {
				Authenticator string `json:"authenticator"`
				Requirement   string `json:"requirement"`
			} `json:"authenticationExecutions"`
		} `json:"authenticationFlows"`
		Organizations []struct {
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
				if c["addOrganizationId"] != "true" || c["addOrganizationAttributes"] != "false" || c["jsonType.label"] != "JSON" || c["access.token.claim"] != "true" {
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
	flows := map[string]bool{}
	for _, f := range r.Flows {
		flows[f.Alias] = true
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
	for _, o := range r.Organizations {
		for _, d := range o.Domains {
			if !d.Verified {
				t.Errorf("organization %s has an unverified domain", o.Alias)
			}
		}
	}
}
