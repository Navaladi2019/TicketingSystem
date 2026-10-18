#!/bin/sh
# Integration stage (T05): every package with `//go:build integration` tests,
# against a throwaway PostgreSQL the script starts and removes. The same on a
# laptop and in CI; needs Docker. Image tag from docs/pins.md.
# Extra arguments go to `go test` (e.g. -run TestCICanary).
set -eu
cd "$(dirname "$0")/../.."

image=postgres:16.10
name="nava-it-pg-$$"

docker run -d --name "$name" \
    -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=test-only \
    --health-cmd 'pg_isready -h 127.0.0.1 -U pgadmin -d postgres' \
    --health-interval 1s --health-timeout 5s --health-retries 90 \
    -p 127.0.0.1::5432 "$image" >/dev/null
trap 'docker rm -f "$name" >/dev/null' EXIT

# Gate on the container's health check (TCP: the image's init server listens
# on the socket only, so a TCP answer means the real server is up).
deadline=$(( $(date +%s) + 120 ))
until [ "$(docker inspect -f '{{.State.Health.Status}}' "$name")" = healthy ]; do
    [ "$(date +%s)" -lt "$deadline" ] || { docker logs "$name"; echo "postgres not healthy within 120s" >&2; exit 1; }
    sleep 1
done
port=$(docker port "$name" 5432/tcp | head -n 1 | sed 's/.*://')
NAVA_TEST_PG_URL="postgres://pgadmin:test-only@127.0.0.1:$port/postgres"
export NAVA_TEST_PG_URL

pkgs=$(grep -rl --include='*_test.go' '^//go:build integration' cmd internal tools 2>/dev/null |
    xargs -n 1 dirname | sort -u | sed 's|^|./|')
echo "integration packages:" $pkgs
# -p 1: the dbtest suites share the cluster's roles.
# shellcheck disable=SC2086
go test -tags integration -p 1 -count=1 "$@" $pkgs
