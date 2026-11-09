#!/bin/sh
# TICK-640: pgbench of insert.sql (8 clients, $SECS s, default 60) in the
# tick640-pg container; prints tps, p50/p95/p99 latency (ms) and WAL bytes
# per insert. Label: $1.
set -eu
MSYS_NO_PATHCONV=1; export MSYS_NO_PATHCONV
cd "$(dirname "$0")/../.."
SECS=${SECS:-60}
MSYS_NO_PATHCONV=1 docker cp scratch/tick-640/insert.sql tick640-pg:/tmp/insert.sql
MSYS_NO_PATHCONV=1 docker exec tick640-pg sh -c 'rm -rf /tmp/pgb && mkdir /tmp/pgb'
w0=$(docker exec tick640-pg psql -X -At -U pgadmin -d nava -c "SELECT pg_current_wal_lsn()")
MSYS_NO_PATHCONV=1 docker exec -e PGOPTIONS='-c session_replication_role=replica' -w /tmp/pgb tick640-pg \
    pgbench -n -U pgadmin -c 8 -j 8 -T "$SECS" -f /tmp/insert.sql -l nava > bin/pgbench-$1.txt 2>&1
w1=$(docker exec tick640-pg psql -X -At -U pgadmin -d nava -c "SELECT pg_current_wal_lsn()")
n=$(grep -o 'number of transactions actually processed: [0-9]*' bin/pgbench-$1.txt | grep -o '[0-9]*$')
tps=$(grep -o 'tps = [0-9.]*' bin/pgbench-$1.txt | head -1)
lat=$(MSYS_NO_PATHCONV=1 docker exec tick640-pg sh -c 'cat /tmp/pgb/pgbench_log.* | awk "{print \$3}" | sort -n | awk "{a[NR]=\$1} END {printf \"p50=%.2f p95=%.2f p99=%.2f\", a[int(NR*0.5)]/1000, a[int(NR*0.95)]/1000, a[int(NR*0.99)]/1000}"')
wal=$(docker exec tick640-pg psql -X -At -U pgadmin -d nava -c "SELECT round(pg_wal_lsn_diff('$w1', '$w0') / $n)")
echo "$1: n=$n $tps $lat ms, WAL/insert=${wal} B"
