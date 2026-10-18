// Command checkpins enforces docs/pins.md (T05, TICK-94): every direct
// require in every go.mod has a row naming its version, every image in the
// compose files and Dockerfiles has a row naming its tag, and no module from
// the Forbidden section appears in any go.mod or go.sum.
//
// Usage: checkpins [-root dir] [-pins file]. It exits 1 and prints one line
// per problem when anything is out of step.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func main() {
	root := flag.String("root", ".", "repository root")
	pinsFile := flag.String("pins", "docs/pins.md", "pins file, relative to -root unless absolute")
	flag.Parse()

	pinsPath := *pinsFile
	if !filepath.IsAbs(pinsPath) {
		pinsPath = filepath.Join(*root, pinsPath)
	}
	problems, err := check(*root, pinsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "check-pins:", err)
		os.Exit(2)
	}
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "check-pins:", p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
	fmt.Println("check-pins: ok")
}

// pins is the parsed docs/pins.md.
type pins struct {
	modules   map[string]string // module path -> version cell
	images    map[string]string // image name -> version cell
	forbidden []string          // module path prefixes
}

// check returns one line per problem found under root.
func check(root, pinsPath string) ([]string, error) {
	p, err := parsePins(pinsPath)
	if err != nil {
		return nil, err
	}
	var problems []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			switch d.Name() {
			case ".git", "testdata", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		switch {
		case d.Name() == "go.mod":
			ps, err := checkGoMod(path, rel, p)
			problems = append(problems, ps...)
			return err
		case d.Name() == "go.sum":
			ps, err := checkForbidden(path, rel, p, sumModules)
			problems = append(problems, ps...)
			return err
		case d.Name() == "Dockerfile" || strings.HasSuffix(d.Name(), ".Dockerfile"):
			ps, err := checkImages(path, rel, p, fromRe)
			problems = append(problems, ps...)
			return err
		case isComposeFile(rel):
			ps, err := checkImages(path, rel, p, imageRe)
			problems = append(problems, ps...)
			return err
		}
		return nil
	})
	return problems, err
}

