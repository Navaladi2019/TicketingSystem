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
// here, through the admin API, and leave an existing realm alone.
var testRealms = []string{"idp-tenant-a", "idp-tenant-b"}

func TestMain(m *testing.M) {
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
		req, _ := http.NewRequest(http.MethodPost, baseURL()+"/admin/realms", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		req.Header.Set("Content-Type", "application/json")
		r, err := c.Do(req)
		if err != nil {
			return fmt.Errorf("import %s: %w", realm, err)
		}
		b, _ := io.ReadAll(r.Body)
		r.Body.Close()
		// 409: already there (an earlier run on this stack).
		if r.StatusCode != http.StatusCreated && r.StatusCode != http.StatusConflict {
			return fmt.Errorf("import %s: %d %s", realm, r.StatusCode, strings.TrimSpace(string(b)))
		}
	}
	return nil
}
