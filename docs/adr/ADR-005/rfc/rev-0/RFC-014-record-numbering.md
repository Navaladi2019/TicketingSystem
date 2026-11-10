---
type: rfc
id: RFC-014
title: Record numbering — create latency under concurrency, configurable formats, reset scope, plugin boundary
status: DRAFT (rev 0)
date: 2026-11-09
issue: TICK-653
amends: PLAN-003 D-3-9
author: Solution Architect
---

# RFC-014: Record numbering

> **Invariants source.** The project description on Multica is empty. I loaded the invariants from `docs/adr/ADR-001-platform-blueprint.md` §Invariants (I1–I9, `main` at `89e0f84d`). The checks below are made against that text.
> **Numbering assumption.** The highest RFC in the repo and on remote branches is RFC-013, so this one is RFC-014. The outcome is a new ADR (ADR-005 assumed) that amends PLAN-003 D-3-9.

## 0. Recommendation

1. **Latency first. Allocate the number outside the create transaction (L2)** if the human accepts gaps. In L2 the counter row is locked only while the server runs one upsert and commits it, with no client round trip inside the lock. My estimate is a same-scope wait p95 of 10–20 ms, which is the only option likely to meet create p95 ≤ 50 ms at 8 concurrent same-type creates. **If the human keeps gapless,** ship L1 (gapless with no round trip inside the lock) and **record a waiver**. My estimate is that L1 still misses at 8 same-scope clients. Section 3 sets out this choice; the RFC does not make it.
2. **Format: a declarative template** per entity type (`{yyyy}{MM}{seq:4}`, `{field:branch}_{yy}_{MM}_{seq:3}`). It is compiled at publish into `(scope parts, head, seq width, tail, overflow)`. The counter statement keeps its shape: one upsert keyed by `(tenant_id, entity_definition_id, scope_key)`, with the number built in SQL as `head || seq || tail`, as `numberSQL` already does. **The format adds zero statements and zero round trips to the lock window.** The default template `{prefix}{seq:7}` with `overflow: grow` gives exactly today's D-3-9 output, so existing numbers are not migrated.
3. **Plugin:** keep only **P-a**, a plugin that authors a template at publish time through the public API. Drop **P-c** (a plugin owns the counter, or is called inside the lock): it violates I8 and the chair's hard line. **P-b**, a per-create plugin call outside the lock, does fit under the hard line. I recommend deferring it with a named trigger (§5). It adds a network hop to a create budget we already miss, there is no plugin runtime to host it, and it lets tenant code create counter rows without bound.
4. Sharding or partitioning `records` (ADR-001 D23) is out of scope. This RFC covers counter throughput only.

## 1. Context, measured

| Fact | Source |
|---|---|
| Counter: `record_counters(tenant_id, entity_definition_id, next)`, PK (tenant, type), RLS forced | `db/migrations/20261101190000_records_pipeline.sql:139` |
| The number is taken in the create's write batch (the upsert comes first) and built in SQL. The lock is held until the commit batch | `internal/records/store/batch.go:193-198` and `internal/records/pipeline/pipeline.go:610` at `4fc57183` |
| Format: `<prefix><7 digits>`, growing past 7 digits. Prefix `^[A-Z]{2,6}$`, set once | `internal/records/store/numbering.go`, `db/migrations/20261031150000_entity_definitions.sql:45` |
| TICK-621 re-run (`4fc57183`, **contended host**, transport floor p50 4.79 / p95 16.88 ms) | TICK-621 comment 01a1c12f |
| — counter hold p50 / p95 | 11.5 / 25.4 (agent), 11.0 / 18.4 (admin), 13.0 / 75.9 ms (small tenant) |
| — counter **wait** p95 at 8 same-type clients | 310 ms (agent), 237 ms (admin) |
| — create p95 at 8 clients | 408 / 301 / 264 ms against a 50 ms target |
| — create p95 at **1 client** | 173.8 ms (p50 54.5). This is not a counter queue: 8 round trips plus two commit fsyncs on a slow floor |
| There is no plugin runtime (no plugin host, no Goja) and no tenant time zone column in the repo | `git grep` on `main` |

