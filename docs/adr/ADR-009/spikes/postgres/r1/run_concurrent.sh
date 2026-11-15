#!/bin/sh
# usage: run_concurrent.sh <batch> <lockmode|none> <label>
B=$1; L=$2; LBL=$3
docker exec -i rfc021-pg psql -U postgres -q -v ON_ERROR_STOP=1 < 07_reset_a.sql 2>/dev/null
printf "set max_parallel_maintenance_workers=0;\nvacuum analyze records;\ncheckpoint;\n" | docker exec -i rfc021-pg psql -U postgres -q
docker exec rfc021-pg sh -c "rm -rf /tmp/pb && mkdir -p /tmp/pb"
docker cp w_update.sql rfc021-pg:/tmp/pb/ >/dev/null; docker cp w_share.sql rfc021-pg:/tmp/pb/ >/dev/null
sleep 20
docker exec -e PGPASSWORD=x rfc021-pg sh -c "cd /tmp/pb && pgbench -h localhost -U nava_app -n -c 4 -j 2 -R 100 -T 20 -l --log-prefix=upd -f w_update.sql postgres >/tmp/pb/upd.out 2>&1 & cd /tmp/pb && pgbench -h localhost -U nava_app -n -c 2 -j 1 -R 25 -T 20 -l --log-prefix=shr -f w_share.sql postgres >/tmp/pb/shr.out 2>&1 &"
if [ "$L" != none ]; then printf "set role nava_worker;\ncall backfill($B, '$L');\n" | docker exec -i rfc021-pg psql -U postgres -q 2>&1 | grep -E "NOTICE|ERROR"; fi
sleep 22
for k in upd shr; do
docker exec rfc021-pg sh -c "cat /tmp/pb/$k.[0-9]* 2>/dev/null | awk '{print \$3/1000}' | sort -n | awk -v k=$k -v l=$LBL '{a[NR]=\$1} END {printf \"%s %s n=%d p50=%.2fms p95=%.2fms p99=%.2fms max=%.1fms\n\", l, k, NR, a[int(NR*0.5)], a[int(NR*0.95)], a[int(NR*0.99)], a[NR]}'"
done
