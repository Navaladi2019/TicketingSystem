#!/bin/sh
# stack-case: job=stack order=20 mode=serial
# nava-worker service (T31b), and nava.interpreter started by the relay on nava-worker-interptest (TICK-367). After sdk and before demo_outbox: both poll nava-tq.
set -eu
. "$(dirname "$0")/../case-env.sh"
exec make --no-print-directory stack-test-worker
