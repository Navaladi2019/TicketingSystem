#!/usr/bin/env python3
"""Product handbook generator (TICK-719, contract C-HB-1).

Builds docs/handbook/src/**/*.md into docs/handbook/**/*.html: pages that open
from file:// with no network request. Standard library only.

    python -I docs/handbook/build/build.py            # write the handbook
    python -I docs/handbook/build/build.py --check    # fail when it is stale

The footer stamp is an input: --sha / --date, else $HANDBOOK_SHA /
$HANDBOOK_DATE, else `git merge-base HEAD origin/main` and that commit's date.
--check rebuilds in memory and compares with the committed files, ignoring the
stamp (so a moving main never makes the handbook stale); it then requires the
stamped sha to be an ancestor of HEAD and mermaid.min.js to match the sha256
pinned in docs/pins.md. See docs/handbook/README.md.
"""

import argparse
import hashlib
import html
import json
import os
import re
import subprocess
import sys
import xml.etree.ElementTree as ET
from urllib.parse import urlparse

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", "..", ".."))
HANDBOOK = "docs/handbook"
SECTIONS = ["vision", "system", "adr", "stories", "workflows", "database",
            "security", "benchmarks", "operations", "glossary"]
HOME = "home"

LIST_KEYS = ["sources", "adrs", "stories", "operations", "tables", "workflows",
             "activities", "tests", "packages", "invariants", "evidence"]
SCALAR_KEYS = ["title", "section", "status", "normative"]


class BuildError(Exception):
    def __init__(self, problems):
        self.problems = problems
        super().__init__("handbook build failed:\n  " + "\n  ".join(problems))


# ---------------------------------------------------------------- mini YAML

def _strip_comment(line):
    out, quote = [], None
    for i, ch in enumerate(line):
        if quote:
            if ch == quote:
                quote = None
        elif ch in "\"'":
            quote = ch
        elif ch == "#" and (i == 0 or line[i - 1] in " \t"):
            break
        out.append(ch)
    return "".join(out).rstrip()


def _unquote(s):
    s = s.strip()
    if len(s) >= 2 and s[0] == s[-1] == '"':
        return json.loads(s) if "\\" in s else s[1:-1]
    if len(s) >= 2 and s[0] == s[-1] == "'":
        return s[1:-1].replace("''", "'")
    return s


def _split_flow(inner):
    items, cur, quote = [], [], None
    for ch in inner:
        if quote:
            cur.append(ch)
            if ch == quote:
                quote = None
        elif ch in "\"'":
            quote = ch
            cur.append(ch)
        elif ch == ",":
            items.append("".join(cur))
            cur = []
        else:
            cur.append(ch)
    if "".join(cur).strip():
        items.append("".join(cur))
    return [_unquote(x) for x in items]


def _scalar(v):
    v = v.strip()
    if v.startswith("[") and v.endswith("]"):
        return _split_flow(v[1:-1])
    if v == "{}":
        return {}
    return _unquote(v)


def parse_yaml(text):
    """A strict subset: block mappings, block lists, flow lists, scalars.

    Every scalar is a string. Duplicate keys are an error."""
    lines = []
    for raw in text.replace("\r\n", "\n").split("\n"):
        s = _strip_comment(raw)
        if s.strip():
            lines.append((len(s) - len(s.lstrip(" ")), s.strip()))
    if not lines or [l[1] for l in lines] == ["{}"]:
        return {}
    value, pos = _block(lines, 0, lines[0][0])
    if pos != len(lines):
        raise ValueError("unexpected content: " + lines[pos][1])
    return value


def _block(lines, pos, indent):
    if lines[pos][1].startswith("- ") or lines[pos][1] == "-":
        return _list(lines, pos, indent)
    return _mapping(lines, pos, indent)


def _mapping(lines, pos, indent):
    out = {}
    while pos < len(lines) and lines[pos][0] == indent and not lines[pos][1].startswith("- "):
        text = lines[pos][1]
        key, sep, rest = text.partition(":")
        if not sep:
            raise ValueError("not a key: value line: " + text)
        key = _unquote(key)
        if key in out:
            raise ValueError("duplicate key " + key)
        rest = rest.strip()
        pos += 1
        if rest:
            out[key] = _scalar(rest)
        elif pos < len(lines) and (lines[pos][0] > indent or
                                   (lines[pos][0] == indent and lines[pos][1].startswith("- "))):
            out[key], pos = _block(lines, pos, lines[pos][0])
        else:
            out[key] = ""
    return out, pos


def _list(lines, pos, indent):
    out = []
    while pos < len(lines) and lines[pos][0] == indent and lines[pos][1].startswith("-"):
        text = lines[pos][1][1:].strip()
        if not text:
            pos += 1
            if pos < len(lines) and lines[pos][0] > indent:
                item, pos = _block(lines, pos, lines[pos][0])
                out.append(item)
            continue
        if re.match(r"^[A-Za-z_][\w-]*\s*:(\s|$)", text):
            # a mapping that starts on the dash line; its other keys sit at indent+2
            inner = [(indent + 2, text)]
            pos += 1
            while pos < len(lines) and lines[pos][0] > indent:
                inner.append(lines[pos])
                pos += 1
            item, _ = _mapping(inner, 0, indent + 2)
            out.append(item)
        else:
            out.append(_scalar(text))
            pos += 1
    return out, pos


# ------------------------------------------------------------------- pages

def read_text(path):
    with open(path, "rb") as f:
        return f.read().decode("utf-8").replace("\r\n", "\n")


def split_front_matter(src):
    src = src.replace("\r\n", "\n")
    if not src.startswith("---\n"):
        raise ValueError("no front matter: the file must start with a --- line")
    rest = src[4:]
    end = rest.find("\n---\n")
    if end < 0:
        if rest.endswith("\n---"):
            return rest[:-4], ""
        raise ValueError("front matter is not closed by a --- line")
    return rest[:end], rest[end + 5:]


class Page:
    def __init__(self, src, front, body):
        self.src = src
        self.front = front
        self.body = body
        self.section = front["section"]
        self.slug = ""

    def out(self):
        if self.section == HOME:
            return "index.html"
        return "%s/%s.html" % (self.section, self.slug)

    def src_rel(self):
        if self.section == HOME:
            return "index.md"
        return "%s/%s.md" % (self.section, self.slug)


def parse_page(src_path, text):
    head, body = split_front_matter(text)
    try:
        raw = parse_yaml(head)
    except ValueError as e:
        raise ValueError("front matter: %s" % e)
    if not isinstance(raw, dict):
        raise ValueError("front matter: not a mapping")
    front = {k: [] for k in LIST_KEYS}
    front.update({k: "" for k in SCALAR_KEYS})
    for k, v in raw.items():
        if k in LIST_KEYS:
            if isinstance(v, str):
                raise ValueError("front matter: %s must be a list, like [a, b]" % k)
            if not isinstance(v, list) or any(not isinstance(x, str) for x in v):
                raise ValueError("front matter: %s must be a list of strings" % k)
            front[k] = v
        elif k in SCALAR_KEYS:
            if not isinstance(v, str):
                raise ValueError("front matter: %s must be a string" % k)
            front[k] = v
        else:
            raise ValueError("front matter: unknown key %r" % k)
    if not front["title"].strip():
        raise ValueError("front matter: title is required")
    if front["status"] == "":
        raise ValueError("front matter: status is required (delivered or in-progress)")
    if front["status"] not in ("delivered", "in-progress"):
        raise ValueError("front matter: status %r is not delivered or in-progress" % front["status"])
    return Page(src_path, front, body)


def load_pages(root):
    pages, problems = [], []
    src_root = os.path.join(root, HANDBOOK, "src")
    for dirpath, _dirs, files in os.walk(src_root):
        for name in files:
            if not name.endswith(".md"):
                continue
            full = os.path.join(dirpath, name)
            rel = os.path.relpath(full, root).replace(os.sep, "/")
            in_src = rel[len(HANDBOOK + "/src/"):]
            try:
                page = parse_page(rel, read_text(full))
            except (ValueError, UnicodeDecodeError) as e:
                problems.append("%s: %s" % (rel, e))
                continue
            d, _, f = in_src.rpartition("/")
            page.slug = f[:-3]
            if d == "" and page.slug == "index":
                if page.section != HOME:
                    problems.append("%s: section must be %r for src/index.md, got %r" % (rel, HOME, page.section))
                    continue
            elif d == "" or "/" in d:
                problems.append("%s: a page lives at src/<section>/<slug>.md" % rel)
                continue
            elif d not in SECTIONS:
                problems.append("%s: %r is not a handbook section" % (rel, d))
                continue
            elif page.section != d:
                problems.append("%s: front matter section %r differs from its directory %r" % (rel, page.section, d))
                continue
            pages.append(page)
    pages.sort(key=lambda p: p.src)
    return pages, problems


