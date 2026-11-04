// Command docindex regenerates the index at the top of
// docs/delivery/decision-log.md from the per-entry files under
// docs/delivery/decisions/ (TICK-295, merge-safety D6).
//
// Usage: docindex [-dir repo-root] [-check]. With -check it writes nothing and
// exits 1 when the index is stale, which is what `task lint` runs.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	logPath     = "docs/delivery/decision-log.md"
	entriesDir  = "docs/delivery/decisions"
	startMarker = "<!-- docindex:start -->"
	endMarker   = "<!-- docindex:end -->"
)

var fileName = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-[a-z0-9]+(-[a-z0-9]+)*\.md$`)

func main() {
	os.Exit(cli(os.Args[1:], os.Stderr))
}

// cli returns the exit code: 0 the index is fresh or was rewritten, 1 it is
// stale (with -check), 2 bad usage or bad input.
func cli(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("docindex", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "repository root")
	check := fs.Bool("check", false, "fail if the index is stale instead of rewriting it")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	stale, err := run(*dir, *check)
	if err != nil {
		fmt.Fprintln(stderr, "docindex:", err)
		return 2
	}
	if stale {
		fmt.Fprintln(stderr, "docindex: the index in "+logPath+" is stale; run `go run ./tools/docindex`")
		return 1
	}
	return 0
}

// run rewrites the index, or with check only compares it. It reports whether
// the index differed from the entry files (in check mode).
func run(root string, check bool) (stale bool, err error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return false, err
	}
	defer r.Close()
	raw, err := r.ReadFile(logPath)
	if err != nil {
		return false, err
	}
	index, err := buildIndex(r)
	if err != nil {
		return false, err
	}
	want, err := splice(string(raw), index)
	if err != nil {
		return false, err
	}
	if want == string(raw) {
		return false, nil
	}
	if check {
		return true, nil
	}
	return false, r.WriteFile(logPath, []byte(want), 0o600)
}

// splice replaces the text between the markers with index.
func splice(doc, index string) (string, error) {
	s := strings.Index(doc, startMarker)
	e := strings.Index(doc, endMarker)
	if s < 0 || e < s {
		return "", errors.New(logPath + " has no " + startMarker + " ... " + endMarker + " block")
	}
	// Keep the log's own line endings: decision-log.md is CRLF (docs/** is -text).
	if strings.Contains(doc, "\r\n") {
		index = strings.ReplaceAll(index, "\n", "\r\n")
		return doc[:s+len(startMarker)] + "\r\n" + index + doc[e:], nil
	}
	return doc[:s+len(startMarker)] + "\n" + index + doc[e:], nil
}

// buildIndex lists the entry files newest first: one bullet per file, with the
// first heading as the title. A file with a bad name or no heading is an error.
func buildIndex(r *os.Root) (string, error) {
	d, err := r.Open(entriesDir)
	if err != nil {
		return "", err
	}
	defer d.Close()
	files, err := d.ReadDir(-1)
	if err != nil {
		return "", err
	}
	type entry struct{ name, date, title string }
	var entries []entry
	for _, f := range files {
		if f.IsDir() || f.Name() == "README.md" {
			continue
		}
		m := fileName.FindStringSubmatch(f.Name())
		if m == nil {
			return "", fmt.Errorf("%s: name must be <YYYY-MM-DD>-<slug>.md", f.Name())
		}
		body, err := r.ReadFile(entriesDir + "/" + f.Name())
		if err != nil {
			return "", err
		}
		title := heading(string(body))
		if title == "" {
			return "", fmt.Errorf("%s: no `## title` or `# title` heading", f.Name())
		}
		entries = append(entries, entry{f.Name(), m[1], title})
	}
	// Newest first; the same day sorts by name, descending, so the order is stable.
	sort.Slice(entries, func(i, j int) bool { return entries[i].name > entries[j].name })

	var b strings.Builder
	b.WriteString("\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "- [%s](decisions/%s)\n", e.title, e.name)
	}
	b.WriteString("\n")
	return b.String(), nil
}

// heading returns the first Markdown heading's text, with LF or CRLF endings.
func heading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		if t, ok := strings.CutPrefix(line, "## "); ok {
			return strings.TrimSpace(t)
		}
		if t, ok := strings.CutPrefix(line, "# "); ok {
			return strings.TrimSpace(t)
		}
	}
	return ""
}
