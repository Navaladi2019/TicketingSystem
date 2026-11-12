---
title: "Glossary"
section: glossary
sources: [README.md, docs/adr/ADR-001-platform-blueprint.md, docs/adr/ADR-001-explained.md, docs/adr/ADR-004-audit-log.md]
adrs: [ADR-001, ADR-004]
status: delivered
---

Terms used across the handbook, including every term that appears in a vision-section heading. Each entry points to the page that explains it most fully. Pages for the security, database and workflow sections are written by other handbook tasks; until they exist, entries point to the section's index.

## Tenancy and identity

- **Tenant.** One customer organisation using the shared platform. Main page: [I1: tenant wall](../vision/tenant-wall.md).
- **Tenant wall.** The row-level-security rule set that makes a query without a tenant return nothing. [I1](../vision/tenant-wall.md).
- **Row-level security (RLS) and FORCE RLS.** A PostgreSQL feature that adds an invisible "only this tenant's rows" filter to every query. `FORCE` applies it even to the table's owner. [I1](../vision/tenant-wall.md).
- **Principal.** The authenticated caller of an API request, a user or a service, resolved from a verified token inside one tenant. [I2](../vision/tenant-source.md).
- **Group.** A set of principals in one tenant, synchronised from the identity provider, used by access rules. [I5](../vision/one-authorization-layer.md).
- **Identity port and adapter.** The core-owned interface to the identity provider and the single package that talks to Keycloak. [I8](../vision/contracts-configuration-advisory-ai.md).
- **Platform-verified source.** A tenant value that comes from a validated token via the identity binding table, or from a row the platform wrote. [I2](../vision/tenant-source.md).

## Database roles

- **Database role and least privilege.** One narrow database account per system actor; none is a superuser, owns objects or bypasses RLS. [I3](../vision/least-privilege-roles.md).
- **Grant manifest.** The single file listing every permission of every role; CI compares the live database with it. [I3](../vision/least-privilege-roles.md).
- **Migrator.** The one-shot job that owns the schema and runs migrations. [Database](../database/index.md).

## Records and writes

- **Entity definition and entity definition version.** A tenant's JSON Schema description of a kind of record, and one published, immutable version of it. [Database](../database/index.md).
- **Record.** One stored item of an entity type, with a `version` used for optimistic concurrency. [I4](../vision/one-write-pipeline.md).
- **Write pipeline.** The single path every write takes: authorisation, defaults, schema validation, checks, commit with audit. [I4](../vision/one-write-pipeline.md).
- **Optimistic concurrency.** An update names the version it expects; a stale one is a typed conflict (HTTP 412 with `If-Match`). [I4](../vision/one-write-pipeline.md).
- **Authorization layer.** The access rules, compiled to SQL and agreeing with the Go single-record check. [I5](../vision/one-authorization-layer.md).
- **ACL and ACL version.** An access-control list for an entity type, and the version number of the published list a decision was made under. [I5](../vision/one-authorization-layer.md).
- **Mask (field mask).** A rule hiding a field from readers who may not see it, on write and on read. [I5](../vision/one-authorization-layer.md).
- **Numbering mode.** How a record's human-readable number is allocated, per tenant, from a counter table. [I8](../vision/contracts-configuration-advisory-ai.md).
- **Idempotency key.** A client-supplied key (`Idempotency-Key`) that makes a repeated write return the first response. [I7](../vision/idempotent-side-effects.md).
- **Idempotent.** Doing it twice has the effect of doing it once. [I7](../vision/idempotent-side-effects.md).

## Audit

- **Audit log.** The per-tenant record of every change, written in the same transaction as the change. [Database](../database/index.md).
- **Seal.** The sequence number and hash assigned to an audit entry by the asynchronous sealer. [Database](../database/index.md).
- **Chain head.** The latest sealed entry of a tenant's audit hash chain; each new seal hashes the previous one. [Database](../database/index.md).

## Workflows

- **Outbox.** A table of work rows saved in the same transaction as the change that caused them. [I6](../vision/durable-path-outbox.md).
- **Relay.** The process that reads outbox rows and starts or signals workflows. [I6](../vision/durable-path-outbox.md).
- **Temporal and workflow.** The workflow engine and a durable process it runs. Workflow code is deterministic; side effects run in activities. [I6](../vision/durable-path-outbox.md).
- **Causation and cascade.** The origin event id and depth carried by every write and outbox row; a cascade is a write that triggers further writes, bounded by a depth limit. [I9](../vision/bounded-cascades.md).

## Process words

- **Invariant.** A rule that must always hold; changing one needs a new signed ADR. [The nine invariants](../vision/invariants.md).
- **Decision (D and G numbers).** A choice recorded in ADR-001, such as D26 or G7, that is not an invariant. [Goals and non-goals](../vision/goals-and-non-goals.md).
- **Goal and non-goal.** What Milestone 1 sets out to do, and what it deliberately leaves out. [Goals and non-goals](../vision/goals-and-non-goals.md).
- **Configuration, not code.** Tenant changes are data, never new code or schema changes. [I8](../vision/contracts-configuration-advisory-ai.md).
- **Advisory AI.** AI may suggest, and change data only above a recorded confidence threshold. [I8](../vision/contracts-configuration-advisory-ai.md).
- **Proof and enforcement point.** The test or ADR section that shows an invariant holds, and the layer that applies it. [The nine invariants](../vision/invariants.md).
- **Known technical risks.** The CTO's register of live risks. [Vision](../vision/index.md).
- **ADR.** Architecture Decision Record. [Architecture decisions](../adr/index.md).