# ------------------------------------------------------------------- facts

class Unwired:
    """The workflow/activity registry hook TICK-727 wires. Until then it knows nothing."""

    def workflows(self):
        return None

    def activities(self):
        return None


class Facts:
    def __init__(self, root, registry):
        self.root = root
        self.registry = registry
        self.adrs, self.adr_list = {}, []
        self.stories, self.story_list = {}, []
        self.operations, self.op_list = {}, []
        self.tables, self.table_list = {}, []
        self.areas = {}
        self.invariants = {}
        self.tests = set()
        self.problems = []
        for load in (self.load_adrs, self.load_stories, self.load_operations,
                     self.load_schema, self.load_invariants, self.load_tests):
            try:
                load()
            except (OSError, ValueError) as e:
                self.problems.append(str(e))

    def path(self, *parts):
        return os.path.join(self.root, *parts)

    # ADRs: file records plus inline "### E1" headings under "## Errata".
    def load_adrs(self):
        d = self.path("docs", "adr")
        name_re = re.compile(r"^(ADR-\d{3}(?:-[AE]\d+)?)-.+\.md$")
        errs = []
        for name in sorted(os.listdir(d)):
            full = os.path.join(d, name)
            m = name_re.match(name)
            if not os.path.isfile(full) or not m or name.endswith("-explained.md"):
                continue
            rel = "docs/adr/" + name
            try:
                head, body = split_front_matter(read_text(full))
            except ValueError as e:
                errs.append("%s: %s" % (rel, e))
                continue
            meta = adr_meta(head)
            ident = meta.get("id") or m.group(1)
            if ident != m.group(1):
                errs.append("%s: front matter id %r differs from the file name's %r" % (rel, ident, m.group(1)))
                continue
            if not meta.get("status", "").strip():
                errs.append("%s: ADR record %s has no status" % (rel, ident))
                continue
            rec = {"id": ident, "title": meta.get("title", ""), "status": short_status(meta["status"]),
                   "date": meta.get("date", ""), "path": rel}
            self.adrs[ident] = rec
            for inl in inline_records(ident, rel, rec, body):
                if inl["id"] in self.adrs:
                    errs.append("%s: inline record %s duplicates another record" % (rel, inl["id"]))
                else:
                    self.adrs[inl["id"]] = inl
        self.adr_list = [self.adrs[k] for k in sorted(self.adrs)]
        if errs:
            raise ValueError("\n".join(errs))

    def load_stories(self):
        p = self.path(HANDBOOK, "src", "stories", "stories.yaml")
        if not os.path.exists(p):
            return
        try:
            doc = parse_yaml(read_text(p))
        except ValueError as e:
            raise ValueError("stories.yaml: %s" % e)
        if not isinstance(doc, dict) or set(doc) - {"stories"}:
            raise ValueError("stories.yaml: the only top-level key is stories")
        items = doc.get("stories", [])
        if items == "":
            items = []
        allowed = {"id", "issue", "title", "status", "verdict_comment", "bruno", "evidence"}
        errs = []
        for s in items:
            if not isinstance(s, dict):
                errs.append("stories.yaml: an entry is not a mapping")
                continue
            unknown = set(s) - allowed
            sid = s.get("id", "")
            if unknown:
                errs.append("stories.yaml: story %s has unknown keys %s" % (sid, sorted(unknown)))
            elif not sid:
                errs.append("stories.yaml: an entry has no id")
            elif sid in self.stories:
                errs.append("stories.yaml: duplicate story %s" % sid)
            elif s.get("status") not in ("delivered", "in-progress", "planned"):
                errs.append("stories.yaml: story %s has status %r, want delivered, in-progress or planned" % (sid, s.get("status")))
            else:
                self.stories[sid] = s
                self.story_list.append(s)
                for ev in s.get("evidence", []) or []:
                    if not ev or os.path.isabs(ev) or ".." in ev.split("/"):
                        errs.append("stories.yaml: story %s evidence %r is not a path inside the repository" % (sid, ev))
                    elif not os.path.exists(self.path(*ev.split("/"))):
                        errs.append("stories.yaml: story %s evidence %r does not exist" % (sid, ev))
        if errs:
            raise ValueError("\n".join(errs))

    # OpenAPI: the bundle is generated (tools/openapibundle), so its layout is fixed:
    # paths at two spaces, methods at four, operation keys at six.
    def load_operations(self):
        lines = read_text(self.path("api", "openapi.bundled.yaml")).split("\n")
        in_paths, path, op, in_resp = False, None, None, False
        methods = ("get", "put", "post", "delete", "patch", "head", "options")

        def finish():
            if op and op.get("id"):
                op["responses"].sort()
                self.operations[op["id"]] = op
                self.op_list.append(op)

        for line in lines:
            if line.startswith("paths:"):
                in_paths = True
                continue
            if not in_paths:
                continue
            if line and not line.startswith(" "):
                break
            m = re.match(r"^  (\S.*?):\s*$", line)
            if m:
                finish()
                op, in_resp = None, False
                path = _unquote(m.group(1))
                continue
            m = re.match(r"^    (%s):\s*$" % "|".join(methods), line)
            if m:
                finish()
                op = {"id": "", "method": m.group(1).upper(), "path": path, "tag": path_tag(path),
                      "summary": "", "responses": []}
                in_resp = False
                continue
            if op is None:
                continue
            m = re.match(r"^      operationId:\s*(.+?)\s*$", line)
            if m:
                op["id"] = _unquote(m.group(1))
                continue
            m = re.match(r"^      summary:\s*(.*?)\s*$", line)
            if m:
                op["summary"] = _unquote(m.group(1))
                continue
            if re.match(r"^      responses:\s*$", line):
                in_resp = True
                continue
            if in_resp:
                m = re.match(r"^        [\"']?([0-9A-Za-z]+)[\"']?:", line)
                if m:
                    op["responses"].append(m.group(1))
                elif line.strip() and not line.startswith("         ") and not line.startswith("        "):
                    in_resp = False
        finish()
        self.op_list.sort(key=lambda o: (o["path"], o["method"]))

    def load_schema(self):
        d = self.path(HANDBOOK, "src", "database")
        p = os.path.join(d, "schema.json")
        if not os.path.exists(p):
            raise ValueError("docs/handbook/src/database/schema.json: missing (a stub "
                             '{"version": 1, "tables": []} is accepted until TICK-733 lands)')
        try:
            doc = json.loads(read_text(p))
        except ValueError as e:
            raise ValueError("schema.json: %s" % e)
        if doc.get("version") != 1:
            raise ValueError("schema.json: version %r, this generator reads version 1" % doc.get("version"))
        for n, t in enumerate(doc.get("tables", [])):
            if not isinstance(t, dict) or not t.get("name"):
                raise ValueError("schema.json: tables[%d] has no name" % n)
            self.tables[t["name"]] = t
            self.table_list.append(t)
        self.table_list.sort(key=lambda t: t["name"])
        ap = os.path.join(d, "areas.yaml")
        if not os.path.exists(ap):
            return
        try:
            areas = parse_yaml(read_text(ap))
        except ValueError as e:
            raise ValueError("areas.yaml: %s" % e)
        errs = []
        for a, names in (areas or {}).items():
            if not isinstance(names, list):
                errs.append("areas.yaml: area %s must be a list of table names" % a)
                continue
            self.areas[a] = names
            for n in names:
                if n not in self.tables:
                    errs.append("areas.yaml: area %s names table %r, which is not in schema.json" % (a, n))
        if errs:
            raise ValueError("\n".join(sorted(errs)))

    def load_invariants(self):
        text = read_text(self.path("docs", "adr", "ADR-001-platform-blueprint.md"))
        inside = False
        row = re.compile(r"^\|\s*\*\*(I\d+)\*\*\s*\|\s*\*\*([^*]+)\*\*")
        for line in text.split("\n"):
            if line.startswith("## "):
                inside = line.startswith("## 1.")
                continue
            m = row.match(line) if inside else None
            if m:
                self.invariants[m.group(1)] = m.group(2).strip().rstrip(".")
        if not self.invariants:
            raise ValueError("ADR-001 section 1: no invariant rows (| **I1** | **Name.** ...) found")

    def load_tests(self):
        decl = re.compile(r"^func (Test|Benchmark|Fuzz|Example)\w*\(", re.M)
        name = re.compile(r"^func (\w+)\(", re.M)
        for dirpath, dirs, files in os.walk(self.root):
            dirs[:] = [x for x in dirs if x not in (".git", "node_modules", "vendor", "scratch")]
            rel = os.path.relpath(dirpath, self.root).replace(os.sep, "/")
            for f in files:
                if not f.endswith("_test.go"):
                    continue
                try:
                    with open(os.path.join(dirpath, f), "rb") as fh:
                        text = fh.read().decode("utf-8", "replace")
                except OSError:
                    continue
                for m in decl.finditer(text):
                    n = name.match(text, m.start()).group(1)
                    self.tests.add(n)
                    self.tests.add(rel + "." + n)


