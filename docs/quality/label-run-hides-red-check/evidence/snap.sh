#!/bin/sh
# snap.sh <case> <min runs on head> : waits until no run on the head is active, then stores raw API output
R=Navaladi2019/TicketingSystem; N=$(sed 's#.*/##' ../spike/pr.txt | tr -d '\r\n'); D=../spike/evidence; mkdir -p $D
H=$(gh pr view $N -R $R --json headRefOid -q .headRefOid)
sleep 12
for i in $(seq 1 70); do
 a=$(gh api "repos/$R/actions/runs?head_sha=$H&per_page=100" -q '"\([.workflow_runs[]|select(.status!="completed")]|length) \(.workflow_runs|length)"'); set -- "$1" "$2" $a; [ "$3" = 0 ] && [ "$4" -ge "$2" ] && break; set -- "$1" "$2"; sleep 8; done
[ "$i" = 70 ] && echo "TIMEOUT: runs not finished" 
sleep 45
for i in $(seq 1 12); do
 gh api graphql -f query='query($n:Int!){repository(owner:"Navaladi2019",name:"TicketingSystem"){pullRequest(number:$n){number isDraft state mergeable mergeStateStatus headRefOid baseRefOid labels(first:20){nodes{name}} baseRef{target{oid}}}}}' -F n=$N > $D/$1-pr.json
 grep -q '"mergeStateStatus":"UNKNOWN"' $D/$1-pr.json || break; sleep 5; done
sleep 10; gh api -i rate_limit | grep -i '^date:' > $D/$1-time.txt
gh api "repos/$R/pulls/$N" -q '{mergeable,mergeable_state,draft}' > $D/$1-rest.json
gh api --paginate "repos/$R/commits/$H/check-runs?filter=all&per_page=100" > $D/$1-checkruns-all.json
gh api "repos/$R/actions/runs?head_sha=$H&per_page=100" -q '[.workflow_runs[]|{id,event,name,status,conclusion,created_at,run_number}]' > $D/$1-runs.json
gh pr checks $N -R $R --required > $D/$1-pr-checks-required.txt 2>&1
echo "== case $1"; cat $D/$1-time.txt $D/$1-pr.json $D/$1-rest.json $D/$1-runs.json; echo
gh api --paginate "repos/$R/commits/$H/check-runs?filter=all&per_page=100" -q '.check_runs|sort_by(.started_at,.id)[]|select(.name|IN("changes","lint","unit","temporal-server","integration","stack","coverage","commit-check-budget","isolation"))|"\(.started_at) \(.check_suite.id) \(.name) \(.conclusion)"' | grep -E "isolation"
echo "-- required"; cat $D/$1-pr-checks-required.txt
T=$(gh api repos/$R/commits/main -q .sha); gh api repos/$R/compare/main...$H -q '"behind_by=\(.behind_by)"' | tee $D/$1-behind.txt; grep -q "$T" $D/$1-pr.json && echo base-ok || echo BASE-MOVED
