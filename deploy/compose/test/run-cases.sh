#!/bin/sh
# The stack-test registry, derived from the files in cases/. Each case carries one header line:
#   # stack-case: job=<stack|stack-db|stack-internal|none>[,<job>] order=<n> mode=<serial|parallel>
# job: the CI job that runs it (none: by name only). order: run order, lower first; the
# default is 20; cases that stop or restart services sort after the read-only ones.
# mode: parallel cases only read the stack, so `PARALLEL=1` runs them at once.
#
#   run-cases.sh list <jobs>              case names in run order (<jobs>: a comma list, or all)
#   run-cases.sh registry                 the registry as YAML
#   run-cases.sh run <jobs>               run them; stops at the first failed serial case
#   run-cases.sh check                    every case has a valid header
# A bad header (missing, unknown key or job, non-numeric order, bad mode) makes every command
# exit 1 naming the file, never a silent skip. CASES_DIR overrides cases/ (the tests use it).
# CI calls `make stack-tests JOB=<job>`; a failure names the case.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
cases=${CASES_DIR:-$dir/cases}
die() { echo "run-cases: $*" >&2; exit 1; }

# TABLE: "order name mode jobs" rows, sorted. Built in this shell (not a pipeline's last
# stage) so a bad header stops the command.
load() {
    rows=$(
        for f in "$cases"/*.sh; do
            [ -f "$f" ] || die "no cases in $cases"
            n=$(basename "$f" .sh)
            h=$(sed -n 's/^# stack-case: //p' "$f" | head -n 1)
            [ -n "$h" ] || die "$f has no '# stack-case:' header"
            j=; o=20; m=serial
            for kv in $h; do
                case $kv in
                job=*) j=${kv#job=} ;;
                order=*) o=${kv#order=} ;;
                mode=*) m=${kv#mode=} ;;
                *) die "$f: unknown header field '$kv'" ;;
                esac
            done
            [ -n "$j" ] || die "$f: no job="
            for one in $(echo "$j" | tr ',' ' '); do
                case $one in stack|stack-db|stack-internal|none) ;; *) die "$f: unknown job '$one'" ;; esac
            done
            case $o in ''|*[!0-9]*) die "$f: order '$o' is not a number" ;; esac
            case $m in serial|parallel) ;; *) die "$f: mode '$m' is not serial or parallel" ;; esac
            printf '%s %s %s %s\n' "$o" "$n" "$m" "$j"
        done
    ) || exit 1
    [ -n "$rows" ] || die "no cases in $cases"
    TABLE=$(printf '%s\n' "$rows" | sort -k1,1n -k2,2)
}

# rows <jobs>: the table rows of the cases that run in <jobs> (a comma list, or all)
rows() {
    printf '%s\n' "$TABLE" | awk -v want="$1" '
        BEGIN { n = split(want, w, ",") }
        { m = split($4, j, ","); ok = 0
          for (a = 1; a <= m; a++) { if (want == "all" && j[a] != "none") ok = 1; for (b = 1; b <= n; b++) if (j[a] == w[b]) ok = 1 }
          if (ok) print }'
}

# names <jobs> [mode]
names() { rows "$1" | awk -v mode="${2:-}" 'mode == "" || $3 == mode { print $2 }'; }

run_case() { # name: its output in a log group, status returned
    printf '::group::stack case %s\n' "$1"
    st=0; sh "$cases/$1.sh" || st=$?
    printf '::endgroup::\n'
    [ "$st" -eq 0 ] || printf '::error title=stack case failed::%s (exit %s)\n' "$1" "$st"
    return "$st"
}

failed=
run_serial() { # names: in order, stops at the first failure
    for c in $1; do
        run_case "$c" || { failed="$failed $c"; return 1; }
    done
}

run_parallel() { # names: at once, each output printed whole when it ends
    tmp=$(mktemp -d)
    for pc in $1; do
        ( run_case "$pc" >"$tmp/$pc.log" 2>&1 ) & echo $! >"$tmp/$pc.pid"
    done
    st=0
    for pc in $1; do
        wait "$(cat "$tmp/$pc.pid")" || { st=1; failed="$failed $pc"; }
        cat "$tmp/$pc.log"
    done
    rm -rf "$tmp"
    return "$st"
}

run() {
    r=$(rows "$1")
    [ -n "$r" ] || die "no case runs in job '$1'"
    if [ "${PARALLEL:-}" = 1 ]; then
        # The serial cases that sort before the first parallel one (healthy) gate the parallel
        # group, which runs next; the rest of the serial cases follow it.
        minp=$(printf '%s\n' "$r" | awk '$3 == "parallel" { print $1; exit }')
        if [ -z "$minp" ]; then
            run_serial "$(names "$1")" || true
        else
            pre=$(printf '%s\n' "$r" | awk -v m="$minp" '$3 == "serial" && $1 + 0 < m + 0 { print $2 }')
            post=$(printf '%s\n' "$r" | awk -v m="$minp" '$3 == "serial" && $1 + 0 >= m + 0 { print $2 }')
            run_serial "$pre" && run_parallel "$(names "$1" parallel)" && run_serial "$post" || true
        fi
    else
        run_serial "$(names "$1")" || true
    fi
    if [ -n "$failed" ]; then printf 'FAILED stack cases:%s\n' "$failed" >&2; return 1; fi
    printf 'ok: stack cases passed (%s)\n' "$1"
}

load
case ${1:-} in
list) names "${2:?jobs}" "${3:-}" ;;
registry) printf '%s\n' "$TABLE" | awk '{ printf "- name: %s\n  make: %s\n  serial: %s\n  order: %s\n  ci_jobs: [%s]\n", $2, "stack-test-" $2, ($3 == "serial" ? "true" : "false"), $1, $4 }' ;;
run) run "${2:?jobs}" ;;
check) echo "ok: $(printf '%s\n' "$TABLE" | wc -l | tr -d ' ') cases" ;;
*) echo "usage: run-cases.sh list|registry|run|check" >&2; exit 1 ;;
esac