def path_tag(p):
    """The spec carries no tags, so the first path segment after /api/v1/ is the tag."""
    p = p[len("/api/v1/"):] if p.startswith("/api/v1/") else p
    return p.lstrip("/").split("/")[0]


def adr_meta(head):
    out = {}
    for line in head.split("\n"):
        m = re.match(r"^([a-z_]+):[ \t]*(.*)$", line)
        if m:
            v = m.group(2).strip()
            if len(v) >= 2 and v[0] in "\"'" and v[-1] == v[0]:
                v = v[1:-1]
            out[m.group(1)] = v
    return out


def short_status(s):
    s = s.strip()
    i = s.find(" (")
    return (s[:i] if i > 0 else s).strip()


def inline_records(parent, rel, parent_rec, body):
    """### E1 (date): title headings under ## Errata or ## Amendments. An inline
    record without a Status: line of its own shows its parent's status."""
    out, section, cur, text = [], False, None, []

    def flush():
        nonlocal cur, text
        if cur is not None:
            m = re.search(r"(?im)^\**\s*status\s*:\**\s*([A-Za-z][A-Za-z -]*[A-Za-z])", "\n".join(text))
            if m:
                cur["status"] = short_status(m.group(1))
            out.append(cur)
        cur, text = None, []

    for line in body.split("\n"):
        if line.startswith("## "):
            flush()
            section = line[3:].strip() in ("Errata", "Amendments")
        elif section and line.startswith("### "):
            flush()
            m = re.match(r"^###\s+([AE]\d+)\b\s*(?:\(([^)]*)\))?\s*:?\s*(.*)$", line)
            if m:
                cur = {"id": parent + "-" + m.group(1), "title": m.group(3).strip(), "date": m.group(2) or "",
                       "path": rel, "status": parent_rec["status"] + " (inherited)"}
        elif cur is not None:
            text.append(line)
    flush()
    return out


# -------------------------------------------------------------- references

def check_refs(f, p):
    out = []
    fm = p.front

    def bad(kind, ref, why):
        out.append("%s: %s %r %s" % (p.src, kind, ref, why))

    for r in fm["adrs"]:
        if r not in f.adrs:
            bad("adrs", r, "is not an ADR, amendment or erratum under docs/adr")
    for r in fm["stories"]:
        if r not in f.stories:
            bad("stories", r, "is not in docs/handbook/src/stories/stories.yaml")
    for r in fm["operations"]:
        if r not in f.operations:
            bad("operations", r, "is not an operationId in api/openapi.bundled.yaml")
    for r in fm["tables"]:
        if r not in f.tables:
            bad("tables", r, "is not a table in docs/handbook/src/database/schema.json")
    for kind, getter in (("workflows", f.registry.workflows), ("activities", f.registry.activities)):
        refs = fm[kind]
        if not refs:
            continue
        names = getter()
        for r in refs:
            if names is None:
                bad(kind, r, "cannot be checked: the registry hook is not wired yet (TICK-727)")
            elif r not in names:
                bad(kind, r, "is not in the registry")
    for r in fm["tests"]:
        if r not in f.tests:
            bad("tests", r, "is not a Test, Benchmark, Fuzz or Example function in any _test.go file")
    for r in fm["packages"]:
        if not r or os.path.isabs(r) or ".." in r.split("/"):
            bad("packages", r, "is not a path inside the repository")
        elif not os.path.isdir(f.path(*r.split("/"))):
            bad("packages", r, "is not a directory in the repository")
    for r in fm["invariants"]:
        if r not in f.invariants:
            bad("invariants", r, "is not an invariant of the ADR-001 section 1 table")
    for r in fm["evidence"]:
        check_path(f, r, "evidence", bad)
    for r in fm["sources"]:
        r2 = r[:-3] if r.endswith("...") else r
        check_path(f, r2.rstrip("/"), "sources", bad)
    if fm["normative"]:
        check_path(f, fm["normative"], "normative", bad)
    return out


def check_path(f, r, kind, bad):
    if not r or os.path.isabs(r) or ".." in r.split("/"):
        bad(kind, r, "is not a path inside the repository")
    elif not os.path.exists(f.path(*r.split("/"))):
        bad(kind, r, "does not exist in the repository")


def backlinks(p, pages):
    kinds = [("Table", "tables"), ("Workflow", "workflows"), ("Story", "stories"),
             ("ADR", "adrs"), ("Operation", "operations"), ("Invariant", "invariants")]
    out = []
    for label, key in kinds:
        for name in sorted(p.front[key]):
            others = [q for q in pages if q is not p and name in q.front[key]]
            if others:
                out.append(("%s %s" % (label, name), others))
    return out


# ------------------------------------------------------------ gen: blocks

GEN_LINE = re.compile(r"^<!--\s*gen:([a-z-]+)((?:\s+[a-z]+=\S+)*)\s*-->\s*$")


def expand_gen(f, body):
    problems, out, fenced = [], [], False
    for line in body.split("\n"):
        if line.startswith("```"):
            fenced = not fenced
        if fenced or not line.strip().startswith("<!-- gen:"):
            out.append(line)
            continue
        m = GEN_LINE.match(line.strip())
        if not m:
            problems.append("malformed generated block %r" % line.strip())
            continue
        args = dict(kv.partition("=")[::2] for kv in m.group(2).split())
        try:
            out.append(gen_block(f, m.group(1), args))
        except ValueError as e:
            problems.append("gen:%s: %s" % (m.group(1), e))
    return "\n".join(out), problems


def gen_block(f, kind, args):
    allow = {"operations": ("tag", "ids"), "tables": ("area",), "er": ("area",),
             "migrations": (), "adr-index": (), "budgets": ()}
    if kind not in allow:
        raise ValueError("unknown block kind")
    for k in args:
        if k not in allow[kind]:
            raise ValueError("unknown argument %r" % k)
    if kind == "operations":
        return gen_operations(f, args)
    if kind == "tables":
        return gen_tables(f, args.get("area", ""))
    if kind == "er":
        return gen_er(f, args.get("area", ""))
    if kind == "migrations":
        return gen_migrations(f)
    if kind == "adr-index":
        return gen_adr_index(f)
    return gen_budgets(f)


def cell(s):
    return str(s).replace("\r", "").replace("\n", " ").replace("|", "\\|").strip()


def gen_operations(f, args):
    tag, ids = args.get("tag", ""), args.get("ids", "")
    if tag and ids:
        raise ValueError("give tag or ids, not both")
    if tag:
        ops = [o for o in f.op_list if o["tag"] == tag]
        if not ops:
            raise ValueError("no operation has tag %r" % tag)
    elif ids:
        ops = []
        for i in ids.split(","):
            if i not in f.operations:
                raise ValueError("operationId %r is not in api/openapi.bundled.yaml" % i)
            ops.append(f.operations[i])
    else:
        raise ValueError("needs tag=<tag> or ids=a,b")
    rows = ["| Operation | Method and path | Summary | Responses |", "|---|---|---|---|"]
    for o in ops:
        rows.append("| `%s` | `%s %s` | %s | %s |" % (o["id"], o["method"], cell(o["path"]), cell(o["summary"]),
                                                      ", ".join(o["responses"])))
    return "\n".join(rows) + "\n"


