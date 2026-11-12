#!/bin/sh
# TICK-714 S1 load: every filter x {ACL (d), no ACL} list, 60 s, 8 clients;
# the capped count for the ACL form at 30 s; the P3-01 (d) control at the
# start and the end (the host is shared, so latency is read against it).
set -u
F="FILTERS=s1_filters.txt"
cd /bench/fi
echo "CONTROL-START $(sh /bench/q.sh load d list large explicit 1 60 8)"
for f in $(cut -d'|' -f1 s1_filters.txt); do
    env $F sh fq.sh load "$f" acl list large 60 8
    env $F sh fq.sh load "$f" none list large 60 8
    env $F sh fq.sh load "$f" acl count large 30 8
done
env $F sh fq.sh load eq_2 acl list small 60 8
echo "CONTROL-END $(sh /bench/q.sh load d list large explicit 1 60 8)"
echo LOAD-DONE
