<!-- SP-1e report (TICK-1043), archived verbatim from the Technical Architect's comment 01a1fceb-7a80 on 2026-11-21. Go harness files are renamed *.go.txt so they are not built. -->

## SP-1e verdict: fails (5) and a region-only caller above K. B1, P3, the capped count and C3 at â‰¤ K hold; W0 above K is bounded only by the caller's other branches

**Verdict (one line):** at â‰¤ K, **C3** is the pick: plan fixed by construction, reads â‰¤ K, no new index. The capped count and the B1 key fix B3(a) and B1. Above K, though, **W0's cost depends on what the caller's *other* branches return**. An agent with a desk branch pays +3.6 to +9.6 ms median. A caller whose access comes only through `under` (S31's own persona) pays **+31 ms at district and +85 ms at a time-uniform district of 1,001 tickets**. That is B3(b), now measured in the full request on the 800k bench. No shape tested is bounded above K at every size.

### Scoring basis
- **Paired wall-clock, scored.** P3-36's floor gate (8 clients Ã— 200 `SELECT 1`, p50 â‰¤ 3 ms) passed at every check: p50 1.13 to 2.25 ms, at the start and end of each phase.
- **Contention:** other tasks' `nava-it-*` clusters and the `nava` dev stack were up, so baseline wall-clock p95 was 15 to 27 ms. Î” cells within Â±2 to 3 ms of each other are not separable.
- **Diagnostic only:** server-side `EXPLAIN` is used for plans and rows read.
- **Threshold:** QA's rebased baseline is not posted yet. The deltas are raw: score them against 150 âˆ’ baseline once QA's number lands.

### Setup
- PostgreSQL 16.10 on `TICK-946/in-my-region` at `ddac06bb`: the 800k seed, the live-only closure, and SP-1c's skewed districts.
- **The B1 key:** `records_org_unit (tenant_id, org_unit_id, entity_definition_id) WHERE org_unit_id IS NOT NULL`.
- **Added data:**
  - **district mixed:** 900 tickets plus 150,000 records of a second entity type (`bench_other`), spread over its 90 branches;
  - **uniform:** three districts holding m = 1,001, 3,000 and 10,000 tickets, drawn at random from regions 4 to 8, so time-uniform.
- **Probe P3e:** P3 plus `entity_definition_id = $e`.
- **Count statement:** its region branch is always CC (`unnest(U)` â†’ closure â†’ `LATERAL (records â€¦ AND NOT b1â€¦b4 LIMIT 1001)`, capped at 1,001).
- **Page statement:** the only part that switches, on the probe. â‰¤ K = 1,000 â†’ C3 or L1; > K â†’ W0.
- **Callers:**
  - **"agents":** SP-1c's 200 agents, whose desk branch fills the page and the count;
  - **"count-only":** a caller with no groups, shares, requests or assignments, so branches 1 to 4 return 0 rows and the list is the region branch alone.
- **Equality:** MC3, ML1 and "L1 everywhere" return the same page ids, in order, and the same capped count as the reference (today's count + C1 page). This holds for **40 of 40** agents in all 10 sets, and for the count-only caller in all 10.

### (4) Mixed entity types: B1 holds
Diagnostic `EXPLAIN` on district mixed (900 tickets + 150,000 other):

| Index key | probe P3e | C2 page, region branch | C3 page, region branch |
| --- | --- | --- | --- |
| old `(tenant_id, org_unit_id)` | **220.3 ms, 150,894 index rows read** | 183.6 ms, 150,900 read | 190.4 ms, 150,900 read |
| **B1** | **1.88 ms, 909 read, index-only** | 4.9 ms, 900 read | 6.6 ms, 900 read |

**Pass criteria:**
- **â‰¤ K shape reads â‰¤ K: pass**, 900 records.
- **Probe â‰¤ 1 ms: not met** on this host (1.88 ms execution, diagnostic). It is 117Ã— the old key, and its wall-clock share sits inside the MC3 cells below.

### (7) The â‰¤ K shape: C3
All three shapes keep their plan family at `n_distinct` = sampled (3,080), 1,000 and 9,000. None flips to the `records_list_updated` walk on the B1 key. Rows read stay â‰¤ m everywhere:
- C2 and C3: 220, 600 and 900;
- L1: 150, 606 and 606.

C3 fixes its plan **by construction**: the `MATERIALIZED` CTE cannot be walked in `updated_at` order. C2's stability here does not undo B2, because T1b's flip was on another seed, and C2 has no structural guard.

Paired wall-clock, median / Î”p95 ms, agents, â‰¤ K sets:

| Set | MC3 (C3) | ML1 (L1) | L1, no probe |
| --- | --- | --- | --- |
| area (220) | **+3.95 / +2.73** | +5.40 / +7.94 | +2.49 / +2.99 |
| thin (600) | **+6.66 / +9.75** | +7.30 / +8.93 | +4.34 / +4.53 |
| mixed (900) | **+6.69 / +8.86** | +7.81 / +8.38 | +4.75 / +6.19 |

**L1 does not beat C3** once the probe is paid. It needs `records_ou_list` (62 MB, against 6 MB for the B1 key). Above K it is unbounded in |S|: region **+44 ms**, and root **1,141 ms** in `EXPLAIN` (182k rows read). "L1, no probe" is cheaper only because it skips the probe's round trip.

**L1's write cost is not resolved.** Its ABA latency drifted, so I can't attribute a number to the index: single-row INSERT p50 was 5.31 / 6.24 / 6.93 ms, and WAL per row was 9.4 / 8.7 / 9.5 KB, dominated by full-page writes. A second bench (`nava-aclperf-pg-5143`, not mine) started during this step. **Pick: C3.** No `records` write-path task is needed.

### (6) The count-only caller's count: bounded, not â‰¤ 1 ms
- **CC count (diagnostic):** it runs at every size and reads â‰¤ 1,010 records. Its region branch takes **0.7 to 3.2 ms**: area 0.69, district 3.15, region 3.08, root 2.38, uniform 2.25 to 2.60.
- **Today's W0 count for the same caller:** 642 ms at district (181,374 rows) and 1,050 ms at uniform m = 1,001 (375,712 rows). B3(a) is confirmed on the bench, and CC fixes it.
- **"â‰¤ 1 ms": not met** on this host. I can't separate it from contention; CPU-bound IO showed up as a 35 s CHECKPOINT during setup.

### Six subtrees and (5): paired wall-clock, median / Î”p95 ms
| Set (units, tickets) | MC3, agents | **MC3, count-only** | ML1, agents | ML1, count-only |
| --- | --- | --- | --- | --- |
| area (10, 220) | +3.95 / +2.73 | +6.09 / +7.56 | +5.40 / +7.94 | +7.12 / +9.15 |
| district (101, 3,379) | +5.67 / +7.72 | **+31.45 / +41.27** | +7.85 / +7.81 | +36.48 / +45.81 |
| thin (101, 600) | +6.66 / +9.75 | +9.12 / +11.72 | +7.30 / +8.93 | +10.71 / +13.00 |
| hot (101, 22,996) | +3.57 / +4.71 | +8.15 / +11.31 | +5.90 / +7.20 | +10.87 / +12.67 |
| region (1,011, 37,930) | +3.95 / +5.12 | +9.37 / +11.04 | +4.02 / +5.51 | +8.94 / +9.27 |
| root (10,111, 380,048) | +2.51 / +2.34 | +5.54 / +8.62 | +2.99 / +4.36 | +7.75 / +9.06 |
| mixed (101, 900) | +6.69 / +8.86 | +9.49 / +13.00 | +7.81 / +8.38 | +12.39 / +16.91 |
| **uniform m=1,001** | +6.12 / +7.48 | **+85.43 / +161.59** | +9.20 / +9.22 | +67.57 / +76.84 |
| uniform m=3,000 | +9.59 / +13.05 | **+47.66 / +63.29** | +6.76 / +7.13 | +34.22 / +35.84 |
| uniform m=10,000 | +7.06 / +11.82 | +20.45 / +26.85 | +5.42 / +6.05 | +14.51 / +14.82 |

**(5) Î” â‰¤ 3.8 ms: fails** at every uniform m, for both callers.

On the agents, `EXPLAIN` shows W0 reading only 1,016, 763 and 542 records at m = 1,001, 3,000 and 10,000. The desk branch fills the merged page, so the walk stops early. For the count-only caller nothing else fills the page. W0 then has to find 51 of the subtree's own rows: about page Ã— N_e / m, the critique's B3(b) formula. That gives **+85 ms at m = 1,001 and +31 ms at district**.

### Known
1. **B1 is right.** The old key costs 220 ms on a mixed-type district. The B1 key costs 1.9 ms and reads â‰¤ K.
2. **C3 is the â‰¤ K shape.** It reads â‰¤ K, its plan is fixed by construction, and it is stable at three `n_distinct` values. Its wall-clock cost is no worse than L1's, and it needs no new index.
3. **CC is bounded** at â‰¤ 1,010 records and 0.7 to 3.2 ms (diagnostic), against 0.6 to 1.1 s for today's count for the same caller.
4. **W0 above K is not bounded by the mechanism.** Its cost depends on the caller's other branches. For a region-only caller, it fails from +20 to +85 ms at 101-unit subtrees with 1,001 to 10,000 tickets, **on the bench scale A2 certifies**. B3(b) is therefore not only a residual risk for larger tenants: at 800k it already fails for the S31 persona.
5. **L1 is not a fix above K.** It is +44 ms at region and 1.1 s at root, because it is O(|S|) index descents.

### Inferred, not measured
- The shape above K needs a bound that does not depend on the other branches. The only data-independent bounds measured are C3/C2, which read â‰¤ m, and L1, which reads â‰¤ |S| Ã— page. Above K, m is unbounded, so neither holds at every size.
- One candidate is a second probe threshold K2 â‰ˆ page Ã— N_e / R, with a row budget R. W0 would run only when m > K2; that is â‰ˆ 20,000 at N_e = 400k, page 51, R = 1,000. Between K and K2, a capped C3 would read â‰¤ K2. That is a per-tenant-size switch, which the Solution Architect declined without data. **This is the data.**
- Another candidate is to accept region-only callers above K as over budget and take it to the CTO as (iii).

### How to rerun
On `TICK-946/in-my-region` (`ddac06bb`), put `probe-main.go`, `probe-sp1d.go` and `probe-sp1e.go` in `bench/acl-perf/region/probe/` (as `main.go`, `sp1d.go` and `sp1e.go`), and `probe.sh` in `bench/acl-perf/region/`.
```sh
BENCH_KEEP=1 NAVA_PERF_ONLY='^agent list, sort=-updated_at$' sh bench/acl-perf/run.sh   # seed (exit 1 if the handler pass is refused; container kept)
export SP1C_PG=<kept container> SP1C_OUT=$(pwd)/bin/sp1e-raw.md
sh bench/acl-perf/region/probe.sh setup     # SP-1c data
sh bench/acl-perf/region/probe.sh e-setup   # mixed + uniform districts, B1 key
sh bench/acl-perf/region/probe.sh e-check   # equality, 40 agents + count-only caller
sh bench/acl-perf/region/probe.sh e-plans   # (4), (7) n_distinct, (5)/(6) plans
SP1C_PAIRS=300 sh bench/acl-perf/region/probe.sh e-wall    # scored, gated; stops with "not scored" if the gate refuses
sh bench/acl-perf/region/probe.sh e-write   # L1 write cost (last: it adds rows)
```

<details><summary>CC count branch, C3 and L1 region branches, as measured</summary>

```sql
-- CC: the count's region branch ($8 = U after renumbering; AND NOT b1..b4 = today's earlier branches)
(SELECT 1 FROM unnest($8::uuid[]) acl_u(id)
  CROSS JOIN LATERAL (SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou WHERE acl_ou.tenant_id = $1 AND acl_ou.ancestor_id = acl_u.id) acl_c
  CROSS JOIN LATERAL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.org_unit_id IS NOT NULL
                      AND r.org_unit_id = acl_c.descendant_id AND NOT b1 AND NOT b2 AND NOT b3 AND NOT b4 LIMIT 1001) acl_x
  LIMIT 1001)
-- C3: the page's region branch at <= K
(WITH acl_m AS MATERIALIZED (SELECT <cols> FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND
   (r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY(ARRAY(SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou
     WHERE acl_ou.tenant_id = $1 AND acl_ou.ancestor_id = ANY($9::uuid[])))) AND NOT b1 AND NOT b2 AND NOT b3 AND NOT b4)
 SELECT * FROM acl_m ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $8)
-- L1 (index records_ou_list ON records (tenant_id, entity_definition_id, org_unit_id, updated_at DESC NULLS FIRST, id DESC) WHERE org_unit_id IS NOT NULL)
(SELECT acl_x.* FROM unnest($9::uuid[]) acl_u(id)
  CROSS JOIN LATERAL (SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou WHERE acl_ou.tenant_id = $1 AND acl_ou.ancestor_id = acl_u.id) acl_c
  CROSS JOIN LATERAL (SELECT <cols> FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.org_unit_id IS NOT NULL
                      AND r.org_unit_id = acl_c.descendant_id AND NOT b1 â€¦ AND NOT b4 ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8) acl_x
  ORDER BY acl_x.sort_key DESC NULLS FIRST, acl_x.id DESC LIMIT $8)
```
**`unnest(U)` needs a canonical U.** The `unnest(U)` renderings (P3e, CC, L1) return a record twice if one unit in U is an ancestor of another. CC is a count, so it would over-count. A2 must canonicalise U: drop any u that is a descendant of another u in U. That is at most 4 Ã— 4 closure-PK lookups, done at publish next to change 5's cap or per request. **Known** from SQL semantics; not exercised by the bench, where |U| = 1.
</details>

**Attached:**
- the harness: `probe-main.go`, `probe-sp1d.go`, `probe-sp1e.go`, `probe.sh`;
- `sp1e-raw.md`: every table, gate line and plan;
- `sp1e-seed.log`.

The bench container has been removed. Nothing has been committed. Every number here is **unverified** for `docs/budgets.md`.