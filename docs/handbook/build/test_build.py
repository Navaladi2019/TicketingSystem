#!/usr/bin/env python3
"""Tests for the handbook generator (TICK-719). Standard library only.

    python -I docs/handbook/build/test_build.py

They build a small fixture repository in a temporary directory, plus the real
handbook once, and never touch git or the network.
"""

import hashlib
import io
import os
import re
import shutil
import sys
import tempfile
import unittest
from contextlib import redirect_stderr, redirect_stdout

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import build  # noqa: E402

SHA = "abcdef0123456789abcdef0123456789abcdef01"
DATE = "2026-11-11"

SPEC = """openapi: 3.1.0
paths:
  /api/v1/me:
    get:
      operationId: getMe
      summary: The caller
      responses:
        "200": {description: ok}
  /api/v1/audit/export:
    post:
      operationId: exportAudit
      summary: 'Export | pipe'
      responses:
        "202":
          description: accepted
        "401":
          description: no
  /api/v1/audit/entries:
    get:
      operationId: listAuditEntries
      summary: List audit entries
      responses:
        "200":
          description: ok
components:
  schemas: {}
"""

ADR1 = """---
type: adr
id: ADR-001
title: Platform blueprint
status: ACCEPTED (approved 2026-10-15)
date: 2026-10-15
---

## 1. Invariants

| # | Invariant | Source |
|---|---|---|
| **I1** | **Tenant wall in the database.** Every tenant table has RLS. | P2 |
| **I2** | **The tenant comes only from a verified source.** Never the client. | P2 |

## 2. Decisions

x

## Errata

### E1 (2026-10-15): command inspection

Ruled by the CTO.

### E2 (2026-10-16): another

Status: SUPERSEDED
"""

SVG = ('<svg xmlns="http://www.w3.org/2000/svg"><!-- payload-start -->QUJD<!-- payload-end -->'
       '<rect width="1" height="1"/></svg>')

SCHEMA = """{"version":1,"tables":[
 {"name":"audit_log","kind":"partitioned","partition_by":"RANGE (occurred_at)","rls":{"enabled":true,"forced":true},
  "columns":[{"name":"id","type":"uuid","nullable":false,"default":"gen_random_uuid()"},{"name":"tenant_id","type":"uuid","nullable":false,"default":""}],
  "primary_key":["tenant_id","id"],
  "foreign_keys":[{"name":"fk","columns":["tenant_id"],"ref_table":"tenants","ref_columns":["id"],"on_delete":"NO ACTION"}],
  "indexes":[{"name":"audit_log_pk","method":"btree","unique":true,"columns":["tenant_id","id"],"include":["occurred_at"],"predicate":""}],
  "policies":[{"name":"tenant","command":"ALL","roles":["nava_app"],"permissive":true,"using":"(tenant_id = x)","check":"(tenant_id = y)"},\n   {"name":"deny_other","command":"SELECT","roles":["nava_app"],"permissive":false,"using":"(tenant_id = z)","check":""}]},\n {"name":"audit_log_default","kind":"partition","partition_of":"audit_log","partition_bound":"DEFAULT","columns":[],"primary_key":[]},
 {"name":"tenants","kind":"table","columns":[{"name":"id","type":"uuid","nullable":false,"default":""}],"primary_key":["id"]}]}"""


# The Go files GoRegistry reads, in the shape of the real ones (TICK-727).
GO_FIXTURE = {
    "internal/outbox/outbox.go": """package outbox

const InterpreterKind = "nava.interpreter"

// SealKind is the sealer.
const SealKind = "audit.seal"

func notOwnEntryPointStart(e Entry) error {
	for _, k := range [...]struct{ kind, via string }{{InterpreterKind, "EnqueueStart"}, {SealKind, "EnqueueSeal"}} {
		_ = k
	}
	return nil
}
""",
    "internal/workerguard/directive.go": """package workerguard

const DirectiveActivity = "nava.interpreter.NextStep"

const DirectiveWorkflow = "nava.interpreter"
""",
    "internal/workerguard/workflowref.go": """package workerguard

const CloseWorkflowRefActivity = "nava.workflow_ref.Close"

func (a *Activities) Register(r worker.ActivityRegistry) {
	r.RegisterActivityWithOptions(a.CloseWorkflowRef, activity.RegisterOptions{Name: CloseWorkflowRefActivity})
}
""",
    "internal/audit/seal/workflow.go": """package seal

const WorkflowType = outbox.SealKind

func RegisterWorkflow(r worker.WorkflowRegistry) {
	r.RegisterWorkflowWithOptions(Workflow, workflow.RegisterOptions{Name: WorkflowType})
}
""",
    "internal/audit/seal/activities.go": """package seal

const (
	RescheduleActivity = outbox.SealKind + ".Reschedule"
	SealActivity       = outbox.SealKind + ".Seal"
)

func (a *Activities) Register(r worker.ActivityRegistry) {
	r.RegisterActivityWithOptions(a.Reschedule, activity.RegisterOptions{Name: RescheduleActivity})
	r.RegisterActivityWithOptions(a.Seal, activity.RegisterOptions{Name: SealActivity})
}
""",
    "internal/workflow/interp/interp.go": """package interp

const WorkflowType = workerguard.DirectiveWorkflow

const (
	NextStepActivity  = workerguard.DirectiveActivity
	ApplyStepActivity = "nava.interpreter.ApplyStep" // a comment
)
""",
    "internal/workflow/interp/workflow.go": """package interp

func RegisterWorkflow(r worker.WorkflowRegistry) {
	r.RegisterWorkflowWithOptions(Workflow, workflow.RegisterOptions{Name: WorkflowType})
}
""",
    "internal/workflow/interp/activities.go": """package interp

func (a *Activities) Register(r worker.ActivityRegistry) {
	r.RegisterActivityWithOptions(a.NextStep, activity.RegisterOptions{Name: NextStepActivity})
	r.RegisterActivityWithOptions(a.ApplyStep, activity.RegisterOptions{Name: ApplyStepActivity})
}
""",
}


