<!-- Archived from TICK-529 comment 01a1a387-f770-7255-8b13-bc5c4bead09b (Technical Architect), 2026-11-03T22:29:19Z -->

## Critique — round 2 (RFC-011 rev 1): blocking items and the step (i) interaction

**VERDICT: APPROVE, on one condition.** PG-B1 and PG-B2 are resolved. The step (i) interaction check found **one verified defect (I1)**. It has a one-line mechanical fix and is not a design dispute, so it does not need a third round. The ADR must carry the fix below. If it does not, sync stalls whenever the auditors group is tombstoned. I spiked everything below on `postgres:16.10`. The code is at the end.

### Blocking items
| Item | Verdict | Evidence |
|---|---|---|
| **PG-B1: D10, expand then contract** | **resolved** | Step (ii) comes after step (i)'s binaries, (ii-b) `VALIDATE` is a separate migration, and `lock_timeout` is kept. Spike R1 also confirms that **the old binary still works against the step (i) schema**. Its hard `DELETE FROM groups` commits with D11 in place: the cascade removes the members first, so the new NO ACTION FK sees none, and `SET NULL` still clears the record, as it does today. So step (i) can deploy before the new sync. |
| **PG-B2: D4's reader list** | **resolved** | Every non-test reader of `groups` at `27860f4` is on the list: `apply.go:285`, `pgstore.go:39,52`, `onboard/store.go:196-214`. R8: the resolver join with the filter stays on index scans (78 buffers, 15 rows). One addition, not blocking: put **D5's policy subquery** (`g.deleted_at IS NOT NULL`) in the "sees all rows" group, so nobody "fixes" it with the D4 filter. |

### Interaction: D11 + D5 + D1's order inside step (i)
| # | Label | Finding | Evidence |
|---|---|---|---|
| I1 | **blocking, fold into the ADR** | **D5 as written does not work on `tenant_auditor_group`.** `nava_groupsync` has no grant and no SELECT policy on it (the branch's `db/grants.yaml` lists only `tenant_admin_group: { SELECT: [] }` for sync, and `tenant_auditor_group_select` is `TO nava_app, nava_worker`). A `DELETE … WHERE` needs SELECT on the columns it reads and the row has to pass a SELECT policy. | R2b, rev 1 as written: `ERROR: permission denied for table tenant_auditor_group`. The tenant's whole sync transaction rolls back, which is the TICK-452 stall again, triggered by tombstoning the auditors group. R2c, with the grant but no policy: **`DELETE 0`, silently.** The pointer survives, still pointing at the tombstoned group, so a revived auditors group regains audit access. That is exactly the outcome S2 rules out. R3, with the grant and a SELECT policy: `DELETE 1`. |
| I2 | **confirmed** | D1's order meets D11 and D5 in one transaction | R2: deleting the members, then tombstoning, then deleting the admin pointer gives `DELETE 1`, `UPDATE 1`, `DELETE 1`, and the commit succeeds. R4: the D5 policy refuses a live pointer (`DELETE 0`). R5: tombstoning before deleting the members fails with `23503 … group_members_group_live_fkey`. |
| I3 | **confirmed, with a plan note** | Revival order | R6: inserting a member before `deleted_at = NULL` fails with `23503`; reviving first, then inserting, succeeds. `apply.go` already sends group writes (batch 1) before memberships (batch 2), so the revive belongs in batch 1. The tombstone and the pointer delete belong in batch 2, **after** the membership deletes. |
| I4 | **non-blocking** | Old binary during the rollout | R7: an old sync instance that meets a tombstoned group which came back in Keycloak does not revive it. It inserts members, and D11 fails the transaction with `23503`. That fails closed and lasts only while old and new sync instances overlap. Inferred fix: deploy sync as stop-old, then start-new. |
| I5 | **non-blocking** | Step (i)'s locks | Measured with 50k groups and 2.5M memberships, in one transaction: adding the `live` column rewrites `groups` (201 ms, new relfilenode), `UNIQUE (tenant_id, id, live)` takes 39 ms, and adding `group_members.live` with its `CHECK` takes 160 ms. All of that runs under **ACCESS EXCLUSIVE on both tables**, which blocks the resolver for about 0.4 s. The FK added `NOT VALID` takes 1 ms; `VALIDATE` takes 166 ms under a non-blocking lock. Recommendation for the Postgres Specialist at plan time: add the `CHECK` as `NOT VALID` and validate it together with the FK, and keep `lock_timeout` on the rewrite migration. |

### Required text for I1 (D5 and D10 step (i))
> D10 step (i) also grants `nava_groupsync` `SELECT (tenant_id, group_id)` on `tenant_auditor_group`, and adds `nava_groupsync` to `tenant_auditor_group_select`'s `TO` list, wherever that table lands. Both the `DELETE … WHERE` and the "was the recorded group removed" probe need them. CI-E5-5b gains a check: tombstoning the recorded auditors group commits, and its pointer row is gone.

The same pair already exists for `tenant_admin_group` (`tag_select`, `SELECT: []`), which is why the admin case passes.

### Spike setup and how to rerun
- **Image and host:** `postgres:16.10` (Debian 16.10-1.pgdg13+1), Docker 29.9.2, run on 2026-11-04.
- **Data:** 10 tenants with 5,000 groups each, 50 members per group (2.5M rows), and one named tenant for the scenarios.
- **Schema:** `groups`, `group_members`, both pointer tables and `records` with their branch FKs, RLS policies and grants.
- **Rerun:** `docker run -d --name r2 -e POSTGRES_HOST_AUTH_METHOD=trust postgres:16.10`, then `psql -U postgres -f r2.sql`.
- **Archive with the rest:** `docs/adr/ADR-004/spikes/technical-architect/rfc-011/`.

(see files in this directory)

(see files in this directory)
