package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const mod = "example.com/m"

// writeProfile writes a profile of n one-statement blocks in pkg/f.go, the
// first covered of them hit once.
func writeProfile(t *testing.T, dir, name, mode string, pkg string, n, covered int) string {
	t.Helper()
	var sb strings.Builder
	fmt.Fprintf(&sb, "mode: %s\n", mode)
	for i := range n {
		hit := 0
		if i < covered {
			hit = 1
		}
		fmt.Fprintf(&sb, "%s/%s/f.go:%d.1,%d.10 1 %d\n", mod, pkg, i+1, i+1, hit)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(sb.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func check(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out bytes.Buffer
	code := run(append([]string{"check", "-module", mod}, args...), &out, &out)
	return code, out.String()
}

// The gate at its edge: 95.00% passes, 94.99% fails, and COVERAGE_MIN (here
// -min) moves the edge.
func TestCheckThreshold(t *testing.T) {
	dir := t.TempDir()
	at := writeProfile(t, dir, "at.txt", "atomic", "a", 10000, 9500)
	below := writeProfile(t, dir, "below.txt", "atomic", "a", 10000, 9499)

	if code, out := check(t, "-min", "95", at); code != 0 || !strings.Contains(out, "total 95.0%") {
		t.Errorf("95.00%%: exit %d, want 0\n%s", code, out)
	}
	code, out := check(t, "-min", "95", below)
	if code != 1 {
		t.Errorf("94.99%%: exit %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "1 more statements to cover") || !strings.Contains(out, "a ") {
		t.Errorf("94.99%%: want the gap and the package table\n%s", out)
	}
	if code, out := check(t, "-min", "94.99", below); code != 0 {
		t.Errorf("-min 94.99 at 94.99%%: exit %d, want 0\n%s", code, out)
	}
	if code, out := check(t, "-min", "95", "-report-only", below); code != 0 || !strings.Contains(out, "report-only") {
		t.Errorf("report-only below the minimum: exit %d, want 0 and a report-only line\n%s", code, out)
	}
}

// The integration profile counts: a block only the integration run covers is
// covered in the total, and repeated blocks are not double-counted.
func TestCheckMergesProfiles(t *testing.T) {
	dir := t.TempDir()
	unit := writeFile(t, dir, "unit.txt", "mode: atomic\n"+
		mod+"/a/f.go:1.1,1.10 2 3\n"+
		mod+"/a/f.go:2.1,2.10 2 0\n"+
		"mode: atomic\n"+ // a concatenated profile repeats the mode line
		mod+"/a/f.go:1.1,1.10 2 1\n")
	it := writeFile(t, dir, "it.txt", "mode: atomic\n"+
		mod+"/a/f.go:2.1,2.10 2 4\n")

	if code, out := check(t, "-min", "100", unit); code != 1 || !strings.Contains(out, "total 50.0% (2/4") {
		t.Errorf("unit alone: exit %d, want 1 at 2/4\n%s", code, out)
	}
	merged := filepath.Join(dir, "merged.txt")
	code, out := check(t, "-min", "100", "-o", merged, unit, it)
	if code != 0 || !strings.Contains(out, "total 100.0% (4/4") {
		t.Errorf("unit + integration: exit %d, want 0 at 4/4\n%s", code, out)
	}
	got, err := os.ReadFile(merged)
	if err != nil {
		t.Fatal(err)
	}
	want := "mode: atomic\n" + mod + "/a/f.go:1.1,1.10 2 4\n" + mod + "/a/f.go:2.1,2.10 2 4\n"
	if string(got) != want {
		t.Errorf("merged profile:\n%s\nwant:\n%s", got, want)
	}

	// merge alone writes the same profile.
	var buf bytes.Buffer
	merged2 := filepath.Join(dir, "merged2.txt")
	if code := run([]string{"merge", "-o", merged2, unit, it}, &buf, &buf); code != 0 {
		t.Fatalf("merge: exit %d\n%s", code, buf.String())
	}
	if got2, _ := os.ReadFile(merged2); string(got2) != want {
		t.Errorf("merge:\n%s\nwant:\n%s", got2, want)
	}
}

func TestCheckSetModeKeepsMax(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "mode: set\n"+mod+"/a/f.go:1.1,1.10 1 1\n")
	b := writeFile(t, dir, "b.txt", "mode: set\n"+mod+"/a/f.go:1.1,1.10 1 0\n")
	out := filepath.Join(dir, "o.txt")
	if code, msg := check(t, "-o", out, a, b); code != 0 {
		t.Fatalf("exit %d\n%s", code, msg)
	}
	if got, _ := os.ReadFile(out); !strings.HasSuffix(string(got), " 1 1\n") {
		t.Errorf("set mode merged to %q, want count 1", got)
	}
}

func TestCheckBadInput(t *testing.T) {
	dir := t.TempDir()
	atomic := writeProfile(t, dir, "atomic.txt", "atomic", "a", 2, 2)
	set := writeProfile(t, dir, "set.txt", "set", "a", 2, 2)
	otherStmts := writeFile(t, dir, "other.txt", "mode: atomic\n"+mod+"/a/f.go:1.1,1.10 7 1\n")
	for name, args := range map[string][]string{
		"missing profile":      {filepath.Join(dir, "nope.txt")},
		"mode mismatch":        {atomic, set},
		"different sources":    {atomic, otherStmts},
		"no mode line":         {writeFile(t, dir, "nomode.txt", mod+"/a/f.go:1.1,1.10 1 1\n")},
		"empty profile":        {writeFile(t, dir, "empty.txt", "")},
		"garbage line":         {writeFile(t, dir, "garbage.txt", "mode: set\nnot a block\n")},
		"no statements":        {writeFile(t, dir, "nostmts.txt", "mode: set\n")},
		"no profiles":          {},
		"no reason":            {"-exclude", writeFile(t, dir, "ex1.txt", "a/f.go\n"), atomic},
		"empty reason":         {"-exclude", writeFile(t, dir, "ex2.txt", "a/f.go  #  \n"), atomic},
		"stale exclusion":      {"-exclude", writeFile(t, dir, "ex3.txt", "a/f.go # gen\nb/... # gone\n"), atomic},
		"no exclusions listed": {"-exclude", writeFile(t, dir, "ex4.txt", "# only comments\n"), atomic},
		"bad pattern":          {"-exclude", writeFile(t, dir, "ex5.txt", "a/[.go # bad\n"), atomic},
	} {
		if code, out := check(t, args...); code != 2 {
			t.Errorf("%s: exit %d, want 2\n%s", name, code, out)
		}
	}
	var buf bytes.Buffer
	if code := run(nil, &buf, &buf); code != 2 {
		t.Errorf("no command: exit %d, want 2", code)
	}
	if code := run([]string{"bogus"}, &buf, &buf); code != 2 {
		t.Errorf("unknown command: exit %d, want 2", code)
	}
	if code := run([]string{"merge", atomic}, &buf, &buf); code != 2 {
		t.Errorf("merge without -o: exit %d, want 2", code)
	}
}

// Excluded files leave the total, and every pattern form matches what its
// comment says.
func TestCheckExclusions(t *testing.T) {
	dir := t.TempDir()
	prof := writeFile(t, dir, "p.txt", "mode: atomic\n"+
		mod+"/api/api.gen.go:1.1,1.10 50 0\n"+
		mod+"/internal/x/y.gen.go:1.1,1.10 50 0\n"+
		mod+"/cmd/tool/main.go:1.1,1.10 50 0\n"+
		mod+"/cmd/tool/flags.go:1.1,1.10 1 1\n"+
		mod+"/internal/helpertest/h.go:1.1,1.10 50 0\n"+
		mod+"/internal/helpertest/sub/h.go:1.1,1.10 50 0\n"+
		mod+"/internal/real/r.go:1.1,1.10 1 1\n")
	ex := writeFile(t, dir, "ex.txt", "# header comment\n\n"+
		"api/api.gen.go   # generated\n"+
		"**/*.gen.go      # generated\n"+
		"cmd/*/main.go    # wiring\n"+
		"internal/helpertest/...  # test-only helper\n"+
		"internal/helpertest/sub/... # overlaps the line above, still not stale\n")
	code, out := check(t, "-min", "100", "-exclude", ex, prof)
	if code != 0 || !strings.Contains(out, "total 100.0% (2/2") {
		t.Errorf("exit %d, want 0 with only cmd/tool/flags.go and internal/real counted\n%s", code, out)
	}
}

func TestByPackageOrder(t *testing.T) {
	p := &profile{mode: "set", blocks: map[block]counts{
		{mod + "/good/f.go", "1.1,1.2"}:  {10, 1},
		{mod + "/small/f.go", "1.1,1.2"}: {1, 0},
		{mod + "/big/f.go", "1.1,1.2"}:   {9, 0},
		{mod + "/big/f.go", "2.1,2.2"}:   {1, 1},
		{mod + "/half/f.go", "1.1,1.2"}:  {1, 0},
		{mod + "/half/f.go", "2.1,2.2"}:  {1, 1},
	}}
	list, total := p.byPackage(mod)
	var names []string
	for _, s := range list {
		names = append(names, s.name)
	}
	if got, want := strings.Join(names, ","), "small,big,half,good"; got != want {
		t.Errorf("order %s, want %s", got, want)
	}
	if total.covered != 12 || total.stmts != 23 {
		t.Errorf("total %d/%d, want 12/23", total.covered, total.stmts)
	}
}

func TestModulePath(t *testing.T) {
	dir := t.TempDir()
	if got, err := modulePath(writeFile(t, dir, "go.mod", "// c\nmodule example.com/x\n\ngo 1.26\n")); err != nil || got != "example.com/x" {
		t.Errorf("modulePath = %q, %v", got, err)
	}
	if _, err := modulePath(writeFile(t, dir, "bad.mod", "go 1.26\n")); err == nil {
		t.Error("no module line: want an error")
	}
}

// The repository's own exclusions file parses, and every line has a reason.
func TestRepoExclusionsFile(t *testing.T) {
	list, err := readExclusions(filepath.Join("..", "coverage-exclude.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("no exclusions")
	}
}