def write(root, rel, content):
    p = os.path.join(root, *rel.split("/"))
    os.makedirs(os.path.dirname(p), exist_ok=True)
    with open(p, "wb") as f:
        f.write(content.encode("utf-8"))


def page(root, rel, title, section, status="delivered", front="", body="x"):
    write(root, "docs/handbook/src/" + rel,
          '---\ntitle: "%s"\nsection: %s\nstatus: %s\n%s---\n\n%s\n' % (title, section, status, front, body))


def new_fixture():
    root = tempfile.mkdtemp(prefix="hbtest-")
    write(root, "api/openapi.bundled.yaml", SPEC)
    write(root, "docs/adr/ADR-001-platform-blueprint.md", ADR1)
    write(root, "docs/adr/ADR-004-audit-log.md",
          '---\nid: ADR-004\ntitle: "Audit: one log"\nstatus: ACCEPTED\ndate: 2026-11-02\n---\nbody\n')
    write(root, "docs/adr/ADR-004-E1-cascade.md",
          "---\nid: ADR-004-E1\ntitle: Cascade entries\nstatus: ACCEPTED\ndate: 2026-11-03\n---\nbody\n")
    write(root, "docs/adr/ADR-004-explained.md", "---\nid: x\ntitle: Explained\n---\nno status, excluded\n")
    write(root, "docs/adr/ADR-004-audit-log.html", "<html></html>")
    write(root, "docs/adr/ADR-004/rfc/RFC-008.md", "no front matter, in a subfolder")
    write(root, "docs/budgets.md", "# Budgets\n\n## Record history\n\n| Budget | Measured as | Limit | Owner |\n"
                                   "|---|---|---|---|\n| History length | counted | 500 | Principal |\n")
    write(root, "db/migrations/00001_init.sql", "-- +goose Up\n-- First migration.\nCREATE TABLE audit_log (id int);\n")
    write(root, "internal/audit/audit_test.go", "package audit\n\nimport \"testing\"\n\nfunc TestChainSeals(t *testing.T) {}\n"
                                                "\nfunc (s *S) TestNotAFunc() {}\n")
    write(root, "internal/audit/audit.go", "package audit\n")
    write(root, "docs/evidence/e.md", "evidence")
    write(root, "docs/handbook/assets/mermaid.min.js", "/* pinned */")
    write(root, "docs/handbook/src/diagrams/ok.excalidraw.svg", SVG)
    write(root, "docs/handbook/src/stories/stories.yaml",
          "stories:\n  - id: S14\n    issue: TICK-409\n    title: Audit log\n    status: delivered\n"
          "    evidence: [docs/evidence/e.md]\n")
    write(root, "docs/handbook/src/database/schema.json", SCHEMA)
    write(root, "docs/handbook/src/database/areas.yaml", "audit: [audit_log, audit_log_default, tenants]\n")
    for rel, text in GO_FIXTURE.items():
        write(root, rel, text)
    page(root, "index.md", "Home", "home", body="Welcome.")
    return root


def pin(root):
    digest = hashlib.sha256(b"/* pinned */").hexdigest()
    write(root, "docs/pins.md", "| `mermaid` (`docs/handbook/assets/mermaid.min.js`) | 12.3.0, sha256 `%s` | x | y |\n" % digest)


class Base(unittest.TestCase):
    def setUp(self):
        self.root = new_fixture()
        self.addCleanup(shutil.rmtree, self.root, ignore_errors=True)

    def build(self, sha=SHA, date=DATE, registry=None):
        return build.build(self.root, sha, date, registry)

    def html(self, rel, **kw):
        return self.build(**kw)[rel].decode("utf-8")

    def fails(self, *wants):
        with self.assertRaises(build.BuildError) as cm:
            self.build()
        text = str(cm.exception)
        for w in wants:
            self.assertIn(w, text)

    def commit(self, sha=SHA):
        out = self.build(sha=sha)
        build.write_output(self.root, out)


class FakeGit:
    def __init__(self, ancestor=True, error=None):
        self.ancestor, self.error, self.asked = ancestor, error, None

    def is_ancestor(self, sha):
        self.asked = sha
        if self.error:
            raise RuntimeError(self.error)
        return self.ancestor


class FakeRegistry:
    def __init__(self, wf, act):
        self._wf, self._act = wf, act

    def workflows(self):
        return self._wf

    def activities(self):
        return self._act


class TestYAML(unittest.TestCase):
    def test_subset(self):
        doc = build.parse_yaml("# c\na: 1\nb: [x, 'y z', \"q\"]\nc:\n  - p\n  - q\nd:\n- r\n"
                               "stories:\n  - id: S1\n    ev: [a]\n  - id: S2\n    title: \"T # not a comment\" # real\n")
        self.assertEqual(doc["b"], ["x", "y z", "q"])
        self.assertEqual(doc["c"], ["p", "q"])
        self.assertEqual(doc["d"], ["r"])
        self.assertEqual(doc["stories"][0], {"id": "S1", "ev": ["a"]})
        self.assertEqual(doc["stories"][1]["title"], "T # not a comment")
        self.assertEqual(build.parse_yaml("{}\n"), {})
        self.assertEqual(build.parse_yaml("stories: []\n"), {"stories": []})

    def test_duplicate_key_is_an_error(self):
        with self.assertRaises(ValueError):
            build.parse_yaml("a: 1\na: 2\n")


