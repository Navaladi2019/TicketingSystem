package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree writes files (slash paths relative to a new temp dir) and returns
// the dir.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func noVet(t *testing.T) vetFunc {
	return func(dir, tags string, _ io.Writer) error {
		t.Errorf("vet called (%s, %s) where it must not be", dir, tags)
		return nil
	}
}

func runCLI(args []string, vet vetFunc) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := cli(args, &stdout, &stderr, vet)
	return code, stdout.String(), stderr.String()
}

// -list prints every module's sets and their files, and vets nothing.
func TestCLIList(t *testing.T) {
	code, stdout, stderr := runCLI([]string{"-dir", "testdata/ok", "-list"}, noVet(t))
	want := ".\taccept,stacktag\tstack/accept_test.go\n" +
		".\tstacktag\tstack/stack_test.go\n" +
		"nested/inner\tinnertag\tinner.go\n"
	if code != 0 || stdout != want || stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q; want 0 and %q", code, stdout, stderr, want)
	}
}

// The gate through the command line: a tag set that does not vet is exit 1
// with the FAIL line naming it; all vetting is exit 0 with the ok line.
func TestCLIVetResult(t *testing.T) {
	failAccept := func(_, tags string, w io.Writer) error {
		if tags == "accept,stacktag" {
			_, _ = io.WriteString(w, "stack/accept_test.go:5:6: postRaw redeclared\n")
			return errors.New("exit status 1")
		}
		return nil
	}
	code, stdout, _ := runCLI([]string{"-dir", "testdata/ok"}, failAccept)
	if code != 1 {
		t.Errorf("failing set: exit %d, want 1", code)
	}
	for _, want := range []string{
		"buildtags: .: -tags accept,stacktag does not vet; it builds stack/accept_test.go\n",
		"buildtags: FAIL: 1 of 3 tag sets: .:accept,stacktag\n",
		"postRaw redeclared",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("failing set: stdout lacks %q:\n%s", want, stdout)
		}
	}

	code, stdout, _ = runCLI([]string{"-dir", "testdata/ok"}, func(string, string, io.Writer) error { return nil })
	if code != 0 || !strings.HasSuffix(stdout, "buildtags: ok, 3 tag sets in 2 modules\n") {
		t.Errorf("all vet: exit %d, stdout:\n%s", code, stdout)
	}
	if !strings.Contains(stdout, "buildtags: nested/inner: go vet -tags innertag ./... (1 files)\n") {
		t.Errorf("all vet: no per-set line for nested/inner:\n%s", stdout)
	}
}

// Bad input is exit 2 with the reason on stderr, before anything is vetted.
func TestCLIBadInput(t *testing.T) {
	var many []string
	for i := 0; i <= maxTags; i++ {
		many = append(many, "t"+string(rune('a'+i)))
	}
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"missing dir", []string{"-dir", filepath.Join(t.TempDir(), "nope")}, "nope"},
		{"bad build line", []string{"-dir", writeTree(t, map[string]string{
			"go.mod": "module example.com/m\n",
			"a/a.go": "//go:build a &&\n\npackage a\n",
		})}, "a.go: "},
		{"too many tags", []string{"-dir", writeTree(t, map[string]string{
			"go.mod": "module example.com/m\n",
			"a/a.go": "//go:build " + strings.Join(many, " && ") + "\n\npackage a\n",
		})}, "buildtags: a/a.go: 17 custom build tags, more than the 16 buildtags tries"},
		{"unknown flag", []string{"-bogus"}, "flag provided but not defined: -bogus"},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := runCLI(c.args, noVet(t))
			if code != 2 || !strings.Contains(stderr, c.want) {
				t.Errorf("exit %d, stderr %q; want 2 and %q", code, stderr, c.want)
			}
		})
	}
}

func TestCLIHelpExitsZero(t *testing.T) {
	code, _, stderr := runCLI([]string{"-h"}, noVet(t))
	if code != 0 || !strings.Contains(stderr, "-list") {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
}

// Untagged files, files no tag set builds and non-Go files add no set; hidden,
// vendor and _ directories are left out like ./... leaves them out.
func TestDiscoverSkips(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"go.mod":            "module example.com/m\n",
		"plain.go":          "package m\n",
		"ignored.go":        "//go:build ignore\n\npackage m\n",
		"notes.txt":         "//go:build nottag\n",
		"vendor/v/v.go":     "//go:build vendortag\n\npackage v\n",
		".hidden/h.go":      "//go:build hiddentag\n\npackage h\n",
		"_skip/s.go":        "//go:build skiptag\n\npackage s\n",
		"pkg/tagged.go":     "// Copyright\n\n//go:build mytag\n\npackage pkg\n",
		"pkg/late.go":       "package pkg\n\n//go:build latetag\n",
		"sub/go.mod":        "module example.com/m/sub\n",
		"sub/inner_test.go": "//go:build subtag\n\npackage sub\n",
	})
	sets, err := discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 1 || strings.Join(sets["mytag"], " ") != "pkg/tagged.go" {
		t.Errorf("discover = %v, want only mytag: pkg/tagged.go", sets)
	}
	mods, err := modules(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(mods, " ") != ". sub" {
		t.Errorf("modules = %v, want [. sub]", mods)
	}
}

func TestDiscoverMissingDir(t *testing.T) {
	if _, err := discover(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("missing dir: want an error")
	}
}

func TestBuildLineErrors(t *testing.T) {
	if _, err := buildLine(filepath.Join(t.TempDir(), "nope.go")); err == nil {
		t.Error("missing file: want an error")
	}
	// A header line longer than the scanner's buffer is an error, not a file
	// read as untagged.
	long := writeTree(t, map[string]string{"long.go": "// " + strings.Repeat("x", bufio.MaxScanTokenSize) + "\n//go:build longtag\n\npackage m\n"})
	if _, err := buildLine(filepath.Join(long, "long.go")); !errors.Is(err, bufio.ErrTooLong) {
		t.Errorf("long header line: err %v, want bufio.ErrTooLong", err)
	}
	// No package clause and no build line: untagged, no error.
	empty := writeTree(t, map[string]string{"e.go": "// just a comment\n"})
	if expr, err := buildLine(filepath.Join(empty, "e.go")); expr != nil || err != nil {
		t.Errorf("comment-only file: %v, %v; want nil, nil", expr, err)
	}
}
