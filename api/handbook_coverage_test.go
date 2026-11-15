package api_test

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/Navaladi2019/TicketingSystem/api"
)

// The handbook gate (TICK-718 "Staying current", enforced by TICK-847): work
// that reached main has a handbook page. This test fails, naming the missing
// item, when
//
//  1. a delivered story in docs/handbook/src/stories/stories.yaml has no
//     src/stories/<id>.md page or no generated stories/<id>.html;
//  2. an ADR, amendment or erratum record docs/adr/ADR-NNN[-An|-En]-*.md has
//     no summary page src/adr/<id>.md that names it in its adrs: list;
//  3. a handbook page names an operationId (front matter operations:, or a
//     <!-- gen:operations ids=... --> block) that is not in the bundle.
//
// docs/handbook/build/build.py checks references too, but only when someone
// runs it; this runs in task test. Fix a failure by writing the page and
// running `task handbook`, never by loosening the check.

const handbookRepoRoot = ".."

var (
	adrRecordName = regexp.MustCompile(`^(ADR-\d{3}(?:-[AE]\d+)?)-.+\.md$`)
	genOperations = regexp.MustCompile(`^<!--\s*gen:operations\b.*?\bids=(\S+).*-->\s*$`)
)

// handbookFront is the part of a page's front matter the gate reads.
type handbookFront struct {
	ADRs       []string `yaml:"adrs"`
	Operations []string `yaml:"operations"`
}

