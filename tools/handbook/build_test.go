package main

import (
	"bytes"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestParsePage(t *testing.T) {
	good := "---\ntitle: T\nsection: system\nstatus: delivered\ntables: [a]\n---\nbody\n"
	p, err := parsePage("x.md", good)
	if err != nil || p.Front.Title != "T" || p.Front.Tables[0] != "a" || strings.TrimSpace(p.Body) != "body" {
		t.Fatalf("good page: %+v %v", p, err)
	}
	if _, err := parsePage("x.md", strings.ReplaceAll(good, "\n", "\r\n")); err != nil {
		t.Errorf("CRLF page: %v", err)
	}
	for name, src := range map[string]string{
		"no front matter": "body only",
		"unclosed":        "---\ntitle: T\n",
		"no title":        "---\nsection: system\nstatus: delivered\n---\n",
		"no status":       "---\ntitle: T\nsection: system\n---\n",
		"bad status":      "---\ntitle: T\nsection: system\nstatus: done\n---\n",
		"unknown key":     "---\ntitle: T\nsection: system\nstatus: delivered\nbogus: 1\n---\n",
	} {
		if _, err := parsePage("x.md", src); err == nil {
			t.Errorf("%s: parsed, want an error", name)
		}
	}
}

func TestSectionMustMatchDirectory(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "A", "security", "delivered", "", "x")
	mustFail(t, root, "docs/handbook/src/system/a.md", `front matter section "security" differs from its directory "system"`)

	root = newFixture(t)
	page(t, root, "bogus/a.md", "A", "bogus", "delivered", "", "x")
	mustFail(t, root, "docs/handbook/src/bogus/a.md", "not a handbook section")
}

func TestUnresolvedReferencesFailNamingPageAndReference(t *testing.T) {
	cases := map[string]struct{ front, ref string }{
		"operation": {"operations: [noSuchOp]\n", "noSuchOp"},
		"table":     {"tables: [no_table]\n", "no_table"},
		"adr":       {"adrs: [ADR-999]\n", "ADR-999"},
		"story":     {"stories: [S99]\n", "S99"},
		"test":      {"tests: [TestNope]\n", "TestNope"},
		"package":   {"packages: [internal/nope]\n", "internal/nope"},
		"invariant": {"invariants: [I42]\n", "I42"},
		"evidence":  {"evidence: [docs/evidence/missing.md]\n", "docs/evidence/missing.md"},
		"source":    {"sources: [internal/nope/...]\n", "internal/nope"},
		"normative": {"normative: docs/adr/nope.md\n", "docs/adr/nope.md"},
		"workflow":  {"workflows: [OutboxDispatch]\n", "OutboxDispatch"},
		"activity":  {"activities: [SomeActivity]\n", "SomeActivity"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := newFixture(t)
			page(t, root, "system/a.md", "A", "system", "delivered", c.front, "x")
			mustFail(t, root, "docs/handbook/src/system/a.md", c.ref)
		})
	}
}

type fakeRegistry struct{ wf, act []string }

func (f fakeRegistry) Workflows() ([]string, bool)  { return f.wf, true }
func (f fakeRegistry) Activities() ([]string, bool) { return f.act, true }

func TestWiredRegistryResolvesWorkflowsAndActivities(t *testing.T) {
	root := newFixture(t)
	page(t, root, "workflows/a.md", "A", "workflows", "delivered", "workflows: [OutboxDispatch]\nactivities: [Send]\n", "x")
	reg := fakeRegistry{wf: []string{"OutboxDispatch"}, act: []string{"Send"}}
	if _, err := Build(Config{Root: root, Sha: testSha, Date: "2026-11-11", Registry: reg}); err != nil {
		t.Fatalf("wired registry: %v", err)
	}
	page(t, root, "workflows/a.md", "A", "workflows", "delivered", "workflows: [Missing]\n", "x")
	if _, err := Build(Config{Root: root, Sha: testSha, Date: "2026-11-11", Registry: reg}); err == nil || !strings.Contains(err.Error(), "Missing") {
		t.Fatalf("unknown workflow: %v", err)
	}
}

