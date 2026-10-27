package telemetry_test

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// Adversarial tests for EgressPublicOnly (TICK-261): a URL a tenant or a
// third party controls, or a redirect from it, must never reach loopback,
// private or link-local space, cloud metadata or the platform's own services.

// publicIP stands in for a public address. The test dial hook routes it to a
// loopback TLS server, so no traffic leaves the host.
func publicIP() netip.Addr { return netip.MustParseAddr("203.0.113.10") }

// fakeInternet is a TLS server reachable as https://example.com (the
// httptest certificate names it) through a guarded client whose DNS answers
// are given per client, one per lookup, the last repeating.
type fakeInternet struct {
	srv     *httptest.Server
	hits    atomic.Int64
	lookups atomic.Int64
	dials   atomic.Int64
}

func newFakeInternet(t *testing.T, h http.HandlerFunc) *fakeInternet {
	t.Helper()
	f := &fakeInternet{}
	f.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		h(w, r)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeInternet) client(t *testing.T, answers ...netip.Addr) *telemetry.OutboundClient {
	t.Helper()
	lookup := func(_ context.Context, _, host string) ([]netip.Addr, error) {
		if host != "example.com" {
			return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
		}
		n := int(f.lookups.Add(1)) - 1
		return []netip.Addr{answers[min(n, len(answers)-1)]}, nil
	}
	// Only the stand-in public address is routed anywhere. Any other address
	// reaching this hook means the guard let it through.
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		f.dials.Add(1)
		if ap := netip.MustParseAddrPort(address); ap.Addr() != publicIP() {
			t.Errorf("guard let a dial to %s through", address)
			return nil, errors.New("test: unexpected dial")
		}
		return (&net.Dialer{}).DialContext(ctx, network, f.srv.Listener.Addr().String())
	}
	roots := f.srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs //nolint:forcetypeassert // the httptest client
	opts := telemetry.WithEgressHooksForTest(telemetry.OutboundOptions{
		Egress:    telemetry.EgressPublicOnly,
		TLSConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
	}, lookup, dial)
	c, _, _ := outbound(t, opts)
	return c
}

func get(ctx context.Context, t *testing.T, c *telemetry.OutboundClient, rawURL string) error {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Close = true // a fresh connection, and so a fresh dial, per request
	return doErr(c, req)
}

func forbiddenTargets() []string {
	return []string{
		"https://127.0.0.1/",
		"https://127.1.2.3/",
		"https://10.1.2.3/",
		"https://172.16.0.1/",
		"https://192.168.1.1/",
		"https://100.64.0.1/",
		"https://0.0.0.0/",
		"https://169.254.169.254/latest/meta-data/",
		"https://[::ffff:127.0.0.1]/",
		"https://[::ffff:10.0.0.1]/",
		"https://[::ffff:169.254.169.254]/",
		"https://[::1]/",
		"https://[::]/",
		"https://[fe80::1]/",
		"https://[fd00:ec2::254]/",
		"https://[fc00::1]/",
		"https://[64:ff9b::7f00:1]/",
		"https://224.0.0.1/",
		"https://255.255.255.255/",
	}
}

func TestEgressRefusesNonPublicURL(t *testing.T) {
	t.Parallel()
	f := newFakeInternet(t, func(http.ResponseWriter, *http.Request) {})
	c := f.client(t, publicIP())
	for _, u := range forbiddenTargets() {
		if err := get(t.Context(), t, c, u); !errors.Is(err, telemetry.ErrEgressRefused) {
			t.Errorf("%s: err = %v, want ErrEgressRefused", u, err)
		}
	}
	if n := f.dials.Load(); n != 0 {
		t.Errorf("%d dials reached the network hook", n)
	}
}

// A name that resolves to a forbidden address is refused like the literal.
func TestEgressRefusesNameResolvingToNonPublic(t *testing.T) {
	t.Parallel()
	for _, a := range []string{"127.0.0.1", "10.0.0.7", "169.254.169.254", "::ffff:127.0.0.1"} {
		f := newFakeInternet(t, func(http.ResponseWriter, *http.Request) {})
		c := f.client(t, netip.MustParseAddr(a))
		if err := get(t.Context(), t, c, "https://example.com/"); !errors.Is(err, telemetry.ErrEgressRefused) {
			t.Errorf("example.com -> %s: err = %v, want ErrEgressRefused", a, err)
		}
		if f.dials.Load() != 0 || f.hits.Load() != 0 {
			t.Errorf("example.com -> %s: dialled", a)
		}
	}
}

