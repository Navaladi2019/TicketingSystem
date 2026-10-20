TEMPORAL_SERVER_IMAGE ?= nava/temporal-server:dev

.PHONY: temporal-server-test temporal-server-image temporal-server-dev-certs temporal-server-run dev-ca operator-tunnel

## Unit, policy and end-to-end tests of the custom Temporal server module.
temporal-server-test:
	cd temporal-server && go vet ./... && go test ./...

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
#                   check-pins, check-openapi, govulncheck. No containers.
#   ci-unit         go test -race ./..., temporal-server unit tests. No containers.
#   ci-integration  -tags integration against a throwaway PostgreSQL (Docker).
#   ci-stack        image build, compose up, GET /healthz smoke, stack tests.

GOLANGCI_LINT ?= golangci-lint
# docs/pins.md, "Toolchain and CI tools".
GOLANGCI_LINT_VERSION := 2.14.0
GOVULNCHECK_VERSION := v1.9.0

.PHONY: build run test test-integration lint generate check-openapi
.PHONY: ci ci-lint ci-unit ci-integration ci-stack tools check-tools fmt-check vet lint-fixture 	check-taskfile check-pins vulncheck vulncheck-temporal-server review-paths dev-ca operator-tunnel stack-test-sdk 	image image-ci stack-up stack-down stack-logs stack-internal smoke

build:
	go build -o bin/nava-core ./cmd/nava-core

# Loads .env when present (copy .env.example). Unlike task run, values in .env
# override variables already set in the shell.
run:
	@set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/nava-core

test:
	go test -race ./...

# Every package with `//go:build integration` tests, against a throwaway
# postgres container the script starts and removes (needs Docker). Pass
# ARGS=... to forward flags to go test.
test-integration:
	sh scripts/ci/test-integration.sh $(ARGS)

lint:
	$(GOLANGCI_LINT) run ./...

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

ci-lint: check-tools fmt-check vet lint lint-fixture check-taskfile check-pins check-openapi vulncheck

ci-unit: test temporal-server-test

ci-integration: test-integration

# Builds the images, brings the stack up, runs every stack test, then tears
# it down (logs first on failure). The workflow runs the same targets one step
# at a time.
STACK_TESTS := healthy smoke pgbouncer temporal temporal_handshake temporal_isolation temporal_custody temporal_operator_reach temporal_health sdk dependencies trace redact idempotent dbroles hba
ci-stack: image
	@status=0; $(MAKE) --no-print-directory stack-up $(addprefix stack-test-,$(STACK_TESTS)) stack-internal || status=$$?; 	if [ $$status -ne 0 ]; then $(MAKE) --no-print-directory stack-logs; fi; 	$(MAKE) --no-print-directory stack-down; exit $$status

# Installs the pinned CI tools into $$(go env GOPATH)/bin.
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)
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

# Not in ci yet: the custom Temporal server module has reachable advisories
# (see docs/pins.md); its owner fixes them, then this joins ci-lint.
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

image:
	docker compose build

# CI image build: buildx bake over compose.yaml while the registry images are
# pulled alongside. IMAGE_CACHE=gha keeps the layer cache in the GitHub Actions
# cache (needs a docker-container builder and the Actions runtime env), so an
# unchanged image is a cache hit and a source change rebuilds one layer.
# db-roles and migrate share nava/nava-db; only db-roles writes its cache.
BAKE_CACHED := db-roles nava-core otel-collector temporal-server
BAKE_CACHE_GHA := $(foreach t,$(BAKE_CACHED),--set $(t).cache-from=type=gha,scope=$(t) --set $(t).cache-to=type=gha,scope=$(t),mode=max) --set migrate.cache-from=type=gha,scope=db-roles
BAKE_CACHE := $(if $(filter gha,$(IMAGE_CACHE)),$(BAKE_CACHE_GHA))
image-ci:
	@docker compose pull --quiet --ignore-buildable --ignore-pull-failures & pull=$$!; \
	docker buildx bake -f compose.yaml --load $(BAKE_CACHE); status=$$?; \
	wait $$pull; exit $$status

stack-up: dev-ca
	docker compose up -d

# stack-test-<name>: one case of deploy/compose/test/stack-test.sh.
stack-test-%:
	sh deploy/compose/test/stack-test.sh $*

# SDK clients dial with their certificates and a worker runs (T28).
stack-test-sdk:
	go test -tags temporalstack -count=1 ./internal/temporalstack

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
## certificates, and the five platform certificates, written to
## deploy/compose/certs/ (git-ignored). Keeps existing certificates; FORCE=1 reissues.
dev-ca:
	@if [ -f $(CERT_DIR)/ca.pem ] && [ -z "$(FORCE)" ]; then echo "$(CERT_DIR) already holds a dev CA (FORCE=1 to reissue)"; 	else cd temporal-server && go run ./cmd/devcerts -installation -out ../$(CERT_DIR); fi

## Explicit operator tunnel to the Temporal UI (profile operator): forwards
## 127.0.0.1:$(UI_PORT) to the UI until interrupted. The UI publishes no port.
# TODO(TODO-006): while the tunnel is open, any container on a non-internal
# network reaches the UI through host.docker.internal and acts as nava-operator.
UI_PORT ?= 18233
operator-tunnel:
	docker run --rm --name nava-operator-tunnel -p 127.0.0.1:$(UI_PORT):8080 		--network $$(docker network ls -q --filter label=com.docker.compose.project=nava --filter label=com.docker.compose.network=operator) 		--entrypoint /usr/local/bin/temporal-server nava/temporal-server:dev proxy -listen :8080 -to temporal-ui:8080