class TestFrontMatter(Base):
    def test_good_and_crlf(self):
        text = "---\ntitle: T\nsection: system\nstatus: delivered\ntables: [a]\n---\nbody\n"
        p = build.parse_page("x.md", text)
        self.assertEqual((p.front["title"], p.front["tables"], p.body.strip()), ("T", ["a"], "body"))
        build.parse_page("x.md", text.replace("\n", "\r\n"))

    def test_bad_pages(self):
        for name, src in {
            "no front matter": "body only",
            "unclosed": "---\ntitle: T\n",
            "no title": "---\nsection: system\nstatus: delivered\n---\n",
            "no status": "---\ntitle: T\nsection: system\n---\n",
            "bad status": "---\ntitle: T\nsection: system\nstatus: done\n---\n",
            "unknown key": "---\ntitle: T\nsection: system\nstatus: delivered\nbogus: 1\n---\n",
            "list as scalar": "---\ntitle: T\nsection: system\nstatus: delivered\ntables: a\n---\n",
        }.items():
            with self.subTest(name), self.assertRaises(ValueError):
                build.parse_page("x.md", src)

    def test_section_must_match_directory(self):
        page(self.root, "system/a.md", "A", "security")
        self.fails("docs/handbook/src/system/a.md", 'front matter section \'security\' differs from its directory \'system\'')

    def test_unknown_section_directory(self):
        page(self.root, "bogus/a.md", "A", "bogus")
        self.fails("docs/handbook/src/bogus/a.md", "not a handbook section")

    def test_missing_home_page(self):
        os.remove(os.path.join(self.root, "docs/handbook/src/index.md"))
        self.fails("home page is missing")


