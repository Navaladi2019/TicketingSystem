package main

// The -copy mode (TICK-1149, ADR-001-E4 vulncheck contract rule 9 as amended
// by ADR-001-A7): the whole of a workflow, comment lines aside, must equal a
// checked-in copy byte for byte, and no step of the workflow may run make
// (A7.3): the scan jobs call govulncheck directly with their environment
// pinned in the step's env:, so no makefile can change what they scan. An
// edit of one file alone (a step if:, an exit 0, a shell:, a paths filter, a
// top-level env: or defaults:, a second job reporting the gate's name, a
// dropped GOTOOLCHAIN: local) fails lint. A deliberate edit of the workflow
// and the copy together passes unless it adds make: that is ADR RS3, caught
// by review, not by this.

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
)

// checkCopy returns the problems of workflowPath against copyPath, or an
// error when a file cannot be read or the workflow is not YAML.
func checkCopy(workflowPath, copyPath string) ([]string, error) {
	wf, err := os.ReadFile(workflowPath) //nolint:gosec // a path the caller chose
	if err != nil {
		return nil, err
	}
	cp, err := os.ReadFile(copyPath) //nolint:gosec // a path the caller chose
	if err != nil {
		return nil, err
	}
	var problems []string
	if p := compareUncommented(wf, cp); p != "" {
		problems = append(problems, fmt.Sprintf("%s differs from %s, comment lines aside: %s", workflowPath, copyPath, p))
	}
	mk, err := makeSteps(workflowPath)
	if err != nil {
		return nil, err
	}
	return append(problems, mk...), nil
}

// makeWord is make as a word of a shell line (ADR-001-A7 A7.3): at the start
// of the line or after a blank, ;, &, | or (, and followed by a blank or the
// end of the line. cmake-tool, automake and make-x are not make.
var makeWord = regexp.MustCompile(`(^|[\s;&|(])make(\s|$)`)

// makeSteps returns one problem per step of workflowPath whose run: has a
// line that runs make, by job id and 1-based step number.
func makeSteps(workflowPath string) ([]string, error) {
	return nil, nil // TICK-1187: red first
}

// numbered is a line of a file and its 1-based line number.
type numbered struct {
	n    int
	text []byte
}

// blockHeader is a line that starts a YAML block scalar (`run: |`, `- >-`,
// `key: |2 # note`). A plain scalar that only looks like one makes the
// comparison stricter, never looser.
var blockHeader = regexp.MustCompile(`(?:^|:\s|-\s)\s*[|>][1-9+-]{0,2}\s*(?:#.*)?$`)

// uncommented returns the lines of data whose first non-blank character is
// not #. Every other byte counts: blank lines, indentation, trailing comments
// and line endings. A # line inside a block scalar (`run: |`) is content, not
// a comment, and counts too (TICK-1167 AC 3): a line indented deeper than
// the block's header line belongs to the block.
func uncommented(data []byte) []numbered {
	var out []numbered
	block := -1 // the indentation of the open block scalar's header line
	for i, line := range bytes.SplitAfter(data, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		t := bytes.TrimLeft(line, " \t")
		indent := len(line) - len(t)
		blank := len(bytes.TrimSpace(t)) == 0
		if block >= 0 && (blank || indent > block) {
			out = append(out, numbered{i + 1, line})
			continue
		}
		block = -1
		if len(t) > 0 && t[0] == '#' {
			continue
		}
		if blockHeader.Match(bytes.TrimRight(t, "\r\n")) {
			block = indent
		}
		out = append(out, numbered{i + 1, line})
	}
	return out
}

// compareUncommented returns "" when a and b are equal once comment lines are
// dropped, else the first differing line of each.
func compareUncommented(a, b []byte) string {
	la, lb := uncommented(a), uncommented(b)
	for i := 0; ; i++ {
		switch {
		case i >= len(la) && i >= len(lb):
			return ""
		case i >= len(la):
			return fmt.Sprintf("workflow ends, copy line %d is %q", lb[i].n, lb[i].text)
		case i >= len(lb):
			return fmt.Sprintf("workflow line %d is %q, copy ends", la[i].n, la[i].text)
		case !bytes.Equal(la[i].text, lb[i].text):
			return fmt.Sprintf("workflow line %d is %q, copy line %d is %q", la[i].n, la[i].text, lb[i].n, lb[i].text)
		}
	}
}
