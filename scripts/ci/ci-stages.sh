#!/bin/sh
# make ci-stages: the CI stages the change against BASE needs, from
# tools/ci/cichanged (the rules make ci-changed uses). CI_LINT_ONLY=1 (a
# pull request without the ci:run label, TICK-563) runs ci-lint alone, before
# any other rule. Two cases run every stage: CI_ALL=1 (the workflow's daily
# run or manual run on main, TICK-652; no longer a push to main) and a change
# under .github/workflows, since only a full run proves a workflow edit.
# CI_FULL=1 (a full run: the ci:full label or a manual run, TICK-652; and a
# pull request into a TICK-* story branch, TICK-638) runs every stage and
# every stack-case area whatever its diff and wins over CI_LINT_ONLY; unlike
# CI_ALL its images follow the rule.
# A push to main diffs against the commit before it (BASE, a sha, TICK-652).
# A BASE that is the zero sha (a new ref) or a commit this clone does not
# have (a force push dropped it) has nothing to diff against, so it runs as
# CI_ALL=1.
# With GITHUB_OUTPUT set, also writes unit, integration and stack as
# true/false for the workflow's job conditions, and areas: the stack-case
# areas (all, none or a comma list, `cichanged -areas`) that make stack-tests
# runs (TICK-563). Every area runs with CI_ALL=1, with CI_FULL=1, with
# CI_AREAS_ALL=1 (a pull request into main) and for a workflow change.
# And images (TICK-628): real or thin, how the stack job builds its images
# (`cichanged -images`). CI_ALL=1 (main) is always real. CI_IMAGES=real or
# thin (a manual run's images input) overrides the rule on any other ref. A
# run without the stack stage writes real, which nothing reads.
set -eu

all="ci-lint ci-unit ci-integration ci-stack"
areas=
base=${BASE:-origin/main}
# Only a sha (a push's before) falls back; a missing ref name stays an error below.
case $base in
'' | *[!0-9a-f]*) ;;
*)
    if [ "$base" = 0000000000000000000000000000000000000000 ] || ! git cat-file -e "$base^{commit}" 2>/dev/null; then
        CI_ALL=1
        echo "ci-stages: BASE $base is the zero sha or not in this clone (new ref or force push), CI_ALL=1" >&2
    fi
    ;;
esac
if [ "${CI_FULL:-}" = 1 ]; then
    stages=$all
    areas=all
    echo "ci-stages: CI_FULL=1 (ci:full, manual run or pull request into a story branch), every stage" >&2
elif [ "${CI_LINT_ONLY:-}" = 1 ]; then
    stages=ci-lint
    areas=none
    echo "ci-stages: CI_LINT_ONLY=1 (no ci:run label), ci-lint only" >&2
elif [ "${CI_ALL:-}" = 1 ]; then
    stages=$all
    areas=all
    echo "ci-stages: CI_ALL=1, every stage" >&2
else
    mb=$(git merge-base "$base" HEAD) || { echo "ci-stages: no merge base with $base; fetch it first" >&2; exit 2; }
    if [ -n "$(git diff --name-only --no-renames "$mb" -- .github/workflows)" ]; then
        stages=$all
        areas=all
        echo "ci-stages: .github/workflows changed, every stage" >&2
    else
        stages=$(go run ./tools/ci/cichanged -v -base "$base")
    fi
fi
# The coverage job merges the unit and the integration profile (TICK-518), so
# a change that runs unit tests runs the integration tests too: coverage then
# runs on every change to Go code and is never skipped as "not needed".
case " $stages " in
    *" ci-unit "*)
        case " $stages " in
            *" ci-integration "*) ;;
            *)
                stages=$(echo "$stages" | sed 's/ci-unit/ci-unit ci-integration/')
                echo "ci-stages: ci-unit needs ci-integration for the coverage job (TICK-518)" >&2
                ;;
        esac
        ;;
esac
echo "$stages"

if [ -z "$areas" ]; then
    case " $stages " in
        *" ci-stack "*)
            if [ "${CI_AREAS_ALL:-}" = 1 ]; then
                areas=all
                echo "ci-stages: CI_AREAS_ALL=1, every stack-case area" >&2
            else
                areas=$(go run ./tools/ci/cichanged -areas -v -base "$base")
            fi
            ;;
        *) areas=none ;;
    esac
fi
echo "ci-stages: stack-case areas: $areas" >&2

images=real
if [ "${CI_ALL:-}" = 1 ]; then
    echo "ci-stages: CI_ALL=1, real images" >&2
else
    case " $stages " in
        *" ci-stack "*)
            case "${CI_IMAGES:-}" in
                real | thin)
                    images=$CI_IMAGES
                    echo "ci-stages: CI_IMAGES=$CI_IMAGES" >&2
                    ;;
                '' | auto) images=$(go run ./tools/ci/cichanged -images -v -base "$base") ;;
                *) echo "ci-stages: CI_IMAGES=$CI_IMAGES: want real, thin or auto" >&2; exit 2 ;;
            esac
            ;;
    esac
fi
echo "ci-stages: stack images: $images" >&2

if [ -n "${GITHUB_OUTPUT:-}" ]; then
    for s in unit integration stack; do
        case " $stages " in
            *" ci-$s "*) echo "$s=true" ;;
            *) echo "$s=false" ;;
        esac
    done >>"$GITHUB_OUTPUT"
    echo "areas=$areas" >>"$GITHUB_OUTPUT"
    echo "images=$images" >>"$GITHUB_OUTPUT"
fi
