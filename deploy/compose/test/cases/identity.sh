#!/bin/sh
# stack-case: job=stack order=14 mode=parallel
# Realm nava, operator custody and the Keycloak adapter (make/identity.mk). Runs before dependencies stops Keycloak.
set -eu
. "$(dirname "$0")/../case-env.sh"
exec make --no-print-directory stack-test-identity
