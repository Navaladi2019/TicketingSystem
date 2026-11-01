---
type: rfc
id: RFC-007
title: Retiring recorded interpreter histories when a GetVersion branch is deleted
status: DRAFT (rev 0)
date: 2026-11-01
issue: TICK-388
source: TICK-350 / PR #331, docs/quality/ci-pipeline.md "Deploy rules (c)", marked open
author: Solution Architect
---

# RFC-007: Retiring recorded interpreter histories

> **Invariants:** the project description of *Ticket Management System* is empty, so I could not load the eight invariants from it. I checked against ADR-001 §2 (I1–I9) and R1/CI-7 as written in the repo. Treat the invariant check in §7 as **incomplete** until the project description carries them.

## 1. Decision

A history is retired by **deleting it in the same PR that deletes its `workflow.GetVersion` branch, and appending a tombstone entry to a manifest file that is itself append-only**: `internal/workflow/testdata/retired-histories.json`. `TestHistoriesAppendOnly` accepts a deletion only when a tombstone names the file **and its exact git blob on `main`**, **and** the tombstone's GetVersion change id was present on `main` and is gone from the work tree. All of these are facts in git and in Go source, so CI checks them with no judgment.

The one fact CI can't see is deploy rule (c): no open production run started before the change. That fact is the **approval**. The Operator records the evidence, the Technical Architect reviews the PR, and **the human owner merges it**. Agents never merge a PR that touches the manifest.

**Form:** a **decision record** (a new plan row D-3-25 in `docs/delivery/plan-records-path.md`, plus the `ci-pipeline.md` section). **No ADR-001 amendment.** The append-only rule is not in ADR-001. It lives in P3-23 (D-3-12, D-3-17) and `ci-pipeline.md`, and this RFC changes no invariant, R1/CI-7 or the Temporal allow-list (§7).

## 2. Problem

Deploy rule (c) allows deleting a `GetVersion` branch once no open run that started before it remains. After the deletion, histories recorded before the branch take a code path that no longer exists, so `TestReplayRecordedHistories` fails on them. Today `TestHistoriesAppendOnly` (`internal/workflow/interp/histories_test.go`) refuses every deletion of a file that is on `main`, and `checkHistorySet` refuses a file missing from `wantHistories`. The result is that the branch can never be deleted. If we bolted on an escape hatch with no checks, any red replay could be "fixed" by deleting the history, and the gate would prove nothing.

## 3. Mechanism

### 3.1 The manifest

`internal/workflow/testdata/retired-histories.json` sits **beside** `histories/`, not inside it, so the `*.json` glob of `checkHistorySet` and the replay set are unchanged.

```json
{
  "format": "nava.retired-histories/v1",
  "retired": [
    {
      "history": "else-default.json",
      "blob": "3b18e512dba79e4c8300dd08aeb37f8e728b8dad",
      "change_id": "step-ref-before-timer",
      "issue": "TICK-412"
    }
  ]
}
```

| Field | Meaning | CI check |
|---|---|---|
| `history` | the file name under `histories/` | non-empty, no `/`, ends `.json` |
| `blob` | the git blob id of the file **as on `main`** (`git ls-tree`) | 40 lowercase hex characters, equal to BASE's blob (R-2) |
| `change_id` | the `workflow.GetVersion` change id whose deletion retires it | non-empty string (R-5, R-6) |
| `issue` | the retirement issue that carries the rule (c) evidence | matches `^TICK-[0-9]+$` |

Strict decode: unknown fields fail, and so does a duplicate `history`.

### 3.2 The retirement PR

One PR does all of the following, and nothing else:

1. Deletes the `GetVersion` branch: the call and its `DefaultVersion` arm.
2. Deletes each history that only replays through that arm from `histories/`, and from `wantHistories`.
3. Appends one manifest entry per deleted file.

Which histories go is not a judgment call either. They are exactly the ones `TestReplayRecordedHistories` fails on after step 1.

### 3.3 Rules `TestHistoriesAppendOnly` enforces (BASE = `origin/main`)

| Rule | Check | Fails when |
|---|---|---|
| **R-1** | A file under `histories/` on BASE whose content differs | always. A retirement is a deletion, never an edit |
| **R-2** | A file under `histories/` on BASE that the work tree deleted | no manifest entry has `history` = its name **and** `blob` = its BASE blob |
| **R-3** | BASE's manifest `retired` list is a **prefix** of the work tree's (entries compared field by field) | an entry was edited, reordered or removed. The manifest is append-only too |
| **R-4** | Each entry's `history` is absent from `histories/` | a retired name is reused. A name keeps one meaning forever |
| **R-5** | No `workflow.GetVersion` call in non-test Go under `internal/workflow/` has `change_id` as its change id (go/parser, work tree) | the branch is still there, so the history was retired for nothing |
| **R-6** | For every entry **added by this change** (not on BASE), a `GetVersion` call with that `change_id` exists in BASE's source (go/parser over `git show BASE:<file>`) | the id never existed. This stops a red replay from being "fixed" by inventing a tombstone |
| **R-7** | Every `GetVersion` change id argument in that source is a string literal | the id is a constant or an expression. R-5 and R-6 need a literal to compare. Only literal ids are allowed |

Without `BASE` the test still skips, as today. A laptop's `origin/main` may be stale. The `unit` job always sets BASE. R-4, R-5 and R-7 need no BASE, so they also run on a laptop as `TestRetiredHistoriesManifest`.

Every rule is a comparison of git objects, JSON fields or Go AST literals. There is no clock, no network and no Temporal call, so the test gives the same answer on every run of the same commit pair.

