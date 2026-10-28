.PHONY: ci ci-isolated ci-changed ci-stages ci-isolated-down ci-lint ci-unit ci-integration ci-stack review-paths

# Gate stages. Each fragment adds its own steps to a stage with a rule that has no recipe
# (ci-lint: my-target), so a new target never edits another fragment.

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

# Prints "security review required: yes/no" and "CTO sign-off required:
# yes/no" for the change against BASE (PLAN-001 D-P10).
review-paths:
	go run ./tools/ci/reviewpaths -base $(BASE)

