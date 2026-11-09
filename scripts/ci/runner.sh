#!/bin/sh
# make ci-runner (TICK-641): which runner the ci workflow's jobs use.
#   RUNNER_INPUT  a manual run's `runner` input (github or self-hosted), or empty
#   RUNNER_VAR    the repository variable CI_RUNNER, the default of every
#                 other event (github or self-hosted), or empty (github)
#   CROSS_REPO    true on a pull request whose head is another repository (a
#                 fork): always github, whatever the input or variable say. A
#                 fork's code must never run on the owner's machine.
# The input wins over the variable. Writes runs_on (JSON for runs-on:
# fromJSON) and self_hosted (true/false) to $GITHUB_OUTPUT, or to stdout
# outside Actions. An unknown value fails the run rather than guess.
set -eu

choice=${RUNNER_INPUT:-}
from="the runner input"
if [ -z "$choice" ]; then
    choice=${RUNNER_VAR:-github}
    from="the CI_RUNNER variable"
fi
case $choice in
    github | self-hosted) ;;
    *) echo "ci-runner: $from is '$choice'; want github or self-hosted" >&2; exit 2 ;;
esac
if [ "${CROSS_REPO:-}" = true ] && [ "$choice" != github ]; then
    echo "ci-runner: a pull request from a fork runs on GitHub-hosted runners only, not $choice"
    choice=github
fi

case $choice in
    self-hosted) runs_on='["self-hosted","wsl"]' self_hosted=true ;;
    *) runs_on='"ubuntu-latest"' self_hosted=false ;;
esac
echo "ci-runner: $choice (runs-on $runs_on)"
out=${GITHUB_OUTPUT:-/dev/stdout}
{
    echo "runs_on=$runs_on"
    echo "self_hosted=$self_hosted"
} >>"$out"