def area_tables(f, area):
    if not area:
        raise ValueError("needs area=<area>")
    if area not in f.areas:
        raise ValueError("area %r is not in docs/handbook/src/database/areas.yaml" % area)
    return [f.tables[n] for n in sorted(f.areas[area])]


def md_list(items):
    return ", ".join("`%s`" % i for i in items) if items else "none"


def code_or_dash(s):
    return "-" if not str(s).strip() else "`%s`" % cell(s).replace("`", "'")


def gen_tables(f, area):
    out = []
    for t in area_tables(f, area):
        out.append("#### `%s`\n" % t["name"])
        kind = t.get("kind", "")
        if t.get("partition_of"):
            kind += " of " + t["partition_of"]
        rls = t.get("rls", {})
        part = ""
        if t.get("partition_by"):
            part += " Partitioned by `%s`." % cell(t["partition_by"]).replace("`", "'")
        if t.get("partition_bound"):
            part += " Partition bound: `%s`." % cell(t["partition_bound"]).replace("`", "'")
        out.append("Kind: %s. Row-level security: enabled %s, forced %s. Primary key: %s.%s\n" % (
            cell(kind), str(bool(rls.get("enabled"))).lower(), str(bool(rls.get("forced"))).lower(),
            md_list(t.get("primary_key", [])), part))
        out.append("| Column | Type | Null | Default |\n|---|---|---|---|")
        for c in t.get("columns", []):
            out.append("| `%s` | `%s` | %s | %s |" % (c["name"], cell(c["type"]), "yes" if c.get("nullable") else "no",
                                                      code_or_dash(c.get("default", ""))))
        out.append("")
        if t.get("indexes"):
            out.append("| Index | Method | Columns | Include | Predicate |\n|---|---|---|---|---|")
            for i in t["indexes"]:
                name = "`%s`%s" % (i["name"], " (unique)" if i.get("unique") else "")
                inc = md_list(i["include"]) if i.get("include") else "-"
                out.append("| %s | %s | %s | %s | %s |" % (name, cell(i.get("method", "")), md_list(i.get("columns", [])),
                                                          inc, code_or_dash(i.get("predicate", ""))))
            out.append("")
        if t.get("policies"):
            out.append("| Policy | Type | Command | Roles | Using | Check |\n|---|---|---|---|---|---|")
            for p in t["policies"]:
                # A RESTRICTIVE policy narrows what the permissive ones grant, so it must stand out.
                ptype = "permissive" if p.get("permissive", True) else "**RESTRICTIVE**"
                out.append("| `%s` | %s | %s | %s | %s | %s |" % (p["name"], ptype, cell(p.get("command", "")),
                                                                  md_list(p.get("roles", [])),
                                                              code_or_dash(p.get("using", "")),
                                                              code_or_dash(p.get("check", ""))))
            out.append("")
    return "\n".join(out) + "\n"


def gen_er(f, area):
    tables = area_tables(f, area)
    names = {t["name"] for t in tables}
    out = ["```mermaid", "erDiagram"]
    for t in tables:
        out.append("    %s {" % t["name"])
        pk = set(t.get("primary_key", []))
        for c in t.get("columns", []):
            typ = re.sub(r"[ (),\"]", "_", c["type"])
            out.append("        %s %s%s" % (typ, c["name"], " PK" if c["name"] in pk else ""))
        out.append("    }")
    for t in tables:
        for fk in t.get("foreign_keys", []):
            if fk["ref_table"] in names:
                out.append('    %s }o--|| %s : "%s"' % (t["name"], fk["ref_table"], ",".join(fk["columns"])))
    out.append("```")
    return "\n".join(out) + "\n"


def gen_migrations(f):
    d = f.path("db", "migrations")
    rows = ["| Migration | Purpose |", "|---|---|"]
    for name in sorted(os.listdir(d)):
        if not name.endswith(".sql"):
            continue
        first = ""
        for line in read_text(os.path.join(d, name)).split("\n"):
            if line.startswith("--") and "+goose" not in line:
                first = line[2:].strip()
                break
        rows.append("| `%s` | %s |" % (name, cell(first)))
    return "\n".join(rows) + "\n"


def gen_adr_index(f):
    rows = ["| Record | Title | Status | Date |", "|---|---|---|---|"]
    for r in f.adr_list:
        rows.append("| `%s` | %s | %s | %s |" % (r["id"], cell(r["title"]), cell(r["status"]), cell(r["date"])))
    return "\n".join(rows) + "\n"


def split_row(line):
    line = line.strip()
    if line.startswith("|"):
        line = line[1:]
    if line.endswith("|") and not line.endswith("\\|"):
        line = line[:-1]
    cols, cur, i = [], [], 0
    while i < len(line):
        if line[i] == "\\" and i + 1 < len(line) and line[i + 1] == "|":
            cur.append("|")
            i += 2
            continue
        if line[i] == "|":
            cols.append("".join(cur).strip())
            cur = []
        else:
            cur.append(line[i])
        i += 1
    cols.append("".join(cur).strip())
    return cols


def gen_budgets(f):
    """Each budget of docs/budgets.md under its section, with its owner; the
    numbers stay in the source."""
    rows = ["| Section | Budget | Owner |", "|---|---|---|"]
    section = ""
    for line in read_text(f.path("docs", "budgets.md")).split("\n"):
        if line.startswith("## "):
            section = line[3:].strip()
            continue
        if not line.startswith("|") or line.startswith("|---") or line.startswith("| Budget"):
            continue
        cols = split_row(line)
        if len(cols) >= 2:
            rows.append("| %s | %s | %s |" % (cell(section), cell(cols[0]), cell(cols[-1])))
    return "\n".join(rows) + "\n"


# ---------------------------------------------------------------- Markdown
# A small CommonMark subset: headings, paragraphs, lists, tables, block quotes,
# fenced code, links, images, code spans, emphasis. Raw HTML in the source is
# never passed through: block HTML is dropped and inline "<" is escaped. The
# only embeds are a mermaid fence and an Excalidraw image, handled below.

class Ctx:
    def __init__(self, facts, page, by_rel, prefix, diagrams):
        self.facts, self.page, self.by_rel = facts, page, by_rel
        self.prefix, self.diagrams = prefix, diagrams
        self.problems, self.mermaid, self.ids = [], False, {}

    def bad(self, msg):
        self.problems.append("%s: %s" % (self.page.src, msg))

    def up_to_root(self):
        return "../.." if self.page.section == HOME else "../../.."

    def repo_path(self, rel):
        p = os.path.normpath(os.path.join(os.path.dirname(self.page.src), rel)).replace(os.sep, "/")
        if p == ".." or p.startswith("../") or os.path.isabs(p):
            return ""
        return p


SCHEME = re.compile(r"^[A-Za-z][A-Za-z0-9+.-]*:")


def check_external(dest):
    """http(s) URLs are plain anchors; every other scheme (javascript:, data:,
    file:, ...) is rejected. Returns True for an allowed external URL, False for
    a relative one."""
    if not SCHEME.match(dest):
        return False
    scheme = urlparse(dest).scheme.lower()
    if scheme in ("http", "https"):
        return True
    raise ValueError("%r: the %s: scheme is not allowed, use a relative link or an https URL" % (dest, scheme))


def link_dest(ctx, dest):
    if dest == "":
        ctx.bad("link with an empty target")
        return "#"
    if dest.startswith("#"):
        return dest
    try:
        if check_external(dest):
            return dest
    except ValueError as e:
        ctx.bad("link: %s" % e)
        return "#"
    file_, _, frag = dest.partition("#")
    frag = "#" + frag if frag else ""
    repo = ctx.repo_path(file_)
    if not repo:
        ctx.bad("link %r leaves the repository" % dest)
        return "#"
    pre = HANDBOOK + "/src/"
    if repo.startswith(pre) and repo.endswith(".md"):
        target = ctx.by_rel.get(repo[len(pre):])
        if target is None:
            ctx.bad("link %r names a handbook page that does not exist" % dest)
            return "#"
        return ctx.prefix + target.out() + frag
    if not os.path.exists(ctx.facts.path(*repo.split("/"))):
        ctx.bad("link %r: %s does not exist in the repository" % (dest, repo))
        return "#"
    return ctx.up_to_root() + "/" + repo + frag


