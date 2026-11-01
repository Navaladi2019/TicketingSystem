---
type: rfc
id: RFC-007
title: Retiring recorded interpreter histories when a GetVersion DefaultVersion arm is retired
status: REVISED (rev 1, the only revision)
date: 2026-11-01
issue: TICK-388
thread: TICK-388 comment 01a19500-9c0f-7c89-acaa-43a17fae2ce5 (rev 0 01a19507, critique 01a1950c-5bf9)
source: TICK-350 / PR #331, docs/quality/ci-pipeline.md "Deploy rules (c)", marked open
pin: go.temporal.io/sdk v1.49.0
author: Solution Architect
---

# RFC-007: Retiring recorded interpreter histories (rev 1)

> **Invariants:** the project description of *Ticket Management System* is empty, so I could not load the eight invariants from it. I checked against ADR-001 §2 (I1–I9) and R1/CI-7 as written in the repo. Treat the invariant check in §7 as **incomplete** until the project description carries them.

## Changelog from rev 0

| Item | Change |
|---|---|
| **O-1 (blocking)** | **Adopted.** Retiring means **raising `minSupported` above `workflow.DefaultVersion` and keeping the `GetVersion` call**. It never means deleting the call. R-5/R-6/R-7 are replaced by R-5′/R-6′/R-7′, and R-8 is new. There is no rebuttal, because I re-checked the TA's SDK citations at v1.49.0 and they hold (§3.0) |
| O-2 | Closed by R-8: every retired history must fail replay with the specific TMPRL1100 "removed support" error. A history that still replays green cannot be retired |
| O-3 | §4 rationale corrected: a wrong retirement **blocks** the missed run. That is recoverable by a fix forward (lower `minSupported` again and redeploy). It does not fail the run for good |
| O-4 | Calls are matched by **import path** `go.temporal.io/sdk/workflow`, whatever the local name, not by the selector text |
| §4 timing | The 24 h window is replaced by the TA's event-based condition, adopted verbatim with one clarification on `T` |
| Q1 / Q3 | Q1 answered by the timing text. Q3 accepted by the CTO as a risk the CTO owns (§4) |
| Q2 | Resolved: the scan covers `internal/workflow/...` (all packages), unchanged |
| R-8 refinement | TMPRL1100 has two messages at the pin. R-8 requires the **"removed support of version -1"** form and the change id, not merely the code (§3.3) |

## 1. Decision

A history is retired **in the same PR that retires the `DefaultVersion` arm of a `workflow.GetVersion` change**. That PR:

- raises the call's `minSupported` from `workflow.DefaultVersion` to the first non-default version;
- deletes the `DefaultVersion` arm, **keeping the call**;
- deletes from `histories/` and `wantHistories` each history that this breaks;
- appends one tombstone per deleted history to an append-only manifest, `internal/workflow/testdata/retired-histories.json`.

`TestHistoriesAppendOnly` accepts a deletion only when all of these hold:

- a tombstone names the file **and** its exact git blob on `main`;
- BASE had the call with `minSupported = DefaultVersion`;
- the work tree still has the call, with `minSupported > DefaultVersion`;
- the history, read from BASE, **fails replay against the work tree with `TMPRL1100 … removed support of version -1 … "<change_id>"`**.

Every one of these is a fact in git, in Go source or in a local replay, so CI checks them with no judgment.

Deploy rule (c) is the one fact CI can't see: no open production run started before the change. It is the **approval**. The Operator records the evidence, the Technical Architect reviews the PR, and **the human owner merges it**. Agents never merge a PR that touches the manifest.

**Form:** a **decision record**: a new plan row D-3-25 in `docs/delivery/plan-records-path.md`, plus the `ci-pipeline.md` "Deploy rules (c)" text (§6). **No ADR-001 amendment.** The append-only rule is not in ADR-001. It lives in P3-23 (D-3-12, D-3-17) and `ci-pipeline.md`, and this RFC changes no invariant, R1/CI-7 or the Temporal allow-list (§7).