func isComposeFile(rel string) bool {
	base := filepath.Base(rel)
	return (strings.HasPrefix(base, "compose") || strings.HasPrefix(base, "docker-compose")) &&
		(strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml"))
}

var (
	// A backticked name in a table's first column.
	codeRe = regexp.MustCompile("`([^`]+)`")
	// FROM [--platform=x] image[:tag] [AS name]
	fromRe = regexp.MustCompile(`(?i)^\s*FROM\s+(?:--\S+\s+)*(\S+)`)
	// image: name[:tag]
	imageRe = regexp.MustCompile(`^\s*image:\s*["']?([^"'\s#]+)`)
	// Version tokens in a version cell.
	tokenRe = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9.+_-]*`)
)

// parsePins reads the Images, Go modules and Forbidden tables. Any other
// section is documentation only.
func parsePins(path string) (*pins, error) {
	f, err := os.Open(path) //nolint:gosec // a path the caller chose
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	p := &pins{modules: map[string]string{}, images: map[string]string{}}
	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if h, ok := strings.CutPrefix(line, "## "); ok {
			section = strings.ToLower(strings.TrimSpace(h))
			continue
		}
		if !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "| ---") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) < 2 {
			continue
		}
		names := codeRe.FindAllStringSubmatch(cells[0], -1)
		for _, n := range names {
			name := strings.TrimSpace(n[1])
			switch section {
			case "go modules":
				p.modules[name] = cells[1]
			case "images":
				p.images[name] = cells[1]
			case "forbidden":
				p.forbidden = append(p.forbidden, name)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(p.modules) == 0 || len(p.images) == 0 {
		return nil, fmt.Errorf("%s: no Go modules or Images table found", path)
	}
	if len(p.forbidden) == 0 {
		return nil, fmt.Errorf("%s: no ## Forbidden table found", path)
	}
	return p, nil
}

// require is one require line of a go.mod.
type require struct {
	path, version string
	indirect      bool
}

// parseGoMod returns the requires of a go.mod, in either form.
func parseGoMod(path string) ([]require, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a go.mod found by the walk
	if err != nil {
		return nil, err
	}
	var reqs []require
	inBlock := false
	for raw := range strings.SplitSeq(string(data), "\n") {
		line := strings.TrimSpace(raw)
		indirect := strings.Contains(line, "// indirect")
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		switch {
		case line == "require (":
			inBlock = true
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		case strings.HasPrefix(line, "require "):
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		case !inBlock:
			continue
		}
		f := strings.Fields(line)
		if len(f) == 2 {
			reqs = append(reqs, require{path: f[0], version: f[1], indirect: indirect})
		}
	}
	return reqs, nil
}

func checkGoMod(path, rel string, p *pins) ([]string, error) {
	reqs, err := parseGoMod(path)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, r := range reqs {
		if f := forbiddenMatch(r.path, p.forbidden); f != "" {
			problems = append(problems, fmt.Sprintf("%s: requires %s %s, which is forbidden (%s)", rel, r.path, r.version, f))
		}
		if r.indirect {
			continue
		}
		cell, ok := p.modules[r.path]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: direct require %s %s has no row in docs/pins.md", rel, r.path, r.version))
			continue
		}
		if !versionListed(cell, r.version) {
			problems = append(problems, fmt.Sprintf("%s: %s is %s, but docs/pins.md pins %q", rel, r.path, r.version, strings.TrimSpace(cell)))
		}
	}
	return problems, nil
}

// sumModules returns the module paths named in a go.sum.
func sumModules(path string) ([]string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a go.sum found by the walk
	if err != nil {
		return nil, err
	}
	var mods []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if f := strings.Fields(line); len(f) > 0 {
			mods = append(mods, f[0])
		}
	}
	return mods, nil
}

func checkForbidden(path, rel string, p *pins, list func(string) ([]string, error)) ([]string, error) {
	mods, err := list(path)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var problems []string
	for _, m := range mods {
		if f := forbiddenMatch(m, p.forbidden); f != "" && !seen[m] {
			seen[m] = true
			problems = append(problems, fmt.Sprintf("%s: %s is in the module graph, which is forbidden (%s)", rel, m, f))
		}
	}
	return problems, nil
}

// forbiddenMatch returns the forbidden prefix that mod falls under, or "".
func forbiddenMatch(mod string, forbidden []string) string {
	for _, f := range forbidden {
		if mod == f || strings.HasPrefix(mod, f+"/") {
			return f
		}
	}
	return ""
}

func checkImages(path, rel string, p *pins, re *regexp.Regexp) ([]string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a file found by the walk
	if err != nil {
		return nil, err
	}
	stages := map[string]bool{} // FROM <earlier stage>
	var problems []string
	for raw := range strings.SplitSeq(string(data), "\n") {
		m := re.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		ref := m[1]
		if f := strings.Fields(raw); len(f) >= 4 && strings.EqualFold(f[len(f)-2], "AS") {
			stages[strings.ToLower(f[len(f)-1])] = true
		}
		if stages[strings.ToLower(ref)] || strings.Contains(ref, "$") {
			continue
		}
		name, tag := splitImage(ref)
		if tag == "" || tag == "latest" {
			problems = append(problems, fmt.Sprintf("%s: image %s has no exact tag", rel, ref))
			continue
		}
		cell, ok := p.images[name]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: image %s has no row in docs/pins.md", rel, name))
			continue
		}
		if !versionListed(cell, tag) {
			problems = append(problems, fmt.Sprintf("%s: image %s:%s, but docs/pins.md pins %q", rel, name, tag, strings.TrimSpace(cell)))
		}
	}
	return problems, nil
}

// splitImage splits name[:tag][@digest]; a colon inside the registry host is
// not a tag separator.
func splitImage(ref string) (name, tag string) {
	ref, _, _ = strings.Cut(ref, "@")
	slash := strings.LastIndex(ref, "/")
	if i := strings.LastIndex(ref, ":"); i > slash {
		return ref[:i], ref[i+1:]
	}
	return ref, ""
}

// versionListed reports whether the version cell names v exactly, or holds a
// "≥ vX" floor that v meets.
func versionListed(cell, v string) bool {
	if slices.Contains(tokenRe.FindAllString(cell, -1), v) {
		return true
	}
	if _, floor, ok := strings.Cut(cell, "≥"); ok {
		f := strings.Fields(floor)
		return len(f) > 0 && compareSemver(v, strings.Trim(f[0], "`")) >= 0
	}
	return false
}

// compareSemver compares vMAJOR.MINOR.PATCH, ignoring any pre-release or
// build suffix. It is enough for the floors in docs/pins.md.
func compareSemver(a, b string) int {
	pa, pb := semverParts(a), semverParts(b)
	for i := range pa {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func semverParts(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var out [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(s)
	}
	return out
}
