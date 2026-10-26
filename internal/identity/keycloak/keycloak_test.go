package keycloak

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

const realmNava = "nava"

func wantInvalid(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func b64(v any) string {
	b, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(b)
}

// TestVerifyMatrix is the A.1 step 1 matrix: each case is a valid tenant
// token with one thing changed.
func TestVerifyMatrix(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	now := h.clock.now()
	s := h.idp.signer(realmNava)
	valid := func() claims { return h.idp.tenantClaims(realmNava, now) }
	with := func(k string, v any) string { c := valid(); c[k] = v; return sign(t, s, c) }
	without := func(k string) string { c := valid(); delete(c, k); return sign(t, s, c) }

	pub, err := x509.MarshalPKIXPublicKey(&s.priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	body := b64(valid())
	hsInput := b64(map[string]any{"alg": "HS256", "kid": s.kid, "typ": "JWT"}) + "." + body
	mac := hmac.New(sha256.New, pub)
	mac.Write([]byte(hsInput))
	hs256WithPublicKey := hsInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	cases := []struct {
		name  string
		token string
		ok    bool
	}{
		{"valid", sign(t, s, valid()), true},
		{"alg none", b64(map[string]any{"alg": "none", "kid": s.kid}) + "." + body + ".", false},
		{"HS256 signed with the RS public key", hs256WithPublicKey, false},
		{"unknown kid", sign(t, signer{kid: "nobody", priv: s.priv}, valid()), false},
		{"wrong iss", with("iss", h.idp.issuer("other")), false},
		{"wrong aud", with("aud", []string{"account"}), false},
		{"operator aud only", with("aud", []string{DefaultOperatorAudience}), false},
		{"no exp", without("exp"), false},
		{"exp 31 s ago", with("exp", now.Add(-31*time.Second).Unix()), false},
		{"exp 29 s ago", with("exp", now.Add(-29*time.Second).Unix()), true},
		{"nbf 31 s ahead", with("nbf", now.Add(31*time.Second).Unix()), false},
		{"nbf 29 s ahead", with("nbf", now.Add(29*time.Second).Unix()), true},
		{"typ ID (an ID token)", with("typ", "ID"), false},
		{"typ Refresh", with("typ", "Refresh"), false},
		{"no typ", without("typ"), false},
		{"no azp", without("azp"), false},
		{"no sub", without("sub"), false},
		{"tampered body", strings.Replace(sign(t, s, valid()), ".", "."+b64(map[string]any{"iss": "x"})[:4], 1), false},
		{"JSON serialisation", `{"payload":"` + body + `"}`, false},
		{"empty", "", false},
		{"oversized", strings.Repeat("a", maxTokenBytes+1), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := h.a.verify(t.Context(), c.token, DefaultAudience)
			if c.ok {
				if err != nil {
					t.Fatalf("rejected a valid token: %v", err)
				}
				return
			}
			wantInvalid(t, err)
		})
	}
}

// An ID token as Keycloak issues it: typ ID, aud the client, no Bearer.
func TestIDTokenRejected(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	c := h.idp.tenantClaims(realmNava, h.clock.now())
	c["typ"], c["aud"] = "ID", "nava-web"
	_, err := h.a.Resolve(t.Context(), sign(t, h.idp.signer(realmNava), c))
	wantInvalid(t, err)
	c["aud"] = []string{"nava-web", DefaultAudience} // the audience alone does not let it through
	_, err = h.a.Resolve(t.Context(), sign(t, h.idp.signer(realmNava), c))
	wantInvalid(t, err)
}

// A key of realm X never verifies a token that says iss=Y, even when both
// realms are trusted.
func TestRealmKeyCannotSignForAnotherIssuer(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{"x", "y"})
	c := h.idp.tenantClaims("y", h.clock.now())
	_, err := h.a.verify(t.Context(), sign(t, h.idp.signer("x"), c), DefaultAudience)
	wantInvalid(t, err)
	// The same key with realm X's own issuer verifies.
	c["iss"] = h.idp.issuer("x")
	if _, err := h.a.verify(t.Context(), sign(t, h.idp.signer("x"), c), DefaultAudience); err != nil {
		t.Fatalf("realm X token rejected: %v", err)
	}
}

