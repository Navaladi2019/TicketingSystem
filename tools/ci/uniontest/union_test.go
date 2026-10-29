// Package uniontest proves that the shared tables merge without conflict when
// two branches each append a row (TICK-295, merge-safety D6). It builds a temp
// git repository that copies the repo's real .gitattributes, so a change to
// the merge attributes that breaks the union merge fails here.
package uniontest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitFlags pins the settings that could change a merge result, so a user's
// global merge.conflictstyle or custom merge driver cannot affect the tests.
func gitFlags() []string {
	return []string{
		"-c", "user.name=t", "-c", "user.email=t@example.invalid",
		"-c", "core.autocrlf=false", "-c", "commit.gpgsign=false",
		"-c", "merge.conflictstyle=merge", "-c", "merge.renormalize=false",
		"-c", "core.attributesFile=" + os.DevNull,
	}
}

func gitCmd(t *testing.T, dir string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append(gitFlags(), args...)...)
	cmd.Dir = dir
	return cmd
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := gitCmd(t, dir, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func put(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

type appendCase struct{ header, rowA, rowB string }

func shared() map[string]appendCase {
	return map[string]appendCase{
		"docs/delivery/todo.md": {
			"# TODO tracker\n\nID rule.\n\n| ID | Status |\n| --- | --- |\n| TODO-001 | Open |\n",
			"| TODO-S5-01 | Open |\n", "| TODO-S7-01 | Open |\n"},
		"docs/security/accepted-risks.md": {
			"# Accepted risks\n\nID rule.\n\n| ID | Sev |\n| --- | --- |\n| AR-001 | Low |\n",
			"| AR-S5-01 | Low |\n", "| AR-S7-01 | Low |\n"},
		"docs/pins.md": {"# Pins\n\n| Name | Version |\n| --- | --- |\n| `a` | 1 |\n",
			"| `b` | 2 |\n", "| `c` | 3 |\n"},
		".gitignore": {"/bin/\n", "/s5.out\n", "/s7.out\n"},
	}
}

func TestTwoBranchesAppendWithoutConflict(t *testing.T) {
	requireGit(t)
	attrs, err := os.ReadFile(filepath.Join("..", "..", "..", ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	for path, f := range shared() {
		t.Run(path, func(t *testing.T) {
			dir := t.TempDir()
			git(t, dir, "init", "-q", "-b", "main")
			put(t, dir, ".gitattributes", string(attrs))
			put(t, dir, path, f.header)
			git(t, dir, "add", "-A")
			git(t, dir, "commit", "-q", "-m", "base")

			for _, b := range []struct{ name, row string }{{"s5", f.rowA}, {"s7", f.rowB}} {
				git(t, dir, "checkout", "-q", "-b", b.name, "main")
				old, _ := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
				put(t, dir, path, string(old)+b.row)
				git(t, dir, "commit", "-q", "-am", b.name)
			}
			git(t, dir, "checkout", "-q", "s5")
			git(t, dir, "merge", "--no-edit", "s7") // fails the test on a conflict

			got, _ := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
			for _, row := range []string{f.rowA, f.rowB} {
				if !strings.Contains(string(got), row) {
					t.Errorf("merged file lost %q:\n%s", row, got)
				}
			}
		})
	}
}

// Without the union attribute the same merge conflicts, so the test above is
// really exercising merge=union and not git's own line merging.
func TestAppendsConflictWithoutUnion(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	put(t, dir, "t.md", "| ID |\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "base")
	for _, b := range []string{"a", "b"} {
		git(t, dir, "checkout", "-q", "-b", b, "main")
		put(t, dir, "t.md", "| ID |\n| "+b+" |\n")
		git(t, dir, "commit", "-q", "-am", b)
	}
	git(t, dir, "checkout", "-q", "a")
	if err := gitCmd(t, dir, "merge", "--no-edit", "b").Run(); err == nil {
		t.Fatal("expected a conflict without merge=union")
	}
}

// The attributes must keep the files text: a -text (binary) file would make
// merge=union fall back to a conflict.
func TestAttributesMarkSharedFilesTextUnion(t *testing.T) {
	requireGit(t)
	root := filepath.Join("..", "..", "..")
	for p := range shared() {
		out := git(t, root, "check-attr", "merge", "text", "eol", "--", p)
		if !strings.Contains(out, "merge: union") || strings.Contains(out, "text: unset") {
			t.Errorf("%s attributes:\n%s", p, out)
		}
	}
}

// merge=union only works for appends. When two branches edit the same existing
// row it keeps both versions, with no conflict and two rows for one id; the
// duplicate-id lint (tools/dupids) is what catches that (TICK-312).
func TestEditSameRowCollidesAndLintCatchesIt(t *testing.T) {
	requireGit(t)
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	attrs, err := os.ReadFile(filepath.Join(root, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	type table struct{ head, id string }
	tables := map[string]table{
		"docs/delivery/todo.md":           {"| ID | Status |\n| --- | --- |\n", "TODO-001"},
		"docs/security/accepted-risks.md": {"| ID | Sev |\n| --- | --- |\n", "AR-001"},
	}
	for path, tb := range tables {
		t.Run(path, func(t *testing.T) {
			dir := t.TempDir()
			git(t, dir, "init", "-q", "-b", "main")
			put(t, dir, ".gitattributes", string(attrs))
			for p, other := range tables {
				put(t, dir, p, other.head+"| "+other.id+" | base |\n")
			}
			git(t, dir, "add", "-A")
			git(t, dir, "commit", "-q", "-m", "base")

			for _, b := range []string{"a", "b"} {
				git(t, dir, "checkout", "-q", "-b", b, "main")
				// Both branches rewrite the same row, each in its own way.
				put(t, dir, path, tb.head+"| "+tb.id+" | edited on "+b+" |\n")
				git(t, dir, "commit", "-q", "-am", b)
			}
			git(t, dir, "checkout", "-q", "a")
			git(t, dir, "merge", "--no-edit", "b") // union merge: no conflict

			merged, _ := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
			if n := strings.Count(string(merged), "| "+tb.id+" |"); n != 2 {
				t.Fatalf("want the id twice after the union merge, got %d:\n%s", n, merged)
			}

			cmd := exec.CommandContext(t.Context(), "go", "run", "./tools/dupids", "-dir", dir)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "id "+tb.id+" appears 2 times") {
				t.Fatalf("dupids should fail on %s: err=%v\n%s", tb.id, err, out)
			}
		})
	}
}
