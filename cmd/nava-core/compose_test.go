package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TestComposePublishesNoAdminPort: the admin listener has no auth, so no
// compose file may publish its port. The only mention allowed is the
// container's own healthcheck URL.
func TestComposePublishesNoAdminPort(t *testing.T) {
	t.Parallel()
	for _, f := range []string{"../../compose.yaml", "../../deploy/compose/compose.internal.yaml"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			code, _, _ := strings.Cut(line, "#")
			if strings.Contains(code, "9090") && !strings.Contains(code, "http://127.0.0.1:9090/") {
				t.Errorf("%s:%d mentions the admin port: %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// composeEnv is a service's environment from a compose file, with
// ${VAR:-default} resolved to its default.
func composeEnv(t *testing.T, file, service string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Services map[string]struct {
			Environment map[string]any `yaml:"environment"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for k, v := range doc.Services[service].Environment {
		s := fmt.Sprint(v)
		out[k] = composeDefault.ReplaceAllString(s, "$1")
	}
	return out
}

var composeDefault = regexp.MustCompile(`\$\{[A-Za-z_]+:-([^}]*)\}`)

// TestComposeKeycloakIssuerPinned (TICK-160): Keycloak's issuer is a fixed
// URL, not the request Host, and nava-core trusts exactly that one issuer.
// Proxy headers are on only with trusted proxy addresses.
func TestComposeKeycloakIssuerPinned(t *testing.T) {
	t.Parallel()
	kc := composeEnv(t, "../../compose.yaml", "keycloak")
	host := kc["KC_HOSTNAME"]
	u, err := url.Parse(host)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Path != "" {
		t.Fatalf("KC_HOSTNAME = %q, want the full public URL (scheme://host[:port])", host)
	}
	if kc["KC_HOSTNAME_STRICT"] != "true" {
		t.Errorf("KC_HOSTNAME_STRICT = %q, want true", kc["KC_HOSTNAME_STRICT"])
	}
	// The host is what a browser reaches: front-channel URLs (login redirects)
	// are built from it. It must be the published port, not a service name.
	if got, want := u.Host, "localhost:"+publishedKeycloakPort(t); got != want {
		t.Errorf("KC_HOSTNAME host = %q, want the host-published address %q", got, want)
	}
	// Dynamic back-channel URLs keep discovery's token and JWKS endpoints on
	// the host the caller used, so they stay reachable inside the network.
	// iss does not follow it: it stays KC_HOSTNAME.
	if kc["KC_HOSTNAME_BACKCHANNEL_DYNAMIC"] != "true" {
		t.Errorf("KC_HOSTNAME_BACKCHANNEL_DYNAMIC = %q, want true", kc["KC_HOSTNAME_BACKCHANNEL_DYNAMIC"])
	}
	core := composeEnv(t, "../../compose.yaml", "nava-core")
	if got, want := core["NAVA_OIDC_ISSUERS"], host+"/realms/nava"; got != want {
		t.Errorf("NAVA_OIDC_ISSUERS = %q, want exactly %q", got, want)
	}
	fb, err := url.Parse(core["NAVA_OIDC_FETCH_BASE_URL"])
	if err != nil || fb.Host == "" || (fb.Path != "" && fb.Path != "/") {
		t.Errorf("NAVA_OIDC_FETCH_BASE_URL = %q, want scheme://host[:port]", core["NAVA_OIDC_FETCH_BASE_URL"])
	} else if fb.Host == u.Host {
		t.Error("NAVA_OIDC_FETCH_BASE_URL is the browser address: nava-core cannot reach it from inside the network")
	}
	// nava-onboard also validates tokens: it trusts the same one issuer,
	// fetches keys from the back channel, and writes that issuer to the
	// tenant binding (TICK-278).
	ob := composeEnv(t, "../../compose.yaml", "nava-onboard")
	if got, want := ob["NAVA_OIDC_ISSUERS"], host+"/realms/nava"; got != want {
		t.Errorf("nava-onboard NAVA_OIDC_ISSUERS = %q, want exactly %q", got, want)
	}
	if got := ob["NAVA_OIDC_FETCH_BASE_URL"]; got != core["NAVA_OIDC_FETCH_BASE_URL"] {
		t.Errorf("nava-onboard NAVA_OIDC_FETCH_BASE_URL = %q, want %q", got, core["NAVA_OIDC_FETCH_BASE_URL"])
	}
	if got, want := ob["NAVA_ONBOARD_ISSUER"], host+"/realms/nava"; got != want {
		t.Errorf("NAVA_ONBOARD_ISSUER = %q, want the pinned issuer %q", got, want)
	}
	if iss, _, _ := strings.Cut(ob["NAVA_OPERATORS"], "|"); iss != host+"/realms/nava" {
		t.Errorf("NAVA_OPERATORS issuer = %q, want the pinned issuer %q", iss, host+"/realms/nava")
	}
}

// publishedKeycloakPort is the host port compose.yaml publishes Keycloak on
// (KEYCLOAK_PORT's default), the one KC_HOSTNAME must name.
func publishedKeycloakPort(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Services map[string]struct {
			Ports []string `yaml:"ports"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	for _, p := range doc.Services["keycloak"].Ports {
		p = composeDefault.ReplaceAllString(p, "$1")
		if parts := strings.Split(p, ":"); len(parts) == 3 && parts[2] == "8080" {
			return parts[1]
		}
	}
	t.Fatal("keycloak publishes no port for 8080")
	return ""
}

// TestKeycloakProxyHeadersNeedTrustedAddresses: a proxy-header setting without
// trusted proxy addresses lets any client forge X-Forwarded-Host.
func TestKeycloakProxyHeadersNeedTrustedAddresses(t *testing.T) {
	t.Parallel()
	if err := checkKeycloakProxy(composeEnv(t, "../../compose.yaml", "keycloak")); err != nil {
		t.Error(err)
	}
	cases := []struct {
		name string
		env  map[string]string
		bad  bool
	}{
		{"none", map[string]string{}, false},
		{"headers with trusted", map[string]string{"KC_PROXY_HEADERS": "xforwarded", "KC_PROXY_TRUSTED_ADDRESSES": "10.0.0.0/8"}, false},
		{"headers alone", map[string]string{"KC_PROXY_HEADERS": "xforwarded"}, true},
		{"forwarded alone", map[string]string{"KC_PROXY_HEADERS": "forwarded"}, true},
		{"legacy proxy alone", map[string]string{"KC_PROXY": "edge"}, true},
		{"trusted alone", map[string]string{"KC_PROXY_TRUSTED_ADDRESSES": "10.0.0.0/8"}, true},
		{"empty trusted", map[string]string{"KC_PROXY_HEADERS": "xforwarded", "KC_PROXY_TRUSTED_ADDRESSES": " "}, true},
	}
	for _, c := range cases {
		if err := checkKeycloakProxy(c.env); (err != nil) != c.bad {
			t.Errorf("%s: err = %v, want error %v", c.name, err, c.bad)
		}
	}
}

func checkKeycloakProxy(env map[string]string) error {
	headers := strings.TrimSpace(env["KC_PROXY_HEADERS"]) != "" || strings.TrimSpace(env["KC_PROXY"]) != ""
	trusted := strings.TrimSpace(env["KC_PROXY_TRUSTED_ADDRESSES"]) != ""
	if headers != trusted {
		return errors.New("KC_PROXY_HEADERS and KC_PROXY_TRUSTED_ADDRESSES must be set together")
	}
	return nil
}