// An issuer off the allow-list is rejected before any HTTP request (SSRF,
// CI-5 "unknown issuer").
func TestUnknownIssuerMakesNoRequest(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava}, "evil")
	for _, iss := range []string{h.idp.issuer("evil"), h.idp.issuer(realmNava) + "/", "http://169.254.169.254/latest", ""} {
		c := h.idp.tenantClaims(realmNava, h.clock.now())
		c["iss"] = iss
		_, err := h.a.Resolve(t.Context(), sign(t, h.idp.signer("evil"), c))
		wantInvalid(t, err)
	}
	if n := h.idp.requests.Load(); n != 0 {
		t.Fatalf("%d outbound requests for untrusted issuers, want 0", n)
	}
}

// 100 tokens with unknown kids cause at most one JWKS fetch in 30 s, beyond
// the first one.
func TestUnknownKidsFetchAtMostOncePer30s(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	s := h.idp.signer(realmNava)
	if _, err := h.a.verify(t.Context(), sign(t, s, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatal(err)
	}
	h.clock.advance(forceInterval) // the gate is open again
	base := h.idp.jwksHits.Load()
	for i := range 100 {
		h.clock.advance(250 * time.Millisecond) // 100 tokens over 25 s
		tok := sign(t, signer{kid: fmt.Sprintf("random-%d", i), priv: s.priv}, h.idp.tenantClaims(realmNava, h.clock.now()))
		_, err := h.a.verify(t.Context(), tok, DefaultAudience)
		wantInvalid(t, err)
	}
	if n := h.idp.jwksHits.Load() - base; n > 1 {
		t.Fatalf("%d JWKS fetches for 100 unknown kids in 30 s, want ≤ 1", n)
	}
}

// Concurrent unknown kids share one fetch.
func TestConcurrentUnknownKidsShareOneFetch(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	s := h.idp.signer(realmNava)
	if _, err := h.a.verify(t.Context(), sign(t, s, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatal(err)
	}
	h.clock.advance(forceInterval)
	base := h.idp.jwksHits.Load()
	var wg sync.WaitGroup
	for i := range 50 {
		tok := sign(t, signer{kid: fmt.Sprintf("c-%d", i), priv: s.priv}, h.idp.tenantClaims(realmNava, h.clock.now()))
		wg.Go(func() {
			if _, err := h.a.verify(t.Context(), tok, DefaultAudience); !errors.Is(err, identity.ErrInvalidToken) {
				t.Errorf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
	wg.Wait()
	if n := h.idp.jwksHits.Load() - base; n != 1 {
		t.Fatalf("%d JWKS fetches for 50 concurrent unknown kids, want 1", n)
	}
}

// A newly rotated key is accepted without a restart.
func TestRotatedKeyAcceptedWithoutRestart(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	old := h.idp.signer(realmNava)
	if _, err := h.a.verify(t.Context(), sign(t, old, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatal(err)
	}
	h.clock.advance(forceInterval)
	rotated := h.idp.rotate(realmNava, "nava-k2")
	if _, err := h.a.verify(t.Context(), sign(t, rotated, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatalf("rotated key rejected: %v", err)
	}
	if _, err := h.a.verify(t.Context(), sign(t, old, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatalf("old key, still published, rejected: %v", err)
	}
}

// A key rotated just after a forced fetch is accepted once the 30 s window
// ends, and refused (not fetched) before.
func TestRotatedKeyWaitsForTheGate(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	if _, err := h.a.verify(t.Context(), sign(t, h.idp.signer(realmNava), h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatal(err)
	}
	rotated := h.idp.rotate(realmNava, "nava-k2")
	h.clock.advance(10 * time.Second)
	base := h.idp.jwksHits.Load()
	_, err := h.a.verify(t.Context(), sign(t, rotated, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience)
	wantInvalid(t, err)
	if n := h.idp.jwksHits.Load() - base; n != 0 {
		t.Fatalf("fetched %d times inside the 30 s window", n)
	}
	h.clock.advance(20 * time.Second)
	if _, err := h.a.verify(t.Context(), sign(t, rotated, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatalf("rotated key rejected after the window: %v", err)
	}
}

// The discovery document cannot redirect key fetches.
func TestDiscoveryDocumentChecked(t *testing.T) {
	t.Parallel()
	for name, hook := range map[string]func(string, map[string]any){
		"issuer mismatch":          func(_ string, d map[string]any) { d["issuer"] = "https://elsewhere.test/realms/nava" },
		"jwks_uri on another host": func(_ string, d map[string]any) { d["jwks_uri"] = "http://169.254.169.254/certs" },
		"jwks_uri in another realm": func(_ string, d map[string]any) {
			d["jwks_uri"] = d["issuer"].(string) + "/../other/protocol/openid-connect/certs"
		},
		"jwks_uri with userinfo": func(_ string, d map[string]any) { d["jwks_uri"] = d["issuer"].(string) + "/certs@evil.test" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			idp := newFakeIdP(t, realmNava, "other")
			idp.discoveryHook = hook
			h := newHarnessFor(t, idp, []string{realmNava})
			_, err := h.a.verify(t.Context(), sign(t, idp.signer(realmNava), idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience)
			wantInvalid(t, err)
			if n := idp.jwksHits.Load(); n != 0 {
				t.Fatalf("%d JWKS fetches after a bad discovery document", n)
			}
		})
	}
}

// An unreachable IdP fails closed, and is not retried inside 30 s.
func TestDiscoveryFailureThrottled(t *testing.T) {
	t.Parallel()
	idp := newFakeIdP(t, "other") // no realm nava: its discovery URL is a 404
	h := newHarnessFor(t, idp, []string{realmNava})
	tok := sign(t, newSigner(t, "k"), idp.tenantClaims(realmNava, h.clock.now()))
	for range 5 {
		_, err := h.a.verify(t.Context(), tok, DefaultAudience)
		wantInvalid(t, err)
	}
	if n := idp.discovery.Load(); n != 0 {
		t.Fatalf("discovery reached the handler %d times for an unknown realm", n)
	}
	if n := idp.requests.Load(); n != 1 {
		t.Fatalf("%d requests in 30 s for a failing issuer, want 1", n)
	}
}

func TestResolveOperator(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	s := h.idp.signer(realmNava)
	now := h.clock.now()

	op, err := h.a.ResolveOperator(t.Context(), sign(t, s, h.idp.operatorClaims(realmNava, now)))
	if err != nil {
		t.Fatalf("operator token rejected: %v", err)
	}
	if op.PrincipalID != "6f1c2a7e-0000-4000-8000-000000000001" {
		t.Errorf("PrincipalID = %q", op.PrincipalID)
	}

	noRole := h.idp.operatorClaims(realmNava, now) // the pre-TICK-57 hole: aud via nava-operator-cli
	noRole["realm_access"] = map[string]any{"roles": []string{"default-roles-nava"}}
	clientRole := h.idp.operatorClaims(realmNava, now) // a client role of the same name is not the realm role
	delete(clientRole, "realm_access")
	clientRole["resource_access"] = map[string]any{"nava-operator-api": map[string]any{"roles": []string{OperatorRole}}}
	roleNoAud := h.idp.operatorClaims(realmNava, now)
	roleNoAud["aud"] = []string{DefaultAudience}
	tenant := h.idp.tenantClaims(realmNava, now)
	unlisted := h.idp.operatorClaims(realmNava, now) // AR-030: role mapped to a user the platform did not list
	unlisted["sub"] = "6f1c2a7e-0000-4000-8000-0000000000ff"

	for name, c := range map[string]claims{
		"aud nava-operator-api, no nava-operator role": noRole,
		"role only as a client role":                   clientRole,
		"role but tenant audience":                     roleNoAud,
		"tenant token":                                 tenant,
		"role and audience, sub not allow-listed":      unlisted,
	} {
		_, err := h.a.ResolveOperator(t.Context(), sign(t, s, c))
		if !errors.Is(err, identity.ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
}

// An operator token through the tenant resolver never resolves a tenant. If
// it also carries the tenant audience it has no organization, so
// ErrTenantUnresolved; with only the operator audience it is not a nava-api
// token at all, so ErrInvalidToken (TICK-94 item 11: one audience per
// verifier). Both are 401.
func TestOperatorTokenThroughIdentityResolver(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	s := h.idp.signer(realmNava)
	c := h.idp.operatorClaims(realmNava, h.clock.now())
	c["aud"] = []string{DefaultOperatorAudience, DefaultAudience}
	if _, err := h.a.Resolve(t.Context(), sign(t, s, c)); !errors.Is(err, identity.ErrTenantUnresolved) {
		t.Errorf("both audiences: err = %v, want ErrTenantUnresolved", err)
	}
	_, err := h.a.Resolve(t.Context(), sign(t, s, h.idp.operatorClaims(realmNava, h.clock.now())))
	wantInvalid(t, err)
}

// Errors carry a fixed reason only: never token values or library text.
func TestErrorsCarryNoTokenValues(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	c := h.idp.tenantClaims(realmNava, h.clock.now())
	c["aud"] = []string{"secret-audience-value"}
	c["sub"] = "alice@tenant-a.test"
	_, err := h.a.Resolve(t.Context(), sign(t, h.idp.signer(realmNava), c))
	wantInvalid(t, err)
	for _, leak := range []string{"secret-audience-value", "alice", "nava-api", h.idp.srv.URL} {
		if strings.Contains(err.Error(), leak) {
			t.Errorf("error %q leaks %q", err, leak)
		}
	}
}

// JWKS and discovery spans are named by route template, never by the URL.
func TestFetchSpansNamedByTemplate(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	if _, err := h.a.verify(t.Context(), sign(t, h.idp.signer(realmNava), h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, s := range h.spans.GetSpans() {
		got[s.Name] = true
		if strings.Contains(s.Name, realmNava) || strings.Contains(s.Name, "127.0.0.1") {
			t.Errorf("span name %q carries the URL", s.Name)
		}
	}
	for _, want := range []string{"GET " + discoveryRoute, "GET " + jwksRoute} {
		if !got[want] {
			t.Errorf("no span %q; got %v", want, got)
		}
	}
}

// The operator allow-list is keyed by (iss, sub): the listed subject in
// another trusted realm is not an operator, and an adapter with no list
// refuses every operator token (AR-030).
func TestOperatorAllowList(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava, "other"})
	now := h.clock.now()
	other := sign(t, h.idp.signer("other"), h.idp.operatorClaims("other", now))
	if _, err := h.a.ResolveOperator(t.Context(), other); !errors.Is(err, identity.ErrInvalidToken) {
		t.Errorf("listed sub, other issuer: err = %v, want ErrInvalidToken", err)
	}

	empty, err := newAdapter(t.Context(), Config{Issuers: h.a.issuers}, h.a.keys.out, h.clock.now)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = empty.Close(context.Background()) })
	op := sign(t, h.idp.signer(realmNava), h.idp.operatorClaims(realmNava, now))
	if _, err := empty.ResolveOperator(t.Context(), op); !errors.Is(err, identity.ErrInvalidToken) {
		t.Errorf("no allow-list: err = %v, want ErrInvalidToken", err)
	}
	if _, err := h.a.ResolveOperator(t.Context(), op); err != nil {
		t.Errorf("listed operator: %v", err)
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	out := h.a.keys.out
	for name, cfg := range map[string]Config{
		"no issuers":      {},
		"empty issuer":    {Issuers: []string{""}},
		"same audiences":  {Issuers: []string{"https://x"}, Audience: "a", OperatorAudience: "a"},
		"default overlap": {Issuers: []string{"https://x"}, Audience: DefaultOperatorAudience},
		"empty operator subject": {Issuers: []string{"https://x"},
			Operators: []OperatorSubject{{Issuer: "https://x"}}},
		"operator issuer not trusted": {Issuers: []string{"https://x"},
			Operators: []OperatorSubject{{Issuer: "https://y", Subject: defaultSub}}},
	} {
		if _, err := New(t.Context(), cfg, out); err == nil {
			t.Errorf("%s: New accepted it", name)
		}
	}
	if _, err := New(t.Context(), Config{Issuers: []string{"https://x"}}, nil); err == nil {
		t.Error("New accepted a nil outbound client")
	}
}
