package telemetry

import (
	"context"
	"net"
	"net/netip"
)

// WithEgressHooksForTest returns opts whose EgressPublicOnly guard resolves
// names with lookup and connects with dial instead of DNS and the network.
// The guard's checks run unchanged between the two.
func WithEgressHooksForTest(opts OutboundOptions,
	lookup func(ctx context.Context, network, host string) ([]netip.Addr, error),
	dial func(ctx context.Context, network, address string) (net.Conn, error),
) OutboundOptions {
	opts.egressHooks = &egressHooks{lookup: lookup, dial: dial}
	return opts
}

// EgressControlForTest returns the net.Dialer Control hook of an
// EgressPublicOnly guard built from opts.
func EgressControlForTest(opts OutboundOptions) (func(network, address string) error, error) {
	g, err := newEgressGuard(opts)
	if err != nil {
		return nil, err
	}
	return func(network, address string) error { return g.control(network, address, nil) }, nil
}
