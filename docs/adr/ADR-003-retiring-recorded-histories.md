---
type: adr
id: ADR-003
title: A recorded interpreter history is retired only by raising GetVersion's minSupported (call kept), with an append-only tombstone CI can check
status: ACCEPTED
date: 2026-11-01
accepted: 2026-11-01 (nava, TICK-388 comment 01a1958a-3b69-70c9-b9f7-e7ecc1f8f4fb: "ACCEPTED and APPROVED")
issue: TICK-388
rfc: docs/adr/ADR-003/rfc/RFC-007-retiring-recorded-histories.md (rev 1, archived; thread under TICK-388 comment 01a19500-9c0f-7c89-acaa-43a17fae2ce5)
supersedes: "docs/quality/ci-pipeline.md 'Deploy rules (c)' (the 'delete a GetVersion branch' wording and its Open note); PLAN-003 D-3-17 rule (c) wording"
source: TICK-350 / PR #331 (P3-23 replay gate), Quality, Testing & CI Advisor
deciders: Solution Architect (author), Technical Architect (APPROVED rev 1, TICK-388 comment 01a19510-22ed-7314-86ed-884dea3bbcf9), CTO
signs: nava (signed 2026-11-01: "ACCEPTED and APPROVED")
---

# ADR-003: Retiring recorded interpreter histories

