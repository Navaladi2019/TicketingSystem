#!/bin/sh
# stack-case: job=none order=10 mode=parallel
# Not a CI step of its own: stack-test.sh temporal_isolation runs it too.
set -eu
. "$(dirname "$0")/../case-env.sh"

# The authoritative half of temporal_isolation, also runnable on its own.
# Docker resolves a container's name on every network it is on, alias or not,
# and an IP is reachable whatever DNS says, so judge by network membership
# (TICK-199): temporal-server is on temporal-backplane and nothing else, and
# shares no network with a platform process. The gateway (temporal) is on both
# by design and out of scope.

t_temporal_confinement
