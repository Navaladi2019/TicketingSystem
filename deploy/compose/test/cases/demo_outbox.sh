#!/bin/sh
# stack-case: job=stack order=22 mode=serial
# The S7 demo, one key, one workflow, two signals.
set -eu
. "$(dirname "$0")/../case-env.sh"
exec make --no-print-directory stack-test-demo_outbox
