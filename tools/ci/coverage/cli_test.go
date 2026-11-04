package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runSplit(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestRunUsage(t *testing.T) {
	code, _, stderr := runSplit()
	if code != 2 || !strings.Contains(stderr, "usage: coverage merge|check") {
		t.Errorf("no command: exit %d, stderr %q", code, stderr)
	}
	code, _, stderr = runSplit("bogus")
	if code != 2 || !strings.Contains(stderr, `coverage: unknown command "bogus" (want merge or check)`) {
		t.Errorf("unknown command: exit %d, stderr %q", code, stderr)
	}
}

func TestMerge(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "mode: count\n"+mod+"/b/f.go:1.1,1.10 1 2\n"+mod+"/a/f.go:2.1,2.10 1 0\n")
	b := writeFile(t, dir, "b.txt", "mode: count\n"+mod+"/a/f.go:2.1,2.10 1 3\n")
	out := filepath.Join(dir, "out.txt")
	code, stdout, stderr := runSplit("merge", "-o", out, a, b)
	if code != 0 || stdout != "coverage merge: 2 profiles, 2 blocks -> "+out+"\n" || stderr != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	// Sorted by file, then position; counts summed in count mode.
	want := "mode: count\n" + mod + "/a/f.go:2.1,2.10 1 3\n" + mod + "/b/f.go:1.1,1.10 1 2\n"
	if got, _ := os.ReadFile(out); string(got) != want {
		t.Errorf("merged:\n%s\nwant:\n%s", got, want)
	}

	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"bad flag", []string{"merge", "-bogus"}, "flag provided but not defined: -bogus"},
		{"no profiles", []string{"merge", "-o", out}, "coverage merge: want -o and at least one profile"},
		{"missing profile", []string{"merge", "-o", out, filepath.Join(dir, "nope.txt")}, "coverage merge: open "},
		{"unwritable output", []string{"merge", "-o", filepath.Join(dir, "no", "such", "dir", "o.txt"), a}, "coverage merge: open "},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := runSplit(c.args...)
			if code != 2 || !strings.Contains(stderr, c.want) {
				t.Errorf("exit %d, stderr %q; want 2 and %q", code, stderr, c.want)
			}
		})
	}
}

// The gate fails on the input it exists to reject: exit 1, the FAIL line on
// stderr, the total and gap on stdout.
func TestCheckFailsBelowMinimum(t *testing.T) {
	dir := t.TempDir()
	p := writeProfile(t, dir, "p.txt", "atomic", "a", 10, 5)
	code, stdout, stderr := runSplit("check", "-module", mod, "-min", "80", p)
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if want := "coverage: FAIL: total below the minimum; the worst-covered packages are listed above\n"; stderr != want {
		t.Errorf("stderr %q, want %q", stderr, want)
	}
	if want := "coverage: total 50.0% (5/10 statements), minimum 80%, 3 more statements to cover\n"; !strings.Contains(stdout, want) {
		t.Errorf("stdout lacks %q:\n%s", want, stdout)
	}
}

func TestCheckErrorsAreExit2(t *testing.T) {
	dir := t.TempDir()
	good := writeProfile(t, dir, "good.txt", "set", "a", 2, 2)
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"bad flag", []string{"-bogus"}, "flag provided but not defined: -bogus"},
		{"no profiles", nil, "coverage check: want at least one profile"},
		{"unwritable -o", []string{"-o", filepath.Join(dir, "no", "such", "o.txt"), good}, "coverage check: open "},
		{"missing exclusions file", []string{"-exclude", filepath.Join(dir, "nope-ex.txt"), good}, "coverage check: open "},
		{"two patterns on a line", []string{"-exclude", writeFile(t, dir, "ex.txt", "a/f.go b/f.go # two\n"), good}, `ex.txt:1: one pattern per line, got "a/f.go b/f.go"`},
		{"no reason", []string{"-exclude", writeFile(t, dir, "ex1.txt", "\na/f.go\n"), good}, `ex1.txt:2: "a/f.go" has no reason`},
		{"stale exclusion", []string{"-exclude", writeFile(t, dir, "ex2.txt", "a/f.go # gen\nb/... # gone\n"), good}, "exclusions that match no file in the profiles (remove them): line 2: b/..."},
		{"two fields", []string{writeFile(t, dir, "two.txt", "mode: set\n"+mod+"/a/f.go:1.1,1.2 1\n")}, `two.txt:2: want "file:pos stmts count"`},
		{"no position", []string{writeFile(t, dir, "nopos.txt", "mode: set\nnoposition 1 1\n")}, `nopos.txt:2: no file:position in "noposition 1 1"`},
		{"bad stmts", []string{writeFile(t, dir, "badn.txt", "mode: set\n"+mod+"/a/f.go:1.1,1.2 x 1\n")}, "badn.txt:2: bad counts in"},
		{"negative count", []string{writeFile(t, dir, "neg.txt", "mode: set\n"+mod+"/a/f.go:1.1,1.2 1 -1\n")}, "neg.txt:2: bad counts in"},
		{"mode mismatch", []string{good, writeProfile(t, dir, "atomic.txt", "atomic", "a", 1, 1)}, "atomic.txt:1: mode atomic, earlier profiles are set"},
		{"different sources", []string{good, writeFile(t, dir, "other.txt", "mode: set\n"+mod+"/a/f.go:1.1,1.10 7 1\n")}, "has 7 statements here and 1 in an earlier profile"},
		{"no mode line", []string{writeFile(t, dir, "nomode.txt", mod+"/a/f.go:1.1,1.10 1 1\n")}, "nomode.txt:1: no mode line before the first block"},
		{"empty profile", []string{writeFile(t, dir, "empty.txt", "\n\n")}, "empty.txt: empty profile (no mode line)"},
		{"line too long", []string{writeFile(t, dir, "long.txt", "mode: set\n"+strings.Repeat("x", 1024*1024+1)+"\n")}, "long.txt: bufio.Scanner: token too long"},
		{"no statements", []string{writeFile(t, dir, "nostmts.txt", "mode: set\n")}, "coverage check: the profiles hold no statements"},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := runSplit(append([]string{"check", "-module", mod}, c.args...)...)
			if code != 2 || !strings.Contains(stderr, c.want) {
				t.Errorf("exit %d, stderr %q; want 2 and %q", code, stderr, c.want)
			}
		})
	}
}