class TestReferences(Base):
    def test_unresolved_references_name_page_and_reference(self):
        cases = {
            "operations": "noSuchOp", "tables": "no_table", "adrs": "ADR-999", "stories": "S99",
            "tests": "TestNope", "packages": "internal/nope", "invariants": "I42",
            "evidence": "docs/evidence/missing.md", "sources": "internal/nope", "workflows": "OutboxDispatch",
            "activities": "SomeActivity",
        }
        for key, ref in cases.items():
            with self.subTest(key):
                page(self.root, "system/a.md", "A", "system", front="%s: [%s]\n" % (key, ref))
                self.fails("docs/handbook/src/system/a.md", ref)
        page(self.root, "system/a.md", "A", "system", front="normative: docs/adr/nope.md\n")
        self.fails("docs/adr/nope.md")

    def test_parent_directory_paths_are_refused(self):
        for key in ("packages", "evidence", "sources"):
            with self.subTest(key):
                page(self.root, "system/a.md", "A", "system", front="%s: [internal/../docs]\n" % key)
                self.fails("system/a.md", "is not a path inside the repository")
        write(self.root, "docs/handbook/src/stories/stories.yaml",
              "stories:\n  - id: S1\n    status: delivered\n    evidence: [../outside.md]\n")
        os.remove(os.path.join(self.root, "docs/handbook/src/database/areas.yaml"))
        self.fails("S1", "is not a path inside the repository")

    def test_a_method_is_not_a_test(self):
        page(self.root, "system/a.md", "A", "system", front="tests: [TestNotAFunc]\n")
        self.fails("TestNotAFunc")

    def test_resolved_references_build(self):
        front = ("adrs: [ADR-001, ADR-001-E1, ADR-004-E1]\nstories: [S14]\noperations: [listAuditEntries]\n"
                 "tables: [audit_log]\ntests: [TestChainSeals, internal/audit.TestChainSeals]\n"
                 "packages: [internal/audit]\ninvariants: [I1, I2]\nevidence: [docs/evidence/e.md]\n"
                 "sources: [internal/audit/...]\nnormative: docs/adr/ADR-004-audit-log.md\n")
        page(self.root, "system/a.md", "A", "system", front=front)
        h = self.html("system/a.html")
        self.assertIn('Summary. The normative text is the linked <a href="../../../docs/adr/ADR-004-audit-log.md">', h)

    def test_wired_registry(self):
        page(self.root, "workflows/a.md", "A", "workflows",
             front="build: production\nworkflows: [OutboxDispatch]\nactivities: [Send]\n")
        self.build(registry=FakeRegistry(["OutboxDispatch"], ["Send"]))
        with self.assertRaises(build.BuildError) as cm:
            self.build(registry=FakeRegistry([], ["Send"]))
        self.assertIn("OutboxDispatch", str(cm.exception))

    def test_go_registry_reads_kinds_and_activities(self):
        r = build.GoRegistry(self.root)
        r.load()
        self.assertEqual(r.kinds, {"nava.interpreter": "EnqueueStart", "audit.seal": "EnqueueSeal"})
        self.assertEqual(r.workflows(), ["audit.seal", "nava.interpreter"])
        self.assertEqual(r.activities(), ["audit.seal.Reschedule", "audit.seal.Seal", "nava.interpreter.ApplyStep",
                                          "nava.interpreter.NextStep", "nava.workflow_ref.Close"])

    def test_go_registry_resolves_front_matter(self):
        page(self.root, "workflows/a.md", "A", "workflows",
             front="build: production\nworkflows: [audit.seal]\nactivities: [audit.seal.Seal]\n")
        self.build()
        page(self.root, "workflows/a.md", "A", "workflows",
             front="build: production\nworkflows: [audit.sealer]\nactivities: [audit.seal.Sealed]\n")
        self.fails("workflows 'audit.sealer' is not in the registry", "activities 'audit.seal.Sealed' is not in the registry")

    def test_go_registry_fails_when_it_finds_nothing(self):
        write(self.root, "internal/outbox/outbox.go", "package outbox\n\nconst SealKind = \"audit.seal\"\n")
        self.fails("no {Kind, \"Enqueue...\"} rows found")

    def test_go_registry_fails_when_outbox_is_missing(self):
        os.remove(os.path.join(self.root, "internal", "outbox", "outbox.go"))
        self.fails("internal/outbox/outbox.go is missing")

    def test_go_registry_fails_on_an_unreadable_register_function(self):
        write(self.root, "internal/workerguard/workflowref.go",
              "package workerguard\n\nfunc (a *Activities) Register(r worker.ActivityRegistry) {\n"
              "\tr.RegisterActivity(a.CloseWorkflowRef)\n}\n")
        self.fails("registers nothing the parser can read")
        os.remove(os.path.join(self.root, "internal", "workerguard", "workflowref.go"))
        self.fails("internal/workerguard has no func (a *Activities) Register(...) function")

    def test_go_registry_fails_on_an_unresolved_name(self):
        write(self.root, "internal/workflow/interp/interp.go", "package interp\n\nconst WorkflowType = workerguard.DirectiveWorkflow\n")
        self.fails("internal/workflow/interp has no constant NextStepActivity")

    def test_go_registry_fails_when_kinds_and_workflows_drift(self):
        write(self.root, "internal/audit/seal/workflow.go",
              "package seal\n\nconst WorkflowType = \"audit.sealer\"\n\nfunc RegisterWorkflow(r worker.WorkflowRegistry) {\n"
              "\tr.RegisterWorkflowWithOptions(Workflow, workflow.RegisterOptions{Name: WorkflowType})\n}\n")
        self.fails("workflow type 'audit.sealer' is registered but is not a kind",
                   "kind 'audit.seal' is registered by no RegisterWorkflow")

    def test_sources_with_lines(self):
        page(self.root, "workflows/a.md", "A", "workflows",
             front="sources: [internal/audit/seal/workflow.go:3, internal/outbox/outbox.go:1-3,5]\n")
        self.build()
        page(self.root, "workflows/a.md", "A", "workflows", front="sources: [internal/audit/seal/workflow.go:99]\n")
        self.fails("names line 99, but internal/audit/seal/workflow.go has")
        page(self.root, "workflows/a.md", "A", "workflows", front="sources: [internal/nope.go:1]\n")
        self.fails("sources 'internal/nope.go' does not exist")

    def test_build_front_matter(self):
        page(self.root, "workflows/a.md", "A", "workflows", front="build: staging\n")
        self.fails("build 'staging' is not production or interptest")
        page(self.root, "workflows/a.md", "A", "workflows", front="workflows: [audit.seal]\n")
        self.fails("says which build registers them")
        page(self.root, "workflows/a.md", "A", "workflows", front="build: interptest\nworkflows: [nava.interpreter]\n")
        self.assertIn("Registered only in the <code>interptest</code> build", self.html("workflows/a.html"))

    def test_schema_version_is_checked(self):
        write(self.root, "docs/handbook/src/database/schema.json", '{"version": 2, "tables": []}')
        self.fails("version 2")
        write(self.root, "docs/handbook/src/database/schema.json", '{"version": 1, "future_key": {}, "tables": []}')
        write(self.root, "docs/handbook/src/database/areas.yaml", "{}\n")
        self.build()

    def test_backlinks(self):
        page(self.root, "database/audit-log.md", "Audit log table", "database", front="tables: [audit_log]\n")
        page(self.root, "stories/s14.md", "Story S14", "stories", front="tables: [audit_log]\nstories: [S14]\n")
        page(self.root, "stories/other.md", "Unrelated", "stories", front="tables: [tenants]\n")
        out = self.build()
        table = out["database/audit-log.html"].decode()
        self.assertIn("Table audit_log", table)
        self.assertIn('<a href="../stories/s14.html">Story S14</a>', table)
        tail = table[table.index('class="backlinks"'):]
        self.assertNotIn("Unrelated", tail)
        self.assertIn('<a href="../database/audit-log.html">Audit log table</a>', out["stories/s14.html"].decode())
        self.assertNotIn("Also named by", out["stories/other.html"].decode())


class TestStamp(Base):
    def test_footer_is_injected_and_normalised_out(self):
        page(self.root, "system/a.md", "A", "system", body="text [home](../index.md)")
        a = self.build(sha="1" * 40, date="2026-01-01")
        b = self.build(sha="2" * 40, date="2026-02-02")
        self.assertIn(b"Content generated against main @ 111111111111 (2026-01-01)", a["index.html"])
        self.assertEqual(sorted(a), sorted(b))
        for name in a:
            if name.endswith(".html"):
                self.assertNotEqual(a[name], b[name], name)
            self.assertEqual(build.normalise(a[name]), build.normalise(b[name]), name)

    def test_bad_stamp(self):
        for sha, date in (("", DATE), ("XYZ", DATE), (SHA, "yesterday")):
            with self.subTest((sha, date)), self.assertRaises(build.BuildError):
                build.build(self.root, sha, date)


