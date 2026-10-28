#!/bin/sh
# The ci workflow's story-branches job (TICK-322), on main only: while
# `make check-story-branches` fails, one open issue "Story branches out of step
# with main" carries its report (opened once, then commented on each red run
# whose report changed); the first green run comments and closes it.
#
# Inputs (the job sets them): GH_TOKEN, GH_REPO (owner/repo), RESULT (success or
# failure, the check step's outcome), RUN_URL. Reads story-branches.md, which
# the check writes. Needs gh and jq.
# By hand, against a scratch repo: GH_REPO=me/scratch RESULT=failure
#   RUN_URL=https://example.invalid make story-branches-issue
set -eu

: "${GH_REPO:?}" "${RESULT:?}" "${RUN_URL:?}"

title="Story branches out of step with main"
label="story-drift"

gh label create "$label" --color D93F0B --description "A story branch's work is not on main (make check-story-branches)" >/dev/null 2>&1 || true

open=$(gh issue list --state open --label "$label" --limit 100 --json number,title \
    --jq "[.[] | select(.title == \"$title\")][0].number // empty")

case "$RESULT" in
failure)
    [ -s story-branches.md ] || { echo "story-branches-issue: no story-branches.md" >&2; exit 1; }
    body=$(printf '%s\n\nRun: %s\n' "$(cat story-branches.md)" "$RUN_URL")
    if [ -z "$open" ]; then
        gh issue create --title "$title" --label "$label" --body "$body"
        exit 0
    fi
    # Every red run would repeat the same report every 6 hours; comment only
    # when the findings differ from the last report posted.
    last=$(gh issue view "$open" --json body,comments \
        --jq '([.body] + [.comments[].body]) | last' | sed '/^Run: /d')
    if [ "$last" = "$(cat story-branches.md)" ]; then
        echo "story-branches-issue: #$open already carries this report"
    else
        gh issue comment "$open" --body "$body"
        echo "story-branches-issue: commented on #$open"
    fi
    ;;
success)
    if [ -n "$open" ]; then
        gh issue close "$open" --comment "Every merged story branch is on main again. Run: $RUN_URL"
        echo "story-branches-issue: closed #$open"
    fi
    ;;
*)
    echo "story-branches-issue: RESULT must be success or failure, got $RESULT" >&2
    exit 2
    ;;
esac
