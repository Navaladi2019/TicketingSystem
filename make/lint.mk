.PHONY: lint check-generate check-ci-steps check-openapi tools check-tools fmt-check vet vet-tags lint-fixture check-taskfile check-pins vulncheck vulncheck-ci vulncheck-temporal-server tools-govulncheck

ci-lint: check-tools fmt-check vet vet-tags lint lint-fixture check-taskfile check-pins check-generate check-ci-steps check-openapi vulncheck-ci

# golangci-lint, then the navalint analyzers (tools/navalint, its own module)
# as go vet's tool, then workflowcheck on internal/workflow (tools/workflowcheck,
# its own module, D-3-23), then the ADR-002-E1 nolint allow-list gate, then the decision-log index check,
# then the duplicate-id check on the shared tables (tools/dupids).
lint:
	$(GOLANGCI_LINT) run ./...
	sh scripts/ci/navalint.sh
	sh scripts/ci/workflowcheck.sh
	sh scripts/ci/nolint-allowlist.sh
	go run ./tools/docindex -check
	go run ./tools/dupids

# Fails when api/openapi.bundled.yaml or api/api.gen.go is not what api/openapi.yaml
# + api/paths/*.yaml generate (TICK-297, TICK-294): any difference or untracked file
# under api/ after regenerating is drift. Run make generate and commit the result.
check-generate:
	sh scripts/ci/openapi-generated.sh

# Fails when .github/workflows/ci.yml stops running a step listed in
# tools/ci/cisteps/required.yaml (TICK-297): a suite or report upload removed,
# disabled or allowed to fail.
check-ci-steps:
	go run ./tools/ci/cisteps

# Additive-only rule for /api/v1: fails when api/openapi.yaml removes or
# narrows anything the base branch (BASE, default origin/main) offers. A
# missing BASE ref fails: a shallow clone must not pass the check by default.
check-openapi:
	@git rev-parse -q --verify $(BASE) >/dev/null || { echo "check-openapi: $(BASE) not found; fetch it first (git fetch origin main)"; exit 1; }
	@sh scripts/ci/check-openapi.sh $(BASE)

# Installs the pinned CI tools into $$(go env GOPATH)/bin.
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)
	$(MAKE) --no-print-directory tools-govulncheck

# govulncheck alone, for the temporal-server CI job.
tools-govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

# A local lint run with another golangci-lint version is not the CI run.
check-tools:
	@$(GOLANGCI_LINT) version | grep -q "version $(GOLANGCI_LINT_VERSION) " || { echo "golangci-lint is not $(GOLANGCI_LINT_VERSION) (docs/pins.md); run make tools"; exit 1; }
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

vulncheck:
	govulncheck ./...

# ci-lint's govulncheck: both modules at once, each when its go.mod or go.sum
# has not passed a scan today on this host (TICK-153 D2, ADR-001-A1 R7).
# VULNCHECK=all scans both regardless.
vulncheck-ci:
	sh scripts/ci/vulncheck.sh

# Required in CI (temporal-server job and ci-lint): zero findings, with no
# exception or allowlist (ADR-001 A1 Decision 3).
vulncheck-temporal-server:
	cd temporal-server && govulncheck ./...

