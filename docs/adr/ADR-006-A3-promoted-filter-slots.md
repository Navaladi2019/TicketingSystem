---
type: adr-amendment
id: ADR-006-A3
title: "Amendment A3 to ADR-006: a filterable field may be promoted to a fixed typed slot column on records, indexed with assignment_group_id, for the band (i)+ACL cases D7 cannot bound"
status: DRAFT
date: 2026-11-28
issue: TICK-1039
rfc: none (amendment from a measurement; the request is the Technical Lead's TICK-1039 comment 01a21f9e-237e-7d2c-9ade-271e2d29bab4; evidence thread 01a21646-3891-7cad-92d9-e371f8b16b78)
supersedes: none
amends: "[[ADR-006-filterable-tenant-fields]] D2, D4, D7, D12, D13 and its rejected option B; [[ADR-006-A1-d13-fanout-bands-350ms]] A1-6 (the (b) fallback) and its rejected option (b)"
source: "TICK-1039 arms, PR #920 (scored on 660e243a, head 60d1f0dd): `docs/data/field-index-measured.md` § TICK-1039, `tick1039-arms-explain.md`; archived in `docs/adr/ADR-006/spikes/a3-promoted-slots/`. Write-path cost: TICK-1205 (not yet run)"
deciders: Solution Architect (author; verdict after critique); Technical Architect (critique; evidence review of the TICK-1039 cells on PR #920); Security & Multi-Tenancy Reviewer (critique of A3-8); CTO (tie-break only if contested; escalation point if TICK-1205 makes the write cost unacceptable); Technical Lead (requester)
---

# ADR-006 amendment A3: promoted filter slots

> **Status: DRAFT, rev 0** (2026-11-28). **Not for critique yet.** It goes to the Technical Architect only after TICK-1205 reports the write-path cost of (b), which is the reason A1 rejected (b). The numbers marked *TICK-1205* below are placeholders for that report. A3-10 lists what TICK-1205 must also cover before this record can be scored.
>
> **Invariants source.** The project description defines no invariants, so the invariant check is incomplete. Checked against [[ADR-001-platform-blueprint]] §1, I1–I9, as ADR-006, A1 and A2 are.
>
> **One deliberate change to the request.** The Technical Lead asked for "a promoted column per field". A column added per tenant field is tenant-motivated DDL, which I8 forbids: the only tenant-motivated DDL I8 allows is an operator-approved `CREATE INDEX CONCURRENTLY … WHERE tenant_id = …`. So this amendment promotes a field into one of a **fixed set of typed slot columns** that ship once, for every tenant, in a normal migration. The field-to-slot mapping is configuration. The index key and the plan shape are the ones arm (b)+group measured; what differs is that a slot column is shared by many tenants and types, so its statistics differ (R-d, gated in A3-10).

## In plain words

When an agent opens a ticket list filtered on a custom field, D7 finds the matching tickets through a side table and then checks each one against what the agent may see. That works when the matches are many and recent. It breaks when the matches are old or mostly hidden from the agent: the database walks the agent's whole visible set looking for 51 rows that are not there. On a 400,000-ticket type that took up to **2.2 seconds** for an agent who sees half the tickets. The "adaptive" fix A1 planned was worse (up to 4.4 s).

The one design that passed every case (11 of 11, at 27–131 ms against 150) stores the field's value on the ticket row itself, in an indexed column whose index also holds the ticket's assignment group. The database can then jump straight to "tickets with this value in my groups, newest first".

This amendment allows that, but only for fields an operator promotes, and without adding a column per field. The platform ships a small fixed set of spare typed columns ("slots") on `records`. Promoting a field assigns it a free slot and copies its values in. Unpromoted fields keep D7 unchanged. What we accept: every promoted field adds an index write to each ticket save for that type, and copying values into a slot rewrites the ticket rows. How much that costs is what TICK-1205 measures, and this amendment is not accepted until it does.

## What was measured

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

**Not measured** (and so not claimed, A1-2): operators other than `num eq`; text and ts slots; 10–20% values with an ACL on a promoted field; a slot shared by several tenants or types; a heap that has not just had `VACUUM FULL`; agents whose visibility comes mostly from shares, requester or assignee rather than groups; types above 400k; any write-path cost.

## Decisions

| # | Decision |
|---|---|
| **A3-1** | **A promoted tier for filterable fields.** A field that is `x-nava-filterable` may also be **promoted**. A promoted field is stored, in addition to its D3 side rows, in one **slot column** on `records`. Its list filter compiles to a predicate on that column (A3-6) instead of D7's `EXISTS`. Fields that are not promoted keep D7, the A1 bands and A2 exactly as they are. |
| **A3-2** | **Slots are fixed, typed and shared.** One platform migration adds `k` nullable columns per value type: `fx_num_1..k double precision`, `fx_text_1..k text COLLATE "C"`, `fx_ts_1..k timestamptz`, `fx_bool_1..k boolean` (the D3 leakproof types). Each slot has **one global partial B-tree**: `(tenant_id, entity_definition_id, fx_<t>_<n>, assignment_group_id, updated_at) WHERE fx_<t>_<n> IS NOT NULL`, the key arm (b)+group measured. Each slot gets `CREATE STATISTICS (mcv) ON tenant_id, entity_definition_id, fx_<t>_<n>`. The slot means a different field for each (tenant, type); the mapping lives in the registry (A3-4). `k` is provisionally **2 per value type (8 columns, 8 indexes)** and is fixed from TICK-1205 item 4 (how many a table can carry). A column with no default is a catalog-only change in PG 16; the indexes are built `CONCURRENTLY` over columns that are all NULL. |
| **A3-3** | **Which fields get promoted: only those an operator approves, and only scalar ones.** Not every filterable field. Promotion is requested per field with `"x-nava-filterable": "promoted"` on a top-level scalar property (422 `promoted_field_not_scalar` for arrays). Publish checks it against a per-type promoted quota in `tenant_limits` whose **default is 0**. An operator raises it under D12/D25 governance (audited `tenant_limits` change, no DDL) when the A1-4 slow-list log shows that tenant's type in band (i)+ACL, or when a type passes 400k records. The quota cannot exceed `k` per value type. Over quota → 422 `promotion_quota_exceeded`. Reason: every promoted field adds an index write to each save of that type (TICK-1205 item 1), and most types never reach the band. |
| **A3-4** | **Who runs DDL: nobody, per field.** The slot columns, indexes and statistics are one normal migration, authored by the Postgres & Data Modeling Specialist, security-reviewed, and applied by the deploy migrator like every other migration. Promoting a field is configuration: `record_index_fields` gains `slot smallint NULL` and `slot_state ∈ {backfilling, ready, removing} NULL`, with a unique key `(tenant_id, entity_definition_id, value_type, slot)`. Raising `k` later is a platform migration, never a tenant request, and needs TICK-1205's carry limit. I8 holds: no tenant change runs DDL. |
| **A3-5** | **Maintenance and backfill reuse ADR-006's machinery.** A `BEFORE INSERT OR UPDATE OF data ON records FOR EACH ROW` trigger sets `NEW.fx_<t>_<n>` with `record_field_coerce` for each slot in `{backfilling, ready}` for the row's type (D4's `SET` list; a value that does not coerce sets NULL and counts, D5). D4's side-row trigger is unchanged, so a promoted field still has its side rows. Promotion runs `FieldIndexPromote(tenant_id, field_id)` from an outbox row: flip `slot_state` to `backfilling`, D9 barrier, D10 batches that `UPDATE records SET fx_… = record_field_coerce(data -> key, …)` (no change to `version`, `updated_at` or any audited column, so `records_audit_commit_check` does not fire), flip to `ready`, barrier. Demotion: `removing`, barrier, batched `SET fx_… = NULL`, verify none remain, clear the slot. The D10 pause rule (live-write p95 within +25%) applies. |
| **A3-6** | **The query.** When the field's `slot_state = ready`, the compiler emits `r.fx_<t>_<n> <op> $v` in the list's `WHERE` and in each ACL `UNION ALL` arm, inside the same single statement (D7's one-statement rule holds). Otherwise it emits D7's `EXISTS`, so a field filters normally while its slot backfills or is removed: no new 422. For a single-valued field the two are equivalent (`EXISTS (v.value = x)` ⇔ `fx = x`); CI-6c's row-ACL oracle gains a slot-vs-`EXISTS` equality test. A slot column is read **only** as a predicate, never in a select list, payload, export or count source (D1 extended); the D7 static test gains that rule. The list's page shape for promoted fields is fixed by TICK-1205 S-b1 (one ordered arm per visible group under a Merge Append) if S-b1 passes; otherwise the measured bitmap-plus-sort shape stays and R-c applies. |
| **A3-7** | **Budget.** A promoted field is held to **150 ms p95 in every band D13 and A1 define, with and without a row ACL**. It is accepted at 150 ms only where a cell with its plan exists (A1-2): today `num eq` at 2–6.6% and 50%, ACL breadth 6% and 50%, 400k per type. Every other cell listed under "Not measured" is scored as unmeasured until A3-10's cells report. Non-promoted fields keep A1's lines. |
| **A3-8** | **Isolation.** Slot columns are columns of `records`, so `records`' forced RLS covers them with no new policy. Comparisons are on the D3 leakproof types, so they push into the Index Cond as `nava_app` (seen in every (b) plan; CI-1 and CI-2 gain the slot operators). A masked field still answers 400 `masked_field` before any predicate (D6 order); masks stay row-independent (R9). Grants: `nava_app` gets no slot privilege (the trigger sets `NEW`; to verify in CI-5b); `nava_worker` gets `UPDATE (fx_…)` only, for A3-5. `db/audited-tables.yaml` leaves the slot columns out of `audited_columns` with the reason "derived from records.data, rebuildable", which is a security-review change (D-P10). The worst a bug can do is mis-filter inside its own tenant, as R8. |
| **A3-9** | **What A1-6 becomes.** A1-6's question is answered: (a) fails (2 / 11), (b)+group passes (11 / 11). (a) is closed and is not built. A1's rejected option (b) and ADR-006's rejected option B are reopened in the form A3-2 gives them. D7 stays the default shape. R-a's revisit trigger now leads to promotion (A3-3), not to the adaptive shape. |
| **A3-10** | **Gate before critique.** TICK-1205 reports items 1–4, S-b1–S-b3 and heap fetches under churn, **plus** these cells on the same harness, which this amendment needs and TICK-1205 does not yet list: (1) the **slot** form, not a per-field column: the `ADD COLUMN` + `CREATE INDEX CONCURRENTLY` cost of the A3-2 migration (all-NULL), and the A3-5 backfill's WAL, bloat and live-write p95 for 400k rows; (2) a **shared slot**: two types (or tenants) mapped to the same slot with different value distributions, checking the estimate and plan per type (R-d); (3) operators `in`, `lt`/`gte` and a `text eq` slot, a6/a50/no ACL; (4) 10% and 20% values with an ACL on the slot. A3-7's scope and `k` are then filled in from the results, and only then does this go to critique. |

**Not changed:** D1 (side tables stay the derived, rebuildable index for every filterable field), D3, D5, D6's check order, D9, D11, R-b, R8, R9, R11, R12; A2's count budget and `total_state` (they apply to promoted fields too); A1's lines for non-promoted fields. I1 (tenant isolation through RLS) holds; I8 holds (A3-4).

## Rejected options

| Option | Why it lost |
|---|---|
| **A column added per promoted field** (what arm (b) literally did) | Tenant-motivated `ALTER TABLE` breaks I8. Columns accumulate per tenant field across all tenants (1,600-column ceiling, catalog churn), and every add takes an `ACCESS EXCLUSIVE` lock on `records`. Would need a CTO escalation of I8; the slot form gets the same index without it. |
| **D25's per-tenant expression index** (`CREATE INDEX CONCURRENTLY … ((data->>'k')) WHERE tenant_id = …`) | The I8-sanctioned path, but `->>` is not leakproof, so under forced RLS the comparison stays a Filter and the index is not used (ADR-006 context, RFC-015 §1). |
| **Promote every filterable field** | Each promotion adds an index write per save; D12 already measured 5 side-table fields at 43–63% of the baseline insert rate. Most types never reach band (i). Opt-in at quota 0 makes the cost appear only where the slow-list log shows the need. |
| **(a) the adaptive shape** | 2 / 11, up to 4,377 ms; it moves the fan-out from the matches to the visible set (A1-6 (3) fixtures). |
| **(b) with the plain index only** | 10 / 11: the a50 broad count reads all 200,000 entries (163.6 ms). Reopens only if TICK-1205 S-b2 shows both indexes are needed. |
| **Cap band (i)+ACL at a visible-set size** (A1-7 S2 fallback) | D7 also fails without an ACL (355.2 ms) and with a 6% agent (192.9–405.6 ms) in these fixtures, so a cap on breadth does not bound it. |
| **Keep D7 and rely on A2's count budget** | A2 bounds the count, not the page. The 2,223 ms cell is the page walk. |
| **Drop a promoted field's side rows** | Saves one write per save, but loses the fallback during promotion and demotion (a field would answer 422 again) and adds a second write path. Revisit only if TICK-1205 shows the double write breaks R-b's 40% floor. |
| **An external search index for promoted fields** | Same reasons as ADR-006 option D: isolation and ACL leave RLS (I1, I5). |

## Risks and accepted residuals (proposed)

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| R-c | The (b)+group page is bounded by matches ∩ visible, not by 51: 15,482 rows sorted in the 131.3 ms cell. It grows with the type. | Technical Lead | TICK-1205 S-b1/S-b3 fail; or the A1-4 slow-list log shows a promoted field above 150 ms |
| R-d | A slot's statistics mix many (tenant, type, field) meanings. The `(tenant_id, entity_definition_id, slot)` MCV statistics are expected to keep estimates per type, but this is unmeasured. A bad estimate could choose a `records_list_updated` walk, the D7 failure. | Postgres specialist | A3-10 (2) fails; or a promoted filter plans a type-wide walk (> 1 s) in telemetry |
| R-e | Promotion and demotion rewrite one heap tuple per record of the type (WAL, bloat until vacuum, row locks held per batch). *Cost: TICK-1205.* | Postgres specialist | Live-write p95 above +25% during a promotion |
| R-f | Only the assignment-group ACL arm is index-assisted. The requester, assignee and share arms filter the slot after their own index; in the fixtures they were tiny. | Technical Lead | Slow-list log shows a promoted field above 150 ms for an agent whose visibility is mostly shares |
| R-g | Each promoted field adds an index write to every save of its type. *Cost: TICK-1205 item 1.* | Postgres specialist | Insert rate under R-b's 40% floor with promotions in place |

## Competitive Gate (ADR-008)

**Result:** to be scored after TICK-1205. Expected PASS: X3 (speed) moves from "150 ms except two 350 ms bands" to "150 ms for promoted fields in every measured band", which narrows accepted risk CP-FW-01 for large tenants. X3's row is updated in the same change when this is accepted.

## Open questions for critique

1. `k`: 2 per value type is a guess until TICK-1205 item 4.
2. Operator approval (quota default 0) vs letting a tenant admin promote within a small default quota. I recommend operator approval because the cost lands on the shared write path and the trigger to need it is observable (A1-4).
3. Whether S-b1's per-group merge shape is worth its compiler complexity, or R-c is accepted at 400k.

## Evidence

- Scored cells, fixture and arm definitions: `docs/adr/ADR-006/spikes/a3-promoted-slots/tick1039-arms.md` (copy of `docs/data/field-index-measured.md` § TICK-1039, PR #920).
- Every arm × fixture plan, `EXPLAIN (ANALYZE, BUFFERS, SETTINGS)` as `nava_app`: `docs/adr/ADR-006/spikes/a3-promoted-slots/tick1039-arms-explain.md` (copy from PR #920). The (b)+group broad a6 page and count are its § "(b)+group broad 50% (200,000), a6".
- Host and contention: `docs/adr/ADR-006/spikes/a3-promoted-slots/tick1009-1039.host`, `tick1009-1039.contention`.
- Write-path cost: TICK-1205 (pending).
