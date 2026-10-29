//go:build stacktag && accept

package stack

import "testing"

// postRaw is declared again, as onboardPostRaw was on main (TICK-286
// finding 4): only -tags accept,stacktag sees both declarations.
func postRaw() string { return Name() }

func TestAccept(t *testing.T) { _ = postRaw() }
