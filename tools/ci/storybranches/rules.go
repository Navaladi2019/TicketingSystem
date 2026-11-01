package main

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

// storyRe matches the branch names the check looks at: TICK-<n>/<slug>.
var storyRe = regexp.MustCompile(`^TICK-[0-9]+/`)

// issueRe matches a story issue identifier in closed.yaml.
var issueRe = regexp.MustCompile(`^TICK-[0-9]+$`)

// PR is the part of a GitHub pull request the rules need (gh pr list --json).
type PR struct {
	Number     int       `json:"number"`
	State      string    `json:"state"` // OPEN, CLOSED or MERGED
	Head       string    `json:"headRefName"`
	Base       string    `json:"baseRefName"`
	HeadOid    string    `json:"headRefOid"`
	CreatedAt  time.Time `json:"createdAt"`
	MergedAt   time.Time `json:"mergedAt"`
	ClosedAt   time.Time `json:"closedAt"`
	IsCrossRep bool      `json:"isCrossRepository"`
}

// stateAt is the PR's state at t: OPEN, MERGED or CLOSED, or "" when it did
// not exist yet. A zero t means now.
func (p PR) stateAt(t time.Time) string {
	if t.IsZero() {
		return p.State
	}
	switch {
	case p.CreatedAt.After(t):
		return ""
	case !p.MergedAt.IsZero() && !p.MergedAt.After(t):
		return "MERGED"
	case !p.ClosedAt.IsZero() && !p.ClosedAt.After(t):
		return "CLOSED"
	}
	return "OPEN"
}

// Resolved accepts a story branch at one exact head: a later push flags it
// again (resolved.yaml).
type Resolved struct {
	Branch string `yaml:"branch"`
	Head   string `yaml:"head"`
	Why    string `yaml:"why"`
}

// Closed records that a story issue is done or cancelled (closed.yaml). CI
// cannot read Multica, so this file is the only "story closed" signal besides
// a merged story PR (TICK-379).
type Closed struct {
	Branch string `yaml:"branch"`
	Status string `yaml:"status"` // done or cancelled
	Issue  string `yaml:"issue"`
}

// Landed reports whether every change head carries since base is on main:
// merging head into some first-parent main commit made at or after since
// leaves that commit's tree unchanged, taking base as the merge base ("" lets
// git pick it; a zero since means head's own commit time). Trees, not
// ancestry, since a squash merge keeps the tree and drops the commits. Any
// main commit since, not only the tip: once the changes land, main may edit
// the same files again, and merging the old branch into the tip would then
// conflict or differ although nothing is missing (TICK-322 review).
type Landed func(base, head string, since time.Time) (bool, error)

// Input is everything the rules look at.
type Input struct {
	Main     string            // the default branch, "main"
	Branches map[string]string // remote branch name -> head sha
	PRs      []PR
	AsOf     time.Time // zero: now
	Resolved []Resolved
	Closed   []Closed
}

// Finding is one problem; Branch groups them in the report.
type Finding struct {
	Branch string
	Text   string
}

func (f Finding) String() string { return f.Branch + ": " + f.Text }

// Check applies the two rules (docs/quality/ci-pipeline.md, "Story branches"):
//
//  1. A story branch whose changes are not on main, while its story PR is
//     merged (changes pushed or merged after the story PR), or while it has
//     no PR out at all and its story is closed in closed.yaml (orphaned).
//  2. An open PR that targets a story branch whose story PR is merged.
//
// A story branch is a TICK-<n>/<slug> branch that some merged PR targets: task
// PRs merge into it. Its story PR is the newest merged PR out of it; an open
// PR out of it means the story is still in flight and rule 1 waits. So does
// a branch with no PR out of it whose story closed.yaml does not list: the
// story PR opens only after every task has merged (branching-strategy.md),
// so until then the branch is in flight, not orphaned (TICK-379).
func Check(in Input, landed Landed) (findings []Finding, accepted []string, err error) {
	res, err := Evaluate(in, landed)
	return res.Findings, res.Accepted, err
}

// InFlight is a story branch Check waits on: its story PR is open, or it has
// none yet and closed.yaml does not list its story. LastTask is the newest
// task PR merged into it, its age in the report (TICK-380).
type InFlight struct {
	Branch    string
	StoryPR   int // the open story PR, 0 when there is none
	LastTask  PR
	TaskCount int
}

// describe is an absolute date, not "N days ago": the report is diffed
// against the last one posted to the story-drift issue, and a moving count
// would re-post an unchanged report on every scheduled run.
func (f InFlight) describe() string {
	s := fmt.Sprintf("last task PR #%d merged on %s, %s merged into it", f.LastTask.Number, day(f.LastTask.MergedAt), countPRs(f.TaskCount))
	if f.StoryPR != 0 {
		return s + fmt.Sprintf(", story PR #%d open", f.StoryPR)
	}
	return s + ", no story PR yet"
}

