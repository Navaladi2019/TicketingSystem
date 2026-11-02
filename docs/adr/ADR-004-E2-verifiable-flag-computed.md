---
type: adr-erratum
id: ADR-004-E2
title: "Erratum E2 to ADR-004 R20 and audit-export-v1 §3/§4: only the trailer's verifiable must equal the computed value; header stubs is the request"
status: PROPOSED
date: 2026-11-02
revision: 2 (Technical Architect critique, TICK-464 comment 01a199ea-b7a7)
issue: TICK-464
rfc: none (erratum raised directly on TICK-464; no RFC round). Evidence under docs/adr/ADR-004/spikes/solution-architect/tick-464/
supersedes: "ADR-004 R20 bullet \"With `stubs=false` the trailer says `verifiable: false`\"; audit-export-v1.md §3 trailer row `verifiable`, §4 preamble sentence on reason codes, §4 step 6, §4 step 8"
amends: "[[ADR-004-audit-log]] R20; docs/api/audit-export-v1.md (on the TICK-418 story branch) §3, §4, §6"
source: Technical Lead review TICK-461 of PR #370 (TICK-425, nava audit verify)
deciders: Solution Architect (author, verdict after critique); Technical Architect (critique); CTO (tie-break)
---

# ADR-004 erratum E2: verifiability is computed; the trailer must agree, the header need not

> **Status: PROPOSED, rev 2.** The Technical Architect critiqued rev 1 (comment `01a199ea-b7a7`) with one blocking finding: an honest `stubs:false` window with entries but no match was still `tampered`. Rev 2 applies it with all the non-blocking text changes. Awaiting the Technical Architect's re-check and the CTO tie-break. Numbered E2 because E1 (TICK-422, PR #367) is open.
>
> **Changelog rev 1 → rev 2.** (1) Step 6: `last_seq` must be after the anchor; zero range lines is a tail gap, not `head_mismatch`; the head hash is checked whenever the last line is at `last_seq`, gaps or not. (2) Outcome table: a "no line matched" row and two forged-head rows. (3) §3 range text qualified for `stubs=false`; a window with entries but no match is not `empty`. (4) §4.1 `head_mismatch` and `seq_missing` rows widened. (5) Three more vectors (ten in all). (6) Residual-risk note on tail truncation. (7) Exporter test for the no-match window.
>
> **Invariants source.** The project description defines no invariants. As in ADR-004, this is checked against [[ADR-001-platform-blueprint]] §1 I1–I9; the invariant check is incomplete against any separate list of eight.

## Decision

1. **The header's `stubs` is the request, not a claim.** It echoes what the caller asked for and is written before the range is read. It governs one thing only: in a `stubs: true` file every `seq` from `anchor_seq + 1` to `last_seq` must appear (otherwise `tampered`, `seq_missing`). A `stubs: false` file may have gaps, or none.
2. **Only the trailer's `verifiable` must equal the computed value.** The trailer is written last, so the exporter knows it. A disagreement is `tampered` (`verifiable_flag_mismatch`).
3. **"Computed" includes the tail.** The range is contiguous only if the anchor is present, no `seq` is skipped between lines, **and the last range line is `last_seq`**. An honest `stubs: false` export whose filter excluded the last entries of the range, **or every entry of it** (anchor, then no range lines), is `unverifiable`, not `tampered` (`head_mismatch`). `last_seq` must be greater than `anchor_seq`. Whenever the last range line is at `last_seq`, its `entry_hash` must be `head_hash`, gaps or not.
4. **The exporter does not buffer.** It streams as today, counts as it goes, and writes `verifiable = (range non-empty) ∧ (count + stub_count = last_seq − anchor_seq)`.
5. **The reason codes are a closed list in §4**, not "the ones the vectors use". The verifier's structural codes, including `seq_missing` and `anchor_mismatch`, are named there.
6. **Not a breaking change to v1.** No existing vector changes its expected outcome. Before this erratum, one class of honest files was judged `tampered`; now those files are `pass` or `unverifiable`. Every file that passed before still passes. No v1 exporter has shipped (TICK-444 is open), so no file in the wild changes its verdict. The format stays `nava.audit.export/v1`.

