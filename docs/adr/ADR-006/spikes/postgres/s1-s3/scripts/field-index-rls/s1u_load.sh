#!/bin/sh
# TICK-714 S1 mitigation: the (d) ACL compiled as a UNION of index-friendly
# branches (fq.sh u_* form), side table as a correlated EXISTS (D18 holds).
set -u
cd /bench/fi
for f in $(cut -d'|' -f1 s1u_filters.txt); do
    env FILTERS=s1u_filters.txt sh fq.sh explain "$f" acl list large > /tmp/ux.txt 2>&1
    echo "EXPLAIN $f $(grep -E 'Execution Time|ERROR' /tmp/ux.txt | head -1)"
    env FILTERS=s1u_filters.txt sh fq.sh load "$f" acl list large 60 8
done
for f in u_eq_40 u_eq_2 u_lt_40; do env FILTERS=s1u_filters.txt sh fq.sh load "$f" acl count large 30 8; done
echo "CONTROL $(sh /bench/q.sh load d list large explicit 1 60 8)"
echo U-DONE
