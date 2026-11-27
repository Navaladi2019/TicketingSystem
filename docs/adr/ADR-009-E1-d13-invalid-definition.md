---
type: adr-erratum
id: ADR-009-E1
title: "Erratum E1 to ADR-009: D13's refusal answers the registered code `invalid_definition`, not `invalid_schema`"
status: PROPOSED
date: 2026-11-28
issue: TICK-915
rfc: none (erratum from an acceptance-kit finding; decision thread under TICK-915 comment 01a21f59-6946-7ef9-a477-1ce40545bc7f)
supersedes: none
amends: "[[ADR-009-schema-version-backfill]] D13 and the `publishEntityTypeVersion` row of 'API test scenarios'; RFC-021 §D13 and CI-9 (archived text is not edited, it is read through this erratum)"
source: "TICK-754 S29 acceptance kit (PR #918), criterion 4: 422 and nothing stored, but code `invalid_definition` (evidence `docs/acceptance/S29/evidence/S29-5.json`); TICK-919 PR #932 chose the registered code"
deciders: Technical Lead (raised it, offered both rulings); QA (proposed fixing the text); Technical Architect (critique); Solution Architect (author, verdict)
---

# ADR-009 erratum E1: D13 answers `invalid_definition`

> **Status: PROPOSED** (2026-11-28). Awaiting the Technical Architect critique; the Solution Architect gives the verdict after it.
>
> **Invariants source.** The project description defines no invariants; checked against [[ADR-001-platform-blueprint]] §1, I1–I9, as ADR-009 is.

## What was wrong

ADR-009 D13 says publishing a property with both `default` and `x-nava-unique` answers **422 `invalid_schema`**. No such code exists. The error catalogue (`api/errors.yaml`, `internal/platform/problem/problem.go`) has exactly one code for "this entity definition is not accepted": **`invalid_definition`** (422, *Invalid entity definition*, with `errors[].pointer`). The OpenAPI text for `createEntityType` and `publishEntityTypeVersion` already says every field rule answers `invalid_definition`. The RFC author wrote a code name that was never registered.

The code built by TICK-919 (PR #932, `metadata.CompileNew`) answers `invalid_definition` with pointer `/definition/schema/properties/<name>/default` and stores nothing. The S29 kit's criterion 4 fails only on that string.

## Correction

| # | Decision |
|---|---|
| **E1-1** | **D13 reads:** *Compile refuses `default` + `x-nava-unique` (422 `invalid_definition`, `errors[].pointer` = `/definition/schema/properties/<name>/default`, `rule` `invalid`). Uniqueness checks stored values.* |
| **E1-2** | **API test scenarios, `publishEntityTypeVersion`:** `default` + `x-nava-unique` → 422 `invalid_definition` with that pointer, and nothing stored. The same applies to `createEntityType` (version 1 is compiled by the same `CompileNew`). RFC-021 D13 and CI-9 are read the same way. |
| **E1-3** | **S29 criterion 4** (TICK-915) reads `422 invalid_definition`. The kit (TICK-754, `S29-5.bru`) asserts `invalid_definition` and the pointer. No code change; PR #932 stands. |
| **E1-4** | **No new code is registered.** `invalid_schema` must not be added to `api/errors.yaml`. |

**Why the text moves, not the code.** A client branches on `code`. Every other definition rule on these two endpoints (unknown keyword, bad pointer, duplicate key, `number_prefix` on publish) already answers `invalid_definition`; a separate code for one rule would make clients handle two codes for the same class of mistake, and the pointer already says which rule failed. Keeping the catalogue small is cheaper than a second code with one user.

**Not changed:** the refusal itself (a unique field still cannot have a default), the 422 status, "nothing stored", the D15 rollout skip, and every other decision in ADR-009. I1–I9 are unaffected: this is a response string.

## Rejected

| Option | Why |
|---|---|
| Register `invalid_schema` and change the code (fix task) | Adds a catalogue entry used by one rule, splits one error class across two codes, and costs a code change, a regenerated catalogue and a kit rerun for no client benefit. Nothing ever answered `invalid_schema`, so no client depends on it. |
| Use `invalid_schema` only for D13 but keep it as an alias of `invalid_definition` | Two names for one thing; the Bruno registry (`docs/acceptance/guide/bruno/lib/nava.js`) and `problem.go` would drift. |

## Accepted consequence

None beyond the text. Not a security or tenant risk, so no row in `docs/security/accepted-risks.md`. The response carries a pointer and a rule, never a field value (unchanged from D13).

## Evidence

- **Catalogue:** `api/errors.yaml` (`invalid_definition`, 422); `internal/platform/problem/problem.go:46`. `invalid_schema` appears nowhere outside ADR-009, RFC-021 and their handbook page.
- **Code:** `metadata.CompileNew` (TICK-919, PR #932) adds a problem at `schema/properties/<name>/default`; unit test `internal/metadata/compile_test.go` (CI-10 comment).
- **Fresh-stack run:** S29 kit, stack `nava-s22s29`, 2026-11-27, `docs/acceptance/S29/evidence/S29-5.json`: `422`, `code: invalid_definition`, `errors: [{pointer: /definition/schema/properties/s29_tier/default, rule: invalid}]`; S29.8 confirms the type is still at the prior version (nothing stored).

## Follow-ups

- ADR-009 md and html carry a one-line erratum note at D13 (this PR).
- Handbook: `docs/handbook/src/adr/adr-009.md` line with D13 and the metadata-engine page say `invalid_definition` (Technical Lead, S29 handbook task).
