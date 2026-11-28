package main

// The -pins mode (TICK-1194): every uses: of a workflow names its action by
// full commit SHA, with the version tag in a trailing comment:
//
//	- uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
//
// A tag or branch can be moved by whoever controls the action's repository,
// and the next run executes whatever it points at, with the job's token and
// secrets. A commit SHA cannot be moved. The comment is what a reviewer and
// oldSetupGo read as the version. A local action (./path) is code of this
// repository and needs no pin; a docker:// image must be pinned by digest.

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// fullSHA is a full 40-hex git commit id. A short SHA is not a pin: GitHub
// rejects it, and a prefix could be made to collide.
var fullSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// actionRef is owner/repo[/path]@ref.
var actionRef = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(/[^@\s]+)?@([^@\s]+)$`)

// dockerDigest is docker://image@sha256:<64 hex>.
var dockerDigest = regexp.MustCompile(`^docker://[^@\s]+@sha256:[0-9a-f]{64}$`)

// versionTag is the comment of a SHA pin: v6, v6.5, v6.5.0.
var versionTag = regexp.MustCompile(`^v[0-9]+(\.[0-9]+)*$`)

// pinComment returns the version a uses: node's trailing comment names,
// "# v6.5.0" read as "v6.5.0", or "" when the comment is not one tag.
func pinComment(n yaml.Node) string {
	c := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(n.LineComment), "#"))
	if !versionTag.MatchString(c) {
		return ""
	}
	return c
}

// unpinned returns one problem per uses: of the workflow at path that is not
// pinned as the -pins mode requires, by file and line.
func unpinned(path string) ([]string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a path the caller chose
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var problems []string
	walkUses(&doc, func(v *yaml.Node) {
		if p := pinProblem(v); p != "" {
			problems = append(problems, fmt.Sprintf("%s:%d: uses: %s %s (TICK-1194)", path, v.Line, v.Value, p))
		}
	})
	return problems, nil
}

// walkUses calls f with the value of every uses: key in the document: a
// step's and a job's (a reusable workflow) alike.
func walkUses(n *yaml.Node, f func(*yaml.Node)) {
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			if k, v := n.Content[i], n.Content[i+1]; k.Value == "uses" && v.Kind == yaml.ScalarNode {
				f(v)
			}
		}
	}
	for _, c := range n.Content {
		walkUses(c, f)
	}
}

// pinProblem is what is wrong with one uses: value, or "".
func pinProblem(v *yaml.Node) string {
	ref := v.Value
	switch {
	case strings.HasPrefix(ref, "./"):
		return ""
	case strings.HasPrefix(ref, "docker://"):
		if dockerDigest.MatchString(ref) {
			return ""
		}
		return "is not pinned by digest: want docker://image@sha256:<64 hex>"
	}
	m := actionRef.FindStringSubmatch(ref)
	if m == nil {
		return "is not owner/repo@ref"
	}
	if !fullSHA.MatchString(m[2]) {
		return "is not pinned by full commit SHA: want owner/repo@<40 hex> # vX.Y.Z"
	}
	if pinComment(*v) == "" {
		return "lacks its version: want a trailing comment # vX.Y.Z naming the tag the SHA is"
	}
	return ""
}

// runPins is run's -pins mode over the given workflow files.
func runPins(paths []string, stdout, stderr io.Writer) int {
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "check-ci-steps: -pins needs one or more workflow files")
		return 2
	}
	var problems []string
	for _, p := range paths {
		ps, err := unpinned(p)
		if err != nil {
			fmt.Fprintln(stderr, "check-ci-steps:", err)
			return 2
		}
		problems = append(problems, ps...)
	}
	for _, p := range problems {
		fmt.Fprintln(stderr, "check-ci-steps:", p)
	}
	if len(problems) > 0 {
		fmt.Fprintln(stderr, "check-ci-steps: pin every action by full commit SHA with its tag in a trailing comment (TICK-1194)")
		return 1
	}
	fmt.Fprintf(stdout, "check-ci-steps: ok: every uses: of %d workflows is pinned by SHA\n", len(paths))
	return 0
}
