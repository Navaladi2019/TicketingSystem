---
type: adr-amendment
id: ADR-006-A3
title: "Amendment A3 to ADR-006: a filterable field may be promoted to a fixed typed slot column on records, indexed with assignment_group_id and a slot-only order key, for the band (i)+ACL cases D7 cannot bound"
status: DRAFT
date: 2026-11-28
issue: TICK-1206
rfc: none (amendment from a measurement; requested in TICK-1039 comment 01a21f9e-237e-7d2c-9ade-271e2d29bab4; evidence threads TICK-1039 01a21646-3891-7cad-92d9-e371f8b16b78 and TICK-1205 01a22128-47c1-7941-89bb-ac217e18f270)
supersedes: none
amends: "[[ADR-006-filterable-tenant-fields]] D2, D4, D7, D12, D13 and its rejected option B; [[ADR-006-A1-d13-fanout-bands-350ms]] A1-6 (the (b) fallback) and its rejected option (b)"
source: "TICK-1039 arms (PR #920, 660e243a); TICK-1205 slot write path, DDL and plan shape (PR #995, run e1f1da99, doc 0b733511); Solution Architect plan-shape prototype of the steered order key (2026-11-28). Archived in `docs/adr/ADR-006/spikes/a3-promoted-slots/`"
deciders: Solution Architect (author; verdict after critique); Technical Architect (critique; evidence review of TICK-1039 and TICK-1205); Security & Multi-Tenancy Reviewer (critique of A3-6 merge construction and A3-8); CTO (tie-break only if contested); Technical Lead (requester; schedules SP-4)
---

# ADR-006 amendment A3: promoted filter slots

> **Status: DRAFT, rev 1** (2026-11-28). TICK-1205's numbers are in. They settle the write side: the slot DDL is online, the backfill passes D10, `k` = 2. They do not settle the read side: the broad-value page still misses with either index set, because the planner prefers a recency-blind walk of `records_list_updated` / `records_list_group` (TICK-1205 S-b1, S-b2; Technical Architect's review, findings 1, 2, 5). Rev 1 answers that with a design change: a **slot-only order key** (A3-6), so the slot index becomes the only path that can return rows in list order. A toy prototype shows the plan shape. **Critique starts when SP-4 (A3-10) has scored it on the TICK-1205 harness.** A3-7's scope and the shape rule's threshold G come from SP-4.
>
> **Invariants source.** The project description defines no invariants, so the invariant check is incomplete. Checked against [[ADR-001-platform-blueprint]] §1, I1–I9, as ADR-006, A1 and A2 are.
>
> **One deliberate change to the request.** The Technical Lead asked for "a promoted column per field". A column added per tenant field is tenant-motivated DDL, which I8 forbids. This amendment promotes a field into one of a **fixed set of typed slot columns** that ship once, for every tenant, in a normal migration. The field-to-slot mapping is configuration.

## Rev 1 changelog

| What | Rev 0 | Rev 1 | Why |
| --- | --- | --- | --- |
| `k` | 2, provisional | **2, fixed** (8 slots) | TICK-1205 item 3: `ADD COLUMN` × 8 is 30 ms with a 6 ms `AccessExclusiveLock`; item 4: 8 filled indexed slots cost +25% insert p95, 32 double it |
| Order key | `updated_at` | **`fx_list_at`**, one new column equal to `updated_at`, present only in the slot indexes | S-b1/S-b2: the planner walks `records_list_updated` or `records_list_group` and filters 100k+ rows. With a key no other index carries, it cannot |
| Indexes per slot | group only | **group + plain**, both keyed on `fx_list_at, id` | TA finding 2: the group index cannot bound a no-ACL page. Write cost of the plain index is SP-4 (d) |
| Page shape | compiled; merge "if S-b1 passes" | **chosen by rule**: no ACL → plain scan; visible groups ≤ G → per-group merge; > G → compiled | S-b1/S-b3/S-c3: neither shape wins alone |
| Promotion cap | quota ≤ `k` per value type | also **≤ 4 promoted fields per type** (8 filled slot indexes) | Item 4: the measured +25% point |
| Backfill | D10 batches | + `REINDEX CONCURRENTLY` of both slot indexes before `ready` | Item 2: backfill bloats the index 1.5–2.1× |
| Statistics | MCV per slot | + `ALTER COLUMN … SET STATISTICS 1000` per slot | S-c1: a 13-row tenant outside the MCV list is estimated at 1 |
| Gate | TICK-1205 | **SP-4** (A3-10) | Steering is unmeasured on the harness |

