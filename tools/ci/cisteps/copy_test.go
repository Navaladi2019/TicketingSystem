package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	repoWorkflow = ".github/workflows/govulncheck.yml"
	repoCopy     = "tools/ci/cisteps/govulncheck.copy.yml"
)

// AC 1 (TICK-1149), TICK-1187 Test 5: the committed workflow and its copy
// pass the -copy check, and no step runs make. TestRepository covers ci.yml
// only.
func TestRepositoryGovulncheckCopy(t *testing.T) {
	problems, err := checkCopy(repo(repoWorkflow), repo(repoCopy))
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("%q", problems)
	}
}

// edit is one change to the text of the real workflow.
type edit func(t *testing.T, wf string) string

// lineIndex returns the index of the first line of lines matching re.
func lineIndex(t *testing.T, lines []string, re string) int {
	t.Helper()
	r := regexp.MustCompile(re)
	for i, l := range lines {
		if r.MatchString(l) {
			return i
		}
	}
	t.Fatalf("govulncheck.yml has no line matching %q; update the V-3 edit to the new workflow", re)
	return -1
}

// insertAt inserts add before line re (after it when after is true).
func insertAt(re string, after bool, add ...string) edit {
	return func(t *testing.T, wf string) string {
		t.Helper()
		lines := strings.Split(wf, "\n")
		i := lineIndex(t, lines, re)
		if after {
			i++
		}
		return strings.Join(append(lines[:i], append(add, lines[i:]...)...), "\n")
	}
}

// atScanLine replaces the govulncheck line of the job that scans target's
// module (vulncheck is vulncheck-root, vulncheck-temporal-server is itself)
// by f(indent of run:, the line without its indent), so an edit can keep the
// command and change only what it tests.
func atScanLine(target string, f func(ind, run string) []string) edit {
	job := target
	if target == "vulncheck" {
		job = "vulncheck-root"
	}
	return func(t *testing.T, wf string) string {
		t.Helper()
		lines := strings.Split(wf, "\n")
		j := lineIndex(t, lines, `^  `+regexp.QuoteMeta(job)+`:\s*$`)
		i := j + 1 + lineIndex(t, lines[j+1:], `^\s+run: go run golang\.org/x/vuln/cmd/govulncheck@\S+ \./\.\.\.\s*$`)
		ind := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " "))]
		return strings.Join(append(lines[:i], append(f(ind, strings.TrimLeft(lines[i], " ")), lines[i+1:]...)...), "\n")
	}
}

// replaceLine replaces the first line matching re by with (none: deleted).
func replaceLine(re string, with ...string) edit {
	return func(t *testing.T, wf string) string {
		t.Helper()
		lines := strings.Split(wf, "\n")
		i := lineIndex(t, lines, re)
		return strings.Join(append(lines[:i], append(with, lines[i+1:]...)...), "\n")
	}
}

// v3Edits are contract test V-3's edits of govulncheck.yml alone (TICK-1149
// AC 2 a-i, TICK-1150 AC 1 the gate cases). Each must fail check-ci-steps.
type v3Edit struct {
	name string
	edit edit
}

