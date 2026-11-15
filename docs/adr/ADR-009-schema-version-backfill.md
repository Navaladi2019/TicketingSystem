---
type: adr
id: ADR-009
title: Records catch up when a published version adds a default — durable per-record backfill, one read rule, effective-value filters, no default on unique fields
status: ACCEPTED
date: 2026-11-15
accepted: 2026-11-15
issue: TICK-897
rfc: RFC-021 (approved rev 1 with the CTO-ruled round-2 fold-ins, docs/adr/ADR-009/rfc/RFC-021-schema-version-backfill.md; thread under TICK-897 comment 01a1de81-1de7)
supersedes: "the metadata-engine compatibility-class row 'make a property required or optional' (Breaking), for the case where the property has a default in the next version"
---

# ADR-009: Records catch up when a published version adds a default

**Status: ACCEPTED** on 2026-11-15 by the Solution Architect, after the comprehension Goldfish check (TICK-897). No human sign-off is needed under the merge policy set by vknavaladi. Round 1: Security, Postgres and the Technical Architect all returned `CHANGES`, and RFC rev 1 resolved every blocking item. Round 2: the Technical Architect and Security returned `APPROVE`. Postgres returned `CHANGES` on one line: the filter must not trust the remaining count before the job has filled its queue. It approves with that fix (D12, first bullet). The CTO ruled the RFC approved with that fix as written (comment `01a1df42-e46c`). Verdict on TICK-897, PR #711.

> **Invariants source.** The project description, where a project-specific list of eight invariants would live, is empty. This ADR is therefore checked against the platform's nine invariants in [[ADR-001-platform-blueprint]] §1, I1–I9, which govern. If a separate project list is ever added, this check is incomplete against it.

## In plain words

- **What changes:** when an admin gives a field a default value, a background job writes that value into every existing record. Until it finishes, searches already count the default as a match. A field that must be unique can no longer have a default.
- **Why:** today an old record shows the default when you open it but is missing when you search for that value. The page and the search disagree, and nothing ever fixes old records that nobody edits.
- **What you'll see when it's done:** right after publishing "priority defaults to 3", a search for priority 3 finds the old records too. The record type's page shows the job's progress and counts. When the job ends, every record has priority 3 stored and one audit entry for the change, and its "last updated" time has not moved.
- **Codes used below:** `I1–I9` = the platform's nine ground rules (ADR-001). I4 = every change to a record goes through one checked write path. `D1–D15` = this ADR's decisions. `R1–R9` = the risks we accept. `CI-n` = acceptance tests (archived RFC §10). `RLS` = the database rule that hides other customers' rows. `ETag` = the version tag a client sends back when it saves. *Tenant* = one customer organisation. *Record type* = a kind of record a tenant defines, such as "incident". *Publish* = saving a new version of a record type. *Audit entry* = one line in the permanent change history. *Background job* = work the platform runs after the publish, without anyone waiting for it. `compatible` = a new version that needs no backfill. `needs_backfill` = a version whose old records need the new value written. *Breaking* = a version that is refused. The full definitions are in `docs/architecture/metadata-engine.md`.

## Terms used in the decisions

- *Outbox row*: a message saved in the same database transaction as the change that caused it. The *relay* (`nava-relay`) reads those messages and starts jobs in *Temporal*, the workflow engine. `D9a` is the platform's Temporal permission rule for what the relay may start (ADR-001-A1/A2). *Start-only* means only the relay may start the type: no running workflow may start it as a child or restart it ("continue-as-new").
- `ADR-006` is the decision that adds fast filter tables per field. Its D4 trigger copies stored values into them, its D9 barrier waits until writes begun before a point in time have finished, and its D10 controller sets the pause between pages of work. `S21` is the story that builds ADR-006. The *sealer* is the job that chains audit entries (ADR-004).
- **The three new tables:** `schema_backfills` (one row per job), `schema_backfill_pending` (the queue of records still to rewrite), and `schema_backfill_failures` (records that could not be rewritten: id and error code only).
- **Window:** a record type's backfill window is **open** from a publish that adds a default until its newest job row is `done`, so while it is `pending`, `filling`, `running` or `failed_records`. While it is open, some records may still lack the default in storage, so filters use the D12 shapes. *Remaining* = that job's pending rows plus failure rows, counted up to 1,001.