// readHandbookPage splits a page into its front matter and body.
func readHandbookPage(file string) (handbookFront, string, error) {
	var front handbookFront
	data, err := os.ReadFile(file)
	if err != nil {
		return front, "", err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	rest, ok := strings.CutPrefix(text, "---\n")
	if !ok {
		return front, "", fmt.Errorf("no front matter")
	}
	head, body, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		head, ok = strings.CutSuffix(rest, "\n---")
		if !ok {
			return front, "", fmt.Errorf("front matter is not closed by a --- line")
		}
	}
	if err := yaml.Unmarshal([]byte(head), &front); err != nil {
		return front, "", fmt.Errorf("front matter: %w", err)
	}
	return front, body, nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// handbookStoryGaps is check 1.
func handbookStoryGaps(root string) []string {
	manifest := filepath.Join(root, "docs", "handbook", "src", "stories", "stories.yaml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		return []string{fmt.Sprintf("read %s: %v", manifest, err)}
	}
	var doc struct {
		Stories []struct {
			ID     string `yaml:"id"`
			Status string `yaml:"status"`
		} `yaml:"stories"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return []string{fmt.Sprintf("%s: %v", manifest, err)}
	}
	var out []string
	delivered := 0
	for _, s := range doc.Stories {
		if s.Status != "delivered" {
			continue
		}
		delivered++
		slug := strings.ToLower(s.ID)
		src := path.Join("docs/handbook/src/stories", slug+".md")
		html := path.Join("docs/handbook/stories", slug+".html")
		if !exists(filepath.Join(root, filepath.FromSlash(src))) {
			out = append(out, fmt.Sprintf("story %s is delivered in stories.yaml but %s is missing; write the page and run task handbook", s.ID, src))
		}
		if !exists(filepath.Join(root, filepath.FromSlash(html))) {
			out = append(out, fmt.Sprintf("story %s is delivered in stories.yaml but %s is missing; run task handbook and commit the output", s.ID, html))
		}
	}
	if delivered == 0 {
		out = append(out, fmt.Sprintf("%s lists no delivered story; the gate would check nothing", manifest))
	}
	return out
}

// handbookADRGaps is check 2.
func handbookADRGaps(root string) []string {
	dir := filepath.Join(root, "docs", "adr")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{fmt.Sprintf("read %s: %v", dir, err)}
	}
	var out []string
	records := 0
	for _, e := range entries {
		m := adrRecordName.FindStringSubmatch(e.Name())
		if e.IsDir() || m == nil || strings.HasSuffix(e.Name(), "-explained.md") {
			continue
		}
		records++
		id := m[1]
		rel := "docs/adr/" + e.Name()
		page := path.Join("docs/handbook/src/adr", strings.ToLower(id)+".md")
		front, _, err := readHandbookPage(filepath.Join(root, filepath.FromSlash(page)))
		switch {
		case os.IsNotExist(err):
			out = append(out, fmt.Sprintf("%s (%s) has no handbook summary page %s; write it and run task handbook", rel, id, page))
		case err != nil:
			out = append(out, fmt.Sprintf("%s: %v", page, err))
		case !contains(front.ADRs, id):
			out = append(out, fmt.Sprintf("%s is the summary page for %s but its adrs: list does not name %s", page, rel, id))
		}
	}
	if records == 0 {
		out = append(out, fmt.Sprintf("no ADR records under %s; the gate would check nothing", dir))
	}
	return out
}

// handbookOperationGaps is check 3: every operationId a page names is in ops.
func handbookOperationGaps(root string, ops map[string]bool) []string {
	src := filepath.Join(root, "docs", "handbook", "src")
	var out []string
	pages := 0
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return err
		}
		pages++
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		front, body, err := readHandbookPage(p)
		if err != nil {
			out = append(out, fmt.Sprintf("%s: %v", rel, err))
			return nil
		}
		for _, id := range front.Operations {
			if !ops[id] {
				out = append(out, fmt.Sprintf("%s: front matter operations names %q, which is not an operationId in api/openapi.bundled.yaml", rel, id))
			}
		}
		fenced := false
		for i, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "```") {
				fenced = !fenced
			}
			m := genOperations.FindStringSubmatch(strings.TrimSpace(line))
			if fenced || m == nil {
				continue
			}
			for _, id := range strings.Split(m[1], ",") {
				if !ops[id] {
					out = append(out, fmt.Sprintf("%s: gen:operations block (body line %d) names %q, which is not an operationId in api/openapi.bundled.yaml", rel, i+1, id))
				}
			}
		}
		return nil
	})
	if err != nil {
		out = append(out, fmt.Sprintf("walk %s: %v", src, err))
	}
	if pages == 0 {
		out = append(out, fmt.Sprintf("no handbook pages under %s; the gate would check nothing", src))
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func bundledOperationIDs(t *testing.T) map[string]bool {
	t.Helper()
	data, err := api.Files.ReadFile(api.SpecFile)
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	ops := map[string]bool{}
	for _, item := range spec.Paths {
		for _, n := range item {
			var op struct {
				OperationID string `yaml:"operationId"`
			}
			if n.Kind == yaml.MappingNode && n.Decode(&op) == nil && op.OperationID != "" {
				ops[op.OperationID] = true
			}
		}
	}
	if len(ops) == 0 {
		t.Fatal("no operationId in the bundle")
	}
	return ops
}

func reportGaps(t *testing.T, gaps []string) {
	t.Helper()
	sort.Strings(gaps)
	for _, g := range gaps {
		t.Error(g)
	}
}

func TestHandbookCoversDeliveredStories(t *testing.T) {
	reportGaps(t, handbookStoryGaps(handbookRepoRoot))
}

func TestHandbookCoversEveryADR(t *testing.T) {
	reportGaps(t, handbookADRGaps(handbookRepoRoot))
}

func TestHandbookOperationsExist(t *testing.T) {
	reportGaps(t, handbookOperationGaps(handbookRepoRoot, bundledOperationIDs(t)))
}

// --- the gate's own failing cases, on a fixture repository ---

// handbookFixture writes a repository that passes every check: one delivered
// story with both pages, one ADR with its summary, one page naming getMe.
func handbookFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"docs/handbook/src/stories/stories.yaml": "stories:\n  - id: S1\n    status: delivered\n  - id: S2\n    status: planned\n",
		"docs/handbook/src/stories/s1.md":        "---\ntitle: S1\nsection: stories\nstatus: delivered\noperations: [getMe]\n---\n\n<!-- gen:operations ids=getMe -->\n",
		"docs/handbook/stories/s1.html":          "<html></html>\n",
		"docs/adr/ADR-001-blueprint.md":          "---\nid: ADR-001\n---\n",
		"docs/adr/ADR-001-explained.md":          "explainer, not a record\n",
		"docs/handbook/src/adr/adr-001.md":       "---\ntitle: ADR-001\nsection: adr\nstatus: delivered\nadrs: [ADR-001]\n---\n\n```\n<!-- gen:operations ids=inAFenceIsNotChecked -->\n```\n",
	}
	for rel, body := range files {
		writeFixture(t, root, rel, body)
	}
	return root
}

