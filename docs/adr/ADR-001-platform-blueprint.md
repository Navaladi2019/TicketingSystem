---
type: adr
id: ADR-001
title: Platform blueprint — invariants, D26, D6, D22, D28 (system actors)
status: ACCEPTED (approved by nava 2026-10-15, TICK-36 comment 01a14052-5768)
date: 2026-10-15
accepted: 2026-10-15
issue: TICK-36
rfc: "[[RFC-001-platform-blueprint]] (archived; thread on TICK-36, rev 1 + critique rounds 1–2 + CTO rulings)"
supersedes: none
---

# ADR-001: Platform blueprint

**Status: ACCEPTED** on 2026-10-15. nava replied `APPROVED`, accepted residuals R1, R2, R3, R7, R8 and R9, and accepted the CTO's ruling to keep Keycloak behind a thin identity port (Annex A.4). Changing an invariant or a decision here now needs a superseding ADR signed by nava.

**Changes on acceptance:** the status above, Annex A.4 (identity port), the Zitadel row in §4, the consequence line in §3, and CI-11. No decision changed.

The body is the decision. **Annex A** holds the binding contracts that Epics 02, 03, 07 and 08 implement against. **Annex B** is the acceptance tests.

> **D-number note.** This ADR uses the TICK-3 numbering, where **D28** is privileged and system-actor access. The TICK-2 gap list numbered differently: its D28 was the real-time channel and its D29 operator access. Human support access is **TICK-40** (D28b).

## 1. Invariants

These are I1–I9, citable by later ADRs.

| # | Invariant | Source |
|---|---|---|
| **I1** | **Tenant wall in the database.** Every tenant table has `tenant_id`, `ENABLE` and `FORCE ROW LEVEL SECURITY`, and a tenant policy on `NULLIF(current_setting('app.tenant_id', true), '')::uuid`. With no tenant context, a query gets no rows or an error, never another tenant's rows. Unique keys and foreign keys include `tenant_id`. A policy that does not test `app.tenant_id` must be on the named allow-list (Annex A.3). | P2, P19, §5 |
| **I2** | **The tenant comes only from a platform-verified source.** That means a validated token resolved through `tenant_identity_binding` (D26), or a row the platform wrote under RLS (`outbox.tenant_id`, D22). Client values, payloads, script arguments, workflow inputs, signal bodies and search attributes are never a source. Temporal is a path like any other (G2). | P2; G2 |
| **I3** | **Least-privilege database roles.** No runtime role is a superuser, has `BYPASSRLS`, owns an object, or is a member (direct or inherited) of any role, `pg_*` included. No object grants to `PUBLIC`, including default ACLs. Every grant comes from one manifest. No `SECURITY DEFINER` function or RLS-bypassing view is reachable from a runtime role (G9). | P2, P19; TICK-3; G9 |
| **I4** | **One write pipeline with optimistic concurrency.** Every writer goes authorisation → defaults/computed → JSON Schema → declarative checks → commit with audit in one transaction. Every update carries the expected record `version`, and a stale write is a typed conflict, never last-writer-wins (G4). | P8, §2.4; G4/D24 |
| **I5** | **One authorization layer, inside the query.** ACLs compile to SQL, and the Go single-record check agrees with them. Field masks apply on write and read. Unreadable, other-tenant and missing records give the same response. | P19, §2.12 |
| **I6** | **The durable path is deterministic, versioned, and entered only through the transactional outbox.** All I/O, clocks, randomness and tenant code run in activities. In-flight executions keep their version. Nothing starts or signals a workflow except the relay dispatching an outbox row committed with the change that caused it (G1). | P7, P9; G1 |
| **I7** | **Every side effect is idempotent.** A retry, replay or redelivery never acts twice. | P10 |
| **I8** | **Contracts are the boundary. Configuration, not code. AI is advisory.** Tenant code, plugins and AI reach data only through the capability contract or the public API, with a scoped token, and never reach the database or Temporal directly. A tenant change never needs DDL. The **only** tenant-motivated DDL is an operator-approved, measured, audited *indexed attribute* (`CREATE INDEX CONCURRENTLY … WHERE tenant_id = …`). It runs as a migrator-class job outside the tenant's change set (D25). Per-tenant numbering uses a counter table, not sequences. AI changes data only above an explicit confidence threshold, and the decision is recorded. | P1, P3, P14, P16, P20, P21; D25 |
| **I9** | **Cascades are bounded (new, G5).** Every write and every outbox row carries a causation record: origin event id plus depth. The pipeline rejects any write beyond a hard platform depth limit, and rejects a cycle (the origin seen again on the same record), with a typed error and an audit entry. The limit is platform configuration, not tenant configuration. Its initial value is set at Epic 05 planning. | P10, P2 (quota exhaustion is a cross-tenant effect); G5 |

