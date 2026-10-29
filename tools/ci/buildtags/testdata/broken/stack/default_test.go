//go:build !stacktag && go1.22

package stack

import "testing"

func TestDefault(t *testing.T) { _ = Name() }
