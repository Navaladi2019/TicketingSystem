#!/bin/sh
# stack-case: job=stack order=16 mode=serial
# Onboarding end to end (T21b, TICK-234). identity runs these tests too until it skips them (TODO-014).
set -eu
. "$(dirname "$0")/../case-env.sh"
exec make --no-print-directory stack-test-onboard