### TICK-2 gap coverage

| Gap | Where it is handled |
|---|---|
| **G1** outbox is the only durable bridge | **I6**, plus the D22 start path (Annex A.2) |
| **G2** Temporal tenancy | **I2** and **D22**. TICK-2's per-tenant payload codec is **deferred** as residual R1, with a trigger. |
| **G4** version contract (D24) | **I4**, implemented by the D6 contract (`version`, ETag/`If-Match`, `412 version_conflict`). Workflow activities re-read and retry. D24 needs no separate RFC unless nava wants one. |
| **G5** causation depth | **New I9.** `outbox` gains `causation_id` and `causation_depth` (Annex A.3). |
| **G7** database topology | **Recorded as a decision, not an invariant:** shared database, shared tables, forced RLS, and `tenant_id` leading every tenant-table index. Moving a large tenant to a dedicated database (same schema, roles and RLS) is an operator option under **D1**. *Why it is not an invariant:* topology is a performance and deployment choice that can change without weakening isolation. I1 and I3 are what keep isolation. |
| **G9** no `BYPASSRLS` in the application | **I3.** Even backup does not use it; it uses a `REPLICATION`-only role (Annex A.3). Human support access is **TICK-40**. |
| **D25** indexed-attribute governance | **I8** clarifies "no DDL". The approval workflow, measurement and quotas stay with **D25**. M1 planning does not need it unless Epic 06's 150 ms list measurement shows an expression index is required. |

## 2. Decisions

| # | Decision | What it prevents |
|---|---|---|
| **D26** | One shared Keycloak realm per installation, on **Keycloak ≥ 26.1.3** (recommended 26.8.1), with **Organizations**, one organization per tenant. The tenant is resolved from `(iss, organization.id)` through `tenant_identity_binding`. The token must name exactly one organization, otherwise 401. Tenant-linked IdPs use `trustEmail=OFF` and the first-broker-login flow `idp-create-user-if-unique` → `idp-add-organization-member`. The resolver checks `idp` and the user's federated links against the tenant's bound IdPs, and **fails closed**. Invitation and add-member reject a user federated to a foreign IdP. Only platform service accounts administer Keycloak. On-premise has its own installation and realm. | A tenant admin minting claims. The `kc.org` picker tamper on 26.0.0–26.1.2 (spike-proven). Account takeover and pre-hijack through a tenant-controlled IdP. |
| **D6** | REST-first `/api/v1`, from an OpenAPI 3.1 contract and additive-only within a major version. RFC 9457 errors with a stable `code`. Opaque HMAC cursors bound to tenant, principal and query. `Idempotency-Key` on writes. ETag/`If-Match` on updates. Per-tenant rate limits. GraphQL arrives later (Epic 23) as a read surface over the same authorization layer. | Two authorization surfaces in M1. Lost updates (G4). Duplicate writes. |
| **D22** | One Temporal namespace `nava` per installation. **M1 requirement:** frontend mTLS with a custom deny-by-default `Authorizer` and `ClaimMapper` per certificate identity, which **also inspects worker commands**. Internode mTLS, and internal ports unreachable from the platform network. Only the relay starts and signals workflows, from outbox rows. Workflow ids are `t:<outbox.tenant_id>:<target_key>`, started with `REJECT_DUPLICATE` and error-on-started. The tenant travels through a `ContextPropagator`, with inbound and outbound guards. Signals are FIFO per target on `target_seq`, assigned under the record's row lock. Payloads carry references only. No Schedules, cron or reset in M1. | Forged starts and signals by any process or certificate. Cross-tenant child, signal and cancel calls from a compromised worker (A7). Duplicate runs. Approve and cancel swapping order. |
| **D28** | One database role per system actor (Annex A.3). No role has `BYPASSRLS`, owns anything, belongs to any role, or inherits from `PUBLIC`. Cross-tenant access goes only through allow-listed, role-scoped policies or a separate database. Backups are physical, through a `REPLICATION`-only role that `pg_hba` keeps to the `replication` pseudo-database. The migrator owns everything, runs as a one-shot job and is granted to no one. | The first cross-tenant job shipping with `BYPASSRLS`. The `SET ROLE` path to the owner (TICK-3 TA point). `PUBLIC` and default-ACL leaks. Logical-decoding exfiltration by the backup role. |

