"""Builds the acceptance guide site (TICK-287) into docs/acceptance/guide/.

    python -I docs/acceptance/guide/build/build.py

Writes the HTML pages, guide-standalone.html (every page in one file), the
Bruno collection (bruno/) and ticketing-acceptance-guide.zip, and fails when an
operation or documented status in api/openapi.bundled.yaml has no Bruno
request (or exemption in coverage-exempt.json). Every page is self-contained:
inline CSS, inline SVG flowcharts, no script, no network. Edit the content here
and in collection.py, never the output files.
"""

import html
import json
import os
import sys
import zipfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import collection  # noqa: E402
from content import PAGES, PREREQ, TROUBLE, ROUTES, NOT_ON_MAIN  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.dirname(HERE)
ROOT = os.path.dirname(os.path.dirname(os.path.dirname(OUT)))
SPEC = os.path.join(ROOT, "api", "openapi.bundled.yaml")
EXEMPT = os.path.join(HERE, "coverage-exempt.json")
ZIP = "ticketing-acceptance-guide.zip"

e = html.escape

CSS = """
:root{--ink:#1d2433;--mute:#5b6577;--line:#d8dde6;--bg:#f6f7f9;--card:#fff;--acc:#2f5bd3;--ok:#1d7a46;--bad:#b42318;--warn:#8a5a00}
*{box-sizing:border-box}body{margin:0;font:15px/1.55 system-ui,-apple-system,Segoe UI,Roboto,sans-serif;color:var(--ink);background:var(--bg)}
header{background:#14213d;color:#fff;padding:14px 24px}header a{color:#fff;text-decoration:none;font-weight:600}
nav{display:flex;flex-wrap:wrap;gap:4px;padding:8px 24px;background:#fff;border-bottom:1px solid var(--line);position:sticky;top:0;z-index:2}
nav a{padding:4px 10px;border-radius:6px;color:var(--ink);text-decoration:none;font-size:14px}nav a:hover,nav a.on{background:#e8eefc;color:var(--acc)}
main{max-width:1100px;margin:0 auto;padding:24px}
section.page{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:24px 28px;margin-bottom:28px}
h1{font-size:26px;margin:0 0 6px}h2{font-size:19px;margin:28px 0 10px;border-bottom:1px solid var(--line);padding-bottom:4px}h3{font-size:16px;margin:18px 0 6px}
.sub{color:var(--mute);margin:0 0 16px}code,pre{font:13px/1.45 ui-monospace,SFMono-Regular,Consolas,monospace}
code{background:#eef1f5;padding:1px 5px;border-radius:4px}pre{background:#0f172a;color:#e2e8f0;padding:12px 14px;border-radius:8px;overflow:auto}pre code{background:none;padding:0;color:inherit}
table{border-collapse:collapse;width:100%;margin:8px 0 16px;font-size:14px}th,td{border:1px solid var(--line);padding:6px 8px;text-align:left;vertical-align:top}th{background:#f0f3f8}
ol.steps{padding-left:0;list-style:none;counter-reset:s}ol.steps>li{counter-increment:s;border:1px solid var(--line);border-radius:8px;padding:10px 14px 10px 52px;margin:10px 0;position:relative}
ol.steps>li:before{content:counter(s);position:absolute;left:14px;top:10px;width:26px;height:26px;border-radius:50%;background:var(--acc);color:#fff;text-align:center;font-weight:700;line-height:26px}
.do{font-weight:600}.exp{color:var(--ok)}.exp:before{content:"Expect: ";font-weight:700}.fail{color:var(--bad)}.fail:before{content:"Failed if: ";font-weight:700}
.where{display:inline-block;font-size:12px;padding:1px 7px;border-radius:10px;background:#eef1f5;color:var(--mute);margin-left:6px}
.card-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(250px,1fr));gap:12px}.card{border:1px solid var(--line);border-radius:8px;padding:12px 14px;background:#fff;text-decoration:none;color:var(--ink)}.card:hover{border-color:var(--acc)}
.card b{display:block;margin-bottom:4px}.btn{display:inline-block;background:var(--acc);color:#fff;padding:10px 18px;border-radius:8px;text-decoration:none;font-weight:600;margin:4px 8px 4px 0}.btn.alt{background:#fff;color:var(--acc);border:1px solid var(--acc)}
.note{border-left:4px solid var(--warn);background:#fff8e6;padding:8px 12px;border-radius:4px;margin:10px 0}.diagram{overflow:auto;border:1px solid var(--line);border-radius:8px;background:#fbfcfe;padding:8px}
.legend{font-size:13px;color:var(--mute)}footer{color:var(--mute);font-size:13px;text-align:center;padding:20px}
@media print{nav,header{display:none}section.page{page-break-after:always;border:0}}
"""

