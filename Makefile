# Tag of the nava/* images compose.yaml builds; `make ci-isolated` sets one per run.
export NAVA_IMAGE_TAG ?= dev
TEMPORAL_SERVER_IMAGE ?= nava/temporal-server:$(NAVA_IMAGE_TAG)

.PHONY: temporal-server-test temporal-server-image temporal-server-dev-certs temporal-server-run dev-ca operator-tunnel

## Unit, policy and end-to-end tests of the custom Temporal server module.
temporal-server-test:
	cd temporal-server && go vet ./...
	go run ./tools/ci/suiteran -- go -C temporal-server test -json ./...

## Build the custom Temporal server image (ADR-001 D22, task T27a).
temporal-server-image:
	docker build -t $(TEMPORAL_SERVER_IMAGE) temporal-server

## Write throwaway development certificates to temporal-server/.devcerts.
temporal-server-dev-certs:
	cd temporal-server && go run ./cmd/devcerts -out .devcerts

## Run the image with the dev config and dev certificates (foreground).
temporal-server-run: temporal-server-dev-certs
	docker run --rm --name nava-temporal-dev -p 7233:7233 -p 9090:9090 \
		-v "$(CURDIR)/temporal-server/.devcerts:/etc/temporal/certs:ro" $(TEMPORAL_SERVER_IMAGE)
# CI entry point (T05). The GitHub workflows only call targets from this file,
# so `make ci` runs the same gates on a laptop (Linux, macOS, or Windows with
# Git Bash and Docker). Taskfile.yml mirrors build, test and lint for local
# use; `make check-taskfile` fails when they drift.
#
# Gates, in order (each stage stops the run when it fails):
#   ci-lint         gofmt, go vet, golangci-lint (+ fixture), check-taskfile,
#                   check-pins, check-openapi, govulncheck (both modules, once a
#                   day per dependency set). No containers.
#   ci-unit         go test -race ./... (one at a time per host), temporal-server
#                   unit tests. No containers.
#   ci-integration  -tags integration against a throwaway PostgreSQL (Docker),
#                   then the isolation suite (make isolation), then the
#                   ci-isolated --down tag check (test-ci-isolated-down).
#   ci-stack        image build, compose up, GET /healthz smoke, stack tests.

GOLANGCI_LINT ?= golangci-lint
# docs/pins.md, "Toolchain and CI tools".
GOLANGCI_LINT_VERSION := 2.14.0
GOVULNCHECK_VERSION := v1.9.0

.PHONY: build run test test-integration test-integration-cleanup reap-integration isolation lint generate check-openapi
.PHONY: ci ci-isolated ci-changed ci-stages ci-isolated-down test-ci-isolated-down ci-lint ci-unit ci-integration ci-stack tools check-tools fmt-check vet lint-fixture 	check-taskfile check-pins vulncheck vulncheck-ci vulncheck-temporal-server tools-govulncheck review-paths dev-ca operator-tunnel stack-test-sdk stack-test-identity stack-test-onboard identity-attacks identity-attacks-fixture stack-idp-realms stack-test-worker 	image image-ci stack-up stack-down stack-logs stack-internal stack-internal-beside stack-main ci-stack-run stack-tests smoke

build:
	go build -o bin/nava-core ./cmd/nava-core

# Loads .env when present (copy .env.example). Unlike task run, values in .env
# override variables already set in the shell.
run:
	@set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/nava-core

# The navalint analyzers are their own module (tools/navalint), so ./...
# does not reach them.
# -trimpath keeps the checkout path out of the build and test cache keys, so a
# new checkout reuses the cached results (TICK-153 O5). Not for navalint: it
# makes its analysistest runs slower. racelock: one go test -race at a time on
# the host, across every checkout and agent (TICK-153 D3); NAVA_RACE_LOCK=off
# skips it.
# suiteran (TICK-230) fails a go test -json run in which a registered
# isolation suite (review-paths.yaml, cto_signoff.isolation_tests) ran zero
# tests; its -tags are the tags the step means to test with, so keep them
# beside go test's own.
test:
	go run ./tools/ci/racelock -- go run ./tools/ci/suiteran -- go test -json -race -trimpath ./...
	go run ./tools/ci/racelock -- go -C tools/navalint test -race ./...

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

