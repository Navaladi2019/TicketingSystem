#!/bin/sh
# TICK-714 S3: late flag of one text field ('priority', values "1".."5") on
# the 400k type. (1) backfill alone: duration and WAL bytes per batch;
# removal; (2) live write mix alone, 180 s (baseline p95); (3) the same mix
# with the backfill running from second 5; diff against a recomputation.
# The live mix: 50% inserts (S1 shape), 50% updates of a flagged key
# (score), 8 clients through PgBouncer, nava_app.
set -u
cd /bench/fi
export PGPASSWORD=bench
T=$(psql -X -At -U pgadmin -d nava -c "SELECT md5('tenant0')::uuid")
E=$(psql -X -At -U pgadmin -d nava -c "SELECT md5('ed0/0')::uuid")
LIVE=${LIVE:-180}

sh fq.sh write insert prep 1 1 >/dev/null; sh fq.sh write update prep 1 1 >/dev/null  # writes /tmp/write_*.sql
pct() {
    cat /tmp/l/log* | awk '{print $3}' | sort -n | awk '
        {v[NR] = $1}
        function at(p,  i) { i = int(NR * p); if (i < NR * p) i++; if (i < 1) i = 1; return v[i] / 1000 }
        END { printf "tx=%d p50_ms=%.2f p95_ms=%.2f p99_ms=%.2f max_ms=%.2f", NR, at(0.50), at(0.95), at(0.99), v[NR] / 1000 }'
}
live() { # live <seconds>
    rm -rf /tmp/l && mkdir -p /tmp/l
    pgbench -h pgb -p 5432 -U nava_app -n -M prepared -c 8 -j 8 -T "$1" -l --log-prefix=/tmp/l/log -D t="$T" \
        -f /tmp/write_insert.sql@1 -f /tmp/write_update.sql@1 nava > /tmp/l/out.txt 2>&1
    echo "LIVE tps=$(sed -n 's/^tps = \([0-9.]*\).*/\1/p' /tmp/l/out.txt | head -n1) $(pct)"
}

echo "### (1) backfill alone (pause 200 ms between batches)"
psql -X -q -U pgadmin -d nava -c "CHECKPOINT"
f=$(sh s3.sh flip "$T" "$E" priority text)
sh s3.sh backfill "$T" "$f" 200 > /tmp/bf1.txt 2>&1
grep -E "BARRIER|BACKFILL" /tmp/bf1.txt
grep BATCH /tmp/bf1.txt | awk '{split($3,a,"="); split($4,b,"="); n++; ms+=a[2]; w+=b[2]; if (a[2]>mx) mx=a[2]} END {printf "BATCHES n=%d avg_ms=%.1f max_ms=%d avg_wal_bytes=%.0f total_wal_mb=%.1f\n", n, ms/n, mx, w/n, w/1048576}'
sh s3.sh diff "$T" "$f"
t0=$(date +%s%3N); sh s3.sh remove "$T" "$f"; echo "REMOVE ms=$(( $(date +%s%3N) - t0 ))"
psql -X -q -U pgadmin -d nava -c "VACUUM rfv_text" -c "CHECKPOINT"

echo "### (2) live mix alone, ${LIVE} s"
live "$LIVE"
psql -X -q -U pgadmin -d nava -c "CHECKPOINT"

echo "### (3) live mix with the backfill"
live "$LIVE" > /tmp/live3.txt &
lp=$!
sleep 5
f=$(sh s3.sh flip "$T" "$E" priority text)
sh s3.sh backfill "$T" "$f" 200 > /tmp/bf3.txt 2>&1
grep -E "BARRIER|BACKFILL" /tmp/bf3.txt
grep BATCH /tmp/bf3.txt | awk '{split($3,a,"="); split($4,b,"="); n++; ms+=a[2]; w+=b[2]; if (a[2]>mx) mx=a[2]} END {printf "BATCHES n=%d avg_ms=%.1f max_ms=%d avg_wal_bytes_incl_live=%.0f\n", n, ms/n, mx, w/n}'
wait $lp
cat /tmp/live3.txt
sh s3.sh diff "$T" "$f"
echo S3-DONE
