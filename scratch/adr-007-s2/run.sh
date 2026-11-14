#!/bin/sh
# ADR-007 spike S2 (TICK-807): a client_credentials service account as a
# member of one organization. Starts a throwaway Keycloak 26.8.1 (start-dev,
# realm nava imported from deploy/compose/keycloak/import, the compose realm)
# and a throwaway PostgreSQL (the docs/pins.md tag), runs s2_test.go against
# them, and removes both. Writes the raw evidence to
# scratch/adr-007-s2/evidence-raw.md. Needs Docker. Run from anywhere:
#
#	sh scratch/adr-007-s2/run.sh
set -eu
cd "$(dirname "$0")/../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

kc_tag=$(sed -n 's/^| `quay.io\/keycloak\/keycloak` | \([0-9.]*\) .*/\1/p' docs/pins.md)
pg_tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$kc_tag" ] && [ -n "$pg_tag" ] || { echo "pins not found in docs/pins.md" >&2; exit 1; }

kc="nava-s2-kc-$$"
pg="nava-s2-pg-$$"
trap 'docker rm -fv "$kc" "$pg" >/dev/null 2>&1 || true' EXIT
trap 'exit 130' INT TERM

wait_healthy() {
	deadline=$(( $(date +%s) + $2 ))
	until [ "$(docker inspect -f '{{.State.Health.Status}}' "$1")" = healthy ]; do
		[ "$(date +%s)" -lt "$deadline" ] || { docker logs "$1" | tail -50; echo "$1 not healthy within $2s" >&2; exit 1; }
		sleep 2
	done
}
hostport() { docker port "$1" "$2/tcp" | head -n 1 | sed 's/.*://'; }

docker run -d --name "$kc" -p 127.0.0.1::8080 \
	-v "$(pwd)/deploy/compose/keycloak/import:/opt/keycloak/data/import:ro" \
	-e KC_BOOTSTRAP_ADMIN_USERNAME=admin -e KC_BOOTSTRAP_ADMIN_PASSWORD=dev-only-admin -e KC_HEALTH_ENABLED=true \
	--health-cmd "exec 3<>/dev/tcp/127.0.0.1/9000 && printf 'GET /health/ready HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n' >&3 && head -n1 <&3 | grep -q ' 200 '" \
	--health-interval 3s --health-timeout 5s --health-retries 100 \
	"quay.io/keycloak/keycloak:$kc_tag" start-dev --import-realm >/dev/null
docker run -d --name "$pg" \
	-e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=test-only -e POSTGRES_INITDB_ARGS=--no-sync \
	--health-cmd 'pg_isready -h 127.0.0.1 -U pgadmin -d postgres' \
	--health-interval 1s --health-timeout 5s --health-retries 90 \
	-p 127.0.0.1::5432 "postgres:$pg_tag" -c fsync=off >/dev/null

wait_healthy "$pg" 120
wait_healthy "$kc" 300

NAVA_S2_KEYCLOAK_URL="http://127.0.0.1:$(hostport "$kc" 8080)"
NAVA_S2_PG_URL="postgres://pgadmin:test-only@127.0.0.1:$(hostport "$pg" 5432)/postgres"
NAVA_S2_EVIDENCE="$(pwd -W 2>/dev/null || pwd)/scratch/adr-007-s2/evidence-raw.md"
NAVA_S2_KEYCLOAK_VERSION="$kc_tag"
export NAVA_S2_KEYCLOAK_URL NAVA_S2_PG_URL NAVA_S2_EVIDENCE NAVA_S2_KEYCLOAK_VERSION
go test -tags spike -count=1 -v -run '^TestSpikeS2$' ./scratch/adr-007-s2/ "$@"
