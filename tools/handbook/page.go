package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// sections are the handbook sections in navigation order (C-HB-1).
func sections() []string {
	return []string{"vision", "system", "adr", "stories", "workflows", "database", "security", "benchmarks", "operations", "glossary"}
}

// homeSection is the section of the one page at src/index.md.
const homeSection = "home"

// FrontMatter is the YAML header every page carries (contract C-HB-1).
type FrontMatter struct {
	Title      string   `yaml:"title"`
	Section    string   `yaml:"section"`
	Sources    []string `yaml:"sources"`
	ADRs       []string `yaml:"adrs"`
	Stories    []string `yaml:"stories"`
	Operations []string `yaml:"operations"`
	Tables     []string `yaml:"tables"`
	Workflows  []string `yaml:"workflows"`
	Activities []string `yaml:"activities"`
	Tests      []string `yaml:"tests"`
	Packages   []string `yaml:"packages"`
	Invariants []string `yaml:"invariants"`
	Evidence   []string `yaml:"evidence"`
	Status     string   `yaml:"status"`
	// Normative names the repository file whose summary this page is; the
	// page then carries the fixed "Summary" banner.
	Normative string `yaml:"normative"`
}

// Page is one source file under docs/handbook/src.
type Page struct {
	// Src is the path relative to the repository root, slash-separated.
	Src     string
	Section string
	Slug    string
	Front   FrontMatter
	Body    string
}

// Out is the output path relative to docs/handbook.
func (p *Page) Out() string {
	if p.Section == homeSection {
		return "index.html"
	}
	return p.Section + "/" + p.Slug + ".html"
}

// srcRel is the source path relative to docs/handbook/src.
func (p *Page) srcRel() string {
	if p.Section == homeSection {
		return "index.md"
	}
	return p.Section + "/" + p.Slug + ".md"
}

// splitFrontMatter separates the leading "---" block from the body.
func splitFrontMatter(src string) (head, body string, err error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	if !strings.HasPrefix(src, "---\n") {
		return "", "", errors.New("no front matter: the file must start with a --- line")
	}
	rest := src[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if strings.HasSuffix(rest, "\n---") {
			return rest[:len(rest)-len("\n---")], "", nil
		}
		return "", "", errors.New("front matter is not closed by a --- line")
	}
	return rest[:end], rest[end+len("\n---\n"):], nil
}

// parsePage parses one source file; src is its path under the repository root.
func parsePage(srcPath, text string) (*Page, error) {
	head, body, err := splitFrontMatter(text)
	if err != nil {
		return nil, err
	}
	var fm FrontMatter
	dec := yaml.NewDecoder(strings.NewReader(head))
	dec.KnownFields(true)
	if err := dec.Decode(&fm); err != nil {
		return nil, fmt.Errorf("front matter: %w", err)
	}
	if strings.TrimSpace(fm.Title) == "" {
		return nil, errors.New("front matter: title is required")
	}
	switch fm.Status {
	case "delivered", "in-progress":
	case "":
		return nil, errors.New("front matter: status is required (delivered or in-progress)")
	default:
		return nil, fmt.Errorf("front matter: status %q is not delivered or in-progress", fm.Status)
	}
	return &Page{Src: srcPath, Section: fm.Section, Front: fm, Body: body}, nil
}

// loadPages reads every docs/handbook/src/<section>/<slug>.md and
// src/index.md. A page whose section differs from its directory is an error.
func loadPages(root string) ([]*Page, []string) {
	srcRoot := filepath.Join(root, "docs", "handbook", "src")
	var pages []*Page
	var problems []string
	known := map[string]bool{}
	for _, s := range sections() {
		known[s] = true
	}
	_ = filepath.WalkDir(srcRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", p, err))
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		inSrc := strings.TrimPrefix(rel, "docs/handbook/src/")
		data, err := os.ReadFile(p) //nolint:gosec // a repository source file chosen by the walk
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", rel, err))
			return nil
		}
		page, err := parsePage(rel, string(data))
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", rel, err))
			return nil
		}
		dir, file := path.Split(inSrc)
		dir = strings.TrimSuffix(dir, "/")
		page.Slug = strings.TrimSuffix(file, ".md")
		switch {
		case dir == "" && page.Slug == "index":
			if page.Section != homeSection {
				problems = append(problems, fmt.Sprintf("%s: section must be %q for src/index.md, got %q", rel, homeSection, page.Section))
				return nil
			}
		case dir == "" || strings.Contains(dir, "/"):
			problems = append(problems, fmt.Sprintf("%s: a page lives at src/<section>/<slug>.md", rel))
			return nil
		case !known[dir]:
			problems = append(problems, fmt.Sprintf("%s: %q is not a handbook section", rel, dir))
			return nil
		case page.Section != dir:
			problems = append(problems, fmt.Sprintf("%s: front matter section %q differs from its directory %q", rel, page.Section, dir))
			return nil
		}
		pages = append(pages, page)
		return nil
	})
	sort.Slice(pages, func(i, j int) bool { return pages[i].Src < pages[j].Src })
	return pages, problems
}
