#!/bin/sh
# stack-case: job=stack order=20 mode=serial
# nava-worker service (T31b). After sdk: its workers poll nava-tq too.
set -eu
. "$(dirname "$0")/../case-env.sh"
exec make --no-print-directory stack-test-worker