// Without -module the module path comes from ./go.mod; with no go.mod that is
// an error.
func TestCheckModuleFromGoMod(t *testing.T) {
	dir := t.TempDir()
	p := writeProfile(t, dir, "p.txt", "set", "pkg/x", 4, 4)
	t.Chdir(dir)
	code, _, stderr := runSplit("check", "-all", p)
	if code != 2 || !strings.Contains(stderr, "coverage check: open go.mod") {
		t.Errorf("no go.mod: exit %d, stderr %q", code, stderr)
	}
	writeFile(t, dir, "go.mod", "module \""+mod+"\"\n")
	code, stdout, stderr := runSplit("check", "-all", p)
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	// The module prefix is stripped from the package name.
	if !strings.Contains(stdout, "\npkg/x ") || strings.Contains(stdout, mod+"/pkg/x") {
		t.Errorf("package not named pkg/x:\n%s", stdout)
	}
}

// -all lists every package even when the total passes; -worst limits the list
// when it fails. Packages outside the module keep their full path, a package
// of zero statements counts as 100%, and at equal cover the most missed
// statements sort first, then the name.
func TestCheckPackageTable(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "p.txt", "mode: set\n\n"+
		mod+"/b/f.go:1.1,1.2 1 1\n"+
		mod+"/b/f.go:2.1,2.2 1 0\n"+
		mod+"/a/f.go:1.1,1.2 1 1\n"+
		mod+"/a/f.go:2.1,2.2 1 0\n"+
		mod+"/c/f.go:1.1,1.2 2 1\n"+
		mod+"/c/f.go:2.1,2.2 2 0\n"+
		mod+"/empty/f.go:1.1,1.2 0 0\n"+
		"other.org/lib/f.go:1.1,1.2 4 1\n")
	code, stdout, _ := runSplit("check", "-module", mod, "-min", "0", "-all", p)
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n")[1:] {
		if f := strings.Fields(line); len(f) > 1 && f[0] != "coverage:" {
			names = append(names, f[0]+"="+f[1])
		}
	}
	if got, want := strings.Join(names, " "), "c=50.0% a=50.0% b=50.0% empty=100.0% other.org/lib=100.0%"; got != want {
		t.Errorf("table %s, want %s\n%s", got, want, stdout)
	}
	if !strings.Contains(stdout, "coverage: total 66.7% (8/12 statements), minimum 0%\ncoverage: ok\n") {
		t.Errorf("no ok line:\n%s", stdout)
	}

	code, stdout, _ = runSplit("check", "-module", mod, "-min", "100", "-worst", "1", p)
	if code != 1 {
		t.Errorf("-worst 1: exit %d, want 1", code)
	}
	if strings.Contains(stdout, "\na ") || !strings.Contains(stdout, "\nc ") {
		t.Errorf("-worst 1: want only the worst package, c:\n%s", stdout)
	}

	// A passing total without -all prints no table.
	code, stdout, _ = runSplit("check", "-module", mod, "-min", "0", p)
	if code != 0 || strings.Contains(stdout, "covered/all") {
		t.Errorf("passing without -all: exit %d, want no table:\n%s", code, stdout)
	}
}

func TestModulePathMissingFile(t *testing.T) {
	if _, err := modulePath(filepath.Join(t.TempDir(), "go.mod")); err == nil {
		t.Error("missing go.mod: want an error")
	}
}
