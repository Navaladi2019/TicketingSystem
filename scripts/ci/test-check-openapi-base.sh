#!/bin/sh
# make test-check-openapi-base (TICK-1179): on a pull_request run,
# check-openapi.sh compares the merge commit it tests against that commit's
# first parent, not against an origin/main fetched later in the run. A main
# that moved on and added a property must not fail the PR (run 45917518839,
# "property diff_nul_escaped removed"); a property the PR really removes still
# fails, and a push run keeps comparing against the base it is given.
# A throwaway git repo with the real check-openapi.sh and openapi-compat; no
# network, no Docker.
set -eu
cd "$(dirname "$0")/../.."
root=$(pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
fail=0
check() { if eval "$2"; then echo "ok:   $1"; else echo "FAIL: $1" >&2; fail=1; fi; }

go build -o "$tmp/openapi-compat" ./cmd/openapi-compat
export OPENAPI_COMPAT="$tmp/openapi-compat"

# spec <file> <property>...: a one-path contract whose Item schema has the
# given properties.
spec() {
	f=$1
	shift
	{
		printf 'openapi: 3.0.3\ninfo: {title: t, version: "1"}\npaths:\n'
		printf '  /items:\n    get:\n      responses:\n        "200":\n'
		printf '          description: ok\n          content:\n            application/json:\n'
		printf '              schema: {$ref: "#/components/schemas/Item"}\n'
		printf 'components:\n  schemas:\n    Item:\n      type: object\n      properties:\n'
		for p in "$@"; do printf '        %s: {type: string}\n' "$p"; done
	} >"$f"
}

repo="$tmp/repo"
mkdir -p "$repo/scripts/ci" "$repo/api"
cp scripts/ci/check-openapi.sh "$repo/scripts/ci/"
cd "$repo"
git init -q -b main
git config user.name t
git config user.email t@example.invalid
git config commit.gpgsign false
git config core.autocrlf false
: >api/approved-breaking-changes.txt
spec api/openapi.bundled.yaml id
git add -A && git commit -qm base
old=$(git rev-parse HEAD)

# main moves on during the run: another PR adds a property.
spec api/openapi.bundled.yaml id diff_nul_escaped
git commit -qam "main adds diff_nul_escaped"
git update-ref refs/remotes/origin/main HEAD

# run <name> <event> <base> <expect: pass|fail> <property>...: the PR
# branch, cut from the old base, sets the Item properties; HEAD is its merge
# into the old base, as actions/checkout gives a pull_request run. With
# head=tip HEAD is the PR branch tip itself (a checkout with ref: head.sha).
run() {
	name=$1 event=$2 base=$3 want=$4
	shift 4
	git checkout -q -B "pr-$name" "$old"
	spec api/openapi.bundled.yaml "$@"
	git commit -qam "pr $name" --allow-empty
	if [ "${head:-merge}" = merge ]; then
		git checkout -q --detach "$old"
		git merge -q --no-ff --no-edit "pr-$name"
	fi
	if GITHUB_EVENT_NAME=$event sh scripts/ci/check-openapi.sh "$base" >"$tmp/$name.log" 2>&1; then got=pass; else got=fail; fi
	check "$name: $want" '[ "$got" = "$want" ] || { sed "s/^/      /" "$tmp/$name.log" >&2; false; }'
}

run pr-unrelated pull_request origin/main pass id other
run pr-removes pull_request origin/main fail other
# A push to main compares against the base it is given (github.event.before).
run push-removes push origin/main fail id
run push-same push "$old" pass id
# HEAD is the PR tip, not a merge, and origin/main is ahead: the base is the
# merge-base with origin/main, the commit the PR is built on (TICK-1179 round 1).
head=tip run pr-tip-unrelated pull_request origin/main pass id other
head=tip run pr-tip-removes pull_request origin/main fail other

cd "$root"
[ "$fail" -eq 0 ] && echo "test-check-openapi-base: ok"
exit "$fail"
