---
type: adr
id: ADR-NNN
title: <the decision in one line>
status: PROPOSED
date: YYYY-MM-DD
issue: TICK-NNN
rfc: RFC-NNN (docs/adr/ADR-NNN/rfc/RFC-NNN-<slug>.md)
supersedes: none
---

<!-- Template for ADRs, amendments (ADR-NNN-A<k>-<slug>.md) and errata (ADR-NNN-E<k>-<slug>.md).
     Copy to docs/adr/ and fill in. Two pages at most. Rules: design-docs and design-triad skills, ADR-008.
     This file starts with "_" so the handbook build does not index it. -->

# ADR-NNN: <title>

**Status: PROPOSED.** <critique verdicts, tie-breaks>

> **Invariants source.** <where I1–I9 were read from>

## In plain words

- **What changes:**
- **Why:**
- **What you'll see when it's done:**
- **Codes used below:**

## Decision

| # | Decision |
|---|---|
| **D1** | |

## Context it rests on

## Consequences

## API test scenarios

<!-- For every API operation added or changed: each documented status, invalid input writes nothing, idempotent replay writes once, cross-tenant refused. Delete if none. -->

| Operation | Scenarios |
|---|---|

## Risks and residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|

## Rejected options

| Option | Why |
|---|---|

**Dissent:** <recorded dissent, or "none">

## Competitive Gate (ADR-008)

<!-- Mandatory (ADR-008 D6). Axes X1a, X1b, X2, X3, X4 and methods M-…: ADR-008 "The axes". Pain-point IDs: docs/adr/competitive/pain-points.md.
     A multiplier ("10x") only when both sides were measured on the same quantity (D4). Limits: apply the D11 five-point test. -->

**Result:** PASS | PASS with accepted risk <CP-id> | N/A
1. **Axis moved:** X<n> <name>: from <today, number> to <after, number>, or "none".
2. **Pain points:** avoids <CP-ids, one clause each>. Risks repeating <CP-ids>: accepted reason, owner, revisit trigger, or "none".
3. **Measured by:** M-X<n> | spike <id> | acceptance check <id> ("planned" if not yet in the repo).
4. **Parity:** I1–I9 unchanged, or the invariant this ADR strengthens.

<!-- N/A form, for internal ADRs only (test tooling, CI, internal mechanics with no tenant-visible effect):
**Result:** N/A. <one line on why no axis, limit, API, configuration surface or upgrade path changes>. Parity: I1–I9 unchanged. -->
