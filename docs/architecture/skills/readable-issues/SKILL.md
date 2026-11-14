---
name: readable-issues
description: "Plain-language header for stories, ADRs and bugs, the bug template, and the two Goldfish checks (comprehension, implementation readiness) adapted from Google's Elephant-Goldfish Model. Use whenever you write or groom a story, feature, ADR, amendment, erratum or bug, run grooming, or are about to release a story from backlog, give an ADR verdict, or move a bug to todo."
---

# Readable issues: plain-language header and Goldfish checks

Adapted from Google's Elephant-Goldfish Model (EGM, Apache-2.0, https://github.com/google/elephant-goldfish-model): `agents/comprehension.md` and `agents/implementation_readiness.md`, Review mode. The author (the Elephant) knows everything; a reviewer with none of that context (the Goldfish) must be able to restate the document. If the Goldfish cannot, the document is unclear, however precise it is. Our reviewers (SA, TA, TL) share the author's context, so they do not catch this; the Goldfish does.

## 1. Plain-language header (first in every story, ADR, amendment, erratum and bug)

The header comes before everything else in the body, including `Depends on:` and plan references. It contains **no internal codes** (no PLAN-, ADR-, §, D#, M#, C#, R#, P#, S#, CI-#, TICK- numbers); each code is explained on the last line instead.

```markdown
**What changes:** <one sentence, in words a new hire understands>
**Why:** <the problem today, and who feels it>
**What you'll see when it's done:** <an observable result: a screen, a response, a number, a message>
**Codes used below:** <one line per code the body uses: `ADR-005` = the record-numbering decision. `M1–M3` = its three database migration steps.>
```

- If nothing visible changes, say so, then say what would go wrong without the change ("nothing visible; ticket numbers keep counting from where they were").
- "What you'll see" is something a person can check, not "it works correctly".
- Below the header the agent-facing detail stays as precise as before (the `delivery-planning`, `design-docs` and bug sections define it).
- Leaf tasks do not need the header; they get the readiness check at grooming (§4).

Worked example (S17, record numbering):
> **What changes:** nothing visible. Ticket numbers keep counting from where they were after the upgrade.
> **Why:** we're changing how the counter table is keyed so number formats can reset per period later. A bad migration could restart numbering or hand out duplicate numbers.
> **What you'll see when it's done:** after the upgrade, the last incident before it was `INC0000042`, the next one is `INC0000043`, and 8 users creating tickets at once never get the same number.
> **Codes used below:** ADR-005 = the record-numbering decision. R1/R2 = first and second release of the change. M1–M3 = its three database migration steps.

## 2. Bug issues

Title: `Bug: <observable symptom>` (e.g. `Bug: second publish of a number format returns 500`). Body, in this order:

1. The plain-language header (§1). "What changes" is the fix as the user sees it; "Why" is who hits the bug and how often.
2. `Depends on:` line.
3. **Steps to reproduce** — commands or requests a fresh agent can paste, with the seeded user and the build/commit.
4. **Expected** and **Actual** — the exact response, log line or screen state; no paraphrase of an error.
5. **Scope** — tenants, types or paths affected; whether data was written wrongly (and needs repair) or only refused.
6. **Acceptance criteria** — the regression test that fails today and passes after the fix (red proof per `tdd`), plus any data repair.
7. Role label and owner, per `team-routing`.

A bug goes `backlog` → `todo` only after it passes the comprehension check (§3).

## 3. Comprehension Goldfish (stories, ADRs, bugs)

**When:** before a story leaves `backlog`; before the ADR verdict (the Solution Architect runs it on the ADR `.md` and the plain section of the `.html`); before a bug goes to `todo`. The author of the document runs it.

**How:** start a fresh subagent with your subagent tool (Agent / Task, a new general agent, **not a fork**: a fork inherits your context and defeats the check). Give it only the prompt below with the issue body (or ADR text) pasted in. Do not give it file paths, the repo, the ADR, or the conversation. It costs one subagent call, not a Multica run.

Prompt (paste verbatim, then the document):

```text
You are the Comprehension Goldfish: a fresh reader who has just joined the project. You know software; you do not know this project. Read ONLY the document below. Do not open files, search, or use tools.

1. Paraphrase first, under two headings, 2-5 sentences each:
   **What this proposes**
   **How the system works today, per this text**
   If you cannot paraphrase a section, report it as a Blocker. A wrong-sounding or impossible paraphrase is itself a finding: say where the text led you.
2. **Undefined terms:** list every term, code, service, table or concept that is specific to this project and is neither defined in the document nor linked. Do not list general engineering terms (HTTP, PostgreSQL, RLS, idempotency), except in the plain-language header ("What changes / Why / What you'll see") and in an "In plain words" section: their reader is not an engineer, so list every technical term used there that is not explained in the same sentence.
3. **Vague lines:** quote each line that asserts something without saying how ("the system handles this", "seamless", "robust") or assumes an unstated setup step.
4. For each finding in 2-3 give Severity (Blocker | Major | Nit) and the section it is in.
No preamble, no suggestions beyond the findings.

--- DOCUMENT ---
<paste the issue body or ADR text here>
```

**Judge it (author):**
- Compare the paraphrase with what you meant. **Wrong or impossible paraphrase = Blocker**: rewrite the text (usually the header) and run the check again with a new fresh subagent. A story or bug does not leave `backlog`; an ADR does not get its verdict.
- Every undefined term: define it in "Codes used below" or inline, or link it. Every vague line: say how.
- Nits are optional.
- Post the paraphrase and what you changed as one comment on the issue (`Goldfish check:` heading), so reviewers can see it ran. A reviewer who sees no such comment on a story, ADR or bug sends it back.

## 4. Readiness Goldfish (every leaf task, at grooming)

Replaces the old grooming question "can its owner start without asking a question?". Run it once per task during grooming (`delivery-planning`, `agent-meetings`), same mechanics as §3: fresh subagent, task body only.

```text
You are the Implementation Readiness Goldfish: the engineer who will write this code, with no other context. Read ONLY the task below. Do not open files, search, or use tools.

Ask: "Do I know exactly what to code, where, and how, from this text?"
1. List every design choice this task leaves to the implementer: an API signature, status code, error code, table or column, type, file or package to change, concurrency or retry rule, error-handling strategy, test to write. One line each, quoting the text that leaves it open.
2. List required elements that are missing: acceptance tests that would fail if it broke, the interface contract (endpoint, table, error codes), API test scenarios for each operation (status x operation, invalid input writes nothing, replay writes once, cross-tenant refused), `Depends on:` line, role label.
3. For each item give Severity: Blocker (cannot start, or two engineers would build different things) | Major | Nit.
No preamble.

--- TASK ---
<paste the task body here>
```

Each Blocker is answered in the task body (the decision itself, or a pointer to the ADR section that makes it) before the task leaves `backlog`. A choice the ADR genuinely leaves open goes back to the Solution Architect, not to the implementer.

## 5. Plain English (everything a human reads)

- No ungrounded adjectives: seamless, robust, blazingly fast, scalable, easy, simple, powerful. Replace with the measurement or the mechanism ("p95 under 50 ms at 1,000 rps, measured in spike S2"; "retries 3 times with backoff, then returns 503").
- Never "the system handles this" (or "automatically", "transparently") without saying what does it and how.
- Spell out a code the first time it appears in anything a human reads, or put it in "Codes used below".