# The tenant isolation suite (T11, ADR-001 CI-2): its own throwaway
# PostgreSQL and a PgBouncer with the real auth_query, tenants A and B seeded,
# every runtime role against every tenant table plus the registered paths
# (internal/tenancy/isolation). Writes ISOLATION_REPORT (default
# isolation-report.md). Needs Docker.
# TODO(TODO-008): the integration job does not upload isolation-report.md as
# an artifact yet; the workflow step needs a push with `workflow` scope.
isolation:
	sh scripts/ci/isolation.sh $(ARGS)

# golangci-lint, then the navalint analyzers (tools/navalint, its own module)
# as go vet's tool, then the ADR-002-E1 nolint allow-list gate.
lint:
	$(GOLANGCI_LINT) run ./...
	sh scripts/ci/navalint.sh
	sh scripts/ci/nolint-allowlist.sh

# Regenerates api/api.gen.go from api/openapi.yaml (oapi-codegen v2.8.0, pinned
# by the go.mod tool directive).
generate:
	go generate ./...

# Additive-only rule for /api/v1: fails when api/openapi.yaml removes or
# narrows anything the base branch (BASE, default origin/main) offers. A
# missing BASE ref fails: a shallow clone must not pass the check by default.
BASE ?= origin/main
check-openapi:
	@git rev-parse -q --verify $(BASE) >/dev/null || { echo "check-openapi: $(BASE) not found; fetch it first (git fetch origin main)"; exit 1; }
	@if git cat-file -e $(BASE):api/openapi.yaml 2>/dev/null; then 		git show $(BASE):api/openapi.yaml > $${TMPDIR:-/tmp}/openapi.base.yaml && 		go run ./cmd/openapi-compat $${TMPDIR:-/tmp}/openapi.base.yaml api/openapi.yaml; 	else echo "no api/openapi.yaml on $(BASE): nothing to compare"; fi

ci: ci-lint ci-unit ci-integration ci-stack

# The four stages on a stack of its own, so several PRs run at once on one
# Docker host (TICK-147, docs/quality/ci-pipeline.md "Parallel runs"): compose
# project nava-<ID>, image tag <ID>, host ports derived from ID. Removes its
# own containers, volumes, networks and image tags on exit, nothing shared.
ci-isolated:
	@[ -n "$(ID)" ] || { echo "usage: make ci-isolated ID=<pr-or-sha>"; exit 2; }
	sh scripts/ci/ci-isolated.sh $(ID)

# ci-isolated with only the stages the change against BASE needs
# (tools/ci/cichanged, TICK-153 O7): docs only is ci-lint, a package no
# integration test, image or stack test depends on skips those stages, and a
# file no rule knows runs all four.
ci-changed:
	@[ -n "$(ID)" ] || { echo "usage: make ci-changed ID=<pr-or-sha>"; exit 2; }
	@stages=$$(go run ./tools/ci/cichanged -v -base $(BASE)) && echo "ci-changed: $$stages" && \
		STAGES="$$stages" sh scripts/ci/ci-isolated.sh $(ID)

# Prints the stages ci-changed would run; the workflow's changes job uses it to
# skip the jobs a pull request does not need. CI_ALL=1 prints every stage.
ci-stages:
	@sh scripts/ci/ci-stages.sh

# Tears down what a killed ci-isolated run left behind (its own project and tags).
ci-isolated-down:
	@[ -n "$(ID)" ] || { echo "usage: make ci-isolated-down ID=<pr-or-sha>"; exit 2; }
	sh scripts/ci/ci-isolated.sh --down $(ID)

# ci-isolated-down untags every nava/*:<id> tag and nothing else (TICK-211):
# throwaway tags on an empty imported image, a fresh id per run. Needs Docker.
test-ci-isolated-down:
	sh scripts/ci/test-ci-isolated-down.sh

ci-lint: check-tools fmt-check vet lint lint-fixture check-taskfile check-pins check-openapi vulncheck-ci

ci-unit: test temporal-server-test

ci-integration: test-integration test-ci-isolated-down test-integration-cleanup

