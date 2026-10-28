#!/bin/sh
# The ci workflow's story-branches job (TICK-322), on main only: while
# `make check-story-branches` fails, one open issue "Story branches out of step
# with main" carries its report (opened once, then commented on each red run
# whose report changed); the first green run comments and closes it.
#
# The job serializes this step on main (job-level concurrency), and the script
# heals what a race left anyway: a red run keeps the oldest open issue and
# closes any other as its duplicate, a green run closes them all.
#
# Inputs (the job sets them): GH_TOKEN, GH_REPO (owner/repo), RESULT (success or
# failure, the check step's outcome), RUN_URL. Reads story-branches.md, which
# the check writes. Needs gh.
# By hand, against a scratch repo: GH_REPO=me/scratch RESULT=failure
#   RUN_URL=https://example.invalid make story-branches-issue
set -eu

: "${GH_REPO:?}" "${RESULT:?}" "${RUN_URL:?}"

title="Story branches out of step with main"
label="story-drift"
# The workflow's GITHUB_TOKEN writes as github-actions; gh prints app/github-actions.
bot="github-actions"

gh label create "$label" --color D93F0B --description "A story branch's work is not on main (make check-story-branches)" >/dev/null 2>&1 || true

# Every open issue with the label and title, oldest first.
open=$(gh issue list --state open --label "$label" --limit 100 --json number,title \
    --jq "[.[] | select(.title == \"$title\") | .number] | sort | .[]")

case "$RESULT" in
failure)
    if [ ! -s story-branches.md ]; then
        # Exit 2: the tool itself failed before writing a report. The check
        # step's log has the error; the job is already red.
        echo "story-branches-issue: no story-branches.md, so the check errored rather than found drift; see the step above" >&2
        exit 0
    fi
    body=$(printf '%s\n\nRun: %s\n' "$(cat story-branches.md)" "$RUN_URL")
    keep=""
    for n in $open; do
        if [ -z "$keep" ]; then
            keep=$n
        else
            gh issue close "$n" --reason "not planned" --comment "Duplicate of #$keep (two runs raced). Run: $RUN_URL"
            echo "story-branches-issue: closed duplicate #$n"
        fi
    done
    if [ -z "$keep" ]; then
        gh issue create --title "$title" --label "$label" --body "$body"
        exit 0
    fi
    # Every red run would repeat the same report every 6 hours; comment only
    # when the findings differ from the last report this workflow posted.
    # People's comments on the issue do not count.
    last=$(gh issue view "$keep" --json body,author,comments --jq "
        ([select((.author.login | ltrimstr(\"app/\")) == \"$bot\") | .body] +
         [.comments[] | select((.author.login | ltrimstr(\"app/\")) == \"$bot\") | .body]) | last // \"\"" | sed '/^Run: /d')
    if [ "$last" = "$(cat story-branches.md)" ]; then
        echo "story-branches-issue: #$keep already carries this report"
    else
        gh issue comment "$keep" --body "$body"
        echo "story-branches-issue: commented on #$keep"
    fi
    ;;
success)
    for n in $open; do
        gh issue close "$n" --comment "Every merged story branch is on main again. Run: $RUN_URL"
        echo "story-branches-issue: closed #$n"
    done
    ;;
*)
    echo "story-branches-issue: RESULT must be success or failure, got $RESULT" >&2
    exit 2
    ;;
esac
