// Package keycloaktest is a fake Keycloak realm for tests outside the
// adapter: it serves a discovery document and a JWKS, mints tenant access
// tokens in the realm's shape, and builds a real keycloak.Adapter that trusts
// it. It lives under internal/identity/keycloak because only that tree may
// know Keycloak's token shape or import jwx (D-P6, CI-11); callers name
// organizations and IdP aliases, never claims.
package keycloaktest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

const (
	realm = "nava"
	kid   = "keycloaktest-k1"
)

// Realm is one fake realm on its own HTTP server, closed with the test.
type Realm struct {
	srv  *httptest.Server
	priv *rsa.PrivateKey
}

// New starts a realm.
func New(t testing.TB) *Realm {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	r := &Realm{priv: priv}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /realms/"+realm+"/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": r.Issuer(), "jwks_uri": r.Issuer() + "/protocol/openid-connect/certs",
		})
	})
	mux.HandleFunc("GET /realms/"+realm+"/protocol/openid-connect/certs", func(w http.ResponseWriter, _ *http.Request) {
		k, err := jwk.PublicKeyOf(&priv.PublicKey)
		if err != nil {
			http.Error(w, "key", http.StatusInternalServerError)
			return
		}
		_ = k.Set(jwk.KeyIDKey, kid)
		_ = k.Set(jwk.AlgorithmKey, jwa.RS256())
		_ = k.Set(jwk.KeyUsageKey, "sig")
		set := jwk.NewSet()
		_ = set.AddKey(k)
		_ = json.NewEncoder(w).Encode(set)
	})
	r.srv = httptest.NewServer(mux)
	t.Cleanup(r.srv.Close)
	return r
}

// Issuer is the realm's iss.
func (r *Realm) Issuer() string { return r.srv.URL + "/realms/" + realm }

// Token describes a tenant access token. The zero value of every field but
// Subject leaves the claim out.
type Token struct {
	// Subject is the Keycloak user id (sub), principals.keycloak_user_id.
	Subject string
	// Orgs maps organization alias to organization id. One entry is a
	// resolvable token; none or two are not.
	Orgs map[string]string
	// IdP is the brokering IdP's alias; empty for a local login.
	IdP string
	// Audiences is aud; empty means the tenant audience nava-api alone.
	Audiences []string
	// RealmRoles are realm_access.roles; empty leaves the claim out.
	RealmRoles []string
}

// Mint signs an access token valid for five minutes from now, the realm's
// access token lifespan (D-P9).
func (r *Realm) Mint(t testing.TB, tok Token) string {
	t.Helper()
	now := time.Now()
	aud := tok.Audiences
	if len(aud) == 0 {
		aud = []string{keycloak.DefaultAudience}
	}
	c := map[string]any{
		"iss": r.Issuer(),
		"sub": tok.Subject,
		"aud": aud,
		"azp": "nava-web",
		"typ": "Bearer",
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
	}
	if tok.Orgs != nil {
		orgs := map[string]any{}
		for alias, id := range tok.Orgs {
			orgs[alias] = map[string]any{"id": id}
		}
		c["organization"] = orgs
	}
	if tok.IdP != "" {
		c["idp"] = tok.IdP
	}
	if len(tok.RealmRoles) > 0 {
		c["realm_access"] = map[string]any{"roles": tok.RealmRoles}
	}
	jt := jwt.New()
	for k, v := range c {
		if err := jt.Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	hdr := jws.NewHeaders()
	_ = hdr.Set(jws.KeyIDKey, kid)
	_ = hdr.Set(jws.TypeKey, "JWT")
	out, err := jwt.Sign(jt, jwt.WithKey(jwa.RS256(), r.priv, jws.WithProtectedHeaders(hdr)))
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// Resolver returns a real adapter that trusts only this realm and resolves
// tenants through store; operators are the subjects on its operator
// allow-list. It is closed with the test.
func (r *Realm) Resolver(t testing.TB, store identity.Store, operators ...string) *keycloak.Adapter {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ops := make([]keycloak.OperatorSubject, 0, len(operators))
	for _, sub := range operators {
		ops = append(ops, keycloak.OperatorSubject{Issuer: r.Issuer(), Subject: sub})
	}
	a, err := keycloak.New(t.Context(), keycloak.Config{Issuers: []string{r.Issuer()}, Store: store, Operators: ops}, out)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close(t.Context()) })
	return a
}
