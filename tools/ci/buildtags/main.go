// Command buildtags compiles every build-tag combination the repository's Go
// files use (TICK-320). go vet ./... and go test ./... build only untagged
// files, so a file behind //go:build identitystack && acceptance could stop
// compiling on main and nothing went red (TICK-286 finding 4: a duplicate
// onboardPostRaw).
//
// It finds every module under dir (each go.mod outside testdata, vendor and
// hidden directories) and reads every //go:build line in it. For each file it
// takes every minimal set of custom tags that builds it: one set for
// identitystack && acceptance, one per side for a || b. It then runs go vet
// -tags <set> ./... in the module once per distinct set. A new tag or module
// is covered without editing ci.yml or the Makefile. GOOS, GOARCH, go1.N and
// compiler tags are not custom: they take their linux/amd64 values, as on the
// CI runner. A file no tag set builds (//go:build ignore) is skipped; a file
// with more than maxTags custom tags is an error.
//
// Usage: buildtags [-dir dir] [-list]; -list prints the sets without vetting.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"go/build/constraint"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// maxTags bounds the subsets tried per file (2^maxTags).
const maxTags = 16

func main() {
	dir := flag.String("dir", ".", "repository root; every module under it is checked")
	list := flag.Bool("list", false, "print the tag sets and their files, do not vet")
	flag.Parse()
	code, err := run(*dir, *list, os.Stdout, goVet)
	if err != nil {
		fmt.Fprintln(os.Stderr, "buildtags:", err)
		os.Exit(2)
	}
	os.Exit(code)
}

// vetFunc runs go vet with tags in the module at dir, writing its output to w.
type vetFunc func(dir, tags string, w io.Writer) error

func goVet(dir, tags string, w io.Writer) error {
	cmd := exec.CommandContext(context.Background(), "go", "vet", "-tags", tags, "./...") //nolint:gosec // tags come from the module's own //go:build lines
	cmd.Dir = dir
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

// run vets each module's tag sets and returns 1 when any set fails. A failing
// set does not stop the others.
func run(dir string, list bool, w io.Writer, vet vetFunc) (int, error) {
	mods, err := modules(dir)
	if err != nil {
		return 0, err
	}
	var failed []string
	total := 0
	for _, mod := range mods {
		sets, err := discover(filepath.Join(dir, mod))
		if err != nil {
			return 0, err
		}
		keys := make([]string, 0, len(sets))
		for k := range sets {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			total++
			if list {
				fmt.Fprintf(w, "%s\t%s\t%s\n", mod, k, strings.Join(sets[k], " "))
				continue
			}
			fmt.Fprintf(w, "buildtags: %s: go vet -tags %s ./... (%d files)\n", mod, k, len(sets[k]))
			if err := vet(filepath.Join(dir, mod), k, w); err != nil {
				failed = append(failed, mod+":"+k)
				fmt.Fprintf(w, "buildtags: %s: -tags %s does not vet; it builds %s\n", mod, k, strings.Join(sets[k], " "))
			}
		}
	}
	if list {
		return 0, nil
	}
	if len(failed) > 0 {
		fmt.Fprintf(w, "buildtags: FAIL: %d of %d tag sets: %s\n", len(failed), total, strings.Join(failed, " "))
		return 1, nil
	}
	fmt.Fprintf(w, "buildtags: ok, %d tag sets in %d modules\n", total, len(mods))
	return 0, nil
}

// skipDir reports whether the walk leaves a directory out, as ./... does.
func skipDir(name string) bool {
	return name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// modules returns the directories under dir, relative and slash-separated
// ("." for dir itself), that hold a go.mod.
func modules(dir string) ([]string, error) {
	var mods []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != dir && skipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() || d.Name() != "go.mod" {
			return nil
		}
		rel, err := filepath.Rel(dir, filepath.Dir(path))
		if err != nil {
			return err
		}
		mods = append(mods, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(mods)
	return mods, err
}

// discover maps each tag set (sorted, comma-joined) to the files of the module
// at dir, relative to it, that the set is a minimal set for. Files that build
// untagged are left out: make vet covers them.
func discover(dir string) (map[string][]string, error) {
	sets := map[string][]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == dir {
				return nil
			}
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		expr, err := buildLine(path)
		if err != nil || expr == nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		minimal, err := minimalSets(expr)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.ToSlash(rel), err)
		}
		for _, tags := range minimal {
			if len(tags) == 0 {
				continue
			}
			k := strings.Join(tags, ",")
			sets[k] = append(sets[k], filepath.ToSlash(rel))
		}
		return nil
	})
	return sets, err
}

// buildLine returns the file's //go:build expression, or nil without one. Like
// go/build it reads only the header, before the package clause.
func buildLine(path string) (constraint.Expr, error) {
	f, err := os.Open(path) //nolint:gosec // a file found by the walk
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if constraint.IsGoBuild(line) {
			expr, err := constraint.Parse(line)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return expr, nil
		}
		if strings.HasPrefix(line, "package ") {
			return nil, nil
		}
	}
	return nil, s.Err()
}

// minimalSets returns every set of custom tags that satisfies expr while no
// smaller subset of it does, each sorted; none when no set satisfies expr. An
// untagged build is the empty set, which then is the only minimal one.
func minimalSets(expr constraint.Expr) ([][]string, error) {
	var custom []string
	seen := map[string]bool{}
	expr.Eval(func(tag string) bool {
		if _, sys := systemTag(tag); !sys && !seen[tag] {
			seen[tag] = true
			custom = append(custom, tag)
		}
		return false
	})
	if len(custom) > maxTags {
		return nil, fmt.Errorf("%d custom build tags, more than the %d buildtags tries", len(custom), maxTags)
	}
	sort.Strings(custom)
	masks := make([]int, 1<<len(custom))
	for m := range masks {
		masks[m] = m
	}
	// By size, so a satisfying mask is minimal unless it contains an earlier one.
	sort.SliceStable(masks, func(i, j int) bool { return popcount(masks[i]) < popcount(masks[j]) })
	var found []int
	var sets [][]string
next:
	for _, mask := range masks {
		for _, f := range found {
			if mask&f == f {
				continue next
			}
		}
		if !expr.Eval(func(tag string) bool {
			if v, sys := systemTag(tag); sys {
				return v
			}
			i := sort.SearchStrings(custom, tag)
			return mask&(1<<i) != 0
		}) {
			continue
		}
		found = append(found, mask)
		var tags []string
		for i, t := range custom {
			if mask&(1<<i) != 0 {
				tags = append(tags, t)
			}
		}
		sets = append(sets, tags)
	}
	return sets, nil
}

func popcount(m int) int {
	n := 0
	for ; m != 0; m &= m - 1 {
		n++
	}
	return n
}

// systemTag reports whether tag is set by the toolchain rather than -tags, and
// its value on the linux/amd64 CI runner.
func systemTag(tag string) (value, ok bool) {
	if strings.HasPrefix(tag, "go1.") {
		return true, true
	}
	switch tag {
	case "linux", "amd64", "unix", "gc", "cgo":
		return true, true
	case "ignore", "gccgo",
		"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js",
		"netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos",
		"386", "arm", "arm64", "loong64", "mips", "mipsle", "mips64", "mips64le",
		"ppc64", "ppc64le", "riscv64", "s390x", "wasm":
		return false, true
	}
	return false, false
}
