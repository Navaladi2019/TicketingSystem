.PHONY: stack-test-sdk

# The Temporal stack suite over mTLS: SDK clients dial and a worker runs (T28),
# CI-H, the CI-6 certificate, network and Authorizer cases (T32a), and the
# CI-6 relay and worker cases and CI-7 (T32b). Writes the CI-6 and CI-7 report
# to CI6_REPORT and fails when a case did not pass.
# suiteran prints the run and keeps the raw stream in CI6_JSON, which
# ci6report then reads (its echo of the same output is dropped).
# The suite takes about 70 s on a GitHub runner but 591-615 s on a host running
# two ci-isolated stacks, past go test's default 600 s, hence -timeout 20m
# (TICK-173).
CI6_REPORT ?= ci6-report.md
CI6_JSON ?= ci6-test.json
stack-test-sdk:
	@status=0; go run ./tools/ci/suiteran -tags temporalstack -tee $(CI6_JSON) -- \
		go test -tags temporalstack -count=1 -timeout 20m -json ./internal/temporalstack || status=$$?; \
	go run ./tools/ci/ci6report -out $(CI6_REPORT) < $(CI6_JSON) >/dev/null || status=1; \
	exit $$status