Two consequences shape every option:
- **The queue is about 7 holds deep.** With 8 clients on one row, the p95 wait is about 7 × hold. To get a wait under 20 ms, the hold must be about 3 ms or less. Today's hold is one client round trip plus a commit fsync, so halving it is not enough.
- **No numbering option fixes the 1-client miss.** If the clean 1-client repeat (still owed on TICK-621) also misses, the 50 ms target fails for reasons outside numbering. I flag this as a precondition rather than solve it here.

## 2. Options for the D-3-9 latency problem (lead table)

"Inside the lock" means the time from the counter upsert to the release of its row lock.

| | Option | Inside the lock | Same-scope wait p95 at 8 clients (estimate unless measured) | Create p95 at 8 clients | Gaps | Verdict |
|---|---|---|---|---|---|---|
| **L0** | Today: gapless, lock until the create commits | 1 client round trip + commit fsync (measured hold p50 10–13 ms) | **237–310 ms measured** | **264–408 ms measured** | none | misses |
| **L1** | Gapless, but the write batch and `COMMIT` go in **one pipelined send**, with the idempotency completion folded in (TICK-621 trigger 1) | server-side only: insert record, audit and outbox rows, then commit fsync. No network time | about 40–55 ms (assumes the hold falls to about 6–8 ms once one 4.8 ms round trip is removed). Lower on a quiet host. **Spike S1** | about 80–100 ms contended. About 50–70 ms quiet (estimate) | none | probably misses at 8 same-scope clients; needs a waiver |
| **L2** | **Gaps allowed. Allocate in its own tiny transaction** (`BEGIN; set_config(app.tenant_id, local); upsert; COMMIT` in one pipelined send), after the checks pass, on a small dedicated allocator pool. Then the create transaction inserts with that number | server-side only: one upsert + commit fsync of a one-row transaction | about 10–20 ms (7 × 1–3 ms). **Spike S1** | the 1-client create + 1 round trip + ≤ 20 ms. Meets **if** the 1-client create meets | when a create fails after allocation (commit error, crash, conflict) | **recommended if gaps are accepted** |
| L2′ | Like L2, but allocate before the database checks (no second pool) | same as L2 | same as L2 | same as L2 | **also** every create refused by unique or group checks | simpler; more gaps. Fallback if the allocator pool is rejected |
| L3 | Blocks of numbers cached per server per (tenant, type, scope) | none per create | ~0 | 1-client create | large gaps on restart or deploy. Numbers not in time order across servers. The cache grows with scope count (branch × day) | rejected for M1. Trigger: one scope needs > ~500 creates/s sustained |
| L4 | Assign the number after commit (outbox → update) | none | 0 | 1-client create | none, but the create response and the first audit entry carry no number. A second write and audit entry per create | rejected: a create must return its number |
| L5 | Keep L0 or L1 and **replace the target**, e.g. "≤ 50 ms at 8 clients spread across types or scopes; same-scope creates serialized, p95 ≤ X" | as L0 or L1 | as L0 or L1 | as L0 or L1 | none | the honest form of gapless: a recorded waiver |

**L2 details that matter for correctness**
- **I7 / idempotency (TICK-482, TICK-498):** the replay check runs before allocation, so a replay returns the stored record and its number and allocates nothing. A retry of a *failed* create allocates a new number. That is a gap, not a duplicate.
- **I1:** the allocator transaction sets `app.tenant_id` with `set_config(..., true)` (local to the transaction) on its own connection, never at session level on a pooled connection. RLS on `record_counters` is unchanged.
- **Pool exhaustion:** the allocator uses a separate pool (proposed size 4, configurable), so a full main pool cannot deadlock waiting on allocation. Allocation has a 200 ms timeout. On timeout the create fails with `ErrRetryable`.
- **Date tokens and the instant:** the allocator returns `now()` from its own transaction, and the create stores that instant as `created_at`. The date in the number and `created_at` then always agree, including at midnight (§4.3).

