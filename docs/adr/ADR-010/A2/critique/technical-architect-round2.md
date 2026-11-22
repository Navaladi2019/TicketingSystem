<!-- Archived verbatim from TICK-1040 comment 01a1fd17, 2026-11-21. -->

## Round-2 check of ADR-010-A2 rev 1 (`53577c4b`): round 1 is closed, but one new claim in rev 1 fails a spike

**Verdict: CHANGES on one item (R2-B1). Everything else in rev 1 is confirmed or non-blocking.** Every round-1 finding is closed. Rev 1 also adds one claim: "P3 and CC are O(K), so nothing replaces the cap". That claim is false when a type's records are sparse under a large S, and I missed this case in round 1. It is settled by a spike, not by debate, so the two-round cap is not at stake: the fix is a rule to add, not a position to argue.

**The crossover cells (A2-12 a) haven't run yet.** QA's harness fix has no PR yet, and `nava-aclperf-pg-575` is up on the host right now. A wakeup is set on TICK-946 for QA's next comment. The cells run once the fix is merged and the host is free, with one added cell (see R2-B1).

### Round-1 items in rev 1

| Item | Verdict | Evidence |
|---|---|---|
| B1 â†’ A2-3 (B1 index key) | **confirmed sound** | SP-1e: probe 220 â†’ 1.9 ms on the mixed district |
| B2 â†’ C3 | **confirmed sound**, and now checked at lower stats too | SP-1e held at `n_distinct` 1,000 to 9,000. My new run R2: at **`n_distinct` = 30**, C3 still plans a Bitmap Heap Scan inside the CTE and reads 900 rows in 0.73 ms. It cannot walk at any statistic I tried. |
| B3(a) â†’ CC always | **confirmed sound** for types that are dense under S | SP-1e 0.7 to 3.2 ms. For the sparse case, see R2-B1. |
| B3(b) â†’ K\* = cÂ·âˆš(pageÂ·NÌ‚_e) | **confirmed as the right model.** c is pending A2-12 (a). | Both sides read about K\* rows at the switch. This matches the m = 1,001 / 3,000 / 10,000 curve in SP-1e. |
| N1 â†’ the A2-1 build | **confirmed sound** | Same SQL as critique T8: 0 orphan rows |
| N2 â†’ the A2-2 lock | **confirmed sound** | `pg_advisory_xact_lock(int4, int4)` with two `hashtext` keys is valid and lasts for the transaction |
| A2-6, A2-8, A2-10, A2-11 | **confirmed sound** | No change to the predicate. The publish cap removes the request-time unprobed path. |

### New findings on rev 1

**R2-B1. blocking: P3, CC and C3 are O(|S|) when m â‰¤ K\*. This contradicts A2-4 ("nothing replaces the cap") and A2-7 ("no shape reads O(|S|)").**

- **Known**, from a spike on PG 16.10:
  - **Setup:** a 10,111-unit tree, the B1 key, U = root, and a type with 500 records in the whole tenant.
  - **Results:**
    - P3: **83.5 ms**;
    - CC (region branch alone): **72.2 ms**;
    - C3: **57.5 ms**.
  - Each one visits all 10,111 closure rows and makes one `records_org_unit` lookup per row (`loops=10111`). Most units have no record of the type, and none of the three can stop early while m â‰¤ K\*.
  - The same tree and U with the dense ticket type: P3 takes 4.2 ms and visits 114 closure rows. SP-1d's root number, +0.7 ms, is this dense case.
- **Inferred:**
  - Cost is about 20 Âµs per unit in S across the three statements, so about 210 ms at 10k units and about 2 s at 100k.
  - Rev 1 deletes `MaxUnderSet` and turns R8 into a metric, so nothing bounds |S|. Old D7 cost +200 ms at 10,000 units and refused anything bigger. A2 would no longer refuse.
  - It affects any type whose m â‰¤ K\* under a large S: a rare type, or one concentrated in a few units (HR cases, assets) under a region-wide policy.
- **Proposed rules** (your choice; both are small):
  1. **Rare-type shortcut:** if NÌ‚_e â‰¤ K\*, skip P3. Use W0 for the page and the `OFFSET 0` walk for the count. Both read â‰¤ NÌ‚_e â‰¤ K\* rows. This covers new and rare types.
  2. **A bound on |S| for the rest:** keep R8 as a groupsync **refusal** at a measured subtree size, not only a metric. Rev 0 already said "any refusal moves to a groupsync tree-size check". The size is set from the cell below.
- **Spike:** I'll add the cell "rare type and concentrated type, U = root and region, P3 + CC + C3 against rule 1" to the A2-12 (a) run. **Pass:** the region-only caller's Î” under the rule is within the worst A2-12 (a) cell at K\*.

