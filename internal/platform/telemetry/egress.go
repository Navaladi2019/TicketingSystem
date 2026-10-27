package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"syscall"
	"time"
)

// Egress selects which destinations an OutboundClient may connect to.
type Egress int

const (
	// EgressAny allows every destination. It is for URLs the platform itself
	// configures: Keycloak, JWKS and discovery, the platform's own services.
	EgressAny Egress = iota
	// EgressPublicOnly allows only public unicast addresses, https and the
	// ports in OutboundOptions.EgressPorts. It is for URLs a tenant or a third
	// party controls: webhooks, LLM providers, tenant-configured endpoints.
	// The address is checked at connect time, for every connection and every
	// redirect hop, so a DNS answer that changes after a URL was checked
	// (rebinding) is caught too.
	EgressPublicOnly
)

// ErrEgressRefused is the cause of the error an EgressPublicOnly client
// returns when the scheme, port or resolved address is not allowed. The
// target is never connected to.
var ErrEgressRefused = errors.New("outbound: destination refused")

// defaultEgressPorts are the ports EgressPublicOnly allows when
// OutboundOptions.EgressPorts is empty.
func defaultEgressPorts() []int { return []int{443, 8443} }

// egressHooks replace name resolution and the raw dial. Only tests set them
// (export_test.go): they let a hostname "resolve" to a chosen address and a
// chosen address reach a loopback test server, without real DNS or traffic.
type egressHooks struct {
	lookup func(ctx context.Context, network, host string) ([]netip.Addr, error)
	dial   func(ctx context.Context, network, address string) (net.Conn, error)
}

// egressGuard is the EgressPublicOnly policy.
type egressGuard struct {
	ports     map[int]bool
	allowHTTP bool
	denied    []netip.Prefix
	lookup    func(ctx context.Context, network, host string) ([]netip.Addr, error)
	dial      func(ctx context.Context, network, address string) (net.Conn, error)
}

func newEgressGuard(opts OutboundOptions) (*egressGuard, error) {
	ports := opts.EgressPorts
	if len(ports) == 0 {
		ports = defaultEgressPorts()
	}
	g := &egressGuard{ports: make(map[int]bool, len(ports)), allowHTTP: opts.EgressAllowHTTP, denied: deniedPrefixes()}
	for _, p := range ports {
		if p < 1 || p > 65535 {
			return nil, errors.New("telemetry: egress ports must be 1-65535")
		}
		g.ports[p] = true
	}
	g.lookup = net.DefaultResolver.LookupNetIP
	// The Control hook re-checks the address the socket is about to connect
	// to, whatever produced it: the last line of defence.
	g.dial = (&net.Dialer{Timeout: opts.DialTimeout, KeepAlive: 30 * time.Second, Control: g.control}).DialContext
	if h := opts.egressHooks; h != nil {
		g.lookup, g.dial = h.lookup, h.dial
	}
	return g, nil
}

// deniedPrefixes are the ranges besides those netip classifies itself
// (loopback, private, link-local, multicast, unspecified) that are not
// public: shared, reserved and special-purpose space, and NAT64 forms of
// IPv4. Cloud metadata (169.254.169.254, fd00:ec2::254) is link-local or ULA.
func deniedPrefixes() []netip.Prefix {
	return []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),          // "this network"
		netip.MustParsePrefix("100.64.0.0/10"),      // CGNAT shared space
		netip.MustParsePrefix("192.0.0.0/24"),       // IETF protocol assignments
		netip.MustParsePrefix("198.18.0.0/15"),      // benchmarking
		netip.MustParsePrefix("240.0.0.0/4"),        // reserved, broadcast included
		netip.MustParsePrefix("64:ff9b::/96"),       // NAT64 well-known prefix
		netip.MustParsePrefix("64:ff9b:1::/48"),     // NAT64 local-use prefix
		netip.MustParsePrefix("100::/64"),           // discard-only
		netip.MustParsePrefix("2001:db8::/32"),      // documentation
		netip.MustParsePrefix("::/96"),              // IPv4-compatible (deprecated)
		netip.MustParsePrefix("::ffff:0:0:0/96"),    // SIIT IPv4-translated
		netip.MustParsePrefix("2002::/16"),          // 6to4: embeds any IPv4
		netip.MustParsePrefix("2001::/32"),          // Teredo: embeds any IPv4
		netip.MustParsePrefix("fec0::/10"),          // site-local (deprecated)
		netip.MustParsePrefix("169.254.169.254/32"), // metadata, also link-local
		netip.MustParsePrefix("fd00:ec2::254/128"),  // AWS IPv6 metadata, also ULA
	}
}

// public reports whether a is a public unicast address. IPv4-mapped IPv6
// (::ffff:a.b.c.d) is judged as the IPv4 address it maps.
func (g *egressGuard) public(a netip.Addr) bool {
	a = a.WithZone("").Unmap()
	if !a.IsValid() || a.IsUnspecified() || a.IsLoopback() || a.IsPrivate() ||
		a.IsLinkLocalUnicast() || a.IsLinkLocalMulticast() || a.IsInterfaceLocalMulticast() ||
		a.IsMulticast() || !a.IsGlobalUnicast() {
		return false
	}
	for _, p := range g.denied {
		if p.Contains(a) {
			return false
		}
	}
	return true
}

func refused(reason string) error { return fmt.Errorf("%w: %s", ErrEgressRefused, reason) }

// dialContext resolves host itself and connects only when every address is
// public: one private answer among public ones refuses the whole name, so a
// DNS operator cannot steer a retry onto it. Addresses are tried in order.
func (g *egressGuard) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return nil, refused("bad address")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || !g.ports[port] {
		return nil, refused("port not allowed")
	}
	var addrs []netip.Addr
	if a, perr := netip.ParseAddr(host); perr == nil {
		addrs = []netip.Addr{a}
	} else {
		addrs, err = g.lookup(ctx, lookupNetwork(network), host)
		if err != nil {
			return nil, err
		}
	}
	if len(addrs) == 0 {
		return nil, refused("no address")
	}
	for _, a := range addrs {
		if !g.public(a) {
			return nil, refused("address not public")
		}
	}
	var last error
	for _, a := range addrs {
		conn, err := g.dial(ctx, network, netip.AddrPortFrom(a.Unmap(), uint16(port)).String()) //nolint:gosec // G115: port checked against 1-65535 in newEgressGuard
		if err == nil {
			return conn, nil
		}
		last = err
	}
	return nil, last
}

func lookupNetwork(network string) string {
	switch network {
	case "tcp4":
		return "ip4"
	case "tcp6":
		return "ip6"
	}
	return "ip"
}

// control is the net.Dialer Control hook: address is the ip:port the socket
// is about to connect to.
func (g *egressGuard) control(_, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return refused("bad address")
	}
	if !g.ports[int(ap.Port())] {
		return refused("port not allowed")
	}
	if !g.public(ap.Addr()) {
		return refused("address not public")
	}
	return nil
}

// schemeGuard refuses a request whose scheme the policy does not allow. It
// sits under http.Client, so it sees every redirect hop.
type schemeGuard struct {
	next      http.RoundTripper
	allowHTTP bool
}

func (t schemeGuard) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" && (req.URL.Scheme != "http" || !t.allowHTTP) {
		return nil, refused("scheme not allowed")
	}
	return t.next.RoundTrip(req)
}
