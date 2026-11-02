#!/bin/sh
# stack-case: job=stack order=58 mode=serial
# P3-24 (TICK-351): nava-worker-interptest killed after ApplyStep's write committed, restarted, and the effect applied once. Kills and restarts that worker, and stops and restarts the always-on nava-worker, so it sorts after the read-only and observability cases.
set -eu
. "$(dirname "$0")/../case-env.sh"
exec make --no-print-directory stack-test-interpreter
