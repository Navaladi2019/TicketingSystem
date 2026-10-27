package telemetry_test

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// Bypass probes from the TICK-261 security review: answer sets, encodings and
// literal forms an attacker controlling a webhook URL or its DNS can choose.

// A name whose answer set mixes a public address with a forbidden one is
// refused whole, whatever the order: the dialer must not try the public one
// first and fall back to the private one on failure.
func TestEgressMixedAnswerSetRefused(t *testing.T) {
	t.Parallel()
	for _, set := range [][]string{
		{"203.0.113.10", "127.0.0.1"},
		{"127.0.0.1", "203.0.113.10"},
		{"203.0.113.10", "::ffff:169.254.169.254"},
		{"2606:4700::1111", "fd00:ec2::254"},
	} {
		f := newFakeInternet(t, func(http.ResponseWriter, *http.Request) {})
		var addrs []netip.Addr
		for _, a := range set {
			addrs = append(addrs, netip.MustParseAddr(a))
		}
		var dials atomic.Int64
		lookup := func(context.Context, string, string) ([]netip.Addr, error) { return addrs, nil }
		dial := func(context.Context, string, string) (net.Conn, error) {
			dials.Add(1)
			return nil, errors.New("test: dial must not happen")
		}
		roots := f.srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs //nolint:forcetypeassert // the httptest client
		opts := telemetry.WithEgressHooksForTest(telemetry.OutboundOptions{
			Egress:    telemetry.EgressPublicOnly,
			TLSConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
		}, lookup, dial)
		c, _, _ := outbound(t, opts)
		if err := get(t.Context(), t, c, "https://example.com/"); !errors.Is(err, telemetry.ErrEgressRefused) {
			t.Errorf("%v: err = %v, want ErrEgressRefused", set, err)
		}
		if dials.Load() != 0 {
			t.Errorf("%v: %d dials", set, dials.Load())
		}
	}
}

// IPv6 forms that embed an internal IPv4 address, and zone-scoped literals.
func TestEgressEmbeddedAndScopedLiteralsRefused(t *testing.T) {
	t.Parallel()
	f := newFakeInternet(t, func(http.ResponseWriter, *http.Request) {})
	c := f.client(t, publicIP())
	for _, u := range []string{
		"https://[::127.0.0.1]/",         // IPv4-compatible
		"https://[::ffff:0:7f00:1]/",     // SIIT
		"https://[2002:7f00:1::]/",       // 6to4 of 127.0.0.1
		"https://[2002:a9fe:a9fe::]/",    // 6to4 of 169.254.169.254
		"https://[2001:0:4136:e378::1]/", // Teredo
		"https://[64:ff9b:1::a00:1]/",    // NAT64 local-use of 10.0.0.1
		"https://[fe80::1%25eth0]/",      // zone-scoped link-local
		"https://[fec0::1]/",             // site-local
		"https://[ff02::1]/",             // multicast
		"https://198.18.0.1/",            // benchmarking
		"https://192.0.0.170/",           // IETF assignments
		"https://240.0.0.1/",             // reserved
	} {
		if err := get(t.Context(), t, c, u); !errors.Is(err, telemetry.ErrEgressRefused) {
			t.Errorf("%s: err = %v, want ErrEgressRefused", u, err)
		}
	}
	if f.dials.Load() != 0 {
		t.Errorf("%d dials reached the network hook", f.dials.Load())
	}
}

// The Control hook is the last line: it must refuse scoped and embedded
// forms too, in case a future dial path skips dialContext.
func TestEgressControlHookEmbeddedForms(t *testing.T) {
	t.Parallel()
	control, err := telemetry.EgressControlForTest(telemetry.OutboundOptions{Egress: telemetry.EgressPublicOnly})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range []string{
		"[fe80::1%eth0]:443", "[::127.0.0.1]:443", "[2002:7f00:1::]:443",
		"[64:ff9b::a9fe:a9fe]:443", "0.0.0.0:443", "[::]:443", "100.100.100.200:443",
	} {
		if err := control("tcp", a); !errors.Is(err, telemetry.ErrEgressRefused) {
			t.Errorf("%s: err = %v, want ErrEgressRefused", a, err)
		}
	}
}

// Same-origin redirects fill in the default
// port; anything else, including a different subdomain, is refused.
func TestEgressSameOriginRedirectBoundary(t *testing.T) {
	t.Parallel()
	f := newFakeInternet(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/explicit-port":
			http.Redirect(w, r, "https://example.com:443/ok", http.StatusFound)
		case "/subdomain":
			http.Redirect(w, r, "https://evil.example.com/ok", http.StatusFound)
		case "/userinfo":
			http.Redirect(w, r, "https://example.com@evil.test/ok", http.StatusFound)
		}
	})
	c := f.client(t, publicIP())
	ctx := telemetry.WithSameOriginRedirects(t.Context())
	if err := get(ctx, t, c, "https://example.com/explicit-port"); err != nil {
		t.Errorf("explicit default port: err = %v, want followed", err)
	}
	for _, p := range []string{"/subdomain", "/userinfo"} {
		if err := get(ctx, t, c, "https://example.com"+p); !errors.Is(err, telemetry.ErrRedirectRefused) {
			t.Errorf("%s: err = %v, want ErrRedirectRefused", p, err)
		}
	}
}

// EgressAllowHTTP does not permit an https to http downgrade on redirect.
func TestEgressAllowHTTPStillRefusesDowngrade(t *testing.T) {
	t.Parallel()
	f := newFakeInternet(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com:443/ok", http.StatusFound)
	})
	roots := f.srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs //nolint:forcetypeassert // the httptest client
	lookup := func(context.Context, string, string) ([]netip.Addr, error) { return []netip.Addr{publicIP()}, nil }
	dial := func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, f.srv.Listener.Addr().String())
	}
	opts := telemetry.WithEgressHooksForTest(telemetry.OutboundOptions{
		Egress: telemetry.EgressPublicOnly, EgressAllowHTTP: true,
		TLSConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
	}, lookup, dial)
	c, _, _ := outbound(t, opts)
	if err := get(telemetry.WithRedirects(t.Context()), t, c, "https://example.com/"); !errors.Is(err, telemetry.ErrRedirectRefused) {
		t.Fatalf("err = %v, want ErrRedirectRefused", err)
	}
}
