#!/bin/sh
# Additive-only rule for /api/v1: fails when the bundled contract removes or
# narrows anything the base ref offers. Before the MS-6 split the base has the
# whole contract in api/openapi.yaml; after it, in api/openapi.bundled.yaml.
# Usage: check-openapi.sh <base-ref>
set -eu
cd "$(dirname "$0")/../.."
base=${1:-origin/main}
tmp=${TMPDIR:-${TEMP:-/tmp}}
for f in api/openapi.bundled.yaml api/openapi.yaml; do
	if git cat-file -e "$base:$f" 2>/dev/null; then
		git show "$base:$f" > "$tmp/openapi.base.yaml"
		exec go run ./cmd/openapi-compat "$tmp/openapi.base.yaml" api/openapi.bundled.yaml
	fi
done
echo "no api spec on $base: nothing to compare"
