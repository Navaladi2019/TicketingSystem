# Region access at 800k: `org_unit_id = ANY(S)` (SP-1, S31 rerun)

Scope: TICK-932, PLAN-011 P11-01, ADR-010 D7 and R1. The question: on the
P3-36 800k bench, does the support-agent incident list (`sort=-updated_at`,
first page) stay at or under the 150 ms p95 list budget
(`docs/quality/acl-perf-800k.md:16`, 146.2 ms today, so 3.8 ms of headroom)
when the read predicate gains D7's leaf
`org_unit_id IS NOT NULL AND org_unit_id = ANY($S)`, for |S| = 10, 100, 1,000
and 10,000?

## S31 rerun after A2 (TICK-1037, R1 per ADR-010-A2 A2-11)

**Status: Step 0 open (one clean baseline pass of two). Region rows built
and smoke-checked; no row scored.**
R1's baseline stays 146.2 ms until a second clean pass exists. Nothing below
is a verdict.

### Step 0: baseline passes (agent list, `sort=-updated_at`, no `under` leaf)

The CTO granted an idle-host window: 2026-11-22, 02:40 to 04:10 UTC
(TICK-1037). Each pass is the unchanged P3-36 handler row: a fresh 800k seed,
8 clients, 20 s, behind the transport floor gate and the TICK-1040 container
watch.

```
NAVA_PERF_ONLY='^agent list, sort=-updated_at$' sh bench/acl-perf/run.sh
```

| Pass | Start (UTC) | Floor p50 / p95 | n | p50 | **p95** | Host during the pass | Status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 02:41:28 | 1.05 / 1.64 | 3,451 | 44.88 | **55.48** | Only the allowed containers: `buildx_buildkit_nava-ci0` (idle), `multica-*` | **clean** |
| 2 | 02:44:54 | 1.64 / 2.27 | 2,956 / 3,963 | 51.34 / 43.25 | 76.17 / 65.74 | `nava-it-pg-68222908` (02:46), `nava-it-pg-118522456` (02:51), each with `-pgb-pooled` / `-pgb-noprep` sidecars, from another task's `scripts/ci/test-integration.sh` | **void** (CTO terms: a new `nava-*` container during a pass voids it) |

- **Pass 2 is not scored**, its second attempt included. The harness gated
  attempt 1 and waited for the containers before attempt 2. The CTO's terms
  void the whole pass, so the 65.74 ms is not patched in.
- **Pass 1 is far under 146.2 ms.** The host was idle (floor 1.05 ms against
  the 3 ms gate). The 146.2 ms (2026-11-09) and 131.96 ms passes ran on a busier
  host. One pass does not replace the baseline. The second clean pass decides,
  and a third runs if the two differ by more than 5 ms.
- Code: `TICK-946/in-my-region` at `7e1055ca` (TICK-1036 and TICK-1071 merged).
- Raw: `acl-perf-region/tick1037-baseline-pass1.md`, `-pass2.md` (header,
  plans, container samples).

### Region rows through the handler (method, for review before scoring)

A2-11 scores the **handler** p95 with a region policy. Before TICK-1037 no
harness measured that: `TestRecordsACLPerf800k` had no org tree or `under`
policy, `bench/acl-perf/region/main.go` is database-only (and calls the
deleted `authz.ExpandUnder`), and the crossover `sp1x` is the probe harness.
TICK-1037 adds the "region:" rows to the P3-36 handler test
(`internal/server/records_perf_region_integration_test.go`). They run after
every other row, so the Step 0 baseline row is untouched.

- **Timed request.** `GET /api/v1/records/<type>?sort=-updated_at` through
  `server.NewHandler`. `ResolveUnder`, the estimate read, the P3 probe
  (`ChooseListShapes`), the count and the page all run inside it.
