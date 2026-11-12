---
title: "Vision"
section: vision
sources: [README.md, docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, docs/cto/risks.md]
adrs: [ADR-001]
status: delivered
---

## What the product is

The platform (`nava-core`) is a multi-tenant ticketing platform. Many customer organisations, called tenants, share one installation, one database and one set of services. The main risk of that design is one tenant seeing or changing another tenant's data, so the architecture is built around keeping tenants apart. ADR-001 records the ground rules; see [goals and non-goals](goals-and-non-goals.md).

## Who uses it

- **Tenant users and applications** call the REST API at `/api/v1` with a token issued by the identity provider.
- **Tenant administrators** publish entity types, set access rules and manage users and groups through the platform API.
- **Platform operators** onboard tenants, run migrations and administer the Temporal workflow engine.
- **Engineers and the product owner** build and accept the platform; this handbook is for them.

## Read next

- [Goals and non-goals](goals-and-non-goals.md)
- [The nine invariants](invariants.md), one page each: [I1](tenant-wall.md), [I2](tenant-source.md), [I3](least-privilege-roles.md), [I4](one-write-pipeline.md), [I5](one-authorization-layer.md), [I6](durable-path-outbox.md), [I7](idempotent-side-effects.md), [I8](contracts-configuration-advisory-ai.md), [I9](bounded-cascades.md)
- [Glossary](../glossary/index.md)
- [Known technical risks](../../../cto/risks.md), kept by the CTO outside the handbook