**Format-agnostic by construction.** In L0, L1 and L2 the statement inside the lock is the same upsert with a third key column, `scope_key`, which is computed before the lock. Rendering is `head || lpad-or-grow(seq) || tail` in SQL (L0/L1) or in Go (L2). No template choice adds a statement, a lookup or a round trip to the lock.

**Scoped counters spread contention only when the scope varies.** A monthly reset does not help: everyone shares this month's row. A field scope (`{field:branch}`) does: 8 creates across 8 branches never wait on each other.

## 3. Gapless vs gaps allowed: the human's product call

The RFC presents this choice and does not decide it. Costs are measured against TICK-621 where a measurement exists.

| | **Gapless** (L0 → L1) | **Gaps allowed** (L2) |
|---|---|---|
| What users see | Every number is used: INC41, INC42, INC43 | A failed create can skip one: INC41, INC43 |
| Create p95, 8 same-type clients | measured 264–408 ms (L0). Estimate 50–100 ms after L1. **Needs a waiver of the 50 ms target** (L5) | estimated ≤ 50 ms if the 1-client create meets. Spike S1 confirms |
| Ceiling per tenant, type and scope (≈ 1 / hold) | ~80–100 creates/s at the measured 10–13 ms hold. About 150 creates/s after L1 (estimate) | ~300–1,000 creates/s (estimate, fsync-bound) |
| At the human's 2-trillion-record scale (≈ 6,300 creates/s across all tenants) | fine when spread out. A single national-portal type in one tenant would not keep up | one scope can take ~5–10× more |
| Who needs it | Legal sequences, e.g. invoice numbers in some jurisdictions | Application or case references (visa, PF) usually do not |
| Build cost | L1 only (small) | L2: allocator pool, a separate transaction, and passing the instant into the create (moderate) |

**Per type?** It is technically possible: the template carries `numbering: gapless | gaps_allowed`, and the pipeline takes L1 or L2. The cost is two create paths and a doubled create test matrix, including the cross-tenant tests. If the human wants it per type, it is supported. If not, one platform-wide mode is cheaper.

**Question for the human (to be asked by the CTO):** gapless, gaps allowed, or per type? And if gapless, what same-scope create target replaces 50 ms?

## 4. Format model (declarative template)

### 4.1 Grammar (validated at publish, JSON Schema plus a Go compiler)
`template := (literal | token)+`, with exactly **one** `{seq:N}`.

| Token | Meaning |
|---|---|
| `{prefix}` | the type's existing `number_prefix` |
| `{yyyy}` `{yy}` `{MM}` `{dd}` | date parts of the record's creation instant in the numbering time zone (§4.3) |
| `{seq:N}` | the counter, zero-padded to N digits (1 ≤ N ≤ 12) |
| `{field:<key>}` | the value of a record property (§4.4) |
| literal | `[A-Z0-9_\-/.]`, at most 16 characters per run |

Limits: rendered length ≤ 64. Unknown tokens are refused, so a later `{plugin:…}` token stays an additive change. Customer examples: `yyyymm____` → `{yyyy}{MM}{seq:4}`, `yyyymmdd__` → `{yyyy}{MM}{dd}{seq:2}`, `BRANCHNAME_yy__MM_` → `{field:branch}_{yy}_{seq:2}_{MM}`, where `seq` may sit in the middle.

**Compiled form**, stored on the published definition version: `scope_parts`, `head` (the token or literal list left of seq), `width`, `tail`, `overflow`, `numbering`, `epoch` (§4.6). Per create, the platform renders `head` and `tail` and the `scope_key` before the lock. Inside the lock it only fills in `seq`.