## Decision

| # | Decision |
|---|---|
| **D1** | A publish that adds any top-level default (every `needs_backfill`, and a `compatible` "optional with default") inserts, **in the publish transaction**, a `schema_backfills` row (`pending`) and an outbox row with `kind='schema.backfill'` and `dedupe_key='schema-backfill:<entity_definition_id>:<version>'`. |
| **D2** | **Actor and tenant scoping.** The relay starts the workflow with id `t:<tenant_id>:schema.backfill:<entity_definition_id>:<target_version>`, built from `outbox.tenant_id`. The workflow has **no tenant argument**. Activities take the tenant from `workerguard.TenantOf(WorkflowExecution.ID)` and use it both as `app.tenant_id` and as an explicit `tenant_id = $1`. The def and version are checked against `schema_backfills` under RLS, and a missing row is a non-retryable stop. `schema.backfill` is in `platformWorkflowTypes` **and** `startOnlyWorkflowTypes`, on the `nava-relay` D9a rule only. The job runs as `nava_worker`. Audit identity: `Source=system`, `actor_system='schema-backfill'`, causation origin = the publish audit entry, depth +1. No new DB role, no new RLS allow-list entry (it stays at 11), no `BYPASSRLS`. |
| **D3** | Each record is written by `pipeline.Write(OpMaterialize)` with an empty patch, through the full I4 pipeline. It compares `ApplyDefaults(stored)` with `stored` before any SQL. If they are equal, there is no write, no bump and no audit, and only the pending row is deleted. `validate` refuses `OpMaterialize` unless the patch is empty, `SourceSystem` and `actor_system='schema-backfill'`. |
| **D4** | One audit entry per rewritten record (`action=update`), plus one `entity_type`-domain entry at start and one at finish. |
| **D5** | `version` +1, so the ETag changes. `schema_version` moves to current. **`updated_at` is kept.** The time of the rewrite is in the audit entry's `occurred_at`. |
| **D6** | `OpMaterialize` starts no `on_updated` workflows, signals no waiting runs and writes no outbox rows. |
| **D7** | **Work queue `schema_backfill_pending`.** After the ADR-006 D9 barrier, the queue fill (`NOT (data ?& default_keys)`) and the flip to `running` commit in **one** transaction. The job reads pages of 100 ids **with no row lock**. Each record gets its **own** `InTenantTx` with `synchronous_commit=off`: the rewrite and its pending-row delete commit together. On a version conflict, the id stays pending. On a refusal or database error, a **separate** transaction moves the id to `schema_backfill_failures`, which holds the id and a code, never a value. No savepoints. The pause between pages is the ADR-006 D10 controller. One data backfill per tenant at a time: a second job for the same tenant stays `pending` until the first ends. At most 4 tenants run a job at once per worker pool. That is a platform setting with a starting value, which the Postgres & Data Modeling Specialist tunes from CI-12. Job row states: `pending` (queued) → `filling` (building the queue) → `running` → `done` or `failed_records`, or `superseded` (D8). A commit lost in a crash under `synchronous_commit=off` leaves the id in pending, so the next pass repeats it. The rewrite is idempotent (D3 compare). |
| **D8** | No lock on `entity_definitions`, and no new grant. Each write reads the current definition in its own transaction. Before each page the job checks `schema_backfills`: if a newer row exists for the type, it marks itself `superseded`, drops its pending rows and ends. The newer job's fill uses all of the target version's defaults and re-reads stored data. Records the superseded job already rewrote are skipped because they hold the key. Its failed records are queued again, because they still lack it. Its own failure rows are dropped with it. |
| **D9** | Poison records never block a type. A `failed_records` type keeps its window open, and its failed records still match filters through the D12 failures branch. The window closes with a later job that ends `done`: the next default-adding publish, or the retry path that Plan ADR-009 decides (OQ-4). |
| **D10** | `GET /entity-types/{key}` gains `backfill {state, target_version, rewritten, skipped, failed, remaining, started_at, finished_at}`. `GET /entity-types/{key}/backfill/failures` needs the entity-type admin permission. |
| **D11** | **Reads.** Every surface that returns a record body (single read, list, GraphQL, embed SDK, the workflow record-read capability, any future export) uses `effective(rec) = cur.ApplyDefaults(rec.Payload(fixedColumnsOf(rec.SchemaVersion)))`, then field masks (the per-role hiding of fields). Audit diffs and snapshots show stored bytes. |
| **D12** | **Filters match effective values.** While the window is open, and only for a defaulted `f` whose predicate is true for the default: <br>• while the newest `schema_backfills` row is `pending` or `filling`, the filter is `(<stored>) OR <f absent>`; <br>• once the row is `running` or `failed_records`: with more than 1,000 remaining, the same `OR` shape; with 1,000 or fewer, `(<stored>) UNION ALL (records ⋈ pending ∪ failures on (tenant_id, record_id), AND <f absent> AND NOT <stored>)`. <br>The ACL and the other filters are in every branch, and `total` (the list's match count, capped at 1,001) uses the same shape. This applies to the JSONB path and the ADR-006 side tables. ADR-006's D4 trigger is unchanged. |
| **D13** | Compile refuses `default` + `x-nava-unique` (422 `invalid_schema`). Uniqueness checks stored values. |
| **D14** | "Make an existing property required" is `needs_backfill` when the next version has a default for it. Every other Breaking class stays refused. |
| **D15** | A rollout migration, the one place allowed to loop over tenants (it runs once at deploy, as the migrator role, with `app.tenant_id` set per tenant), enqueues, per tenant and per type with any default, a `schema_backfills` row and an outbox row. It writes no `data` and does not scan `records`. It **skips** (queues no job for) any type whose current version already has `default` + `x-nava-unique`, and lists those types in its output for the Technical Lead, because backfilling one would only produce unique conflicts. D13 stops new ones at the next publish. |

**Grants (`db/grants.yaml`):** `nava_worker` gets DML on the three new tables. `nava_app` gets **SELECT only** on them. No change to any existing table. All three tables have `FORCE RLS`, a PK led by `tenant_id`, and a composite FK to `entity_definitions`.

## Context it rests on

- The read path fills in defaults (`internal/server/records.go:247-252`, `records_list.go:453`). Filters and uniqueness read stored values only (`records_list.go:216`, `store/unique.go:65`).
- Updates already apply current defaults (`pipeline.go:758-821`). Every update starts `on_updated` workflows (`eventsink.go:88-130`).
- A published default never changes, because changing or removing one is Breaking. **D11 and D12 depend on this.**

## Consequences

- Per rewritten record: about 7.3 ms of work and about 4 KB of WAL. `records` grew 204→318 MiB and `audit_log` 53→161 MiB at 100k (TA spike).
- Live-write p95 rose **+1.7%** at 20k with per-record transactions, against **+28% at the same 20k** (and +416% at 100k) for the rejected batch lock. A publish during a run took 49 ms, against 20 ms idle.
- Filters on a defaulted field cost more only while the window is open. Measured page plus capped count: 20–22 ms with 1,000 remaining (UNION ALL; the OR shape took 84–91 ms there, so the switch picks UNION ALL), and 14–15 ms with 5,000 remaining (OR, faster than UNION ALL's 28–31 ms there). The count part of the OR shape grows with type size (R3).
- **Sequencing:** D7 reuses ADR-006's D9 barrier and D10 pause controller, and D12 must also land on ADR-006's filter tables. Both are built by S21, which has not started. Plan ADR-009 must either schedule the backfill story after S21, or build those two shared pieces first and have S21 reuse them. Until the filter tables exist, D12 applies to the JSONB filter only.
- Plan tasks: the three tables and grants, `OpMaterialize`, the workflow plus the Authorizer entry, D12 in `records_list.go` and in S21, D10, D13, D15, and the `updated_at` description in `api/paths/records.yaml` ("the last change to the record's effective content; a schema backfill bumps `version` without changing it"). **OQ-4** (`backfill:retry` endpoint) is deferred to `Plan ADR-009`, where the Technical Lead decides its timing.
- Handbook: the metadata-engine (class table, D13), record-write-pipeline (`OpMaterialize`) and ADR-006 filter pages change.

## API test scenarios

| Operation | Scenarios |
|---|---|
| `POST /api/v1/entity-types/{key}/versions` (`publishEntityTypeVersion`) | Adds a default → 201, plus a job row and an outbox row in the same transaction. Adds no default → 201 and no job. `default` + `x-nava-unique` → 422 `invalid_schema` and nothing stored. A forced rollback leaves no job row and no outbox row. A redelivered outbox row → one workflow (dedupe key and workflow id). Cross-tenant type → 404, the same response as a missing one. |
| `GET /api/v1/entity-types/{key}` (`getEntityType`) | `backfill` is present when a job exists and absent otherwise. Cross-tenant → 404. |
| `GET /api/v1/entity-types/{key}/backfill/failures` (new) | 200 for an entity-type admin, 404 for a plain user and for a cross-tenant type. Read-only, so it writes nothing. |
| record list with `filter[f]=` | Row `pending`: the records that lack f are returned. Row `running`, many or few remaining: the same set and `total`. An ACL-hidden record is never returned. |

Acceptance tests CI-1 to CI-12 are in the archived RFC §10. They include CI-3 (1)–(7) for cross-tenant and start-only, and CI-7 (a) for the `pending` case.

## Risks and residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R1 | WAL and storage burst, about 400 MB of WAL per 100k records. The 100k run with per-record transactions is unmeasured. | Postgres & Data Modeling Specialist | CI-12: live p95 > +25% at pause 0, or > 30 min for 100k |
| R2 | One 412 per edit form open during a rewrite | Solution Architect | support reports, or the SDK lacks re-read on 412 |
| R3 | The cost of the `OR` shape's capped count grows with type size (about 80 ms at 100k just above the switch, and about 4× that at 400k, inferred) | Technical Architect | S21 measures > 150 ms at 400k |
| R4 | Sorting on a defaulted field while the window is open puts records that lack it in the null position | Solution Architect | any tenant-visible report |
| R5 | Sealer lag from per-record audit (unmeasured) | Postgres & Data Modeling Specialist | CI-12 sealer lag > the ADR-004 SLO |
| R6 | `failed_records` keeps records unmaterialised | Technical Lead | any type > 7 days in `failed_records` (OQ-4) |
| R7 | Rests on "a published default never changes" | Solution Architect | any RFC that makes changing or removing a default non-Breaking |
| R8 | `version` moves while `updated_at` does not | Solution Architect | an integration that relies on the two moving together |
| R9 | Single Temporal history (start-only, so no continue-as-new): about 8k events per 100k records. That is 1,000 page activities plus 1,000 pause timers at about 4 events each. It is an assumption. | Technical Architect | CI-12 counts more than 25k events for 100k |

## Rejected options

| Option | Why |
|---|---|
| Batches of 100 under `FOR UPDATE SKIP LOCKED` (RFC rev 0) | +416% live p95 and 73 conflicts at 100k. A livelock on a database error. Savepoints overflow the subxact cache. |
| `FOR SHARE` on `entity_definitions` to serialise with publish | `42501` as `nava_worker`. It needs a new grant and policy, and it delayed a publish by 3–8 s. |
| Never backfill, and compute effective values in every query | A permanent unindexable cost, and defaults duplicated into plpgsql |
| Bulk `UPDATE … data \|\| defaults` from a migrator job | Bypasses I4 (validation, version, audit), and puts an operator credential on a tenant action |
| Lazy only, with stored-only filters | This is the bug |
| Refuse filters while the job runs | Unusable on large types |
| Count-only D12 shape choice | Picks an empty queue while `pending` or `filling`, which is the bug again (PG round 2) |

**Dissent:** none recorded. The round-1 PG-1/PG-5 against TA-2 lock-wait disagreement was settled by measurement and the CTO's ruling.

## Competitive Gate (ADR-008)

**Result:** PASS
1. **Axis moved:** X2 change and upgrade cost (a). Adding a field with a default to a type that already has records goes from "publish, then old records stay wrong in search until each is edited" to **one publish**, with no admin follow-up step and no platform staff.
2. **Pain points:** avoids CP-SF-02: the tenant writes no batch code around per-transaction ceilings to backfill. The platform's job is throttled per tenant. Risks repeating: none.
3. **Measured by:** CI-2 and CI-7 (planned), and spike CI-12 (in part: `docs/adr/ADR-009/spikes/`).
4. **Parity:** I1–I9 unchanged. The backfill uses I4's one write pipeline instead of bypassing it.