func TestResolvedReferencesBuild(t *testing.T) {
	root := newFixture(t)
	front := "adrs: [ADR-001, ADR-001-E1, ADR-004-E1]\nstories: [S14]\noperations: [listAuditEntries]\ntables: [audit_log]\n" +
		"tests: [TestChainSeals, internal/audit.TestChainSeals]\npackages: [internal/audit]\ninvariants: [I1, I2]\nevidence: [docs/evidence/e.md]\n" +
		"sources: [internal/audit/...]\nnormative: docs/adr/ADR-004-audit-log.md\n"
	page(t, root, "system/a.md", "A", "system", "delivered", front, "x")
	out := buildFixture(t, root)
	html := string(out["system/a.html"])
	if !strings.Contains(html, `Summary. The normative text is the linked <a href="../../../docs/adr/ADR-004-audit-log.md">`) {
		t.Errorf("normative banner missing:\n%s", html)
	}
}

func TestBackLinks(t *testing.T) {
	root := newFixture(t)
	page(t, root, "database/audit-log.md", "Audit log table", "database", "delivered", "tables: [audit_log]\n", "x")
	page(t, root, "stories/s14.md", "Story S14", "stories", "delivered", "tables: [audit_log]\nstories: [S14]\n", "x")
	page(t, root, "stories/other.md", "Unrelated", "stories", "delivered", "tables: [tenants]\n", "x")
	out := buildFixture(t, root)
	table := string(out["database/audit-log.html"])
	if !strings.Contains(table, "Table audit_log") || !strings.Contains(table, `<a href="../stories/s14.html">Story S14</a>`) {
		t.Errorf("table page does not list the story that names it:\n%s", table)
	}
	if i := strings.Index(table, `class="backlinks"`); i < 0 || strings.Contains(table[i:], "Unrelated") {
		t.Errorf("back-links missing or list a page naming another table")
	}
	story := string(out["stories/s14.html"])
	if !strings.Contains(story, `<a href="../database/audit-log.html">Audit log table</a>`) {
		t.Errorf("story page does not list the table page:\n%s", story)
	}
	if strings.Contains(string(out["stories/other.html"]), "Also named by") {
		t.Errorf("a page nothing else names has a back-link section")
	}
}

func TestFooterStampIsInjectedAndNormalisedOut(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "A", "system", "delivered", "", "text [home](../index.md)")
	a, err := Build(Config{Root: root, Sha: "1111111111111111111111111111111111111111", Date: "2026-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(Config{Root: root, Sha: "2222222222222222222222222222222222222222", Date: "2026-02-02"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(a["index.html"]), "Content generated against main @ 111111111111 (2026-01-01)") {
		t.Errorf("footer sha missing from index.html")
	}
	if len(a) != len(b) {
		t.Fatalf("file sets differ: %d vs %d", len(a), len(b))
	}
	for name := range a {
		if bytes.Equal(a[name], b[name]) && strings.HasSuffix(name, ".html") {
			t.Errorf("%s did not take the stamp", name)
		}
		if !bytes.Equal(normalise(a[name]), normalise(b[name])) {
			t.Errorf("%s differs by more than the footer stamp", name)
		}
	}
}

func TestBuildRejectsBadStamp(t *testing.T) {
	root := newFixture(t)
	for _, c := range []Config{{Root: root, Sha: "", Date: "2026-01-01"}, {Root: root, Sha: "XYZ", Date: "2026-01-01"}, {Root: root, Sha: testSha, Date: "yesterday"}} {
		if _, err := Build(c); err == nil {
			t.Errorf("stamp %q %q accepted", c.Sha, c.Date)
		}
	}
}

func TestMissingHomePageFails(t *testing.T) {
	root := newFixture(t)
	write(t, root, "docs/handbook/src/index.md", "")
	if err := removeFile(root, "docs/handbook/src/index.md"); err != nil {
		t.Fatal(err)
	}
	mustFail(t, root, "home page is missing")
}