SVG_NS = "http://www.w3.org/2000/svg"
XLINK_NS = "http://www.w3.org/1999/xlink"
# The only elements an embedded diagram may hold, all in the SVG namespace.
SVG_ELEMENTS = frozenset(["svg", "g", "path", "rect", "ellipse", "line", "polyline", "polygon", "text", "tspan",
                          "defs", "style", "symbol", "use", "mask", "clipPath", "metadata"])
SVG_REMOTE_CSS = re.compile(r"(?is)@import|url\(\s*[\"']?\s*(?!#)")


def check_excalidraw_svg(data):
    """An allowlist, not a denylist: the SVG is parsed (entities decoded by the
    parser), every element must be a listed SVG element, no attribute may be an
    on* handler, and href / xlink:href may only point at "#id"."""
    out = []
    if "payload-start" not in data:
        out.append('has no embedded scene: export from Excalidraw with "Embed scene"')
    if re.search(r"(?i)<!DOCTYPE|<!ENTITY", data):
        return out + ["declares a DOCTYPE or entity"]
    try:
        root = ET.fromstring(data)
    except ET.ParseError as e:
        return out + ["is not well-formed XML: %s" % e]
    bad_elements, bad_attrs = set(), set()
    for el in root.iter():
        if not isinstance(el.tag, str):
            continue
        ns, _, local = el.tag[1:].partition("}") if el.tag.startswith("{") else ("", "", el.tag)
        if (ns, local) not in {(SVG_NS, n) for n in SVG_ELEMENTS} and (ns != "" or local not in SVG_ELEMENTS):
            bad_elements.add(el.tag)
        for name, value in el.attrib.items():
            local_attr = name.rpartition("}")[2].lower()
            if local_attr.startswith("on"):
                bad_attrs.add(name)
            elif local_attr == "href" and not value.strip().startswith("#"):
                bad_attrs.add("%s=%s" % (name, value[:40]))
            elif local_attr == "style" and SVG_REMOTE_CSS.search(value):
                bad_attrs.add("style=%s" % value[:40])
        if el.tag.endswith("style") and el.text and SVG_REMOTE_CSS.search(el.text):
            bad_attrs.add("a <style> with @import or url()")
    if bad_elements:
        out.append("contains elements outside the allowed SVG set: %s" % ", ".join(sorted(bad_elements)))
    if bad_attrs:
        out.append("contains an event handler, an external href or a remote style: %s" % ", ".join(sorted(bad_attrs)))
    return out


def render_image(ctx, alt, dest):
    try:
        if check_external(dest):
            ctx.bad("image %r: only images stored in the repository are allowed (nothing is loaded over the network)" % dest)
            return ""
    except ValueError:
        ctx.bad("image %r: only images stored in the repository are allowed (nothing is loaded over the network)" % dest)
        return ""
    if not dest.endswith(".excalidraw.svg"):
        ctx.bad("image %r: embed Excalidraw diagrams as <name>.excalidraw.svg under docs/handbook/src/diagrams; "
                "Mermaid goes in a mermaid fence" % dest)
        return ""
    repo = ctx.repo_path(dest)
    pre = HANDBOOK + "/src/diagrams/"
    if not repo.startswith(pre) or "/" in repo[len(pre):]:
        ctx.bad("image %r must be a file directly under docs/handbook/src/diagrams" % dest)
        return ""
    full = ctx.facts.path(*repo.split("/"))
    if not os.path.isfile(full):
        ctx.bad("image %r does not exist" % dest)
        return ""
    data = read_text(full)
    for p in check_excalidraw_svg(data):
        ctx.bad("image %r: %s" % (dest, p))
    name = repo[len(pre):]
    ctx.diagrams[name] = data.encode("utf-8")
    return ('<figure class="diagram" data-kind="excalidraw"><div class="dz-wrap"><img class="dz-img" src="%sdiagrams/%s" '
            'alt="%s"></div></figure>' % (ctx.prefix, html.escape(name), html.escape(alt)))


def parse_link_tail(s, i):
    """s[i] is just after "]". Parse "(dest "title")"; return (dest, end) or None."""
    if i >= len(s) or s[i] != "(":
        return None
    j = i + 1
    while j < len(s) and s[j] in " \t":
        j += 1
    if j < len(s) and s[j] == "<":
        k = s.find(">", j)
        if k < 0:
            return None
        dest, j = s[j + 1:k], k + 1
    else:
        depth, k = 0, j
        while k < len(s):
            c = s[k]
            if c == "\\" and k + 1 < len(s):
                k += 2
                continue
            if c == "(":
                depth += 1
            elif c == ")":
                if depth == 0:
                    break
                depth -= 1
            elif c in " \t":
                break
            k += 1
        dest, j = s[j:k], k
    while j < len(s) and s[j] in " \t":
        j += 1
    if j < len(s) and s[j] in "\"'":
        q = s[j]
        k = s.find(q, j + 1)
        if k < 0:
            return None
        j = k + 1
        while j < len(s) and s[j] in " \t":
            j += 1
    if j >= len(s) or s[j] != ")":
        return None
    return re.sub(r"\\([!-/:-@\[-`{-~])", r"\1", dest), j + 1


def inline(ctx, text):
    holes = []

    def hole(h):
        holes.append(h)
        return "\x00%d\x00" % (len(holes) - 1)

    out, i, n = [], 0, len(text)
    while i < n:
        c = text[i]
        if c == "\\" and i + 1 < n and text[i + 1] in "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~":
            out.append(hole(html.escape(text[i + 1])))
            i += 2
        elif c == "`":
            m = re.match(r"`+", text[i:])
            run = m.group(0)
            k = text.find(run, i + len(run))
            while k >= 0 and text[k + len(run):k + len(run) + 1] == "`":
                k = text.find(run, k + len(run) + 1)
            if k < 0:
                out.append(run)
                i += len(run)
            else:
                code = text[i + len(run):k].replace("\n", " ")
                if len(code) > 2 and code[0] == " " and code[-1] == " " and code.strip():
                    code = code[1:-1]
                out.append(hole("<code>%s</code>" % html.escape(code)))
                i = k + len(run)
        elif c == "!" and text[i + 1:i + 2] == "[" or c == "[":
            is_img = c == "!"
            start = i + (2 if is_img else 1)
            depth, k = 1, start
            while k < n and depth:
                if text[k] == "\\":
                    k += 2
                    continue
                if text[k] == "[":
                    depth += 1
                elif text[k] == "]":
                    depth -= 1
                k += 1
            tail = parse_link_tail(text, k) if depth == 0 else None
            if tail is None:
                out.append(c)
                i += 1
                continue
            label, (dest, end) = text[start:k - 1], tail
            if is_img:
                alt = re.sub(r"[*_`]", "", label)
                out.append(hole(render_image(ctx, alt, dest)))
            else:
                out.append(hole('<a href="%s">%s</a>' % (html.escape(link_dest(ctx, dest)), inline(ctx, label))))
            i = end
        elif c == "<":
            m = re.match(r"<([A-Za-z][A-Za-z0-9+.-]*:[^\s<>]*)>", text[i:])
            if m:
                url = m.group(1)
                try:
                    check_external(url)
                    out.append(hole('<a href="%s">%s</a>' % (html.escape(url), html.escape(url))))
                except ValueError as e:
                    ctx.bad("link: %s" % e)
                i += m.end()
            else:
                out.append("&lt;")
                i += 1
        else:
            out.append(html.escape(c, quote=False))
            i += 1
    s = "".join(out)
    s = re.sub(r"\*\*(?=\S)(.+?)(?<=\S)\*\*", r"<strong>\1</strong>", s)
    s = re.sub(r"(?<![\w])__(?=\S)(.+?)(?<=\S)__(?![\w])", r"<strong>\1</strong>", s)
    s = re.sub(r"(?<![\w*])\*(?=[^\s*])(.+?)(?<=[^\s*])\*(?![\w*])", r"<em>\1</em>", s)
    s = re.sub(r"(?<![\w])_(?=[^\s_])(.+?)(?<=[^\s_])_(?![\w])", r"<em>\1</em>", s)
    s = re.sub(r"~~(?=\S)(.+?)(?<=\S)~~", r"<del>\1</del>", s)
    return re.sub("\x00(\\d+)\x00", lambda m: holes[int(m.group(1))], s)


def plain_inline(text):
    return re.sub(r"[*_`~]|\[|\]\([^)]*\)", "", text)


