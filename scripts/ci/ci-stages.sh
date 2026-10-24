#!/bin/sh
# make ci-stages: the CI stages the change against BASE needs, from
# tools/ci/cichanged (the rules make ci-changed uses). Two cases run every
# stage: CI_ALL=1 (the workflow's manual run) and a change under
# .github/workflows, since only a full run proves a workflow edit.
# With GITHUB_OUTPUT set, also writes unit, integration and stack as
# true/false for the workflow's job conditions.
set -eu

all="ci-lint ci-unit ci-integration ci-stack"
if [ "${CI_ALL:-}" = 1 ]; then
    stages=$all
    echo "ci-stages: CI_ALL=1, every stage" >&2
else
    base=${BASE:-origin/main}
    mb=$(git merge-base "$base" HEAD) || { echo "ci-stages: no merge base with $base; fetch it first" >&2; exit 2; }
    if [ -n "$(git diff --name-only --no-renames "$mb" -- .github/workflows)" ]; then
        stages=$all
        echo "ci-stages: .github/workflows changed, every stage" >&2
    else
        stages=$(go run ./tools/ci/cichanged -v -base "$base")
    fi
fi
echo "$stages"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
    for s in unit integration stack; do
        case " $stages " in
            *" ci-$s "*) echo "$s=true" ;;
            *) echo "$s=false" ;;
        esac
    done >>"$GITHUB_OUTPUT"
fi