## In plain words

When an agent opens a ticket list filtered on a custom field, D7 finds the matches through a side table and checks each against what the agent may see. That breaks when the matches are old or mostly hidden: on a 400,000-ticket type it took up to **2.2 seconds**. Storing the value on the ticket row, in an index that also holds the assignment group, passed every case on TICK-1039.

TICK-1205 then measured what that costs. Saving a ticket costs about what D7 already costs; adding the spare columns takes 30 ms of locking; copying a field's values in slows live saves by 18.7%, under the 25% limit. That part is settled.

It also found a new problem. When the newest tickets of a type do not carry the value, the database ignores the new index and walks tickets newest-first, filtering out 100,000 rows before it fills a page. It chooses that walk because it assumes the value is spread evenly over time. PostgreSQL has no way to tell it otherwise (no "hints"). Rev 1 removes the choice: lists on a promoted field sort by a copy of the update time that **only the new indexes contain**, so the walk can no longer produce rows in order. In a small prototype the bad page went from 191 ms to 0.2 ms. One scored run on the real harness (SP-4) confirms it before anyone is asked to approve.

## What was measured

### TICK-1039: the read side (arms)

TICK-1039, PR #920, commit `660e243a`, 2026-11-28 00:04Z, on a host that passed the gate (floor p50 0.54–1.52 ms, no `nava`/bench containers). Seed 800k records, a 400k type, 8 clients, 20 s per cell, `work_mem = 4 MB`, `nava_app` with `force_custom_plan`. Field `pk` (num, one value per record). a6 = `agent-1` (sees 6.2%), a50 = `a50-1` (sees 50.1%, a superset of a6). Line 150 ms p95; within 10% is provisional (A1-8).

| Fixture (p95 ms) | D7 | (a) adaptive | (b) plain | **(b) + group** |
| --- | --- | --- | --- | --- |
| anti-correlated 2% (8,000 oldest), a6 | 192.9 | 1,178.0 | 71.7 | **66.8** |
| anti-correlated 2%, a50 | 216.1 | 2,362.3 | 63.6 | **69.4** |
| anti-correlated 2%, no ACL | 355.2 | 4,376.6 | 50.9 | **76.6** |
| none visible (26,593 records), a6 | 124.8 | 1,249.9 | 78.6 | **27.2** |
| none visible, a50 | 2,223.2 | 2,662.0 | 79.8 | **30.5** |
| partly correlated (494 visible), a6 | 211.3 | 349.7 | 99.7 | **62.9** |
| partly correlated, a50 | 420.9 | 534.9 | 101.8 | **74.1** |
| 19,999 invisible side rows, a6 | 135.3 (prov.) | 1,204.6 | 71.9 | **29.6** |
| 19,999 invisible, a50 | 242.0 | 2,301.6 | 69.9 | **31.0** |
| broad 50% (200,000), a6 | 405.6 | 84.9 | 79.4 | **131.3** |
| broad 50%, a50 | 212.4 | 85.8 | 163.6 (prov. miss) | **60.9** |
| **cells that pass** | 2 / 11 | 2 / 11 | 10 / 11 | **11 / 11** |

