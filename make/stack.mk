.PHONY: ci-stack ci-stack-run stack-tests stack-main stack-internal-beside stack-internal stack-up stack-down stack-logs image image-ci smoke

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
# Fragments add their own stack tests with STACK_TESTS_SERIAL += <name> (the stack-test-<name>
# target). Those run in include (alphabetical) order, before the disruptive cases below.
STACK_TESTS_SERIAL += dependencies trace logs redact idempotent dbroles
STACK_TESTS = healthy $(STACK_TESTS_PARALLEL) $(STACK_TESTS_SERIAL)
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
BAKE_CACHED := db-roles keycloak nava-core nava-groupsync nava-onboard nava-purge nava-relay nava-worker otel-collector temporal-server
BAKE_CACHE_GHA := $(foreach t,$(BAKE_CACHED),--set $(t).cache-from=type=gha,scope=$(t) --set $(t).cache-to=type=gha,scope=$(t),mode=max,ignore-error=true) --set migrate.cache-from=type=gha,scope=db-roles
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

