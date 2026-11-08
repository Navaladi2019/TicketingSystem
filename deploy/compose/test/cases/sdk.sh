#!/bin/sh
# stack-case: job=stack order=12 mode=serial
# The Temporal SDK suite (internal/temporalstack); writes the CI-6 report. The Go suite keeps its recipe in make/temporal-sdk.mk.
# Serial (TICK-453): it stops nava-worker while its bodies poll nava-tq, so it does not only read the stack.
set -eu
. "$(dirname "$0")/../case-env.sh"
with_worker_restored make --no-print-directory stack-test-sdk
