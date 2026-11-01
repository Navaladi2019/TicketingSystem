// Command storybranches fails when a story branch's work is not on main
// (TICK-322): a story branch TICK-<n>/<slug> whose changes are missing from
// main although its story PR is merged, or that has no PR out of it although
// closed.yaml records its story as done or cancelled, and an open PR that targets a story branch already merged into main. The
// rules are in rules.go; docs/quality/ci-pipeline.md ("Story branches")
// explains them.
//
// CI cannot read Multica, so a merged story PR and closed.yaml are its only
// story-closed signals.
// Squash merges break ancestry, so "on main" compares trees: a branch is on
// main when `git merge-tree --write-tree --merge-base=<story PR head> main
// <branch>` yields main's own tree, that is, merging what the branch gained
// since its story PR would change nothing.
//
// It reads the remote's branches (git ls-remote) and every pull request
// (gh pr list), so it needs git >= 2.40, gh and GH_TOKEN. -as-of replays the
// pull requests as they stood at a time (branch heads stay current).
// -report writes the findings as Markdown for the CI issue.
//
// Usage: storybranches [-remote origin] [-main main] [-as-of RFC3339]
// [-resolved tools/ci/storybranches/resolved.yaml]
// [-closed tools/ci/storybranches/closed.yaml] [-report file]
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

func main() {
	remote := flag.String("remote", "origin", "git remote holding the branches")
	mainBranch := flag.String("main", "main", "the default branch")
	asOf := flag.String("as-of", "", "judge pull requests as they stood at this RFC 3339 time (default now)")
	resolvedFile := flag.String("resolved", "tools/ci/storybranches/resolved.yaml", "story branch heads accepted as they are")
	closedFile := flag.String("closed", "tools/ci/storybranches/closed.yaml", "story branches whose story issue is done or cancelled")
	report := flag.String("report", "", "write the findings as Markdown to this file")
	flag.Parse()

	if err := run(*remote, *mainBranch, *asOf, *resolvedFile, *closedFile, *report); err != nil {
		var f failed
		if errors.As(err, &f) {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "check-story-branches:", err)
		os.Exit(2)
	}
}

type failed int

func (f failed) Error() string { return fmt.Sprintf("%d findings", int(f)) }

func run(remote, mainBranch, asOf, resolvedFile, closedFile, report string) error {
	in := Input{Main: mainBranch}
	if asOf != "" {
		t, err := time.Parse(time.RFC3339, asOf)
		if err != nil {
			return fmt.Errorf("-as-of: %w", err)
		}
		in.AsOf = t
	}
	var err error
	if in.Resolved, err = readResolved(resolvedFile); err != nil {
		return err
	}
	if in.Closed, err = readClosed(closedFile); err != nil {
		return err
	}
	if _, err = git("fetch", "--quiet", "--prune", remote); err != nil {
		return err
	}
	if in.Branches, err = branches(remote); err != nil {
		return err
	}
	if in.PRs, err = pullRequests(); err != nil {
		return err
	}
	mainRef := remote + "/" + mainBranch
	findings, accepted, err := Check(in, func(base, head string, since time.Time) (bool, error) {
		return landed(remote, mainRef, base, head, since)
	})
	if err != nil {
		return err
	}

	when := "now"
	if !in.AsOf.IsZero() {
		when = "as of " + day(in.AsOf)
	}
	for _, a := range accepted {
		fmt.Println("check-story-branches: accepted:", a)
	}
	for _, f := range findings {
		fmt.Fprintln(os.Stderr, "check-story-branches:", f)
	}
	if report != "" {
		if err := os.WriteFile(report, []byte(markdown(findings, accepted, mainRef, when)), 0o600); err != nil {
			return err
		}
	}
	if len(findings) > 0 {
		fmt.Fprintf(os.Stderr, "check-story-branches: %d findings (%s); see \"Story branches\" in docs/quality/ci-pipeline.md\n", len(findings), when)
		return failed(len(findings))
	}
	fmt.Printf("check-story-branches: ok: every merged story branch is on %s (%s)\n", mainRef, when)
	return nil
}

func markdown(findings []Finding, accepted []string, mainRef, when string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "`make check-story-branches` against `%s`, %s: %d findings.\n\n", mainRef, when, len(findings))
	for _, f := range findings {
		fmt.Fprintf(&b, "- `%s`: %s\n", f.Branch, f.Text)
	}
	if len(accepted) > 0 {
		b.WriteString("\nAccepted in `tools/ci/storybranches/resolved.yaml`:\n\n")
		for _, a := range accepted {
			fmt.Fprintf(&b, "- %s\n", a)
		}
	}
	b.WriteString("\nWhat each finding means and how to clear it: \"Story branches\" in `docs/quality/ci-pipeline.md`.\n")
	return b.String()
}

