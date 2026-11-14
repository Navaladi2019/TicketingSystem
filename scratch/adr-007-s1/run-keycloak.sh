#!/bin/sh
# Starts the S1 scratch Keycloak (the pinned 26.8.1 image, start-dev, H2) on
# localhost:18181 with realms spike-s1 and spike-upstream. Not the compose
# stack; nothing else on the host is touched. Prints the image digest.
# Stop: docker rm -f nava-spike-s1
set -eu
here=$(cd "$(dirname "$0")" && pwd)
docker rm -f nava-spike-s1 >/dev/null 2>&1 || true
MSYS_NO_PATHCONV=1 docker run -d --name nava-spike-s1 -p 127.0.0.1:18181:8080 \
  -e KC_BOOTSTRAP_ADMIN_USERNAME=admin -e KC_BOOTSTRAP_ADMIN_PASSWORD=dev-only-admin \
  -e KC_HOSTNAME=http://localhost:18181 \
  -v "$here/realm:/opt/keycloak/data/import:ro" \
  quay.io/keycloak/keycloak:26.8.1 start-dev --import-realm >/dev/null
i=0
until curl -fsS http://localhost:18181/realms/spike-s1/.well-known/openid-configuration >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -gt 90 ]; then docker logs nava-spike-s1 2>&1 | tail -40; exit 1; fi
  sleep 2
done
docker inspect --format '{{index .RepoDigests 0}}' quay.io/keycloak/keycloak:26.8.1
