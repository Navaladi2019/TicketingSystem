// Command openapi-compat fails when api/openapi.yaml removes or narrows
// anything that api/openapi.yaml on the base branch offered: the additive-only
// rule for /api/v1.
//
// Usage: openapi-compat <base-spec> <new-spec> [<approved-file>]
//
// The approved file lists breaking changes the owner of the contract accepted,
// one per line: the exact "breaking:" text, then " # " and the issue that
// approved it. A line whose reason does not start with an issue key, and a blank or "#" line, is ignored, so
// an entry cannot be added without naming its issue. A listed change no longer
// reported is harmless; an unlisted one still fails.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/apidiff"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run returns the exit code: 0 additive, 1 breaking, 2 usage or unreadable input.
func run(args []string) int {
	if len(args) != 2 && len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: openapi-compat <base-spec> <new-spec> [<approved-file>]")
		return 2
	}
	base, err := os.ReadFile(args[0]) //nolint:gosec // a CLI reads the paths its caller names
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	next, err := os.ReadFile(args[1]) //nolint:gosec // a CLI reads the paths its caller names
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	problems, err := apidiff.Check(base, next)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if len(args) == 3 {
		approved, err := os.ReadFile(args[2]) //nolint:gosec // a CLI reads the paths its caller names
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		problems = withoutApproved(problems, approved)
	}
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "breaking:", p)
	}
	if len(problems) > 0 {
		return 1
	}
	return 0
}

// withoutApproved drops each problem the approved file lists with a reason.
func withoutApproved(problems []string, approved []byte) []string {
	ok := map[string]bool{}
	for _, line := range strings.Split(string(approved), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		change, reason, found := strings.Cut(line, " # ")
		if found && hasIssueKey(strings.TrimSpace(reason)) {
			ok[strings.TrimSpace(change)] = true
		}
	}
	var out []string
	for _, p := range problems {
		if !ok[p] {
			out = append(out, p)
		}
	}
	return out
}

// hasIssueKey reports whether a reason starts with the issue that carries the
// approval, such as TICK-939.
func hasIssueKey(reason string) bool {
	return regexp.MustCompile(`^[A-Z]+-[0-9]+\b`).MatchString(reason)
}
