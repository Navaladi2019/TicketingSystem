.PHONY: test-integration test-integration-cleanup reap-integration bench-relay isolation test-ci-isolated-down

ci-integration: test-integration test-ci-isolated-down test-integration-cleanup

# Every package with `//go:build integration` tests, against a throwaway
# postgres container the script starts and removes (needs Docker), then the
# isolation suite (make isolation). Pass ARGS=... to forward flags to go test;
# with ARGS the isolation suite does not run. The ci-isolated --down tag check
# is its own target (test-ci-isolated-down, in ci-integration and its own
# workflow step), so this loop stays fast (TICK-224).
test-integration:
	sh scripts/ci/test-integration.sh $(ARGS)
	$(if $(ARGS),@echo "isolation suite skipped (ARGS set); run make isolation",sh scripts/ci/isolation.sh)

# test-integration.sh leaves no container or network behind after a run with
# go test arguments or a TERM part way (TICK-185). Its own ci.yml step and in
# ci-integration, not part of make test-integration (as TICK-224). Needs Docker.
test-integration-cleanup:
	sh scripts/ci/test-integration-cleanup.sh

# Removes the nava-it-* and nava-iso-* containers and networks that a killed
# integration or isolation run left, once older than REAP_OLDER (a Go
# duration, default 6h): never a compose stack (nava, nava-<id>) or a younger
# run (TICK-185). REAP_DRY=1 lists them only. Needs Docker.
REAP_OLDER ?= 6h
reap-integration:
	go run ./tools/ci/itreap -older $(REAP_OLDER) $(if $(REAP_DRY),-n)

# T33: the relay's claim latency (BenchmarkClaim, internal/relay) against
# ADR-001's 5.4 ms baseline, on the same throwaway clusters as
# test-integration. Report-only: a nightly workflow runs it and nothing
# gates on it. Needs Docker.
# TODO(TODO-011): no nightly workflow runs it yet; the agent token cannot add
# .github/workflows/nightly.yml (no `workflow` scope).
bench-relay:
	sh scripts/ci/test-integration.sh -run '^$$' -bench '^BenchmarkClaim$$' -benchtime 200x

# The tenant isolation suite (T11, ADR-001 CI-2): its own throwaway
# PostgreSQL and a PgBouncer with the real auth_query, tenants A and B seeded,
# every runtime role against every tenant table plus the registered paths
# (internal/tenancy/isolation). Writes ISOLATION_REPORT (default
# isolation-report.md). Needs Docker.
# The integration job uploads it as the isolation-report artifact.
isolation:
	sh scripts/ci/isolation.sh $(ARGS)

# ci-isolated-down untags every nava/*:<id> tag and nothing else (TICK-211):
# throwaway tags on an empty imported image, a fresh id per run. Needs Docker.
test-ci-isolated-down:
	sh scripts/ci/test-ci-isolated-down.sh

