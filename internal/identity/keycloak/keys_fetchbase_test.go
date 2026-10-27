package keycloak

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// TICK-160: a fetch base moves where discovery and JWKS are requested, never
// the issuer they are checked against.
func TestReachFetchBase(t *testing.T) {
	t.Parallel()
	const iss = "http://keycloak:8080/realms/nava"
	const certs = iss + "/protocol/openid-connect/certs"
	base, _ := url.Parse("http://localhost:18081")
	for _, tc := range []struct {
		name string
		s    *keySource
		want string
	}{
		{"no base", &keySource{}, certs},
		{"base", &keySource{fetchBase: base}, "http://localhost:18081/realms/nava/protocol/openid-connect/certs"},
	} {
		if got := tc.s.reach(iss, certs); got != tc.want {
			t.Errorf("%s: reach = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestInvalidFetchBaseRejected(t *testing.T) {
	t.Parallel()
	for _, b := range []string{"localhost:18081", "ftp://x", "http://u:p@x", "http://"} {
		if _, err := newKeySource(t.Context(), nil, []string{"http://k/realms/a"}, b, nil, nil, noop.NewMeterProvider()); err == nil {
			t.Errorf("fetch base %q accepted", b)
		}
	}
}

// A back-channel discovery document names the pinned (browser) issuer and a
// jwks_uri on the fetch base: accepted. A jwks_uri on any other host, another
// realm, or a mismatched issuer is refused.
func TestDiscoverViaFetchBase(t *testing.T) {
	t.Parallel()
	const iss = "http://localhost:18081/realms/nava"
	for _, tc := range []struct {
		name   string
		doc    func(base string) map[string]string
		wantOK bool
	}{
		{"jwks on fetch base", func(b string) map[string]string {
			return map[string]string{"issuer": iss, "jwks_uri": b + "/realms/nava/protocol/openid-connect/certs"}
		}, true},
		{"jwks on issuer host", func(string) map[string]string {
			return map[string]string{"issuer": iss, "jwks_uri": iss + "/protocol/openid-connect/certs"}
		}, true},
		{"jwks on other host", func(string) map[string]string {
			return map[string]string{"issuer": iss, "jwks_uri": "http://evil.example/realms/nava/protocol/openid-connect/certs"}
		}, false},
		{"jwks in other realm", func(b string) map[string]string {
			return map[string]string{"issuer": iss, "jwks_uri": b + "/realms/other/protocol/openid-connect/certs"}
		}, false},
		{"issuer follows the fetch host", func(b string) map[string]string {
			return map[string]string{"issuer": b + "/realms/nava", "jwks_uri": b + "/realms/nava/protocol/openid-connect/certs"}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var srv *httptest.Server
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/realms/nava/.well-known/openid-configuration" {
					http.NotFound(w, r)
					return
				}
				_ = json.NewEncoder(w).Encode(tc.doc(srv.URL))
			}))
			t.Cleanup(srv.Close)
			cfg, err := config.Load(func(string) (string, bool) { return "", false })
			if err != nil {
				t.Fatal(err)
			}
			tel, err := telemetry.New(t.Context(), cfg.Telemetry)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
			out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{Timeout: 2 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			base, _ := url.Parse(srv.URL)
			s := &keySource{out: out, fetchBase: base}
			got, err := s.discover(t.Context(), iss)
			if (err == nil) != tc.wantOK {
				t.Fatalf("discover err = %v, want ok %v", err, tc.wantOK)
			}
			if tc.wantOK && got != srv.URL+"/realms/nava/protocol/openid-connect/certs" {
				t.Errorf("jwks fetched from %q, want it on the fetch base", got)
			}
		})
	}
}
