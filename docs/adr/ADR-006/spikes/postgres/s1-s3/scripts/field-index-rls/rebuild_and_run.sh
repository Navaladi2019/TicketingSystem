#!/bin/sh
# TICK-714: rebuild the S1 bench after the host restart (same seed, same
# scripts; the S2 term table is left empty), then the R10 A/B and S3.
set -eu
cd "$(dirname "$0")/../.."
export MSYS_NO_PATHCONV=1
P=nava-rlb-pg-fi714b
BENCH_KEEP=1 BENCH_ID=fi714b BENCH_OUT=bin/fi714b-setup.txt sh bench/records-list/run.sh setup custom >/dev/null 2>&1
docker exec $P mkdir -p /bench/fi
docker cp scratch/field-index-rls/. $P:/bench/fi
for f in 20_s1_data.sql 21_s1_schema.sql 22b_s1_populate_noterms.sql 23_split.sql; do
    echo "### $f"; docker exec $P psql -X -v ON_ERROR_STOP=1 -U pgadmin -d nava -f /bench/fi/$f 2>&1 | grep -E "ERROR|large_tenant|^ +[0-9]+ \|" || true
done
docker exec $P psql -X -q -U pgadmin -d nava -c "ALTER STATISTICS rfv_bool_value_mcv SET STATISTICS 10000" -c "ANALYZE rfv_bool"
docker exec $P psql -X -At -U pgadmin -d nava -c "SELECT count(*), count(*) FILTER (WHERE tenant_id = md5('tenant0')::uuid) FROM records" -c "SELECT sum(pg_prewarm(c.oid)) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind IN ('r','i') AND (c.relname LIKE 'records%' OR c.relname LIKE 'rfv%')"
echo "### R10 / coerce A/B"; docker exec $P sh /bench/fi/s1_ab.sh 2>&1
echo "### cleanup of A/B inserts"; docker exec $P psql -X -q -U pgadmin -d nava -c "DELETE FROM records WHERE number = 0" -c "VACUUM (ANALYZE) records" -c "VACUUM rfv_text, rfv_num, rfv_ts, rfv_bool"
echo "### S3"; docker exec $P sh /bench/fi/s3_run.sh 2>&1
echo ALL-DONE
