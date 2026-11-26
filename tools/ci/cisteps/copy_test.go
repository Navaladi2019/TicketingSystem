package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	repoWorkflow = ".github/workflows/govulncheck.yml"
	repoCopy     = "tools/ci/cisteps/govulncheck.copy.yml"
)

// pinned is a make runner that prints exactly the pinned recipes.
func pinned(target string) ([]byte, error) {
	for _, r := range pinnedRecipes() {
		if r.target == target {
			return []byte(r.recipe + "\n"), nil
		}
	}
	return nil, errors.New("no such target")
}

// recipesWith is a make runner that prints the pinned recipes, target's
// replaced by recipe.
func recipesWith(target, recipe string) makeRunner {
	return func(t string) ([]byte, error) {
		if t == target {
			return []byte(recipe + "\n"), nil
		}
		return pinned(t)
	}
}

// AC 1 (TICK-1149): the committed workflow, its copy and the real make
// recipes pass the -copy check. TestRepository covers ci.yml only.
func TestRepositoryGovulncheckCopy(t *testing.T) {
	problems, err := checkCopy(repo(repoWorkflow), repo(repoCopy), makeIn(repo("")))
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

// atMakeLine replaces the `run: make <target>` line by f(indent of run:).
func atMakeLine(target string, f func(ind string) []string) edit {
	return func(t *testing.T, wf string) string {
		t.Helper()
		lines := strings.Split(wf, "\n")
		i := lineIndex(t, lines, `^\s+run: make `+regexp.QuoteMeta(target)+`\s*$`)
		ind := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " "))]
		return strings.Join(append(lines[:i], append(f(ind), lines[i+1:]...)...), "\n")
	}
}

// v3Edits are contract test V-3's edits of govulncheck.yml alone (TICK-1149
// AC 2 a-i). Each must fail check-ci-steps. The gate cases (no always(), name
// without the base test) come with the gate job, TICK-1150.
type v3Edit struct {
	name string
	edit edit
}

