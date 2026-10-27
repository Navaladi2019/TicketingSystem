package keycloak

import (
	"net/url"
	"testing"

	"go.opentelemetry.io/otel/metric/noop"
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