# Builds the images, brings the stack up, runs every stack test, then tears
# it down (logs first on failure). The workflow runs the same targets one step
# at a time.
# healthy gates the rest. The read-only checks and the two Go suites then run
# at once (TICK-153 O10): none changes what another reads, and each matches
# only lines it caused. The rest run one at a time, in this order: dependencies
# stops Keycloak, trace counts every trace the Collector saw, idempotent
# restarts the stack, and dbroles changes roles. worker polls nava-tq, the
# queue the sdk suite's workers poll, so it runs after that suite.
STACK_TESTS_PARALLEL := smoke pgbouncer temporal temporal_handshake temporal_isolation temporal_custody temporal_operator_reach temporal_health hba sdk identity
STACK_TESTS_SERIAL := worker dependencies trace logs redact idempotent dbroles
STACK_TESTS := healthy $(STACK_TESTS_PARALLEL) $(STACK_TESTS_SERIAL)
stack-tests:
	$(MAKE) --no-print-directory -j1 stack-test-healthy
	$(MAKE) --no-print-directory -j $(words $(STACK_TESTS_PARALLEL)) --output-sync=target $(addprefix stack-test-,$(STACK_TESTS_PARALLEL))
	$(MAKE) --no-print-directory -j1 $(addprefix stack-test-,$(STACK_TESTS_SERIAL))
ci-stack: $(if $(IMAGE_CACHE),image-ci,image)
	@$(STACK_LOCK) $(MAKE) --no-print-directory ci-stack-run

# The main stack and its tests, and the internal overlay on a project of its
# own, after it or beside it (STACK_INTERNAL, TICK-153 O8).
# Each tears its project down; the main one prints its logs first on failure.
# The certificates come first: a bind mount of a missing file would make
# Docker create a directory in its place.
# This throwaway run, which ends in down -v, adds compose.ci.yaml, whose
# no-durability Postgres flags are now also in compose.yaml (TICK-166), so the
# dev project has them too; the overlay is kept as a duplicate.
ci-stack-run: export COMPOSE_FILE := compose.yaml:deploy/compose/compose.test.yaml:deploy/compose/compose.ci.yaml
ci-stack-run: dev-ca
	@$(MAKE) --no-print-directory $(if $(filter after,$(STACK_INTERNAL)),-j1,-j 2) stack-main stack-internal-beside

stack-main:
	@status=0; $(MAKE) --no-print-directory -j1 stack-up stack-tests || status=$$?; if [ $$status -ne 0 ]; then $(MAKE) --no-print-directory stack-logs; fi; $(MAKE) --no-print-directory stack-down; exit $$status

# The internal overlay needs a fresh stack. Host ports 0: Docker picks free
# ones, so it cannot collide with the main stack's.
INTERNAL_PROJECT = $(COMPOSE_PROJECT)-int
INTERNAL_LOG = $(or $(TMPDIR),/tmp)/$(INTERNAL_PROJECT).log
stack-internal-beside:
	@echo "stack-internal-beside: $(INTERNAL_PROJECT), log in $(INTERNAL_LOG), printed when it ends"; status=0; COMPOSE_PROJECT_NAME=$(INTERNAL_PROJECT) NAVA_CORE_PORT=0 KEYCLOAK_PORT=0 sh deploy/compose/test/stack-test.sh internal >"$(INTERNAL_LOG)" 2>&1 || status=$$?; if [ $$status -ne 0 ]; then COMPOSE_PROJECT_NAME=$(INTERNAL_PROJECT) docker compose $(STACK_FILES) logs --no-color --tail 200 >>"$(INTERNAL_LOG)" 2>&1; fi; COMPOSE_PROJECT_NAME=$(INTERNAL_PROJECT) docker compose $(STACK_FILES) down -v --remove-orphans >>"$(INTERNAL_LOG)" 2>&1; echo "=== stack-internal-beside log"; cat "$(INTERNAL_LOG)"; echo "stack-internal-beside: exit $$status"; exit $$status

# At most STACK_SLOTS compose stacks on the host at once, the rest queued
# (TICK-153 D4); a ci-stack run holds one, or two with STACK_INTERNAL=beside.
# N = 7 from the measured memory, docs/quality/ci-optimizations.md.
# NAVA_STACK_SLOTS=off skips the queue.
# STACK_INTERNAL=after (the default) runs the internal overlay after the main
# stack, on one slot; =beside runs it at the same time, on two. Measured on
# this host, after is faster: two Keycloaks starting at once cost more than
# the overlap saves (669 s against 698 s, docs/quality/ci-optimizations.md).
STACK_SLOTS ?= 7
STACK_INTERNAL ?= after
STACK_LOCK = go run ./tools/ci/racelock -port 47620 -slots $(STACK_SLOTS) -take $(if $(filter after,$(STACK_INTERNAL)),1,2) -env NAVA_STACK_SLOTS -what "compose stack" --

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

# Prints "security review required: yes/no" and "CTO sign-off required:
# yes/no" for the change against BASE (PLAN-001 D-P10).
review-paths:
	go run ./tools/ci/reviewpaths -base $(BASE)