func v3Edits() []v3Edit {
	return []v3Edit{
		{"a step if", atMakeLine("vulncheck", func(ind string) []string {
			return []string{ind + "if: github.actor == 'nobody'", ind + "run: make vulncheck"}
		})},
		{"b job if", insertAt(`^jobs:\s*$`, true, "  vulncheck-root-gone:", "    if: github.event_name == 'never'", "    runs-on: ubuntu-latest", "    steps:", "      - run: make vulncheck")},
		{"b job if on the scanning job", func(t *testing.T, wf string) string {
			lines := strings.Split(wf, "\n")
			j := lineIndex(t, lines, `^  vulncheck-root:\s*$`)
			return strings.Join(append(lines[:j+1], append([]string{"    if: github.event_name == 'never'"}, lines[j+1:]...)...), "\n")
		}},
		{"c exit 0 before the make line", atMakeLine("vulncheck", func(ind string) []string {
			return []string{ind + "run: |", ind + "  exit 0", ind + "  make vulncheck"}
		})},
		{"d step shell", atMakeLine("vulncheck-temporal-server", func(ind string) []string {
			return []string{ind + "shell: 'true {0}'", ind + "run: make vulncheck-temporal-server"}
		})},
		{"e make line in a printed heredoc", atMakeLine("vulncheck", func(ind string) []string {
			return []string{ind + "run: |", ind + "  cat <<'EOF'", ind + "  make vulncheck", ind + "  EOF"}
		})},
		{"f another job named vulncheck", insertAt(`^jobs:\s*$`, true, "  vulncheck-other:", "    name: vulncheck", "    runs-on: ubuntu-latest", "    steps:", "      - run: 'true'", "")},
		{"g top-level defaults.run.shell", insertAt(`^jobs:\s*$`, false, "defaults:", "  run:", "    shell: 'true {0}'", "")},
		{"h top-level env", insertAt(`^jobs:\s*$`, false, "env:", "  GOFLAGS: -mod=mod", "")},
		{"i paths entry added", insertAt(`^    paths:\s*$`, true, "      - '**/*.md'")},
		{"i paths entry changed", func(t *testing.T, wf string) string {
			lines := strings.Split(wf, "\n")
			i := lineIndex(t, lines, `^      - go\.mod\s*$`)
			lines[i] = strings.Replace(lines[i], "go.mod", "go.mod.unused", 1)
			return strings.Join(lines, "\n")
		}},
		{"make line commented out", atMakeLine("vulncheck", func(ind string) []string {
			return []string{ind + "run: 'true'", ind + "# run: make vulncheck"}
		})},
		{"trailing comment changed", atMakeLine("vulncheck", func(ind string) []string {
			return []string{ind + "run: make vulncheck # || true"}
		})},
		{"indentation changed", atMakeLine("vulncheck", func(ind string) []string {
			return []string{ind + " run: make vulncheck"}
		})},
		{"blank line added", insertAt(`^jobs:\s*$`, true, "")},
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
			if code := runWith([]string{"-workflow", p, "-copy", cp}, &stdout, &stderr, pinned); code != 1 {
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

// V-3 (AC 2 j): `|| true` in either recipe fails, with the recipe named.
func TestCopyCatchesRecipeEdits(t *testing.T) {
	for _, r := range pinnedRecipes() {
		for name, recipe := range map[string]string{
			"or true":         r.recipe + " || true",
			"GOWORK dropped":  strings.Replace(r.recipe, "GOWORK=off ", "", 1),
			"a second line":   r.recipe + "\necho scanned",
			"a make warning":  "make: warning: something\n" + r.recipe,
			"nothing printed": "",
		} {
			t.Run(r.target+"/"+name, func(t *testing.T) {
				problems, err := checkCopy(repo(repoWorkflow), repo(repoCopy), recipesWith(r.target, recipe))
				if err != nil {
					t.Fatal(err)
				}
				want := "make -n " + r.target + " prints "
				if len(problems) != 1 || !strings.HasPrefix(problems[0], want) {
					t.Fatalf("want one problem starting %q, got %q", want, problems)
				}
			})
		}
	}
}

// V-3 (AC 2 j) through real make: a temporary Makefile whose recipe ends in
// `|| true` fails.
func TestCopyCatchesRecipeEditsWithMake(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("no make on PATH")
	}
	for _, r := range pinnedRecipes() {
		t.Run(r.target, func(t *testing.T) {
			dir := t.TempDir()
			var mk strings.Builder
			for _, o := range pinnedRecipes() {
				line := o.recipe
				if o.target == r.target {
					line += " || true"
				}
				mk.WriteString(o.target + ":\n\t" + line + "\n")
			}
			if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(mk.String()), 0o600); err != nil {
				t.Fatal(err)
			}
			problems, err := checkCopy(repo(repoWorkflow), repo(repoCopy), makeIn(dir))
			if err != nil {
				t.Fatal(err)
			}
			want := "make -n " + r.target + ` prints "` + r.recipe + ` || true"`
			if len(problems) != 1 || !strings.HasPrefix(problems[0], want) {
				t.Fatalf("want one problem starting %q, got %q", want, problems)
			}
		})
	}
}

// V-3 (AC 2 k) documents RS3: the same edit of the workflow and the copy
// passes. The check stops an edit of one file, not a deliberate edit of both.
func TestCopyPassesTheSameEditOfBoth(t *testing.T) {
	wf, cp := readRepo(t, repoWorkflow), readRepo(t, repoCopy)
	for _, c := range v3Edits() {
		t.Run(c.name, func(t *testing.T) {
			problems, err := checkCopy(writeTemp(t, c.edit(t, wf)), writeTemp(t, c.edit(t, cp)), pinned)
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
		"indented comment added": atMakeLine("vulncheck", func(ind string) []string { return []string{ind + "# the root module", ind + "run: make vulncheck"} }),
		"tab-indented comment":   insertAt(`^jobs:\s*$`, true, "\t# note"),
	} {
		t.Run(name, func(t *testing.T) {
			edited := e(t, wf)
			if edited == wf {
				t.Fatal("the edit changed nothing")
			}
			problems, err := checkCopy(writeTemp(t, edited), repo(repoCopy), pinned)
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
	}
	for _, c := range cases {
		if got := compareUncommented([]byte(c.a), []byte(c.b)); got != c.want {
			t.Errorf("compare(%q, %q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

// The -copy flag's usage: with -required, or without -workflow, is exit 2;
// an unreadable file or a failing make is exit 2; the repository passes.
func TestRunCopyMode(t *testing.T) {
	wf, cp := repo(repoWorkflow), repo(repoCopy)
	missing := filepath.Join(t.TempDir(), "nope.yml")
	cases := []struct {
		name    string
		args    []string
		mk      makeRunner
		code    int
		wantErr string
		wantOut string
	}{
		{"ok", []string{"-workflow", wf, "-copy", cp}, pinned, 0, "", "check-ci-steps: ok: " + wf + " equals " + cp + " and the vulncheck recipes are pinned\n"},
		{"copy with required", []string{"-workflow", wf, "-copy", cp, "-required", "tools/ci/cisteps/required.yaml"}, pinned, 2, "-copy needs -workflow and excludes -required", ""},
		{"copy without workflow", []string{"-copy", cp}, pinned, 2, "-copy needs -workflow and excludes -required", ""},
		{"empty copy", []string{"-workflow", wf, "-copy", ""}, pinned, 2, "-copy needs -workflow", ""},
		{"copy unreadable", []string{"-workflow", wf, "-copy", missing}, pinned, 2, "check-ci-steps: open " + missing, ""},
		{"workflow unreadable", []string{"-workflow", missing, "-copy", cp}, pinned, 2, "check-ci-steps: open " + missing, ""},
		{"make fails", []string{"-workflow", wf, "-copy", cp}, func(string) ([]byte, error) {
			return []byte("make: *** No rule to make target 'vulncheck'.  Stop."), errors.New("exit status 2")
		}, 2, "make -n vulncheck: exit status 2: make: *** No rule", ""},
		{"recipe edited", []string{"-workflow", wf, "-copy", cp}, recipesWith("vulncheck", "GOWORK=off govulncheck ./... || true"), 1, "edit " + wf + " and " + cp + " together", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runWith(c.args, &stdout, &stderr, c.mk); code != c.code {
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
