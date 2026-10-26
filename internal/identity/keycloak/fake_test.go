package keycloak

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// fakeIdP serves Keycloak-shaped realms: a discovery document and a JWKS per
// realm, and counts the requests.
type fakeIdP struct {
	t   *testing.T
	srv *httptest.Server

	mu     sync.Mutex
	realms map[string]*fakeRealm
	// discoveryHook, if set, rewrites a realm's discovery document.
	discoveryHook func(realm string, doc map[string]any)

	requests  atomic.Int64 // every request
	jwksHits  atomic.Int64 // JWKS requests, all realms
	discovery atomic.Int64
}

type fakeRealm struct {
	keys []signer // published in the JWKS
}

type signer struct {
	kid  string
	priv *rsa.PrivateKey
}

func newFakeIdP(t *testing.T, realms ...string) *fakeIdP {
	t.Helper()
	f := &fakeIdP{t: t, realms: map[string]*fakeRealm{}}
	for _, r := range realms {
		f.realms[r] = &fakeRealm{keys: []signer{newSigner(t, r+"-k1")}}
	}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func newSigner(t *testing.T, kid string) signer {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return signer{kid: kid, priv: priv}
}

func (f *fakeIdP) issuer(realm string) string { return f.srv.URL + "/realms/" + realm }

func (f *fakeIdP) signer(realm string) signer {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.realms[realm].keys[0]
}

// rotate publishes a new key, first in the set, and keeps the old one.
func (f *fakeIdP) rotate(realm, kid string) signer {
	s := newSigner(f.t, kid)
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.realms[realm]
	r.keys = append([]signer{s}, r.keys...)
	return s
}

func (f *fakeIdP) serve(w http.ResponseWriter, r *http.Request) {
	f.requests.Add(1)
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/realms/"), "/")
	f.mu.Lock()
	realm, ok := f.realms[parts[0]]
	f.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch strings.Join(parts[1:], "/") {
	case ".well-known/openid-configuration":
		f.discovery.Add(1)
		iss := f.issuer(parts[0])
		doc := map[string]any{"issuer": iss, "jwks_uri": iss + "/protocol/openid-connect/certs"}
		if f.discoveryHook != nil {
			f.discoveryHook(parts[0], doc)
		}
		_ = json.NewEncoder(w).Encode(doc)
	case "protocol/openid-connect/certs":
		f.jwksHits.Add(1)
		set := jwk.NewSet()
		f.mu.Lock()
		keys := realm.keys
		f.mu.Unlock()
		for _, s := range keys {
			k, err := jwk.PublicKeyOf(&s.priv.PublicKey)
			if err != nil {
				f.t.Error(err)
				return
			}
			_ = k.Set(jwk.KeyIDKey, s.kid)
			_ = k.Set(jwk.AlgorithmKey, jwa.RS256())
			_ = k.Set(jwk.KeyUsageKey, "sig")
			_ = set.AddKey(k)
		}
		_ = json.NewEncoder(w).Encode(set)
	default:
		http.NotFound(w, r)
	}
}

// fakeClock is the adapter's clock; tests move it.
type fakeClock struct{ ns atomic.Int64 }

func newFakeClock() *fakeClock {
	c := &fakeClock{}
	c.ns.Store(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC).UnixNano())
	return c
}

func (c *fakeClock) now() time.Time          { return time.Unix(0, c.ns.Load()).UTC() }
func (c *fakeClock) advance(d time.Duration) { c.ns.Add(int64(d)) }

// claims is a token body; build starts from a valid tenant access token.
type claims map[string]any

func (f *fakeIdP) tenantClaims(realm string, now time.Time) claims {
	return claims{
		"iss":          f.issuer(realm),
		"sub":          "6f1c2a7e-0000-4000-8000-000000000001",
		"aud":          []string{DefaultAudience, "account"},
		"azp":          "nava-web",
		"typ":          "Bearer",
		"iat":          now.Unix(),
		"exp":          now.Add(5 * time.Minute).Unix(),
		"organization": map[string]any{"tenant-a": map[string]any{"id": "0b6f9a8e-org-a"}},
	}
}

func (f *fakeIdP) operatorClaims(realm string, now time.Time) claims {
	c := f.tenantClaims(realm, now)
	delete(c, "organization")
	c["aud"] = []string{DefaultOperatorAudience}
	c["azp"] = "nava-operator-cli"
	c["realm_access"] = map[string]any{"roles": []string{OperatorRole, "default-roles-nava"}}
	return c
}

// sign signs c with s as RS256 with kid s.kid.
func sign(t *testing.T, s signer, c claims) string {
	t.Helper()
	tok := jwt.New()
	for k, v := range c {
		if err := tok.Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	hdr := jws.NewHeaders()
	_ = hdr.Set(jws.KeyIDKey, s.kid)
	_ = hdr.Set(jws.TypeKey, "JWT")
	out, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), s.priv, jws.WithProtectedHeaders(hdr)))
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

type harness struct {
	idp   *fakeIdP
	clock *fakeClock
	a     *Adapter
	spans *tracetest.InMemoryExporter
	store *memStore
}

// newHarness builds an adapter that trusts the given realms of a new fake
// IdP, whose realms are trusted plus any in extra.
func newHarness(t *testing.T, trusted []string, extra ...string) *harness {
	t.Helper()
	idp := newFakeIdP(t, append(slicesClone(trusted), extra...)...)
	return newHarnessFor(t, idp, trusted)
}

func newHarnessFor(t *testing.T, idp *fakeIdP, trusted []string) *harness {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	spans := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), cfg.Telemetry, telemetry.WithSpanExporter(spans))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	issuers := make([]string, len(trusted))
	for i, r := range trusted {
		issuers[i] = idp.issuer(r)
	}
	clock := newFakeClock()
	store := newMemStore()
	// defaultSub is the one allow-listed operator, in realm nava when trusted.
	var operators []OperatorSubject
	if slices.Contains(trusted, realmNava) {
		operators = []OperatorSubject{{Issuer: idp.issuer(realmNava), Subject: defaultSub}}
	}
	a, err := newAdapter(t.Context(), Config{Issuers: issuers, MeterProvider: tel.MeterProvider, Store: store, Operators: operators}, out, clock.now)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close(t.Context()) })
	return &harness{idp: idp, clock: clock, a: a, spans: spans, store: store}
}

func slicesClone(s []string) []string { return append([]string(nil), s...) }
