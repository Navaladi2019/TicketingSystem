---
type: adr-erratum
id: ADR-005-E2
title: "Erratum E2 to ADR-005: overflow `grow` is also refused when a digit comes directly before `{seq:N}`"
status: ACCEPTED
accepted: 2026-11-13 (Solution Architect verdict on TICK-775, after the Technical Architect critique on TICK-667 comment 01a1d155-4d94; no dissent, no CTO tie-break needed)
date: 2026-11-13
issue: TICK-775
rfc: none (erratum from an implementation finding; decision thread under TICK-667 comment 01a1d107-6172-793b-b6ce-4d9de2ce7a98)
supersedes: none
amends: "[[ADR-005-record-numbering]] §3.5 (the `grow` sentence)"
source: "TICK-667 (P6-03, PR #615): the number-format compiler refuses `grow` when a digit touches `{seq:N}` on either side; §3.5 only states the 'after' side (Principal Software Engineer, comment 01a1d13e-68f0)"
deciders: Technical Architect (critique APPROVE, ruling and proposed wording, grow-injectivity spike, comment 01a1d155-4d94); Technical Lead (decision: keep the code's stricter rule, fix the ADR by erratum, comment 01a1d157-d84b; no dissent); Security & Multi-Tenancy Reviewer (PR #615 APPROVE, did not rule on §3.5, comment 01a1d14a-77a5); Solution Architect (author, verdict)
---

# ADR-005 erratum E2: `grow` is refused when a digit touches `{seq:N}` on either side

> **Status: ACCEPTED** (2026-11-13, Solution Architect verdict on TICK-775). The Technical Architect approved the code's rule and proposed this wording; the Technical Lead decided to keep the code and fix the text. No dissent; no CTO tie-break was needed.
>
> **Invariants source.** The project description defines no invariants; checked against [[ADR-001-platform-blueprint]] §1, I1–I9, as ADR-005 is.

## What was wrong

ADR-005 §3.5 says `grow` is allowed when *every variable-width token is followed by a literal outside its character set, or ends the number*. Read literally, that only constrains what comes **after** `{seq:N}`. So `{prefix}{yy}{seq:3}` with `grow` passes (`{seq}` ends the number, `{prefix}` is fixed per type), and so does `{prefix}-{yyyy}{MM}{seq:4}`.

The compiler merged by TICK-667 (PR #615, `internal/records/numbering/template.go`, `unambiguous()`) also refuses a digit directly **before** `{seq:N}`, as TICK-667's acceptance criterion asks. Text and code disagree, and the difference is user-visible: the S18 admin API (TICK-672) returns this 422 to the person publishing a format.

## Correction

**§3.5, second sentence** reads, from this erratum on:

> `grow` is allowed only when the number reads unambiguously without the template: no token or literal that renders a digit is directly before or directly after `{seq:N}`, and every other variable-width token is followed by a literal outside its character set or ends the number. `{field:br}{seq:3}` and `{prefix}{yy}{seq:3}` are refused with `number_format_invalid`; use `refuse`.

| # | Decision |
|---|---|
| **E2-1** | **The code's rule is the rule.** A date part (`{yyyy}` `{yy}` `{MM}` `{dd}`), a `{field:…}` token, or a literal whose adjacent character is a digit, directly before or after `{seq:N}`, makes `grow` fail at publish with `number_format_invalid` (422), `field: overflow`, the byte position of `{seq:N}`, and fixed reason text (`grow needs a non-digit before {seq:N}; use refuse` / `… after …`). PR #615 is unchanged. |
| **E2-2** | **Unchanged formats.** The default `{prefix}{seq:7}`/`grow` (epoch 0) is still accepted: `{prefix}` renders letters. `{prefix}-{seq:4}`, `{prefix}-{yyyy}-{seq:4}` and `{prefix}/{yy}/{seq:2}/{MM}` stay valid with `grow`. |
| **E2-3** | **Newly refused under `grow`; publish them with `refuse`.** `{prefix}{yy}{seq:3}`, `{prefix}-{yyyy}{MM}{seq:4}` (the §3.1 example `VISA-2026110001`), `{prefix}-{yyyy}{MM}{dd}{seq:2}`, `{prefix}2{seq:1}`. The §3.1 examples are valid templates; read with E2, the date-then-sequence ones need `overflow: refuse`. |
| **E2-4** | **What the rule is for.** It is a **readability** rule: a person or an external system can split the number into its parts without knowing the template. It is not what keeps numbers unique in S17 (see Evidence). Uniqueness across epochs is `Disjoint`'s job (§3.3); the records unique index stays the backstop. |
| **E2-5** | **Error catalogue.** TICK-672 adds `number_format_invalid` with this reason text to `api/errors.yaml` together with the admin API, and the admin UI shows "use `refuse`" as the fix. |

**Why adopt the stricter rule rather than relax the code.** Relaxing a publish rule later is always safe: no published format and no issued number becomes invalid. Tightening it after tenants have published formats strands those formats. No format has been published yet (custom formats stay off until M3 and S18), so this is the cheapest moment to be strict.

**Not changed:** §3.1 (grammar, leading-`{prefix}` rule), §3.3 (epochs, disjointness), the `refuse` behaviour (`number_space_exhausted`, 409, alert at 90 %), and I1, I4, I7, I8 as in ADR-005 §7. The rule is evaluated at publish only, in the compiler; it adds nothing to the create path.

## Rejected

| Option | Why |
|---|---|
| Keep §3.5 as written and relax the code to "after only" | Accepts `{prefix}{yy}{seq:3}`/grow, which renders `INC26999` then `INC261000`: unique, but unreadable without the template (is that year 26, counter 1000, or year 261?). Contradicts TICK-667's acceptance criterion. And loosening is reversible later; tightening is not. |
| Drop the grow rule entirely for S17 (it is not needed for uniqueness) | Same readability problem, and the "after" half becomes load-bearing for uniqueness once `{field:…}` compiles in S19 (`{field:br}{seq:3}`); removing it now means re-adding a stricter rule after formats exist. |

## Accepted consequence

Not a security or tenant risk, so no row in `docs/security/accepted-risks.md`. A date-then-sequence format (`{prefix}-{yyyy}{MM}{seq:4}`) must use `refuse`, so when its counter exceeds the width inside one scope (10,000 records in a month for `{seq:4}`), creates of that type fail with `number_space_exhausted` (409) until an admin publishes a wider format (a new epoch). The 90 % alert from §3.5 gives warning. Revisit if a tenant asks for date-then-sequence numbers that grow; the fix then is a separator (`{prefix}-{yyyy}{MM}-{seq:4}`), which `grow` accepts.

## Evidence

- **The code:** `unambiguous()` in `internal/records/numbering/template.go` (PR #615, head `12c587e`), described in `docs/architecture/number-format-compiler.md` step 5.
- **Superset, so safe:** every template §3.5 refuses, the compiler also refuses; the compiler additionally refuses digit-before cases (Technical Architect ruling, TICK-667 comment 01a1d155-4d94).
- **Grow is injective either way in S17** (Technical Architect spike, rerun on TICK-775 with identical output): five templates with a digit before or after `{seq:N}`, 1,259,690 allocations each, every one past its width, **0 clashes**. `docs/adr/ADR-005/spikes/technical-architect/e2-grow-injectivity/`.
- **Disjointness unaffected:** `TestDisjointProperty` (about 45k pairs) and `FuzzDisjoint` (2.87M execs locally) pass; the Technical Architect's four mutants of `disjoint.go` were all killed (same comment).
