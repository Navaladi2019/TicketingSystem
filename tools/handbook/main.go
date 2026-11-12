// Command handbook builds the product handbook: docs/handbook/src/**/*.md
// into docs/handbook/**/*.html, pages that open from file:// with no network
// request (TICK-719, contract C-HB-1).
//
// Usage: handbook [-root dir] [-sha <commit>] [-date YYYY-MM-DD] [-check]
//
// The footer stamp is an input: -sha, else $HANDBOOK_SHA, else
// `git merge-base HEAD origin/main`; -date likewise, else the commit's date.
// With -check nothing is written: the handbook is rebuilt in memory and
// compared with the committed files (footer stamp ignored); the stamped sha
// must be an ancestor of HEAD.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run returns the exit code: 0 ok, 1 the build or the check found problems, 2 usage or environment.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("handbook", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	sha := fs.String("sha", os.Getenv("HANDBOOK_SHA"), "main commit the content is generated against (default $HANDBOOK_SHA, else git merge-base HEAD origin/main)")
	date := fs.String("date", os.Getenv("HANDBOOK_DATE"), "date of that commit, YYYY-MM-DD (default $HANDBOOK_DATE, else the commit's date)")
	check := fs.Bool("check", false, "rebuild in memory and fail when the committed handbook is stale")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	g := gitCLI{dir: *root}
	cfg := Config{Root: *root, Scanner: noRules}

	if *check {
		problems, err := Check(cfg, g)
		if err != nil {
			fmt.Fprintln(stderr, "handbook:", err)
			return 1
		}
		for _, p := range problems {
			fmt.Fprintln(stderr, "handbook:", p)
		}
		if len(problems) > 0 {
			return 1
		}
		fmt.Fprintln(stdout, "handbook: ok")
		return 0
	}

	if *sha == "" {
		out, err := g.run("merge-base", "HEAD", "origin/main")
		if err != nil {
			fmt.Fprintln(stderr, "handbook: cannot find the main commit (git merge-base HEAD origin/main failed); pass -sha or set HANDBOOK_SHA:", err)
			return 2
		}
		*sha = out
	}
	if *date == "" {
		out, err := g.run("show", "-s", "--format=%cs", *sha)
		if err != nil {
			fmt.Fprintln(stderr, "handbook: cannot read the date of", *sha+"; pass -date or set HANDBOOK_DATE:", err)
			return 2
		}
		*date = out
	}
	cfg.Sha, cfg.Date = *sha, *date
	out, err := Build(cfg)
	if err != nil {
		fmt.Fprintln(stderr, "handbook:", err)
		return 1
	}
	if err := Write(*root, out); err != nil {
		fmt.Fprintln(stderr, "handbook:", err)
		return 2
	}
	fmt.Fprintf(stdout, "handbook: wrote %d files under docs/handbook\n", len(out))
	return 0
}

// noRules is the sensitivity scanner with an empty rule set; HB-17 (TICK-735)
// supplies the rules.
func noRules(string, []byte) []string { return nil }

type gitCLI struct{ dir string }

func (g gitCLI) run(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // fixed git subcommands; arguments are a commit id read from flags or git itself
	cmd.Dir = g.dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// IsAncestor is `git merge-base --is-ancestor <sha> HEAD`.
func (g gitCLI) IsAncestor(sha string) (bool, error) {
	cmd := exec.CommandContext(context.Background(), "git", "merge-base", "--is-ancestor", sha, "HEAD") //nolint:gosec // sha comes from a footer matched by a hex regexp
	cmd.Dir = g.dir
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git merge-base --is-ancestor: %w (CI lint needs fetch-depth: 0)", err)
}