- **Data**, on the P3-36 seed, large tenant. The SQL is ported from the
  bench harnesses, so the sets are the crossover cells':
  - the SP-1 tree (10,111 units) in `org_units`, every live path in
    `org_unit_closure`, and `records.org_unit_id` = the requester's primary
    branch (5% NULL);
  - SP-1c's district hot and thin, SP-1e's uniform m = 1,001 / 3,000 /
    10,000 and sp1x's m = 4,500;
  - `bench_rare` (500 records, one per random branch) and `bench_conc`
    (20,000 in district pre 9101). Here they are real entity types made
    through the API, not copied definitions.
  - SP-1e's "district mixed" is not built: it isn't one of TICK-1037's sets.
  - Then `VACUUM FULL`, `ANALYZE`, `pg_prewarm` and `CHECKPOINT`, as in
    region.sql.
- **N̂_e and K\*.** One cycle of the TICK-1071 refresher
  (`estimates.Refresher`), as `nava_worker`. The job lists tenants from
  `tenant_identity_binding`, which the P3-36 seed leaves empty, so every perf
  tenant gets a binding first, as a live tenant has. Each row logs |S|, m,
  N̂_e, K\* and the A2-5 shape.
- **Policy.** The P3-36 document in format 2, plus a last policy `regional`:
  `all[member_of regional, under {of: caller}]`, on `ticket`, `bench_rare`
  and `bench_conc`. Last is TICK-951's worst branch position.
  ReadBranchBound is 6.
- **Personas.** Both are in the regional group. Each row puts them in the
  set's units (`org_unit_members`).
  - **agents:** the 200 P3-36 agents (desk, mine, regional); the desk branch
    fills the page.
  - **count-only:** a principal whose only grant is `regional`. No record
    names it, and nothing is shared with it.
- **Paired, 1 client.** Each scored request alternates with its base: the
  same P3-36 agent and path without the regional group, so `regional` folds
  away (no `under` leaf, no probe). The base p95 is reported beside the row,
  for information only. The floor gate and the TICK-1040 container watch are
  P3-36's.
- **Pass:** every row, both personas, p95 ≤ 150 ms. Verdict on the worst
  persona.

