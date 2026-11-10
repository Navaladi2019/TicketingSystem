.PHONY: ci-stack ci-stack-run stack-tests stack-main stack-internal-beside stack-internal stack-up stack-down stack-logs image image-ci ci-binaries smoke

# Builds the images, brings the stack up, runs every stack test, then tears
# it down (logs first on failure). The workflow runs `make stack-tests JOB=<job>`
# on each of its stack jobs.
# The tests are the files in deploy/compose/test/cases/, ordered and assigned to a CI job by the
# "# stack-case:" header of each (run-cases.sh); adding one needs no edit here or in ci.yml.
# healthy gates the rest. The read-only (parallel) cases then run at once under PARALLEL=1
# (TICK-153 O10), as ci-stack does; the CI steps set PARALLEL=1 too (TICK-628). The serial cases follow in
# order: dependencies stops Keycloak, trace counts every trace the Collector saw, idempotent
# restarts the stack, and dbroles changes roles. worker polls nava-tq, the queue the sdk
# suite's workers poll, so it sorts after that suite.
# JOB: a comma list of CI jobs (stack, stack-db, stack-internal), or all.
# AREAS (TICK-563): the stack-case areas to run, all (the default: every case),
# none, or a comma list of keycloak, pgbouncer and db; a case whose area= is not
# listed prints "SKIP <case> (area <x> not touched)". CI sets it from
# `cichanged -areas`; a case without area= always runs.
JOB ?= stack,stack-db
stack-tests:
	@sh deploy/compose/test/run-cases.sh run $(JOB)
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
	@status=0; $(MAKE) --no-print-directory -j1 stack-up stack-tests PARALLEL=1 || status=$$?; if [ $$status -ne 0 ]; then $(MAKE) --no-print-directory stack-logs; fi; $(MAKE) --no-print-directory stack-down; exit $$status

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
# The gha cache-to is best effort (ignore-error=true): concurrent runs that
# export the same scope can race on a blob (BlobNotFound, not_found), and a
# failed cache write must only cost build time, never fail the build.
# cache-to is set only with CACHE_WRITE=true, which ci.yml sets on main
# (TICK-400). Every other ref reads main's layers. A cache written by a task
# branch or a pull request can be read by no other ref, and those caches
# filled the repository's 10 GB and evicted main's.
BAKE_CACHED := db-roles keycloak nava-core nava-groupsync nava-onboard nava-purge nava-relay nava-worker otel-collector temporal-server
# Every compose.yaml service with a build: (TestBakeTargetsCoverCompose).
BAKE_TARGETS := $(BAKE_CACHED) migrate
# TEMPORAL_SERVER_PREBUILT=1 (make temporal-server-pull found the published
# image, TICK-641): bake every target but temporal-server, and read or write
# no cache for it. Unset: bake's default group, every target, as before.
BAKE_BUILT := $(if $(filter 1,$(TEMPORAL_SERVER_PREBUILT)),$(filter-out temporal-server,$(BAKE_CACHED)),$(BAKE_CACHED))
BAKE_ONLY := $(if $(filter 1,$(TEMPORAL_SERVER_PREBUILT)),$(filter-out temporal-server,$(BAKE_TARGETS)))
comma := ,
BAKE_CACHE_TO = $(if $(filter true,$(CACHE_WRITE)),--set $(1).cache-to=type=gha$(comma)scope=$(1)$(comma)mode=max$(comma)ignore-error=true)
BAKE_CACHE_GHA := $(foreach t,$(BAKE_BUILT),--set $(t).cache-from=type=gha,scope=$(t) $(call BAKE_CACHE_TO,$(t))) --set migrate.cache-from=type=gha,scope=db-roles
BAKE_CACHE := $(if $(filter gha,$(IMAGE_CACHE)),$(BAKE_CACHE_GHA))
PULL_POLICY := $(if $(filter local,$(IMAGE_CACHE)),--policy missing)
#
# IMAGE_THIN=1 (TICK-628): the Go services' images come from
# deploy/compose/Dockerfile.ci, one target per binary, over the binaries
# make ci-binaries built on the host into CI_BIN_DIR, so the build does no
# go build in Docker. Pull request stack runs set it when the change does not
# touch the image build (cichanged -images); main never does. The images keep
# the real Dockerfiles' final stage (TestDockerfileCIMatchesRealFinalStages);
# the risk that a pull request no longer builds the real build stage is
# AR-TICK-628-01. CI_THIN_SERVICES maps each compose service to its target;
# TestThinServicesCoverCompose fails when a Go service of compose.yaml is
# missing from it. The interptest stack worker (compose.test.yaml, profile
# interptest) is built from Dockerfile.ci's nava-worker-interptest target over
# CI_BIN_DIR/nava-worker-interptest (-tags interptest), and tagged directly:
# bake builds compose.yaml only. The interpreter cases start it without
# --build when NAVA_INTERPTEST_PREBUILT=1 (the stack job sets it with
# IMAGE_THIN=1), so a thin run does no go build in Docker at all.
CI_BIN_DIR := .ci-bin
CI_BINARIES := nava-core nava-db nava-groupsync nava-onboard nava-purge nava-relay nava-worker
CI_THIN_SERVICES := db-roles=nava-db migrate=nava-db nava-core=nava-core nava-groupsync=nava-groupsync nava-onboard=nava-onboard nava-purge=nava-purge nava-relay=nava-relay nava-worker=nava-worker
BAKE_THIN := $(foreach m,$(CI_THIN_SERVICES),--set $(word 1,$(subst =, ,$(m))).dockerfile=deploy/compose/Dockerfile.ci --set $(word 1,$(subst =, ,$(m))).target=$(word 2,$(subst =, ,$(m))))
image-ci:
	@$(if $(filter 1,$(IMAGE_THIN)),for b in $(CI_BINARIES) nava-worker-interptest; do [ -x $(CI_BIN_DIR)/$$b ] || { echo "image-ci: IMAGE_THIN=1 needs $(CI_BIN_DIR)/$$b: run make ci-binaries first" >&2; exit 2; }; done; echo "image-ci: thin CI images from deploy/compose/Dockerfile.ci (IMAGE_THIN=1)";) \
	docker compose pull --quiet --ignore-buildable --ignore-pull-failures $(PULL_POLICY) & pull=$$!; \
	docker buildx bake -f compose.yaml --load $(BAKE_CACHE) $(if $(filter 1,$(IMAGE_THIN)),$(BAKE_THIN)) $(BAKE_ONLY); status=$$?; \
	$(if $(filter 1,$(IMAGE_THIN)),[ $$status -ne 0 ] || docker buildx build --load -f deploy/compose/Dockerfile.ci --target nava-worker-interptest -t nava/nava-worker-interptest:$${NAVA_IMAGE_TAG:-dev} . || status=$$?;) \
	wait $$pull; exit $$status

# The Go service binaries for IMAGE_THIN=1, in one go build with the real
# Dockerfiles' flags, for linux/amd64 (the runner; GOOS/GOARCH override it).
# No build tags: the production nava-worker (TICK-367). Then the interptest
# stack worker, the same package with -tags interptest (the S11a fakes).
ci-binaries:
	go version
	CGO_ENABLED=0 GOOS=$${GOOS:-linux} GOARCH=$${GOARCH:-amd64} go build -trimpath -o $(CI_BIN_DIR)/ $(addprefix ./cmd/,$(CI_BINARIES))
	CGO_ENABLED=0 GOOS=$${GOOS:-linux} GOARCH=$${GOARCH:-amd64} go build -trimpath -tags interptest -o $(CI_BIN_DIR)/nava-worker-interptest ./cmd/nava-worker

stack-up: dev-ca
	docker compose up -d

# stack-test-<name>: one case, deploy/compose/test/cases/<name>.sh (make/ fragments with their own recipe override this).
stack-test-%:
	sh deploy/compose/test/stack-test.sh $*

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

