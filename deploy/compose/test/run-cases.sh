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
# CI calls `make stack-tests JOB=<job>`; a failure names the case.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)

# order name mode jobs, sorted
table() {
    for f in "$dir"/cases/*.sh; do
        n=$(basename "$f" .sh)
        h=$(sed -n 's/^# stack-case: //p' "$f" | head -n 1)
        [ -n "$h" ] || { echo "run-cases: $f has no '# stack-case:' header" >&2; return 1; }
        get() { printf '%s\n' "$h" | tr ' ' '\n' | sed -n "s/^$1=//p"; }
        j=$(get job); o=$(get order); m=$(get mode)
        [ -n "$j" ] || { echo "run-cases: $f: no job=" >&2; return 1; }
        case ${o:-20} in *[!0-9]*) echo "run-cases: $f: order is not a number" >&2; return 1 ;; esac
        case ${m:-serial} in serial|parallel) ;; *) echo "run-cases: $f: mode is serial or parallel" >&2; return 1 ;; esac
        printf '%s %s %s %s\n' "${o:-20}" "$n" "${m:-serial}" "$j"
    done | sort -k1,1n -k2,2
}

# names <jobs> [mode]
names() {
    table | awk -v want="$1" -v mode="${2:-}" '
        BEGIN { n = split(want, w, ",") }
        mode != "" && $3 != mode { next }
        { m = split($4, j, ","); ok = 0
          for (a = 1; a <= m; a++) { if (want == "all" && j[a] != "none") ok = 1; for (b = 1; b <= n; b++) if (j[a] == w[b]) ok = 1 }
          if (ok) print $2 }'
}

run_case() { # name: its output in a log group, status returned
    printf '::group::stack case %s\n' "$1"
    st=0; sh "$dir/cases/$1.sh" || st=$?
    printf '::endgroup::\n'
    [ "$st" -eq 0 ] || printf '::error title=stack case failed::%s (exit %s)\n' "$1" "$st"
    return "$st"
}

run() {
    failed=; pfailed=
    if [ "${PARALLEL:-}" = 1 ]; then
        # The serial cases that sort before the first parallel one (healthy) gate the rest.
        par=$(names "$1" parallel | tr '\n' ' ')
        done_par=
        for n in $(names "$1"); do
            case " $par " in
            *" $n "*) continue ;;
            esac
            if [ -z "$done_par" ] && [ -n "$par" ]; then
                # run the parallel group right after the first serial case (healthy)
                run_parallel "$par" || failed="$failed$pfailed"
                done_par=1
            fi
            run_case "$n" || { failed="$failed $n"; break; }
        done
        [ -z "$done_par" ] && [ -n "$par" ] && { run_parallel "$par" || failed="$failed$pfailed"; }
    else
        for n in $(names "$1"); do
            run_case "$n" || { failed="$failed $n"; break; }
        done
    fi
    if [ -n "$failed" ]; then printf 'FAILED stack cases:%s\n' "$failed" >&2; return 1; fi
    printf 'ok: stack cases passed (%s)\n' "$1"
}

run_parallel() { # names: at once, each output printed whole when it ends
    tmp=$(mktemp -d)
    for pc in $1; do
        ( run_case "$pc" >"$tmp/$pc.log" 2>&1 ) & echo $! >"$tmp/$pc.pid"
    done
    st=0
    for pc in $1; do
        wait "$(cat "$tmp/$pc.pid")" || { st=1; pfailed="$pfailed $pc"; }
        cat "$tmp/$pc.log"
    done
    rm -rf "$tmp"
    return "$st"
}

case ${1:-} in
list) names "${2:?jobs}" "${3:-}" ;;
registry) table | awk '{ printf "- name: %s\n  make: %s\n  serial: %s\n  order: %s\n  ci_jobs: [%s]\n", $2, "stack-test-" $2, ($3 == "serial" ? "true" : "false"), $1, $4 }' ;;
run) run "${2:?jobs}" ;;
check) table >/dev/null && echo "ok: $(table | wc -l | tr -d ' ') cases" ;;
*) echo "usage: run-cases.sh list|registry|run|check" >&2; exit 1 ;;
esac
