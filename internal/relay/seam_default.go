//go:build !relaytest

package relay

import "context"

// crashSeam does nothing in a normal build. The relaytest build replaces it
// (seam_relaytest.go).
func crashSeam(context.Context, Claimed) {}