**Smoke check, not scored** (2026-11-22, 03:17 to 03:40 UTC; 2 s per row;
container watch off locally; TICK-1084's `nava-it-*` containers were up).
Every row ran and returned 200. The sets match the crossover cells:

| Set | \|S\| | m | N̂_e | K\* | Shape |
| --- | --- | --- | --- | --- | --- |
| area / district / region / root | 10 / 101 / 1,011 / 10,111 | 220 / 3,379 / 37,930 / 380,048 | 400,000 | 3,613 | C3 / C3 / W0 / W0 |
| district hot / thin | 101 | 22,996 / 600 | 400,000 | 3,613 | W0 / C3 |
| uniform m = 1,001 / 3,000 / 4,500 / 10,000 | 101 | as named | 400,000 | 3,613 | C3 / C3 / W0 / W0 |
| `bench_rare` at root | 10,111 | 500 | 500 | 1,000 | A2-5a (rare, no probe) |
| `bench_conc`, regions 0–8 | 9,099 | 0 | 20,000 | 1,000 | C3 |

Its latencies (n ≈ 10 per row, contended host) are not recorded as results.
The scored run is in the CTO's second window: 2026-11-22, 08:00 to 10:00
UTC. It runs Step 0's remaining pass(es) first, on the unchanged P3-36 row,
then:

```
NAVA_PERF_SECONDS=30 NAVA_PERF_ONLY='^region:' sh bench/acl-perf/run.sh
```

That takes about 2 min of seed, 17 min of region setup and 24 rows × ~35 s.
The paired table goes to `bin/acl-perf-800k-raw-region.md`.

## S31 rerun on the real schema (TICK-951 P11-18, 2026-11-21)

**R1: FAIL, confirmed on the real migration, index and expansion.** On S31's
story branch, the region leaf costs **+94.4 ms p95 for a 1,011-unit region**
when set S comes from the runtime's own `authz.ExpandUnder` in the request.
At |S| = 10,000 it costs **+199.9 ms p95**. These are 1-client deltas against
a baseline measured in the same run. Projected onto the 146.2 ms handler p95,
that is about 241 ms and 346 ms. Even the database work alone is over the
150 ms budget: 156.9 ms and 262.4 ms p95. Under ADR-010 R1 this is the
trigger, and it goes on S31 (TICK-946). The fallback decision is still open:
SP-1b (TICK-969) tested the closure table, and the Solution Architect's
option (i), SP-1c, has not run.

What changed from SP-1:

- **Schema.** `records.org_unit_id`, its FK and the partial index
  `records_org_unit (tenant_id, org_unit_id) WHERE org_unit_id IS NOT NULL`
  now come from S31's migrations (TICK-947), not from `region.sql`. The bench
  indexes are dropped. There is no R1-index phase, since SP-1 and SP-1b showed
  the planner does not use that index where it would matter.
- **Leaf.** The region branch is the same text `authz.CompileRead` renders,
  `(r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY($n::uuid[]))`. The
  harness checks this on every expanded request.
- **Expansion.** The new "ExpandUnder + list" rows run `authz.ExpandUnder` for
  `{"under": {"of": [root]}}` inside the request transaction, then the count
  and page with the S it returns. That is the list handler's order of work.
  The "|S| = k" rows bind the first k units, as SP-1 did, so the two runs
  compare.

### 1 client, 10 s per row (scored by delta)

ms per request. Baseline is the 4-branch list with no region leaf, from the
same run: p50 37.5, p95 62.4.

| Row | Units in S | n | p50 | **p95** | Δp50 / **Δp95** | SP-1 Δp95 (plain) |
| --- | --- | --- | --- | --- | --- | --- |
| \|S\| = 10, bound | 10 | 316 | 30.5 | 48.6 | −7.0 / −13.9 | +2.7 |
| \|S\| = 100, bound | 100 | 234 | 38.0 | 75.3 | +0.6 / +12.9 | +0.8 |
| \|S\| = 1,000, bound | 1,000 | 185 | 51.2 | 91.4 | +13.7 / **+29.0** | +7.7 |
| \|S\| = 10,000, bound | 10,000 | 53 | 193.0 | **262.4** | +155.5 / **+199.9** | +157.6 |
| ExpandUnder + list, area 0/0/0 | 10 | 243 | 38.0 | 62.2 | +0.5 / −0.3 | n/a |
| ExpandUnder + list, district 0/0 | 101 | 180 | 53.8 | 84.3 | +16.3 / **+21.8** | n/a |
| ExpandUnder + list, region 0 | 1,011 | 108 | 83.5 | **156.9** | +46.0 / **+94.4** | n/a |
| ExpandUnder + list, root | 10,111: refused | 51 | 179.4 | **304.8** | +141.9 / **+242.3** | n/a |

ExpandUnder alone (BEGIN, set_config, ExpandUnder, COMMIT; floor 13.4 p50,
25.9 p95):

| Subtree | Units | p50 | p95 |
| --- | --- | --- | --- |
| area 0/0/0 | 10 | 12.9 | 24.9 |
| district 0/0 | 101 | 19.8 | 41.3 |
| region 0 | 1,011 | 41.4 | 120.0 |
| root | 10,111: refused | 222.3 | 321.7 |

### 8 clients, 20 s per row (not scored)

The floor was 5.9 ms p50 and 14.3 ms p95, against P3-36's 3 ms gate. The
P3-36 handler run refused to measure (`host busy: transport floor p50 5.29 ms
> 3 ms gate`). The |S| = 100 row is an outlier: 831 ms p95 with n = 451,
while its plan executes in 3.3 ms. That points to a contention spike during
that row, not a plan change.

| Row | n | p50 | p95 |
| --- | --- | --- | --- |
| baseline | 4,095 | 35.6 | 63.6 |
| \|S\| = 10 / 100 / 1,000 / 10,000, bound | 2,303 / 451 / 1,472 / 372 | 61.0 / 312.0 / 98.5 / 365.3 | 132.8 / 831.2 / 196.0 / 1,171.5 |
| ExpandUnder + list, area / district / region / root | 2,790 / 2,363 / 1,333 / 241 | 53.4 / 63.4 / 110.7 / 615.8 | 83.4 / 102.0 / 197.4 / 967.2 |
| ExpandUnder alone, area / district / region / root | 5,809 / 3,785 / 1,432 / 346 | 19.4 / 31.9 / 86.6 / 432.9 | 42.7 / 60.2 / 189.7 / 624.1 |

### What the rerun adds to SP-1 and SP-1b

1. **Plans: no change.** In every bound row (EXPLAIN of agent 1) each of the five branches keeps its own
   index scan under the `Merge Append`. The region branch walks
   `records_list_updated` with a filter, and the page executes in 3 to 4 ms
   at every |S|. As in SP-1, the time goes to planning and binding S, not to
   reading rows. The partial index changes nothing here.
2. **The real expansion is the larger cost below 10,000 units.** At region
   size, ExpandUnder alone is 120 ms p95 against SP-1's 28.9 ms for D7's
   bare walk. The real query also walks each seed's ancestors to drop
   tombstoned branches, and it returns `(caller, root, id)` rows instead of
   a count. (Known: these numbers. Inferred, not profiled: the extra cost is
   the walk up and shipping 1,011 rows to Go.) After subtracting the floor,
   H1 (≤ 5 ms p95) holds only at area size: about +15 ms at district size and
   +94 ms at region size.
3. **A refusal is not cheap (Breaks under load).** A caller whose subtree is
   over `authz.MaxUnderSet` is refused only after the whole walk: 10,111 rows
   reach Go, then `ExpandUnder` counts them. That costs 304.8 ms p95 for a 500
   response. A walk capped at `MaxUnderSet + 1` rows would refuse at once.
   This is routed to the `under` leaf's owner (TICK-950) as a finding. It
   does not change R1.

### Known gaps

- **The host was busier than in SP-1.** Its transport floor was 13.4 ms p50
  in the 1-client pass, against SP-1's 5.8 ms. The `nava` dev stack (13
  containers), another task's integration container (`nava-t1022-it-1108`),
  `buildx_buildkit_nava-ci0` and `multica-*` were up. That is outside P3-36's
  allow-list. They are not this run's containers, so they were not stopped.
  Absolute values are not scored. The verdict uses deltas, and the rows that
  decide it (+94 ms and +200 ms p95) are far above the noise.
  The negative deltas at 10 units are that noise.
