---
name: design-docs
description: "How to write RFCs, ADRs and PLANs that a reviewer can read in fifteen minutes — structure, decision tables, diagrams that render, and what never goes in a decision record. Use whenever you write or revise an RFC, ADR, PLAN, design review turn or architecture summary, or draw a Mermaid diagram for one."
---

# Design document standards

**The reader:** one human reviewer with about fifteen minutes per document, reading it as a rendered HTML page. Every document leads with the recommendation and the decisions the human must make, then the detail that supports them. If a section does not change a decision, cut it.

**RFC (the proposal and its debate)**
1. Summary — 3–6 bullets: what we decide, the main trade-off, what the human must accept.
2. Decisions table — `| # | Decision | Why / what it prevents |`, one row per binding choice, numbered (D1, D2…) so reviews can cite them.
3. Options considered — each with the reason it lost. A rejected option without a reason will be proposed again.
4. Risks and accepted residuals — each with an owner and a revisit trigger.
5. Test/acceptance spec — the tests that prove the decisions (CI-1, CI-2…), each with what it catches.
6. Open questions — only ones the human or a named role must answer.

**ADR (the decision, after the RFC is approved)** — at most two pages: decision, context it rests on, consequences, rejected options, recorded dissent, revisit triggers. Status stays `PROPOSED` until the Solution Architect's verdict (after the Technical Architect critique and any CTO tie-break); the Solution Architect never sets `ACCEPTED` on a contested design without that verdict. No debate history (it stays in the archived RFC).

**PLAN** — summary, critical path, sequencing with reasons, what the milestone cannot absorb and the proposed cut, then the task graph. Each task: one outcome, acceptance criteria, the interface contract it must honour, dependencies, a role tag. When the plan splits work into stories, each story lists the files and packages it touches, and no two stories touch the same file; a real dependency is written `Depends on: <story>`, and that story starts only after the one it depends on is merged into `main`.

**Where they live and how they ship** — paths, branch name and PR target are in the `design-triad` skill ("Where design artefacts live", "Branch and PR").

**Writing**
- Claims that rest on measurement cite the spike or file that measured them; unverified claims say "unverified" and how to verify. A confident wrong number costs more than an honest "unknown".
- Name concrete things: tables, columns, endpoints, error codes, flags. "Handle errors properly" is not a decision.
- Link documents as wikilinks (`[[ADR-001-…]]`); tasks link the ADR only.

**Mermaid diagrams (they must render — a broken diagram hides the decision)**
- `sequenceDiagram`: never put `;` in a message — it ends the statement and breaks the parse. Write "BEGIN, then set_config(…)" not "BEGIN; set_config(…)". Avoid `#` too (it starts an entity code).
- `flowchart`: quote labels that contain `()`, `[]`, `{}`, `:`, `|`, `<`, `>` or `#`: `A["validate (2020-12)"]`. Use `<br/>` for line breaks inside quoted labels.
- Prefer several small diagrams to one big one; each diagram should answer one question (who calls whom, in what order, what states exist).
- Before finishing, re-read every diagram line for these characters.