LANE_COLORS = {
    "Tester": "#14213d", "Bruno": "#f4aa41", "Shell": "#334155", "Browser": "#7c3aed",
    "nava-core": "#2f5bd3", "nava-onboard": "#0e7490", "Keycloak": "#b45309", "Postgres": "#1d4ed8",
    "nava-groupsync": "#047857", "Temporal": "#6d28d9", "nava-relay": "#be185d", "Temporal UI": "#6d28d9",
    "Docker": "#0369a1",
}


def svg_flow(title, lanes, msgs):
    """A swimlane flow: one column per actor/service, numbered arrows top to
    bottom. msgs: (from, to, label, kind) with kind req|ok|fail|note."""
    colw, top, rowh = 172, 58, 40
    w = colw * len(lanes) + 20
    h = top + rowh * len(msgs) + 30
    x = {l: 10 + colw * i + colw // 2 for i, l in enumerate(lanes)}
    out = [f'<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="{e(title)}" width="{w}" height="{h}" viewBox="0 0 {w} {h}" font-family="system-ui,Segoe UI,sans-serif" font-size="12">',
           '<defs>' + ''.join(
               f'<marker id="m{k}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0,0L10,5L0,10z" fill="{c}"/></marker>'
               for k, c in (("req", "#334155"), ("ok", "#1d7a46"), ("fail", "#b42318"))) + '</defs>']
    for l in lanes:
        c = LANE_COLORS.get(l, "#475569")
        out.append(f'<rect x="{x[l]-64}" y="8" width="128" height="30" rx="6" fill="{c}"/>'
                   f'<text x="{x[l]}" y="27" fill="#fff" text-anchor="middle" font-weight="600">{e(l)}</text>'
                   f'<line x1="{x[l]}" y1="38" x2="{x[l]}" y2="{h-10}" stroke="{c}" stroke-opacity=".35" stroke-dasharray="4 4"/>')
    for i, (a, b, label, kind) in enumerate(msgs):
        y = top + rowh * i + 14
        n = i + 1
        color = {"req": "#334155", "ok": "#1d7a46", "fail": "#b42318", "note": "#8a5a00"}[kind]
        if a == b or kind == "note":
            bw = max(140, 7 * len(label) + 30)
            bx = min(max(x[a] - bw // 2, 4), w - bw - 4)
            fill = {"ok": "#e7f5ec", "fail": "#fdecea"}.get(kind, "#fff8e6")
            out.append(f'<rect x="{bx}" y="{y-13}" width="{bw}" height="24" rx="5" fill="{fill}" stroke="{color}"/>'
                       f'<text x="{bx + bw/2}" y="{y+3}" text-anchor="middle" fill="{color}">{n}. {e(label)}</text>')
            continue
        x1, x2 = x[a], x[b]
        d = 1 if x2 > x1 else -1
        out.append(f'<line x1="{x1 + 3*d}" y1="{y+6}" x2="{x2 - 3*d}" y2="{y+6}" stroke="{color}" stroke-width="1.6"'
                   + (' stroke-dasharray="5 3"' if kind != "req" else "") + f' marker-end="url(#m{kind})"/>'
                   f'<text x="{(x1+x2)/2}" y="{y}" text-anchor="middle" fill="{color}">{n}. {e(label)}</text>')
    out.append("</svg>")
    return "".join(out)


def md(s):
    """Tiny inline markup: `code`, **bold**, [text](href). Escapes the rest."""
    import re
    s = e(s)
    s = re.sub(r"`([^`]+)`", r"<code>\1</code>", s)
    s = re.sub(r"\*\*([^*]+)\*\*", r"<b>\1</b>", s)
    s = re.sub(r"\[([^\]]+)\]\(([^)]+)\)", r'<a href="\2">\1</a>', s)
    return s


def block(item):
    """Content blocks: str paragraph, ('pre', text), ('note', text), ('table', head, rows), ('ul', items), ('h3', text)."""
    if isinstance(item, str):
        return f"<p>{md(item)}</p>"
    kind = item[0]
    if kind == "pre":
        return f"<pre><code>{e(item[1].strip(chr(10)))}</code></pre>"
    if kind == "note":
        return f'<div class="note">{md(item[1])}</div>'
    if kind == "h3":
        return f"<h3>{md(item[1])}</h3>"
    if kind == "ul":
        return "<ul>" + "".join(f"<li>{md(x)}</li>" for x in item[1]) + "</ul>"
    if kind == "table":
        head = "".join(f"<th>{md(h)}</th>" for h in item[1])
        rows = "".join("<tr>" + "".join(f"<td>{md(c)}</td>" for c in r) + "</tr>" for r in item[2])
        return f"<table><thead><tr>{head}</tr></thead><tbody>{rows}</tbody></table>"
    raise ValueError(kind)


def steps(items):
    out = ['<ol class="steps">']
    for s in items:
        where = f'<span class="where">{e(s["where"])}</span>' if s.get("where") else ""
        out.append(f'<li><div class="do">{md(s["do"])}{where}</div>')
        if s.get("pre"):
            out.append(f"<pre><code>{e(s['pre'].strip(chr(10)))}</code></pre>")
        out.append(f'<div class="exp">{md(s["expect"])}</div><div class="fail">{md(s["fail"])}</div></li>')
    out.append("</ol>")
    return "".join(out)


def nav(current, link):
    items = [("index", "Home"), ("prerequisites", "Prerequisites")] + [(p["id"], p["short"]) for p in PAGES] + \
            [("coverage", "Coverage"), ("troubleshooting", "Troubleshooting")]
    return "<nav>" + "".join(
        f'<a href="{link(i)}"{" class=on" if i == current else ""}>{e(t)}</a>' for i, t in items) + "</nav>"


def page_body(p, link):
    out = [f'<h1>{e(p["title"])}</h1><p class="sub">{md(p["story"])}</p>']
    out.append("<h2>What you can test</h2>" + "".join(block(b) for b in p["what"]))
    out.append("<h2>Flowchart</h2>")
    for f in p["flows"]:
        out.append(f'<h3>{e(f["title"])}</h3><div class="diagram">{svg_flow(f["title"], f["lanes"], f["msgs"])}</div>')
    out.append('<p class="legend">Solid arrow: what you trigger. Green dashed: the expected answer. Red dashed: a refusal the test expects. Amber box: something that happens on its own.</p>')
    out.append("<h2>Click by click</h2>" + steps(p["steps"]))
    if p.get("extra"):
        out.append("".join(block(b) for b in p["extra"]))
    out.append("<h2>Bruno</h2>" + "".join(block(b) for b in p["bruno"]))
    return "".join(out)


def index_body(link):
    cards = "".join(f'<a class="card" href="{link(p["id"])}"><b>{e(p["short"])}: {e(p["title"])}</b>{md(p["blurb"])}</a>' for p in PAGES)
    return f"""<h1>TicketingSystem acceptance guide: stories S1–S8 and S11a</h1>
<p class="sub">Everything delivered on <code>main</code> in stories S1 to S8, plus S11a from its story branch (<code>TICK-334/workflow-interpreter</code>), tried by hand on your own machine. Each use case has a flowchart, the exact clicks, what you should see and what a failure looks like. The Bruno collection walks the same steps with automatic checks, and adds a test suite for every API operation.</p>
<p><a class="btn" href="{ZIP}" download>Download the kit (.zip)</a><a class="btn alt" href="guide-standalone.html" download>Download as one HTML file</a></p>
<p class="legend">The zip holds this site, the Bruno collection (with its <code>local</code> environment) and the helper scripts. Unzip it anywhere: the pages work offline from a double-click. The scripts run from the repository checkout, at <code>docs/acceptance/guide/tools/</code>.</p>
<h2>Start here</h2><ol><li><a href="{link('prerequisites')}">Prerequisites</a>: clone, start the stack, seed the two tenants, open the tunnels, open the collection in Bruno.</li>
<li>Pick a use case below, or run the whole collection: in Bruno's Runner, or headless with <code>sh docs/acceptance/guide/tools/api-test.sh</code> (about 3 minutes with the 15-second group sync, plus up to 5 minutes for the last folder).</li>
<li>Something red? <a href="{link('troubleshooting')}">Troubleshooting</a>.</li></ol>
<h2>Use cases</h2><div class="card-grid">{cards}</div>
<h2>What is not on <code>main</code></h2>{''.join(block(b) for b in NOT_ON_MAIN)}
<p class="legend">The <a href="{link('coverage')}">coverage page</a> maps every route on <code>main</code> and every documented API outcome to its Bruno request, or says why it cannot be called from outside.</p>"""


def coverage_body(folders, api_rows):
    reqs = list(collection.requests(folders))
    out = ["<h1>Route coverage</h1><p class='sub'>Every route on <code>main</code> (each <code>Handle</code>/<code>HandleFunc</code> in <code>cmd/</code> and <code>internal/</code>, and every operation in <code>api/openapi.yaml</code>) and where it is tested.</p>",
           block(("table", ["Service and listener", "Route", "Callable from the host?", "Covered by"], ROUTES))]
    out.append("<h2>Every documented API outcome</h2>")
    out.append(block("Each operation in `api/openapi.bundled.yaml` and each status it documents, with the Bruno requests that assert that status. "
                     "`build.py` and the Go test `TestBrunoCoversEveryOperation` (`api/bruno_coverage_test.go`) fail when an operation has no request, "
                     "or a status has neither a request nor a reason in `build/coverage-exempt.json`."))
    out.append(block(("table", ["Operation", "Route", "Status", "Asserted by", "Not asserted because"],
                      [(o, f"`{r}`", s, by, why) for o, r, s, by, why in api_rows])))
    out.append("<h2>Every request in the collection</h2>")
    out.append(block(("table", ["Folder", "Request", "Method", "URL"],
                      [(f, n, m, f"`{u}`") for f, n, m, u in reqs])))
    return "".join(out)


def simple_body(title, sub, blocks):
    return f"<h1>{e(title)}</h1><p class='sub'>{md(sub)}</p>" + "".join(block(b) for b in blocks)


def doc(title, current, body, link):
    return f"""<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{e(title)}: TicketingSystem acceptance</title><style>{CSS}</style></head>
<body><header><a href="{link('index')}">TicketingSystem · acceptance guide S1–S8, S11a</a></header>{nav(current, link)}
<main>{body}</main><footer>Generated by docs/acceptance/guide/build/build.py (TICK-287). Dev-only credentials only.</footer></body></html>
"""


# Every zip entry gets the same timestamp and mode, so a rebuild from the same
# sources writes the same members and CI's drift check (TICK-391) sees no
# change from the clock or the checkout's file modes.
ZIP_TIME = (1980, 1, 1, 0, 0, 0)


def _read(path):
    with open(path, "rb") as f:
        return f.read()


def _add(z, arcname, data):
    info = zipfile.ZipInfo(arcname, date_time=ZIP_TIME)
    info.compress_type = zipfile.ZIP_DEFLATED
    info.external_attr = 0o644 << 16
    info.create_system = 3  # Unix, whatever host builds it
    z.writestr(info, data)


def main():
    with open(SPEC, encoding="utf-8") as f:
        spec = f.read()
    with open(EXEMPT, encoding="utf-8") as f:
        exempt = json.load(f)
    folders = collection.write(os.path.join(OUT, "bruno"))
    api_rows = collection.coverage(folders, spec, exempt)

    bodies = [("index", "Home", lambda link: index_body(link)),
              ("prerequisites", "Prerequisites", lambda link: simple_body(*PREREQ)),
              *[(p["id"], p["short"], (lambda p: lambda link: page_body(p, link))(p)) for p in PAGES],
              ("coverage", "Coverage", lambda link: coverage_body(folders, api_rows)),
              ("troubleshooting", "Troubleshooting", lambda link: simple_body(*TROUBLE))]

    files = lambda i: f"{i}.html"
    written = []
    for pid, title, body in bodies:
        name = files(pid)
        with open(os.path.join(OUT, name), "w", encoding="utf-8", newline="\n") as f:
            f.write(doc(title, pid, body(files), files))
        written.append(name)

    anchors = lambda i: f"#{i}"
    sections = "".join(f'<section class="page" id="{pid}">{body(anchors)}</section>' for pid, _, body in bodies)
    standalone = doc("All use cases", "", sections, anchors).replace(
        f'<a class="btn" href="{ZIP}" download>', f'<a class="btn" href="{ZIP}" download title="Needs the zip beside this file">')
    with open(os.path.join(OUT, "guide-standalone.html"), "w", encoding="utf-8", newline="\n") as f:
        f.write(standalone)
    written.append("guide-standalone.html")

    with zipfile.ZipFile(os.path.join(OUT, ZIP), "w", zipfile.ZIP_DEFLATED) as z:
        for name in written:
            with open(os.path.join(OUT, name), encoding="utf-8") as f:
                page = f.read()
            # The zip cannot hold itself: inside it, the download button says so.
            page = page.replace(f'href="{ZIP}" download', 'href="index.html" title="This folder is the kit"').replace(
                "Download the kit (.zip)", "You have the kit: this folder")
            _add(z, f"ticketing-acceptance-guide/{name}", page.encode("utf-8"))
        for sub in ("bruno", "tools"):
            for d, dirs, names in sorted(os.walk(os.path.join(OUT, sub))):
                dirs.sort()
                for name in sorted(names):
                    path = os.path.join(d, name)
                    rel = os.path.relpath(path, OUT).replace(os.sep, "/")
                    _add(z, f"ticketing-acceptance-guide/{rel}", _read(path))
        for name in ("README.md", "acceptance-run.md"):
            if os.path.exists(os.path.join(OUT, name)):
                _add(z, f"ticketing-acceptance-guide/{name}", _read(os.path.join(OUT, name)))
    print("wrote", ", ".join(written), "bruno/", ZIP)


if __name__ == "__main__":
    main()
