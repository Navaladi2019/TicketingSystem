package main

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func at(hhmm string) time.Time {
	t, err := time.Parse(time.RFC3339, "2026-10-28T"+hhmm+":00Z")
	if err != nil {
		panic(err)
	}
	return t
}

func mergedPR(n int, head, base, oid, created, mergedAt string) PR {
	return PR{Number: n, State: "MERGED", Head: head, Base: base, HeadOid: oid, CreatedAt: at(created), MergedAt: at(mergedAt), ClosedAt: at(mergedAt)}
}

func openPR(n int, head, base, created string) PR {
	return PR{Number: n, State: "OPEN", Head: head, Base: base, CreatedAt: at(created)}
}

const (
	s5 = "TICK-47/group-changes-within-10-minutes"
	s6 = "TICK-48/api-ground-rules"
	s7 = "TICK-49/outbox-row-runs-one-workflow"
)

// The milestone-1 state TICK-286 found (TICK-322): S7 landed, S5 got three
// task PRs after its story PR merged, S6 never had a story PR.
func milestone1() Input {
	return Input{
		Main: "main",
		Branches: map[string]string{
			"main": "m", s5: "s5-late", s6: "s6-head", s7: "s7-head",
			"TICK-297/ci-lint-drift-guards": "t297", "agent/x/1": "x",
		},
		PRs: []PR{
			mergedPR(200, "TICK-245/guard", s7, "t245", "01:00", "02:00"),
			mergedPR(223, s7, "main", "s7-head", "03:00", "12:00"),
			mergedPR(187, "TICK-47/acceptance-kit", s5, "kit", "01:00", "02:00"),
			mergedPR(221, s5, "main", "s5-story", "02:00", "16:47"),
			mergedPR(251, "TICK-297/ci-lint-drift-guards", s5, "t297", "16:26", "17:19"),
			mergedPR(252, "tick-282", s5, "t282", "16:30", "17:32"),
			mergedPR(255, "TICK-283/x", s5, "t283", "16:40", "17:32"),
			mergedPR(175, "TICK-213/x", s6, "t213", "01:00", "06:03"),
			// Closed demo PRs into a task branch do not make it a story branch.
			{Number: 253, State: "CLOSED", Head: "demo/a", Base: "TICK-297/ci-lint-drift-guards", CreatedAt: at("16:50"), ClosedAt: at("17:00")},
			// Only TICK-<n>/ branches are story branches.
			mergedPR(114, "sec/y", "agent/x/1", "y", "01:00", "02:00"),
		},
	}
}

// landedAt says which (base, head) pairs are on main; anything else is not.
// It records each call as base..head, plus @hh:mm for a non-zero since.
func landedAt(on ...string) (Landed, *[]string) {
	var calls []string
	return func(base, head string, since time.Time) (bool, error) {
		k := base + ".." + head
		c := k
		if !since.IsZero() {
			c += "@" + since.UTC().Format("15:04")
		}
		calls = append(calls, c)
		return slices.Contains(on, k), nil
	}, &calls
}

func texts(fs []Finding) []string {
	s := make([]string, len(fs))
	for i, f := range fs {
		s[i] = f.String()
	}
	return s
}

func TestMilestone1Replay(t *testing.T) {
	in := milestone1()
	landed, calls := landedAt("s7-head..s7-head")
	fs, _, err := Check(in, landed)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		s5 + ": story PR #221 merged on 2026-10-28 16:47Z, but changes since then are not on main: PRs #251, #252, #255 merged into it afterwards (head s5-late, story PR head s5-stor)",
		s6 + ": orphaned: PR #175 merged into it, no PR out of it is merged or open, and its changes are not on main (head s6-head)",
	}
	if got := texts(fs); !slices.Equal(got, want) {
		t.Fatalf("findings:\n got %q\nwant %q", got, want)
	}
	// Story PR heads are the merge base and main is searched from the story
	// PR's merge on; an orphan lets git pick both.
	wantCalls := []string{"s5-story..s5-late@16:47", "..s6-head", "s7-head..s7-head@12:00"}
	if !slices.Equal(*calls, wantCalls) {
		t.Fatalf("landed calls: got %q, want %q", *calls, wantCalls)
	}
}

// After PR #267 brings S5's late PRs to main and #266 merges S6, it is green.
func TestGreenOnceSynced(t *testing.T) {
	in := milestone1()
	in.PRs = append(in.PRs, mergedPR(266, s6, "main", "s6-head", "18:04", "20:00"))
	landed, _ := landedAt("s7-head..s7-head", "s5-story..s5-late", "s6-head..s6-head")
	fs, _, err := Check(in, landed)
	if err != nil || len(fs) > 0 {
		t.Fatalf("want green, got %q, %v", texts(fs), err)
	}
}

