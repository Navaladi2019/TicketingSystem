---
title: "Goals and non-goals"
section: vision
sources: [README.md, docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md]
adrs: [ADR-001]
status: delivered
---

This page follows ADR-001 and the README, and the Milestone 1 scope as recorded in the ADR-001 reading guide. It adds no goals of its own.

## Goals

- **Keep tenants apart.** One tenant can never read or change another's data. The [nine invariants](invariants.md) are the rules that guarantee it.
- **Know the tenant from a verified source only** (decision D26): the identity provider's token, resolved through a platform lookup table.
- **One predictable API** (decision D6): REST at `/api/v1`, generated from an OpenAPI contract, with stable error codes, idempotency keys and optimistic concurrency.
- **Reliable background work** (decision D22): saved changes start workflows through a transactional outbox and a relay, with workflows kept inside their tenant.
- **Narrow database accounts** (decision D28): one account per system actor, none able to skip the tenant wall.
- **Third-party systems behind ports and adapters**, so a vendor can be replaced without touching the core (README, design principles).

## Non-goals for Milestone 1

ADR-001 and its reading guide list what is deliberately left out, because each item adds a path that is not yet secured:

- Temporal Schedules, cron and reset.
- Partitioned tables.
- Tenant self-service administration of the identity provider (residual R3); the platform API does it.
- GraphQL. It arrives later, as a read surface over the same authorization layer.
- Encrypting workflow histories per tenant (residual R1). Workflow payloads carry references only.

## Planned, not on `main`

Stories S12, S18, S19 and S20 are planned and are not described anywhere in this handbook yet; the [stories section](../stories/index.md) lists them.