func TestGenOperationsMatchesSpecForOneTag(t *testing.T) {
	root := newFixture(t)
	page(t, root, "operations/audit.md", "Audit", "operations", "delivered", "", "<!-- gen:operations tag=audit -->\n")
	html := string(buildFixture(t, root)["operations/audit.html"])
	for _, want := range []string{"listAuditEntries", "GET /api/v1/audit/entries", "List audit entries", "exportAudit", "POST /api/v1/audit/export", "Export | pipe", "202, 401"} {
		if !strings.Contains(html, want) {
			t.Errorf("table lacks %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, "getMe") {
		t.Errorf("tag=audit lists an operation of another tag")
	}
	if strings.Index(html, "listAuditEntries") > strings.Index(html, "exportAudit") {
		t.Errorf("operations not in path order")
	}
}

func TestGenOperationsByIDsAndErrors(t *testing.T) {
	root := newFixture(t)
	page(t, root, "operations/a.md", "A", "operations", "delivered", "", "<!-- gen:operations ids=getMe,exportAudit -->\n")
	html := string(buildFixture(t, root)["operations/a.html"])
	if !strings.Contains(html, "getMe") || !strings.Contains(html, "exportAudit") || strings.Contains(html, "listAuditEntries") {
		t.Errorf("ids= selection wrong:\n%s", html)
	}
	for name, block := range map[string]string{
		"unknown id":     "<!-- gen:operations ids=nope -->",
		"unknown tag":    "<!-- gen:operations tag=nope -->",
		"no argument":    "<!-- gen:operations -->",
		"both":           "<!-- gen:operations tag=audit ids=getMe -->",
		"unknown kind":   "<!-- gen:nothing -->",
		"unknown arg":    "<!-- gen:operations color=red -->",
		"malformed":      "<!-- gen:operations tag= -->",
		"unknown area":   "<!-- gen:tables area=nope -->",
		"er no argument": "<!-- gen:er -->",
	} {
		root := newFixture(t)
		page(t, root, "operations/a.md", "A", "operations", "delivered", "", block+"\n")
		if _, err := Build(Config{Root: root, Sha: testSha, Date: "2026-11-11"}); err == nil || !strings.Contains(err.Error(), "operations/a.md") {
			t.Errorf("%s: want a failure naming the page, got %v", name, err)
		}
	}
}

func TestGenOperationsAgainstRealBundledSpec(t *testing.T) {
	raw, err := readRepoFile("api/openapi.bundled.yaml")
	if err != nil {
		t.Skip("no repository root:", err)
	}
	var doc struct {
		Paths map[string]map[string]struct {
			ID string `yaml:"operationId"`
		} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{}
	for p, item := range doc.Paths {
		if strings.HasPrefix(p, "/api/v1/audit/") {
			for m, op := range item {
				want[op.ID] = strings.ToUpper(m) + " " + p
			}
		}
	}
	if len(want) == 0 {
		t.Fatal("the real spec has no audit operations")
	}
	root := newFixture(t)
	write(t, root, "api/openapi.bundled.yaml", string(raw))
	page(t, root, "operations/audit.md", "Audit", "operations", "delivered", "", "<!-- gen:operations tag=audit -->\n")
	html := string(buildFixture(t, root)["operations/audit.html"])
	for id, route := range want {
		if !strings.Contains(html, id) || !strings.Contains(html, route) {
			t.Errorf("generated table lacks %s %s", id, route)
		}
	}
	if got := strings.Count(html, "<tr>") - 1; got != len(want) {
		t.Errorf("generated %d rows, the spec has %d audit operations", got, len(want))
	}
}

func TestGenTablesAndER(t *testing.T) {
	root := newFixture(t)
	page(t, root, "database/audit.md", "Audit", "database", "delivered", "", "<!-- gen:tables area=audit -->\n\n<!-- gen:er area=audit -->\n")
	out := buildFixture(t, root)
	html := string(out["database/audit.html"])
	for _, want := range []string{"audit_log", "gen_random_uuid()", "audit_log_pk", "(unique)", "nava_app", "erDiagram", `data-kind="mermaid"`} {
		if !strings.Contains(html, want) {
			t.Errorf("tables block lacks %q", want)
		}
	}
	if !strings.Contains(string(out["assets/mermaid.min.js"]), "pinned") {
		t.Errorf("pinned mermaid not copied")
	}
	if !strings.Contains(html, `assets/mermaid.min.js`) {
		t.Errorf("page with a diagram does not load mermaid")
	}
}

func TestAreasNamingAnUnknownTableFail(t *testing.T) {
	root := newFixture(t)
	write(t, root, "docs/handbook/src/database/areas.yaml", "audit: [no_such_table]\n")
	mustFail(t, root, "no_such_table")
}

func TestGenMigrationsAndBudgets(t *testing.T) {
	root := newFixture(t)
	page(t, root, "database/m.md", "M", "database", "delivered", "", "<!-- gen:migrations -->\n")
	page(t, root, "benchmarks/b.md", "B", "benchmarks", "delivered", "", "<!-- gen:budgets -->\n")
	out := buildFixture(t, root)
	if h := string(out["database/m.html"]); !strings.Contains(h, "00001_init.sql") || !strings.Contains(h, "First migration.") {
		t.Errorf("migrations block wrong:\n%s", h)
	}
	if h := string(out["benchmarks/b.html"]); !strings.Contains(h, "History length") || !strings.Contains(h, "Record history") || !strings.Contains(h, "Principal") {
		t.Errorf("budgets block wrong:\n%s", h)
	}
}

func TestGenBlockInsideCodeFenceIsLiteral(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "A", "system", "delivered", "", "```\n<!-- gen:operations tag=nope -->\n```\n")
	if !strings.Contains(string(buildFixture(t, root)["system/a.html"]), "gen:operations tag=nope") {
		t.Errorf("a gen line inside a code fence was expanded")
	}
}

func TestADRIndex(t *testing.T) {
	root := newFixture(t)
	page(t, root, "adr/index.md", "ADRs", "adr", "delivered", "", "<!-- gen:adr-index -->\n")
	html := string(buildFixture(t, root)["adr/index.html"])
	for _, want := range []string{"ADR-001", "ADR-004", "ADR-004-E1", "ADR-001-E1", "ADR-001-E2", "ACCEPTED (inherited)", "SUPERSEDED", "Audit: one log"} {
		if !strings.Contains(html, want) {
			t.Errorf("ADR index lacks %q", want)
		}
	}
	for _, not := range []string{"explained", "RFC-008"} {
		if strings.Contains(html, not) {
			t.Errorf("ADR index lists %q", not)
		}
	}
	if strings.Contains(html, "ACCEPTED (approved") {
		t.Errorf("status keeps its parenthetical note")
	}
}

func TestADRWithoutStatusFailsTheBuild(t *testing.T) {
	root := newFixture(t)
	write(t, root, "docs/adr/ADR-006-no-status.md", "---\nid: ADR-006\ntitle: No status\n---\nbody\n")
	mustFail(t, root, "ADR-006-no-status.md", "has no status")
}

func TestLinks(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/b.md", "B", "system", "delivered", "", "x")
	page(t, root, "system/a.md", "A", "system", "delivered", "",
		"[page](b.md#top) [home](../index.md) [gh](https://github.com/Navaladi2019/TicketingSystem) [repo](../../../../docs/evidence/e.md) [anchor](#x)")
	html := string(buildFixture(t, root)["system/a.html"])
	for _, want := range []string{`href="../system/b.html#top"`, `href="../index.html"`, `href="https://github.com/Navaladi2019/TicketingSystem"`, `href="../../../docs/evidence/e.md"`, `href="#x"`} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered page lacks %s:\n%s", want, html)
		}
	}
}

