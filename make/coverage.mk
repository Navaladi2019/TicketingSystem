.PHONY: check-coverage

# The coverage gate (TICK-518): merges the unit profile (make test,
# coverage.out) and the integration profile against a real PostgreSQL (make
# test-integration, coverage-integration.out), drops the files listed in
# tools/ci/coverage-exclude.txt (each with its reason), and fails when the
# total is below COVERAGE_MIN percent, listing the 20 worst-covered packages.
# It writes the merged profile (coverage-total.out) and coverage.html even
# when it fails. COVERAGE_REPORT_ONLY=1 prints the gap and exits 0: the CI
# step runs that way until the total reaches the minimum (TICK-518 rollout).
# Run make test and make test-integration first; on a laptop without Docker,
# COVERAGE_PROFILES=coverage.out checks the unit half alone.
COVERAGE_MIN ?= 95
COVERAGE_PROFILES ?= coverage.out coverage-integration.out
COVERAGE_REPORT_ONLY ?=
check-coverage:
	@status=0; \
	go run ./tools/ci/coverage check -min $(COVERAGE_MIN) -exclude tools/ci/coverage-exclude.txt \
		-o coverage-total.out $(if $(COVERAGE_REPORT_ONLY),-report-only -all) $(COVERAGE_PROFILES) || status=$$?; \
	if [ -f coverage-total.out ]; then go tool cover -html=coverage-total.out -o coverage.html || status=2; fi; \
	exit $$status
