# CI entry point (T05). Taskfile.yml mirrors these commands for local use
# (go-task); keep the two in step.

GOLANGCI_LINT ?= golangci-lint

.PHONY: build run test test-integration lint

build:
	go build -o bin/nava-core ./cmd/nava-core

# Loads .env when present (copy .env.example). Unlike task run, values in .env
# override variables already set in the shell.
run:
	@set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/nava-core

test:
	go test -race ./...

# Needs NAVA_TEST_PG_URL: the superuser of a disposable PostgreSQL (see
# internal/db/dbtest). -p 1: the tests share the cluster's roles.
test-integration:
	go test -tags integration -p 1 -count=1 ./internal/db/dbtest/

lint:
	$(GOLANGCI_LINT) run ./...
