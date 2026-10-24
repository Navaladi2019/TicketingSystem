# Keycloak with the Quarkus build done at image build time, so `start --optimized`
# boots without re-augmenting (TICK-156: `start-dev` re-ran the build on every
# start, 105-190 s on the dev host). Build context is the repo root.
# Build-time options only here; runtime ones (database, hostname, HTTP) are set
# in compose.yaml. They must match what compose.yaml passes: KC_DB is fixed here.
FROM quay.io/keycloak/keycloak:26.8.1
ENV KC_DB=postgres \
    KC_HEALTH_ENABLED=true \
    KC_CACHE=local
RUN /opt/keycloak/bin/kc.sh build
