#!/bin/sh
# The ci workflow's main-red job (TICK-289): a push to main whose run failed
# opens a P0 issue "main is red: <short sha>" with the failing jobs and the run
# URL. While one main-is-red issue is open, a later red push comments on it
# instead of opening another, so one incident stays one issue.
#
# Also the govulncheck workflow's main-red job (TICK-641), on a push or a
# scheduled run: WORKFLOW names the workflow in the body (default ci), EVENT
# the event (default push).
# Inputs (the job sets them): GH_TOKEN, GH_REPO (owner/repo), SHA, RUN_URL and
# NEEDS_JSON, the workflow's toJSON(needs). Needs gh and jq.
# Not a make target: TICK-289 forbids Makefile edits while S5 and S7 are open.
# By hand, against a scratch repo: GH_REPO=me/scratch SHA=$(git rev-parse HEAD)
#   RUN_URL=https://example.invalid NEEDS_JSON='{"lint":{"result":"failure"}}'
#   sh scripts/ci/main-red.sh
set -eu

: "${GH_REPO:?}" "${SHA:?}" "${RUN_URL:?}" "${NEEDS_JSON:?}"

short=$(printf '%s' "$SHA" | cut -c1-7)
title="main is red: $short"
failed=$(printf '%s' "$NEEDS_JSON" | jq -r '[to_entries[] | select(.value.result == "failure") | .key] | join(", ")')
[ -n "$failed" ] || failed="(none reported failure; see the run)"

case ${EVENT:-push} in
    schedule) what='a scheduled run on `main`' ;;
    *) what='a push to `main`' ;;
esac
body=$(printf '`%s` failed on %s at %s.\n\nFailing jobs: %s\n\nRun: %s\n' "${WORKFLOW:-ci}" "$what" "$SHA" "$failed" "$RUN_URL")

# The label may not exist yet in a fresh repository; an existing one is kept.
gh label create P0 --color B60205 --description "Production-blocking; fix today" >/dev/null 2>&1 || true

open=$(gh issue list --state open --label P0 --limit 100 --json number,title \
    --jq '[.[] | select(.title | startswith("main is red:"))][0].number // empty')

if [ -n "$open" ]; then
    gh issue comment "$open" --body "$body"
    echo "main-red: commented on #$open ($title)"
else
    gh issue create --title "$title" --label P0 --body "$body"
fi
