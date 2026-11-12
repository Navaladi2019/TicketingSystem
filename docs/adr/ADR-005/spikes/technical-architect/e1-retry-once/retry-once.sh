#!/bin/sh
# mode savepoint = gapless (retry in an exception block = savepoint, same top tx)
# mode freshtx   = gaps    (COMMIT, then retry in a new transaction)
c=$1; N=${2:-5000}
q() { docker exec -i -e PGOPTIONS="$PGOPTS" "$c" psql -U postgres -qAtX "$@"; }
run() { # $1 label $2 mode
  PGOPTS= q -c "TRUNCATE rc;"
  for w in 1 2 3 4 5 6 7 8; do
    q -c "DO \$\$ DECLARE k bigint; c text; first int := 0; second text := ''; again bool; BEGIN
      FOR i IN 1..$N LOOP
        k := (extract(epoch FROM clock_timestamp()) * 1000)::bigint; again := false;
        BEGIN
          INSERT INTO rc (t,d,scope_key,next) VALUES (1,k,'',2) ON CONFLICT (t,d,scope_key) DO UPDATE SET next = rc.next + 1;
        EXCEPTION WHEN unique_violation THEN first := first + 1; again := true;
        END;
        IF again AND '$2' = 'freshtx' THEN COMMIT; END IF;
        IF again THEN
          BEGIN
            INSERT INTO rc (t,d,scope_key,next) VALUES (1,k,'',2) ON CONFLICT (t,d,scope_key) DO UPDATE SET next = rc.next + 1;
          EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS c = RETURNED_SQLSTATE; second := second || c || ' ';
          END;
        END IF;
        COMMIT;
      END LOOP;
      RAISE NOTICE 'R:% %', first, second; END \$\$;" 2>&1 | sed -n 's/.*R://p' > "/tmp/rr-w$w" &
  done; wait
  firsts=$(cat /tmp/rr-w? | awk '{s+=$1} END{print s}')
  seconds=$(cat /tmp/rr-w? | cut -d' ' -f2- | tr ' ' '\n' | grep -v '^$' | sort | uniq -c | tr '\n' ' ')
  stats=$(PGOPTS= q -c "SELECT count(*) || ' rows, ' || sum(next-1) || ' numbers' FROM rc;")
  echo "$1: 8 x $N; $stats; first-attempt 23505: $firsts; retry failures: ${seconds:-none}"
}
PGOPTS= q -c "DROP TABLE IF EXISTS rc; CREATE TABLE rc (t int, d bigint, scope_key text NOT NULL DEFAULT '', next bigint NOT NULL, CONSTRAINT rc_pkey PRIMARY KEY (t,d)); CREATE UNIQUE INDEX rc_scoped_key ON rc (t,d,scope_key);"
PGOPTS= ; run "RC, savepoint (gapless)" savepoint
PGOPTS= ; run "RC, fresh tx (gaps)" freshtx
PGOPTS='-c default_transaction_isolation=repeatable\ read'; run "RR, savepoint (gapless)" savepoint
PGOPTS='-c default_transaction_isolation=repeatable\ read'; run "RR, fresh tx (gaps)" freshtx
