//go:build stacktag

package stack

import "testing"

func postRaw() string { return Name() }

func TestStack(t *testing.T) {
	if postRaw() != "stack" {
		t.Fatal("name")
	}
}
