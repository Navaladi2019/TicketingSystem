# TICK-1118 spike: does the delta closure write equal the full rebuild?

**Verdict: yes.** 0 of 3,803 runs on random trees differ from the full rebuild (3,563 of the runs were deltas). A mutant that leaves out one touched unit differs in 1,011 of 1,887 runs.

- **Code under test:** `internal/groupsync/closure.go` at `57573b39`, with only the package clause changed. `main.go.txt` calls its `rebuildClosure` and `fullRebuildClosure`.
- **Database:** `postgres:16.10` in a throwaway container, removed afterwards. Plain tables with the primary key and the composite foreign keys; no RLS, no grants.
- **Trees:** 40 seeds, 50 steps each, up to 70 units, 1 to 6 operations per step (add, move, tombstone, revive). *valid*: what a directory snapshot can give. *hostile*: also live units under tombstoned ones, and parent cycles.
- **Comparison:** `closureDriftSQL` (two-way `EXCEPT`) after every run, in the run's transaction.
- **Mutant:** one id is dropped from the touched list whenever it holds more than one.
- **Drift phase:** 300 seeds, 30-unit trees, one row added or removed, then a delta run, then a run with no touched unit. 25 seeds planted no drift or touched nothing and were skipped.
- **Output:** `output.txt`. The first run stopped at its 570 s limit before the drift phase (the host was running another stack), so that phase ran on its own. Its "tenant B unchanged" line covers the drift phase only. After the first run, `main.go.txt` gained only the `PHASE` switch and lost a timing suffix.
- **Not covered:** timing, concurrency, RLS.

To run it again: put `main.go.txt` (as `main.go`) and `closure.go` in one directory of this module, set `PGURL`, then `go run` it; `PHASE=drift` runs the last phase alone.
