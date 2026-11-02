#!/bin/sh
# Builds Temporal's determinism checker from its pinned module
# (tools/workflowcheck, D-3-23) and runs it on the workflow code. Test files
# are not checked: they hold test workflows, never registered on a worker.
# Arguments are packages; the default is ./internal/workflow/...
# Build tags go through GOFLAGS (the checker's -tags flag has no effect):
# scripts/ci/lint-fixture.sh sets GOFLAGS=-tags=lintfixture.
set -eu
cd "$(dirname "$0")/../.."
tool="bin/workflowcheck$(go env GOEXE)"
go -C tools/workflowcheck build -o "../../$tool" go.temporal.io/sdk/contrib/tools/workflowcheck
[ $# -gt 0 ] || set -- ./internal/workflow/...
"./$tool" -test=false -config tools/workflowcheck/workflowcheck.config.yaml "$@"