**Fixed inputs, not reopened:**
- PostgreSQL ≥ 15
- PgBouncer ≥ 1.21 with `max_prepared_statements > 0`
- Temporal stays on `default` until this ADR is accepted; Epic 08 then creates `nava`
- Keycloak "26 or later" from TICK-4 is narrowed to **≥ 26.1.3**, which only excludes a proven-broken range (CTO ruling)

## 3. Consequences

- **We own a custom Temporal server build.** It carries the Authorizer and ClaimMapper, as a **separate Go module** from the SDK code (R7), and is re-verified on every Temporal upgrade.
- **Every cross-tenant database path is enumerable.** CI (Annex B) fails on any new one.
- **Keycloak is administered only through the platform API** in M1 (R3).
- **Only the identity port talks to the IdP.** `IdentityResolver` and `DirectoryAdmin` (Annex A.4) are the only code that knows Keycloak. They are built in Epic 03. A later IdP switch is a contained, spike-gated adapter change, not a rewrite.
- **Per-target FIFO costs about 11 µs and +23% on outbox inserts,** plus one partial index (8.8 MB at 50k pending rows). The normal claim is 5.4 ms against 5.0 ms without it. The worst case must be bounded (Annex A.2).

**Pins for the Technical Lead to commit at planning** (TA spike text, verbatim):

```
keycloak: >= 26.1.3 (recommended 26.8.1, tested 2026-10-14)
  known-broken: 26.0.0?–26.1.2. Org-selection form accepts kc.org for an org the
  user is not a member of, and the token carries that org's id. Tested vulnerable:
  26.0.7, 26.0.8, 26.1.0, 26.1.2. Fixed: 26.1.3+. Spike: TICK-36 TA critique round 1.
keycloak realm export: organization scope mapper addOrganizationId=true,
  jsonType.label=JSON, addOrganizationAttributes=false
temporal server: 1.28.4 tested (SQL visibility on PostgreSQL 16, Keyword search attribute)
go.temporal.io/sdk: v1.49.0 tested; requires Go >= 1.26
temporal custom server module: go.temporal.io/server v1.28.4 + go.temporal.io/api v1.50.1, Go >= 1.25.8,
  built as a separate Go module from SDK code (SDK v1.49.0 needs api v1.63.5)
```

## 4. Rejected options

| Option | Why it lost |
|---|---|
| Zitadel instead of Keycloak (TA assessment on TICK-36, CTO ruling) | No native groups, and Epic 03 syncs groups and the org hierarchy. AGPL-3.0, while we ship the IdP on-premise; Keycloak is Apache-2.0. Our tenant-isolation spikes (A1, A2, A6, TA-B1) were run on Keycloak, and Zitadel's tamper behaviour is untested. It is stronger on per-tenant self-administration (R3) and footprint. Revisit triggers are in Annex A.4. |
| Realm per tenant | Every tenant admin is a realm admin. It needs a JWKS and issuer per tenant, and config drifts. Kept as an escape hatch through the binding table. |
| Tenant id as a user attribute or claim | An editable attribute becomes a forgery path (I2). |
| Brokered auto-link by verified email | Cross-tenant account takeover. |
| GraphQL-first, or REST and GraphQL equal in M1 | Two authorization surfaces, plus cost governance, before Epic 06 is proven. |
| Namespace per tenant | Pollers grow with tenant count, onboarding needs an admin operation, and persistence stays shared anyway. Revisit with D23. |
| Tenant in workflow input or a search attribute | Caller-shaped. The spike showed the search attribute can be changed from workflow code. |
| Network isolation of port 7233 alone, or the default JWT authorizer | Neither can stop a valid certificate, or separate start from poll by API. |
| API-level Authorizer without command inspection | A7: a worker certificate created and signalled workflows in tenant B through `RespondWorkflowTaskCompleted`. |
| Outbox `id` as Temporal request id | Not exposed by Go SDK v1.49.0. |
| FIFO ordered by UUIDv7 `id` | `id` is generated before the lock, so approve and cancel can swap. |
| `nava_system` role with `BYPASSRLS` | One leaked credential reads every tenant. |
| Backup with `BYPASSRLS` and `pg_dump` | Arbitrary SQL across tenants. Kept as the fallback only if a customer requires logical dumps, with its credential on no runtime host. |
| NOLOGIN group roles, or `ALTER DEFAULT PRIVILEGES … GRANT` | Harder to audit, and they grant silently. |
| `SECURITY DEFINER` functions | They bypass RLS invisibly. Banned by Epic 02. |