**Plan shapes** (`tick1039-arms-explain.md`, A1-2's rule: the plan, not the number, draws the line):
- **D7, none visible, a50 (2,223 ms):** walks 200,113 rows of `records_list_updated`, probing the side table per record, and finds nothing. The fan-out is the visible set.
- **(a), every non-broad cell:** the pre-count finds > 4,500 side rows (cap fixed before the run), so the walk-and-probe shape runs; with anti-correlated or invisible values it probes `rfv_num_pkey` 24,339–392,102 times (1.18–1.96M buffers). A1-6 (1) bounds the page only for values independent of `updated_at`; A1-6 (3) built fixtures that are not.
- **(b)+group, broad 50%, a6 (131.3 ms, the worst passing cell):** page = bitmap scan of `spike_records_pk_group` (15,482 rows = matches ∩ visible) and a top-N heapsort, 42 ms single-client, 14,267 buffers. Count = index-only scan stopping at the 1,001 cap, 1.4 ms, `Heap Fetches: 0`. **The page is bounded by matches ∩ visible, not by the page size** (R-c).
- **(b) plain, broad 50%, a50 (163.6 ms):** the count joins all 200,000 index entries of the value; without the group key nothing narrows it.

### TICK-1205: the write side, DDL and plan shape of the slot form

PR #995, run `e1f1da99`, 04:54–07:00Z, PG 16.10 with P3-01 settings, 800k records with a 400k type, 8 clients. The host gate passed (floor p50 1.66 / 2.58 ms, no `nava*` or bench containers). The Technical Architect confirmed items 1–4, D10, S-b3 and S-c1 as sound for this amendment, and corrected the cause of the broad-value miss (findings 1 and 2, adopted in `0b733511`).

| Check | Result | Bears on |
| --- | --- | --- |
| 1. Write amplification | Insert: no cost over D7 at the same n. Promoted-field update: +2 to +18 ms p95, 1.4–3.1× WAL. Unrelated update: unchanged with the group index. No update is HOT in any arm, because `records` already indexes `updated_at` and `version` | R-g |
| 2. Size per field | Group index 37 MB fresh, 55–78 MB after the backfill (400k rows); plain 29 MB; heap +0.4–3.5 MB | A3-5 reindex, R-g |
| 3. DDL | `ADD COLUMN` × 8: 30 ms, `AccessExclusiveLock` seen for 6 ms, writers waited ≤ 30 ms. CIC on an empty slot 1.5–2.1 s, `ShareUpdateExclusiveLock` only. Live p95 104.3 → 107.3 ms, 0 errors | A3-2, A3-4 |
| D10 backfill | Live p95 +18.7% (101.7 → 120.8 ms) against a +25% budget: **pass**. 2m41s–3m17s per 400–451k rows when run alone; 1.5–3.9 GB WAL; every update non-HOT | A3-5, R-e |
| 4. Slots per table | Insert p95 62.9 / 78.6 / 99.5 / 131.5 / 192.6 ms at 0 / 8 / 16 / 32 / 64 filled, indexed slots; planning 5.4 → 8.4 ms | `k`, A3-3 cap |
| S-b1 per-group merge | **Fail**, 1 of 6. Anti-correlated arms are bounded (≤ 33 rows per slot scan). Broad arms never use the slot index: each walks `records_list_group` and filters 1,993–3,618 rows | A3-6 |
| S-b2 no-ACL broad 50% | **Fail on both index sets** (290.0 / 386.0 ms). The planner walked `records_list_updated` and filtered out 100,514 / 125,697 recent rows that lack the value | A3-6, A3-2 index set |
| S-b3 2.6M-record type | **Pass** with the merge (105.3 ms, 4 slot scans of ≤ 56 rows); the compiled shape took 6,762 ms | A3-6 |
| Heap fetches under churn | Count plans 215–478, page plans 0, at 73% all-visible | A2 count |
| S-c1 shared slot | No plan changed. Estimates within 2.1×, except a 13-row tenant estimated at 1; its p95 was 36.7 ms | R-d |
| S-c2 / S-c3 | Selective operators pass for a6. The slot fixes TICK-1009's 10%/20% ACL band (7 of 8 compiled cells pass). A broad `in` (17%) and every a50 merge cell miss | A3-6 rule, A3-7 |

**What the plans say** (Technical Architect findings 1, 2, 5): every broad-value miss is the same event. The planner costs a `LIMIT 51` walk of an index ordered by `updated_at` as if the value were spread evenly over time, but the newest ~100k rows of the type do not carry it. This is the anti-correlation case (like pk 40). It is a real production shape, for example a value that was common and is rare now, and heap maintenance does not change it. The slot index could have returned the page directly; the planner did not choose it. The merge bounds the page only when its arms land on the slot index, which happened at 2.6M records and not at 800k.

### Solution Architect prototype: the steered order key (plan shape only)

`docs/adr/ADR-006/spikes/a3-promoted-slots/sa-steer-prototype.md`. PG 16.10, a toy 800k seed shaped like TICK-1205's (the newest 100k rows of the type lack the broad value), forced RLS, a non-owner role. **Not a scored run:** no host gate, and no latency claim beyond a single-shot `EXPLAIN ANALYZE`.

| Query | Order key | Plan | Rows filtered / buffers / time |
| --- | --- | --- | --- |
| no ACL, broad value, after churn | `updated_at` | `records_list_updated` walk with a Filter | 100,026 / 202,773 / 191 ms |
| no ACL, broad value, after churn | slot-only key | plain slot index, Index Cond only | 0 / 4 / 0.2 ms |
| no ACL, page 2, cursor on `timezone('UTC', updated_at)` | expression key | slot index; the cursor is a **Filter** (`timezone` is not leakproof) | 27,200 / – / 7.2 ms |
| no ACL, page 2, cursor on a leakproof key | leakproof key | slot index; the cursor is in the **Index Cond** | 0 / 7 / 0.1 ms |
| 25-arm merge (a50), broad, after churn | slot-only key | 25 scans of the slot group index, 3 rows each | 0 / – / 0.3 ms |

Two facts carry into A3-6:

1. If no index except the slot indexes carries the order key, a slot index is the only ordered path. The only other plan the planner can pick is a bitmap scan plus a top-N sort of the matches, which is bounded by the number of matches.
2. The cursor stays in the Index Cond only if its comparison is leakproof. A plain `timestamptz` column gets that from built-in operators. An expression key would need a custom `LEAKPROOF` function. Creating one needs a superuser (`nava_migrator` is `NOSUPERUSER`, ADR-001 I3), and it goes against TICK-339's rule never to mark such functions leakproof. So the key is a column.

The prototype could not reproduce the merge arms falling back to `records_list_group`: on its seed the unsteered arms already chose the slot index. SP-4 tests the steered arms on the seed where they fell back.

**Not measured yet** (and so not claimed, A1-2): the steered key on the TICK-1205 harness; the plain index's write cost without TICK-1205's run-order confound; the shape rule's threshold G; agents whose visibility comes mostly from shares, requester or assignee.

## Decisions

| # | Decision |
|---|---|
| **A3-1** | **A promoted tier for filterable fields.** A field that is `x-nava-filterable` may also be **promoted**. A promoted field is stored in one **slot column** on `records`, in addition to its D3 side rows. Its list filter compiles to a predicate on that column (A3-6) instead of D7's `EXISTS`. Fields that are not promoted keep D7, the A1 bands and A2 exactly as they are. |
| **A3-2** | **Slots are fixed, typed and shared; k = 2.** One platform migration adds these as nullable columns with no default, which is catalog-only in PG 16: `fx_num_1..2 double precision`, `fx_text_1..2 text COLLATE "C"`, `fx_ts_1..2 timestamptz`, `fx_bool_1..2 boolean` (the D3 leakproof types), and **`fx_list_at timestamptz`**, the slot order key (A3-6). Each slot has **two partial B-trees**, both `WHERE fx_<t>_<n> IS NOT NULL`: the **group index** `(tenant_id, entity_definition_id, fx_<t>_<n>, assignment_group_id, fx_list_at DESC, id DESC)` and the **plain index** `(tenant_id, entity_definition_id, fx_<t>_<n>, fx_list_at DESC, id DESC)`. Both are built `CONCURRENTLY` on all-NULL columns (1.5–2.1 s each, TICK-1205 item 3). Each slot gets `CREATE STATISTICS (mcv) ON tenant_id, entity_definition_id, fx_<t>_<n>` and `ALTER COLUMN … SET STATISTICS 1000`. A partial index costs a write only on rows where its slot is filled, so the save cost scales with the number of promoted fields per type, not with `k`. Raising `k` is a platform migration and needs a new item-4 measurement above 8 filled slots. |
| **A3-3** | **Which fields get promoted: only those an operator approves, only scalar ones, at most 4 per type.** Promotion is requested with `"x-nava-filterable": "promoted"` on a top-level scalar property (422 `promoted_field_not_scalar` for arrays). Publish checks a per-type promoted quota in `tenant_limits`. The **default is 0**. An operator raises it under D12/D25 governance (an audited change, no DDL) when the A1-4 slow-list log shows that tenant's type in band (i)+ACL, or when a type passes 400k records. The quota is capped at `k` per value type and at **4 per type in total**: 4 fields × 2 indexes = 8 filled slot indexes, which is TICK-1205 item 4's +25% insert p95 point. Over quota → 422 `promotion_quota_exceeded`. |
| **A3-4** | **Who runs DDL: nobody, per field.** The slot columns, `fx_list_at`, the indexes and the statistics are one normal migration, authored by the Postgres & Data Modeling Specialist, security-reviewed, and applied by the deploy migrator. Promoting a field is configuration: `record_index_fields` gains `slot smallint NULL` and `slot_state ∈ {backfilling, ready, removing} NULL`, with a unique key `(tenant_id, entity_definition_id, value_type, slot)`. I8 holds. |
| **A3-5** | **Maintenance and backfill reuse ADR-006's machinery.** A `BEFORE INSERT OR UPDATE OF data, updated_at ON records FOR EACH ROW` trigger, named so it fires after the trigger that sets `updated_at`, does two things. It sets `NEW.fx_<t>_<n>` with `record_field_coerce` for each slot of the row's type that is in `{backfilling, ready}`. It sets `NEW.fx_list_at := NEW.updated_at` when any slot of the row is non-NULL, and NULL otherwise. D4's side-row trigger is unchanged. Promotion runs `FieldIndexPromote(tenant_id, field_id)` from an outbox row: set `backfilling`, D9 barrier, D10 batches of 1,000 that set the slot and `fx_list_at` (no change to `version`, `updated_at` or any audited column), then **`REINDEX INDEX CONCURRENTLY` on both slot indexes** (this recovers the 1.5–2.1× backfill bloat, about 4 s per index at 400k), then set `ready`, barrier. Demotion: set `removing`, barrier, batched `SET fx_… = NULL` (and `fx_list_at = NULL` where no slot remains), verify, clear. D10's pause rule (+25%) applies; TICK-1205 measured +18.7%. |
| **A3-6** | **The query: a slot-only order key and a shape rule.** When the field's `slot_state = ready`, the compiler emits `r.fx_<t>_<n> <op> $v`, orders by **`r.fx_list_at DESC, r.id DESC`** and puts the keyset cursor on `(r.fx_list_at, r.id) < ($c, $i)`. `fx_list_at` equals `updated_at` by construction, so the order users see does not change. But no index except the slot indexes carries it, so a walk of `records_list_updated` or `records_list_group` can no longer produce the page in order (prototype, fact 1). When the slot is not `ready`, the compiler emits D7's `EXISTS` and the usual order, so there is no new 422. The shape is chosen from inputs the compiler already has. **(S1) No row ACL:** one scan of the plain index. **(S2) A row ACL with ≤ G visible groups:** a per-group merge, with one `ORDER BY … LIMIT 51` arm per distinct visible group on the group index, plus the requester, assignee and share arms as today, under a Merge Append, in one statement (D7's one-statement rule holds). **(S3) More than G groups:** the compiled shape (`assignment_group_id = ANY(…)`), a bitmap scan of the group index plus a top-N sort of matches ∩ visible (R-c). G is provisionally **8** and SP-4 fixes it. The merge is built inside `authz.ReadBranches` and the store, never through TICK-1205's test-only `listQuery` hook. Group IDs are de-duplicated before the split, and a cross-tenant test covers it (Security review of PR #995). A slot column and `fx_list_at` are read **only** as predicates and order keys, never in a select list, payload, export or count source (D1 extended); the D7 static test gains that rule. CI-6c's oracle gains a slot-vs-`EXISTS` equality test for each shape. A trigger test asserts `fx_list_at = updated_at` on every promoted row after an insert, an update and a backfill. |
| **A3-7** | **Budget.** A promoted field is held to **150 ms p95 in every band D13 and A1 define, with and without a row ACL**. It is accepted at 150 ms only where a scored cell with its plan exists (A1-2). After TICK-1205 that covers the write side (D10, items 1–4). On the read side it covers selective operators for a6, the 10%/20% compiled cells with an ACL (7 of 8), and the 2.6M merge. The broad-value cells, the anti-correlated a25/a50 cells and the no-ACL cells are scored as **unmeasured** until SP-4 reports on the steered shapes. Non-promoted fields keep A1's lines. |
| **A3-8** | **Isolation.** Slot columns and `fx_list_at` are columns of `records`, so forced RLS covers them with no new policy. Comparisons are on the D3 leakproof types and `timestamptz`, so they push into the Index Cond as `nava_app`; CI-1 and CI-2 gain the slot operators and the cursor. A masked field still answers 400 `masked_field` before any predicate (D6 order). Grants: `nava_app` gets no slot privilege, because the trigger sets `NEW` (CI-5b verifies this); `nava_worker` gets `UPDATE (fx_…, fx_list_at)` only, for A3-5. `db/audited-tables.yaml` leaves these columns out of `audited_columns`, with the reason "derived from records.data and updated_at, rebuildable"; that is a security-review change (D-P10). No custom `LEAKPROOF` function is added. The worst a bug can do is mis-filter or mis-order inside its own tenant (R8). |
| **A3-9** | **What A1-6 becomes.** (a) fails (2 / 11) and is not built. A1's rejected option (b) and ADR-006's rejected option B reopen in A3-2's form. D7 stays the default shape. R-a's revisit trigger now leads to promotion (A3-3). |
| **A3-10** | **Gate before critique: SP-4, one scored run on the TICK-1205 harness** (same seed, the churned heap as TICK-1205 left it, the host gate, 8 clients). **(a) DDL:** the A3-2 migration with `fx_list_at` and both indexes per slot; wall time and locks. **(b) Read:** rerun S-b1 (a6/a25/a50 × broad 50% and anti-correlated 2%), S-b2 broad and `num in` 17%, S-b3, and the a50 cells of S-c2/S-c3, each on S1/S2/S3 with the steered key. *Pass:* every S1 and S2 arm uses a slot index and reads ≤ 51 + ε rows, and p95 ≤ 150 ms. **(c) G:** the crossover between S2 and S3 per fixture at a6/a25/a50 (3/13/25 groups). G is the largest group count at which S2 is not worse than S3 in any cell. **(d) Write:** SP-3's ABAB run (group only vs group + plain, 3 reps each, same heap state) for an insert, a promoted-field update and an unrelated update, with `fx_list_at` maintained. *Material cost:* the mean difference is larger than the spread within each arm. **(e) Backfill with `REINDEX CONCURRENTLY`:** D10 live p95 and the index size after it. SP-1 is not needed: with the steered key, the cause of the walk no longer decides the plan. (b) covers SP-2. |

**Not changed:** D1, D3, D5, D6's check order, D9, D11, R-b, R8, R9, R11, R12; A2's count budget and `total_state` (they apply to promoted fields too, and the count keeps its own index-only path); A1's lines for non-promoted fields. I1 holds. I3 holds: no superuser object and no new grant to `nava_app`. I8 holds (A3-4).

## Rejected options

| Option | Why it lost |
|---|---|
| **A column added per promoted field** (arm (b) literally) | Tenant-motivated `ALTER TABLE` breaks I8. Columns accumulate across tenants, and each add takes an `AccessExclusiveLock` (74 ms, 66 ms of it queued, TICK-1205 item 3). |
| **D25's per-tenant expression index** | `->>` is not leakproof, so under forced RLS the comparison stays a Filter (RFC-015 §1). |
| **Promote every filterable field** | Each promotion adds two index writes to every save of its type, and item 4 shows insert p95 doubling at 32 filled slots. Opt-in at quota 0 puts the cost only where the slow-list log shows the need. |
| **(a) the adaptive shape** | 2 / 11, up to 4,377 ms. |
| **Group index only** (rev 0) | It cannot bound a no-ACL page: with no group predicate it either sorts every match or walks (TA finding 2). S-b2 does not support it. |
| **Plain index only** | 10 / 11 on TICK-1039: the a50 broad count reads all 200,000 entries. Without the group key, a per-group arm cannot be bounded. |
| **Order by `updated_at` and rely on the planner** (rev 0) | The planner's `LIMIT` costing assumes the value is spread over time and picks the walk: S-b1 broad 159–559 ms, S-b2 290–386 ms, S-b3 compiled 6,762 ms. |
| **An expression order key on `timezone('UTC', updated_at)`** | It steers page 1, but its cursor is not leakproof and stays a Filter, so page N reads N × 51 rows (prototype: 27,200 rows filtered on page 2). |
| **A custom `LEAKPROOF` order-key function** | It needs a superuser to create; `nava_migrator` is `NOSUPERUSER` (I3), and TICK-339 forbids marking such functions leakproof. A column gives the same plan with built-in operators. |
| **`pg_hint_plan`, or planner GUCs per query** | `pg_hint_plan` is a new extension on the shared database, an operational dependency every managed host would have to offer. `SET LOCAL` settings are not index-specific. The order key removes the planner's choice instead of overriding it. |
| **A periodic `VACUUM` or fill-factor policy** | Not the cause: the filtered rows are live (TA finding 1). |
| **Always use the per-group merge** | It loses on a50 with a selective value (114 → 313 ms) and in every a50 S-c3 cell: 25 arms plus their count cost 30–43 ms before the first row. |
| **Cap band (i)+ACL at a visible-set size** (A1-7 S2) | D7 also fails without an ACL (355.2 ms) and for a 6% agent, so a breadth cap does not bound it. It stays the fallback if A3 is rejected. |
| **Drop a promoted field's side rows** | It loses the fallback during promotion and demotion. TICK-1205 shows the double write adds no insert latency over D7. |
| **An external search index** | Isolation and ACL leave RLS (I1, I5), as with ADR-006 option D. |

## Risks and accepted residuals (proposed)

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R-c | Shape S3 (more than G visible groups) is bounded by matches ∩ visible, not by 51. TICK-1039 sorted 15,482 rows in a 131.3 ms cell. | Technical Lead | SP-4 (b) S3 cells miss; or the slow-list log shows a promoted field above 150 ms for an agent with more than G groups |
| R-d | Tail estimates on a shared slot: a tenant outside the MCV list is estimated at 1 (13× off in S-c1). No plan changed. `SET STATISTICS 1000` widens the list. | Postgres specialist | A promoted filter plans a type-wide walk (> 1 s) in telemetry |
| R-e | Promotion rewrites one heap tuple per record of the type: 1.5–3.9 GB of WAL and 2m41s–3m17s per 400–451k rows. The heap grew 35% over six backfills; autovacuum reuses the space but does not shrink the file. | Postgres specialist | Live-write p95 above +25% during a promotion |
| R-f | Only the assignment-group arm is index-assisted. The requester, assignee and share arms filter the slot after their own index. | Technical Lead | The slow-list log shows a promoted field above 150 ms for a share-heavy agent |
| R-g | Each promoted field adds two index writes to every save of its type. With the group index, a promoted-field update costs +2 to +18 ms p95 and 1.4–3.1× the WAL. The plain index's share is SP-4 (d). | Postgres specialist | Insert rate under R-b's 40% floor with promotions in place |
| R-h | `fx_list_at` drifts from `updated_at` if a writer bypasses the trigger (a disabled trigger, or a direct `UPDATE` by an owner role). The order is then wrong inside the tenant; nothing leaks. | Postgres specialist | The trigger test, or a nightly count of `fx_list_at <> updated_at`, is non-zero |

## Competitive Gate (ADR-008)

**Result:** scored after SP-4. Expected PASS: X3 (speed) moves from "150 ms except two 350 ms bands" to "150 ms for promoted fields in every measured band", which narrows accepted risk CP-FW-01 for large tenants.

## Open questions for critique

1. The shape rule (A3-6): a visible-group threshold G, or something the compiler would have to estimate. I recommend the count: it is deterministic and already in hand.
2. Operator approval (quota default 0), or a tenant admin within a small default quota. I recommend the operator: the cost lands on the shared write path, and the signal (A1-4) is observable.
3. `fx_list_at` as a stored column, or an expression key. The column costs 8 bytes on promoted-type rows and one assignment in the trigger. The expression needs a superuser `LEAKPROOF` function.
4. The cap of 4 promoted fields per type.

## Evidence

- TICK-1039 arms and plans: `docs/adr/ADR-006/spikes/a3-promoted-slots/tick1039-arms.md` and `tick1039-arms-explain.md`; host and contention in `tick1009-1039.*`.
- TICK-1205: `docs/adr/ADR-006/spikes/a3-promoted-slots/tick1205/tick1205-slots.md` (a copy of PR #995's "Arm (b) write path" section at `0b733511`), `tick1205-slots-raw.md` and `tick1205-slots.host`. One plan per cell: `docs/data/field-index-measured/tick1205-slots-explain.md` on PR #995.
- The steered order key prototype: `docs/adr/ADR-006/spikes/a3-promoted-slots/sa-steer-prototype.md`.
- SP-4: pending.