## 2. Problem

Deploy rule (c) allows retiring a `GetVersion` change once no open run that started before it remains. After that, histories recorded before the change no longer replay, because they have no version marker. Today `TestHistoriesAppendOnly` (`internal/workflow/interp/histories_test.go`) refuses every deletion of a file that is on `main`, and `checkHistorySet` refuses a file that is missing from `wantHistories`. So the arm can never be retired. If we bolted on an escape hatch with no checks, any red replay could be "fixed" by deleting the history, and the gate would prove nothing.

## 3. Mechanism

### 3.0 Why "keep the call" (verified at the pin)

I re-read these in `go.temporal.io/sdk@v1.49.0/internal` myself:

- `workflow.go:2457-2472`: the documented removal form is `GetVersion(ctx, id, 1, 1)`. The comment says: *"It is recommended to keep the GetVersion() call even if single branch is left … it ensures that if there is older version execution still running, it will fail here and not proceed."*
- `internal_event_handlers.go:953-976`: on replay with no marker, `GetVersion` resolves to `DefaultVersion`. `validateVersion` then panics with `[TMPRL1100] Workflow code removed support of version -1. for "<changeID>" changeID …` when `minSupported > DefaultVersion`. The same code is also used for *"Workflow code is too old to support version …"* (`version > maxSupported`). That is a different failure, so R-8 requires the "removed support" text.

The TA's spike shows the consequence on our own four timer-bearing histories. With the **call deleted** (the form rev 0 required), replay is **green**: version markers are skipped by the determinism check (`internal_task_handlers.go:1539-1544`), so a missed old run silently runs the new arm. With the **call kept** at `minSupported=1`, replay fails with TMPRL1100. Rev 0's R-5 therefore required the unsafe form and forbade the safe one. That is withdrawn.

Under the default `WorkflowPanicPolicy` (`BlockWorkflow`, `worker.go:542-548`, which the repo does not override), a missed production run **blocks** on its next workflow task instead of continuing.

### 3.1 The manifest

`internal/workflow/testdata/retired-histories.json` sits **beside** `histories/`, not inside it, so the `*.json` glob in `checkHistorySet` and the replay set are unchanged.

