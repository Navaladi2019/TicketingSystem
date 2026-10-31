#!/bin/sh
# Two transactions claim the same key at once. Session 1 holds its claim for 3 s,
# then COMMITs (run 1) or ROLLBACKs (run 2). Session 2 must block until session 1
# ends, then get 0 rows after a COMMIT, or START after a ROLLBACK.
T=aaaaaaaa-0000-0000-0000-000000000001
for END in COMMIT ROLLBACK; do
  K=$(cat /proc/sys/kernel/random/uuid)
  SQL="SET ROLE app; BEGIN; SELECT set_config('app.tenant_id','$T',true) IS NULL AS x;
INSERT INTO cascade_start VALUES ('$T','$K','00000000-0000-0000-0000-0000000000b0','00000000-0000-0000-0000-0000000000d1',gen_random_uuid()) ON CONFLICT DO NOTHING RETURNING 'START' AS outcome;"
  echo "=== concurrent: session 1 ends with $END"
  printf '%s\nSELECT pg_sleep(3) IS NULL AS x;\n%s;\n' "$SQL" "$END" | psql -U postgres -qAt | sed 's/^/s1: /' | grep -v '^s1: f$' &
  sleep 1
  S=$(date +%s)
  printf '%s\nCOMMIT;\n' "$SQL" | psql -U postgres -qAt | sed 's/^/s2: /' | grep -v '^s2: f$'
  echo "s2: returned after $(( $(date +%s) - S )) s (blocked on session 1's uncommitted key)"
  wait
done