func countPRs(n int) string {
	if n == 1 {
		return "1 task PR"
	}
	return fmt.Sprintf("%d task PRs", n)
}

// Result is what Evaluate finds: findings fail the job, Accepted and InFlight
// are information only.
type Result struct {
	Findings []Finding
	Accepted []string
	InFlight []InFlight
}

// Evaluate is Check that also reports the in-flight story branches.
func Evaluate(in Input, landed Landed) (Result, error) {
	var res Result
	var findings []Finding
	var accepted []string
	into := map[string][]PR{} // base -> PRs targeting it
	out := map[string][]PR{}  // head -> PRs out of it
	for _, p := range in.PRs {
		if p.IsCrossRep || p.stateAt(in.AsOf) == "" {
			continue
		}
		into[p.Base] = append(into[p.Base], p)
		out[p.Head] = append(out[p.Head], p)
	}

	var stories []string
	for b := range in.Branches {
		if b == in.Main || !storyRe.MatchString(b) {
			continue
		}
		if slices.ContainsFunc(into[b], func(p PR) bool { return p.stateAt(in.AsOf) == "MERGED" }) {
			stories = append(stories, b)
		}
	}
	sort.Strings(stories)

	for _, b := range stories {
		head := in.Branches[b]
		if i := slices.IndexFunc(in.Resolved, func(r Resolved) bool { return r.Branch == b && r.Head == head }); i >= 0 {
			accepted = append(accepted, fmt.Sprintf("%s at %s: %s", b, short(head), in.Resolved[i].Why))
			continue
		}

		var open, story *PR
		for i := range out[b] {
			p := &out[b][i]
			switch p.stateAt(in.AsOf) {
			case "OPEN":
				open = p
			case "MERGED":
				if story == nil || p.MergedAt.After(story.MergedAt) {
					story = p
				}
			}
		}
		tasks := merged(into[b], in.AsOf, time.Time{})
		flight := InFlight{Branch: b, LastTask: tasks[len(tasks)-1], TaskCount: len(tasks)}
		if open != nil {
			flight.StoryPR = open.Number
			res.InFlight = append(res.InFlight, flight)
			continue // in flight: its story PR is open
		}

		if story == nil {
			ci := slices.IndexFunc(in.Closed, func(c Closed) bool { return c.Branch == b })
			if ci < 0 {
				res.InFlight = append(res.InFlight, flight)
				continue // in flight: tasks still merging, story PR not open yet
			}
			ok, err := landed("", head, time.Time{})
			if err != nil {
				return Result{}, fmt.Errorf("%s: %w", b, err)
			}
			if !ok {
				findings = append(findings, Finding{b, fmt.Sprintf(
					"orphaned: story %s is %s (closed.yaml), %s merged into it, no PR out of it is merged or open, and its changes are not on %s (head %s)",
					in.Closed[ci].Issue, in.Closed[ci].Status, list(tasks), in.Main, short(head))})
			}
			continue
		}

		// Rule 2: an open PR into a story branch whose story PR is merged.
		for _, p := range into[b] {
			if p.stateAt(in.AsOf) == "OPEN" {
				findings = append(findings, Finding{b, fmt.Sprintf(
					"open PR #%d targets it, but its story PR #%d merged on %s; retarget it to %s",
					p.Number, story.Number, day(story.MergedAt), in.Main)})
			}
		}

		ok, err := landed(story.HeadOid, head, story.MergedAt)
		if err != nil {
			return Result{}, fmt.Errorf("%s: %w", b, err)
		}
		if !ok {
			late := merged(into[b], in.AsOf, story.MergedAt)
			why := "commits pushed to it afterwards"
			if len(late) > 0 {
				why = list(late) + " merged into it afterwards"
			}
			findings = append(findings, Finding{b, fmt.Sprintf(
				"story PR #%d merged on %s, but changes since then are not on %s: %s (head %s, story PR head %s)",
				story.Number, day(story.MergedAt), in.Main, why, short(head), short(story.HeadOid))})
		}
	}
	res.Findings, res.Accepted = findings, accepted
	return res, nil
}

// merged lists the PRs merged by asOf and after since, oldest first.
func merged(prs []PR, asOf, since time.Time) []PR {
	var m []PR
	for _, p := range prs {
		if p.stateAt(asOf) == "MERGED" && p.MergedAt.After(since) {
			m = append(m, p)
		}
	}
	sort.Slice(m, func(i, j int) bool { return m[i].MergedAt.Before(m[j].MergedAt) })
	return m
}

func list(prs []PR) string {
	if len(prs) == 0 {
		return "no PR"
	}
	s := make([]string, len(prs))
	for i, p := range prs {
		s[i] = fmt.Sprintf("#%d", p.Number)
	}
	if len(s) == 1 {
		return "PR " + s[0]
	}
	return "PRs " + strings.Join(s, ", ")
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func day(t time.Time) string { return t.UTC().Format("2006-01-02 15:04Z") }