func v3Edits() []v3Edit {
	return []v3Edit{
		{"a step if", atScanLine("vulncheck", func(ind, run string) []string {
			return []string{ind + "if: github.actor == 'nobody'", ind + run}
		})},
		{"b job if", insertAt(`^jobs:\s*$`, true, "  vulncheck-root-gone:", "    if: github.event_name == 'never'", "    runs-on: ubuntu-latest", "    steps:", "      - run: go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...")},
		{"b job if on the scanning job", replaceLine(`^    if: needs\.vulncheck-detect\.outputs\.root == 'true'\s*$`, "    if: github.event_name == 'never'")},
		{"c exit 0 before the scan line", atScanLine("vulncheck", func(ind, run string) []string {
			return []string{ind + "run: |", ind + "  exit 0", ind + "  " + strings.TrimPrefix(run, "run: ")}
		})},
		{"d step shell", atScanLine("vulncheck-temporal-server", func(ind, run string) []string {
			return []string{ind + "shell: 'true {0}'", ind + run}
		})},
		{"e scan line in a printed heredoc", atScanLine("vulncheck", func(ind, run string) []string {
			return []string{ind + "run: |", ind + "  cat <<'EOF'", ind + "  " + strings.TrimPrefix(run, "run: "), ind + "  EOF"}
		})},
		{"f another job named vulncheck", insertAt(`^jobs:\s*$`, true, "  vulncheck-other:", "    name: vulncheck", "    runs-on: ubuntu-latest", "    steps:", "      - run: 'true'", "")},
		{"g top-level defaults.run.shell", insertAt(`^jobs:\s*$`, false, "defaults:", "  run:", "    shell: 'true {0}'", "")},
		{"h top-level env", insertAt(`^jobs:\s*$`, false, "env:", "  GOFLAGS: -mod=mod", "")},
		// TICK-1150 (contract rule 1): the paths filter is gone; re-adding one fails.
		{"i paths re-added", insertAt(`^    types: \[opened, synchronize, reopened\]\s*$`, true, "    paths:", "      - go.mod", "      - go.sum")},
		// TICK-1150 (contract rules 4 and 5, AC 1): the gate cases.
		{"gate without always()", replaceLine(`^    if: always\(\)\s*$`)},
		{"gate if: success()", replaceLine(`^    if: always\(\)\s*$`, "    if: success()")},
		{"gate name without the base test", replaceLine(`^    name: \$\{\{ github\.base_ref == 'main'`, "    name: vulncheck")},
		{"gate needs one scan dropped", replaceLine(`^    needs: \[vulncheck-detect, vulncheck-root, vulncheck-temporal-server\]\s*$`, "    needs: [vulncheck-detect, vulncheck-root]")},
		{"gate input from the wrong job", replaceLine(`^\s+TS_RESULT: `, "          TS_RESULT: ${{ needs.vulncheck-root.result }}")},
		{"scan without detect's if", replaceLine(`^    if: needs\.vulncheck-detect\.outputs\.root == 'true'\s*$`)},
		// TICK-1148 security review of PR #912: detect and the gate run
		// main's scripts, not the pull request's.
		{"gate checks out the pull request", replaceLine(`^          ref: main\s*$`, "          ref: ${{ github.head_ref }}")},
		{"detect runs the pull request's script", replaceLine(`^\s+git show origin/main:scripts/ci/vulncheck-detect\.sh`, "          cp scripts/ci/vulncheck-detect.sh \"$RUNNER_TEMP/vulncheck-detect.sh\"")},
		{"detect shallow", replaceLine(`^          fetch-depth: 0\s*$`, "          fetch-depth: 1")},
		{"scan line commented out", atScanLine("vulncheck", func(ind, run string) []string {
			return []string{ind + "run: 'true'", ind + "# " + run}
		})},
		{"trailing comment changed", atScanLine("vulncheck", func(ind, run string) []string {
			return []string{ind + run + " # || true"}
		})},
		// Still YAML (the workflow is parsed), and the same schedule.
		{"indentation changed", replaceLine(`^    - cron: `, "  - cron: '17 3 */2 * *'")},
		{"spacing after run: changed", atScanLine("vulncheck", func(ind, run string) []string {
			return []string{ind + strings.Replace(run, "run: ", "run:  ", 1)}
		})},
		// TICK-1187 Test 4 (ADR-001-A7 V-3): the scan's pinned environment
		// is part of the copy; dropping GOTOOLCHAIN: local from one scan
		// step lets go download a newer toolchain than go.mod's.
		{"scan step without GOTOOLCHAIN: local", replaceLine(`^\s+GOTOOLCHAIN: local\s*$`)},
		{"scan step GOFLAGS set", replaceLine(`^\s+GOFLAGS: ''\s*$`, "          GOFLAGS: -tags=nothing")},
		{"blank line added", insertAt(`^jobs:\s*$`, true, "")},
		// TICK-1167 AC 3: a # line inside a run: | block is the step's
		// input, not a YAML comment.
		{"# line inside a run: | block", insertAt(`^\s+git show origin/main:scripts/ci/vulncheck-detect\.sh`, true, `          # sh "$RUNNER_TEMP/stub.sh"`)},
		{"# line replaces a run: | line", replaceLine(`^\s+sh "\$RUNNER_TEMP/vulncheck-detect\.sh"`, `          # sh "$RUNNER_TEMP/vulncheck-detect.sh"`)},
		{"last line dropped", func(t *testing.T, wf string) string {
			return strings.TrimRight(wf, "\n")[:strings.LastIndex(strings.TrimRight(wf, "\n"), "\n")+1]
		}},
	}
}

