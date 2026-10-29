#!/bin/sh
# stack-case: job=stack order=15 mode=serial
# The CI-5 adversarial identity suite (T19); writes ci5-report.md. Before dependencies stops Keycloak.
set -eu
. "$(dirname "$0")/../case-env.sh"
exec make --no-print-directory identity-attacks
