// Command buildtags compiles every build-tag combination the module's Go files
// use (TICK-320). go vet ./... and go test ./... build only untagged files, so
// a file behind //go:build identitystack && acceptance could stop compiling on
// main and nothing went red (TICK-286 finding 4: a duplicate onboardPostRaw).
//
// It reads every //go:build line in the module (testdata, vendor and nested
// modules are skipped, as ./... skips them), finds for each file the smallest
// set of custom tags that builds it, and runs go vet -tags <set> ./... once per
// distinct set. A new tag is covered without editing ci.yml or the Makefile.
// GOOS, GOARCH, go1.N and compiler tags are not custom: they take their
// linux/amd64 values, as on the CI runner. A file no tag set builds (//go:build
// ignore) is skipped.
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

func main() {
	dir := flag.String("dir", ".", "module directory")
	list := flag.Bool("list", false, "print the tag sets and their files, do not vet")
	flag.Parse()
	code, err := run(*dir, *list, os.Stdout, goVet)
	if err != nil {
		fmt.Fprintln(os.Stderr, "buildtags:", err)
		os.Exit(2)
	}
	os.Exit(code)
}

// vetFunc runs go vet with tags in dir, writing its output to w.
type vetFunc func(dir, tags string, w io.Writer) error

func goVet(dir, tags string, w io.Writer) error {
	cmd := exec.CommandContext(context.Background(), "go", "vet", "-tags", tags, "./...") //nolint:gosec // tags come from the module's own //go:build lines
	cmd.Dir = dir
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

// run vets each tag set and returns 1 when any set fails.
func run(dir string, list bool, w io.Writer, vet vetFunc) (int, error) {
	sets, err := discover(dir)
	if err != nil {
		return 0, err
	}
	keys := make([]string, 0, len(sets))
	for k := range sets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if list {
		for _, k := range keys {
			fmt.Fprintf(w, "%s\t%s\n", k, strings.Join(sets[k], " "))
		}
		return 0, nil
	}
	var failed []string
	for _, k := range keys {
		fmt.Fprintf(w, "buildtags: go vet -tags %s ./... (%d files)\n", k, len(sets[k]))
		if err := vet(dir, k, w); err != nil {
			failed = append(failed, k)
			fmt.Fprintf(w, "buildtags: -tags %s does not vet; it builds %s\n", k, strings.Join(sets[k], " "))
		}
	}
	if len(failed) > 0 {
		fmt.Fprintf(w, "buildtags: FAIL: %d of %d tag sets: %s\n", len(failed), len(keys), strings.Join(failed, " "))
		return 1, nil
	}
	fmt.Fprintf(w, "buildtags: ok, %d tag sets\n", len(keys))
	return 0, nil
}

// discover maps each tag set (sorted, comma-joined) to the files, relative to
// dir, it is the smallest set for. Files that build untagged are left out:
// make vet covers them.
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
			n := d.Name()
			if n == "testdata" || n == "vendor" || strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") {
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
		tags, ok := smallestSet(expr)
		if !ok || len(tags) == 0 {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		k := strings.Join(tags, ",")
		sets[k] = append(sets[k], filepath.ToSlash(rel))
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

// smallestSet returns the fewest custom tags that satisfy expr, sorted; ok is
// false when no set does.
func smallestSet(expr constraint.Expr) ([]string, bool) {
	var custom []string
	seen := map[string]bool{}
	expr.Eval(func(tag string) bool {
		if _, sys := systemTag(tag); !sys && !seen[tag] {
			seen[tag] = true
			custom = append(custom, tag)
		}
		return false
	})
	sort.Strings(custom)
	if len(custom) > 16 {
		custom = custom[:16]
	}
	best := -1
	for mask := 0; mask < 1<<len(custom); mask++ {
		if best >= 0 && popcount(mask) >= popcount(best) {
			continue
		}
		on := map[string]bool{}
		for i, t := range custom {
			if mask&(1<<i) != 0 {
				on[t] = true
			}
		}
		if expr.Eval(func(tag string) bool {
			if v, sys := systemTag(tag); sys {
				return v
			}
			return on[tag]
		}) {
			best = mask
		}
	}
	if best < 0 {
		return nil, false
	}
	var tags []string
	for i, t := range custom {
		if best&(1<<i) != 0 {
			tags = append(tags, t)
		}
	}
	return tags, true
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