func readRepo(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(repo(p))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// V-3 (AC 2 a-i): an edit of the workflow alone fails, through run, with exit
// 1 and the first differing line of each file; the copy is not touched.
func TestCopyCatchesWorkflowEdits(t *testing.T) {
	wf := readRepo(t, repoWorkflow)
	cp := repo(repoCopy)
	before := readRepo(t, repoCopy)
	for _, c := range v3Edits() {
		t.Run(c.name, func(t *testing.T) {
			edited := c.edit(t, wf)
			if edited == wf {
				t.Fatal("the edit changed nothing")
			}
			p := writeTemp(t, edited)
			var stdout, stderr bytes.Buffer
			if code := run([]string{"-workflow", p, "-copy", cp}, &stdout, &stderr); code != 1 {
				t.Fatalf("exit %d, want 1; stderr %q", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), "differs from") || !strings.Contains(stderr.String(), "workflow line") && !strings.Contains(stderr.String(), "workflow ends") {
				t.Errorf("stderr %q does not name the differing lines", stderr.String())
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout %q, want empty", stdout.String())
			}
		})
	}
	if readRepo(t, repoCopy) != before {
		t.Fatal("the test changed the copy")
	}
}

// V-3 (AC 2 k) documents RS3: the same edit of the workflow and the copy
// passes. The check stops an edit of one file, not a deliberate edit of both.
func TestCopyPassesTheSameEditOfBoth(t *testing.T) {
	wf, cp := readRepo(t, repoWorkflow), readRepo(t, repoCopy)
	for _, c := range v3Edits() {
		t.Run(c.name, func(t *testing.T) {
			problems, err := checkCopy(writeTemp(t, c.edit(t, wf)), writeTemp(t, c.edit(t, cp)))
			if err != nil {
				t.Fatal(err)
			}
			if len(problems) > 0 {
				t.Fatalf("%q", problems)
			}
		})
	}
}

// AC 3: a comment-only change to the workflow passes.
func TestCopyIgnoresCommentLines(t *testing.T) {
	wf := readRepo(t, repoWorkflow)
	for name, e := range map[string]edit{
		"header comment added":   func(_ *testing.T, wf string) string { return "# a new note\n" + wf },
		"comment removed":        func(_ *testing.T, wf string) string { return regexp.MustCompile(`(?m)^#.*\n`).ReplaceAllString(wf, "") },
		"indented comment added": insertAt(`^\s+run: go run golang\.org/x/vuln/cmd/govulncheck@`, false, "        # the root module"),
		// TICK-1167 AC 3: a comment after a run: | block, no deeper than
		// the block's header line, is a YAML comment.
		"comment after a run: | block": insertAt(`^  vulncheck-root:\s*$`, false, "        # after the block"),
	} {
		t.Run(name, func(t *testing.T) {
			edited := e(t, wf)
			if edited == wf {
				t.Fatal("the edit changed nothing")
			}
			problems, err := checkCopy(writeTemp(t, edited), repo(repoCopy))
			if err != nil {
				t.Fatal(err)
			}
			if len(problems) > 0 {
				t.Fatalf("%q", problems)
			}
		})
	}
}

func TestCompareUncommentedNamesTheLines(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{"x\ny\n", "x\ny\n", ""},
		{"# c\nx\n", "x\n# d\n", ""},
		{"# c\nx\ny\n", "x\nz\n", `workflow line 3 is "y\n", copy line 2 is "z\n"`},
		{"x\n", "x\ny\n", `workflow ends, copy line 2 is "y\n"`},
		{"x\ny", "x\n", `workflow line 2 is "y", copy ends`},
		{"x\n", "x", `workflow line 1 is "x\n", copy line 1 is "x"`},
		{"x\r\n", "x\n", `workflow line 1 is "x\r\n", copy line 1 is "x\n"`},
		// TICK-1167 AC 3: # lines inside block scalars count.
		{"run: |\n  a\n  # b\n", "run: |\n  a\n", `workflow line 3 is "  # b\n", copy ends`},
		{"- run: |\n    a\n\n    # b\nk: v\n", "- run: |\n    a\n\nk: v\n", `workflow line 4 is "    # b\n", copy line 4 is "k: v\n"`},
		{"run: | # note\n  # b\n", "run: | # note\n", `workflow line 2 is "  # b\n", copy ends`},
		{"run: >-\n  a\n# top\nk: v\n", "run: >-\n  a\nk: v\n", ""},
		{"run: echo a | b\n  # c\n", "run: echo a | b\n", ""},
	}
	for _, c := range cases {
		if got := compareUncommented([]byte(c.a), []byte(c.b)); got != c.want {
			t.Errorf("compare(%q, %q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

// The -copy flag's usage: with -required, or without -workflow, is exit 2;
// an unreadable file is exit 2; a step that runs make is exit 1; the
// repository passes.
func TestRunCopyMode(t *testing.T) {
	wf, cp := repo(repoWorkflow), repo(repoCopy)
	missing := filepath.Join(t.TempDir(), "nope.yml")
	withMake, withMakeCopy := writeBoth(t, strings.Replace(readRepo(t, repoWorkflow), "run: go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...", "run: make vulncheck", 1))
	cases := []struct {
		name    string
		args    []string
		code    int
		wantErr string
		wantOut string
	}{
		{"ok", []string{"-workflow", wf, "-copy", cp}, 0, "", "check-ci-steps: ok: " + wf + " equals " + cp + " and runs no make\n"},
		{"copy with required", []string{"-workflow", wf, "-copy", cp, "-required", "tools/ci/cisteps/required.yaml"}, 2, "-copy needs -workflow and excludes -required", ""},
		{"copy without workflow", []string{"-copy", cp}, 2, "-copy needs -workflow and excludes -required", ""},
		{"empty copy", []string{"-workflow", wf, "-copy", ""}, 2, "-copy needs -workflow", ""},
		{"copy unreadable", []string{"-workflow", wf, "-copy", missing}, 2, "check-ci-steps: open " + missing, ""},
		{"workflow unreadable", []string{"-workflow", missing, "-copy", cp}, 2, "check-ci-steps: open " + missing, ""},
		{"runs make", []string{"-workflow", withMake, "-copy", withMakeCopy}, 1, "edit " + withMake + " and " + withMakeCopy + " together, or not at all, and run no make in it", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(c.args, &stdout, &stderr); code != c.code {
				t.Fatalf("exit %d, want %d; stderr %q", code, c.code, stderr.String())
			}
			if stdout.String() != c.wantOut {
				t.Errorf("stdout %q, want %q", stdout.String(), c.wantOut)
			}
			if !strings.Contains(stderr.String(), c.wantErr) {
				t.Errorf("stderr %q lacks %q", stderr.String(), c.wantErr)
			}
			if c.wantErr == "" && stderr.Len() != 0 {
				t.Errorf("stderr %q, want empty", stderr.String())
			}
		})
	}
}
