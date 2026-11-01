#!/bin/sh
# The S11a demo (make demo-interpreter-*, TICK-352): one nava.interpreter run
# of P3-20's fixture definition on the stack, started and signalled through
# outbox rows (cmd/demo-interpreter, outbox.Enqueue in InTenantTx as
# nava_app), run by nava-worker-interptest (compose.test.yaml, the S11a
# fakes) while the always-on nava-worker is stopped.
#
# Usage: sh scripts/dev/demo-interpreter.sh <action>   (the stack is up)
#   up        install fixture v1 for tenant-a and tenant-b, stop nava-worker,
#             start nava-worker-interptest. PAUSE=1: ApplyStep pauses after
#             its write commits (for the crash drill).
#   start     TENANT=a|b KEY=<name> PRIORITY=high|low: a record of TENANT
#             with that priority, and a start row on the tenant's current
#             definition version. RECORD_ID=<uuid> names an existing record
#             instead and writes no record file (the cross-tenant check).
#   approve   TENANT KEY (or RECORD_ID): the `approve` signal row for the
#             newest open run on the record.
#   publish-v2 TENANT: install fixture v2 as a new version; the tenant's
#             later starts use it.
#   pinning   TENANT KEY PRIORITY: start a run on the current version and
#             publish v2 the moment the relay dispatches its start row, so v2
#             lands while the run waits on its 5 s timer.
#   crash     TENANT KEY: with PAUSE=1, send approve, wait until mark_approved's
#             write committed and ApplyStep paused, SIGKILL the worker,
#             release the pause and start the same container again.
#   writes    the fake writer's calls (calls.jsonl) and applied effects.
#   status    TENANT KEY: the record's outbox rows and workflow_ref.
#   down      remove nava-worker-interptest, start nava-worker again.
set -eu
cd "$(dirname "$0")/../.."

# Git Bash would rewrite the container paths.
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi

PROJECT=${COMPOSE_PROJECT_NAME:-nava}
export COMPOSE_PROJECT_NAME="$PROJECT"
ACTION=${1:-}
TENANT=${TENANT:-a}
case "$TENANT" in
a) TENANT_ID=00000000-0000-4000-8000-0000000000a1 ;;
b) TENANT_ID=00000000-0000-4000-8000-0000000000b1 ;;
*) echo "demo-interpreter: TENANT must be a or b" >&2; exit 2 ;;
esac
# The definition versions the demo installs (any UUIDs; fixed so a rerun is stable).
V1=51a11000-0000-4000-8000-000000000001
V2=51a11000-0000-4000-8000-000000000002
FIXTURE=internal/workflow/def/testdata
# Host side of the fixture mount (read-only in the worker), and the worker's
# writer and pause directories in the interptest-state volume, which outlives
# a container kill.
LOCAL=.local/s11a
STATE=/run/interptest/state/s11a
mkdir -p "$LOCAL"
[ -n "${PAUSE:-}" ] || PAUSE=$(cat "$LOCAL/pause" 2>/dev/null || echo 0)
export NAVA_INTERPTEST_FIXTURES="./$LOCAL/fixtures"
# FileWriter keeps its files under <root>/writes.
export NAVA_INTERPTEST_WRITES_DIR="$STATE"
if [ "$PAUSE" = 1 ]; then export NAVA_INTERPTEST_PAUSE_DIR="$STATE/pause"; else export NAVA_INTERPTEST_PAUSE_DIR=; fi
SVC=nava-worker-interptest

# COMPOSE_FILE, when set, must include deploy/compose/compose.test.yaml.
if [ -n "${COMPOSE_FILE:-}" ]; then
	dc() { docker compose --profile interptest "$@"; }
else
	dc() { docker compose -f compose.yaml -f deploy/compose/compose.test.yaml --profile interptest "$@"; }
fi
# oneshot runs a command in a fresh, removed container of the worker image, as root.
oneshot() { dc run --rm --no-deps -T --user 0 --entrypoint sh "$SVC" -c "$1"; }
psql_q() {
	dc exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -A -F " | " -v ON_ERROR_STOP=1 -U pgadmin -d nava -f -'
}
psql_t() {
	dc exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -A -t -v ON_ERROR_STOP=1 -U pgadmin -d nava -f -'
}
now() { date -u +%H:%M:%S.%3N; }

