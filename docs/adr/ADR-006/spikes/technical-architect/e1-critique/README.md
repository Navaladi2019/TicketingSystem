# ADR-006-E1: Technical Architect critique spikes (TICK-977)

Archived verbatim from the Technical Architect critique on TICK-977 (comment 01a1f14b-c007, 2026-11-19). PostgreSQL 16.10 (`postgres:16.10`), run in throwaway containers. Nothing was kept running.

The findings table is the critique as the TA posted it. The Solution Architect's dispositions are recorded in the rev 2 changelog of `docs/adr/ADR-006-E1-d9-barrier-waits-past-xmax.md`.

---

## Critique: ADR-006-E1 (PR #742)

**Overall:** the fence is right, and `[xmax, M)` is complete for every xid assigned before the fence. I found **no blocking finding**. There are 3 non-blocking findings. Two of them are text the erratum needs before ACCEPTED: E1-2 leaves out a precondition, and the AR leans on discipline and a detector that does not exist. The third is a contradiction about the E1-3 gate. Everything below was run on `postgres:16.10` in throwaway containers. Code is at the bottom.

### Findings

| # | Verdict | Mechanism | Evidence | Known / Inferred |
|---|---|---|---|---|
| 1 | **confirmed sound** | Q1: completeness of xip ∪ `[xmax, M)` | PG keeps every running xid below `xmax` in xip and skips any xid ≥ `xmax` (`procarray.c` GetSnapshotData: "If the XID is >= xmax, we can skip it"). Any xid assigned before the snapshot is < M, because M is assigned later under the same XidGenLock order. So every xid running at the snapshot is in xip or in `[xmax, M)`. | Known: source and SA spike. Inferred: the lock-ordering argument. |
| 2 | **confirmed sound** | Q1: subtransactions. `pg_current_snapshot` leaves subxids out of xip | Spike S4: a newest-xid writer inside `SAVEPOINT`, snapshot `735:735:`. E1 waits on `735,736` (top and sub). The parent always gets its xid before the child, so the top-level xid is always caught. | Known |
| 3 | **confirmed sound** | Commit window under synchronous replication: clog is set before the SyncRep wait, and the xact stays in the procarray | Spike S1, `synchronous_standby_names='ghost'`, writer 732 blocked in SyncRep: `pg_xact_status = in progress`, row visible to others `= false`. After release, visible. So `openSQL` cannot report "done" while the row is still invisible to the next backfill snapshot. | Known |
| 4 | **confirmed sound** | Q2: xid burn | Spike S5: snapshot `738:738:` → fence → `739:739:`. Exactly 1 xid per `takeSnapshot`. That runs once per Barrier activity, plus once more only on a retry with no heartbeat details. Flips are 3 per field lifecycle, so this adds no measurable wraparound pressure, and an aborted xid holds back no xmin. Rolling back an xid-holding transaction writes one abort WAL record. | Known: count. Inferred: the WAL record, from `RecordTransactionAbort`. |
| 5 | **confirmed sound** | Q2: read-only and replica | Spike S2: the fence works inside `BEGIN READ ONLY` (xid 733) and under `default_transaction_read_only=on` (734), so no side effect there. Spike S3, hot standby: `ERROR: cannot execute pg_current_xact_id() during recovery`. This **fails closed**: the activity errors and Temporal retries. xip-only would silently have used a standby snapshot (`735:735:`). That is a small improvement the erratum can claim. | Known |
| 6 | **confirmed sound** | Range cost of `fenceSQL` | The range is xids above the latest *completed* xid, so it scales with concurrent writers, not with the oldest open transaction. Each entry is one `pg_xact_status` lookup. Clog truncation gives `NULL`, which `= 'in progress'` counts as done. That is correct: a truncated xid finished long ago. | Inferred: size bound. Known: SQL semantics. |
| 7 | **non-blocking** | E1-2 leaves out a precondition: **the D4 sync trigger must be AFTER ROW** | Spike 2. A state-reading **BEFORE** ROW trigger at **READ COMMITTED**, on the first write of a transaction: the barrier wait set is `(empty)`, writer xid 740 arrives after the fence, and the row saw state `off`. The writer has no xid until `heap_insert`, which runs *after* a BEFORE trigger's registry read. Today's `records_field_values_sync` is `AFTER INSERT OR UPDATE OF data` (`db/migrations/20261115130000_field_index_trigger.sql:216`), so it holds. The purge trigger is BEFORE DELETE but reads no registry state, so it is unaffected. Control: AFTER + RC saw `on`. E1-2 currently credits "READ COMMITTED + volatile function" alone; it also needs "AFTER ROW". | Known |
| 8 | **non-blocking** | Q3: is AR-TICK-745-01 stated honestly? Partly. | (a) The RR hole is real. Spike 2: RR writer, AFTER trigger, wait set `(empty)`, xid 745, saw `off`. Severity "same-tenant mis-filter, no cross-tenant effect" holds: the trigger is SECURITY INVOKER under the writer's RLS. (b) **"A code path is caught in design review"** is discipline, not a mechanism. No records path uses RR/SERIALIZABLE today (`git grep` over non-test `.go`/`.sql` on the PR #738 head: 0 hits). So a mechanical guard costs nothing, see the proposal below. (c) **The revisit trigger names a detector that does not exist in production.** "Side-row recompute mismatch in telemetry" exists only as the CI-4/CI-5 test harness (`internal/fieldindex/backfill/harness_integration_test.go:235`, `sideDiff`). No runtime metric emits it. A silent miss would not be seen. (d) The AR also leaves out finding 7 (BEFORE trigger). | Known |
| 9 | **non-blocking** | E1-3 gate contradicts the plan | The md says "**E1 is not ACCEPTED until this test is merged**" (E1-3; also the html §E1-3). The trigger comment says E1-3 "stays in TICK-976". Either TICK-976 merges before the verdict, or E1-3's text changes to "ACCEPTED on the spike evidence; TICK-976 pins the regression". The verdict should not contradict the record. I lean towards the first, since the test is small, but scope and sequencing are the SA's call. | Known |