## 5. Residuals (accepted by nava, 2026-10-15)

| # | Residual | Mitigation | Owner | Revisit trigger |
|---|---|---|---|---|
| **R1** | Temporal histories are cross-tenant data and are not encrypted in M1 | References-only payloads (CI-7). Separate database and role. mTLS everywhere. Operator-only UI. | Solution Architect | The first regulated-data tenant, or TICK-41 decided. Then a per-tenant encrypting codec. |
| **R2** | The migrator is all-powerful over its tables. The backup role and the backup set hold every byte, including Keycloak and Temporal if they share the cluster. | One-shot migrator job. Post-run catalogue check. `pg_hba` reject line for `nava_backup`. The WAL archive and base backups get the same access control as the primary. | Technical Lead | Any request to run DDL from a runtime process, or a logical-dump requirement |
| **R3** | No tenant self-administration of Keycloak in M1 | The platform API does it | CTO / nava | Epic 26 scoping |
| **R7** | A custom Temporal server build to own | A minimal Authorizer module, pinned, with a CI test of the allow-list | Technical Lead | Every Temporal upgrade |
| **R8** | A compromised `nava-relay` or `nava-core` certificate can start or signal for any tenant. A compromised **`nava-worker`** certificate polls every tenant's tasks and histories from the shared queue, and its activities can set any `app.tenant_id`, so it reads and writes any tenant under RLS. The Authorizer stops only its cross-tenant *commands*. | Separate credentials and pools. The relay reads the tenant only from `outbox`. Inbound guards. Command inspection. `set_config` only inside the database helper (lint rule). | Security reviewer | Any new holder of a Temporal certificate, or a requirement for per-tenant workers |
| **R9** | Head-of-line blocking per target while a signal backs off | It blocks only that target. `NotFound` or "already completed" is terminal at once. A dead row unblocks the target and raises an alert. The worst-case bound is in Annex A.2. | Technical Lead | Epic 08 flood test |

**Dissent: none recorded.** All three reviewers (Security, Postgres, TA) approved. TA-B3 was settled by a CTO ruling on a verified spike.

---

## Annex A: binding contracts

### A.1 D26: tenant resolution

**1. Validate the token.**
- Signature from the JWKS of the realm named by `iss`. A key from realm X with `iss=Y` is rejected.
- `alg` allow-list, `aud`, and `exp`/`nbf` with ±60 s skew.
- `typ`/`azp`, so an ID token is rejected.

**2. Read the organization.** The `organization` claim is an object keyed by alias. Exactly one key is required, otherwise **401 `tenant_unresolved`**. Read its `.id`.

**3. Check the identity provider.** This is fail-closed and needs both checks:
- If the token has an `idp` claim, it must be one of the tenant's bound IdP aliases.
- Independently of the claim, the user's federated links (recorded on `principals` by group sync from the Keycloak admin API) must all be IdPs bound to the tenant. If either check fails, or the links are unknown for a brokered user, the result is 401.

**4. Resolve the tenant.** Look up `tenant_identity_binding(issuer, org_id, idp_aliases[])`, which has `UNIQUE (issuer, org_id)`. More than one row is an error.

**5. Set the tenant for the transaction.** `set_config('app.tenant_id', $1, true)`, per transaction, and only inside the database helper.

**Realm export:**
- Organization mapper: `addOrganizationId=true`, `jsonType.label=JSON`, `addOrganizationAttributes=false`.
- `oidc-usersessionmodel-note-mapper` with `user.session.note=identity_provider` and `claim.name=idp`, in the access token.
- Tenant-linked IdPs: `trustEmail=OFF`, and first-broker-login `idp-create-user-if-unique` → `idp-add-organization-member`, both REQUIRED. No `idp-confirm-link`, `idp-auto-link` or `idp-email-verification`.
- The org has verified domains only.