def heading_id(ctx, text):
    base = re.sub(r"[^\w\s-]", "", plain_inline(text).lower()).strip()
    base = re.sub(r"\s+", "-", base) or "section"
    n = ctx.ids.get(base, 0)
    ctx.ids[base] = n + 1
    return base if n == 0 else "%s-%d" % (base, n)


FENCE = re.compile(r"^( {0,3})(`{3,}|~{3,})\s*([^`\s]*)[^`]*$")
HEADING = re.compile(r"^ {0,3}(#{1,6})\s+(.*?)(?:\s+#+)?\s*$")
HR = re.compile(r"^ {0,3}([-*_])(?:\s*\1){2,}\s*$")
LIST_ITEM = re.compile(r"^( *)([-*+]|\d{1,9}[.)])( +|$)(.*)$")
TABLE_DELIM = re.compile(r"^\s*\|?\s*:?-{1,}:?\s*(\|\s*:?-{1,}:?\s*)*\|?\s*$")


def render_blocks(ctx, lines):
    out, i, n = [], 0, len(lines)

    def starts_block(k):
        s = lines[k]
        return bool(FENCE.match(s) or HEADING.match(s) or HR.match(s) or s.lstrip().startswith(">")
                    or (LIST_ITEM.match(s) and LIST_ITEM.match(s).group(4).strip()))

    while i < n:
        line = lines[i]
        if not line.strip():
            i += 1
            continue
        m = FENCE.match(line)
        if m:
            ind, fence, lang = len(m.group(1)), m.group(2), m.group(3)
            body, i = [], i + 1
            while i < n and not (lines[i].strip().startswith(fence[0] * len(fence))
                                 and set(lines[i].strip()) == {fence[0]}):
                body.append(lines[i][ind:] if lines[i][:ind].strip() == "" else lines[i])
                i += 1
            i += 1
            code = "\n".join(body) + ("\n" if body else "")
            if lang == "mermaid":
                ctx.mermaid = True
                out.append('<figure class="diagram" data-kind="mermaid"><div class="dz-wrap"><pre class="mermaid-src">%s'
                           '</pre></div></figure>' % html.escape(code))
            else:
                cls = ' class="language-%s"' % html.escape(lang) if lang else ""
                out.append("<pre><code%s>%s</code></pre>" % (cls, html.escape(code)))
            continue
        m = HEADING.match(line)
        if m:
            level, text = len(m.group(1)), m.group(2)
            out.append('<h%d id="%s">%s</h%d>' % (level, html.escape(heading_id(ctx, text)), inline(ctx, text), level))
            i += 1
            continue
        if HR.match(line):
            out.append("<hr>")
            i += 1
            continue
        if line.lstrip().startswith(">"):
            quote = []
            while i < n and lines[i].lstrip().startswith(">"):
                quote.append(re.sub(r"^\s*> ?", "", lines[i]))
                i += 1
            out.append("<blockquote>\n%s\n</blockquote>" % render_blocks(ctx, quote))
            continue
        if "|" in line and i + 1 < n and TABLE_DELIM.match(lines[i + 1]) and "-" in lines[i + 1]:
            head = split_row(line)
            if len(head) == len(split_row(lines[i + 1])):
                aligns = []
                for d in split_row(lines[i + 1]):
                    aligns.append("left" if d.startswith(":") and not d.endswith(":") else
                                  "right" if d.endswith(":") and not d.startswith(":") else
                                  "center" if d.startswith(":") and d.endswith(":") else "")
                i += 2
                rows = []
                while i < n and lines[i].strip() and "|" in lines[i]:
                    rows.append(split_row(lines[i]))
                    i += 1

                def td(tag, k, text):
                    a = ' align="%s"' % aligns[k] if k < len(aligns) and aligns[k] else ""
                    return "<%s%s>%s</%s>" % (tag, a, inline(ctx, text), tag)

                t = ["<table>", "<thead>", "<tr>"] + [td("th", k, c) for k, c in enumerate(head)] + ["</tr>", "</thead>"]
                if rows:
                    t.append("<tbody>")
                    for r in rows:
                        r = (r + [""] * len(head))[:len(head)]
                        t += ["<tr>"] + [td("td", k, c) for k, c in enumerate(r)] + ["</tr>"]
                    t.append("</tbody>")
                t.append("</table>")
                out.append("\n".join(t))
                continue
        m = LIST_ITEM.match(line)
        if m and (m.group(4).strip() or m.group(3) == ""):
            i = render_list(ctx, lines, i, out)
            continue
        if line.lstrip().startswith("<") and re.match(r"^\s*<(/?[A-Za-z][A-Za-z0-9-]*(\s|/?>|$)|!--)", line):
            while i < n and lines[i].strip():
                i += 1
            out.append("<!-- raw HTML omitted -->")
            continue
        para = [line.strip()]
        i += 1
        while i < n and lines[i].strip() and not starts_block(i) and not (
                "|" in lines[i] and i + 1 < n and TABLE_DELIM.match(lines[i + 1])):
            para.append(lines[i].strip())
            i += 1
        rendered = inline(ctx, "\n".join(para))
        if rendered.startswith("<figure") and rendered.endswith("</figure>") and rendered.count("<figure") == 1:
            out.append(rendered)  # a block element must not sit inside <p>
        else:
            out.append("<p>%s</p>" % rendered)
    return "\n".join(out)


def render_list(ctx, lines, i, out):
    n = len(lines)
    m = LIST_ITEM.match(lines[i])
    base, ordered = len(m.group(1)), m.group(2)[0].isdigit()
    start = int(re.match(r"\d+", m.group(2)).group(0)) if ordered else 1
    items, loose = [], False
    while i < n:
        m = LIST_ITEM.match(lines[i])
        if not m or len(m.group(1)) != base or m.group(2)[0].isdigit() != ordered:
            break
        content_indent = len(m.group(1)) + len(m.group(2)) + max(1, len(m.group(3)))
        body = [m.group(4)]
        i += 1
        blank = False
        while i < n:
            s = lines[i]
            if not s.strip():
                blank = True
                body.append("")
                i += 1
                continue
            indent = len(s) - len(s.lstrip(" "))
            nxt = LIST_ITEM.match(s)
            if indent >= content_indent:
                body.append(s[content_indent:])
            elif nxt and len(nxt.group(1)) <= indent and indent < content_indent:
                break
            elif not blank and not (FENCE.match(s) or HEADING.match(s) or HR.match(s)):
                body.append(s.strip())  # lazy continuation of the paragraph
            else:
                break
            blank = False
            i += 1
        while body and not body[-1].strip():
            body.pop()
        if "" in body[:-1] and any(b.strip() for b in body[body.index("") + 1:]) and \
                not any(LIST_ITEM.match(b) for b in body[body.index("") + 1:body.index("") + 2]):
            loose = True
        items.append(body)
        j = i
        while j < n and not lines[j].strip():
            j += 1
        if j < n and j != i:
            nm = LIST_ITEM.match(lines[j])
            if nm and len(nm.group(1)) == base and nm.group(2)[0].isdigit() == ordered:
                loose = True
                i = j
            else:
                break
    tag = "ol" if ordered else "ul"
    attr = ' start="%d"' % start if ordered and start != 1 else ""
    parts = ["<%s%s>" % (tag, attr)]
    for body in items:
        inner = render_blocks(ctx, body)
        if not loose:
            inner = re.sub(r"^<p>(.*?)</p>(\n|$)", r"\1\2", inner, count=1, flags=re.S) if inner.startswith("<p>") else inner
        parts.append("<li>%s</li>" % inner)
    parts.append("</%s>" % tag)
    out.append("\n".join(parts))
    return i


def plain_text(body_html):
    s = re.sub(r'(?s)<figure class="diagram" data-kind="mermaid">.*?</figure>', " ", body_html)
    s = re.sub(r"(?s)<[^>]*>", " ", s)
    return " ".join(html.unescape(s).split())


# -------------------------------------------------------------- page layout

def nav_for(pages):
    nav = []
    for s in SECTIONS:
        index, items = "", []
        for p in pages:
            if p.section != s:
                continue
            if p.slug == "index":
                index = p.out()
            else:
                items.append((p.slug, p.front["title"], p.out()))
        items.sort()
        if index or items:
            nav.append((s, index, items))
    return nav


def esc(s):
    return html.escape(s, quote=True)


