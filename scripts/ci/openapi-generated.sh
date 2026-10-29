#!/bin/sh
# MS-6 (TICK-294): api/openapi.bundled.yaml and api/api.gen.go are generated
# from api/openapi.yaml and api/paths/*.yaml. Regenerate and fail on any
# difference, so an edit to the source without `go generate ./api` is caught
# and generated files are never hand-merged.
set -eu
cd "$(dirname "$0")/../.."
go generate ./api
if ! git diff --exit-code -- api/ || [ -n "$(git ls-files --others --exclude-standard -- api/)" ]; then
	git ls-files --others --exclude-standard -- api/ | sed 's/^/untracked: /'
	echo "openapi-generated: api/ differs from a fresh 'go generate ./api'; run it and commit the result" >&2
	exit 1
fi
