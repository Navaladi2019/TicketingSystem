.PHONY: lint check-generate check-guide check-ci-steps check-openapi test-check-openapi-base tools check-tools fmt-check vet vet-tags lint-fixture check-taskfile check-pins vulncheck vulncheck-ci vulncheck-temporal-server tools-govulncheck check-golangci-lint

ci-lint: check-tools fmt-check vet vet-tags lint lint-fixture check-taskfile check-pins check-generate check-guide check-ci-steps check-openapi test-check-openapi-base test-handbook test-handbook-coverage handbook-build vulncheck-ci

# golangci-lint, then the navalint analyzers (tools/navalint, its own module)
# as go vet's tool, then workflowcheck on internal/workflow (tools/workflowcheck,
# its own module, D-3-23), then the ADR-002-E1 nolint allow-list gate, then the decision-log index check,
# then the Competitive Gate check on docs/adr (tools/adrgate, CI-1..CI-4),
# then the duplicate-id check on the shared tables (tools/dupids), then the
# upgrade-compatibility baseline (M-X2, TICK-881): fails when an item in the
# base's internal/configsurface/testdata/inventory.json ($BASE, default
# origin/main) is missing from HEAD's.
lint:
	$(GOLANGCI_LINT) run ./...
	$(GOLANGCI_LINT) run --build-tags "integration perf" ./...
	$(GOLANGCI_LINT) run --build-tags "identitystack acceptance ci5" ./...
	$(GOLANGCI_LINT) run --build-tags "temporalstack demooutbox" ./...
	$(GOLANGCI_LINT) run --build-tags "workerstack interpcrash" ./...
	$(GOLANGCI_LINT) run --build-tags "stacktag accept" ./...
	$(GOLANGCI_LINT) run --build-tags "recordhistories" ./...
	$(GOLANGCI_LINT) run --build-tags "interptest" ./...
	sh scripts/ci/navalint.sh
	sh scripts/ci/workflowcheck.sh
	sh scripts/ci/nolint-allowlist.sh
	go run ./tools/docindex -check
	go run ./tools/adrgate
	go run ./tools/dupids
	go run ./internal/configsurface/cmd/compat

# Fails when api/openapi.bundled.yaml or api/api.gen.go is not what api/openapi.yaml
# + api/paths/*.yaml generate (TICK-297, TICK-294): any difference or untracked file
# under api/ after regenerating is drift. Run make generate and commit the result.
check-generate:
	sh scripts/ci/openapi-generated.sh

# Fails when docs/acceptance/guide/ (pages, Bruno collection with
# lib/nava.js REGISTERED, kit zip) is not what build/build.py writes from
# api/errors.yaml, the bundled spec and the build sources (TICK-391).
check-guide:
	sh scripts/ci/guide-generated.sh

# Fails when .github/workflows/ci.yml stops running a step listed in
# tools/ci/cisteps/required.yaml (TICK-297): a suite or report upload removed,
# disabled or allowed to fail.
# ci.yml against required.yaml. govulncheck.yml must equal, comment lines
# aside, its checked-in copy tools/ci/cisteps/govulncheck.copy.yml (TICK-1149,
# ADR-001-E4 contract rule 9): edit the workflow and the copy together. No
# step of govulncheck.yml may run make: a run: line with make as a word fails
# (ADR-001-A7 rule 9, TICK-1187), so no makefile can change what the scan
# jobs scan, and this check reads no makefile. A deliberate edit of both
# files that adds no make passes; review catches that (RS3).
check-ci-steps:
	go run ./tools/ci/cisteps
	go run ./tools/ci/cisteps -workflow .github/workflows/govulncheck.yml -copy tools/ci/cisteps/govulncheck.copy.yml

# Additive-only rule for /api/v1: fails when api/openapi.yaml removes or
# narrows anything the base branch (BASE, default origin/main) offers. A
# missing BASE ref fails: a shallow clone must not pass the check by default.
check-openapi:
	@git rev-parse -q --verify $(BASE) >/dev/null || { echo "check-openapi: $(BASE) not found; fetch it first (git fetch origin main)"; exit 1; }
	@sh scripts/ci/check-openapi.sh $(BASE)

# check-openapi.sh on a pull_request run compares against the merge commit's
# first parent, so a main that moved on mid-run cannot fail the PR; a real
# removal still fails (TICK-1179). A throwaway git repo, no network.
test-check-openapi-base:
	sh scripts/ci/test-check-openapi-base.sh

# Installs the pinned CI tools into $$(go env GOPATH)/bin.
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)
	$(MAKE) --no-print-directory tools-govulncheck

# govulncheck alone, installed for ci-lint (vulncheck-ci) and check-tools. A
# version bump edits GOVULNCHECK_VERSION in Makefile, the govulncheck row of
# docs/pins.md, both govulncheck@ lines of the govulncheck workflow and its
# copy, and Taskfile.yml's vulncheck task; check-pins (V-6) fails on the first
# three disagreeing (ADR-001-A7 A7.2).
tools-govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

# A local lint run with another golangci-lint version is not the CI run.
check-tools: check-golangci-lint
	@command -v govulncheck >/dev/null || { echo "govulncheck is missing; run make tools"; exit 1; }

fmt-check:
	@out=$$(git ls-files '*.go' | grep -v '^docs/' | xargs gofmt -l); 	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...

# go vet once per build-tag set the Go files use (TICK-320): make vet builds
# untagged files only, so a broken //go:build identitystack && acceptance file
# went unnoticed on main (TICK-286 finding 4). The sets come from the
# //go:build lines of every module (tools/ci/buildtags), so a new tag or
# module needs no edit here.
vet-tags:
	go run ./tools/ci/buildtags

lint-fixture:
	sh scripts/ci/lint-fixture.sh

check-taskfile:
	go run ./tools/ci/checktaskfile

check-pins:
	go run ./tools/ci/checkpins

# Local use; the workflow runs the same command directly (ADR-001-A7), with
# its other Go settings pinned in the step's env: (CI is the gate, R-A7-3).
# Local targets need the installed Go to be at least the module's go line
# (go1.27); go run otherwise picks govulncheck's own toolchain and fails to
# load our packages.
# GOWORK=off (ADR-001-E4 contract rule 7): a go.work beside the module would
# change the scanned versions without changing go.mod or go.sum.
vulncheck:
	GOWORK=off go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

# ci-lint's govulncheck: both modules at once, each when its go.mod or go.sum
# has not passed a scan today on this host (TICK-153 D2, ADR-001-A1 R7).
# VULNCHECK=all scans both regardless.
vulncheck-ci:
	sh scripts/ci/vulncheck.sh

# Local use; the workflow runs the same command directly (ADR-001-A7, job
# vulncheck-temporal-server, whose result the pull request's vulncheck check
# reports). Zero findings, with no exception or allowlist (ADR-001 A1
# Decision 3).
# Local targets need the installed Go to be at least the module's go line
# (go1.27); go run otherwise picks govulncheck's own toolchain and fails to
# load our packages.
vulncheck-temporal-server:
	cd temporal-server && GOWORK=off go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...


# golangci-lint alone at the pinned version (docs/pins.md): check-tools, and
# lint-temporal-server, whose job installs no govulncheck before it lints.
check-golangci-lint:
	@$(GOLANGCI_LINT) version | grep -q "version $(GOLANGCI_LINT_VERSION) " || { echo "golangci-lint is not $(GOLANGCI_LINT_VERSION) (docs/pins.md); run make tools"; exit 1; }
