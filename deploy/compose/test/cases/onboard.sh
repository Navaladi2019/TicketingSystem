#!/bin/sh
# stack-case: job=stack order=16 mode=serial
# Onboarding end to end (T21b, TICK-234). The only run of these tests: identity skips them (TICK-628).
set -eu
. "$(dirname "$0")/../case-env.sh"
exec make --no-print-directory stack-test-onboard