// landed: see Landed. A merge conflict counts as not on main at that commit.
// It tries main's first-parent commits newest first, back to an hour before
// since (committer clocks), at most maxCandidates of them.
func landed(remote, mainRef, base, head string, since time.Time) (bool, error) {
	if base != "" {
		if _, err := git("cat-file", "-e", base+"^{commit}"); err != nil {
			// The story PR's head is gone from every branch (force-pushed or
			// deleted); GitHub keeps it under refs/pull/*, so fetch it by sha.
			if _, err := git("fetch", "--quiet", remote, base); err != nil {
				return false, err
			}
		}
	}
	if since.IsZero() {
		at, err := git("log", "-1", "--format=%cI", head)
		if err != nil {
			return false, err
		}
		if since, err = time.Parse(time.RFC3339, at); err != nil {
			return false, fmt.Errorf("commit time of %s: %w", head, err)
		}
	}
	list, err := git("rev-list", "--first-parent", fmt.Sprintf("--max-count=%d", maxCandidates),
		"--since="+since.Add(-time.Hour).UTC().Format(time.RFC3339), mainRef)
	if err != nil {
		return false, err
	}
	candidates := strings.Fields(list)
	if len(candidates) == 0 {
		candidates = []string{mainRef}
	}
	for _, c := range candidates {
		ok, err := mergesClean(base, c, head)
		if ok || err != nil {
			return ok, err
		}
	}
	return false, nil
}

// maxCandidates bounds the main commits landed tries per branch.
const maxCandidates = 500

// mergesClean reports whether merging head into commit c (merge base base, or
// git's when "") yields c's own tree.
func mergesClean(base, c, head string) (bool, error) {
	args := []string{"merge-tree", "--write-tree", "--no-messages"}
	if base != "" {
		args = append(args, "--merge-base="+base)
	}
	cmd := exec.CommandContext(context.Background(), "git", append(args, c, head)...) //nolint:gosec // git with refs and shas from git and gh
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil // conflicts
	}
	if err != nil {
		return false, fmt.Errorf("git %s: %w: %s", strings.Join(cmd.Args[1:], " "), err, stderr.String())
	}
	tree, _, _ := strings.Cut(out.String(), "\n")
	cTree, err := git("rev-parse", c+"^{tree}")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(tree) == cTree, nil
}

func branches(remote string) (map[string]string, error) {
	out, err := git("ls-remote", "--heads", remote)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		sha, ref, ok := strings.Cut(line, "\t")
		if ok {
			m[strings.TrimPrefix(ref, "refs/heads/")] = sha
		}
	}
	return m, nil
}

func pullRequests() ([]PR, error) {
	cmd := exec.CommandContext(context.Background(), "gh", "pr", "list", "--state", "all", "--limit", "5000", "--json",
		"number,state,headRefName,baseRefName,headRefOid,createdAt,mergedAt,closedAt,isCrossRepository")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr list: %w: %s", err, stderr.String())
	}
	var prs []PR
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("gh pr list: %w", err)
	}
	return prs, nil
}

func readResolved(path string) ([]Resolved, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a path the caller chose
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f struct {
		Resolved []Resolved `yaml:"resolved"`
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, r := range f.Resolved {
		if r.Branch == "" || len(r.Head) != 40 || r.Why == "" {
			return nil, fmt.Errorf("%s: %q needs branch, a full 40-character head sha and why", path, r.Branch)
		}
	}
	return f.Resolved, nil
}

// readClosed reads closed.yaml; a missing file lists no closed story.
func readClosed(path string) ([]Closed, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a path the caller chose
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseClosed(path, data)
}

func parseClosed(path string, data []byte) ([]Closed, error) {
	var f struct {
		Closed []Closed `yaml:"closed"`
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, c := range f.Closed {
		if !storyRe.MatchString(c.Branch) || (c.Status != "done" && c.Status != "cancelled") || !issueRe.MatchString(c.Issue) {
			return nil, fmt.Errorf("%s: %q needs a TICK-<n>/<slug> branch, status done or cancelled, and issue TICK-<n>", path, c.Branch)
		}
	}
	return f.Closed, nil
}

func git(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // git with arguments from this file
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}
