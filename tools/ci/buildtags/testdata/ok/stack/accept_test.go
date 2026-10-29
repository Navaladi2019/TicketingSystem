//go:build stacktag && accept

package stack

import "testing"

func acceptPostRaw() string { return postRaw() }

func TestAccept(t *testing.T) { _ = acceptPostRaw() }