### 4.2 Reset scope and counter rows
- `scope_key` = the canonical join of the template's date granularity and field values, e.g. `e1|d:202611|f:BLR`. An empty scope (`e0|`) is today's per-type counter.
- **Reset scope = the tokens shown.** An optional `reset` may be *coarser* (yearly reset with `{MM}` shown stays unique). A *finer* reset than shown, e.g. a daily reset with no `{dd}`, is refused at publish because it would collide.
- Table change: `record_counters` PK becomes `(tenant_id, entity_definition_id, scope_key)`, with `scope_key text NOT NULL DEFAULT ''` and a `CHECK (length(scope_key) <= 128)`. The policy and FK are unchanged. It still leads with `tenant_id` (I1), and it is still a counter table, not a sequence (I8). Existing rows get `''`. The table holds a few thousand rows, so the PK rebuild is trivial. Postgres specialist to confirm the migration shape.
- **Who creates scope rows:** the same upsert, on the first create in a scope. Nothing is created in advance.
- **Growth:** period scopes add ≤ 366 rows per type per year (daily). Field scopes add rows per distinct value. **Bound:** `{field:…}` is allowed only on properties with an enumerated or reference-code value set (≤ 1,000 values, checked at publish). Free text never makes a scope.
- **Purge:** a period scope row whose period ended more than 2 days ago can be deleted by `nava_purge`. A create can never land in a closed period, because the instant is taken in the database (§4.3). Field-only rows are never purged. This is optional; the volume is small.

### 4.3 Dates
- **Instant:** the database `now()` of the transaction that takes the number: the create transaction in L0/L1, the allocator transaction in L2. In L2 that instant also becomes `created_at`. The application clock is never used.
- **Time zone:** there is no tenant time zone today. Proposal: a tenant setting `time_zone` (IANA name, default `UTC`) and an optional per-template override. Changing it changes future scope keys only. Numbers stay unique because the date text is in both the scope key and the number.

### 4.4 Field tokens: Security review required
A property may appear in a number only if **all** of these hold, each checked at publish:
1. It is required at create.
2. It is **immutable after create**. This is a new schema keyword (`x-nava-immutable`), enforced by the write pipeline (I4).
3. Its rendered value matches `^[A-Z0-9]{1,10}$` after a declared mapping (enum value → code).
4. It is **not field-masked for any role that can read the record** (I5). A number is shown wherever the record is referenced: lists, links, notifications, email subjects. A masked field inside a number leaks.

**Open for Security:** what happens when an ACL change *later* masks a field already used in a template? My proposal is that the ACL publish is refused while a live template references the field. The alternative is to leave existing numbers as they are and record the exposure as an accepted risk. Security decides.

### 4.5 Overflow
Per template, `overflow: grow | refuse`.
- **`grow`** is allowed only when the template parses unambiguously: every variable-width token is followed by a literal separator outside its character set, or ends the number. Example of the danger: `{field:br}{seq:3}` with br=`A1`, seq 1100 gives `A11100`, and br=`A11`, seq 100 gives `A11100`. That is a collision. The publish step refuses it.
- **`refuse`**: the create fails with a typed `number_space_exhausted` error (409), naming the type and scope. It is audited and alerted on at 90 % use of the width.
- The records unique index on `(tenant_id, entity_definition_id, number)` remains the backstop. A violation there is a 500-class bug, never silent.

### 4.6 Template changes across versions
- Existing numbers are never rewritten.
- Each distinct compiled template gets an `epoch` (1, 2, …), which is part of `scope_key`, so a new template never continues an old counter wrongly.
- **Disjointness rule at publish:** a new template must have a different literal or `{prefix}` head from every earlier epoch of the type. Otherwise the publish is refused, e.g. `{yyyy}{seq:5}` then `{yyyy}{MM}{seq:4}` could collide. The unique index is the backstop.
- A create uses the template of the definition version it is written under, which is already recorded on the record.

## 5. Plugin option, assessed against I8 as written

I8: *"Tenant code, plugins and AI reach data only through the capability contract or the public API, with a scoped token, and never reach the database or Temporal directly … Configuration, not code."* Chair's hard line: **no plugin call inside the counter lock.**

