//go:build identitystack

package identitystack

// The pinned negative fixture of CI-5.06 (T19). Keycloak 26.0.0-26.1.2 let
// the organization picker accept kc.org for an organization the user is not
// a member of, and put that organization in the token (ADR-001 D26, TA-B1,
// docs/pins.md). This test runs the same orgPicker flow CI-5.06 runs, against
// a Keycloak 26.1.2, and requires the tamper to SUCCEED there: it proves the
// case can see the break, so its pass on the pinned Keycloak means something.
//
// Skipped by default: it needs its own Keycloak. make identity-attacks-fixture
// starts the negative-fixture Keycloak docs/pins.md names and sets
// NAVA_CI5_KC_FIXTURE_URL.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// fixtureKeycloak is the version the fixture must run on.
const fixtureKeycloak = "26.1.2"

func TestKC2612FixtureOrgPickerTamperSucceeds(t *testing.T) {
	base := strings.TrimRight(os.Getenv("NAVA_CI5_KC_FIXTURE_URL"), "/")
	if base == "" {
		t.Skip("negative fixture for CI-5.06: run make identity-attacks-fixture (Keycloak " + fixtureKeycloak + ")")
	}
	c := newJarClient(t)
	code, out := fixtureForm(t, c, base+"/realms/master/protocol/openid-connect/token", url.Values{
		"grant_type": {"password"}, "client_id": {"admin-cli"}, "username": {"admin"}, "password": {"dev-only-admin"},
	})
	tok, _ := out["access_token"].(string)
	if code != http.StatusOK || tok == "" {
		t.Fatalf("master admin token: %d %v", code, out)
	}
	call := func(method, path string, body any) (int, []byte) {
		t.Helper()
		var rdr io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		}
		req, _ := http.NewRequestWithContext(context.Background(), method, base+"/admin"+path, rdr)
		req.Header.Set("Authorization", "Bearer "+tok)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return do(t, c, req)
	}
	_, b := call(http.MethodGet, "/serverinfo", nil)
	var info struct {
		SystemInfo struct{ Version string } `json:"systemInfo"`
	}
	if json.Unmarshal(b, &info) != nil || info.SystemInfo.Version != fixtureKeycloak {
		t.Fatalf("the fixture must run on Keycloak %s, got %q", fixtureKeycloak, info.SystemInfo.Version)
	}

	// A realm shaped like nava where it matters: organizations on, a public
	// client with the organization scope, a user in two organizations.
	realm := "ci5-fixture-" + uuid.NewString()[:8]
	must := func(want int, method, path string, body any) []byte {
		t.Helper()
		code, b := call(method, path, body)
		if code != want {
			t.Fatalf("%s %s: %d %s", method, path, code, b)
		}
		return b
	}
	must(http.StatusCreated, http.MethodPost, "/realms", map[string]any{"realm": realm, "enabled": true, "organizationsEnabled": true})
	t.Cleanup(func() { call(http.MethodDelete, "/realms/"+realm, nil) })
	must(http.StatusCreated, http.MethodPost, "/realms/"+realm+"/clients", map[string]any{
		"clientId": "ci5-web", "publicClient": true, "standardFlowEnabled": true, "directAccessGrantsEnabled": true,
		"redirectUris": []string{"http://localhost:*"},
	})
	var clients []struct{ ID string }
	_ = json.Unmarshal(must(http.StatusOK, http.MethodGet, "/realms/"+realm+"/clients?clientId=ci5-web", nil), &clients)
	var scopes []struct{ ID, Name string }
	_ = json.Unmarshal(must(http.StatusOK, http.MethodGet, "/realms/"+realm+"/client-scopes", nil), &scopes)
	i := slices.IndexFunc(scopes, func(s struct{ ID, Name string }) bool { return s.Name == "organization" })
	if len(clients) != 1 || i < 0 {
		t.Fatalf("client or organization scope missing: %v %v", clients, scopes)
	}
	must(http.StatusNoContent, http.MethodPut, "/realms/"+realm+"/clients/"+clients[0].ID+"/optional-client-scopes/"+scopes[i].ID, nil)
	// The organization mapper as realm nava pins it (ADR-001 §3): ids, JSON.
	var mappers []map[string]any
	_ = json.Unmarshal(must(http.StatusOK, http.MethodGet, "/realms/"+realm+"/client-scopes/"+scopes[i].ID+"/protocol-mappers/models", nil), &mappers)
	for _, pm := range mappers {
		if pm["protocolMapper"] != "oidc-organization-membership-mapper" {
			continue
		}
		cfg, _ := pm["config"].(map[string]any)
		if cfg == nil {
			cfg = map[string]any{}
		}
		cfg["addOrganizationId"], cfg["addOrganizationAttributes"], cfg["jsonType.label"], cfg["multivalued"] = "true", "false", "JSON", "true"
		pm["config"] = cfg
		must(http.StatusNoContent, http.MethodPut, "/realms/"+realm+"/client-scopes/"+scopes[i].ID+"/protocol-mappers/models/"+pm["id"].(string), pm)
	}

	orgs := map[string]string{}
	for _, alias := range []string{"home", "decoy", "target"} {
		must(http.StatusCreated, http.MethodPost, "/realms/"+realm+"/organizations", map[string]any{
			"name": alias, "alias": alias, "enabled": true, "domains": []map[string]any{{"name": alias + ".invalid"}},
		})
	}
	var orgList []struct{ ID, Alias string }
	_ = json.Unmarshal(must(http.StatusOK, http.MethodGet, "/realms/"+realm+"/organizations", nil), &orgList)
	for _, o := range orgList {
		orgs[o.Alias] = o.ID
	}
	must(http.StatusCreated, http.MethodPost, "/realms/"+realm+"/users", map[string]any{
		"username": "mallory", "email": "mallory@home.invalid", "emailVerified": true, "enabled": true, "firstName": "Mal", "lastName": "Lory",
		"credentials": []map[string]any{{"type": "password", "value": "dev-only-mallory", "temporary": false}},
	})
	var users []struct{ ID string }
	_ = json.Unmarshal(must(http.StatusOK, http.MethodGet, "/realms/"+realm+"/users?exact=true&username=mallory", nil), &users)
	if len(users) != 1 {
		t.Fatalf("user mallory: %v", users)
	}
	for _, alias := range []string{"home", "decoy"} {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
			base+"/admin/realms/"+realm+"/organizations/"+orgs[alias]+"/members", strings.NewReader(`"`+users[0].ID+`"`))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		if code, b := do(t, c, req); code != http.StatusCreated {
			t.Fatalf("add mallory to %s: %d %s", alias, code, b)
		}
	}

	res := orgPicker(t, newJarClient(t), base, realm, "ci5-web", "mallory", "dev-only-mallory", "target")
	if !res.Shown {
		t.Fatalf("no organization picker on %s, so the fixture never tried the tamper: %.600s", fixtureKeycloak, res.Page)
	}
	if res.Token == "" {
		t.Fatalf("the kc.org tamper got no token on Keycloak %s: CI-5.06 is not shown to detect it: %.600s", fixtureKeycloak, res.Page)
	}
	if ids := orgIDs(t, res.Token); !slices.Contains(ids, orgs["target"]) {
		t.Fatalf("the kc.org tamper did not reach the target on Keycloak %s (organizations %s): CI-5.06 is not shown to detect it", fixtureKeycloak, describeOrgs(ids))
	}
	t.Logf("as expected on Keycloak %s: picker offered %v, kc.org=target gave a token for the non-member organization target", fixtureKeycloak, res.Choices)
}

func fixtureForm(t *testing.T, c *http.Client, u string, form url.Values) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, u, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body := do(t, c, req)
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	return code, out
}