// The real path, no hooks: loopback by literal and by name (localhost) is
// refused before any connection, even on an allowed port.
func TestEgressRefusesLoopbackWithRealDialer(t *testing.T) {
	t.Parallel()
	var hits atomic.Int64
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	t.Cleanup(srv.Close)
	port := srv.Listener.Addr().(*net.TCPAddr).Port //nolint:forcetypeassert // a TCP listener
	c, _, _ := outbound(t, telemetry.OutboundOptions{Egress: telemetry.EgressPublicOnly, EgressPorts: []int{port}})
	for _, host := range []string{"127.0.0.1", "localhost"} {
		u := "https://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/"
		if err := get(t.Context(), t, c, u); !errors.Is(err, telemetry.ErrEgressRefused) {
			t.Errorf("%s: err = %v, want ErrEgressRefused", u, err)
		}
	}
	if hits.Load() != 0 {
		t.Fatal("a refused request reached the server")
	}
}

// The Control hook re-checks the address the socket connects to.
func TestEgressControlHook(t *testing.T) {
	t.Parallel()
	control, err := telemetry.EgressControlForTest(telemetry.OutboundOptions{Egress: telemetry.EgressPublicOnly})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range []string{"127.0.0.1:443", "10.0.0.1:443", "169.254.169.254:443", "[::ffff:127.0.0.1]:443", "[fd00:ec2::254]:443", "203.0.113.10:80"} {
		if err := control("tcp", a); !errors.Is(err, telemetry.ErrEgressRefused) {
			t.Errorf("%s: err = %v, want ErrEgressRefused", a, err)
		}
	}
	for _, a := range []string{"203.0.113.10:443", "[2606:4700::1111]:8443"} {
		if err := control("tcp", a); err != nil {
			t.Errorf("%s: err = %v, want allowed", a, err)
		}
	}
}

func TestEgressSchemeAndPort(t *testing.T) {
	t.Parallel()
	f := newFakeInternet(t, func(http.ResponseWriter, *http.Request) {})
	c := f.client(t, publicIP())
	for _, u := range []string{"http://example.com/", "https://example.com:8080/", "https://example.com:22/"} {
		if err := get(t.Context(), t, c, u); !errors.Is(err, telemetry.ErrEgressRefused) {
			t.Errorf("%s: err = %v, want ErrEgressRefused", u, err)
		}
	}
	if err := get(t.Context(), t, c, "https://example.com/"); err != nil {
		t.Fatalf("public https on 443 refused: %v", err)
	}
	if f.hits.Load() != 1 {
		t.Fatalf("server hits = %d, want 1", f.hits.Load())
	}
}

// DNS rebinding: example.com answers public, then loopback. The guard checks
// the resolved address of every connection, so the second request is refused
// at dial time and reaches nothing.
func TestEgressRebindingRefusedAtDial(t *testing.T) {
	t.Parallel()
	f := newFakeInternet(t, func(http.ResponseWriter, *http.Request) {})
	c := f.client(t, publicIP(), netip.MustParseAddr("127.0.0.1"))
	if err := get(t.Context(), t, c, "https://example.com/"); err != nil {
		t.Fatalf("first request (public answer) failed: %v", err)
	}
	if err := get(t.Context(), t, c, "https://example.com/"); !errors.Is(err, telemetry.ErrEgressRefused) {
		t.Fatalf("second request (loopback answer): err = %v, want ErrEgressRefused", err)
	}
	if f.hits.Load() != 1 || f.dials.Load() != 1 {
		t.Fatalf("hits = %d, dials = %d, want 1 and 1", f.hits.Load(), f.dials.Load())
	}
}

// A public URL that redirects to a forbidden one is refused, by default and
// when the call opts into redirects: then each hop goes through the guard.
func TestEgressRedirectToNonPublicRefused(t *testing.T) {
	t.Parallel()
	for _, target := range forbiddenTargets() {
		f := newFakeInternet(t, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target, http.StatusFound)
		})
		c := f.client(t, publicIP())
		if err := get(telemetry.WithRedirects(t.Context()), t, c, "https://example.com/start"); !errors.Is(err, telemetry.ErrEgressRefused) {
			t.Errorf("opt-in redirect to %s: err = %v, want ErrEgressRefused", target, err)
		}
		if err := get(t.Context(), t, c, "https://example.com/start"); !errors.Is(err, telemetry.ErrRedirectRefused) {
			t.Errorf("default redirect to %s: err = %v, want ErrRedirectRefused", target, err)
		}
		if f.dials.Load() != 2 {
			t.Errorf("redirect to %s: %d dials, want only the 2 first hops", target, f.dials.Load())
		}
	}
}