// An open story PR means the story is in flight: neither rule fires for it.
func TestOpenStoryPRIsInFlight(t *testing.T) {
	in := milestone1()
	in.PRs = append(in.PRs, openPR(266, s6, "main", "18:04"))
	landed, calls := landedAt("s7-head..s7-head")
	fs, _, err := Check(in, landed)
	if err != nil {
		t.Fatal(err)
	}
	if got := texts(fs); len(got) != 1 || !strings.HasPrefix(got[0], s5+":") {
		t.Fatalf("want only the S5 finding, got %q", got)
	}
	if slices.Contains(*calls, "..s6-head") {
		t.Fatal("an in-flight story branch was compared with main")
	}

	// -as-of before #266 existed replays S6 as orphaned.
	in.AsOf = at("18:00")
	fs, _, _ = Check(in, landed)
	if !slices.ContainsFunc(fs, func(f Finding) bool { return f.Branch == s6 && strings.HasPrefix(f.Text, "orphaned") }) {
		t.Fatalf("as of 18:00, want S6 orphaned, got %q", texts(fs))
	}
}

func TestOpenPRIntoMergedStory(t *testing.T) {
	in := milestone1()
	in.PRs = append(in.PRs, openPR(270, "TICK-330/x", s7, "19:00"))
	landed, _ := landedAt("s7-head..s7-head", "s5-story..s5-late")
	fs, _, err := Check(in, landed)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		s6 + ": orphaned: PR #175 merged into it, no PR out of it is merged or open, and its changes are not on main (head s6-head)",
		s7 + ": open PR #270 targets it, but its story PR #223 merged on 2026-10-28 12:00Z; retarget it to main",
	}
	if got := texts(fs); !slices.Equal(got, want) {
		t.Fatalf("findings:\n got %q\nwant %q", got, want)
	}
}

// A push after the story PR with no PR behind it is still caught.
func TestPushAfterStoryPR(t *testing.T) {
	in := milestone1()
	in.Branches[s7] = "s7-pushed"
	landed, _ := landedAt("s5-story..s5-late")
	fs, _, _ := Check(in, landed)
	if !slices.ContainsFunc(fs, func(f Finding) bool {
		return f.Branch == s7 && strings.Contains(f.Text, "commits pushed to it afterwards")
	}) {
		t.Fatalf("want S7 flagged, got %q", texts(fs))
	}
}

// resolved.yaml accepts a branch at one head only.
func TestResolvedPinsHead(t *testing.T) {
	in := milestone1()
	in.Resolved = []Resolved{{Branch: s6, Head: "s6-head", Why: "abandoned, TICK-999"}}
	landed, _ := landedAt("s7-head..s7-head", "s5-story..s5-late")
	fs, acc, _ := Check(in, landed)
	if len(fs) != 0 || len(acc) != 1 || !strings.Contains(acc[0], "TICK-999") {
		t.Fatalf("want S6 accepted, got findings %q accepted %q", texts(fs), acc)
	}
	in.Branches[s6] = "s6-newer"
	fs, acc, _ = Check(in, landed)
	if len(fs) != 1 || fs[0].Branch != s6 || len(acc) != 0 {
		t.Fatalf("a new push must flag S6 again, got findings %q accepted %q", texts(fs), acc)
	}
}

func TestLandedErrorStops(t *testing.T) {
	_, _, err := Check(milestone1(), func(string, string, time.Time) (bool, error) { return false, fmt.Errorf("boom") })
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want the landed error, got %v", err)
	}
}

func TestStateAt(t *testing.T) {
	p := mergedPR(1, "a", "b", "o", "10:00", "12:00")
	for _, c := range []struct{ t, want string }{{"09:00", ""}, {"11:00", "OPEN"}, {"12:00", "MERGED"}} {
		if got := p.stateAt(at(c.t)); got != c.want {
			t.Errorf("at %s: got %q, want %q", c.t, got, c.want)
		}
	}
	c := PR{State: "CLOSED", CreatedAt: at("10:00"), ClosedAt: at("11:00")}
	if c.stateAt(at("10:30")) != "OPEN" || c.stateAt(at("11:30")) != "CLOSED" || c.stateAt(time.Time{}) != "CLOSED" {
		t.Error("closed PR replays wrong")
	}
}