### Proposed changes (for the SA to commit on the PR #742 branch)

**E1-2, append:**
> The guarantee also needs the D4 sync trigger to stay **AFTER ROW**. Then the writer's xid exists before the trigger reads the registry. A BEFORE ROW trigger that reads registry state reopens the hole even at READ COMMITTED: a first write gets its xid at `heap_insert`, after the read (TA spike, TICK-977). No state-dependent BEFORE trigger may be added on `records`.

**AR-TICK-745-01, replace the mitigation and revisit columns with:**
> Mitigation: `record_field_values_sync` raises `field index sync requires READ COMMITTED` when `current_setting('transaction_isolation') <> 'read committed'`. A write at a higher level fails loudly instead of mis-indexing silently. A catalogue test asserts that the trigger is AFTER ROW (`pg_trigger.tgtype`) and that no other trigger on `records` reads `record_index_fields`. Residual: none from isolation once the guard ships. Until it ships, the AR stands as an operator rule.
> Revisit: a proposal to raise isolation on a records path, a change of trigger timing, or a guard error in logs. Drop "recompute mismatch in telemetry" unless a runtime recompute check is added.

The guard and the catalogue test are code. They are a follow-up for the Principal Software Engineer in the story branch, not a condition for E1. If the SA prefers to keep the AR as an operator rule only, then (b) and (c) must still be fixed in the AR text so it does not claim detection we lack.

### No spike outstanding
Every point above is settled by the runs below. Nothing here needs verify-by-spike.

<details><summary>Spike 1: sync-rep window, read-only, standby, subxact, xid cost (raw output + script)</summary>

```
PostgreSQL 16.10 (Debian 16.10-1.pgdg13+1) on x86_64-pc-linux-gnu
=== S1: sync-rep window
writer xid (waiting in SyncRep): 732
pg_xact_status        = in progress
row visible to others = false
xid in new xip        = false
pg_visible_in_snapshot= false
after release: visible=true visible_in_snapshot=true
=== S2: fence under read-only
fence in READ ONLY txn: 733
fence with default_transaction_read_only: 734
=== S3: fence on a hot standby
in recovery: true
standby snapshot: 735:735:
ERROR:  cannot execute pg_current_xact_id() during recovery
=== S4: subtransaction writer that is the newest xid
snapshot 735:735:
E1 waits on: 735,736
top 735
=== S5: xid cost of one fence
before 738:738:
after  739:739:
```