### Outcome table (binding)

`C` = computed contiguity (decision 3). Empty ranges are unchanged: `empty`, and `verifiable` must be `false`.

| header `stubs` | range | trailer `verifiable` | outcome | reason |
|---|---|---|---|---|
| `true` | contiguous | `true` | `pass` (or `pending`, R24) | |
| `true` | any `seq` missing, tail included, or no range line at all | any | `tampered` | `seq_missing` |
| `true` or `false` | contiguous | `false` | `tampered` | `verifiable_flag_mismatch` |
| `false` | contiguous (filter excluded nothing) | `true` | **`pass`** (was `tampered`) | |
| `false` | gap, tail included | `false` | `unverifiable` (tail gap was `tampered`, `head_mismatch`) | |
| `false` | gap | `true` | `tampered` | `verifiable_flag_mismatch` |
| `false` | window has entries, **no line matched** (anchor, no range lines) | `false` | `unverifiable` (was `tampered`, `head_mismatch`) | |
| any | last line at `last_seq`, its `entry_hash` ≠ `head_hash` (gaps or not) | any | `tampered` | `head_mismatch` |
| any | trailer `last_seq` ≤ `anchor_seq` | any | `tampered` | `head_mismatch` |

## Context

- `audit-export-v1.md` §4 step 8 required **both** the header's `stubs` and the trailer's `verifiable` to equal the computed verifiability. The exporter (TICK-444) writes the header first, from the request. When every entry in the range matches the filter, a `stubs=false` export has no gaps: the header says `false` and the computed value is `true`. The verifier (PR #370) correctly follows the spec and reports `tampered` with exit 1 on a file nobody touched. The spec was wrong, not the code.
- While checking that case, a second defect of the same kind surfaced: the verifier only counts gaps **between** lines. A `stubs=false` export whose filter excludes the **last** entries of the range has no internal gap, so it reaches step 6 as "contiguous" and fails `head_mismatch`. The spec's own definition ("every `seq` in `anchor_seq+1..last_seq` appears") already makes that file not verifiable; step 6 and the code disagreed with it.
- ADR-004 R20 said "With `stubs=false` the trailer says `verifiable: false`". That is the same mistake at the ADR level: the trailer is a computed fact, not a copy of the request.

### Evidence (spike, red to green)

`docs/adr/ADR-004/spikes/solution-architect/tick-464/`: `spike464_test.go.txt` builds seven files from the golden `unfiltered` vector and runs the PR #370 verifier (commit `47bdec9`). `verify-step8.patch` is the change in decision 2–3 (about 15 lines in `onTrailer`). `output.txt` has both runs:

| case | PR #370 today | with the patch |
|---|---|---|
| B: `stubs:false`, complete, `verifiable:true` | `tampered` `verifiable_flag_mismatch` | **`pass`** |
| C: `stubs:false`, tail excluded, `verifiable:false` | `tampered` `head_mismatch` | **`unverifiable`** |
| F: `stubs:true`, tail missing | `tampered` `head_mismatch` | `tampered` `seq_missing` |
| A (TA-6F), D, E, G | unchanged verdicts | unchanged verdicts |

With the patch, `go test ./internal/audit/...` passes: every committed vector keeps its outcome.

**Rev 2.** `verify-step8.patch` is now rev 1 plus the Technical Architect's step-6 switch; rev 1 is kept as `verify-step8.rev1.patch`. Cases H–L come from the critique (`spike464ta_test.go.txt`); the Solution Architect reproduced them (`output-rev2.txt`):

| case | rev 1 patch | rev 2 patch |
|---|---|---|
| H: `stubs:false` 1..5, no line matched (honest) | `tampered` `head_mismatch` | **`unverifiable`** |
| I: `stubs:false` 1..5, seq 2 excluded, `head_hash` forged | `unverifiable` | **`tampered` `head_mismatch`** |
| J: `stubs:false` 1..5, seq 5 excluded, `head_hash` forged | `unverifiable` | `unverifiable` (no line at `last_seq`, not checkable) |
| K: `stubs:false` complete 1..5, `head_hash` forged | `tampered` `head_mismatch` | `tampered` `head_mismatch` |
| L: `stubs:true` 1..5, every range line deleted | `tampered` `head_mismatch` | `tampered` `seq_missing` |
| A–G | | identical to rev 1 |

`go test ./internal/audit/... -count=1` passes with rev 2; every committed vector is unchanged.

## Options considered

| Option | What it means | Why not / why |
|---|---|---|
| **A. Trailer-only check (chosen)** | Header `stubs` = request; trailer `verifiable` = computed; verifier checks the trailer only. | The trailer is the one line written after the range, so it is the only place a computed fact can honestly go without buffering. |
| B. Exporter buffers the range, then writes the header | Header `stubs` becomes "computed" too. | Defeats streaming: a range can reach the R21 caps (pages of 5,000, 15-minute cap), so buffering means holding up to the whole export in memory or a temp file before the first byte. R19 commits `audit/export` before the first byte, so a long buffer also widens the lease window. And it buys nothing: both flags are unhashed, so the verifier must compute regardless. |
| C. Exporter runs a `COUNT` first, then writes the header | Same as B without buffering. | An extra query per export and a second place that must agree with the stream. It also rewrites what the caller asked for (`stubs=false` would print as `true`), so the header no longer records the request. |
| D. Drop the header from every check, including `seq_missing` | A gap in a `stubs:true` file becomes `unverifiable` instead of `tampered`. | Rejected. An honest exporter never leaves a gap under `stubs=true`, so a gap there is evidence of deletion and should say so. It costs nothing to keep. |

## What this does not change

- **What a pass proves (§4 step 9).** A pass still needs computed contiguity from the anchor to `head_hash`. The header flag was never authenticated (it is not hashed), so relaxing it removes no protection: an attacker who deletes lines and flips both flags gets `unverifiable` before and after this erratum. AR-TICK-409-01 (no external anchor) is unchanged; no new accepted risk. One verdict does move: truncating the tail of a `stubs:true` file and flipping both flags now gives `unverifiable` rather than `tampered` (`head_mismatch`). That is the same strength as deleting lines in the middle and flipping both flags, which was already `unverifiable`.
- **Tenant isolation, R19–R24, the canonical form, the hashes.** Untouched.

## Replacement text

These are the normative words. The follow-up tasks apply them to `docs/api/audit-export-v1.md` on the story branch.

**ADR-004 R20**, last bullet, reads:

> With `stubs=false` excluded entries are left out. The trailer's `verifiable` is computed by the exporter: `true` only when the range is non-empty and every `seq` from `first_seq − 1` to `head_seq` is present; the header's `stubs` echoes the request.

**audit-export-v1 §3, trailer table, `verifiable`:**

> `true` when the range is non-empty and every `seq` in `anchor_seq+1..last_seq` has a line (`count + stub_count = last_seq − anchor_seq`); otherwise `false`. The exporter computes it while streaming. A `stubs=false` export whose filter excluded nothing in the range is `verifiable: true`.

Add under the `header` heading: *`stubs` is the request as received; it is not a statement about the range.*

**§3 range and anchor**, the bullet "Every `seq` in `first_seq..last_seq` then appears exactly once, ascending" reads:

> With `stubs=true`, every `seq` in `first_seq..last_seq` then appears exactly once, ascending. With `stubs=false` only matching entries appear, ascending. A window with sealed entries but no matching entry is **not** empty: it has the anchor, no range lines, and `verifiable: false`.

The bullet "If no sealed entry falls in the window, there is no anchor…" is unchanged; it covers only a window with no sealed entry at all.

**§4 preamble**, the sentence "The first `tampered` check names its reason; the reason codes are the ones used in the vectors" reads:

> The first `tampered` check names its reason, one of the codes in §4.1. The vectors use a subset.

**§4 step 3**, last sentence reads: *A gap is allowed only in a `stubs: false` file; in a `stubs: true` file it is `tampered` (`seq_missing`).*

**§4 step 6** reads:

> 6. `last_seq` must be greater than `anchor_seq` (`head_mismatch`). A range line beyond `last_seq` is `tampered` (`head_mismatch`). If the last range line's `seq` is below `last_seq`, or there is no range line, that is a gap at the tail (`seq_missing` under `stubs: true`, otherwise the range is not contiguous). When the last range line's `seq` is `last_seq`, its `entry_hash` is `head_hash` (`head_mismatch`), whether or not the range has gaps.

**§4 step 8** reads:

> 8. **Verifiability is computed, not read.** The file is contiguous when the anchor is present and every `seq` in `anchor_seq+1..last_seq` appears (steps 3 and 6). The trailer's `verifiable` must equal that value, else `tampered` (`verifiable_flag_mismatch`). The header's `stubs` is the request: it decides only whether a gap is `seq_missing` (step 3), and `stubs: false` over a contiguous range is legal. Outcome: contiguous gives `pass` (or `pending`, step 7); not contiguous gives `unverifiable`; an empty range (`anchor_seq: null`) gives `empty`, its `verifiable` must be `false`, and it is reported as neither a pass nor a failure.

**New §4.1 Reason codes.** Stable strings; renaming one is a breaking change to v1.

| Code | Step | Meaning |
|---|---|---|
| `malformed_line` | 1, 10 | not one strict JSON object of its kind; blank line, CR line end, invalid UTF-8, over-long line |
| `unknown_kind` | 10 | a `kind` other than header, anchor, entry, stub, trailer |
| `header_invalid` | 1 | first line is not a v1 header |
| `line_order` | 1 | a line out of §3 order (second header, range line after an unsealed line, anything after the trailer) |
| `trailer_missing` | 1 | the file ends without a trailer |
| `count_mismatch` | 1 | trailer counts disagree with the lines; an empty range with non-null `last_seq`/`head_hash` |
| `anchor_missing` | 2 | the first chain line is not the anchor (R20) |
| `anchor_invalid` | 2 | the anchor's own members disagree (genesis with `seq ≠ 0`, non-genesis without hashes) |
| `anchor_mismatch` | 2 | the anchor is not at `after_seq` when set, or not genesis when `from` is unset; or the trailer's `anchor_seq` names another `seq` (R20) |
| `seq_duplicate` | 3 | a `seq` appears twice (R15) |
| `seq_out_of_order` | 3 | a `seq` lower than the one before |
| `seq_missing` | 3, 6 | a gap, tail included, in a `stubs: true` file, including a range with no lines (R15, R20) |
| `line_state_mismatch` | 3 | a line's `sealed` / `in_verified_range` / `seq` contradict each other |
| `entry_hash_mismatch` | 3, 4 | `prev_hash` is not the previous `entry_hash`, or `entry_hash` does not recompute |
| `envelope_sha256_mismatch` | 4 | envelope does not hash to `envelope_sha256` |
| `header_tenant_mismatch` | 4a | an entry's tenant is not the header's |
| `diff_state_mismatch` | 5 | diff, salt, erasure and redaction members contradict `diff_sha256` |
| `diff_missing` | 5 | `diff_sha256` set, entry neither erased nor redacted, diff or salt missing |
| `diff_sha256_mismatch` | 5 | `SHA-256(salt ‖ diff)` is not `diff_sha256` |
| `head_mismatch` | 6 | `last_seq` not after the anchor; a range line beyond `last_seq`; the line at `last_seq` whose `entry_hash` is not `head_hash` |
| `erasure_unlisted` | 7 | erased, no chain-valid `audit/erase` lists it (R24) |
| `verifiable_flag_mismatch` | 8 | trailer `verifiable` is not the computed value |

`anchor_mismatch` is checked offline only when the header fixes `first_seq` without time: `after_seq` set, or no `from`. With a `from` bound, `first_seq` depends on `occurred_at`, which a stub does not carry, so the anchor's position is not checkable from the file (§4 step 9 limits).

**§6 `exports` row** gains the cases below.

## What the vectors should say

`internal/audit/chain/vectors/export.go` (`exportCases`). Existing cases keep their outcomes; `verifiable-flag-false-on-complete-file` (TA-6F) stays `tampered` / `verifiable_flag_mismatch` and its comment changes to "the trailer claims a complete file is not verifiable". Add these ten:

| name | build | expect | reason |
|---|---|---|---|
| `stubs-false-contiguous` | unfiltered, `stubs: false`, `verifiable: true` | `pass`, all lines `verified` | |
| `stubs-false-tail-excluded` | range 1..5 (`last_seq` 5, `head_hash` of seq 5), seq 5 excluded, `stubs: false`, `verifiable: false`, `stub_count` 0 | `unverifiable` | |
| `stubs-false-gap-flagged-verifiable` | `stubs-false-honest` with `verifiable: true` | `tampered` | `verifiable_flag_mismatch` |
| `stubs-true-seq-missing` | unfiltered range 1..5, seq 2 line removed, counts adjusted | `tampered` | `seq_missing` |
| `stubs-true-tail-missing` | unfiltered range 1..5, seq 5 line removed, counts adjusted | `tampered` | `seq_missing` |
| `anchor-not-after-seq` | header `after_seq: 2`, anchor is genesis | `tampered` | `anchor_mismatch` |
| `anchor-not-genesis` | header with no `from`, anchor non-genesis at seq 2, range 3..7 | `tampered` | `anchor_mismatch` |
| `stubs-false-none-matched` | range 1..5, no line matched: anchor, no range lines, `count` 0, `stubs: false`, `verifiable: false` | `unverifiable` | |
| `stubs-true-all-missing` | range 1..5, every range line removed, `stubs: true` | `tampered` | `seq_missing` |
| `stubs-false-gap-head-forged` | range 1..5, seq 2 excluded, `stubs: false`, `verifiable: false`, `head_hash` replaced | `tampered` | `head_mismatch` |

Ranges ending at seq 5 avoid seq 6/7 (the erased entry and its `audit/erase`), so R24 does not fire first. `vectors.ReasonSeqMissing` and `vectors.ReasonAnchorMismatch` are added and joined to `TestReasonCodesMatchVectors`. Regenerate with `go run ./internal/audit/chain/cmd/genvectors`; this is an additive vector change under D-5-7 and needs Technical Lead review.

## Follow-up (Technical Lead creates the tasks once this is accepted)

1. **Spec + vectors (TICK-418 branch):** apply the replacement text to `docs/api/audit-export-v1.md`; add the vectors above.
2. **Verifier (`internal/audit/verify/verify.go`):** `onTrailer` per `verify-step8.patch` (rev 2): `last_seq` after the anchor; tail gap (zero lines included) handled before the head check; head hash checked whenever the last line is at `last_seq`; step 8 compares `t.Verifiable` only. Update the `ReasonCodes` comment in `report.go` ("the first eight are the codes the vectors use" becomes "§4.1 lists all of them").
3. **Exporter (TICK-444):** header `stubs` = request; trailer `verifiable = anchor present ∧ count + stub_count = last_seq − anchor_seq`. No buffering, no pre-count. Tests: a `stubs=false` export over a fully matching range verifies `pass`; a `stubs=false` export over a window that has sealed entries but no matching entry verifies `unverifiable` with a non-zero exit status.

Must land before TICK-444 and TICK-447 merge. Does not block TICK-425.

## Acceptance

- The ten new vectors exist and the verifier produces their outcomes; every existing vector is unchanged.
- An honest `stubs=false` export with no excluded entry verifies `pass` (exit 0) end to end (TICK-444 test).
- `audit-export-v1.md` §4 lists every reason code the verifier can emit (a test in `internal/audit/verify` compares the constants against §4.1 or a list kept beside it).
