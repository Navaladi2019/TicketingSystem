package keycloak

import "testing"

// Security review of T15a (TICK-152): a redirect that appears only after the
// first JWKS fetch succeeded, on the refresh path an unknown kid forces, must
// not load another realm's keys. The forced refresh and httprc's background
// refresh both send through the resource's routedClient, so this also guards
// the background path.
func TestRefreshRedirectToOtherRealmNotFollowed(t *testing.T) {
	t.Parallel()
	idp := newFakeIdP(t, realmNava, "other")
	h := newHarnessFor(t, idp, []string{realmNava})
	if _, err := h.a.verify(t.Context(), sign(t, idp.signer(realmNava), idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatalf("honest token rejected: %v", err)
	}
	idp.redirect = func(p string) string {
		if p == jwksPath(realmNava) {
			return jwksPath("other")
		}
		return ""
	}
	h.clock.advance(forceInterval)
	forged := sign(t, idp.signer("other"), idp.tenantClaims(realmNava, h.clock.now()))
	_, err := h.a.verify(t.Context(), forged, DefaultAudience)
	wantInvalid(t, err)
	if n := idp.hitsOn(jwksPath("other")); n != 0 {
		t.Fatalf("refresh followed a same-origin redirect: %d requests for realm other's JWKS", n)
	}
	if n := idp.hitsOn(jwksPath(realmNava)); n != 2 {
		t.Fatalf("%d JWKS requests for nava, want 2 (initial + forced refresh)", n)
	}
	// The keys nava had stay usable: a refused refresh does not wipe them.
	if _, err := h.a.verify(t.Context(), sign(t, idp.signer(realmNava), idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatalf("nava's cached key lost after a refused refresh: %v", err)
	}
}

func TestJWKSURIUnderIssuerExtra(t *testing.T) {
	t.Parallel()
	const iss = "https://id.example.test/realms/nava"
	for _, uri := range []string{
		"https://ID.example.test/realms/nava/certs",
		"https://id.example.test./realms/nava/certs",
		"https://id.example.test:443/realms/nava/certs",
		iss + "/certs/%2e",
		iss + "/%252e%252e/other/certs",
		iss + "/..;/other/certs",
		iss + "//other.test/certs",
		iss + "/certs\t",
		iss + "/certs\r\nHost: evil",
		"https://id.example.test/realms/nava2/certs",
		"https://id.example.test/realms/NAVA/certs",
		iss + "/certs%25",
		iss + "/protocol//openid-connect/certs",
		iss + "/certs//",
	} {
		// %252e%252e and // are same-host and under the issuer as Go sends
		// them, but a proxy that decodes twice or merges slashes would not
		// see them so: rejected outright.
		if jwksUnderIssuer(iss, uri) {
			t.Errorf("jwksUnderIssuer(%q) = true, want false", uri)
		}
	}
}
