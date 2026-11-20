# ADR-006-E1 spike: the newest running xid is not in `pg_snapshot_xip`

**Question.** Does `pg_snapshot_xip(pg_current_snapshot())` list every transaction in progress when the snapshot is taken?

**Answer: no.** It lists only in-progress xids below the snapshot's `xmax`, and `xmax` is one past the highest *completed* xid. A running transaction that holds the newest assigned xid sits at or above `xmax` and is not listed. The PostgreSQL manual says so in the snapshot components table (`xip`: "The list includes only those active IDs between xmin and xmax; there might be active IDs higher than xmax.").

**Run** (Solution Architect, 2026-11-19, `postgres:16.10`, `newest-xid.sh`):

```
PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu, ...
--- snapshot while writer runs (xmin:xmax:xip)
731:731:
xip-only waits on: (nothing)
--- E1 snapshot + fence
fence M 732
E1 waits on: 731
writer xid 731
```

The same shape (`1869:1869:` with 1869 in progress) was first seen by the Principal Software Engineer in TICK-745 integration testing (PR #738, comment 01a1f122-2a9e).

**Why the RFC-015 spikes passed.** `spikes/postgres/s1-s3/scripts/field-index-rls/s3.sh` and `spikes/technical-architect/r1-r3/` used xip only. Their held writer was never the newest xid: other writers kept completing later xids, so `xmax` moved past it and it was listed. The newest-xid case was not constructed.

**Timebox.** 15 minutes. Throwaway container; nothing else touched.
