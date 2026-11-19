#!/bin/sh
# TICK-961 acceptance (Go 1.27 upgrade): one command from a clean state to the
# verdict, on this checkout of the story branch.
#
#   1. Toolchain (AC1): go.mod and temporal-server/go.mod, `go build ./...` in
#      both modules, and the Go version stamped into the nava-core image.
#   2. A fresh stack (`docker compose down -v`, `up --build --wait`), the
#      guide seed (tenants A and B), admin@tenant-b.test (S18 kit.mjs admin-b,
#      one group sync), /readyz.
#   3. tools/kit.mjs: AC2 (v7 ids), AC3 (strict bodies, nothing written), AC5
#      (tenant B against tenant A's new rows) and the two exports for AC4.
#   4. AC4: `nava audit verify` on both exports; the golden hash tests
#      (vectors, S13 samples, verify, cmd/nava) and the story diff on golden
#      files (none).
#
# Usage, from the repository root (Git Bash or any POSIX shell), Docker up:
#   sh docs/acceptance/go-1-27-upgrade/tools/run-kit.sh
# Ports and the project name come from .env, as compose reads them. The
# verified run used COMPOSE_PROJECT_NAME=nava-t961 and ports 48080 / 48081.
# KIT_SKIP_UP=1 skips the down -v / up of step 2 (the stack is already fresh).
set -eu
cd "$(dirname "$0")/../../../.."
if [ -f .env ]; then set -a; . ./.env; set +a; fi
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nava-t961}
export COMPOSE_PROJECT_NAME
EV=${T961_EVIDENCE:-docs/acceptance/go-1-27-upgrade/evidence}
rm -rf "$EV"
mkdir -p "$EV"
FAILED=
dc() { MSYS_NO_PATHCONV=1 docker compose "$@"; }

echo "t961: 1. toolchain"
{
  echo "commit: $(git rev-parse HEAD)"
  echo "host go: $(go version)"
  grep -E '^(go|toolchain) ' go.mod | sed 's/^/go.mod: /'
  grep -E '^(go|toolchain) ' temporal-server/go.mod | sed 's/^/temporal-server\/go.mod: /'
} > "$EV/toolchain.txt"
if go build ./... && (cd temporal-server && go build ./...); then
  echo "go build ./... (root and temporal-server): ok" >> "$EV/toolchain.txt"
else
  echo "go build ./...: FAILED" >> "$EV/toolchain.txt"; FAILED="$FAILED AC1-build"
fi
grep -q '^toolchain go1.27.4$' go.mod && grep -q '^toolchain go1.27.4$' temporal-server/go.mod || FAILED="$FAILED AC1-gomod"

if [ "${KIT_SKIP_UP:-0}" != 1 ]; then
  echo "t961: 2. fresh stack ($COMPOSE_PROJECT_NAME)"
  make dev-ca
  dc down -v
  dc up -d --build --wait
fi
dc ps --format '{{.Service}} {{.Status}}' > "$EV/stack-ps.txt"
env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/seed.sh
node docs/acceptance/S18/tools/kit.mjs admin-b
dc run --rm --no-deps nava-groupsync --once
node docs/acceptance/S18/tools/kit.mjs me-b "$EV"
dc exec -T nava-core wget -q -O - http://127.0.0.1:9090/readyz > "$EV/readyz.txt" || FAILED="$FAILED readyz"
# The Go version the nava-core image was built with.
cid=$(dc ps -q nava-core)
rm -rf "$EV/.bin" && mkdir -p "$EV/.bin"
docker cp "$cid:/usr/local/bin/nava-core" "$EV/.bin/nava-core" >/dev/null
go version "$EV/.bin/nava-core" | sed "s#$EV/.bin/##" | tee -a "$EV/toolchain.txt"
go version "$EV/.bin/nava-core" | grep -q 'go1.27.4$' || FAILED="$FAILED AC1-image"
rm -rf "$EV/.bin"

echo "t961: 3. API and database checks"
node docs/acceptance/go-1-27-upgrade/tools/kit.mjs "$EV" 2>&1 | tee "$EV/kit.log" || true
grep -q '^FAIL' "$EV/kit.log" && FAILED="$FAILED kit"
grep -q 'checks pass$' "$EV/kit.log" || FAILED="$FAILED kit-incomplete"

echo "t961: 4. nava audit verify and golden hashes"
task build >/dev/null
NAVA=bin/nava
[ -x bin/nava.exe ] && NAVA=bin/nava.exe
for t in a b; do
  set +e
  "$NAVA" audit verify "$EV/export-tenant-$t.jsonl" > "$EV/verify-tenant-$t.txt" 2>&1
  rc=$?
  set -e
  echo "exit $rc" >> "$EV/verify-tenant-$t.txt"
  cat "$EV/verify-tenant-$t.txt"
  [ "$rc" = 0 ] || FAILED="$FAILED AC4-verify-$t"
done
go test -count=1 ./internal/audit/chain/... ./internal/audit/verify/... ./internal/audit/jsondiff/... ./cmd/nava/... ./tools/acceptance/s13kit/... \
  > "$EV/golden-tests.txt" 2>&1 || FAILED="$FAILED AC4-golden-tests"
{
  echo "story diff against origin/main on golden files and S13 samples (empty = unchanged):"
  git diff --stat origin/main...HEAD -- internal/audit/chain/testdata docs/acceptance/S13 temporal-server/authz/testdata internal/audit/verify/testdata
} > "$EV/golden-diff.txt"
[ "$(git diff --name-only origin/main...HEAD -- internal/audit/chain/testdata docs/acceptance/S13 temporal-server/authz/testdata internal/audit/verify/testdata | wc -l)" = 0 ] || FAILED="$FAILED AC4-golden-diff"
cat "$EV/golden-tests.txt"

if [ -n "$FAILED" ]; then echo "t961: FAILED:$FAILED; evidence in $EV" | tee "$EV/result.txt" >&2; exit 1; fi
echo "t961: PASS; evidence in $EV" | tee "$EV/result.txt"