```bash
#!/usr/bin/env bash
set -uo pipefail
NAME=ta-e1-spike
docker run -d --rm --name $NAME -e POSTGRES_PASSWORD=x postgres:16.10 >/dev/null
trap 'docker rm -f $NAME >/dev/null 2>&1 || true' EXIT
for _ in $(seq 1 30); do docker exec $NAME pg_isready -U postgres -q && break; sleep 1; done; sleep 2
P="docker exec -i $NAME psql -U postgres -qAtX"
$P -c "create table r(id int)"
# S1 sync-rep window
$P -c "alter system set synchronous_standby_names='ghost'" -c "select pg_reload_conf()" >/dev/null; sleep 1
$P -c "begin; insert into r values (1); select 'writer xid '||pg_current_xact_id(); commit;" > w1.out 2>&1 &
sleep 2
X=$($P -c "select backend_xid from pg_stat_activity where wait_event='SyncRep'")
$P -c "select 'pg_xact_status = '||pg_xact_status('$X'::xid8)" \
   -c "select 'row visible to others = '||(count(*)>0) from r" \
   -c "select 'pg_visible_in_snapshot= '||pg_visible_in_snapshot('$X'::xid8, pg_current_snapshot())"
$P -c "alter system reset synchronous_standby_names" -c "select pg_reload_conf()" >/dev/null; sleep 1; wait
# S2 read-only
$P -c "begin read only; select pg_current_xact_id(); rollback;"
$P -c "set default_transaction_read_only=on; begin; select pg_current_xact_id(); rollback;"
# S3 standby
$P -c "select pg_create_physical_replication_slot('s')" >/dev/null
docker exec $NAME bash -c "pg_basebackup -U postgres -D /tmp/sb -R -S s -X stream && chown -R postgres /tmp/sb && chmod 700 /tmp/sb && su postgres -c 'pg_ctl -D /tmp/sb -o \"-p 5433\" -l /tmp/sb.log start'"
sleep 3
docker exec -i $NAME psql -U postgres -p 5433 -qAtX -c "select pg_current_snapshot()" -c "begin" -c "select pg_current_xact_id()"
# S4 subxact newest-xid writer: hold 'begin; savepoint a; insert ...; pg_sleep(6); commit' in background,
#    then run the E1 snapshot+fence query from e1-newest-xid/newest-xid.sh
# S5 xid cost: pg_current_snapshot(); begin; pg_current_xact_id(); rollback; pg_current_snapshot()
```
</details>

<details><summary>Spike 2: which writers escape xip ∪ [xmax, M) (raw output + script)</summary>

```
=== BEFORE ROW trigger, READ COMMITTED, first write of the txn
barrier wait set: (empty)
writer xid 740
writer row saw state: off
=== AFTER ROW trigger, REPEATABLE READ, snapshot taken before the flip
barrier wait set: (empty)
writer xid 745
writer row saw state: off
=== AFTER ROW trigger, READ COMMITTED (control)
barrier wait set: (empty)
writer xid 749
writer row saw state: on
```

```sql
-- same container bootstrap as spike 1, then:
create table reg(state text); insert into reg values ('off');
create table rec(id int, saw text);
create function after_fn() returns trigger language plpgsql as $$
begin update rec set saw = (select state from reg) where id = new.id; return null; end $$;
create function before_fn() returns trigger language plpgsql as $$
begin new.saw := (select state from reg); perform pg_sleep(4); return new; end $$;
-- per case: reset reg='off'; start the writer in background; sleep 1.5;
--   update reg set state='on' (the flip commits); run the E1 snapshot+fence query (the barrier);
--   wait; select saw from rec.
-- writers:
--   BEFORE/RC : begin; insert into rec values (1); commit;                     (trigger before_fn)
--   AFTER/RR  : begin isolation level repeatable read; select 1; select pg_sleep(3); insert ...; commit;
--   AFTER/RC  : begin; select pg_sleep(3); insert ...; commit;                 (control)
```
</details>

Setup: Docker 29.9.2, `postgres:16.10` (same image as the SA spike), single node plus a `pg_basebackup` hot standby for S3. Timebox: about 20 minutes. Nothing was kept running.