A change id is never reused after its branch is deleted (R-5 runs on every entry, forever). Reuse would be unsafe anyway: an old marker in a stray history would match the new branch.

### 3.4 Proof the check works (tests in the implementing PR)

`TestAppendOnlyCheck` (scratch repository) gains these cases. Each one is red without the rule and green with it:

- a deletion with a matching tombstone passes;
- a deletion whose tombstone has the wrong blob fails (R-2);
- an edited or removed BASE tombstone fails (R-3);
- a retired name re-added fails (R-4);
- a tombstone whose `change_id` is still called fails (R-5);
- a tombstone whose `change_id` never existed on BASE fails (R-6);
- `GetVersion(ctx, id, …)` with a non-literal `id` fails (R-7);
- an edit to a history that has a tombstone still fails (R-1).

## 4. Who approves, and why it is outside CI

Rule (c) is a fact about **production**: an Operator `ListWorkflowExecutions` query (`WorkflowType = 'nava.interpreter' AND ExecutionStatus = 'Running' AND StartTime < '<deploy time of the branch>'`) returns nothing. CI has no production access, and it must not get any (operator-only reach, ADR-001 R1). So:

| Step | Who | Recorded where |
|---|---|---|
| Runs the rule (c) query and posts the query text, the branch's deploy time, the UTC time of the run and the empty result | **Operator** | the issue named in `issue` |
| Reviews the retirement PR against §3.2 and the evidence | **Technical Architect** (owner of deploy rules a–c) | PR review comment |
| Merges the PR | **Human owner (nava) only.** The "MergePr" label goes on the issue and the PR, and agents never merge it, even though it is not a story or an ADR | GitHub merge |

**Why the human owner.** Once a merged retirement deploys, any open old run that the query missed fails its next workflow task. There is no forward fix: deploy rule (b) forbids rolling back across a recorded `GetVersion`, and restoring a deleted branch is a new deploy. That is a production risk call, so a human makes it.

**Timing.** Apply the deploy rules in the order (a), then (c). The evidence must be newer than the branch's deploy. Run the query again if the PR waits more than 24 h, so the evidence is no older than a day when the PR merges.

## 5. Options considered

| Option | Verdict | Why |
|---|---|---|
| **A. Delete + append-only tombstone manifest with blob and change-id checks** | **Chosen** | Every condition can be checked in git and the AST. One small file. Git keeps the deleted content, recoverable by `blob` |
| B. Move the file to `histories/retired/`, which the replay skips | Rejected | It keeps dead fixtures in the tree forever and needs the same tombstone data anyway (which change id, which issue). `git mv` alone shows no reason. The R1 walk would also have to cover a directory nobody replays |
| C. Allow-list as a Go slice in `histories_test.go` | Rejected | Same content as A, but it mixes data into code that is edited for other reasons, and the append-only check over Go source is a text diff, not a structured prefix check |
| D. Re-record the old histories on the new code | Rejected | It breaks the point of the gate. A re-recorded history proves nothing about runs already in flight |
| E. CI enforces approval: `CODEOWNERS` + "require review from code owners" on the manifest | Rejected for now | Agent PRs are opened from the owner's own account (`Navaladi2019`, e.g. PR #331), and GitHub does not count an author's own review, so every retirement PR would be blocked. It is also a repo-wide branch-protection change. Revisit if agents get their own GitHub identity |
| F. CI runs the rule (c) query itself | Rejected | It would give CI production Temporal reach (ADR-001 R1, operator-only UI). The test would also depend on live state, so it would not be deterministic |

## 6. Acceptance criteria (for the Technical Lead)

1. `retired-histories.json` exists with `"retired": []` and the v1 format.
2. `TestHistoriesAppendOnly` implements R-1 to R-6 with BASE. `TestRetiredHistoriesManifest` implements the schema checks, R-4, R-5 and R-7 without BASE.
3. `TestAppendOnlyCheck` covers every case in §3.4, each one shown red then green.
4. `docs/quality/ci-pipeline.md` "Deploy rules (c)" replaces the **Open** note with §3.2 and §4. `make/workflow.mk` and the `wantHistories` comment ("none is ever removed") say "except by a tombstone (RFC-007)".
5. A new plan row **D-3-25** in `plan-records-path.md` records the decision.
6. No production code changes. The change touches `internal/workflow/interp/*_test.go`, testdata and docs only.

Sequencing: this lands on PR #331's line (base `TICK-334/workflow-interpreter`), or right after it, before the first `GetVersion` branch exists. Nothing blocks on it today.

## 7. Invariant check (incomplete, see header)

| Invariant | Effect |
|---|---|
| ADR-001 I7 idempotency | none. This is test and doc only |
| Temporal determinism | **Strengthened.** A history leaves the gate only when the branch it needs is provably gone (R-5, R-6) |
| R1 / CI-7 references-only | none. Deleted files leave the walk. The manifest holds a file name, a blob id, a change id and an issue key, all with no tenant data |
| Tenant isolation, ACL, claims | none |
| Temporal allow-list (D9) / Worker Versioning | none |

## 8. Open questions for reviewers

1. **Technical Architect:** is a 24-hour freshness window for the rule (c) evidence right, or should it be "the query is run after the PR is approved"?
2. **QA advisor:** should R-5 scan all of `internal/workflow/` or only `internal/workflow/interp/`? The draft says all of `internal/workflow/`, in case the interpreter is split.
3. **CTO:** is it acceptable that "agents never merge a manifest PR" is enforced by process (the MergePr label), not by GitHub, until option E is viable?
