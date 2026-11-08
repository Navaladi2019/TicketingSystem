package telemetry

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

func guard(t *testing.T, lookup func(context.Context, string, string) ([]netip.Addr, error), dial func(context.Context, string, string) (net.Conn, error)) *egressGuard {
	t.Helper()
	g, err := newEgressGuard(OutboundOptions{})
	if err != nil {
		t.Fatal(err)
	}
	g.lookup, g.dial = lookup, dial
	return g
}

func TestDialContextRefusals(t *testing.T) {
	t.Parallel()
	pub := netip.MustParseAddr("93.184.216.34")
	var dialed []string
	failing := errors.New("connect failed")
	g := guard(t,
		func(_ context.Context, network, host string) ([]netip.Addr, error) {
			switch host {
			case "empty.example":
				return nil, nil
			case "broken.example":
				return nil, errors.New("lookup failed")
			case "mixed.example":
				return []netip.Addr{pub, netip.MustParseAddr("10.0.0.1")}, nil
			}
			dialed = append(dialed, "lookup:"+network)
			return []netip.Addr{pub, netip.MustParseAddr("93.184.216.35")}, nil
		},
		func(_ context.Context, _, address string) (net.Conn, error) { return nil, failing })

	tests := []struct{ name, address, want string }{
		{"no port", "example.com", "bad address"},
		{"port not allowed", "example.com:22", "port not allowed"},
		{"port not a number", "example.com:http", "port not allowed"},
		{"no address", "empty.example:443", "no address"},
		{"private among public", "mixed.example:443", "address not public"},
		{"literal private", "10.0.0.1:443", "address not public"},
	}
	for _, tt := range tests {
		_, err := g.dialContext(t.Context(), "tcp", tt.address)
		if !errors.Is(err, ErrEgressRefused) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want a refusal %q", tt.name, err, tt.want)
		}
	}
	if _, err := g.dialContext(t.Context(), "tcp", "broken.example:443"); err == nil || errors.Is(err, ErrEgressRefused) {
		t.Errorf("a lookup failure is returned as is: %v", err)
	}
	// Every address is tried; the last dial error comes back.
	if _, err := g.dialContext(t.Context(), "tcp4", "ok.example:443"); !errors.Is(err, failing) {
		t.Errorf("dial failure: err = %v", err)
	}
	if len(dialed) != 1 || dialed[0] != "lookup:ip4" {
		t.Errorf("lookup networks = %v", dialed)
	}
}

func TestLookupNetwork(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{"tcp4": "ip4", "tcp6": "ip6", "tcp": "ip", "": "ip"} {
		if got := lookupNetwork(in); got != want {
			t.Errorf("lookupNetwork(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestControlRefusesBadAddress(t *testing.T) {
	t.Parallel()
	g := guard(t, nil, nil)
	if err := g.control("tcp", "not-an-address", nil); !errors.Is(err, ErrEgressRefused) {
		t.Fatalf("err = %v, want a refusal", err)
	}
}

func TestOriginDefaultPorts(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"https://Example.com/x":    "https://example.com:443",
		"http://Example.com/x":     "http://example.com:80",
		"ftp://example.com/x":      "ftp://example.com:",
		"https://example.com:8443": "https://example.com:8443",
	} {
		u, _ := url.Parse(in)
		if got := origin(u); got != want {
			t.Errorf("origin(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDenyHandlerGroups(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := slog.NewJSONHandler(&buf, nil)
	h := denyHandler{next: base}
	if got := h.WithGroup(""); got.(denyHandler).path != "" {
		t.Fatal("an empty group name must leave the handler unchanged")
	}
	nested := h.WithGroup("a").WithGroup("b").(denyHandler)
	if nested.path != "a.b" {
		t.Fatalf("path = %q, want a.b", nested.path)
	}
	slog.New(nested).InfoContext(t.Context(), "m", slog.Int("k", 1))
	if !strings.Contains(buf.String(), `"a":{"b":{"k":1}}`) {
		t.Fatalf("groups lost: %s", buf.String())
	}
}

func TestRecoverAdminPassesAbortThrough(t *testing.T) {
	t.Parallel()
	h := recoverAdmin(slog.New(slog.DiscardHandler), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if v := recover(); v != http.ErrAbortHandler { //nolint:errorlint // identity, as net/http
			t.Fatalf("recovered %v, want http.ErrAbortHandler", v)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
}
