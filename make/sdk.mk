# TICK-827 (PLAN-008 P8-22, ADR-007 D10, RFC-017 CI-E10): the TypeScript SDK
# in sdk/ts, as the ci workflow's sdk job runs it. Node is the version pinned
# in sdk/ts/.nvmrc; the toolchain is pinned by sdk/ts/package-lock.json
# (npm ci). Same commands as task sdk:gen, sdk:test and sdk:build (TICK-826).

SDK_DIR := sdk/ts

.PHONY: sdk-deps check-sdk-generated sdk-lint sdk-test sdk-pack ci-sdk-changed

sdk-deps:
	cd $(SDK_DIR) && npm ci

## CI-E10: regenerate the client from api/openapi.bundled.yaml and fail when
## the committed sdk/ts/src/generated differs or gains an untracked file.
check-sdk-generated: sdk-deps
	cd $(SDK_DIR) && npm run gen
	@git diff --exit-code -- $(SDK_DIR)/src/generated || { echo "check-sdk-generated: $(SDK_DIR)/src/generated is stale against api/openapi.bundled.yaml; run task sdk:gen and commit the result (CI-E10)" >&2; exit 1; }
	@untracked=$$(git ls-files --others --exclude-standard -- $(SDK_DIR)/src/generated); \
	[ -z "$$untracked" ] || { echo "check-sdk-generated: untracked generated files: $$untracked; run task sdk:gen and commit them (CI-E10)" >&2; exit 1; }

## Strict type-check of the package, its tests included.
sdk-lint: sdk-deps
	cd $(SDK_DIR) && npm run lint

## The SDK tests, the type tests among them.
sdk-test: sdk-deps
	cd $(SDK_DIR) && npm test

## Build into sdk/ts/dist and npm pack the tarball into sdk/ts; nothing is
## published (D-8-7).
sdk-pack: sdk-deps
	cd $(SDK_DIR) && npm run build && npm pack

## Whether the ci workflow's sdk job runs: prints true or false (TICK-827).
ci-sdk-changed:
	@sh scripts/ci/sdk-changed.sh
