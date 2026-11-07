#!/bin/sh
# stack-case: job=stack order=22 mode=serial
# The S7 demo, one key, one workflow, two signals.
# It stops nava-worker while its relayProbe worker polls nava-tq (TICK-453); with_worker_restored brings it back.
set -eu
. "$(dirname "$0")/../case-env.sh"
with_worker_restored make --no-print-directory stack-test-demo_outbox