- **One pass per row, not paired.** SP-1b's paired method would tighten the
  small deltas. It cannot change the verdict.
- **Database work only.** The handler measurement was refused by its floor
  gate, so the projection adds deltas to P3-36's 146.2 ms.

Raw reports (every plan): `bin/acl-perf-region-real-raw.md` (8 clients) and
`bin/acl-perf-region-real-1c-raw.md` (1 client). Both are attached to
TICK-951, not committed.

## SP-1 verdict (2026-11-15, bench schema)

**R1: FAIL at |S| = 10,000 with both indexes.** The region leaf adds about
136 ms p50 and 158 ms p95 per list request, even with one client. Projected
onto the 146.2 ms handler p95, that is about 300 ms, twice the budget. R1's
fallback index does not help, because the time is not spent scanning rows.

| |S| | Verdict | Why |
| --- | --- | --- |
| 10, 100 | **holds** | The added cost is within noise (≤ 3 ms p95, one client). The region branch reads ≤ 950 index entries. |
| 1,000 | **not settled; marginal** | +7.7 ms p95 with one client, against 3.8 ms of headroom. This host was contended, so the difference cannot be scored. An idle-host re-run decides it. |
| 10,000 | **fails** | +158 ms p95 with one client. Planning `= ANY(<10,000 uuids>)` costs 15–36 ms per statement, and the list runs two statements. Binding the 10,000-element array twice adds more. Execution stays under 3 ms. |

Under R1, this result triggers the closure-table fallback (ADR-010 option D).
S31 waits for a design decision. That decision belongs to the Solution
Architect. The measurement below suggests two cheaper candidates to check by
spike first: see "Implications".

## Results

### 8 clients, 20 s per row (the P3-36 load)

ms per request. A request is the list's database work: `BEGIN`,
`set_config('app.tenant_id')`, count, page, `COMMIT`.

