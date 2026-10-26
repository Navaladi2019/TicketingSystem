//go:build identitystack

package identitystack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testRealms are the two test-only IdP realms the tenant IdPs broker to. The
// stack does not import them at first boot (TICK-166: every realm imported
// there lengthens Keycloak's start); the tests that need them create them
// here, through the admin API, fresh on every run: delete (204, or 404 when
// absent), then create (201 required). `make stack-idp-realms` runs the same code.
var testRealms = []string{"idp-tenant-a", "idp-tenant-b"}

func TestMain(m *testing.M) {
	// make identity-attacks-fixture runs only the CI-5.06 negative fixture,
	// against its own Keycloak 26.1.2 and with no stack to import into.
	if os.Getenv("NAVA_CI5_KC_FIXTURE_URL") != "" {
		os.Exit(m.Run())
	}
	if err := importTestRealms(); err != nil {
		fmt.Fprintln(os.Stderr, "identitystack:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func importTestRealms() error {
	c := &http.Client{Timeout: 60 * time.Second}
	user, pass := os.Getenv("KEYCLOAK_ADMIN"), os.Getenv("KEYCLOAK_ADMIN_PASSWORD")
	if user == "" {
		user = "admin"
	}
	if pass == "" {
		pass = "dev-only-admin"
	}
	resp, err := c.PostForm(baseURL()+"/realms/master/protocol/openid-connect/token", url.Values{
		"grant_type": {"password"}, "client_id": {"admin-cli"}, "username": {user}, "password": {pass},
	})
	if err != nil {
		return fmt.Errorf("master admin token: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if resp.StatusCode != http.StatusOK || json.Unmarshal(body, &tok) != nil || tok.AccessToken == "" {
		return fmt.Errorf("master admin token: %d %s", resp.StatusCode, body)
	}
	for _, realm := range testRealms {
		raw, err := os.ReadFile(filepath.Join("..", "..", "deploy", "compose", "keycloak", "test-realms", "realm-"+realm+".json"))
		if err != nil {
			return err
		}
		del, _ := http.NewRequest(http.MethodDelete, baseURL()+"/admin/realms/"+realm, nil)
		del.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		dr, err := c.Do(del)
		if err != nil {
			return fmt.Errorf("delete %s: %w", realm, err)
		}
		io.Copy(io.Discard, dr.Body)
		dr.Body.Close()
		if dr.StatusCode != http.StatusNoContent && dr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("delete %s: %d", realm, dr.StatusCode)
		}
		if err := dropLinkedUsers(c, tok.AccessToken, realm); err != nil {
			return err
		}
		req, _ := http.NewRequest(http.MethodPost, baseURL()+"/admin/realms", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		req.Header.Set("Content-Type", "application/json")
		r, err := c.Do(req)
		if err != nil {
			return fmt.Errorf("import %s: %w", realm, err)
		}
		b, _ := io.ReadAll(r.Body)
		r.Body.Close()
		if r.StatusCode != http.StatusCreated {
			return fmt.Errorf("import %s: %d %s", realm, r.StatusCode, strings.TrimSpace(string(b)))
		}
	}
	return nil
}

// TestIdPRealmsExist is what `make stack-idp-realms` runs: TestMain has just
// created the realms, this checks that each one answers.
func TestIdPRealmsExist(t *testing.T) {
	for _, realm := range testRealms {
		resp, err := http.Get(baseURL() + "/realms/" + realm + "/.well-known/openid-configuration")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("realm %s: %d", realm, resp.StatusCode)
		}
	}
}

// dropLinkedUsers deletes the users of realm `nava` that earlier brokered
// logins created from the IdP realm (its alias in `nava` is the realm name). A
// recreated IdP realm gives its users new ids, and a first broker login would
// otherwise meet the old linked user and fail with a conflict.
func dropLinkedUsers(c *http.Client, token, idpAlias string) error {
	do := func(method, path string) (int, []byte, error) {
		req, _ := http.NewRequest(method, baseURL()+"/admin/realms/nava"+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r, err := c.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body)
		return r.StatusCode, b, nil
	}
	code, b, err := do(http.MethodGet, "/users?max=1000&idpAlias="+url.QueryEscape(idpAlias))
	if err != nil || code != http.StatusOK {
		return fmt.Errorf("list nava users linked to %s: %d %s %v", idpAlias, code, b, err)
	}
	var us []struct{ ID string }
	if err := json.Unmarshal(b, &us); err != nil {
		return err
	}
	for _, u := range us {
		if code, b, err := do(http.MethodDelete, "/users/"+u.ID); err != nil || code != http.StatusNoContent {
			return fmt.Errorf("delete nava user %s: %d %s %v", u.ID, code, b, err)
		}
	}
	return nil
}
