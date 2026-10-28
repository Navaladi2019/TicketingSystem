#!/bin/sh
# Acceptance guide (TICK-287): local-only port forwards for the parts of the
# stack that publish no port, so Postman can reach them from the host.
#
#   127.0.0.1:18082  -> nava-onboard:8081   operator onboarding API (S4)
#   127.0.0.1:19090  -> nava-core:9090      admin listener: /healthz /readyz /metrics (S1)
#   127.0.0.1:18233  -> temporal-ui:8080    Temporal UI (S7; needs COMPOSE_PROFILES=operator)
#
# Each forward is a detached container of the nava/temporal-server image running
# its TCP `proxy` (the same one `make operator-tunnel` uses), on the project's
# published (non-internal) network and the network of its target. They bind
# 127.0.0.1 only. Like the operator tunnel (TODO-006), while they run, other
# containers on a non-internal network can reach the targets through the host:
# stop them when you are done testing. Not for anything but a local dev stack.
#
# Usage: sh docs/acceptance/guide/tools/tunnels.sh up|down|status
set -eu
cd "$(dirname "$0")/../../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
if [ -f .env ]; then set -a; . ./.env; set +a; fi

PROJECT=${COMPOSE_PROJECT_NAME:-nava}
IMAGE="nava/temporal-server:${NAVA_IMAGE_TAG:-dev}"
ONBOARD_PORT=${ACCEPT_ONBOARD_PORT:-18082}
ADMIN_PORT=${ACCEPT_ADMIN_PORT:-19090}
UI_PORT=${ACCEPT_UI_PORT:-18233}

net() {
  docker network ls -q --filter "label=com.docker.compose.project=$PROJECT" \
    --filter "label=com.docker.compose.network=$1" | head -n 1
}

# forward <name> <host port> <target network> <target host:port>
forward() {
  name="$PROJECT-accept-$1"
  docker rm -f "$name" >/dev/null 2>&1 || true
  pub=$(net default)
  [ -n "$pub" ] || { echo "tunnels: no default network for project $PROJECT; is the stack up?" >&2; exit 1; }
  target_net=$(net "$3")
  [ -n "$target_net" ] || { echo "tunnels: no $3 network for project $PROJECT" >&2; exit 1; }
  docker create --name "$name" --network "$pub" -p "127.0.0.1:$2:8080" \
    --entrypoint /usr/local/bin/temporal-server "$IMAGE" proxy -listen :8080 -to "$4" >/dev/null
  docker network connect "$target_net" "$name"
  docker start "$name" >/dev/null
  echo "tunnels: 127.0.0.1:$2 -> $4"
}

case "${1:-}" in
up)
  forward onboard "$ONBOARD_PORT" internal nava-onboard:8081
  forward admin "$ADMIN_PORT" internal nava-core:9090
  if [ -n "$(docker compose ps -q temporal-ui 2>/dev/null)" ]; then
    forward ui "$UI_PORT" operator temporal-ui:8080
  else
    echo "tunnels: temporal-ui is not running (set COMPOSE_PROFILES=operator, then docker compose up -d); skipped"
  fi
  ;;
down)
  for n in onboard admin ui; do docker rm -f "$PROJECT-accept-$n" >/dev/null 2>&1 || true; done
  echo "tunnels: stopped"
  ;;
status)
  docker ps --filter "name=$PROJECT-accept-" --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
  ;;
*)
  echo "usage: $0 up|down|status" >&2; exit 2 ;;
esac