| |S| | Index | n | p50 | **p95** | p99 | Region-branch plan (page) | Region branch: rows read / removed | Page exec / plan (ms, single EXPLAIN) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| none (baseline, 4 branches) | plain | 1411 | 82.5 | **306.7** | 436.6 | n/a | n/a | 3.7 / 1.6 |
| 10 | plain | 1443 | 96.4 | **219.7** | 369.7 | Index Scan `records_bench_org_unit` + top-N sort | 220 / 10 | 3.3 / 3.4 |
| 100 | plain | 1497 | 95.0 | **193.7** | 276.4 | Index Scan `records_list_updated`, ordered, filter | 949 / 943 | 5.2 / 3.2 |
| 1,000 | plain | 1285 | 112.7 | **228.3** | 301.2 | Index Scan `records_list_updated`, ordered, filter | 329 / 298 | 2.7 / 6.8 |
| 10,000 | plain | 374 | 419.4 | **615.6** | 674.7 | Index Scan `records_list_updated`, ordered, filter | 52 / 2 | 2.9 / **36.0** |
| 10 | plain + R1 | 2397 | 62.8 | **99.4** | 125.2 | Index Scan `records_bench_org_unit_updated` + top-N sort | 220 / 10 | 5.7 / 3.5 |
| 100 | plain + R1 | 2116 | 70.4 | **114.8** | 212.6 | Index Scan `records_list_updated` (R1 not chosen) | 949 / 943 | 6.0 / 4.2 |
| 1,000 | plain + R1 | 1807 | 83.4 | **136.5** | 173.5 | Index Scan `records_list_updated` (R1 not chosen) | 329 / 298 | 3.3 / 16.6 |
| 10,000 | plain + R1 | 402 | 393.8 | **516.1** | 579.3 | Index Scan `records_list_updated` (R1 not chosen) | 52 / 2 | 5.5 / **35.9** |

**Index per branch: yes, in every row.** Each of the five branches has its
own index scan under one `Merge Append`. No row fell back to a sequential
scan or to one sort over the whole tenant. The count statement never runs the
region branch: the desk branch alone already returns its 1,001 rows.

**This table is contended and is not scored.** The host's transport floor
(`BEGIN`, `set_config`, `COMMIT`) was 9.0 ms p50 and 30.0 ms p95 here. The
P3-36 gate allows 3 ms p50. The baseline p95 of 306.7 ms is therefore twice
the 146.2 ms of the scored P3-36 run. The "plain + R1" rows ran later, in a
quieter window, which is why even |S| = 10 looks faster there. Only the
|S| = 10,000 rows stand far above the noise.

### 1 client, 10 s per row (cost per request, no queueing)

| |S| | Index | n | p50 | **p95** | Δ p50 / Δ p95 vs baseline |
| --- | --- | --- | --- | --- | --- |
| none (baseline) | plain | 324 | 29.8 | **47.5** | n/a |
| 10 | plain | 336 | 27.2 | **50.2** | −2.6 / +2.7 |
| 100 | plain | 318 | 29.7 | **48.3** | −0.1 / +0.8 |
| 1,000 | plain | 250 | 38.7 | **55.2** | +8.9 / +7.7 |
| 10,000 | plain | 61 | 166.0 | **205.1** | **+136.2 / +157.6** |
| 10 | plain + R1 | 343 | 28.1 | **42.8** | −1.7 / −4.7 |
| 100 | plain + R1 | 304 | 31.3 | **47.5** | +1.5 / 0.0 |
| 1,000 | plain + R1 | 233 | 41.4 | **64.2** | +11.7 / +16.6 |
| 10,000 | plain + R1 | 60 | 161.4 | **235.1** | **+131.6 / +187.6** |

Transport floor in this pass: 5.8 ms p50, 10.1 ms p95.

### Subtree expansion (D7's recursive query alone)

ms per request: `BEGIN`, `set_config`, the recursive query over `org_units`, `COMMIT`.

| Subtree root | Units | 8 clients p50 / p95 | 1 client p50 / p95 |
| --- | --- | --- | --- |
| area | 10 | 12.0 / 23.3 | 10.9 / 20.4 |
| district | 101 | 16.7 / 37.1 | 16.2 / 31.7 |
| region | 1,011 | 22.2 / 41.6 | 15.9 / 28.9 |
| root | 10,111 | 79.6 / 102.4 | 33.5 / 50.8 |