class TestGen(Base):
    def test_operations_for_a_tag(self):
        page(self.root, "operations/audit.md", "Audit", "operations", body="<!-- gen:operations tag=audit -->")
        h = self.html("operations/audit.html")
        for want in ("listAuditEntries", "GET /api/v1/audit/entries", "List audit entries", "exportAudit",
                     "POST /api/v1/audit/export", "Export | pipe", "202, 401"):
            self.assertIn(want, h)
        self.assertNotIn("getMe", h)
        self.assertLess(h.index("listAuditEntries"), h.index("exportAudit"))

    def test_operations_by_ids(self):
        page(self.root, "operations/a.md", "A", "operations", body="<!-- gen:operations ids=getMe,exportAudit -->")
        h = self.html("operations/a.html")
        self.assertIn("getMe", h)
        self.assertIn("exportAudit", h)
        self.assertNotIn("listAuditEntries", h)

    def test_malformed_blocks_fail_naming_the_page(self):
        for name, block in {
            "unknown id": "<!-- gen:operations ids=nope -->", "unknown tag": "<!-- gen:operations tag=nope -->",
            "no argument": "<!-- gen:operations -->", "both": "<!-- gen:operations tag=audit ids=getMe -->",
            "unknown kind": "<!-- gen:nothing -->", "unknown arg": "<!-- gen:operations color=red -->",
            "malformed": "<!-- gen:operations tag= -->", "unknown area": "<!-- gen:tables area=nope -->",
            "er no argument": "<!-- gen:er -->",
        }.items():
            with self.subTest(name):
                page(self.root, "operations/a.md", "A", "operations", body=block)
                self.fails("operations/a.md")

    def test_operations_against_the_real_bundled_spec(self):
        real = os.path.join(build.ROOT, "api", "openapi.bundled.yaml")
        if not os.path.exists(real):
            self.skipTest("no repository spec")
        text = build.read_text(real)
        write(self.root, "api/openapi.bundled.yaml", text)
        # An independent scan: every operationId and route under /api/v1/audit/.
        want, path = {}, None
        for line in text.split("\n"):
            m = re.match(r"^  (/\S*):\s*$", line)
            if m:
                path = m.group(1)
            m = re.match(r"^    (get|put|post|delete|patch):\s*$", line)
            if m and path:
                method = m.group(1).upper()
            m = re.match(r"^      operationId:\s*(\S+)", line)
            if m and path and path.startswith("/api/v1/audit/"):
                want[m.group(1)] = "%s %s" % (method, path)
        self.assertTrue(want, "the real spec has no audit operations")
        page(self.root, "operations/audit.md", "Audit", "operations", body="<!-- gen:operations tag=audit -->")
        h = self.html("operations/audit.html")
        for ident, route in want.items():
            self.assertIn(ident, h)
            self.assertIn(route, h)
        self.assertEqual(h.count("<tr>") - 1, len(want))

    def test_tables_and_er(self):
        page(self.root, "database/audit.md", "Audit", "database",
             body="<!-- gen:tables area=audit -->\n\n<!-- gen:er area=audit -->")
        out = self.build()
        h = out["database/audit.html"].decode()
        for want in ("audit_log", "gen_random_uuid()", "audit_log_pk", "(unique)", "nava_app", "erDiagram", 'data-kind="mermaid"'):
            self.assertIn(want, h)
        self.assertIn("assets/mermaid.min.js", h)

    def test_tables_render_the_isolation_fields(self):
        page(self.root, "database/audit.md", "Audit", "database", body="<!-- gen:tables area=audit -->")
        h = self.html("database/audit.html")
        self.assertIn("<th>Type</th>", h)
        self.assertIn("<strong>RESTRICTIVE</strong>", h)          # permissive: false
        self.assertEqual(h.count("<strong>RESTRICTIVE</strong>"), 1)
        self.assertIn("<td>permissive</td>", h)                   # the other policy
        self.assertIn("<th>Include</th>", h)
        self.assertIn("<code>occurred_at</code>", h)              # index include
        self.assertIn("Partitioned by <code>RANGE (occurred_at)</code>", h)
        self.assertIn("Partition bound: <code>DEFAULT</code>", h)

    def test_schema_entry_without_a_name_is_a_named_error(self):
        write(self.root, "docs/handbook/src/database/schema.json", '{"version":1,"tables":[{"kind":"table"}]}')
        write(self.root, "docs/handbook/src/database/areas.yaml", "{}")
        self.fails("tables[0] has no name")

    def test_area_naming_an_unknown_table(self):
        write(self.root, "docs/handbook/src/database/areas.yaml", "audit: [no_such_table]\n")
        self.fails("no_such_table")

    def test_migrations_and_budgets(self):
        page(self.root, "database/m.md", "M", "database", body="<!-- gen:migrations -->")
        page(self.root, "benchmarks/b.md", "B", "benchmarks", body="<!-- gen:budgets -->")
        out = self.build()
        m = out["database/m.html"].decode()
        self.assertIn("00001_init.sql", m)
        self.assertIn("First migration.", m)
        b = out["benchmarks/b.html"].decode()
        for want in ("History length", "Record history", "Principal"):
            self.assertIn(want, b)

    def test_gen_line_in_a_code_fence_is_literal(self):
        page(self.root, "system/a.md", "A", "system", body="```\n<!-- gen:operations tag=nope -->\n```")
        self.assertIn("gen:operations tag=nope", self.html("system/a.html"))

    def test_adr_index(self):
        page(self.root, "adr/index.md", "ADRs", "adr", body="<!-- gen:adr-index -->")
        h = self.html("adr/index.html")
        for want in ("ADR-001", "ADR-004", "ADR-004-E1", "ADR-001-E1", "ADR-001-E2", "ACCEPTED (inherited)",
                     "SUPERSEDED", "Audit: one log"):
            self.assertIn(want, h)
        for no in ("explained", "RFC-008", "ACCEPTED (approved"):
            self.assertNotIn(no, h)

    def test_adr_without_status_fails(self):
        write(self.root, "docs/adr/ADR-006-no-status.md", "---\nid: ADR-006\ntitle: No status\n---\nbody\n")
        self.fails("ADR-006-no-status.md", "has no status")