# install <tenant id> <version> <fixture name>: the definition and its step-id map.
install() {
	d="$LOCAL/fixtures/$1/definitions"
	mkdir -p "$d"
	cp "$FIXTURE/$3.json" "$d/$2.json"
	cp "$FIXTURE/$3.ids.json" "$d/$2.ids.json"
	echo "$2" > "$LOCAL/fixtures/$1/current-version"
}

hostbin() {
	mkdir -p bin
	go build -trimpath -o bin/demo-interpreter-host ./cmd/demo-interpreter
}
# record_id: RECORD_ID, else derived from the tenant and KEY.
record_id() {
	if [ -n "${RECORD_ID:-}" ]; then echo "$RECORD_ID"; return; fi
	[ -n "${KEY:-}" ] || { echo "demo-interpreter: KEY=<name> is required" >&2; exit 2; }
	hostbin
	DEMO_ACTION=record-id TENANT_ID=$TENANT_ID DEDUPE_KEY=$KEY bin/demo-interpreter-host
}

# enqueue <action> <dedupe key> <record> <target id>: cmd/demo-interpreter on the internal network.
enqueue() {
	net=$(docker network ls -q --filter "label=com.docker.compose.project=$PROJECT" --filter label=com.docker.compose.network=internal | head -n 1)
	[ -n "$net" ] || { echo "demo-interpreter: no internal network for compose project $PROJECT; is the stack up?" >&2; exit 1; }
	arch=$(docker version -f '{{.Server.Arch}}')
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -o bin/demo-interpreter-linux ./cmd/demo-interpreter
	ctr=$(docker create --network "$net" --entrypoint /demo-interpreter \
		-e "DEMO_ACTION=$1" -e "DEDUPE_KEY=$2" -e "TENANT_ID=$TENANT_ID" -e "RECORD_ID=$3" -e "VERSION_ID=$4" -e "SIGNAL_ID=$4" \
		-e "NAVA_DATABASE_URL=postgres://nava_app:${NAVA_APP_DB_PASSWORD:-dev-only-app}@pgbouncer:5432/nava?sslmode=disable" \
		"nava/nava-core:${NAVA_IMAGE_TAG:-dev}")
	status=0
	{ docker cp bin/demo-interpreter-linux "$ctr":/demo-interpreter && docker start -a "$ctr"; } || status=$?
	docker rm -f "$ctr" >/dev/null
	return "$status"
}

# approve_id: the approve signal's id in the step-id map of the version the tenant started on.
approve_id() {
	sed -n 's/.*"signal:approve": *"\([0-9a-f-]*\)".*/\1/p' "$FIXTURE/fixture-v1.ids.json"
}

do_start() {
	rec=$(record_id)
	version=$(cat "$LOCAL/fixtures/$TENANT_ID/current-version")
	if [ -z "${RECORD_ID:-}" ]; then
		mkdir -p "$LOCAL/fixtures/$TENANT_ID/records"
		printf '{"priority":"%s"}' "${PRIORITY:-high}" > "$LOCAL/fixtures/$TENANT_ID/records/$rec.json"
		echo "[$(now)] record $rec of tenant $TENANT: priority ${PRIORITY:-high}"
	fi
	echo "[$(now)] start: tenant $TENANT, definition version $version"
	enqueue start "${KEY:-$rec}:start" "$rec" "$version"
}

send_approve() {
	rec=$(record_id)
	echo "[$(now)] approve: tenant $TENANT, record $rec"
	enqueue signal "${KEY:-$rec}:approve" "$rec" "$(approve_id)"
}

case "$ACTION" in
up)
	install "00000000-0000-4000-8000-0000000000a1" "$V1" fixture-v1
	install "00000000-0000-4000-8000-0000000000b1" "$V1" fixture-v1
	echo "$PAUSE" > "$LOCAL/pause"
	echo "[$(now)] fixture v1 ($V1) installed for tenant-a and tenant-b; pause after commit: $PAUSE"
	oneshot "mkdir -p $STATE/writes $STATE/pause && chown -R 10001:10001 $STATE"
	dc stop nava-worker
	dc up -d --build --no-deps --wait --wait-timeout 300 "$SVC"
	echo "[$(now)] nava-worker stopped; $SVC healthy and the only nava-tq poller"
	;;
