#!/bin/sh
# TICK-665: 8 workers race the FIRST upsert of the same new (tenant, type)
# keys (the key is the current 1 ms tick, so the workers collide on each new
# row). Counts unique_violation per index name, for (a) the 2-column pkey only
# (today), (b) pkey + scoped_key (the M2..M3 window) with the R0 and R1
# statements, (c) after M3. Usage: sh race-stress.sh <container> [iterations]
c=$1; N=${2:-20000}
q() { docker exec -i "$c" psql -U postgres -qAtX "$@"; }
run() { # $1 label, $2 conflict target, $3 column list, $4 values
  q -c "TRUNCATE rc;"
  for w in 1 2 3 4 5 6 7 8; do
    q -c "DO \$\$ DECLARE k bigint; c text; s text := ''; BEGIN
      FOR i IN 1..$N LOOP
        k := (extract(epoch FROM clock_timestamp()) * 1000)::bigint;
        BEGIN
          INSERT INTO rc ($3) VALUES ($4) ON CONFLICT ($2) DO UPDATE SET next = rc.next + 1;
        EXCEPTION WHEN unique_violation THEN
          GET STACKED DIAGNOSTICS c = CONSTRAINT_NAME; s := s || c || ' ';
        END;
        COMMIT;
      END LOOP;
      RAISE NOTICE 'E:%', s; END \$\$;" 2>&1 | sed -n 's/.*E://p' > "/tmp/rc-w$w" &
  done; wait
  errs=$(cat /tmp/rc-w? | tr ' ' '\n' | grep -v '^$' | sort | uniq -c | tr '\n' ' ')
  stats=$(q -c "SELECT count(*) || ' rows, ' || sum(next - 1) || ' numbers taken' FROM rc;")
  echo "$1: 8 x $N upserts; $stats; unique_violation: ${errs:-none}"
}
q -c "DROP TABLE IF EXISTS rc; CREATE TABLE rc (t int, d bigint, scope_key text NOT NULL DEFAULT '', next bigint NOT NULL, CONSTRAINT rc_pkey PRIMARY KEY (t, d));" 2>/dev/null
run "(a) pkey only, R0" "t, d" "t, d, next" "1, k, 2"
q -c "CREATE UNIQUE INDEX rc_scoped_key ON rc (t, d, scope_key);"
run "(b) pkey + scoped_key, R0" "t, d" "t, d, next" "1, k, 2"
run "(b) pkey + scoped_key, R1" "t, d, scope_key" "t, d, scope_key, next" "1, k, '', 2"
q -c "ALTER TABLE rc DROP CONSTRAINT rc_pkey; ALTER TABLE rc ADD CONSTRAINT rc_pkey PRIMARY KEY USING INDEX rc_scoped_key;"
run "(c) after M3, R1" "t, d, scope_key" "t, d, scope_key, next" "1, k, '', 2"