class TestMarkdown(Base):
    def test_links(self):
        page(self.root, "system/b.md", "B", "system")
        page(self.root, "system/a.md", "A", "system",
             body="[page](b.md#top) [home](../index.md) [gh](https://github.com/x/y) "
                  "[repo](../../../../docs/evidence/e.md) [anchor](#x)")
        h = self.html("system/a.html")
        for want in ('href="../system/b.html#top"', 'href="../index.html"', 'href="https://github.com/x/y"',
                     'href="../../../docs/evidence/e.md"', 'href="#x"'):
            self.assertIn(want, h)

    def test_dangerous_and_broken_links_fail(self):
        for name, md in {
            "javascript": "[x](javascript:alert(1))", "data": "[x](data:text/html;base64,AAAA)",
            "vbscript": "[x](vbscript:x)", "file": "[x](file:///etc/passwd)", "autolink": "<javascript:alert(1)>",
            "missing page": "[x](nope.md)", "missing repo": "[x](../../../../docs/nope.md)",
            "escapes repo": "[x](../../../../../outside.md)", "remote image": "![x](https://example.com/a.png)",
            "non-excalidraw": "![x](../diagrams/photo.png)", "missing svg": "![x](../diagrams/none.excalidraw.svg)",
        }.items():
            with self.subTest(name):
                page(self.root, "system/a.md", "A", "system", body=md)
                self.fails("system/a.md")

    def test_raw_html_is_not_passed_through(self):
        page(self.root, "system/a.md", "A", "system",
             body="<script>alert(1)</script>\n\ntext <img src=x onerror=alert(2)> and "
                  "<iframe src=\"https://evil.example\"></iframe>\n\n<div onclick=\"x()\">hi</div>")
        h = self.html("system/a.html")
        for bad in ("<script>", "<img src=x", "<iframe", "<div onclick"):
            self.assertNotIn(bad, h)
        self.assertEqual(build.offline_file_violations("system/a.html", h), [])

    def test_blocks_and_inline(self):
        page(self.root, "system/a.md", "A", "system",
             body="## Heading *one*\n\n- a\n- b `code <x>`\n  - nested\n\n1. one\n2. two\n\n> quote\n\n"
                  "| h1 | h2 |\n|---|:-:|\n| c1 | **c2** |\n\n```go\nfmt.Println(\"<>\")\n```\n\n"
                  "snake_case_word and ~~gone~~ and _em_\n\n---\n")
        h = self.html("system/a.html")
        for want in ('<h2 id="heading-one">Heading <em>one</em></h2>', "<ul>", "<li>a</li>",
                     "<code>code &lt;x&gt;</code>", "<ul>\n<li>nested</li>", "<ol>", "<blockquote>", "<th>h1</th>",
                     '<th align="center">h2</th>', "<strong>c2</strong>", '<code class="language-go">',
                     "&lt;&gt;", "snake_case_word", "<del>gone</del>", "<em>em</em>", "<hr>"):
            self.assertIn(want, h)

    def test_excalidraw_embed(self):
        page(self.root, "system/a.md", "A", "system", body="![Two boxes](../diagrams/ok.excalidraw.svg)")
        out = self.build()
        h = out["system/a.html"].decode()
        self.assertIn('<img class="dz-img" src="../diagrams/ok.excalidraw.svg" alt="Two boxes">', h)
        self.assertNotIn("<svg", h)
        self.assertEqual(out["diagrams/ok.excalidraw.svg"].decode(), SVG)

    def test_image_figure_has_no_paragraph_wrapper(self):
        page(self.root, "system/a.md", "A", "system", body="![Two boxes](../diagrams/ok.excalidraw.svg)\n\ntext")
        h = self.html("system/a.html")
        self.assertNotIn("<p><figure", h)
        self.assertNotIn("</figure></p>", h)
        self.assertIn("<p>text</p>", h)

    def test_excalidraw_refuses_bad_svg(self):
        bad = {
            "no scene": '<svg xmlns="http://www.w3.org/2000/svg"/>',
            "script": "<svg><!-- payload-start --><script>x()</script></svg>",
            "handler": '<svg><!-- payload-start --><rect onload="x()"/></svg>',
            "foreign": "<svg><!-- payload-start --><foreignObject/></svg>",
            "remote image": '<svg><!-- payload-start --><image href="https://example.com/a.png"/></svg>',
            "remote font": "<svg><!-- payload-start --><style>@font-face{src:url(https://example.com/f.woff2)}</style></svg>",
            "prefixed script": '<svg xmlns:svg="http://www.w3.org/2000/svg"><!-- payload-start --><svg:script>x()</svg:script></svg>',
            "data link": '<svg><!-- payload-start --><a href="data:text/html,x"><text>x</text></a></svg>',
            "entity scheme": '<svg><!-- payload-start --><a href="javascript&#x3a;x"><text>x</text></a></svg>',
            "xlink javascript": '<svg xmlns:xlink="http://www.w3.org/1999/xlink"><!-- payload-start --><use xlink:href="javascript:x()"/></svg>',
            "doctype": '<!DOCTYPE svg [<!ENTITY e "x">]><svg><!-- payload-start --></svg>',
            "not xml": "<svg><!-- payload-start --><rect></svg>",
        }
        for name, svg in bad.items():
            with self.subTest(name):
                write(self.root, "docs/handbook/src/diagrams/bad.excalidraw.svg", svg)
                page(self.root, "system/a.md", "A", "system", body="![x](../diagrams/bad.excalidraw.svg)")
                self.fails("bad.excalidraw.svg")

    def test_allowlist_accepts_a_normal_diagram(self):
        ok = ('<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><!-- payload-start -->'
              '<defs><style>.a{fill:red}</style></defs><g><path d="M0 0"/><use xlink:href="#a"/><text><tspan>x</tspan></text></g></svg>')
        self.assertEqual(build.check_excalidraw_svg(ok), [])
        self.assertEqual(build.check_excalidraw_svg(SVG), [])

    def test_mermaid_fence(self):
        page(self.root, "system/a.md", "A", "system",
             body='```mermaid\nflowchart LR\n  a["</pre><script>x()</script>"] --> b\n```')
        h = self.html("system/a.html")
        self.assertIn('<pre class="mermaid-src">flowchart LR', h)
        self.assertNotIn("<script>x()", h)
        self.assertIn('<script src="../assets/mermaid.min.js">', h)

    def test_page_without_diagram_does_not_load_mermaid(self):
        self.assertNotIn("mermaid.min.js", self.html("index.html"))

    def test_chrome(self):
        page(self.root, "stories/wip.md", "WIP", "stories", status="in-progress")
        out = self.build()
        h = out["stories/wip.html"].decode()
        for want in ("In progress: not on main yet", 'class="crumbs"', 'href="../index.html">Handbook', 'id="hb-q"',
                     "Content-Security-Policy", "WIP"):
            self.assertIn(want, h)
        self.assertNotIn("crumbs", out["index.html"].decode())