def render_page(p, pages, nav, body, has_mermaid, stamp):
    prefix = "" if p.section == HOME else "../"
    o = ["<!DOCTYPE html>", '<html lang="en">', "<head>", '<meta charset="utf-8">',
         "<meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'self'; script-src 'self'; "
         "style-src 'self' 'unsafe-inline'; img-src 'self'; object-src 'none'\">",
         '<meta name="viewport" content="width=device-width, initial-scale=1">',
         "<title>%s - Nava handbook</title>" % esc(p.front["title"]),
         '<link rel="stylesheet" href="%sassets/handbook.css">' % prefix, "</head>",
         '<body data-root="%s">' % prefix, '<aside class="sidebar">',
         '<a class="brand" href="%sindex.html">Nava handbook</a>' % prefix,
         '<div class="search"><input id="hb-q" type="search" placeholder="Search the handbook" autocomplete="off" '
         'aria-label="Search"><ul id="hb-results" hidden></ul></div>', '<nav aria-label="Sections">']
    for name, index, items in nav:
        o.append('<div class="nav-section"><a class="nav-title" href="%s">%s</a>' % (
            esc(prefix + index) if index else "#", esc(name)))
        li = "".join('<li%s><a href="%s">%s</a></li>' % (' class="current"' if href == p.out() else "",
                                                          esc(prefix + href), esc(title)) for _s, title, href in items)
        o.append("<ul>%s</ul></div>" % li)
    o += ["</nav>", "</aside>", "<main>"]
    if p.section != HOME:
        o.append('<nav class="crumbs" aria-label="Breadcrumb"><a href="%sindex.html">Handbook</a> / '
                 '<a href="%s%s/index.html">%s</a> / <span>%s</span></nav>' % (
                     prefix, prefix, p.section, esc(p.section), esc(p.front["title"])))
    if p.front["normative"]:
        href = prefix + "../../" + p.front["normative"]
        o.append('<p class="banner">Summary. The normative text is the linked <a href="%s"><code>%s</code></a>.</p>' % (
            esc(href), esc(p.front["normative"])))
    if p.front["status"] == "in-progress":
        o.append('<p class="badge">In progress: not on main yet</p>')
    o += ["<article>", '<h1 class="page-title">%s</h1>' % esc(p.front["title"]), body, "</article>"]
    bl = backlinks(p, pages)
    if bl:
        o.append('<section class="backlinks"><h2>Also named by</h2>')
        for label, others in bl:
            o.append("<h3>%s</h3><ul>%s</ul>" % (esc(label), "".join(
                '<li><a href="%s">%s</a></li>' % (esc(prefix + q.out()), esc(q.front["title"])) for q in others)))
        o.append("</section>")
    o += ['<footer><p class="hb-footer">%s</p></footer>' % esc(stamp), "</main>",
          '<script src="%sassets/search-index.js"></script>' % prefix]
    if has_mermaid:
        o.append('<script src="%sassets/mermaid.min.js"></script>' % prefix)
    o += ['<script src="%sassets/handbook.js"></script>' % prefix, "</body>", "</html>", ""]
    return "\n".join(o)


def search_index_js(entries):
    """A script, not a JSON file: fetch() is blocked on file://. The text can never
    close the script element, open "<!--" or break a JS string."""
    data = json.dumps(entries, ensure_ascii=False, separators=(",", ":"))
    for ch, rep in (("<", "\\u003c"), (">", "\\u003e"), ("&", "\\u0026"),
                    ("\u2028", "\\u2028"), ("\u2029", "\\u2029")):
        data = data.replace(ch, rep)
    return ("window.HB_SEARCH = " + data + ";\n").encode("utf-8")


def short_sha(sha):
    return sha[:12]


SHA_RE = re.compile(r"^[0-9a-f]{7,40}$")
DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")


def build(root, sha, date, registry=None):
    """Render the whole handbook in memory: {path relative to docs/handbook: bytes}.
    Every problem is collected and raised together, each naming page and reference."""
    problems = []
    if not SHA_RE.match(sha or ""):
        problems.append("stamp sha %r is not a 7-40 digit lower-case hex commit" % sha)
    if not DATE_RE.match(date or ""):
        problems.append("stamp date %r is not YYYY-MM-DD" % date)
    facts = Facts(root, registry or Unwired())
    problems += facts.problems
    pages, pp = load_pages(root)
    problems += pp
    by_rel = {p.src_rel(): p for p in pages}
    if "index.md" not in by_rel:
        problems.append("docs/handbook/src/index.md: the home page is missing")
    nav = nav_for(pages)
    stamp = "Content generated against main @ %s (%s)" % (short_sha(sha or ""), date)
    out, diagrams, entries = {}, {}, []
    for p in pages:
        problems += check_refs(facts, p)
        prefix = "" if p.section == HOME else "../"
        expanded, gp = expand_gen(facts, p.body)
        problems += ["%s: %s" % (p.src, g) for g in gp]
        ctx = Ctx(facts, p, by_rel, prefix, diagrams)
        body = render_blocks(ctx, expanded.split("\n"))
        problems += ctx.problems
        out[p.out()] = render_page(p, pages, nav, body, ctx.mermaid, stamp).encode("utf-8")
        entries.append({"url": p.out(), "title": p.front["title"], "section": p.section, "text": plain_text(body)})
    if problems:
        raise BuildError(sorted(problems))
    for name, data in diagrams.items():
        out["diagrams/" + name] = data
    out["assets/search-index.js"] = search_index_js(entries)
    for a in ("handbook.css", "handbook.js"):
        with open(os.path.join(HERE, "assets", a), "rb") as f:
            out["assets/" + a] = f.read().replace(b"\r\n", b"\n")
    mermaid = os.path.join(root, HANDBOOK, "assets", "mermaid.min.js")
    if not os.path.exists(mermaid):
        raise BuildError(["docs/handbook/assets/mermaid.min.js: the pinned Mermaid is missing"])
    with open(mermaid, "rb") as f:
        out["assets/mermaid.min.js"] = f.read()
    problems = offline_violations(out)
    if problems:
        raise BuildError(problems)
    return out


# ------------------------------------------------------------ offline check
# The handbook must open from file:// with no network request. Plain <a href>
# to GitHub is allowed: an anchor loads nothing. The pinned mermaid.min.js is a
# vendored bundle holding http(s) strings (XML namespaces, upstream doc links)
# that are never fetched; it is skipped, and the page CSP (script-src 'self')
# stops it loading more. search-index.js is page text as data (a page may quote
# a URL or fetch() in prose), so it is skipped too.

LOADING_TAG = re.compile(r"(?is)<(script|img|link|iframe|frame|embed|object|source|video|audio|track|input|image|use|meta)\b[^>]*>")
REMOTE_ATTR = re.compile(r"(?is)\b(?:src|href|data|srcset|poster|xlink:href|action|formaction)\s*=\s*[\"']?\s*(?:https?:)?//")
REMOTE_REFRESH = re.compile(r"(?is)http-equiv\s*=\s*[\"']?refresh")
CSS_IMPORT = re.compile(r"(?is)@import\b")
CSS_URL = re.compile(r"(?is)url\(\s*[\"']?\s*(?:https?:)?//")
JS_NETWORK = re.compile(r"\b(?:fetch|importScripts|sendBeacon)\s*\(|XMLHttpRequest|new\s+(?:WebSocket|EventSource)\b")
JS_REMOTE = re.compile(r"""(?i)["'`]\s*(?:https?:)?//[a-z0-9]""")


def offline_file_violations(name, data):
    s = data.decode("utf-8", "replace") if isinstance(data, bytes) else data
    out = []
    if name.endswith(".html"):
        for m in LOADING_TAG.finditer(s):
            if REMOTE_ATTR.search(m.group(0)) or REMOTE_REFRESH.search(m.group(0)):
                out.append("%s: remote load in %s" % (name, m.group(0)))
        out += ["%s: %s" % (name, m.group(0)) for m in CSS_IMPORT.finditer(s)]
        out += ["%s: %s" % (name, m.group(0)) for m in CSS_URL.finditer(s)]
        for want in ("default-src 'self'", "script-src 'self'", "object-src 'none'", "style-src 'self'"):
            if want not in s:
                out.append("%s: the page lacks the CSP %s" % (name, want))
        if "<script>" in s:
            out.append("%s: has an inline script" % name)
    elif name.endswith(".css"):
        out += ["%s: %s" % (name, m.group(0)) for m in CSS_IMPORT.finditer(s)]
        out += ["%s: %s" % (name, m.group(0)) for m in CSS_URL.finditer(s)]
    elif name.endswith(".js"):
        out += ["%s: %s" % (name, m.group(0)) for m in JS_NETWORK.finditer(s)]
        out += ["%s: remote string %s" % (name, m.group(0)) for m in JS_REMOTE.finditer(s)]
    return out