**R2-N1. non-blocking: canonical U must also drop exact duplicates.**
- In A2-4, the `c.ancestor_id <> u` test keeps both copies of an id that appears twice in U, so `unnest(U)` counts its records twice.
- This happens if a policy lists an id twice, or if the caller has two `org_unit_members` rows for one unit.
- **Fix:** run `DISTINCT` on U before the ancestor test, add a duplicate id to the `u_not_canonical` mutant, and make A2-8 refuse duplicate ids at publish.

**R2-N2. non-blocking: K\* ignores the request's own filters.**
- A data-property filter with selectivity f has no ordered index, so W0 reads about pageÂ·N_e/(mÂ·f) rows. The effective crossover is âˆš(pageÂ·N_e/f).
- This is the same class as R-006 (PR #815, broad-ACL page walk). I suggest adding a residual row "A2-R5: selective unindexed filters move the crossover; see R-006", with no mechanism change. The worst persona stays the design point.

**R2-N3. nit: the CI plan test should cover a second sort key.**
- Every allowed sort (`created_at`, `updated_at`, and the fixed columns per `internal/server/records_list.go:147-158`) has an ordered list index (migrations `20261109140000` to `20261109140500`, and `20261110100000`). So W0's walk exists for each one.
- A2-5's plan test pins only `records_list_updated`. Add `sort=-created_at` so the fence is checked on a second index.

### What A2 needs for ACCEPTED, from my side
- **R2-B1:** add rule 1 and a measured |S| bound (rule 2) to the text, or the agreed equivalent. Correct the O(K) sentences in A2-4 and A2-7.
- **R2-N1:** fold in at acceptance, as you do with c.
- **A2-12 (a):** my cells, plus the R2-B1 cell. They are queued behind QA's fix.

<details><summary>Spike R1/R2 (PG 16.10, throwaway, superuser with no RLS, single run after warm-up; plan and rows read, not budget-grade)</summary>

**R1 setup:**
- Tree: root â†’ 10 â†’ 100 â†’ 10,000 leaves (10,111 units), with a live-roots closure.
- Records: 400k type-1 records over the leaves, and 500 type-3 records over the leaves.
- Indexes: `records_org_unit (tenant_id, org_unit_id, entity_definition_id) WHERE org_unit_id IS NOT NULL` and `records_list_updated`.

| Statement (U = root) | type 3 (500 records) | type 1 (400k) |
|---|---|---|
| P3, LIMIT 4,501 | 83.5 ms, 10,111 closure rows, 10,111 lookups | 4.2 ms, 114 closure rows |
| CC, LIMIT 1,001 | 72.2 ms, 10,111 lookups | not run |
| C3 page, LIMIT 51 | 57.5 ms, `ANY($0)` with 10,111 elements | not run |

**R2 setup:** a separate database with 30 units, 400,900 type-1 records, of which 900 sit in unit 0. `pg_stats.n_distinct(org_unit_id)` = 30. C3 on U = {unit 0}: `CTE acl_m â†’ Bitmap Heap Scan`, 900 rows, 0.73 ms.

```sql
-- P3 (R1a); CC is the same with LIMIT 1001
SELECT count(*) FROM (SELECT 1 FROM unnest(ARRAY[:root]::uuid[]) u(id)
  CROSS JOIN LATERAL (SELECT descendant_id FROM org_unit_closure WHERE tenant_id = :t AND ancestor_id = u.id) c
  CROSS JOIN LATERAL (SELECT 1 FROM records r WHERE r.tenant_id = :t AND r.entity_definition_id = 3
    AND r.org_unit_id IS NOT NULL AND r.org_unit_id = c.descendant_id LIMIT 4501) x LIMIT 4501) p;
-- C3 (R1d, R2)
WITH acl_m AS MATERIALIZED (SELECT r.id, r.updated_at FROM records r WHERE r.tenant_id = :t AND r.entity_definition_id = 3
  AND r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY(ARRAY(SELECT descendant_id FROM org_unit_closure
  WHERE tenant_id = :t AND ancestor_id = ANY(ARRAY[:root]::uuid[]))))
SELECT * FROM acl_m ORDER BY updated_at DESC, id DESC LIMIT 51;
```
**Rerun:** build the tree with `generate_series` (10 / 100 / 10,000 children per level), build the closure with A2-1's live-roots SQL, insert the records with `random()` timestamps, `VACUUM ANALYZE`, then run the statements above under `EXPLAIN (ANALYZE, BUFFERS)`. The containers have been removed; nothing was committed.
</details>