// Rebinding between hops: the first hop is public; the redirect names the
// same host, which now resolves to loopback.
func TestEgressRedirectRebindingRefused(t *testing.T) {
	t.Parallel()
	f := newFakeInternet(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			w.Header().Set("Connection", "close")
			http.Redirect(w, r, "https://example.com/next", http.StatusFound)
		}
	})
	c := f.client(t, publicIP(), netip.MustParseAddr("127.0.0.1"))
	err := get(telemetry.WithRedirects(t.Context()), t, c, "https://example.com/start")
	if !errors.Is(err, telemetry.ErrEgressRefused) {
		t.Fatalf("err = %v, want ErrEgressRefused", err)
	}
	if f.hits.Load() != 1 {
		t.Fatalf("hits = %d, want only the first hop", f.hits.Load())
	}
}

func TestEgressRedirectPolicy(t *testing.T) {
	t.Parallel()
	f := newFakeInternet(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/same":
			http.Redirect(w, r, "/ok", http.StatusFound)
		case "/other-port":
			http.Redirect(w, r, "https://example.com:8443/ok", http.StatusFound)
		case "/downgrade":
			http.Redirect(w, r, "http://example.com/ok", http.StatusFound)
		case "/loop":
			http.Redirect(w, r, "/loop", http.StatusFound)
		}
	})
	c := f.client(t, publicIP())
	ctx := t.Context()
	cases := []struct {
		name string
		ctx  context.Context
		path string
		ok   bool
	}{
		{"unmarked", ctx, "/same", false},
		{"WithoutRedirects", telemetry.WithoutRedirects(ctx), "/same", false},
		{"WithRedirects", telemetry.WithRedirects(ctx), "/same", true},
		{"same-origin", telemetry.WithSameOriginRedirects(ctx), "/same", true},
		{"same-origin other port", telemetry.WithSameOriginRedirects(ctx), "/other-port", false},
		{"https to http", telemetry.WithRedirects(ctx), "/downgrade", false},
		{"more than 3 hops", telemetry.WithRedirects(ctx), "/loop", false},
	}
	for _, tc := range cases {
		err := get(tc.ctx, t, c, "https://example.com"+tc.path)
		if tc.ok && err != nil {
			t.Errorf("%s: err = %v, want followed", tc.name, err)
		}
		if !tc.ok && !errors.Is(err, telemetry.ErrRedirectRefused) {
			t.Errorf("%s: err = %v, want ErrRedirectRefused", tc.name, err)
		}
	}
}

// The transport ignores HTTPS_PROXY: a proxy would connect on the client's
// behalf, past the dial guard. Not parallel: it sets the environment.
func TestEgressIgnoresEnvironmentProxy(t *testing.T) {
	var proxyHits atomic.Int64
	proxy := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { proxyHits.Add(1) }))
	t.Cleanup(proxy.Close)
	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("HTTP_PROXY", proxy.URL)
	f := newFakeInternet(t, func(http.ResponseWriter, *http.Request) {})
	c := f.client(t, publicIP())
	if err := get(t.Context(), t, c, "https://example.com/"); err != nil {
		t.Fatal(err)
	}
	if proxyHits.Load() != 0 || f.hits.Load() != 1 {
		t.Fatalf("proxy hits = %d, server hits = %d", proxyHits.Load(), f.hits.Load())
	}
}

func TestEgressRejectsBadOptions(t *testing.T) {
	t.Parallel()
	_, tel, _ := outbound(t, telemetry.OutboundOptions{})
	for _, o := range []telemetry.OutboundOptions{
		{Egress: telemetry.EgressPublicOnly, EgressPorts: []int{0}},
		{Egress: telemetry.EgressPublicOnly, EgressPorts: []int{70000}},
		{Egress: telemetry.Egress(99)},
	} {
		if _, err := telemetry.NewOutboundClient(tel, o); err == nil {
			t.Errorf("%+v accepted", o)
		}
	}
}