class TestSearch(Base):
    def test_index_escapes_script_breakers(self):
        nasty = "</script><!-- <b>&amp; \u2028 \u2029 \"quote\" \\"
        s = build.search_index_js([{"url": "a.html", "title": nasty, "section": "x", "text": nasty}]).decode()
        for bad in ("</script", "<!--", "<", ">", "&", "\u2028", "\u2029"):
            self.assertNotIn(bad, s)
        self.assertTrue(s.startswith("window.HB_SEARCH = ["))
        self.assertIn("\\u003c/script\\u003e", s)
        self.assertIn("\\u003c!--", s)

    def test_index_holds_page_text(self):
        page(self.root, "system/a.md", "Alpha", "system",
             body="The word flamingo and `codeword`.\n\n```go\nfmt.Println(\"fenced\")\n```")
        idx = self.html("assets/search-index.js")
        for want in ("flamingo", "codeword", "fenced", '"url":"system/a.html"', '"title":"Alpha"'):
            self.assertIn(want, idx)


class TestOffline(Base):
    def test_rules_catch_every_kind_of_network_load(self):
        bad = {
            "a.html": '<script src="https://cdn.example/x.js"></script>', "b.html": '<img src="http://example.com/a.png">',
            "c.html": '<link rel="stylesheet" href="//fonts.example/x.css">', "d.html": "<iframe src='https://example.com'></iframe>",
            "e.html": '<style>@import "x.css";</style>', "f.html": '<div style="background:url(https://example.com/a.png)"></div>',
            "g.html": '<meta http-equiv="refresh" content="0;url=https://example.com">',
            "h.html": '<script\n src = "https://cdn.example/x.js"></script>',
            "a.css": '@import url("https://x.example/a.css");', "b.css": "body{background:url(https://x.example/a.png)}",
            "c.css": "body{background:url(//x.example/a.png)}", "a.js": 'fetch("data.json")',
            "b.js": "var x = new XMLHttpRequest();", "c.js": 'var u = "https://example.com/x";',
            "d.js": 'navigator.sendBeacon("/x")', "e.js": 'new WebSocket("wss://x")',
        }
        for name, src in bad.items():
            with self.subTest(name):
                v = build.offline_file_violations(name, src)
                self.assertTrue([x for x in v if "lacks the CSP" not in x], src)
        good = {
            "a.css": "body{background:url(img.png)}", "a.js": 'var s = "search"; document.getElementById("x");',
        }
        for name, src in good.items():
            self.assertEqual(build.offline_file_violations(name, src), [], name)
        anchor = ('<a href="https://github.com/x">repo</a><script src="../assets/x.js"></script><img src="../d/a.svg">'
                  '<link rel="stylesheet" href="assets/handbook.css">')
        self.assertEqual([x for x in build.offline_file_violations("a.html", anchor) if "lacks the CSP" not in x], [])

    def test_fixture_loads_nothing_over_the_network(self):
        page(self.root, "system/a.md", "A", "system",
             body="```mermaid\nflowchart LR\n a-->b\n```\n![x](../diagrams/ok.excalidraw.svg)\n[gh](https://github.com/x)")
        self.assertEqual(build.offline_violations(self.build()), [])


