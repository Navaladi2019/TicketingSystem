#!/bin/sh
# stack-case: job=stack order=12 mode=parallel
# The Temporal SDK suite (internal/temporalstack); writes the CI-6 report. The Go suite keeps its recipe in make/temporal-sdk.mk.
set -eu
. "$(dirname "$0")/../case-env.sh"
exec make --no-print-directory stack-test-sdk
