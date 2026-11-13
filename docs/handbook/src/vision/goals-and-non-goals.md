---
title: "Goals and non-goals"
section: vision
sources: [docs/product/vision.md, docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md]
adrs: [ADR-001]
status: delivered
---

The goals and non-goals come from [the product vision](../../../product/vision.md). ADR-001 and the invariants show how the platform meets the first ones.

## Goals

- **Model a new kind of case through configuration alone.** A tenant adds a record type with no code change and no schema change. Delivered for fields, access rules, masks and numbering; see [what the product is](index.md).
- **Keep tenants apart.** One tenant can never read or change another tenant's data. The [nine invariants](invariants.md) are the rules that guarantee it.
- **Audit every change.** An auditor can prove the export is complete and untampered. Delivered: see [audit](../database/audit.md).
- **Start a workflow exactly once.** A saved change starts its workflow once, even after retries and failures, through a transactional outbox and a relay. The outbox path is delivered; creating a record does not yet start a workflow (S11b, in progress).
- **Be embeddable.** Other applications embed the platform through a widget or an API and SDK. Not built yet.

## Non-goals

- **A fixed ITSM, lending or visa product.** These are examples of tenant configuration, not features.
- **Tenant code inside the platform.** Extensions go through the public API with a scoped token, never with direct database or workflow access (ADR-001, I8).
- **AI that changes data on its own** below an explicit confidence threshold (ADR-001, I8).

## Decisions that shape the platform

- The tenant is known from a verified source only (D26): the identity provider's token, resolved through a platform lookup table.
- One predictable REST API at `/api/v1` (D6), generated from an OpenAPI contract, with stable error codes, idempotency keys and optimistic concurrency.
- Narrow database accounts (D28): one per system actor, none able to skip the tenant wall.
- Third-party systems sit behind ports and adapters, so a vendor can be replaced without touching the core.

## Left out of Milestone 1

ADR-001 and its reading guide list what is deliberately left out, because each item adds a path that is not yet secured:

- Temporal Schedules, cron and reset.
- Partitioned tables, except `audit_log` (ADR-004). General partitioning is Epic 30.
- Tenant self-service administration of the identity provider (residual R3); the platform API does it.
- GraphQL. It arrives later, as a read surface over the same authorization layer.
- Encrypting workflow histories per tenant (residual R1). Workflow payloads carry references only.

## Planned, not on `main`

Stories S12, S18, S19 and S20 are planned and are not described in this handbook; the [stories section](../stories/index.md) lists them.
