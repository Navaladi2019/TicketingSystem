#!/bin/sh
# make check-full-suite (TICK-638): the full-suite job of the ci workflow on a
# pull request into a TICK-* story branch. RESULTS lists job=result for every
# suite job (the workflow passes needs.<job>.result). Fails unless every one
# of them is success: on a story-branch pull request every job must run, so a
# skipped job is not green, and neither is a missing one.
#
# One exception (TICK-695): temporal-server may be skipped when the changes
# job decided the module did not change (TEMPORAL_SERVER=false, the
# needs.changes.outputs.temporal_server of make ci-temporal-server-changed).
# The module has its own go.mod; the story -> main PR still runs the job
# whenever the story touched it.
set -eu

want="changes lint unit temporal-server integration commit-check-budget stack coverage"
results=" ${RESULTS:-} "
bad=0
for job in $want; do
    case "$results" in
        *" $job=success "*) echo "full-suite: $job: success" ;;
        *" $job=skipped "*)
            if [ "$job" = temporal-server ] && [ "${TEMPORAL_SERVER:-}" = false ]; then
                echo "full-suite: $job: skipped, the module did not change (TICK-695)"
            else
                echo "full-suite: $job: skipped, want success" >&2
                bad=1
            fi
            ;;
        *" $job="*)
            r=${results#*" $job="}
            r=${r%% *}
            echo "full-suite: $job: $r, want success" >&2
            bad=1
            ;;
        *)
            echo "full-suite: $job: no result, want success" >&2
            bad=1
            ;;
    esac
done
if [ "$bad" = 1 ]; then
    echo "full-suite: a pull request into a story branch runs every job; a skipped job is not green (TICK-638; temporal-server only when the module changed, TICK-695)" >&2
    exit 1
fi
echo "full-suite: every job succeeded"