**Platform rules:**
- Keycloak user and group ids come only from `principals` and `groups` rows read under RLS.
- Adding an existing user to another organization goes through an invitation. Invitation and add-member **reject** a user whose federated link points to an IdP not bound to the target tenant (CTO ruling).
- Tenant groups live under `/t/<tenant_id>/`. Group sync also requires the user to be a member of the organization.
- Access tokens live 5 minutes or less.

**Why bind on the id, not the alias:** the alias is immutable (spike: 400 "Cannot change the alias"). The id is still the binding key, because a deleted and recreated organization could reuse an alias.

### A.2 D22: Temporal access and the tenant path

**Network and TLS:**
- The frontend (7233) requires client certificates from the installation CA. A rogue CA fails at the handshake.
- `global.tls.internode` requires client certificates between services.
- History, matching, internal-frontend and worker ports (7234, 7235, 7239, 6933–6939) are **unreachable from the platform network**.
- Sandbox hosts, plugins, Mastra and anything tenant-supplied have no route to Temporal and hold no certificate.
- The UI is operator-only.

**Authorizer allow-list (deny by default, namespace `nava`):**

| Identity | APIs |
|---|---|
| all three below | `GetSystemInfo` |
| `nava-relay` | `StartWorkflowExecution`, `SignalWorkflowExecution`, `SignalWithStartWorkflowExecution` |
| `nava-core` | `RequestCancelWorkflowExecution`, `UpdateWorkflowExecution`, `QueryWorkflow`, `DescribeWorkflowExecution`, `GetWorkflowExecutionHistory` |
| `nava-worker` | `DescribeNamespace`, `ShutdownWorker`, `PollWorkflowTaskQueue`, `PollActivityTaskQueue`, `RespondWorkflowTask*`, `RespondActivityTask*`, `RecordActivityTaskHeartbeat`, `RespondQueryTaskCompleted`, `GetWorkflowExecutionHistory`, `RecordWorkerHeartbeat` |
| `temporal-system` | All. This is the server's own system worker, on the `systemWorker` certificate. |
| `nava-operator` | Admin and visibility, from the operator network only |

**Command inspection (TA-B3, adopted as written):**

> "The Authorizer also inspects `CallTarget.Request` for `RespondWorkflowTaskCompleted` from `nava-worker`. It decodes the task token's `WorkflowId`. It denies the call if any `StartChildWorkflowExecution`, `SignalExternalWorkflowExecution` or `RequestCancelExternalWorkflowExecution` command targets a workflow id whose tenant prefix differs, or if a `ContinueAsNewWorkflowExecution` command changes the workflow id. CI-6 asserts both the deny and the allow path."

A denied task retries until someone acts, so **each deny raises an alert** that includes the certificate, the workflow id and the command.

**Writing outbox rows (producer, inside the write pipeline):**
- Each row carries `causation_id` and `causation_depth` (I9).
- It takes `target_seq` = the record's `version` after this write, assigned under the record's row lock. A producer that signals a workflow without otherwise changing the record must still lock the record and bump its `version`. There is no other way to get a `target_seq`.
- A **start** row has `target_key = <kind>:<own outbox id>`. In the same transaction the producer inserts `workflow_ref(tenant_id, record_id, kind, outbox_id, closed_at)`.
- A **signal** row's `target_key` is read from `workflow_ref` under RLS. So a signal row queues behind its start row on the same target.

**What the relay does:**
- It builds every workflow id as **`t:<outbox.tenant_id>:<target_key>`**, and never reads a full id from a column.
- Starts use `REJECT_DUPLICATE` and `WorkflowExecutionErrorWhenAlreadyStarted = true`. `AlreadyStarted` counts as dispatched. To restart a failed workflow, a new event writes a new outbox row.
- On a signal, `NotFound` or "already completed" is **terminal at once**: the row is marked dead and an alert is raised.

**Claim order (fair across tenants, FIFO per target):**
- Walk the pending index per tenant and take at most 2 rows per tenant with `LATERAL … FOR UPDATE SKIP LOCKED`.
- A row is a candidate only if no pending row exists for the same `(tenant_id, target_key)` with a lower `target_seq`.

