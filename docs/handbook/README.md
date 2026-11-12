# Product handbook

Pages for engineers: what the platform is, how it is built, where each part
lives. The handbook describes **only what is on `main`**; a page about work
still on a story branch is marked `status: in-progress`. The repository is
public: no secrets, tokens, internal hostnames or personal data in any page.

Open `docs/handbook/index.html` straight from a clone (`file://`). Nothing is
fetched from the network: no CDN, no font, no `fetch()`.

## Edit and rebuild

1. Edit or add `docs/handbook/src/<section>/<slug>.md`. Sections: `vision`,
   `system`, `adr`, `stories`, `workflows`, `database`, `security`,
   `benchmarks`, `operations`, `glossary`. `src/index.md` is the home page.
2. Run `task handbook` (`python -I docs/handbook/build/build.py`, Python 3,
   standard library only, from the repository root). It writes
   `docs/handbook/**/*.html` and `docs/handbook/assets/{handbook.css,handbook.js,search-index.js}`.
3. Commit the sources and the generated files together.
4. `task handbook:check` (`... build.py --check`) rebuilds in memory and fails
   when the committed HTML is stale. It is not part of `task lint` or CI.
5. `python -I docs/handbook/build/test_build.py` runs the generator's tests.

Never edit the generated HTML. The viewer (`handbook.css`, `handbook.js`) is
edited in `docs/handbook/build/assets/`; `task handbook` copies it to
`docs/handbook/assets/`.

### The footer stamp

Every page ends with `Content generated against main @ <sha> (<date>)`. The
build takes the stamp from `-sha` / `-date`, else `$HANDBOOK_SHA` /
`$HANDBOOK_DATE`, else `git merge-base HEAD origin/main` and that commit's
date. `handbook:check` ignores the stamp when it diffs, so a merge to `main`
never makes the handbook stale; it separately requires the stamped sha to be
an ancestor of `HEAD` (`git merge-base --is-ancestor`, full history needed).

## Front matter

Every page starts with YAML. Unknown keys fail the build, and so does every
reference that does not resolve (the error names the page and the reference).

```yaml
---
title: "Audit log"
section: stories
status: delivered                 # delivered | in-progress
sources: [docs/adr/ADR-004-audit-log.md, internal/audit/..., internal/relay/dispatch.go:29]  # files (or file:line) this page was checked against; must exist
adrs: [ADR-004, ADR-004-E7]       # ADR, amendment or erratum IDs under docs/adr
stories: [S14]                    # ids in src/stories/stories.yaml
operations: [listAuditEntries]    # operationIds in api/openapi.bundled.yaml
tables: [audit_log]               # tables in src/database/schema.json
workflows: [audit.seal]           # kinds in internal/outbox/outbox.go (parsed, TICK-727)
activities: [audit.seal.Seal]     # names the seal, interp and workerguard Register functions register
build: production                 # workflows pages: production | interptest (which build registers it)
tests: [TestChainSeals]           # Test, Benchmark, Fuzz or Example functions in any _test.go file (or dir.Name)
packages: [internal/audit]        # directories
invariants: [I1, I6]              # rows of the ADR-001 section 1 table
evidence: [docs/acceptance/x.md]  # files that must exist
normative: docs/adr/ADR-004-audit-log.md  # adds the "Summary. The normative text is the linked .md." banner
---
```

Every other page that names the same table, workflow, story, ADR, operation or
invariant is listed at the foot of the page ("Also named by").

`workflows:` and `activities:` resolve against `GoRegistry` in
`docs/handbook/build/build.py`, which parses the kind list in
`internal/outbox/outbox.go` and the `Register` functions of `seal`, `interp`
and `workerguard` with the standard library. It fails the build when it finds
nothing, so a parser that drifted from the code cannot pass. A `sources:`
entry may name lines (`path:12`, `path:12-20,31`); the build checks the file
has them, and the reviewer checks what they say against the footer sha.

## Diagrams

- Mermaid: a fenced block with the language `mermaid`. Rendered lazily when it
  scrolls into view, with `securityLevel: 'strict'`. The source stays in the
  page as the print and no-script fallback.
- Excalidraw: export with **Embed scene** to
  `docs/handbook/src/diagrams/<name>.excalidraw.svg` and write
  `![alt text](../diagrams/<name>.excalidraw.svg)`. It is shown as an `<img>`
  (never inline SVG). The build refuses an SVG with no embedded scene, a script,
  an event handler, or a remote reference.

Every diagram gets zoom (wheel while the pointer is over it, or + and -), pan
(drag), reset and full screen from the viewer. Authors do nothing extra.

## Generated blocks

Write one of these on a line of its own; the build replaces it. Never write
the content by hand.

| Block | Content | From |
|---|---|---|
| `<!-- gen:operations tag=audit -->` | operations whose first path segment after `/api/v1/` is the tag | `api/openapi.bundled.yaml` |
| `<!-- gen:operations ids=a,b -->` | those operationIds | same |
| `<!-- gen:tables area=audit -->` | columns, indexes and policies of the area's tables | `src/database/schema.json`, `src/database/areas.yaml` |
| `<!-- gen:er area=audit -->` | a Mermaid `erDiagram` of the area | same |
| `<!-- gen:migrations -->` | every migration with its first comment line | `db/migrations` |
| `<!-- gen:adr-index -->` | every ADR, amendment and erratum with status and date | `docs/adr` |
| `<!-- gen:budgets -->` | each budget with its owner; the numbers stay in the source | `docs/budgets.md` |

The ADR index reads `docs/adr/ADR-NNN(-[AE]N)?-*.md` plus `### [AE]N` headings
under a parent's `## Errata` or `## Amendments`. It skips `*-explained.md`,
`*.html` and `ADR-NNN/` folders. A record without a status fails the build; an
inline erratum with no `Status:` line of its own shows its parent's status as
`<status> (inherited)`. Never print a count of records in prose.

`src/database/schema.json` is written by the integration suite (TICK-733); its
format (`version: 1`) is documented in `src/database/schema.md`. Until it
lands, the stub `{"version": 1, "tables": []}` is accepted. Tables are named
by their bare name (schema `public` only). Any `version` other than 1 fails
the build; unknown keys are ignored. `src/database/areas.yaml` maps an area
to its table names.

## Content security policy

Every page carries `default-src 'self'; script-src 'self'; style-src 'self'
'unsafe-inline'; img-src 'self'; object-src 'none'`. `'unsafe-inline'` is for
styles only: Mermaid writes inline styles into its SVG, and `style-src 'self'`
alone leaves diagrams unstyled. Scripts stay `'self'` with no inline script.
This deviation from a plain `style-src 'self'` was accepted on TICK-719.

## Story manifest

`src/stories/stories.yaml`: `stories:` is a list with `id` (S1...), `issue`
(TICK-nn), `title`, `status` (`delivered`, `in-progress`, `planned`),
`verdict_comment` (link text only), `bruno` (folder under
`docs/acceptance/guide/bruno/`) and `evidence` (repository paths, which must
exist).

## Pinned Mermaid

`assets/mermaid.min.js` is committed, not fetched; its version and sha256 are
in `docs/pins.md`. To bump it, replace the file and the pin row together.

## Not hand-edited

`index.html`, `<section>/*.html`, `diagrams/`, `assets/handbook.css`,
`assets/handbook.js` and `assets/search-index.js` are generated.
