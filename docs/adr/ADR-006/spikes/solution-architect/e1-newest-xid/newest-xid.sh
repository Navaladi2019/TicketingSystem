#!/usr/bin/env bash
# ADR-006-E1 spike: pg_snapshot_xip misses a running transaction that holds
# the newest assigned xid; a fence xid taken after the snapshot catches it.
# Needs Docker. Uses a throwaway container; removes it at the end.
set -euo pipefail
NAME=adr006-e1-spike
docker run -d --rm --name "$NAME" -e POSTGRES_PASSWORD=x postgres:16.10 >/dev/null
trap 'docker rm -f "$NAME" >/dev/null 2>&1 || true' EXIT
for _ in $(seq 1 30); do docker exec "$NAME" pg_isready -U postgres -q && break; sleep 1; done
P="docker exec -i $NAME psql -U postgres -qAtX"

$P -c "select version()"

# Writer: takes the newest xid and holds it open for 8 s.
$P -c "begin; select 'writer xid '||pg_current_xact_id(); select pg_sleep(8); commit;" > writer.out &
sleep 2

echo "--- snapshot while the writer runs (xmin:xmax:xip)"
$P -c "select pg_current_snapshot()"
echo "--- D9 as written (xip only)"
$P -c "select 'xip-only waits on: '||coalesce(string_agg(x::text,','),'(nothing)') from pg_snapshot_xip(pg_current_snapshot()) x"
echo "--- E1 (snapshot, then fence M, same transaction)"
$P <<'SQL'
begin;
select pg_snapshot_xmax(pg_current_snapshot()) as xmax \gset
select 'fence M '||pg_current_xact_id();
select 'E1 waits on: '||coalesce(string_agg(x::text,','),'(nothing)') from (
  select x::text::xid8 x from pg_snapshot_xip(pg_current_snapshot()) x
  union all
  select n::text::xid8 from generate_series(:'xmax'::text::bigint, pg_current_xact_id()::text::bigint - 1) n
   where pg_xact_status(n::text::xid8) = 'in progress') s;
rollback;
SQL
wait
cat writer.out
