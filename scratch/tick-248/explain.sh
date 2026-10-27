#!/bin/sh
# TICK-248: EXPLAIN (ANALYZE, BUFFERS) of claimSQL before and after the
# stranded-head step, three runs each, on the data seed.sql writes. Each run
# is rolled back, so every run sees the same state.
#   sh scratch/tick-248/explain.sh <psql connection args...>
# PSQL overrides the client, e.g. PSQL="docker exec -i <container> psql".
# BASE is the revision with the old SQL (default: the fix's parent commit);
# the new SQL is the working tree's. Full plans land in OUT (default: a new
# temporary directory).
set -eu
cd "$(dirname "$0")/../.."
base=${BASE:-b72094f}
out=${OUT:-$(mktemp -d)}
mkdir -p "$out"
echo "plans in $out"
extract() { # Go source on stdin -> claimSQL with ScanCap 200, PerTenant 2, 30 s lease, 24 h age
    sed -n '/^const claimSQL = `/,/`$/p' | sed '1s/^const claimSQL = `//; $s/`$//' |
        sed 's/\$1/200/g; s/\$2/2/g; s/\$3/30/g; s/\$4/86400/g; s/\$5/'"'"'expired'"'"'/g'
}
git show "$base:internal/relay/relay.go" | extract >"$out/claim_old.sql"
extract <internal/relay/relay.go >"$out/claim_new.sql"
for v in old new; do
    for run in 1 2 3; do
        echo "=== $v run $run"
        { echo 'BEGIN;'; echo 'EXPLAIN (ANALYZE, BUFFERS)'; cat "$out/claim_$v.sql"; echo ';'
          # Tenant a's rows this claim leased: X's head at attempt 2 after the fix.
          echo "SELECT tenant_id, target_key, target_seq, attempts FROM outbox
                WHERE tenant_id = '00000000-0000-0000-0000-00000000000a'
                  AND available_at > now() + interval '20 seconds' AND attempts >= 1 ORDER BY 1, 2, 3;"
          echo 'ROLLBACK;'; } | ${PSQL:-psql} -X -q "$@" >"$out/$v-$run.txt"
        grep -m 1 'Buffers: shared' "$out/$v-$run.txt"
        grep -E 'Execution Time|^ *00000000' "$out/$v-$run.txt" || true
        sed -n '/CTE picked/{n;p}' "$out/$v-$run.txt" | sed 's/^ */picked: /'
        sed -n '/CTE stranded/{n;p}' "$out/$v-$run.txt" | sed 's/^ */stranded: /'
    done
done