def offline_violations(output):
    out = []
    for name in sorted(output):
        if name in ("assets/mermaid.min.js", "assets/search-index.js"):
            continue  # vendored bundle; page text as data
        out += offline_file_violations(name, output[name])
    return out


# ----------------------------------------------------------------- writing

def generated_files(base):
    found = set()
    for dirpath, dirs, files in os.walk(base):
        rel_dir = os.path.relpath(dirpath, base).replace(os.sep, "/")
        if rel_dir == ".":
            dirs[:] = [d for d in dirs if d not in ("src", "build")]
        for f in files:
            rel = f if rel_dir == "." else rel_dir + "/" + f
            if rel != "README.md":
                found.add(rel)
    return found


def is_generated_name(rel):
    """Only these paths are the build's own: section pages, index.html, the three
    generated assets and copied diagrams. Anything else under docs/handbook is
    never deleted by a build; --check reports it instead."""
    if rel in ("index.html", "assets/handbook.css", "assets/handbook.js", "assets/search-index.js"):
        return True
    parts = rel.split("/")
    if len(parts) == 2 and parts[0] in SECTIONS and parts[1].endswith(".html"):
        return True
    return len(parts) == 2 and parts[0] == "diagrams" and parts[1].endswith(".svg")


def write_output(root, output):
    base = os.path.join(root, HANDBOOK)
    for rel in sorted(generated_files(base)):
        if rel in output or not is_generated_name(rel):
            if rel not in output and rel != "assets/mermaid.min.js":
                print("handbook: leaving %s/%s alone: the build does not own it" % (HANDBOOK, rel), file=sys.stderr)
            continue
        os.remove(os.path.join(base, *rel.split("/")))
    for rel, data in output.items():
        p = os.path.join(base, *rel.split("/"))
        os.makedirs(os.path.dirname(p), exist_ok=True)
        if os.path.exists(p):
            with open(p, "rb") as f:
                if f.read() == data:
                    continue
        with open(p, "wb") as f:
            f.write(data)


STAMP_RE = re.compile(rb"Content generated against main @ [^<]*")
STAMP_SHA_RE = re.compile(rb"Content generated against main @ ([0-9a-f]{7,40}) \(")
PIN_RE = re.compile(r"(?m)^\| `mermaid`[^\n]*sha256 `([0-9a-f]{64})`")


def normalise(data):
    """Drop the footer's sha and date, so a moving main never makes the pages stale."""
    return STAMP_RE.sub(b"Content generated against main @ STAMP", data)


def no_rules(rel, data):
    """The sensitivity-scanner hook (HB-17, TICK-735): one finding string per
    problem. The shipped rule set is empty."""
    return []


class Git:
    def __init__(self, root):
        self.root = root

    def run(self, *args):
        return subprocess.run(["git"] + list(args), cwd=self.root, capture_output=True, text=True, check=True).stdout.strip()

    def is_ancestor(self, sha):
        r = subprocess.run(["git", "merge-base", "--is-ancestor", sha, "HEAD"], cwd=self.root,
                           capture_output=True, text=True)
        if r.returncode in (0, 1):
            return r.returncode == 0
        raise RuntimeError("git merge-base --is-ancestor: %s (CI lint needs fetch-depth: 0)" % r.stderr.strip())


def check(root, git=None, scanner=no_rules, registry=None):
    """Rebuild in memory and compare with the committed files. Returns problems."""
    out = build(root, "0000000", "1970-01-01", registry)
    base = os.path.join(root, HANDBOOK)
    have = generated_files(base)
    problems = []
    for rel in sorted(out):
        p = os.path.join(base, *rel.split("/"))
        if not os.path.exists(p):
            problems.append("%s/%s is missing; run `task handbook` and commit it" % (HANDBOOK, rel))
        else:
            with open(p, "rb") as f:
                if normalise(f.read()) != normalise(out[rel]):
                    problems.append("%s/%s is stale; run `task handbook` and commit it" % (HANDBOOK, rel))
        have.discard(rel)
    for rel in sorted(have):
        problems.append("%s/%s is not generated by the build; remove it" % (HANDBOOK, rel))
    problems += check_mermaid_pin(root)
    problems += check_stamp(base, git)
    if scanner:
        scanned = [(rel, out[rel]) for rel in sorted(out)]
        for extra in ("src/database/schema.json",):
            p = os.path.join(base, *extra.split("/"))
            if os.path.exists(p):
                with open(p, "rb") as f:
                    scanned.append((extra, f.read()))
        d = os.path.join(base, "src", "diagrams")
        if os.path.isdir(d):
            for name in sorted(os.listdir(d)):
                if name.endswith(".svg"):
                    with open(os.path.join(d, name), "rb") as f:
                        scanned.append(("src/diagrams/" + name, f.read()))
        for rel, data in scanned:
            for finding in scanner(rel, data):
                problems.append("sensitivity scan: %s: %s" % (rel, finding))
    return problems


def check_mermaid_pin(root):
    try:
        pins = read_text(os.path.join(root, "docs", "pins.md"))
    except OSError as e:
        return ["docs/pins.md: %s" % e]
    m = PIN_RE.search(pins)
    if not m:
        return ["docs/pins.md has no `mermaid` row with a sha256"]
    with open(os.path.join(root, HANDBOOK, "assets", "mermaid.min.js"), "rb") as f:
        got = hashlib.sha256(f.read()).hexdigest()
    if got != m.group(1):
        return ["docs/handbook/assets/mermaid.min.js has sha256 %s, docs/pins.md pins %s" % (got, m.group(1))]
    return []


def check_stamp(base, git):
    p = os.path.join(base, "index.html")
    if not os.path.exists(p):
        return []
    with open(p, "rb") as f:
        m = STAMP_SHA_RE.search(f.read())
    if not m:
        return ["docs/handbook/index.html has no footer stamp"]
    if git is None:
        return []
    sha = m.group(1).decode()
    try:
        ok = git.is_ancestor(sha)
    except (RuntimeError, OSError) as e:
        return ["footer sha %s: %s" % (sha, e)]
    if not ok:
        return ["footer sha %s is not an ancestor of HEAD; rebuild with `task handbook` on a branch that contains it" % sha]
    return []


# --------------------------------------------------------------------- CLI

def main(argv=None):
    ap = argparse.ArgumentParser(description="Build the product handbook (docs/handbook).")
    ap.add_argument("--root", default=ROOT, help="repository root")
    ap.add_argument("--sha", default=os.environ.get("HANDBOOK_SHA", ""),
                    help="main commit the content is generated against (default $HANDBOOK_SHA, else git merge-base HEAD origin/main)")
    ap.add_argument("--date", default=os.environ.get("HANDBOOK_DATE", ""),
                    help="date of that commit, YYYY-MM-DD (default $HANDBOOK_DATE, else the commit's date)")
    ap.add_argument("--check", action="store_true", help="fail when the committed handbook is stale")
    args = ap.parse_args(argv)
    root = os.path.abspath(args.root)
    git = Git(root)
    try:
        if args.check:
            problems = check(root, git)
            for p in problems:
                print("handbook:", p, file=sys.stderr)
            if problems:
                return 1
            print("handbook: ok")
            return 0
        sha, date = args.sha, args.date
        if not sha:
            try:
                sha = git.run("merge-base", "HEAD", "origin/main")
            except (subprocess.CalledProcessError, OSError) as e:
                print("handbook: cannot find the main commit (git merge-base HEAD origin/main failed); "
                      "pass --sha or set HANDBOOK_SHA: %s" % e, file=sys.stderr)
                return 2
        if not date:
            try:
                date = git.run("show", "-s", "--format=%cs", sha)
            except (subprocess.CalledProcessError, OSError) as e:
                print("handbook: cannot read the date of %s; pass --date or set HANDBOOK_DATE: %s" % (sha, e), file=sys.stderr)
                return 2
        output = build(root, sha, date)
        write_output(root, output)
        print("handbook: wrote %d files under docs/handbook" % len(output))
        return 0
    except BuildError as e:
        print("handbook:", e, file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
