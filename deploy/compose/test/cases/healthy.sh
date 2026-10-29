#!/bin/sh
# stack-case: job=stack,stack-db order=0 mode=serial
set -eu
. "$(dirname "$0")/../case-env.sh"

# Every container healthy, db-roles and migrate done, within the S1 budget of
# 180 s from the first container's creation, so `up -d` counts (TICK-164). The
# test and CI overlays only publish a port and drop durability, and the
# operator profile adds the Temporal UI, so the CI stack has the same budget.
# The internal overlay keeps 300 s (t_internal). The 180 s gate holds when the
# stack has the host to itself, as on a GitHub runner. While other compose
# stacks run on the host it stretches to CONTENDED_WAIT_SECONDS (300 s): two
# Keycloaks booting at once took 233 s (docs/quality/ci-optimizations.md), and
# a busy host failed the 180 s budget with nothing wrong (TICK-200).
t_healthy() { wait_healthy "${WAIT_SECONDS:-180}" "$(stack_created)" "${CONTENDED_WAIT_SECONDS:-300}"; }

t_healthy
