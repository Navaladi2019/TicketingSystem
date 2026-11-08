# TICK-538: ADR-004-E5 step (ii) contract, lock and time evidence

`bench.sql` seeds 50,000 groups (200 tenants x 250) and 1,000,000 records
(900,000 with an assignment group, 602 MB with indexes) on a throwaway
`postgres:16.10` container (Docker 29.9.2, Intel i7-1260P, 16 threads, Windows
11 host, `shared_buffers=1GB`), then runs the two migrations' SQL as written.
Output: `bench-out.txt`.

| Step | Locks held | Time |
| --- | --- | --- |
| (ii-a) `DROP` + `ADD ... ON DELETE RESTRICT NOT VALID` | ACCESS EXCLUSIVE on `records` and `groups`, to COMMIT | 13.9 ms |
| (ii-b) `VALIDATE CONSTRAINT` | SHARE UPDATE EXCLUSIVE on `records`, ROW SHARE on `groups` (reads and writes go on) | 4.88 s |
| Tombstone a group a record names | no RI trigger on `records` (the referenced key does not change) | 14.8 ms (cold) |
| Owner `DELETE` of an unreferenced group | RESTRICT check: `Trigger for constraint records_assignment_group_fkey` 99.2 ms | 151 ms |

The RESTRICT check scans `records`: `records_tenant_group_open` is partial
(`state <> 'closed'`), so it cannot serve the RI query. No index is added for
it. Only the owner can delete a group after step (ii), and group sync, the one
runtime deleter, tombstones instead (TICK-535). Under the old FK the same scan ran for
every group delete (`ON DELETE SET NULL`).
