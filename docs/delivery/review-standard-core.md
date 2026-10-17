---
type: review-standard
subsystem: core (nava-core, platform packages, background binaries, temporal-server)
owner: Technical Lead
reviewed-by: Technical Architect
source: TICK-89 (how the OpenTelemetry gap got past the T01 review), TICK-93
date: 2026-10-16
---

# Review standard: core

These rules bind every review of a PLAN-001 or later task in the core subsystem, whoever the reviewer is. They add to the `change-review` skill and to PLAN-001 D-P10 (security paths); they do not replace either.

## Why this exists

The T01 review (TICK-89) checked the code against its spec and found a real bug inside that frame. It never checked the frame. The spec fixed a telemetry stack, an inbound-trace rule and two metric names that later tasks (Temporal, pgx, the relay) would have had to retrofit, and the review accepted a new platform-wide dependency on one reviewer's say. Both gaps are closed here.

## Rule 1: spec challenge first

Before reviewing code against a task's spec, check the spec itself:

1. **Consumers.** List the PLAN tasks that consume what this change produces: a library, a protocol or wire format, a metric or attribute shape, a config key, a table or column, an error code. For each one, does the spec's choice fit how that task will use it?
2. **Trust boundaries.** Which boundary does the change touch: the public listener, tenant data, the identity port, Temporal certificates, the database roles? Does the spec say what crosses it and what must not (raw paths, tenant or principal ids, emails, baggage, tokens)?
3. **Invariants.** Tenant id only from the validated token, validation before every write, no unbounded label or attribute, no globals.

If the spec is the problem, say so in the verdict and route it to the spec's owner (the Technical Lead for PLAN tasks). Do not block the implementer for building what was asked.

When the spec itself fixes a cross-cutting choice, rule 2 wins: `VERDICT: CHANGES`, with the choice routed to the Technical Architect and the implementer recorded as not at fault.

**Every verdict states the result**, as one line before the findings:

```
Spec challenge: done. Consumers checked: <tasks> | none (<reason>). Boundaries: <list> | none (<reason>). Result: no conflict | conflict → <issue>.
```

A verdict without this line is incomplete, and the Technical Lead sends it back.

## Rule 2: new dependencies and cross-cutting choices go to the Technical Architect

A **cross-cutting choice** is any choice that other tasks will inherit:

- a new direct Go module or container image (a `require` without `// indirect`, a service in compose), or a major-version change of one; indirect requires and patch/minor bumps of pinned modules are not choices unless `pins.md` marks the target version known-broken;
- a protocol, wire format or propagation rule (headers, trace context, cursor or token format);
- a new metric, span or log attribute key, or a shape outside the telemetry row below, that other code or dashboards will read; a metric whose name and bounded attributes the task spec already states is not a new choice;
- a shared construction point (HTTP client, DB pool, telemetry providers, router);
- a listener, port or network placement.

The Technical Architect stress-tests the choice before the Technical Lead accepts it. In practice:

- The implementer names the choice in the PR description (`New dependency:` / `Cross-cutting choice:`), with the version and the reason.
- The reviewer does not approve a PR with an unexamined choice. The verdict either links the Technical Architect's comment that accepted it, or is `VERDICT: CHANGES` with the choice routed to the Technical Architect.
- The reviewer reads `git diff <base> -- '**/go.mod' 'compose*.y*ml'` rather than relying on the PR description alone. Once T05 lands, CI fails on any direct require that is not in `pins.md`.
- An accepted dependency is added to `docs/pins.md` with its reason in the same PR; until T05 creates the file, the PR description and the accepting TICK-94 comment are the record, and T05 copies them in.
- Choices already decided in ADR-001, in a PLAN-001 decision (D-Pxx) or in the decision log need no new stress test; cite the decision. A choice that appears only in a task description is not a decision: it goes through this rule even if the PLAN fixed it.

## What a core review checks

| Area | Expectation |
|---|---|
| Telemetry | Providers passed explicitly, no `otel.Set*` globals. Metric attributes bounded: never tenant, principal, workflow or request ids, never raw paths or client-supplied strings. Spans pass the attribute filter on server **and** client side. |
| Public listener | Everything on it is treated as external: new trace root, inbound `traceparent` only as a link, no inbound baggage. |
| Errors | Never echo values (tokens, emails, ids from the request). |
| Tenant data | Through `tenancy.InTenantTx` only, cross-tenant negative test present. |
| Tests | Risky paths have a failing-case test; integration tests against real PostgreSQL, Keycloak or Temporal where the risk lives there (`testing-ci`). |
| Size | One task per PR; a rewrite found in review is a new task unless the human says otherwise. |

## Change log

| Date | Change | By |
|---|---|---|
| 2026-10-16 | Created with rules 1 and 2 after TICK-89. | Technical Lead |
| 2026-10-17 | Technical Architect review (TICK-95) applied: rule 2 exemption limited to recorded decisions (R2-1); definitions narrowed to direct requires and new attribute keys; diff check and pre-T05 pin record; rule 2 wins over rule 1; `none (<reason>)` form. | Technical Lead |