**Worst-case bound:**
- When a head backs off, the same `UPDATE` pushes its successors' `available_at` to the head's.
- The relay scans at most 200 rows per tenant per poll.
- Epic 08's flood test includes the case where every head is backing off.

**Guards in the worker:**
- The tenant travels in the `nava-tenant` header through a `ContextPropagator`.
- **Workflow start:** a missing header, or one that differs from the id prefix, fails the workflow non-retryably, with a metric.
- **Signal from the wrong tenant:** **dropped, with an alert. It never fails the receiving workflow.**
- **Update from the wrong tenant:** rejected by the update validator.
- **Outbound child, signal-external, cancel-external, or continue-as-new with a changed id:** rejected in the worker, then again at the Authorizer.
- **Activity:** a missing header fails it non-retryably. The database helper refuses a transaction with no tenant.
- **Never panic in a guard.** A panic produces a poison loop.
- `NavaTenantId` is for visibility only.

**Signal dedupe (I7):**
- The signal payload carries `outbox.id`. The handler keeps a bounded set of applied ids, carried across continue-as-new.
- The set's window must be longer than the relay's maximum redelivery age, from first attempt to `dead_at`.

### A.3 D28: roles, tables, policies

**Global settings (first migration):**
- `REVOKE ALL ON DATABASE nava FROM PUBLIC`
- `REVOKE ALL ON SCHEMA public FROM PUBLIC`
- `ALTER DEFAULT PRIVILEGES FOR ROLE nava_migrator REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC`
- **no** default `GRANT`

Grants come only from the manifest `db/grants.yaml`. All roles are `NOSUPERUSER NOCREATEDB NOCREATEROLE`.

| Actor | Role | Owns | Member of | BYPASSRLS | Access |
|---|---|---|---|---|---|
| Migrations (one-shot, direct, not through PgBouncer) | `nava_migrator` | all objects, plus `datdba` | none; granted to no one | No | Subject to `FORCE`. Backfills run per tenant, batched by key range. A post-run check covers RLS, `FORCE` and `PUBLIC`/default ACLs. |
| Core API | `nava_app` | — | none | No | Manifest DML under the tenant policy. `INSERT` on `outbox` and `workflow_ref`. `binding_lookup`. |
| Workflow workers | `nava_worker` | — | none | No | As `nava_app`. Tenant from the header. `INSERT … RETURNING` needs matching `SELECT` grants. |
| Outbox relay | `nava_relay` | — | none | No | `SELECT` on `outbox` except `dedupe_key` and `last_error`. `UPDATE (available_at, attempts, last_error, dispatched_at, dead_at)`. `SELECT` on `workflow_ref` is **not** needed. |
| Purge | `nava_purge` | — | none | No | `DELETE`, plus `SELECT` on key and predicate columns, through paired `FOR SELECT` and `FOR DELETE` policies with the same expiry predicate. Covers `idempotency_keys` (`expires_at < now()`), `outbox` (dispatched > 7 days or dead > 30 days) and `workflow_ref` (`closed_at` older than 30 days, an initial value that D23 may change). No audit purge in M1 (D23). |
| Onboarding | `nava_onboard` | — | none | No | `INSERT` on `tenants` and `tenant_identity_binding`. `SELECT` on its own provisioning row. Seeds through the pipeline grants, with `WITH CHECK (tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'))`. Idempotent on the request id. |
| Group sync | `nava_groupsync` | — | none | No | `binding_lookup`. DML on `principals` (including federated links), `groups`, `group_members` and `org_units`, one transaction per tenant. |
| Backups | `nava_backup` | — | none | No; `REPLICATION` | Physical only. `pg_hba`: `host replication nava_backup <backup-host>`, then **`host all nava_backup 0.0.0.0/0 reject`** and the IPv6 equivalent, placed **before** any catch-all line. |
| Partition maintenance | — | — | — | — | No partitioned tenant tables in M1 (D23) |
| Temporal persistence | `temporal` | DBs `temporal` and `temporal_visibility` | none | No | No `CONNECT` on `nava`, and the reverse |
| Keycloak / bootstrap | `keycloak` / image superuser | own DB / — | none | — | Bootstrap is used by the init script only. It is in no service's environment. |