```json
{
  "format": "nava.retired-histories/v1",
  "retired": [
    {
      "history": "else-default.json",
      "blob": "<40-hex git blob id of the file on main>",
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
| `change_id` | the `GetVersion` change id whose `DefaultVersion` arm was retired | non-empty string (R-5′, R-6′, R-8) |
| `issue` | the retirement issue that carries the rule (c) evidence | matches `^TICK-[0-9]+$` |

Decoding is strict: unknown fields fail, and so does a duplicate `history`.

### 3.2 The retirement PR

One PR does all of these, and nothing else:

1. **Raises `minSupported`** of `workflow.GetVersion(ctx, "<change_id>", workflow.DefaultVersion, N)` to the first non-default version (normally `1`, giving `GetVersion(ctx, "<change_id>", 1, N)`). It deletes the code that runs only for `DefaultVersion`. **The call stays, forever.** It is also the place a later change to the same id raises `maxSupported`.
2. Deletes each history that now fails replay with R-8's error from `histories/`, and from `wantHistories`.
3. Appends one manifest entry per deleted file.

Which histories go is no longer prose. R-8 proves every deleted one broke for this reason, and the existing replay test proves every remaining one is still green.

### 3.3 Rules `TestHistoriesAppendOnly` enforces (BASE = `origin/main`)

"A `GetVersion` call" means a call expression whose function resolves to `GetVersion` in the package with import path **`go.temporal.io/sdk/workflow`**, under any local name or alias (go/parser plus the file's import specs; a dot-import is refused by R-7′). The scan covers non-test `.go` files under `internal/workflow/...`.

| Rule | Check | Fails when |
|---|---|---|
| **R-1** | A file under `histories/` on BASE whose content differs | always. A retirement is a deletion, never an edit |
| **R-2** | A file under `histories/` on BASE that the work tree deleted | no manifest entry has `history` = its name **and** `blob` = its BASE blob |
| **R-3** | BASE's manifest `retired` list is a **prefix** of the work tree's (entries compared field by field) | an entry was edited, reordered or removed. The manifest is append-only too |
| **R-4** | Each entry's `history` is absent from `histories/` | a retired name is reused. A name keeps one meaning forever |
| **R-5′** | For **every** entry: the work tree has at least one `GetVersion` call with `change_id`, and every such call has `minSupported > workflow.DefaultVersion` | the call was deleted, or the `DefaultVersion` arm is still supported |
| **R-6′** | For every entry **added by this change** (not on BASE): BASE's source (`git show BASE:<file>`, same scan) has a `GetVersion(…, change_id, workflow.DefaultVersion, …)` call | the id never existed, or it never supported `DefaultVersion`, so there is nothing to retire. This stops a red replay being "fixed" with an invented tombstone |
| **R-7′** | At every `GetVersion` call, `changeID` is a string literal, `minSupported` is `workflow.DefaultVersion` or an int literal ≥ 0, `maxSupported` is an int literal, and the package is not dot-imported | an argument is a constant, variable or expression. R-5′, R-6′ and R-8 need literals to compare |
| **R-8** | For every entry **added by this change**: the history, read from BASE by its `blob` (`git cat-file blob`), is replayed against the work tree with the local `worker.WorkflowReplayer` (the same setup as `TestReplayRecordedHistories`). The replay must fail with an error containing `TMPRL1100`, `removed support of version -1` and `"<change_id>"`. All histories still in `histories/` must replay green (the existing test) | a history was retired that the removal did not break (O-2), or it broke for another reason |

R-8 replays only entries added by this change. Older tombstones are frozen by R-3, and their replay outcome under later code is irrelevant. This keeps the rule a property of one commit pair.

**Without `BASE`** the test still skips, as today. A laptop's `origin/main` may be stale. The `unit` job always sets BASE. R-4, R-5′ and R-7′ need no BASE, so they also run on a laptop as `TestRetiredHistoriesManifest`.

**Determinism.** Every rule compares git objects, JSON fields or Go AST literals, or runs an in-process replay, which is what `TestReplayRecordedHistories` already does. There is no clock, no network and no Temporal server, so the test gives the same answer on every run of the same commit pair.

**Change ids are never reused.** This follows directly from R-5′: the call, and so the id, stays in the code forever.

### 3.4 Proof the check works (tests in the implementing PR)

`TestAppendOnlyCheck` (scratch repository) gains these cases. Each one is red without its rule and green with it:

- a deletion with a matching tombstone, `minSupported` raised and the call kept passes;
- a tombstone with the wrong blob fails (R-2);
- an edited or removed BASE tombstone fails (R-3);
- a retired name re-added fails (R-4);
- **the call deleted fails** (R-5′);
- **`minSupported` still `DefaultVersion` fails** (R-5′);
- a tombstone whose `change_id` never had a `DefaultVersion` call on BASE fails (R-6′);
- a non-literal id, a non-literal version or a dot-import fails (R-7′);
- an aliased import (`tw "go.temporal.io/sdk/workflow"`) is still matched (O-4);
- **a retired history that still replays green fails** (R-8);
- **a retired history that fails with the "too old to support" TMPRL1100, or any other error, fails** (R-8);
- an edit to a history that has a tombstone still fails (R-1).

The R-8 cases reuse the TA's spike pattern, an interceptor that wraps `NewTimer` (TICK-388 comment `01a1950c-5bf9`), over `signal.json`, so they need no recording.

## 4. Who approves, and why it is outside CI

Rule (c) is a fact about **production**: an Operator `ListWorkflowExecutions` query (`WorkflowType = 'nava.interpreter' AND ExecutionStatus = 'Running' AND StartTime < '<T>'`) returns nothing. CI has no production access and must not get any (operator-only reach, ADR-001 R1). So:

| Step | Who | Recorded where |
|---|---|---|
| Runs the rule (c) query **after the TA approves the PR** and posts `T`, the query text, the UTC time it ran and the empty result | **Operator** | the issue named in `issue` |
| Reviews the retirement PR against §3.2 and the evidence | **Technical Architect** (owner of deploy rules a–c) | PR review comment |
| Merges the PR | **Human owner (nava) only.** The "MergePr" label goes on the issue and the PR, and agents never merge it, even though it is not a story or an ADR | GitHub merge |

**Why the human owner (corrected, O-3).** If the query missed an old run, the deployed retirement makes that run **block** on its next workflow task (TMPRL1100 under `BlockWorkflow`). It does not continue wrongly, and it is not lost. The recovery is a fix forward that rule (b) allows: lower `minSupported` back to `DefaultVersion`, restore the arm and redeploy, and the run resumes. Because a production run stalls until a human-driven redeploy, a human makes the call. The process is still recoverable.

**Process-only enforcement (Q3, accepted by the CTO).** CI can't tell an agent's PR from the owner's, because both come from the `Navaladi2019` account. "Agents never merge a manifest PR" is therefore enforced by process. **Accepted risk, owned by the CTO.** It is revisited when agents get their own GitHub identity, at which point it is enforced with CODEOWNERS plus "require review from code owners" on `retired-histories.json`. The CTO adds it to `docs/cto/risks.md`.

**Timing (replaces rev 0's 24 h window).**

> The Operator runs the rule (c) query **after the Technical Architect approves the PR**. `T` is the **completion** time of the latest interpreter-worker deploy that introduced `change_id`: the moment rule (a) stopped the last old worker, not the moment the rollout started. If that deploy was later redeployed or rolled back, `T` is the latest such completion. The evidence comment states `T`, the query, the UTC time it ran and the empty result. It stays valid until merge **unless** a `nava.interpreter` reset or an interpreter-worker deploy or rollback happens after the query. In that case, run it again. There is no elapsed-time limit.

Why a time limit was wrong: for a fixed `T`, the set of runs `Running AND StartTime < T` can only shrink, so re-running the query later never finds more. Only a wrong `T`, a reset, or a deploy can invalidate the evidence. Whether a reset run's visibility `StartTime` falls before `T` is **not verified**, so the rule treats any reset as invalidating the evidence. The TA's optional `TemporalChangeVersion` refinement is **deferred**. It is a verify-by-spike item to run only if rule (c) blocks retirements in practice.

## 5. Options considered

| Option | Verdict | Why |
|---|---|---|
| **A. Delete the history, append an append-only tombstone, keep the call with raised `minSupported`, and prove the break by replay (R-8)** | **Chosen** | Every condition is checkable in git, the AST or a local replay. A missed production run blocks loudly instead of silently running the wrong arm. Git keeps the deleted content, recoverable by `blob` |
| A0. Rev 0: delete the `GetVersion` call (R-5 "call gone") | **Rejected (O-1)** | At v1.49.0 the replay gate goes **green** on old histories, and a missed old run silently executes the new arm. It also forbids the SDK-recommended form |
| B. Move the file to `histories/retired/`, which the replay skips | Rejected | Dead fixtures stay in the tree forever, and it needs the same tombstone data anyway. R-8 already replays the retired file from BASE by blob, so a copy in the tree adds nothing |
| C. Allow-list as a Go slice in `histories_test.go` | Rejected | Same content as A, but data mixed into code that is edited for other reasons. An append-only check over Go source is a text diff, not a structured prefix check |
| D. Re-record the old histories on the new code | Rejected | It defeats the gate. A re-recorded history proves nothing about runs already in flight |
| E. CI enforces approval: CODEOWNERS + "require review from code owners" | Deferred (CTO-owned risk, §4) | Agent PRs come from the owner's own account (`Navaladi2019`, e.g. PR #331). GitHub does not count an author's own review, so every retirement PR would be blocked |
| F. CI runs the rule (c) query itself | Rejected | It would give CI production Temporal reach (ADR-001 R1), and the test would depend on live state, so it would not be deterministic |

## 6. Acceptance criteria (for the Technical Lead)

1. `internal/workflow/testdata/retired-histories.json` exists with `"retired": []` and the v1 format.
2. `TestHistoriesAppendOnly` implements R-1 to R-4, R-5′, R-6′, R-7′ and R-8 with BASE. `TestRetiredHistoriesManifest` implements the schema checks, R-4, R-5′ and R-7′ without BASE. Calls are matched by import path.
3. `TestAppendOnlyCheck` and the R-8 replay cases cover every bullet in §3.4, each one shown red then green.
4. `docs/quality/ci-pipeline.md`:
   - "Deploy rules (c)" is replaced with the text below;
   - the `make/workflow.mk` comment and the `wantHistories` comment ("none is ever removed") say "except by a tombstone (RFC-007)";
   - "Shipping a change that the gate rejects" option 1 gains the sentence: "The call is never deleted; retiring its old arm is rule (c)."
5. A new plan row **D-3-25** in `plan-records-path.md` records the decision. Rule (c) in D-3-17 is reworded to "retire a `GetVersion` change's `DefaultVersion` arm (raise `minSupported`, keep the call) only when …".
6. No production code changes. The change touches `internal/workflow/interp/*_test.go`, testdata and docs only.
7. The CTO adds the Q3 accepted risk to `docs/cto/risks.md`. This is tracked by the CTO and is not a Technical Lead task.

**Replacement text for `ci-pipeline.md` "Deploy rules (c)":**

> - **(c)** Retire a `GetVersion` change's old arm **only by raising its `minSupported`** above `workflow.DefaultVersion` and deleting the `DefaultVersion` code. **Never delete the `GetVersion` call**: kept, it makes any old run that is still open fail on TMPRL1100 and block, where a deleted call lets it run the new arm silently. Do this only when an Operator `ListWorkflowExecutions` query (`WorkflowType = 'nava.interpreter' AND ExecutionStatus = 'Running' AND StartTime < '<T>'`, where `T` is the completion of the latest interpreter-worker deploy that introduced the change) returns nothing. Run the query after the Technical Architect approves the PR. Run it again if a `nava.interpreter` reset or an interpreter-worker deploy or rollback happens before the merge.
>   **Retiring the histories this breaks.** In the same PR, delete each history that now fails replay with `TMPRL1100 … removed support of version -1 … "<change_id>"` from `histories/` and from `wantHistories`, and append one entry per file (`history`, `blob` on main, `change_id`, `issue`) to `internal/workflow/testdata/retired-histories.json`. `TestHistoriesAppendOnly` checks the tombstone, the kept call and the replay failure (RFC-007, D-3-25). The evidence goes on the `issue`. Only the human owner merges a PR that touches the manifest ("MergePr" label); agents never do.
>   If an old run was missed and blocks after the deploy, fix forward: lower `minSupported` back to `DefaultVersion`, restore the arm, and redeploy.

Sequencing: this lands on PR #331's line (base `TICK-334/workflow-interpreter`), or right after it, before the first `GetVersion` change exists. Nothing blocks on it today.

## 7. Invariant check (incomplete, see header)

| Invariant | Effect |
|---|---|
| ADR-001 I7 idempotency | none. This is test and doc only |
| Temporal determinism | **Strengthened.** A history leaves the gate only when it provably fails on the retired arm (R-8), and the kept call makes production fail loudly in the same case |
| R1 / CI-7 references-only | none. The manifest holds a file name, a blob id, a change id and an issue key, with no tenant data. R-8 replays BASE blobs that already passed the R1 walk |
| Tenant isolation, ACL, claims | none |
| Temporal allow-list (D9) / Worker Versioning | none |

## 8. Open questions

None blocking. Deferred: the `TemporalChangeVersion` query refinement (verify-by-spike, only if rule (c) blocks retirements in practice).
