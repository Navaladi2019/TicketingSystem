package main

import (
	"slices"
	"testing"
)

func TestWithoutApproved(t *testing.T) {
	t.Parallel()
	problems := []string{"a changed", "b removed", "c removed"}
	approved := []byte("# header\n\na changed # TICK-1 accepted by the owner\nb removed\nc removed #   \n")
	got := withoutApproved(problems, approved)
	// b has no reason and c an empty one: both still fail.
	if want := []string{"b removed", "c removed"}; !slices.Equal(got, want) {
		t.Errorf("withoutApproved = %q, want %q", got, want)
	}
	if got := withoutApproved(problems, nil); !slices.Equal(got, problems) {
		t.Errorf("nothing approved = %q, want all", got)
	}
}
