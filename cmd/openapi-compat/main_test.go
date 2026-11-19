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

func TestWithoutApprovedNeedsAnIssueKeyAndTheExactText(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		problems []string
		approved string
		want     []string // what still fails
	}{
		{"a reason with no issue key", []string{"GET /x: type changed"}, "GET /x: type changed # because\n", []string{"GET /x: type changed"}},
		{"a key in the middle of the reason", []string{"GET /x: type changed"}, "GET /x: type changed # see TICK-1\n", []string{"GET /x: type changed"}},
		{"a lower-case key", []string{"GET /x: type changed"}, "GET /x: type changed # tick-1\n", []string{"GET /x: type changed"}},
		{"a key with no number", []string{"GET /x: type changed"}, "GET /x: type changed # TICK-\n", []string{"GET /x: type changed"}},
		{"a key glued to letters", []string{"GET /x: type changed"}, "GET /x: type changed # TICK-1x\n", []string{"GET /x: type changed"}},
		{"a prefix of the problem", []string{"GET /x: type changed from a"}, "GET /x: type changed # TICK-1 ok\n", []string{"GET /x: type changed from a"}},
		{"a longer text than the problem", []string{"GET /x: type changed"}, "GET /x: type changed from a # TICK-1 ok\n", []string{"GET /x: type changed"}},
		{"a key then words", []string{"GET /x: type changed"}, "GET /x: type changed # TICK-939: approved\n", nil},
		{"a CRLF line", []string{"GET /x: type changed"}, "GET /x: type changed # TICK-939 ok\r\n", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := withoutApproved(tc.problems, []byte(tc.approved)); !slices.Equal(got, tc.want) {
				t.Errorf("still failing = %q, want %q", got, tc.want)
			}
		})
	}
}