After subtracting the floor (5.8 / 10.1 ms with one client), the expansion
costs about 28 ms p50 and 41 ms p95 at about 10,000 units. RFC-022 §7
assumed at most 5 ms p95 (hypothesis H1). **H1 fails above about 100 units**,
before any list runs. (Known: these numbers. Inferred: the cost is the
recursive CTE's per-level index lookups. It was not profiled.)

### Where the |S| = 10,000 time goes

Each statement was planned alone with `EXPLAIN (SUMMARY)`, on a minimal
query: one branch, `ORDER BY updated_at DESC, id DESC LIMIT 51`.

| Leaf rendering | Planning time |
| --- | --- |
| `org_unit_id = ANY('<10,000 uuids>'::uuid[])` (D7 as written) | 15.1, 18.4, 19.5, 24.1 ms (4 runs) |
| same, 1,000 uuids | 1.8, 2.0 ms |
| same, 10,000 uuids, column statistics off (`SET STATISTICS 0`) | 17.3 ms |
| `org_unit_id = ANY((SELECT $S)::uuid[])` (initplan, values hidden from the planner) | 0.24 ms |
| `org_unit_id IN (SELECT unnest($S))` | 0.41 ms |

- **Known:**
  - Planning time grows with |S|, roughly linearly (about 10× from 1,000 to 10,000). Inside the full five-branch page and count statements it is 36 ms each.
  - Column statistics do not change it, so the cost is not the MCV selectivity loop.
  - `plan_cache_mode = force_custom_plan` (db-roles) replans every request.
- **Inferred, not profiled:**
  - The rest of the +136 ms p50 is the bind and transfer of the 10,000-element `uuid[]` in each of the two statements, plus executor setup of the array.

## Implications for the design (for the Solution Architect)

1. **D7's 10,000-id cap is above what the list budget allows.** With D7 as
   written, the most |S| that measurably fits is between 100 and 1,000. 1,000
   is marginal against 3.8 ms of headroom.
2. **R1's index `(tenant_id, org_unit_id, updated_at DESC, id DESC)` is not
   the remedy.** The planner chooses it only at |S| = 10, where the plain
   index is already fine. Execution is under 6 ms in every row.
3. **Run a short spike before the closure table.** Two renderings cut
   planning to under 0.5 ms:
   - `ANY((SELECT $S)::uuid[])`
   - `IN (SELECT unnest($S))`

   Both keep "the plan resolves the caller, the row test sees only the row"
   (no second table in the predicate). Both are unmeasured end to end:
   - they still bind 10,000 ids twice;
   - the initplan form hides the values from the planner, which can change
     the plan choice. For example, it may pick the bitmap path at
     |S| = 10,000, which would read about 375k rows.

   A closure-table join avoids binding S entirely, but it still pays for the
   expansion (about 41 ms p95 at about 10,000 units).
4. **Expansion H1 (≤ 5 ms p95) fails.** If S stays in the design, cache the
   subtree per (tenant, org-tree version) or store the closure.

## Setup

- **Code:** harness at `main` `c276a374`.
  - Seed: the unchanged P3-36 harness (`TestRecordsACLPerf800k`), run with `BENCH_KEEP=1 NAVA_PERF_ONLY='^agent list, sort=-updated_at$'`.
  - That run's own handler measurement was refused by its floor gate: `FLOOR … p50=8.18 ms > 3 ms gate; no numbers taken`. This report therefore has no same-day handler p95.
- **Data:** P3-36's shape, 800,000 records. The large tenant has 400,000 records, 5,000 users and 50 groups. No runtime code was changed.
- **Region data:** bench database only, `bench/acl-perf/region/region.sql`. It is not a migration.
  - The large tenant's tree is 1 root → 10 regions → 10 districts each → 10 areas each → 9 branches each, 10,111 units in all. Subtree sizes: area 10, district 101, region 1,011, root 10,111.
  - Each user's primary unit is a branch (leaf), chosen by md5 hash. 5% of users have no unit.
  - `records.org_unit_id` is the requester's primary unit (ADR-010 D8). 19,952 of the 400,000 records (5.0%) are NULL. The requester skew (r3²) makes units uneven: n_distinct is 3,254 over 9,000 leaves.
  - After the `UPDATE`, `VACUUM FULL records` rewrote the table, then `VACUUM (ANALYZE)` and `CHECKPOINT` ran, and every relation was `pg_prewarm`ed again before each index phase. The baseline was measured after this rewrite, on the same table.
- **S_k:** the first k units, in pre-order, of the subtree under one node.
  - |S| = 10: area 0/0/0, the whole subtree, 220 records.
  - |S| = 100: district 0/0, first 100 of its 101 units, 3,243 records.
  - |S| = 1,000: region 0, first 1,000 of its 1,011 units, 37,768 records.
  - |S| = 10,000: root, first 10,000 of its 10,111 units, 375,367 records (94%).
- **SQL:** the count and page statements are the handler's own, as captured in `docs/quality/acl-perf-800k/final-2026-11-09-full.md` ("agent list, sort=-updated_at").
  - The bench adds `pf_regional_view` (`all[is_regional_mgr, in_my_region]`, with `is_regional_mgr` folded to `true`) as a fifth `authz.ReadBranches` branch.
  - That branch is last, so it carries `AND NOT` of the four earlier branches, including `NOT EXISTS (record_shares …)`. That is the worst position.
  - The list runs as `nava_app` in the tenant transaction, with `QueryExecModeExec` and `plan_cache_mode = force_custom_plan`. The 200 P3-36 agents take turns.
- **Indexes:**
  - "plain" is `records (tenant_id, org_unit_id)`, D7's index.
  - "plain + R1" adds `(tenant_id, org_unit_id, updated_at DESC, id DESC)`.
- **Method:** P3-36's. 8 clients, 20 s per row, a warm-up of each request (one pass over the 200 agents), one run per row, nearest-rank percentiles (the same `pct` as `records_perf_integration_test.go:1271`). A second pass used 1 client, 10 s per row.
- **Machine:** the P3-36 reference host. Windows 11, i7-1260P, Docker 29.9.2 on WSL2 (kernel 5.15.167.4), VM with 16 CPUs and 24.5 GiB.
  - `postgres:16.10` with P3-36's settings: `shared_buffers=2GB`, `effective_cache_size=6GB`, `max_wal_size=8GB`, `track_io_timing=on`.
  - **Not idle:** another task's 12-container `nava-s18` stack, plus `buildx_buildkit_nava-ci0` and `multica-*`, were up during the whole run. That is outside P3-36's allow-list.
- **Date:** 2026-11-15, 16:40–17:40 UTC.

## Known gaps

- **The host was contended**, so no row is scored against 150 ms on its
  absolute value. The verdict uses the delta against a baseline measured in
  the same database and on the same host. At |S| = 10,000 that delta (+136 ms
  p50 and +158 ms p95 with one client) is far larger than the noise.
- **Database work only, not through the handler.** The handler does not have
  the leaf yet (S31), so these numbers leave out the handler's three small
  lookups and its Go time. Those are the same with or without the leaf.
- **One S per size, all the 200 agents' worst branch position.**

## Reproduce

```
sh bench/acl-perf/region/run.sh          # about 20 minutes: seed, region.sql, the 8-client pass
SP1_PG=<kept container> SP1_CLIENTS=1 SP1_SECONDS=10 sh bench/acl-perf/region/run.sh   # the 1-client pass
```

S31 rerun on the real schema (TICK-951): the same commands with `SP1_INDEX=real`, for example `SP1_INDEX=real sh bench/acl-perf/region/run.sh`, then `SP1_PG=<kept container> SP1_INDEX=real SP1_CLIENTS=1 SP1_SECONDS=10 sh bench/acl-perf/region/run.sh`. Run them from the story branch, so the seed carries S31's migrations.

It needs Docker and Go. It is not in CI. The raw reports, with every plan,
go to `bin/acl-perf-region-raw.md`. The container is kept. Remove it with
`docker rm -f <name>`.