func TestDangerousAndBrokenLinksFail(t *testing.T) {
	for name, md := range map[string]string{
		"javascript":     "[x](javascript:alert(1))",
		"data":           "[x](data:text/html;base64,AAAA)",
		"vbscript":       "[x](vbscript:x)",
		"file":           "[x](file:///etc/passwd)",
		"autolink":       "<javascript:alert(1)>",
		"missing page":   "[x](nope.md)",
		"missing repo":   "[x](../../../../docs/nope.md)",
		"escapes repo":   "[x](../../../../../outside.md)",
		"remote image":   "![x](https://example.com/a.png)",
		"non-excalidraw": "![x](../diagrams/photo.png)",
		"missing svg":    "![x](../diagrams/none.excalidraw.svg)",
	} {
		root := newFixture(t)
		page(t, root, "system/a.md", "A", "system", "delivered", "", md)
		if _, err := Build(Config{Root: root, Sha: testSha, Date: "2026-11-11"}); err == nil || !strings.Contains(err.Error(), "system/a.md") {
			t.Errorf("%s: want a failure naming the page, got %v", name, err)
		}
	}
}

func TestRawHTMLIsNotPassedThrough(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "A", "system", "delivered", "",
		"<script>alert(1)</script>\n\ntext <img src=x onerror=alert(2)> and <iframe src=\"https://evil.example\"></iframe>\n\n<div onclick=\"x()\">hi</div>\n")
	html := string(buildFixture(t, root)["system/a.html"])
	for _, bad := range []string{"alert(1)", "onerror", "<iframe", "onclick", "evil.example"} {
		if strings.Contains(html, bad) {
			t.Errorf("raw HTML reached the page: %q", bad)
		}
	}
}