**Allow-list: policies that do not test `app.tenant_id`.** Each names a `FOR` command and a `TO` role. None is `ALL` or `PUBLIC`.
- `outbox_relay_select`, `outbox_relay_update`
- `binding_lookup` (`FOR SELECT TO nava_app, nava_groupsync`), `binding_onboard` (`FOR INSERT TO nava_onboard`)
- `idempotency_purge`, `idempotency_purge_select`
- `outbox_purge`, `outbox_purge_select`

That is the 8 entries from round 2. **Plus** `workflow_ref_purge` and `workflow_ref_purge_select`, which come from the Postgres retention item the CTO ruled into the ADR. **Total: 10.**

**`outbox` columns:**
- `id uuid` (UUIDv7) and `tenant_id`, with PK `(tenant_id, id)`
- `kind`, `kind_version`
- `dedupe_key` with `UNIQUE (tenant_id, dedupe_key)`, for producer `ON CONFLICT DO NOTHING`
- `target_key`, `target_seq bigint`
- `causation_id uuid`, `causation_depth smallint` (I9)
- `payload jsonb`, references only
- `trace_context`
- `created_at`, `available_at`
- `attempts`, `last_error`
- `dispatched_at`, `dead_at`

`outbox` indexes:
- `(tenant_id, available_at) WHERE dispatched_at IS NULL AND dead_at IS NULL`
- `(tenant_id, target_key, target_seq)` with the same predicate

**`workflow_ref` columns:**
- `tenant_id`, `record_id`, `kind`, `outbox_id uuid`, `closed_at`
- Index `(tenant_id, record_id, kind, created_at DESC)`
- The final activity writes `closed_at`.

### A.4 Identity port (accepted with ADR-001)

The platform reaches the identity provider (IdP) **only** through two Go interfaces in the core, built in Epic 03. Nothing else in the platform talks to the IdP directly: no other code calls the Keycloak admin API, reads Keycloak-specific token claims, or depends on a Keycloak library. The **Keycloak adapter** is the only implementation in M1, and A.1 is its contract.

| Interface | What it does | Who may call it |
|---|---|---|
| **`IdentityResolver`** | Takes a validated access token. Returns `(tenant_id, principal_id, groups[], idp_alias)`, or a typed `tenant_unresolved` error. | The core API's authentication middleware, and nothing else |
| **`DirectoryAdmin`** | Creates a tenant organization. Creates, invites and disables users. Manages group membership. Lists directory changes since a cursor. | Tenant onboarding (`nava_onboard`) and group sync (`nava_groupsync`) only |

**Contract every adapter must meet** (adapter-independent; Keycloak meets it as specified in A.1):
- `tenant_id` comes only from `tenant_identity_binding`, keyed by the IdP's issuer plus an immutable organization identifier, never from a claim value (I2).
- A token that resolves to zero or several tenants is rejected, and so is any doubt about the brokering IdP. It fails closed (401).
- `groups[]` are platform group ids for this tenant, never raw IdP group names from an unverified claim.
- `DirectoryAdmin` never links a brokered login to an existing account by email, and its invite and add-member calls reject a user federated to an IdP that isn't bound to the target tenant (A.1).
- "List changes since a cursor" is how group sync keeps the deprovisioning bound (access-token lifetime plus sync interval).

**Changing the IdP.** A different adapter, for example Zitadel, changes D26. It needs a superseding ADR signed by nava, plus spikes at the depth of A1, A2 and A6 against that IdP. The Technical Architect's assessment lists the epics it would touch (01, 02, 03, 26).

**Revisit Zitadel when** (CTO ruling):
- Zitadel ships native user groups **and** its SCIM support is generally available under AGPL; or
- tenant self-administration (Epic 26) becomes a hard requirement before a spike has proved Keycloak 26.8's per-organization admin permissions.

## Annex B: acceptance tests

