#!/bin/sh
# make bruno-cli (TICK-1003): installs the pinned Bruno CLI into
# tools/bruno/node_modules from the committed package-lock.json with npm ci,
# so every transitive dependency is the locked version and integrity. Before
# this, `npx @usebruno/cli@<version>` resolved the dependency tree fresh on
# every run and fetched a tarball published 32 s earlier, which the registry
# did not serve yet (run 44604840061, 404 flow-estree-0.341.0.tgz).
#
# A no-op when node_modules already matches the lockfile (the stamp file is a
# copy of the lockfile it was installed from), so docs/acceptance/guide/
# tools/api-test.sh calls it before every run and pays for npm ci only once.
# The Bruno CLI runs no install scripts (no hasInstallScript in the lock), so
# --ignore-scripts. Prints the path of the CLI's entry point on stdout.
#
# Bumping the CLI: edit the version in tools/bruno/package.json, run
# `npm install --package-lock-only --ignore-scripts` in tools/bruno, commit
# both files (docs/quality/ci-pipeline.md, "Bruno CLI").
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
cli="$dir/node_modules/@usebruno/cli/bin/bru.js"
stamp="$dir/node_modules/.nava-lock.json"

command -v npm >/dev/null 2>&1 || { echo "bruno-cli: npm not found; install Node.js (verified with 22)" >&2; exit 2; }
if [ -f "$cli" ] && [ -f "$stamp" ] && cmp -s "$dir/package-lock.json" "$stamp"; then
    printf '%s\n' "$cli"
    exit 0
fi
echo "bruno-cli: npm ci in $dir (from package-lock.json)" >&2
# npm ci's own output goes to stderr: stdout is the CLI path only.
(cd "$dir" && npm ci --ignore-scripts --no-audit --no-fund >&2)
[ -f "$cli" ] || { echo "bruno-cli: $cli missing after npm ci" >&2; exit 1; }
cp "$dir/package-lock.json" "$stamp"
printf '%s\n' "$cli"
