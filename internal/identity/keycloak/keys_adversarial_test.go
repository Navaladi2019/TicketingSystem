package keycloak

import (
	"strings"
	"testing"
)

// Adversarial tests for T15a (TICK-152, D-P10): a hostile discovery document
// or a redirect from an allow-listed IdP, or a proxy in front of it, cannot
// steer key fetches to keys other than the issuer's own.

func TestJWKSURIUnderIssuer(t *testing.T) {
	t.Parallel()
	const iss = "https://id.example.test/realms/nava"
	for _, tc := range []struct {
		uri  string
		want bool
	}{
		{iss + "/protocol/openid-connect/certs", true},
		{iss + "/certs", true},
		{iss, false},
		{iss + "/", false},
		{iss + "x/certs", false},
		{"http://id.example.test/realms/nava/protocol/openid-connect/certs", false},
		{"HTTPS://id.example.test/realms/nava/certs", false},
		{"https://other.example.test/realms/nava/certs", false},
		{"https://id.example.test:8443/realms/nava/certs", false},
		{iss + "/%2e%2e/other/certs", false},
		{iss + "/%2E%2E/other/certs", false},
		{iss + "/protocol%2fopenid-connect/certs", false},
		{iss + "/%2fother/certs", false},
		{iss + "/../other/certs", false},
		{iss + "/./certs/..", false},
		{iss + "/certs?x=1", false},
		{iss + "/certs?", false},
		{iss + "/certs#frag", false},
		{iss + "/certs@evil.test", false},
		{iss + `\..\nava2\certs`, false},
		{"https://user@id.example.test/realms/nava/certs", false},
		{iss + "/%zz", false},
	} {
		if got := jwksUnderIssuer(iss, tc.uri); got != tc.want {
			t.Errorf("jwksUnderIssuer(%q) = %v, want %v", tc.uri, got, tc.want)
		}
	}
}

// An https issuer whose discovery document names an http jwks_uri on the same
// host is rejected before any JWKS fetch.
func TestDiscoveryDowngradeToHTTPRejected(t *testing.T) {
	t.Parallel()
	idp := newFakeTLSIdP(t, realmNava)
	idp.discoveryHook = func(_ string, d map[string]any) {
		d["jwks_uri"] = "http://" + strings.TrimPrefix(d["jwks_uri"].(string), "https://")
	}
	h := newHarnessFor(t, idp, []string{realmNava})
	if !strings.HasPrefix(idp.issuer(realmNava), "https://") {
		t.Fatalf("issuer %q is not https", idp.issuer(realmNava))
	}
	_, err := h.a.verify(t.Context(), sign(t, idp.signer(realmNava), idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience)
	wantInvalid(t, err)
	if n := idp.discovery.Load(); n != 1 {
		t.Fatalf("%d discovery fetches, want 1", n)
	}
	// An http fetch would open a second connection: Go's TLS server answers
	// it with a 400 before the handler, so jwksHits alone cannot show it.
	if n := idp.conns.Load(); n != 1 {
		t.Fatalf("%d connections, want 1 (discovery only): the http jwks_uri was fetched", n)
	}
	if n := idp.jwksHits.Load(); n != 0 {
		t.Fatalf("%d JWKS fetches after an http jwks_uri", n)
	}
	// The same harness accepts the token once the document is honest, so the
	// rejection above was the scheme, not the TLS setup.
	honest := newFakeTLSIdP(t, realmNava)
	h2 := newHarnessFor(t, honest, []string{realmNava})
	if _, err := h2.a.verify(t.Context(), sign(t, honest.signer(realmNava), honest.tenantClaims(realmNava, h2.clock.now())), DefaultAudience); err != nil {
		t.Fatalf("token from an honest https IdP rejected: %v", err)
	}
}

// A 302 from the discovery or JWKS URL to another host is not followed: the
// other host, which serves its own keys under the same kid, is never asked,
// and the token signed with its key fails closed.
func TestCrossHostRedirectNotFollowed(t *testing.T) {
	t.Parallel()
	for name, path := range map[string]string{
		"discovery": "/realms/" + realmNava + "/.well-known/openid-configuration",
		"jwks":      jwksPath(realmNava),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			idp := newFakeIdP(t, realmNava)
			evil := newFakeIdP(t, realmNava) // its own nava-k1, a different key
			evil.discoveryHook = func(_ string, d map[string]any) {
				// Claim the trusted issuer and point at the evil keys.
				d["issuer"] = idp.issuer(realmNava)
				d["jwks_uri"] = idp.issuer(realmNava) + "/protocol/openid-connect/certs"
			}
			idp.redirect = func(p string) string {
				if p == path {
					return evil.srv.URL + p
				}
				return ""
			}
			h := newHarnessFor(t, idp, []string{realmNava})
			forged := sign(t, evil.signer(realmNava), idp.tenantClaims(realmNava, h.clock.now()))
			_, err := h.a.verify(t.Context(), forged, DefaultAudience)
			wantInvalid(t, err)
			if !strings.Contains(err.Error(), "keys unavailable") {
				t.Errorf("err = %v, want keys unavailable", err)
			}
			if n := evil.requests.Load(); n != 0 {
				t.Fatalf("redirect followed: %d requests reached the other host", n)
			}
			if n := idp.hitsOn(path); n != 1 {
				t.Fatalf("%d requests for %s, want 1", n, path)
			}
		})
	}
}

// Policy: discovery and JWKS fetches follow no redirect, same-origin
// included. A same-host redirect from realm nava's JWKS to realm other's
// would hand nava's tokens another realm's keys, the very thing the jwks_uri
// path check forbids.
func TestSameOriginRedirectNotFollowed(t *testing.T) {
	t.Parallel()
	idp := newFakeIdP(t, realmNava, "other")
	idp.redirect = func(p string) string {
		if p == jwksPath(realmNava) {
			return jwksPath("other")
		}
		return ""
	}
	h := newHarnessFor(t, idp, []string{realmNava})
	forged := sign(t, idp.signer("other"), idp.tenantClaims(realmNava, h.clock.now()))
	_, err := h.a.verify(t.Context(), forged, DefaultAudience)
	wantInvalid(t, err)
	if n := idp.hitsOn(jwksPath("other")); n != 0 {
		t.Fatalf("same-origin redirect followed: %d requests for realm other's JWKS", n)
	}
	// Nava's own key fails closed too: the set never loaded.
	h.clock.advance(forceInterval)
	_, err = h.a.verify(t.Context(), sign(t, idp.signer(realmNava), idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience)
	wantInvalid(t, err)
}

// A jwks_uri on another live host, serving keys under the issuer's kid, is
// rejected and that host is never asked.
func TestDiscoveryJWKSOnOtherHostNotFetched(t *testing.T) {
	t.Parallel()
	idp := newFakeIdP(t, realmNava)
	evil := newFakeIdP(t, realmNava)
	idp.discoveryHook = func(_ string, d map[string]any) {
		d["jwks_uri"] = evil.srv.URL + jwksPath(realmNava)
	}
	h := newHarnessFor(t, idp, []string{realmNava})
	forged := sign(t, evil.signer(realmNava), idp.tenantClaims(realmNava, h.clock.now()))
	_, err := h.a.verify(t.Context(), forged, DefaultAudience)
	wantInvalid(t, err)
	if n := evil.requests.Load(); n != 0 {
		t.Fatalf("%d requests reached the other host", n)
	}
}