func TestExcalidrawEmbedNeedsSceneAndNoActiveContent(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "A", "system", "delivered", "", "![Two boxes](../diagrams/ok.excalidraw.svg)")
	out := buildFixture(t, root)
	html := string(out["system/a.html"])
	if !strings.Contains(html, `<img class="dz-img" src="../diagrams/ok.excalidraw.svg" alt="Two boxes">`) || strings.Contains(html, "<svg") {
		t.Errorf("excalidraw must be an <img>, never inline svg:\n%s", html)
	}
	if string(out["diagrams/ok.excalidraw.svg"]) != fixtureSvg {
		t.Errorf("svg not copied byte for byte")
	}
	for name, svg := range map[string]string{
		"no scene":     `<svg xmlns="http://www.w3.org/2000/svg"/>`,
		"script":       `<svg><!-- payload-start --><script>x()</script></svg>`,
		"handler":      `<svg><!-- payload-start --><rect onload="x()"/></svg>`,
		"foreign":      `<svg><!-- payload-start --><foreignObject/></svg>`,
		"remote image": `<svg><!-- payload-start --><image href="https://example.com/a.png"/></svg>`,
		"remote font":  `<svg><!-- payload-start --><style>@font-face{src:url(https://example.com/f.woff2)}</style></svg>`,
	} {
		root := newFixture(t)
		write(t, root, "docs/handbook/src/diagrams/bad.excalidraw.svg", svg)
		page(t, root, "system/a.md", "A", "system", "delivered", "", "![x](../diagrams/bad.excalidraw.svg)")
		if _, err := Build(Config{Root: root, Sha: testSha, Date: "2026-11-11"}); err == nil || !strings.Contains(err.Error(), "bad.excalidraw.svg") {
			t.Errorf("%s: want a failure naming the svg, got %v", name, err)
		}
	}
}

func TestMermaidFenceKeepsSourceAndLoadsPinnedScript(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "A", "system", "delivered", "", "```mermaid\nflowchart LR\n  a[\"</pre><script>x()</script>\"] --> b\n```\n")
	html := string(buildFixture(t, root)["system/a.html"])
	if !strings.Contains(html, `<pre class="mermaid-src">flowchart LR`) || strings.Contains(html, "<script>x()") {
		t.Errorf("mermaid source not kept escaped in the page:\n%s", html)
	}
	if !strings.Contains(html, `<script src="../assets/mermaid.min.js">`) {
		t.Errorf("mermaid script not loaded")
	}
}

func TestPageWithoutDiagramDoesNotLoadMermaid(t *testing.T) {
	root := newFixture(t)
	if strings.Contains(string(buildFixture(t, root)["index.html"]), "mermaid.min.js") {
		t.Errorf("a page with no diagram loads the 5 MB renderer")
	}
}

func TestInProgressBadgeAndPageChrome(t *testing.T) {
	root := newFixture(t)
	page(t, root, "stories/wip.md", "WIP", "stories", "in-progress", "", "x")
	out := buildFixture(t, root)
	html := string(out["stories/wip.html"])
	for _, want := range []string{"In progress: not on main yet", `class="crumbs"`, `href="../index.html">Handbook`, `id="hb-q"`, "Content-Security-Policy", "WIP"} {
		if !strings.Contains(html, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(string(out["index.html"]), "crumbs") {
		t.Errorf("home page has breadcrumbs")
	}
}

func TestSearchIndexEscapesScriptBreakers(t *testing.T) {
	nasty := "</script><!-- <b>&amp; \u2028 \u2029 \"quote\" \\"
	data, err := searchIndexJS([]searchEntry{{URL: "a.html", Title: nasty, Section: "x", Text: nasty}})
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, bad := range []string{"</script", "<!--", "<", ">", "&", "\u2028", "\u2029"} {
		if strings.Contains(s, bad) {
			t.Errorf("search-index.js contains %q:\n%s", bad, s)
		}
	}
	if !strings.HasPrefix(s, "window.HB_SEARCH = [") || !strings.Contains(s, `\u003c/script\u003e`) || !strings.Contains(s, `\u003c!--`) {
		t.Errorf("unexpected encoding: %s", s)
	}
}

func TestSearchIndexHoldsPageText(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "Alpha", "system", "delivered", "", "The word flamingo and `codeword`.\n\n```go\nfmt.Println(\"fenced\")\n```\n")
	idx := string(buildFixture(t, root)["assets/search-index.js"])
	for _, want := range []string{"flamingo", "codeword", "fenced", `"url":"system/a.html"`, `"title":"Alpha"`} {
		if !strings.Contains(idx, want) {
			t.Errorf("search index lacks %q", want)
		}
	}
}