# --- Compose stack ---------------------------------------------------------
# The trace stack test needs every ordinary trace kept.
export OTEL_TAIL_SAMPLE_PERCENT ?= 100
# T28: the stack tests run with the operator profile (Temporal UI) and the test
# overlay that publishes the Temporal gateway on loopback for the SDK test.
export COMPOSE_PROFILES ?= operator
export COMPOSE_FILE ?= compose.yaml:deploy/compose/compose.test.yaml
# Windows Docker splits COMPOSE_FILE on ";" unless told otherwise.
export COMPOSE_PATH_SEPARATOR ?= :
STACK_FILES := -f compose.yaml -f deploy/compose/compose.internal.yaml
# The compose project: `name: nava` in compose.yaml unless COMPOSE_PROJECT_NAME
# names another.
COMPOSE_PROJECT := $(or $(COMPOSE_PROJECT_NAME),nava)

image:
	docker compose build

# CI image build: buildx bake over compose.yaml while the registry images are
# pulled alongside. IMAGE_CACHE=gha keeps the layer cache in the GitHub Actions
# cache (needs a docker-container builder and the Actions runtime env), so an
# unchanged image is a cache hit and a source change rebuilds one layer.
# db-roles and migrate share nava/nava-db; only db-roles writes its cache.
# IMAGE_CACHE=local keeps the layers in the local builder's own BuildKit cache,
# which concurrent builds share safely (a type=local cache directory is not
# safe for two writers at once), and pulls only the registry images that are
# missing. With IMAGE_CACHE set, ci-stack builds with image-ci.
BAKE_CACHED := db-roles keycloak nava-core nava-groupsync nava-onboard nava-purge nava-relay nava-worker otel-collector temporal-server
BAKE_CACHE_GHA := $(foreach t,$(BAKE_CACHED),--set $(t).cache-from=type=gha,scope=$(t) --set $(t).cache-to=type=gha,scope=$(t),mode=max) --set migrate.cache-from=type=gha,scope=db-roles
BAKE_CACHE := $(if $(filter gha,$(IMAGE_CACHE)),$(BAKE_CACHE_GHA))
PULL_POLICY := $(if $(filter local,$(IMAGE_CACHE)),--policy missing)
image-ci:
	@docker compose pull --quiet --ignore-buildable --ignore-pull-failures $(PULL_POLICY) & pull=$$!; \
	docker buildx bake -f compose.yaml --load $(BAKE_CACHE); status=$$?; \
	wait $$pull; exit $$status

stack-up: dev-ca
	docker compose up -d

# stack-test-<name>: one case of deploy/compose/test/stack-test.sh.
stack-test-%:
	sh deploy/compose/test/stack-test.sh $*

# The Temporal stack suite over mTLS: SDK clients dial and a worker runs (T28),
# CI-H, and the CI-6 certificate, network and Authorizer cases (T32a). Writes
# the CI-6 report to CI6_REPORT and fails when a CI-6 case did not pass.
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

# The nava-worker service (T31b, TICK-159): ready with /readyz green, /metrics
# and the probes on its unpublished admin listener without httpobs, and a
# workflow's workflow_ref activity run in the service through the guard, which
# /metrics counts.
stack-test-worker:
	go test -tags workerstack -count=1 ./internal/workerstack

# Creates the two test IdP realms (idp-tenant-a, idp-tenant-b) on the running
# Keycloak, fresh, with the code the identity tests start with (TICK-166). A
# plain `docker compose up` imports only `nava`; a manual S2 step needs these.
stack-idp-realms:
	go test -tags identitystack -count=1 -run ^TestIdPRealmsExist$$ ./internal/identitystack

# The realm `nava` of the running Keycloak (T14, ADR-001 A.1): tenant and idp
# claims, token lifespan, operator role custody, realm export round trip.
# TestCI5 needs a platform database: make identity-attacks runs it. The
# CI-5.06 negative fixture needs Keycloak 26.1.2: make identity-attacks-fixture.
# It also runs the onboarding tests below (stack-test-onboard).
# TODO(TODO-012): the GitHub workflow's stack job has no step for
# stack-test-onboard yet (the step needs a push with `workflow` scope), so this
# target runs those tests too; once the step exists, add -skip '$(ONBOARD_TESTS)'.
stack-test-identity:
	rm -f $(IDENTITY_STREAM)
	go test -tags identitystack -count=1 -skip '^TestCI5$$' -json ./internal/identitystack > $(IDENTITY_STREAM) || { cat $(IDENTITY_STREAM); exit 1; }