> **Status: ACCEPTED, 2026-11-01.** nava approved it on TICK-388 (comment `01a1958a`: "ACCEPTED and APPROVED"), accepting residuals RS-1 to RS-4 (§7). Delivery plans the implementation in TICK-395, which carries the binding `ci-pipeline.md` "Deploy rules (c)" text from RFC-007 rev 1 §6. RS-1 is also R-004 in `docs/cto/risks.md` (PR #341). It comes from RFC-007 rev 1. The Technical Architect, the only reviewer, approved rev 1 (comment `01a19510`) after one blocking objection in round 1 (O-1) was adopted. The CTO ruled that this is a standalone ADR, not an ADR-001 amendment (comment `01a19511`): it sets a deploy rule that only a human can approve, but it changes no ADR-001 invariant. The plain-language explainer is `ADR-003-explained.html`.
>
> **Invariants:** the project description is empty, so the eight invariants could not be loaded from it. §9 checks against ADR-001 §2 and R1/CI-7 as written in the repo, and is marked **incomplete**.

## 1. Decision

1. **A `GetVersion` change's old arm is retired only by raising `minSupported`.** This means `workflow.GetVersion(ctx, "<change_id>", workflow.DefaultVersion, N)` becomes `GetVersion(ctx, "<change_id>", 1, N)` (or the first non-default version), and the `DefaultVersion` code is deleted. **The call is never deleted.**
2. **In the same PR**, each recorded history that now fails replay with `TMPRL1100 … removed support of version -1 … "<change_id>"` is deleted from `internal/workflow/testdata/histories/` and from `wantHistories`. One tombstone per deleted file is appended to the append-only manifest `internal/workflow/testdata/retired-histories.json`.
3. **`TestHistoriesAppendOnly` enforces this mechanically** (§3), using git objects, Go AST literals and a local replay. There is no clock, no network and no judgment.
4. **Approval is human, outside CI.** The Operator posts the rule (c) production query evidence after the Technical Architect approves the PR. **Only the human owner merges** a PR that touches the manifest ("MergePr" label). Agents never do.
5. **Scope:** only the `DefaultVersion` arm. Retiring a later recorded version (≥ 0) is out of scope and needs its own decision (§7, RS-3).

## 2. Context

The P3-23 replay gate (PR #331) replays every committed history in `internal/workflow/testdata/histories/` against the current `nava.interpreter` code. `TestHistoriesAppendOnly` (`internal/workflow/interp/histories_test.go`) forbids editing or deleting any history that is on `main`. Deploy rule (c) allowed deleting a `GetVersion` branch once no open run that started before it remained. After that, the old histories no longer replay, but they can't be deleted either. `ci-pipeline.md` marked this as **Open**.

The round-1 spike (`ADR-003/spikes/technical-architect/rfc-007-getversion-removal.md`, SDK v1.49.0) showed that the obvious form, deleting the call, is unsafe. With the call deleted:

- old histories replay **green**, because version markers are skipped by the determinism check;
- a production run missed by the query would **silently run the new arm**.

With the call kept and `minSupported` raised, the same histories fail with TMPRL1100. Under the default `BlockWorkflow` panic policy, a missed run **blocks** instead of continuing.

## 3. Design

### 3.1 The manifest

`internal/workflow/testdata/retired-histories.json` sits beside `histories/`, so the replay glob is unchanged. It is decoded strictly: unknown fields fail, and so does a duplicate `history`.

```json
{ "format": "nava.retired-histories/v1",
  "retired": [ { "history": "else-default.json", "blob": "<40-hex blob on main>",
                 "change_id": "step-ref-before-timer", "issue": "TICK-412" } ] }
```

### 3.2 Rules (BASE = `origin/main`)

"A `GetVersion` call" means a call resolved through the import path `go.temporal.io/sdk/workflow`, under any alias, in non-test `.go` files under `internal/workflow/...`.

| Rule | Fails when |
|---|---|
| **R-1** | a history on BASE was edited. A retirement is a deletion, never an edit |
| **R-2** | a history on BASE was deleted without a tombstone whose `history` and `blob` both match BASE |
| **R-3** | BASE's `retired` list is not a field-for-field prefix of the work tree's. The manifest is append-only |
| **R-4** | a retired name exists again under `histories/` |
| **R-5′** | for any entry: no `GetVersion` call with `change_id` remains, or one has `minSupported ≤ DefaultVersion` |
| **R-6′** | for an entry added by this change: BASE had no `GetVersion(…, change_id, workflow.DefaultVersion, …)` |
| **R-7′** | any `GetVersion` argument is not a literal (string id, `workflow.DefaultVersion` or an int), or the package is dot-imported |
| **R-8** | for an entry added by this change: the BASE blob, replayed against the work tree with the local `worker.WorkflowReplayer`, does **not** fail with `TMPRL1100`, `removed support of version -1` and `"<change_id>"`. All remaining histories must replay green (the existing test) |

Without `BASE` the test skips, as today. R-4, R-5′ and R-7′ also run without BASE as `TestRetiredHistoriesManifest`. Each rule gets a red-then-green case in `TestAppendOnlyCheck` (RFC-007 §3.4).

### 3.3 Approval

| Step | Who |
|---|---|
| Reviews the retirement PR | Technical Architect |
| Then runs `WorkflowType = 'nava.interpreter' AND ExecutionStatus = 'Running' AND StartTime < '<T>'` and posts `T`, the query, the UTC run time and the empty result on the tombstone's `issue`. `T` is the **completion** time of the latest interpreter-worker deploy that introduced `change_id`. The query must be run again if a `nava.interpreter` reset or an interpreter-worker deploy or rollback happens before the merge. There is no elapsed-time limit | Operator |
| Merges | human owner (nava) only |

## 4. Consequences (land with this ADR)

- **PLAN-003:**
  - new row **D-3-25** in `docs/delivery/plan-records-path.md`. It is in this PR;
  - the D-3-17 rule (c) wording becomes "retire a `GetVersion` change's `DefaultVersion` arm only by raising its `minSupported` and keeping the call". It is in this PR.
- **`docs/quality/ci-pipeline.md` "Deploy rules (c)"** is replaced by the text in RFC-007 rev 1 §6. That section exists only on PR #331's line (base `TICK-334/workflow-interpreter`), not on `main`. So the text lands in the implementing PR on that line, together with:
  - the test change;
  - the `make/workflow.mk` and `wantHistories` comments ("except by a tombstone, ADR-003");
  - the sentence in "Shipping a change that the gate rejects" option 1.
- **Implementation:** test code, testdata and docs only, with no production code change. The acceptance criteria are in RFC-007 rev 1 §6. The Technical Lead plans it in TICK-395.

## 5. Options considered and rejected

| Option | Why rejected |
|---|---|
| Delete the `GetVersion` call (RFC-007 rev 0) | Replay goes green on old histories, and a missed run silently executes the new arm (spike, 4 of 4 histories) |
| Move retired files to `histories/retired/` | Dead fixtures stay in the tree, and it needs the same tombstone data anyway. R-8 already reads the BASE blob |
| Allow-list as a Go slice in the test | The same data mixed into code, and the append-only check over Go source becomes a text diff |
| Re-record old histories on the new code | Proves nothing about runs already in flight, which defeats the gate |
| CI enforces approval through CODEOWNERS and "require review from code owners" | Agent PRs come from the owner's own account (`Navaladi2019`), and GitHub doesn't count an author's own review, so every retirement PR would be blocked. Deferred (RS-1) |
| CI runs the rule (c) query | Production Temporal reach from CI (ADR-001 R1), and a live-state test is not deterministic |
| A 24 h freshness window for the evidence | Elapsed time can't invalidate it. For a fixed `T`, the result set only shrinks. Replaced by the event-based re-run rule |

## 6. Evidence

- **SDK source at v1.49.0**, read by both reviewers:
  - `internal/workflow.go:2457-2472`: keep the call;
  - `internal/internal_event_handlers.go:953-976`: the two TMPRL1100 messages; replay with no marker resolves to `DefaultVersion`;
  - `internal/internal_task_handlers.go:1539-1544`: markers are skipped by the determinism check;
  - `internal/worker.go:542-548`: the default panic policy is `BlockWorkflow`.
- **Spike** (Technical Architect, `main` 69f3a62): the call deleted gives **GREEN** on `continue-as-new.run1`, `else-default`, `signal` and `timeout`. The call kept with `minSupported=1` gives `[TMPRL1100] Workflow code removed support of version -1. for "spike-rfc007-timer" changeID` on all four. Archived under `docs/adr/ADR-003/spikes/`.
- **Red → green tests** are an acceptance criterion of the implementation (RFC-007 §3.4). They don't exist yet, because this ADR is design only.

## 7. What is being accepted (residual risks)

| # | Risk | Owner | Revisit when |
|---|---|---|---|
| **RS-1** | "Agents never merge a manifest PR" is enforced by **process only** (the "MergePr" label), not by GitHub. CI can't tell an agent's PR from the owner's, because both use the `Navaladi2019` account. To be added to `docs/cto/risks.md` | CTO | Agents get their own GitHub identity. Then enforce it with CODEOWNERS plus "require review from code owners" on `retired-histories.json` |
| **RS-2** | Rule (c) is a human-checked production fact. If the query misses a run (wrong `T`, an unnoticed reset), that run **blocks** on TMPRL1100 after the deploy. It does not go wrong, and recovery is a fix forward: lower `minSupported`, restore the arm, redeploy | Operator, with the Technical Architect | The first retirement. The optional `TemporalChangeVersion` query refinement is a verify-by-spike if rule (c) proves too strict |
| **RS-3** | **Only the `DefaultVersion` arm is in scope** (Technical Architect, comment `01a19510`). R-8 requires `removed support of version -1`, so a history pinned at a recorded version ≥ 0 can't be retired under this ADR | Solution Architect | The first need to retire a non-default version: a new decision |
| **RS-4** | Whether a reset run's visibility `StartTime` falls before `T` is **not verified**. The rule therefore treats any reset as invalidating the evidence | Operator | Only if resets become routine |

## 8. Rollout and sequencing

Nothing blocks on this today. It matters at the first `GetVersion` retirement. The implementation lands on PR #331's line, or right after it, before the first `GetVersion` change exists.

## 9. Invariant check (incomplete: the project description has no invariants)

| Invariant | Effect |
|---|---|
| ADR-001 I7, idempotency | none: test and doc only |
| Temporal determinism | strengthened: a history leaves the gate only when it provably fails on the retired arm, and production fails loudly in the same case |
| R1 / CI-7, references only | none: the manifest holds a file name, a blob id, a change id and an issue key |
| Tenant isolation, ACL, claims | none |
| D9 allow-list, Worker Versioning | none |