func writeFixture(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHandbookGateFailingCases(t *testing.T) {
	ops := map[string]bool{"getMe": true}
	all := func(root string) []string {
		return append(append(handbookStoryGaps(root), handbookADRGaps(root)...), handbookOperationGaps(root, ops)...)
	}
	if gaps := all(handbookFixture(t)); len(gaps) != 0 {
		t.Fatalf("clean fixture: want no gaps, got %q", gaps)
	}
	cases := []struct {
		name   string
		mutate func(t *testing.T, root string)
		want   string
	}{
		{"delivered story page deleted", func(t *testing.T, root string) {
			mustRemove(t, root, "docs/handbook/src/stories/s1.md")
		}, "story S1 is delivered in stories.yaml but docs/handbook/src/stories/s1.md is missing"},
		{"delivered story html deleted", func(t *testing.T, root string) {
			mustRemove(t, root, "docs/handbook/stories/s1.html")
		}, "story S1 is delivered in stories.yaml but docs/handbook/stories/s1.html is missing"},
		{"ADR with no summary", func(t *testing.T, root string) {
			writeFixture(t, root, "docs/adr/ADR-002-new-decision.md", "---\nid: ADR-002\n---\n")
		}, "docs/adr/ADR-002-new-decision.md (ADR-002) has no handbook summary page docs/handbook/src/adr/adr-002.md"},
		{"erratum with no summary", func(t *testing.T, root string) {
			writeFixture(t, root, "docs/adr/ADR-001-E9-a-correction.md", "---\nid: ADR-001-E9\n---\n")
		}, "docs/adr/ADR-001-E9-a-correction.md (ADR-001-E9) has no handbook summary page docs/handbook/src/adr/adr-001-e9.md"},
		{"summary page does not name its ADR", func(t *testing.T, root string) {
			writeFixture(t, root, "docs/handbook/src/adr/adr-001.md", "---\ntitle: ADR-001\nadrs: [ADR-007]\n---\n")
		}, "docs/handbook/src/adr/adr-001.md is the summary page for docs/adr/ADR-001-blueprint.md but its adrs: list does not name ADR-001"},
		{"fake operation in front matter", func(t *testing.T, root string) {
			writeFixture(t, root, "docs/handbook/src/system/api.md", "---\ntitle: API\noperations: [getMe, fakeOperation]\n---\n")
		}, `docs/handbook/src/system/api.md: front matter operations names "fakeOperation"`},
		{"fake operation in a gen block", func(t *testing.T, root string) {
			writeFixture(t, root, "docs/handbook/src/system/api.md", "---\ntitle: API\n---\n\n<!-- gen:operations ids=getMe,fakeOperation -->\n")
		}, `docs/handbook/src/system/api.md: gen:operations block (body line 2) names "fakeOperation"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := handbookFixture(t)
			c.mutate(t, root)
			gaps := all(root)
			if len(gaps) != 1 || !strings.Contains(gaps[0], c.want) {
				t.Fatalf("want one gap containing %q, got %q", c.want, gaps)
			}
		})
	}
}

func mustRemove(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		t.Fatal(err)
	}
}