# The CI-5 adversarial identity suite (T19, ADR-001 Annex B CI-5) against the
# running Keycloak, with a throwaway PostgreSQL for the platform database the
# resolver and one real group sync use. Writes the CI-5 report to CI5_REPORT
# and fails when a CI-5 case did not pass. Needs the stack's Keycloak.
CI5_REPORT ?= ci5-report.md
IDENTITY_STREAM ?= identity-stream.json
identity-attacks:
	CI5_REPORT=$(CI5_REPORT) IDENTITY_STREAM=$(IDENTITY_STREAM) sh scripts/ci/identity-attacks.sh

# The CI-5.06 negative fixture: the same kc.org tamper must SUCCEED on a
# throwaway Keycloak 26.1.2 (the known-broken pin), so CI-5.06 passing on the
# pinned Keycloak means something. Needs Docker, not the stack. Opt-in.
identity-attacks-fixture:
	sh scripts/ci/identity-attacks.sh fixture

# Onboarding end to end (T21b, TICK-234): nava-onboard onboards a tenant, the
# invited admin signs in through the tenant's IdP, one nava-groupsync cycle
# runs, and nava-core's GET /api/v1/me shows the new tenant and no other; the
# nava-onboard/nava-core separation; kill before the binding, then retry,
# against the real nava_onboard PGStore. Needs the stack up (make stack-up).
ONBOARD_TESTS := ^(TestOnboard(edAdminSignsInAndMeShowsTenant|DeploymentSeparation|KillBeforeBindThenRetryLiveStack|DomainClaimedIsConflict)|TestInvitedToTenantANotLinkedThroughTenantBIdP|TestSquattedInviteeIsNotPlacedInTheOtherTenant|TestUninvitedLocalUserNotLinked|TestInviteIsSingleUse|TestExpiredInviteNotLinked|TestInvitedUserInTwoTenantsNotLinked|TestInvitedUserHasNoLocalLogin|TestDisabledInviteeNotLinked|TestUnverifiedUpstreamEmailNotPlaced|TestIdPEmailVerifiedAttributeNotUserEditable|TestUserCannotRenameEmailToPendingInvite|TestAccountAPIClosedToPlainBrokeredUser|TestRealmFileEmailVerifiedMapper)$$
stack-test-onboard:
	go test -tags identitystack -count=1 -run '$(ONBOARD_TESTS)' ./internal/identitystack

# The compose smoke: GET /healthz from inside the compose network.
smoke: stack-test-smoke

# The internal overlay needs a fresh stack.
stack-internal:
	docker compose down -v
	sh deploy/compose/test/stack-test.sh internal

stack-logs:
	docker compose $(STACK_FILES) logs --no-color --tail 200

stack-down:
	docker compose $(STACK_FILES) down -v

CERT_DIR ?= deploy/compose/certs

## Dev installation CA for the compose stack (ADR-001 D22, T28): the installation
## (frontend client) CA, a separate internode CA, the frontend and internode
## certificates, the five platform certificates and nava-unknown (a CN that is
## no identity, for the CI-6 suite; never mounted), written to
## deploy/compose/certs/ (git-ignored). Keeps existing certificates; FORCE=1 reissues.
dev-ca:
	@if [ -f $(CERT_DIR)/ca.pem ] && [ -z "$(FORCE)" ]; then echo "$(CERT_DIR) already holds a dev CA (FORCE=1 to reissue)"; 	else cd temporal-server && go run ./cmd/devcerts -installation -unlisted -out ../$(CERT_DIR); fi

## Explicit operator tunnel to the Temporal UI (profile operator): forwards
## 127.0.0.1:$(UI_PORT) to the UI until interrupted. The UI publishes no port.
# TODO(TODO-006): while the tunnel is open, any container on a non-internal
# network reaches the UI through host.docker.internal and acts as nava-operator.
UI_PORT ?= 18233
operator-tunnel:
	docker run --rm --name $(COMPOSE_PROJECT)-operator-tunnel -p 127.0.0.1:$(UI_PORT):8080 		--network $$(docker network ls -q --filter label=com.docker.compose.project=$(COMPOSE_PROJECT) --filter label=com.docker.compose.network=operator) 		--entrypoint /usr/local/bin/temporal-server $(TEMPORAL_SERVER_IMAGE) proxy -listen :8080 -to temporal-ui:8080