| # | Test | Catches |
|---|---|---|
| CI-1 | Every tenant table has RLS and `FORCE`. Every non-tenant policy is on the 10-entry allow-list, with `FOR` and `TO`. No `SECURITY DEFINER` function. Views are `security_invoker`. | I1, I3 |
| CI-2 | Per runtime role: no tenant set, `''`, and after a prior `SET LOCAL` through PgBouncer each give 0 rows or an error | Fail-open RLS, pool bleed |
| CI-3 | Recursive `pg_auth_members` is empty for runtime roles. No one is a member of `nava_migrator`. `pg_has_role(runtime, relowner, 'MEMBER') = false`. `datdba = nava_migrator`. `aclexplode(coalesce(acl, acldefault(type, owner)))` has no `PUBLIC` grantee on any platform object. | I3 |
| CI-4 | `has_table_privilege`, `has_column_privilege` and `has_function_privilege` equal the manifest exactly, for every runtime role | Grant creep |
| CI-5 | All of these are rejected:<br/>• zero or two organizations<br/>• an unknown issuer<br/>• a key from realm X with `iss=Y`<br/>• an ID token used as an access token<br/>• the `kc.org` tamper<br/>• a brokered login with an existing email<br/>• an out-of-domain brokered user<br/>• an `idp` not bound to the tenant<br/>• a user federated to a foreign IdP (at login and at invitation)<br/>• a duplicate binding<br/><br/>An in-domain brokered user gets the organization and `idp`. | D26, I2 |
| CI-6 | The following are rejected:<br/>• an unlisted or rogue-CA certificate<br/>• a platform pod connecting to the internal Temporal ports<br/>• a relay, core or worker certificate calling outside its allow-list<br/>• worker-certificate child, signal-external or cancel-external to another tenant, denied by the Authorizer, with an alert<br/>• continue-as-new with a changed id<br/><br/>These still work: a same-tenant child completes. Every SDK client dials and its worker starts.<br/><br/>Other cases:<br/>• no header, or a mismatched header, fails the workflow<br/>• a wrong-tenant signal is dropped and alerts, and the workflow keeps running<br/>• a redelivered start after completion starts no second run<br/>• a redelivered signal is applied once, including across continue-as-new<br/>• a signal queues behind its start<br/>• two relays never reorder one target<br/>• `NotFound` on a signal is terminal | D22, I2, I6, I7 |
| CI-7 | The workflow input and activity payload schema rejects record-data fields | R1 creep |
| CI-8 | Byte-identical 404s. Cross-tenant cursor replay returns 400. Idempotency replay returns the same response, and a different body returns 409. A stale `If-Match` returns 412. A masked-field filter returns 400. | D6, I4, I5, I7 |
| CI-9 | `nava_relay` cannot read `records`. `nava_purge` cannot read data columns or unexpired rows, and its purge actually deletes expired rows. `nava_onboard` cannot write to a non-provisioning tenant. `nava_backup` is rejected for a SQL session on `nava` and on `postgres`, and for a walsender with `replication=database`. | D28 |
| CI-10 | A write cascade beyond the depth limit, or a cycle, is rejected with a typed error and audited | I9 |
| CI-11 | Only the identity adapter package imports a Keycloak client, calls the Keycloak admin API or reads Keycloak-specific claims (static import and lint check). `IdentityResolver` and `DirectoryAdmin` pass a contract test suite that any future adapter must also pass. | A.4 identity port |

## Annex C: carried items

These are non-blocking and accepted as the reviewers stated them. Each is already reflected in Annex A or B.

**Security**
- **S-N1:** unique binding.
- **S-N2:** onboarding `WITH CHECK`; group sync requires org membership.
- **S-N3:** `FOR`/`TO`-scoped policies.
- **S-N4:** ids come from rows; invitation-only adds.
- **S-N5:** issuer and key binding.
- **Poison head:** terminal on `NotFound`.

**Postgres**
- **PG-N1:** `set_config` lint rule.
- **PG-N2:** fair claim shape.
- **PG-N3:** onboarding and pipeline grants, and the `RETURNING` grant.
- **PG-N4:** split policies; tenant policies `TO` the runtime roles.
- **PG-N5:** backfills batched by key range.
- **PG Q3a:** one manifest.
- **FIFO worst-case bound.**
- **`workflow_ref`:** stores `outbox_id`, with retention.

**Technical Architect**
- **Mapper settings.**
- **Alias immutability:** reasoning corrected.
- **`ContextPropagator`.**
- **Search attribute:** visibility only.
- **Guards:** non-retryable, never panic.
- **Go and SDK pins:** set together.
- **Minimal allow-list.**
- **Signal behind start.**
- **Applied-id window.**
- **R7:** separate module.