start)
	do_start
	;;
pinning)
	# Publish v2 the moment the relay dispatches the v1 run's start row, so it
	# lands while the run waits on its 5 s cool_off timer.
	[ -n "${KEY:-}" ] || { echo "demo-interpreter: KEY=<name> is required" >&2; exit 2; }
	(
		i=0
		until [ "$(printf "SELECT 1 FROM outbox WHERE tenant_id = '%s' AND dedupe_key = '%s:start' AND dispatched_at IS NOT NULL;" "$TENANT_ID" "$KEY" | psql_t)" = 1 ]; do
			i=$((i + 1)); [ "$i" -lt 600 ] || { echo "demo-interpreter: start row not dispatched within 2 min" >&2; exit 1; }
			sleep 0.2
		done
		echo "[$(now)] the v1 start row is dispatched: the run is starting its 5 s timer"
		install "$TENANT_ID" "$V2" fixture-v2
		echo "[$(now)] fixture v2 ($V2) published for tenant $TENANT, while the v1 run waits on its timer"
	) &
	watcher=$!
	do_start
	wait "$watcher"
	;;
approve)
	send_approve
	;;
publish-v2)
	install "$TENANT_ID" "$V2" fixture-v2
	echo "[$(now)] fixture v2 ($V2) published for tenant $TENANT: its later starts use v2; runs already started keep their version"
	;;
crash)
	[ "$(cat "$LOCAL/pause")" = 1 ] || { echo "demo-interpreter: run 'up' with PAUSE=1 first" >&2; exit 2; }
	send_approve
	i=0
	until marker=$(dc exec -T "$SVC" sh -c "ls $STATE/pause/*.paused 2>/dev/null | head -n 1") && [ -n "$marker" ]; do
		i=$((i + 1)); [ "$i" -lt 120 ] || { echo "demo-interpreter: no pause marker within 60 s" >&2; exit 1; }
		sleep 0.5
	done
	echo "[$(now)] ApplyStep committed its write and paused: $marker"
	before=$(docker inspect -f '{{.State.StartedAt}}' "$(dc ps -q "$SVC")")
	dc kill -s SIGKILL "$SVC"
	id=$(dc ps -a -q "$SVC")
	echo "[$(now)] killed: container ${id%"${id#????????????}"} running=$(docker inspect -f '{{.State.Running}} exit={{.State.ExitCode}}' "$id")"
	echo "calls in the volume after the kill (the killed attempt's write committed):"; oneshot "cat $STATE/writes/calls.jsonl"
	oneshot "touch ${marker%.paused}.release && chown 10001:10001 ${marker%.paused}.release"
	dc start "$SVC"
	until [ "$(docker inspect -f '{{.State.Health.Status}}' "$id")" = healthy ]; do sleep 1; done
	echo "[$(now)] same container started again (started $before, now $(docker inspect -f '{{.State.StartedAt}}' "$id")), healthy"
	;;
writes)
	oneshot "echo '--- calls.jsonl'; cat $STATE/writes/calls.jsonl 2>/dev/null; echo '--- applied effects (one file per key)'; for f in $STATE/writes/applied/*.json; do [ -f \"\$f\" ] && { cat \"\$f\"; echo; }; done; true"
	;;
status)
	rec=$(record_id)
	printf "SELECT o.kind, o.dedupe_key, o.dispatched_at IS NOT NULL AS dispatched, o.dead_at IS NOT NULL AS dead, coalesce(o.last_error, '') AS last_error FROM outbox o WHERE o.tenant_id = '%s' AND o.dedupe_key LIKE '%s:%%' ORDER BY o.created_at;\nSELECT r.kind, r.outbox_id, r.closed_at IS NOT NULL AS closed FROM workflow_ref r WHERE r.tenant_id = '%s' AND r.record_id = '%s';\n" \
		"$TENANT_ID" "${KEY:-$rec}" "$TENANT_ID" "$rec" | psql_q
	;;
down)
	dc rm --stop --force "$SVC"
	dc up -d --no-deps --wait --wait-timeout 180 nava-worker
	echo "[$(now)] $SVC removed; nava-worker healthy again"
	;;
*)
	sed -n '2,27p' "$0" >&2
	exit 2
	;;
esac