| | Shape | Inside the lock? | I8 | Other costs | Verdict |
|---|---|---|---|---|---|
| **P-a** | A plugin *authors a template* and submits it through the public publish API. The platform validates and compiles it as in §4 | no | **fits**: a public API client producing configuration | none new; nothing to build beyond §4 | **keep** (the roadmap hook) |
| **P-b** | Per create, before the lock: the plugin returns `(scope_key, rendered head/tail)`, and the platform keeps the counter | no | fits the letter (no database access; the input is the candidate record through the capability contract) | +1 network hop and timeout on every create, against a 50 ms budget we already miss. No plugin runtime or sandbox exists to host it. A tenant-chosen `scope_key` creates counter rows without bound. Uniqueness of rendered output cannot be proven at publish. On error the create must be **refused**: falling back to the default format would silently mix number families in one type | **defer.** Trigger: (1) a plugin-runtime ADR exists with sandbox, timeout and determinism guarantees, **and** (2) a required format cannot be expressed in §4.1 |
| **P-c** | The plugin owns or increments the counter, or is called inside the lock | **yes** | **violates I8** (database access) and the chair's hard line | n/a | **drop** |

If P-b is revived, its minimum contract is a pure function of `(definition version, candidate record fields, instant)`, with no I/O, a 20 ms timeout, deterministic output, a `scope_key` validated against a declared value set, and refuse-on-error. That would be its own RFC.

## 6. Invariants touched

| Invariant | Effect | Status |
|---|---|---|
| I1 tenant wall | `record_counters` keeps `tenant_id` first in the PK, with FORCE RLS. The L2 allocator sets the tenant GUC locally per transaction on its own pool | holds; Security to confirm the L2 GUC path |
| I4 one write pipeline | Allocation is a pipeline step. The immutability of field tokens is enforced in the pipeline | holds |
| I5 masks | §4.4 rule 4 | **open for Security** |
| I7 idempotency | A replay returns the stored number. A failed create's retry takes a new one | holds |
| I8 no DDL / no sequences / plugins | Counter table, a new key column, no per-tenant DDL. Plugins: P-a only | holds |
| D23 | Out of scope | — |

## 7. Unverified assumptions and the spike

| # | Assumption | How to settle |
|---|---|---|
| A1 | In pgx v5, pipelining `write batch + COMMIT` (L1) or `BEGIN…COMMIT` (L2) in one send keeps the network round trip out of the row-lock hold | **Spike S1** (Technical Architect, 2 h): the bench-acl-perf create scenario, 8 clients, same type and scope, L0 vs L1 vs L2, on a **quiet** host. Report hold and wait p50/p95 and create p95 |
| A2 | The hold and wait estimates in §2 and §3 | S1 |
| A3 | The 1-client create meets 50 ms on a quiet host | the clean TICK-621 repeat. A precondition for every option |
| A4 | Enumerated-property detection and an `x-nava-immutable` keyword are feasible in the current metadata engine | Postgres & Data Modeling Specialist / JSON Schema review |

## 8. Sequencing (for the plan, not decided here)
1. Latency: L1, or L2 per the human's call. This is independent of formats and unblocks the P3-36 trigger.
2. `scope_key` + templates (dates, seq width, overflow, epoch). Default template = today.
3. Field tokens, after the Security sign-off.
P-b is not planned.

## 9. Questions for reviewers
- **Human (via the CTO):** gapless, gaps allowed, or per type? If gapless, what same-scope target? Confirm the required formats.
- **Technical Architect:** A1 and A2 through S1. Also: is L2's dedicated allocator pool the right way to avoid a pool deadlock, or is L2′ preferable?
- **Security & Multi-Tenancy Reviewer:** §4.4 (field disclosure, an ACL change after publish), the L2 GUC path, and the P-b boundary should it be revived.
- **Postgres & Data Modeling Specialist:** the PK migration on `record_counters` and the purge predicate.
