#!/bin/sh
# vulncheck-gate (ADR-001-E4 contract rule 4, TICK-1148): the result of the
# govulncheck workflow's vulncheck job, which runs with if: always().
#
# Inputs (env):
#   DETECT_RESULT  needs.vulncheck-detect.result
#   ROOT_NEEDED    needs.vulncheck-detect.outputs.root
#   TS_NEEDED      needs.vulncheck-detect.outputs.temporal_server
#   ROOT_RESULT    needs.vulncheck-root.result
#   TS_RESULT      needs.vulncheck-temporal-server.result
#
# Exits 0 only when detect succeeded and each module either ran and succeeded
# because detect said it was needed, or was skipped because detect said it was
# not. Anything else (failure, cancelled, a timeout, an empty or unknown value,
# a needed module skipped, an unneeded one run) exits 1 and says which input
# failed. A skipped job reports "Success" to a required check, so a skip is a
# pass only when detect asked for it.
set -u

me=vulncheck-gate
fail=0

if [ "${DETECT_RESULT:-}" = success ]; then
    echo "$me: DETECT_RESULT=success"
else
    echo "$me: DETECT_RESULT='${DETECT_RESULT:-}', want success" >&2
    fail=1
fi

# module NAME NEEDED_VAR NEEDED RESULT_VAR RESULT
module() {
    case "$3:$5" in
        true:success)
            echo "$me: $1 needed and succeeded"
            ;;
        false:skipped)
            echo "$me: $1 not needed and skipped"
            ;;
        true:*)
            echo "$me: $1 needed ($2=true) but $4='$5', want success" >&2
            fail=1
            ;;
        false:*)
            echo "$me: $1 not needed ($2=false) but $4='$5', want skipped" >&2
            fail=1
            ;;
        *)
            echo "$me: $2='$3', want true or false ($4='$5')" >&2
            fail=1
            ;;
    esac
}

module root ROOT_NEEDED "${ROOT_NEEDED:-}" ROOT_RESULT "${ROOT_RESULT:-}"
module temporal-server TS_NEEDED "${TS_NEEDED:-}" TS_RESULT "${TS_RESULT:-}"

[ "$fail" = 0 ] || echo "$me: failed" >&2
exit "$fail"
