---
type: adr
id: ADR-005
title: Record numbering — a number allocator outside the create transaction (L2s), declarative number formats linked to entity types, scoped counters
status: ACCEPTED
date: 2026-11-10
issue: TICK-653
rfc: docs/adr/ADR-005/rfc/RFC-014-record-numbering.md (rev 1, archived with rev-0/ and rev-1/; thread under TICK-653 comment 01a1c137-ed1f-71c2-8ee9-9b9aa87380b4)
supersedes: none
amends: PLAN-003 D-3-9 (docs/delivery/plan-records-path.md:88)
source: Human request on TICK-648 ("OPEN AN RFC", 2026-11-09), plugin option requested by vknavaladi (TICK-653, 2026-11-09 17:21 UTC); P3-36 trigger from TICK-621 (create p95 > 50 ms)
deciders: Solution Architect (author); Technical Architect (rev 1 APPROVE with findings 1–5, spike S1, comment 01a1c19d-bb65); Security & Multi-Tenancy Reviewer (round 1 APPROVE with C1–C7, comment 01a1c147-92d1; rev 1 additions APPROVE with C8–C13, comment 01a1c310-2851); Postgres & Data Modeling Specialist (round 1 CHANGES, two blocking items closed by the Technical Architect's re-check, comment 01a1c14c-b6d3); CTO (chair rulings, comment 01a1c30c-b7e4); vknavaladi (per-type `numbering_mode`, TICK-621, 2026-11-10 02:06 UTC, recorded in PR #551); CTO (leading-`{prefix}` rule, comment 01a1c387-b543)
---

# ADR-005: Record numbering

> **Status: ACCEPTED (2026-11-10, Solution Architect verdict on TICK-653 after the Technical Architect critique and CTO rulings; no dissent).** All reviewers have approved RFC-014 rev 1 or had their blocking items closed (CTO ruling, comment 01a1c30c-b7e4). Security approved rev 1's additions with conditions C8–C13 (comment 01a1c310-2851), written in §3.7. **The gaps policy is decided, per type** (vknavaladi on TICK-621, 2026-11-10 02:06 UTC; recorded as D-3-9 in PR #551): `entity_definitions.numbering_mode` is `gaps` (default) or `gapless`. A `gaps` type allocates with **L2s**; a `gapless` type allocates with **L1** (§2.1). Every format starts with `{prefix}` (§3.1, CTO ruling 01a1c387-b543).
>
> **Erratum E1 (ACCEPTED 2026-11-12, TICK-665):** §4.2 "old and new binaries coexist" holds for existing counters only; see [[ADR-005-E1-first-create-race-in-expand-window]].
>
> **Erratum E2 (ACCEPTED 2026-11-13, TICK-775):** §3.5 `grow` is also refused when a digit comes directly **before** `{seq:N}`, not only after it; `{prefix}{yy}{seq:3}` and `{prefix}-{yyyy}{MM}{seq:4}` need `refuse`. See [[ADR-005-E2-grow-refused-when-digit-touches-seq]].
>
> **Invariants source.** The project description defines no invariants, so this ADR is checked against [[ADR-001-platform-blueprint]] §1, I1–I9 (`main` at `89e0f84d`).
>
> Reviewer conditions are written as rules and tagged: **[TA-n]** Technical Architect finding n, **[Sec Cn]** Security condition n (C8 and C11 are **High if missed**), **[PG n]** Postgres item n, **[CTO]** chair ruling.

## Context

Every record gets a human-facing number. Today (PLAN-003 D-3-9) it is `<PREFIX><7 digits>`, taken by one upsert on `record_counters(tenant_id, entity_definition_id, next)` **inside the create transaction**, so the counter row stays locked until the create commits.

Two problems:
1. **Latency.** With 8 clients creating the same type in one tenant, the counter queue dominates: TICK-621 measured counter wait p95 237–310 ms and create p95 264–408 ms against a 50 ms target. Spike S1 on a quiet host measured wait p95 150–179 ms for today's design (L0).
2. **Formats.** Tenants building generic applications (visa, PF and so on) want numbers such as `yyyymm____` (monthly reset), `yyyymmdd__` (daily reset) or `BRANCH_yy__MM_` (per branch and month). The format is fixed in code today (`internal/records/store/numbering.go`). vknavaladi asked that a format can be "created dynamically and linked", possibly as a plugin.

## Decision

1. **The number is allocated in its own tiny transaction (L2s), before the create transaction and after authorisation and validation.** It runs on a dedicated allocator pool, in one pipelined send: `BEGIN; set_config(app.tenant_id, …, true); SET LOCAL synchronous_commit = off; upsert … RETURNING; COMMIT`. The create transaction then inserts the record with that number. This is the path for **`gaps` types**, the default. A **`gapless` type** takes its number last inside the create transaction (L1), so a rolled-back create gives its number back. The mode is per type, `entity_definitions.numbering_mode`, decided on TICK-621 (§2.1).
2. **Number formats are declarative templates that start with the type's `{prefix}`**, e.g. `{prefix}-{yyyy}{MM}{seq:4}` or `{prefix}/{field:branch}/{yy}/{seq:3}`, validated and compiled at publish (§3). Nothing in a format adds a statement or a round trip to the counter lock.
3. **A number format is tenant-owned, versioned metadata (`number_formats`)**, created at runtime by an admin (ships first) or, once the plugin principal of §3.7 C11 exists, by a plugin through the public API, and **linked** to entity types by a pinned `(number_format_id, format_version)` [CTO]. This is how "created dynamically and linked" is met: configuration, no code.
4. **Counters are scoped:** `record_counters` gains `scope_key`, and its primary key becomes `(tenant_id, entity_definition_id, scope_key)`, changed by an expand/contract migration (§4).
5. **The default format is today's format.** `{prefix}{seq:7}` with `overflow: grow` is epoch 0 and uses `scope_key = ''`, so every existing type continues its existing counter. No existing number is migrated or rewritten.
6. **Plugins that run code per create are deferred with a trigger (P-b); plugins that own the counter or run inside the lock are rejected (P-c).**
7. Sharding or partitioning `records` (ADR-001 D23) is out of scope.

## 1. Options considered

### 1.1 Latency (the lead question)

Measured by spike S1 (Technical Architect, quiet host, 8 clients on one type and scope, PG 16.10, pgx v5.11.0; `docs/adr/ADR-005/spikes/technical-architect/s1/`). Counter wait is measured on the server.

| | Option | Counter wait p95 | Create p95 (SQL level) | Throughput | Gaps | Verdict |
|---|---|---|---|---|---|---|
| L0 | Today: lock held until the create commits | 150 / 179 ms | 171 / 207 ms | 103–123/s | none | rejected: misses |
| L1 | Gapless; write batch and `COMMIT` in one pipelined send | 108 / 120 ms | 133 / 142 ms | 142–149/s | none | **chosen for `gapless` types**; waiver for those types only |
| L2 | Own transaction, synchronous commit | 31 / 36 ms | 69 / 120 ms | 143–180/s | on failed creates | superseded by L2s |
| **L2s** | L2 with `SET LOCAL synchronous_commit = off` | **0.84 / 0.99 ms** (contended host 0.97 / 1.20) | **39.5** ms (one rep 114 ms under transport noise) | **258/s** | on failed creates | **chosen for `gaps` types** (the default) |
| L2′ | Allocate on the main pool before the checks | 22 / 51 ms | 87 / 133 ms | 134/s | also on every refused create | **rejected [CTO]**: allocation before authorisation lets a principal with no create permission advance any type's counter, a side effect ahead of I4 |
| L3 | Blocks of numbers cached per server | ~0 | — | — | large, on restart | rejected for M1. Trigger: one scope needs > ~500 creates/s sustained |
| L4 | Number assigned after commit | 0 | — | — | none | rejected: a create must return its number |

### 1.2 Format model

| Option | What it is | Verdict |
|---|---|---|
| Declarative template on each type | Template string stored on the entity definition | rejected in its plain form: a tenant with 30 types sharing a format edits 30 types |
| **Declarative template as linked metadata** | `number_formats` object, versioned, linked by pinned version | **chosen** |
| Plugin computes the number (P-b) | Tenant code runs per create, outside the lock, returns the rendered parts | **deferred** (§6) |
| Plugin owns the counter (P-c) | Tenant code increments or is called inside the lock | **rejected**: violates I8 and the chair's hard line |

## 2. The allocator (L2s for `gaps` types, L1 for `gapless` types)

### 2.1 Gaps policy — decided, per type

**Decided by vknavaladi on TICK-621 (2026-11-10 02:06 UTC), recorded as D-3-9 in PR #551** (`docs/delivery/decisions/2026-11-10-d-3-9-numbers-may-have-gaps.md`). Each type chooses `entity_definitions.numbering_mode`: `gaps` (default) or `gapless`, set together with `number_prefix` and changeable later.

| | **`gaps` (default): L2s** | **`gapless`: L1** |
|---|---|---|
| How the number is taken | Dedicated allocator pool, `tenancy.AllocTx`, `SET LOCAL synchronous_commit = off`, committed before the create transaction begins (§2.2) | Taken **last inside the create transaction**; a rolled-back create gives its number back |
| What users see | A create that fails after allocation leaves a hole (INC41, INC43). After a database crash a few never-used numbers may be handed out again; no number is ever on two records | Every number used, no holes |
| Counter wait p95, 8 same-scope clients | ~1 ms | 108–120 ms |
| 50 ms create target (8 clients) | **applies**; met at the database level (A3 still owed) | **waived for `gapless` types only**: recorded waiver, about create p95 ≤ 200 ms at 8 clients (exact value after the clean re-run) |
| Ceiling per tenant, type and scope | ≥ 258 creates/s measured | ~140–150 creates/s |

**Gaps are never reclaimed.** A shared free list would need its own lock (every create serialises again), and a reclaimed number would be older than numbers already issued, breaking sort-by-number. Both modes use the same counter row, the same scoped key (§4) and the same compiled format (§3): one allocator with two placements, not two numbering systems.

#### 2.1.1 Changing the mode
- **Both paths upsert the same row**, `(tenant_id, entity_definition_id, scope_key)`, with the same `ON CONFLICT … DO UPDATE SET next = next + 1`. Whichever path holds the row lock, the next caller sees the advanced value, so a mode change cannot produce a duplicate number. A `gapless` rollback returns only a number no other caller could have read, because it held the row lock until the rollback.
- **The change applies to creates that start after the definition change.** A create reads `numbering_mode` with the definition at its start; creates already running finish on the path they began with. No counter row is rewritten.
- **Crash safety holds across the switch.** A `gapless` create commits synchronously, so its fsync also flushes any earlier `gaps` (L2s) counter commit (the WAL-prefix argument below).
- **Pinned test:** 8 concurrent creators on one type and scope while the type's mode is flipped `gaps` → `gapless` → `gaps` several times, with some creates forced to fail; assert that no two records share a number.

### 2.2 Rules

| # | Rule | Tag |
|---|---|---|
| A1 | Allocation runs only through a new pipelined `tenancy.AllocTx` beside `InTenantTx`, so the navalint setconfig analyzer (PG-N1) covers it. No hand-written `set_config`; never session-level (`false`) | [Sec C1] |
| A2 | The allocator's tenant is `tenancy.TenantOf(createTx)`, never a separate parameter. The result carries `(tenant, definition, scope_key, epoch, number, instant)`, and the create checks it against its own transaction before inserting | [Sec C2] |
| A3 | The allocator pool logs in as `nava_app`. Startup refuses any other role on it: owner, migrator or any `BYPASSRLS` role | [Sec C3] |
| A4 | `SET LOCAL synchronous_commit = off` appears only inside `tenancy.AllocTx`, never at session level. `SET LOCAL` resets at transaction end, so it is safe on a PgBouncer transaction pool | [CTO], [TA-6] |
| A5 | Dedicated pool, default size 4 per API replica. The pool total (4 × replicas) is added to the `max_connections` and PgBouncer sizing. Allocation timeout 200 ms; on timeout the create fails with `ErrRetryable`. A second pool is required because `InTenantTx` holds its main-pool connection for the whole of `fn` (`internal/tenancy/tenancy.go:85-93`); allocating from the same pool inside `fn` can deadlock when the pool is exhausted | [CTO], [TA Q1] |
| A6 | Allocation runs **after** the replay check, authorisation and validation, immediately before the create transaction. A replay returns the stored record and its number and allocates nothing (I7, TICK-482, TICK-498). A retry of a failed create allocates a new number (a gap, never a duplicate) | [CTO] |
| A7 | The allocator statement computes the date parts, the date portion of `scope_key` and `period_end` **in SQL** from `now() AT TIME ZONE $tz`. Go passes the compiled pattern and the field parts. The allocator returns `now()`, and the create stores it as `created_at`, so the date in the number always matches `created_at`, including at midnight. No round trip is added; `now()` is fixed at transaction start. (Under gapless L1, `now()` is returned by the existing definition-read round trip.) | [TA-2] |
| A8 | The pool's acquire wait is a metric of its own (`nava_numbering_alloc_acquire_seconds`), alongside allocation latency and timeouts | [TA Q1] |

**Why L2s cannot put one number on two records.** The allocator commit only advances a counter. A record using that number commits later with a synchronous fsync, and `XLogFlush` flushes all WAL through the given position (`src/backend/access/transam/xlog.c`, REL_16), which includes the earlier counter commit. A crash can therefore lose only counter advances whose numbers never reached a committed record. Spike S1 confirmed it in 3 of 3 forced crashes (`docker kill -s KILL`, `wal_writer_delay=10s`): runs 2 and 3 lost 3 and 2 counter advances, those numbers were reissued after restart, and there were 0 duplicates and 0 errors in the creates that followed. The guarantee depends on WAL being replayed as a prefix, which holds for crash recovery, physical replicas and PITR; see AR-TICK-653-03.

## 3. Number formats

### 3.1 Grammar (JSON Schema plus a Go compiler, at publish)
`template := (literal | token)+` with exactly one `{seq:N}`.

| Token | Meaning |
|---|---|
| `{prefix}` | the type's existing `number_prefix` |
| `{yyyy}` `{yy}` `{MM}` `{dd}` | date parts of the allocation instant in the numbering time zone |
| `{seq:N}` | the counter, zero-padded to N digits, 1 ≤ N ≤ 12 |
| `{field:<key>}` | the mapped code of an enum property (§3.4) |
| literal | `[A-Z0-9_\-/.]`, at most 16 characters per run |

Rendered length ≤ 64. Unknown tokens are refused, so a new token type is an additive change.

**Leading-`{prefix}` rule** [CTO, comment 01a1c387-b543]. Every template **starts with `{prefix}`**, and the token right after it **must not produce letters**: it cannot be a literal starting with `A–Z` or a `{field:…}` token. It may be a date part, `{seq:N}`, or a literal starting with a digit or one of `_-/.`. Otherwise the publish is refused at compile time with **`number_format_prefix_required`** (422), naming the offending position. Why: `number_prefix` is `^[A-Z]{2,6}$` and unique per tenant, so the leading run of letters of any number names exactly one type. A number therefore stays unique across the whole tenant (search by number finds one record) with no check across types.

Examples (prefix `VISA`): `yyyymm____` → `{prefix}-{yyyy}{MM}{seq:4}` (`VISA-2026110001`); `yyyymmdd__` → `{prefix}-{yyyy}{MM}{dd}{seq:2}` (`VISA-2026111001`); `BRANCH_yy__MM_` → `{prefix}/{field:branch}/{yy}/{seq:2}/{MM}` (`VISA/BLR/26/01/11`). Refused with `number_format_prefix_required`: `{yyyy}{MM}{seq:4}` (no prefix), `{prefix}{field:branch}{seq:3}` and `{prefix}X{seq:4}` (letters right after the prefix).

**Compiled form**, snapshotted on the published **definition version**: `scope_parts`, `head`, `width`, `tail`, `overflow`, `epoch`, `compiled_sha256`, `time_zone`. A create reads only this snapshot, never `number_formats`, so formats add nothing to the create path or the lock.

### 3.2 `number_formats` and links [CTO], [TA-3]
- `number_formats` (head) and `number_format_versions` (immutable), shaped and granted as in §3.7 C9, `tenant_id` leading, FORCE RLS (I1). Published through the same configuration publish path, with the same validator and compiler; audited (`subject_type = number_format`).
- An entity **definition version** links a pinned `(number_format_id, format_version)` through a composite FK that leads with `tenant_id` (§3.7 C8). The link lives on the immutable `entity_definition_versions`, never on the mutable `entity_definitions` row. No link means the default (epoch 0).
- **A definition publish keeps the pinned version unless that publish explicitly bumps it.** An unrelated publish (e.g. adding a field) never changes a type's numbering.
- **"Re-publish linked types"** in the admin view runs N independent per-type publishes, each checked for epoch and disjointness on its own, and reports success or refusal per type. **No automatic fan-out.**
- Field tokens in a shared format are validated per linked type at that type's publish (§3.4).
- **Plugin-authored formats (P-a):** the same validator and compiler as a human publish, no plugin-only path. **Plugin writes stay off until the plugin principal of §3.7 C11 is specified; admin-UI format publish ships first.** The audit actor is set by the server (§3.7 C12).
- "Re-publish linked types" authorises each type separately for the caller (§3.7 C13).

### 3.3 Epochs and disjointness
- **Epoch = identity of the compiled template within a type**, not of the definition version. A publish with the same `compiled_sha256` keeps the epoch and continues the counter. A changed compiled template gets the next epoch. Re-adopting an earlier template reuses its epoch and continues that counter. `{prefix}{seq:7}`/`grow` is always epoch 0 with `scope_key = ''`. [PG 2]
- **Epoch registry** [TA-4]: `numbering_epochs(tenant_id, entity_definition_id, epoch, compiled_sha256, literal_head, created_at)` with `UNIQUE (tenant_id, entity_definition_id, epoch)` and `UNIQUE (tenant_id, entity_definition_id, compiled_sha256)`, and the composite FK `(tenant_id, entity_definition_id) REFERENCES entity_definitions (tenant_id, id)` [Sec C8]; `tenant_id` leading, FORCE RLS, isolation-registry rows. The epoch is taken inside the publish transaction under `FOR UPDATE` [Sec C10], so two concurrent publishes cannot both claim epoch N+1. Epoch 0 is implicit (no row needed).
- **Disjointness rule** [TA-1]: the *literal head* of an epoch is the literal text before its first variable token (`{prefix}` resolves to its literal). For every pair of epochs of a type, refuse the publish if one literal head is a prefix of the other (equal heads included) **unless** the first character where the longer head continues lies outside the character set of the shorter template's next token. Example: epoch 0 `INC` + digits vs `INC-{yyyy}…` is allowed (`-` is not a digit); epoch 0 vs `{prefix}{yy}{seq:3}` is refused (same head `INC`, both continue with digits); `{prefix}{yy}{seq:3}` vs `{prefix}2{seq:4}` is refused (`INC` is a prefix of `INC2`, and `2` is a digit). A property test over generated template pairs asserts that no two accepted epochs can render the same string. The records unique index `(tenant_id, entity_definition_id, number)` stays the backstop; a violation there is a 500-class bug, never silent.
- **The disjointness check stays within one type.** The leading-`{prefix}` rule (§3.1) guarantees that numbers of two different types cannot clash, so no check across types is needed.
- Existing numbers are never rewritten.

### 3.4 Field tokens [Sec C5–C7]
A property may appear in a number only if all of these hold, checked at publish **and on every change to the mask set**:
1. required at create;
2. **immutable after create**: schema keyword `x-nava-immutable`, enforced in the one write pipeline (I4) on every write path (PATCH, bulk, import, `nava_worker` workflow writes, schema-version upgrades), one test per path [Sec C7];
3. an **enum** property in M1, mapped to `^[A-Z0-9]{1,10}$` by a declared mapping. Reference-code tokens are deferred; if added, they render only after the reference-read check, with the same 404 for unreadable and missing [Sec C6];
4. not field-masked for any role that can read the record (I5).

**A mask added after a field is already in a format.** A mask change on a field the current format references is accepted only together with a new definition version whose format drops that field (a new epoch). A mask-only publish gets a 422 naming the format and the fix. The re-check runs on ACL or mask publish, on a new role with read on the type, and on a role's mask edits. Numbers already issued stay as they are (AR-TICK-653-01).

**Dependency:** field tokens ship only after S10 per-field masks exist (`internal/audit/read/mask.go` has none yet) [Sec C5].

### 3.5 Overflow
Per format, `overflow: grow | refuse`. `grow` is allowed only when the template parses unambiguously (every variable-width token is followed by a literal outside its character set, or ends the number); `{field:br}{seq:3}` is refused. `refuse` fails the create with `number_space_exhausted` (409) naming the type and scope, audited, alerted at 90 % of the width.

> **Erratum E2 (2026-11-13):** the `grow` sentence above is replaced by: *`grow` is allowed only when the number reads unambiguously without the template: no token or literal that renders a digit is directly before or directly after `{seq:N}`, and every other variable-width token is followed by a literal outside its character set or ends the number. `{field:br}{seq:3}` and `{prefix}{yy}{seq:3}` are refused with `number_format_invalid`; use `refuse`.* It is a readability rule; uniqueness across epochs is §3.3's job. Date-then-sequence examples in §3.1 (`{prefix}-{yyyy}{MM}{seq:4}`) need `refuse`. See [[ADR-005-E2-grow-refused-when-digit-touches-seq]].

### 3.6 Time zone
A tenant setting `time_zone` (IANA name, default `UTC`), with an optional per-format override. A change affects future scope keys only.

### 3.7 Security conditions on formats, links, epochs and plugins [Sec C8–C13]
Security approved rev 1's additions with these conditions (comment 01a1c310-2851). They are rules of this ADR, stated as written.

| # | Rule | If missed |
|---|---|---|
| **C8** | **Every link is a composite foreign key that leads with `tenant_id`.** The pinned link is `(tenant_id, number_format_id, format_version) REFERENCES number_format_versions (tenant_id, format_id, version)`. `numbering_epochs` has `(tenant_id, entity_definition_id) REFERENCES entity_definitions (tenant_id, id)`. Reason: PostgreSQL runs FK checks as the table owner and ignores RLS (`20261031150000_entity_definitions.sql:3-5`), so a single-column `number_format_id` FK would let a tenant A admin link tenant B's format UUID: a cross-tenant pointer and an existence oracle (FK pass vs "not found"). With the composite key, B's id and a made-up id both fail with the same `number_format_not_found` (422) | **High** |
| **C9** | **Same shape as `entity_definitions` / `entity_definition_versions`.** Head `number_formats`: PK `(tenant_id, id)`, `UNIQUE (tenant_id, key)` (never a global `key`, which would be a cross-tenant uniqueness oracle), `current_version`. Immutable `number_format_versions`: PK `(tenant_id, format_id, version)`. `ENABLE` + `FORCE ROW LEVEL SECURITY` on `number_formats`, `number_format_versions` and `numbering_epochs`, with the canonical `NULLIF(current_setting('app.tenant_id', true), '')::uuid` predicate on every policy. Grants: `nava_app` SELECT and INSERT, plus column-level UPDATE of `current_version, updated_at` on the head table only (as `db/grants.yaml:117`); `nava_worker` SELECT only; **no UPDATE or DELETE grant or policy on versions or epochs for any role**; `nava_purge` never touches them. If onboarding seeds default formats, `nava_onboard` INSERT uses the pinned "tenant is provisioning" policy, as for definitions. The link lives on the immutable `entity_definition_versions`; the UPDATE grant on `entity_definitions` stays `current_version, updated_at` | will bite at scale |
| **C10** | **The epoch is taken inside the tenant publish transaction.** The disjointness check and the new-epoch INSERT run in the same `InTenantTx` as the definition publish, after `SELECT … FOR UPDATE` on that tenant's `entity_definitions` row, so the UNIQUE constraints never turn a race into a 500 and every epoch read goes through RLS | will bite at scale |
| **C11** | **Plugin writes stay off until the plugin principal is specified; admin-UI format publish ships first.** There is no plugin identity today: `identity.Principal` has no kind or scopes (`internal/identity/identity.go:30`), `Resolve` reads no `scope` claim (`internal/identity/keycloak/keycloak.go:197`), and `principals` holds Keycloak users only. **Precondition for enabling plugin writes**, the five-point contract: (1) one Keycloak client per (plugin, tenant), never one shared client that picks its tenant per request; (2) the tenant comes from the binding, never from a claim value, header or body (I2): either the client's service-account user belongs to exactly one organization and goes through `Resolve` unchanged, or a platform binding `(iss, azp) → tenant`, written only by the tenant admin's install flow and RLS-forced like `tenant_identity_binding`; (3) the service account is synced as a `principals` row, so `audit_log.actor_principal_id` is a real FK `(tenant_id, actor_principal_id)`, not a free-text `actor_system`; (4) scope narrows, never grants: the effective right is (token scope) ∩ (the plugin principal's tenant ACL); (5) `number_format:write` does not imply `entity_definition:publish`: linking a format, bumping a type's pinned version or "re-publish linked types" need definition publish on each type | **High** |
| **C12** | **`source = plugin` and `source_ref = <client id>` are set by the server** from the resolved principal's kind and `azp`, never from the request (otherwise within-tenant attribution forgery, Medium). The PR adding formats adds `number_format` to `audit_log_subject_type_check` (closed vocabulary, `20261102233600_audit_log.sql:109`), using `NOT VALID` then `VALIDATE` | will bite at scale |
| **C13** | **"Re-publish linked types" checks authorisation per type for the caller.** A principal with format-write but no publish right on type X gets a per-type refusal for X; the per-type report shows no type the caller could not already list (I4, within the tenant) | will bite at scale |

## 4. Counter table and migration

### 4.1 Shape
`record_counters` gains `scope_key text NOT NULL DEFAULT ''` with `CHECK (length(scope_key) <= 128)`, and `period_end timestamptz NULL`. PK becomes `(tenant_id, entity_definition_id, scope_key)`. `tenant_id` still leads (I1); still a counter table, no sequences, no per-tenant DDL (I8). Non-default scope keys are `e<epoch>|<parts>`, e.g. `e1|d:202611|f:BLR`. The upsert's INSERT arm sets `period_end`; the UPDATE arm never touches it. Measured at 1.43M rows: 9 shared hits, 0.37 ms on an existing scope row, the same as a default-scope row [PG].

### 4.2 Expand / contract [PG 1], [TA-7]

| Step | What | Measured |
|---|---|---|
| M1 | `ADD COLUMN scope_key … DEFAULT ''`, `ADD COLUMN period_end`, `CHECK … NOT VALID`, then `VALIDATE` | 12 ms + 7 ms, catalog only |
| M2 | `CREATE UNIQUE INDEX CONCURRENTLY` on `(tenant_id, entity_definition_id, scope_key)` and the partial `(period_end) WHERE period_end IS NOT NULL`, each in its own `NO TRANSACTION` migration, drop-if-exists then create (as `00005`) | 50 ms at 20k rows |
| Deploy | 3-column `ON CONFLICT`, still writing only `''`. Same PR rewrites `internal/tenancy/isolation/registry.go:454-462` on the new key [Sec C4] | old and new binaries coexist **for existing counters only**; a type's first create can fail with 23505 until M3, so R1 retries once ([[ADR-005-E1-first-create-race-in-expand-window]]) |
| M3 | After the deploy is fully rolled out: `SET LOCAL lock_timeout = '2s'`; drop the old PK and `ADD PRIMARY KEY USING INDEX`; retry up to 5 times with backoff | 1.6 ms |

- **Publish gate.** Until M3 has run, a non-default format publish is refused with `numbering_templates_unavailable` (422). The check reads the catalog (does `record_counters_pkey` cover `scope_key`?) and is cached per process with a **60 s TTL**, so processes booted before M3 open within a minute without a restart [TA-5].
- **No rollback past M3.** After M3 the old binary's `ON CONFLICT` no longer matches. The runbook marks M3 as the point of no return.
- **Pinned tests.** (1) `Compile("{prefix}{seq:7}", grow)` gives `scope_key == ""` and epoch 0. (2) Integration: seed a counter at 42 under the old key, run the new create path, expect `INC0000043`. (3) Migration test: between M2 and M3 the old 2-column and new 3-column upserts both succeed and a non-empty `scope_key` insert fails; after M3 the 3-column upsert succeeds. [PG 1, 2]

### 4.3 Growth and purge [PG 3–5]
- At publish, the **product** of the cardinalities of all `{field:…}` tokens in a format must be ≤ 1,000. Live rows per type and epoch are then ≤ 1,000 field-only rows, or ≤ 1,000 × (open periods + 2 days) for a period scope.
- Purge is **mandatory** for any scope with a period part: `nava_purge` deletes `WHERE period_end < now() - interval '2 days'` in batches of 1,000 through the partial index (11.6 ms per batch; 0.63 ms for the empty last batch, against 210+ ms without the index; 6.5 MB at 912k rows; 99.1 % of counter upserts stay HOT).
- Paired `nava_purge` SELECT and DELETE policies on that predicate (the `outbox` pattern). `nava_app` gets no DELETE. Field-only rows (`period_end IS NULL`) are never purged. This is an ADR-001 A.3 allow-list entry with its own isolation-registry rows: purge cannot see or delete open-period or field-only rows [Sec C4].
- **Why 2 days:** the grace must exceed the widest UTC-offset span, 26 h (UTC−12 to UTC+14), which is how far a time-zone change can reopen a local date after its `period_end`. An early purge would surface as a 500 from the records unique index, never a silent duplicate.

## 5. Isolation tests required [Sec C4]
Rewritten on the new key, plus:
- tenant A takes from tenant B's counter at a non-empty `scope_key` → Denied;
- `nava_worker` starts a tenant A row on tenant B's type with a scope → Denied;
- `nava_purge` cannot see or delete open-period or field-only rows; `nava_app` has no DELETE;
- `number_formats` and `numbering_epochs`: cross-tenant read, publish and link → Denied;
- the allocator pool refuses to start under a non-`nava_app` role.

**Required tests on the implementation PR that adds the format tables** [Sec C8–C13], in `internal/tenancy/isolation/registry.go`, beside the generic CI-2 checks:
- `number_formats`, `number_format_versions`, `numbering_epochs`: read tenant B's rows by id and by key, as `nava_app` and `nava_worker` → `OwnRowsOnly`;
- insert a tenant B format, version or epoch as `nava_app` → `Denied`;
- `nava_worker` INSERT into any of the three → `Denied`;
- `nava_app` UPDATE or DELETE on versions or epochs → `Denied` (no grant);
- **FK case:** tenant A publishes a definition version whose link names `(tenant A, format id of B, 1)` → `Denied`;
- `nava_onboard` seeds a format for an active tenant B → `Denied` (if seeding exists);
- **HTTP:** a tenant A token on `GET`/`PUT /…/number-formats/{B's id}` → byte-identical 404 to a missing id; a forged `tenant_id` in the format publish body → `StoredUnderA`;
- when plugins land (C11): a tenant A plugin token with a forged tenant header → `StoredUnderA`; a plugin token with format-write only that tries to bump a type's pinned version → 403.

## 6. Plugins

| | Shape | Verdict |
|---|---|---|
| **P-a** | A plugin authors a `number_format` through the public API, with a scoped token | **in scope** (§3.2); admin-UI publish ships first, plugin writes stay off until the §3.7 C11 contract is met |
| **P-b** | A plugin runs per create, before the lock, returning the rendered parts; the platform keeps the counter | **deferred** |
| **P-c** | A plugin owns the counter or runs inside the lock | **rejected** (I8, chair's hard line) |

**P-b revival trigger** (all three): (1) a plugin-runtime ADR exists with sandbox, timeout and determinism guarantees; (2) a tenant needs a format §3.1 cannot express, recorded on an issue with the example; (3) measurements show at least 20 ms of create p95 headroom at 8 clients.
**P-b minimum contract** (its own RFC): a pure function of `(definition version, candidate record fields, instant)`, no I/O, 20 ms timeout, deterministic, refuse-on-error (never fall back to another format); input is the candidate record after the plugin principal's field masks; runtime isolated per tenant with no shared state; output limited to parts from a declared value set, with the platform building `scope_key`. It would plug in as a new token type inside a `number_format`, so the link model does not change. [Sec]

## 7. Invariants

| Invariant | Effect | Status |
|---|---|---|
| I1 tenant wall | `record_counters`, `number_formats`, `numbering_epochs` lead with `tenant_id`, FORCE RLS; allocator via `tenancy.AllocTx`, tenant from `TenantOf(createTx)`, role `nava_app` | holds |
| I4 one write pipeline | Allocation is a pipeline step after authorisation; `x-nava-immutable` enforced on every write path; L2′ rejected for violating the order | holds |
| I5 masks | field-token rule 4, coupled mask publish, re-check on mask-set change | holds, with AR-TICK-653-01 |
| I7 idempotency | replay returns the stored number; a failed create's retry takes a new one | holds |
| I8 contracts, no DDL, no sequences | counter table with a key column; formats are configuration; P-a only | holds |
| D23 | out of scope | — |

## 8. Accepted risks (rows added to `docs/security/accepted-risks.md` in this PR)

| ID | Sev | Risk | Trigger to revisit |
|---|---|---|---|
| AR-TICK-653-01 | 🟡 Low | Numbers issued before a field was masked keep showing the field's code (audit rows, outbox events, sent emails, external references) | A tenant asks to recall numbers; a regulated tenant masks a field already in numbers |
| AR-TICK-653-02 | 🟡 Low | A per-scope sequence reveals the volume per scope (e.g. records per branch this month) to anyone who can read one record in it; within the tenant only | A tenant treats per-scope volume as confidential |
| AR-TICK-653-03 | 🟡 Low | The L2s no-duplicate guarantee assumes WAL replays as a prefix (crash recovery, physical replicas, PITR) | **Logical replication on `record_counters`** reopens this ADR [CTO] |
| AR-TICK-653-04 | ⚪ Info | The allocator pool adds 4 connections per API replica to `max_connections` and PgBouncer | More than ~10 API replicas, or connection pressure on PostgreSQL |
| AR-S17-01 (E1) | 🟡 Low | During the R1 rollout (after M2, before M3), an R0 binary can fail a type's **first** create with 23505 when it races another first create; no duplicate number | R1 rollout or the R1→M3 gap exceeds one deploy cycle ([[ADR-005-E1-first-create-race-in-expand-window]]) |
| (product) | ⚪ Info | Gaps in numbering on `gaps` types (the default) after failed creates and after a crash; gaps are never reclaimed | A tenant's legal gapless requirement: that type sets `numbering_mode = gapless` (D-3-9, PR #551) |

## 9. Evidence
- **Spike S1** (Technical Architect): L0, L1, L2, L2s, L2′ at 8 clients, quiet and contended hosts; 1-client SQL-level create p95 (L0 31.5, L1 15.3, L2 23.6, L2s 18.9 ms); A4 crash check 3/3 with no duplicates. `docs/adr/ADR-005/spikes/technical-architect/s1/`.
- **Counter scope bench** (Postgres): migration steps M1–M3 timed, the interim and post-M3 `ON CONFLICT` errors reproduced, upsert plans at 20k and 1.43M rows, purge plans with and without the partial index. `docs/adr/ADR-005/spikes/postgres/counter-scope/`.
- **TICK-621** re-run on `4fc57183` (contended): the trigger numbers in Context.

## 10. Still owed (not blocking this ADR)
- **A3:** 1-client create over HTTP ≤ 50 ms on a quiet host, owed on TICK-621. The SQL part is 15–32 ms p95, which suggests most of the 174 ms HTTP miss is outside the database (inferred). [CTO]
- **Budgets:** with the plan, add to `docs/budgets.md`: counter wait p95 at 8 same-scope clients under L2s ≤ 2 ms (measured 0.84–1.20 ms); create p95 1 client over HTTP ≤ 50 ms (unverified); and the L1 waiver value for `gapless` types (the 50 ms 8-client row applies to `gaps` types only). [CTO]

## 11. Amendment to PLAN-003 D-3-9
D-3-9 (`docs/delivery/plan-records-path.md:88`) is replaced, on acceptance, by: *"Numbering per ADR-005: the number is allocated by `tenancy.AllocTx` on a dedicated `nava_app` pool (L2s) after authorisation and validation, keyed by `(tenant_id, entity_definition_id, scope_key)`, with the format taken from the definition version's compiled number-format snapshot. Default `{prefix}{seq:7}` (epoch 0, `scope_key = ''`). Each type chooses its mode in `entity_definitions.numbering_mode` (`text NOT NULL DEFAULT 'gaps' CHECK (numbering_mode IN ('gaps','gapless'))`), set where `number_prefix` is set and changeable later (the counter row carries on either way). For a `gapless` type it runs last inside the create transaction (the TICK-648 behaviour): a rolled-back create gives its number back, and that type's concurrent creates queue on its counter row, a cost paid only by that type and not held to the 50 ms 8-client target. **In `gaps` mode** a create that fails or rolls back after taking its number leaves that number unused, and it is **never reclaimed** (a shared free list would re-serialize every create and make numbers non-monotonic). No sequences (I8)."* The L2s path is the `gaps` mode; the per-type sentences are word for word from D-3-9 as amended in PR #551. The PLAN for this ADR edits the row.

## 12. Sequencing (for the PLAN)
1. **Already planned; the ADR plan opens no second task.** Both allocation paths are P3-36e (TICK-657) on TICK-621. P3-36e builds the `gaps` path as ADR-005 L2s (`tenancy.AllocTx`, dedicated pool, metrics, rules A1–A8) and keeps L1 for `gapless`, so there is one allocator. The mode-switch test of §2.1.1 is part of P3-36e's acceptance. Independent of formats; closes the P3-36 trigger.
2. Counter key expand/contract M1 → M2 → deploy → M3, with registry rows.
3. `number_formats`, links, `numbering_epochs` (with C8–C10, C12, C13 and their registry rows), templates (dates, width, overflow), `period_end` and mandatory purge, publish gate. Admin-UI publish only. *Depends on:* step 2 (M3 done).
   - Plugin writes to formats are a later step. *Depends on:* the plugin principal of §3.7 C11.
4. Field tokens (enum only), coupled mask publish, re-check. *Depends on:* step 3 and S10 field masks.

P-b is not planned.
