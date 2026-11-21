#!/bin/sh
# Acceptance kits (TICK-1021): prints the nava/* image tag for this checkout,
# so two checkouts on one host never share an image. With the default `dev`
# tag they did: one stack rebuilt nava/nava-db:dev from its branch, and the
# S30 fresh stack then ran that branch's migrations (TICK-935).
#
#   <commit, 12 hex>                  clean tree
#   <commit, 12 hex>-dirty-<8 hex>    tracked changes or untracked files; the
#                                     hash covers their content, so an edit
#                                     gives a new tag and needs a rebuild
#
# Usage, from the repository root:
#   export NAVA_IMAGE_TAG=$(sh scripts/acceptance/image-tag.sh)
set -eu
cd "$(git rev-parse --show-toplevel)"
tag=$(git rev-parse --short=12 HEAD)
untracked=$(git ls-files --others --exclude-standard)
if ! git diff --quiet HEAD -- || [ -n "$untracked" ]; then
    # The diff against HEAD, the untracked files' names, and their blob ids.
    sum=$({
        git diff --binary HEAD --
        printf '%s\n' "$untracked"
        git ls-files -z --others --exclude-standard | xargs -0 -r git hash-object --
    } | git hash-object --stdin | cut -c1-8)
    tag="$tag-dirty-$sum"
fi
echo "$tag"
