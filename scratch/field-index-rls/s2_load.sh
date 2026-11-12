#!/bin/sh
# TICK-714 S2: term table (D12) at 400k, 3 searchable fields. Each term is a
# correlated EXISTS with field_id = ANY(readable searchable ids).
set -u
cd /bench/fi
psql -X -q -At -U pgadmin -d nava -c "SELECT pg_prewarm('record_search_terms_pkey', 'read'), pg_prewarm('record_search_terms', 'read')" >/dev/null
for f in $(cut -d'|' -f1 s2_filters.txt); do
    for a in acl none; do
        env FILTERS=s2_filters.txt sh fq.sh explain "$f" $a list large > /tmp/sx.txt 2>&1
        cat /tmp/sx.txt >> /bench/fi/s2_explain.txt
        echo "EXPLAIN $f $a $(grep -E 'Execution Time|ERROR' /tmp/sx.txt | head -1)"
        env FILTERS=s2_filters.txt sh fq.sh load "$f" $a list large 30 8
    done
done
echo S2-DONE
