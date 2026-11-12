---
title: "Accepted-risk register"
section: security
status: delivered
sources: [docs/security/accepted-risks.md]
---

Every risk the project has knowingly accepted. This page is rendered at build
time from [`docs/security/accepted-risks.md`](../../../security/accepted-risks.md),
which stays authoritative: the rows below are copied as written, and only
their relative links are rebased so they open from the handbook. Each layer
page in this section lists the rows that apply to it.

## Severity

| Flag | Meaning |
| --- | --- |
| 🔴 High | A tenant-isolation break if one precondition fails (for example, one stolen credential) |
| 🟠 Medium | Data exposure or a weakened control, but needs more than one failure, or the control is in doubt |
| 🟡 Low | Availability, cost or operability; no tenant data at risk |
| ⚪ Info | Process cost or a known limitation |

A severity marked † was assessed when this register was built (2026-10-24),
because the source gives none. An unmarked severity is the source's own rating.

Status: **Accepted** (signed off as is), **Fix proposed** (a cheap fix is
written down; prefer fixing over keeping it), **Control disputed** (the
recorded mitigation does not hold as written; re-decide).

## Register

<!-- gen:accepted-risks -->

## Adding a row

A decision, ADR, review or doc that accepts a new risk adds an
`AR-<story>-NN` row to `docs/security/accepted-risks.md` in the same change; a
fix deletes the row and notes the fix at the source. Rebuild the handbook
with `task handbook`.
