package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// checkRefs resolves every reference in a page's front matter. Each failure
// names the page and the reference, so the author can find it.
func (f *Facts) checkRefs(p *Page) []string {
	var out []string
	bad := func(kind, ref, why string) {
		out = append(out, fmt.Sprintf("%s: %s %q %s", p.Src, kind, ref, why))
	}
	fm := p.Front
	for _, r := range fm.ADRs {
		if _, ok := f.ADRs[r]; !ok {
			bad("adrs", r, "is not an ADR, amendment or erratum under docs/adr")
		}
	}
	for _, r := range fm.Stories {
		if _, ok := f.Stories[r]; !ok {
			bad("stories", r, "is not in docs/handbook/src/stories/stories.yaml")
		}
	}
	for _, r := range fm.Operations {
		if _, ok := f.Operations[r]; !ok {
			bad("operations", r, "is not an operationId in api/openapi.bundled.yaml")
		}
	}
	for _, r := range fm.Tables {
		if _, ok := f.Tables[r]; !ok {
			bad("tables", r, "is not a table in docs/handbook/src/database/schema.json")
		}
	}
	f.checkRegistry(fm.Workflows, "workflows", f.Registry.Workflows, bad)
	f.checkRegistry(fm.Activities, "activities", f.Registry.Activities, bad)
	for _, r := range fm.Tests {
		if !f.Tests[r] {
			bad("tests", r, "is not a Test, Benchmark, Fuzz or Example function in any _test.go file")
		}
	}
	for _, r := range fm.Packages {
		if st, err := os.Stat(filepath.Join(f.Root, filepath.FromSlash(r))); err != nil || !st.IsDir() {
			bad("packages", r, "is not a directory in the repository")
		}
	}
	for _, r := range fm.Invariants {
		if _, ok := f.Invariants[r]; !ok {
			bad("invariants", r, "is not an invariant of the ADR-001 section 1 table")
		}
	}
	for _, r := range fm.Evidence {
		f.checkPath(r, "evidence", bad)
	}
	for _, r := range fm.Sources {
		f.checkPath(strings.TrimSuffix(strings.TrimSuffix(r, "..."), "/"), "sources", bad)
	}
	if fm.Normative != "" {
		f.checkPath(fm.Normative, "normative", bad)
	}
	return out
}

func (f *Facts) checkPath(r, kind string, bad func(kind, ref, why string)) {
	if r == "" || filepath.IsAbs(r) || strings.Contains(r, "..") {
		bad(kind, r, "is not a path inside the repository")
		return
	}
	if _, err := os.Stat(filepath.Join(f.Root, filepath.FromSlash(r))); err != nil {
		bad(kind, r, "does not exist in the repository")
	}
}

func (f *Facts) checkRegistry(refs []string, kind string, get func() ([]string, bool), bad func(kind, ref, why string)) {
	if len(refs) == 0 {
		return
	}
	names, wired := get()
	if !wired {
		for _, r := range refs {
			bad(kind, r, "cannot be checked: the registry hook is not wired yet (TICK-727)")
		}
		return
	}
	have := map[string]bool{}
	for _, n := range names {
		have[n] = true
	}
	for _, r := range refs {
		if !have[r] {
			bad(kind, r, "is not in the registry")
		}
	}
}

// Backlink is a group of pages that name the same thing as the page.
type Backlink struct {
	Label string // "Table audit_log"
	Pages []*Page
}

// backlinks lists, for page p, every other page that names the same ADR,
// story, operation, table, workflow or invariant, grouped by that name.
func backlinks(p *Page, all []*Page) []Backlink {
	type key struct{ kind, name string }
	index := map[key][]*Page{}
	add := func(pg *Page, kind string, names []string) {
		for _, n := range names {
			index[key{kind, n}] = append(index[key{kind, n}], pg)
		}
	}
	for _, pg := range all {
		add(pg, "Table", pg.Front.Tables)
		add(pg, "Workflow", pg.Front.Workflows)
		add(pg, "Story", pg.Front.Stories)
		add(pg, "ADR", pg.Front.ADRs)
		add(pg, "Operation", pg.Front.Operations)
		add(pg, "Invariant", pg.Front.Invariants)
	}
	var out []Backlink
	mine := func(kind string, names []string) {
		sorted := append([]string(nil), names...)
		sort.Strings(sorted)
		for _, n := range sorted {
			var others []*Page
			for _, pg := range index[key{kind, n}] {
				if pg != p {
					others = append(others, pg)
				}
			}
			if len(others) > 0 {
				out = append(out, Backlink{Label: kind + " " + n, Pages: others})
			}
		}
	}
	mine("Table", p.Front.Tables)
	mine("Workflow", p.Front.Workflows)
	mine("Story", p.Front.Stories)
	mine("ADR", p.Front.ADRs)
	mine("Operation", p.Front.Operations)
	mine("Invariant", p.Front.Invariants)
	return out
}
