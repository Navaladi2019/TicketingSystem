package keycloak

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// TICK-260: the admin client must not follow a redirect. A 307 from the token
// endpoint would resend client_secret; a same-host 302 on the admin API would
// resend the bearer token.
func adminFor(t *testing.T, base string) (*admin, *account) {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return &admin{base: base, realm: "nava", out: out, now: time.Now},
		&account{creds: ClientCredentials{ClientID: "c", ClientSecret: "s3cret"}}
}

func TestAdminTokenRedirectRefused(t *testing.T) {
	t.Parallel()
	var hits atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	t.Cleanup(target.Close)
	kc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/collect", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(kc.Close)
	a, acct := adminFor(t, kc.URL)
	_, err := a.bearer(t.Context(), acct)
	if !errors.Is(err, telemetry.ErrRedirectRefused) {
		t.Fatalf("err = %v, want ErrRedirectRefused", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("redirect target received %d requests (client secret leak)", hits.Load())
	}
}

func TestAdminGetSameHostRedirectRefused(t *testing.T) {
	t.Parallel()
	var leaked atomic.Int64
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			leaked.Add(1)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(other.Close)
	kc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/realms/nava/protocol/openid-connect/token" {
			_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":300}`))
			return
		}
		http.Redirect(w, r, other.URL+r.URL.Path, http.StatusFound) // same hostname, other port
	}))
	t.Cleanup(kc.Close)
	a, acct := adminFor(t, kc.URL)
	var v []string
	err := a.getJSON(t.Context(), acct, "/users", "/users", nil, &v)
	if !errors.Is(err, telemetry.ErrRedirectRefused) {
		t.Fatalf("err = %v, want ErrRedirectRefused", err)
	}
	if leaked.Load() != 0 {
		t.Fatal("bearer token reached the redirect target")
	}
}
