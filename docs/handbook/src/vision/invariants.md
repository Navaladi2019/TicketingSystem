---
title: "The nine invariants"
section: vision
sources: [docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, docs/cto/risks.md]
adrs: [ADR-001]
status: delivered
---

An invariant is a rule that must always hold. ADR-001 section 1 lists nine, I1 to I9. Later designs cannot break one without a new ADR signed by the product owner. Each page below says the rule in plain words, names the layer that enforces it and lists the tests that prove it.

| ID | Invariant |
|---|---|
| [I1](tenant-wall.md) | Tenant wall in the database |
| [I2](tenant-source.md) | The tenant comes only from a platform-verified source |
| [I3](least-privilege-roles.md) | Least-privilege database roles |
| [I4](one-write-pipeline.md) | One write pipeline with optimistic concurrency |
| [I5](one-authorization-layer.md) | One authorization layer, inside the query |
| [I6](durable-path-outbox.md) | The durable path is deterministic, versioned and entered only through the outbox |
| [I7](idempotent-side-effects.md) | Every side effect is idempotent |
| [I8](contracts-configuration-advisory-ai.md) | Contracts are the boundary; configuration, not code; AI is advisory |
| [I9](bounded-cascades.md) | Cascades are bounded |

## Decisions that are not invariants

Decision **G7** (one shared database, shared tables, forced row-level security, `tenant_id` leading every tenant-table index) is recorded as a decision, not an invariant. Moving a large tenant to its own database is an operator option. Isolation does not depend on the topology: I1 and I3 keep it.

## Known technical risks

The [known technical risks](../../../cto/risks.md) are tracked in the CTO's register, outside the handbook.