class TestCheck(Base):
    def setUp(self):
        super().setUp()
        pin(self.root)

    def test_clean_even_when_main_moves(self):
        page(self.root, "system/a.md", "A", "system")
        self.commit()
        g = FakeGit()
        self.assertEqual(build.check(self.root, g), [])
        self.assertEqual(g.asked, SHA[:12])
        self.commit("1234567890abcdef1234567890abcdef12345678")
        self.assertEqual(build.check(self.root, FakeGit()), [])

    def test_stale_missing_and_extra_files(self):
        page(self.root, "system/a.md", "A", "system")
        self.commit()
        page(self.root, "system/a.md", "A", "system", body="edited without a rebuild")
        problems = "\n".join(build.check(self.root, FakeGit()))
        self.assertIn("system/a.html is stale", problems)
        self.assertIn("search-index.js is stale", problems)
        self.commit()
        os.remove(os.path.join(self.root, "docs/handbook/system/a.html"))
        write(self.root, "docs/handbook/orphan.html", "x")
        problems = "\n".join(build.check(self.root, FakeGit()))
        self.assertIn("system/a.html is missing", problems)
        self.assertIn("orphan.html is not generated", problems)

    def test_stamp_must_be_an_ancestor(self):
        self.commit()
        self.assertIn("is not an ancestor of HEAD", "\n".join(build.check(self.root, FakeGit(ancestor=False))))
        self.assertIn("shallow clone", "\n".join(build.check(self.root, FakeGit(error="shallow clone"))))

    def test_mermaid_must_match_its_pin(self):
        self.commit()
        write(self.root, "docs/handbook/assets/mermaid.min.js", "/* tampered */")
        self.assertIn("docs/pins.md pins", "\n".join(build.check(self.root, FakeGit())))
        write(self.root, "docs/pins.md", "no row")
        self.assertIn("no `mermaid` row", "\n".join(build.check(self.root, FakeGit())))

    def test_scanner_hook_sees_every_artefact(self):
        page(self.root, "system/a.md", "A", "system", body="![x](../diagrams/ok.excalidraw.svg)")
        self.commit()
        seen = set()

        def scan(rel, data):
            seen.add(rel)
            return ["rule X matched"] if rel == "system/a.html" else []

        problems = "\n".join(build.check(self.root, FakeGit(), scanner=scan))
        for want in ("index.html", "system/a.html", "assets/search-index.js", "assets/mermaid.min.js",
                     "src/database/schema.json", "src/diagrams/ok.excalidraw.svg", "diagrams/ok.excalidraw.svg"):
            self.assertIn(want, seen)
        self.assertIn("sensitivity scan: system/a.html: rule X matched", problems)
        self.assertEqual(build.no_rules("x", b"password=hunter2"), [])

    def test_write_removes_stale_files_but_keeps_sources_pin_and_build(self):
        page(self.root, "system/a.md", "A", "system")
        self.commit()
        write(self.root, "docs/handbook/README.md", "readme")
        write(self.root, "docs/handbook/build/build.py", "# the generator")
        write(self.root, "docs/handbook/system/old.html", "stale generated page")
        write(self.root, "docs/handbook/notes.txt", "not ours")
        write(self.root, "docs/handbook/gone/old.html", "not in a section")
        self.commit()
        for keep in ("README.md", "src/index.md", "assets/mermaid.min.js", "system/a.html", "build/build.py",
                     "notes.txt", "gone/old.html"):
            self.assertTrue(os.path.exists(os.path.join(self.root, "docs/handbook", keep)), keep)
        self.assertFalse(os.path.exists(os.path.join(self.root, "docs/handbook/system/old.html")))
        # --check still reports what the build does not own
        problems = "\n".join(build.check(self.root, FakeGit()))
        self.assertIn("notes.txt is not generated", problems)


class TestCLI(Base):
    def run_cli(self, *args):
        out, err = io.StringIO(), io.StringIO()
        with redirect_stdout(out), redirect_stderr(err):
            code = build.main(["--root", self.root] + list(args))
        return code, out.getvalue(), err.getvalue()

    def test_builds_with_injected_stamp_and_no_git(self):
        code, _out, err = self.run_cli("--sha", SHA, "--date", DATE)
        self.assertEqual(code, 0, err)
        with open(os.path.join(self.root, "docs/handbook/index.html"), encoding="utf-8") as f:
            self.assertIn("main @ abcdef012345 (2026-11-11)", f.read())

    def test_build_failure_exits_1_naming_page_and_reference(self):
        page(self.root, "system/a.md", "A", "system", front="operations: [nope]\n")
        code, _out, err = self.run_cli("--sha", SHA, "--date", DATE)
        self.assertEqual(code, 1)
        self.assertIn("docs/handbook/src/system/a.md", err)
        self.assertIn("nope", err)


class TestRealHandbook(unittest.TestCase):
    def test_builds_and_loads_nothing_over_the_network(self):
        out = build.build(build.ROOT, SHA, DATE)  # build() also runs the offline scan
        for want in ("index.html", "assets/handbook.css", "assets/handbook.js", "assets/search-index.js",
                     "assets/mermaid.min.js"):
            self.assertTrue(out[want], want)
        for s in build.SECTIONS:
            self.assertTrue(out[s + "/index.html"], s)
        self.assertEqual(build.offline_violations(out), [])


if __name__ == "__main__":
    unittest.main()
