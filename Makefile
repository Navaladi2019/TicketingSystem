# CI entry point (T05). Taskfile.yml mirrors these commands for local use
# (go-task); keep the two in step.

GOLANGCI_LINT ?= golangci-lint

.PHONY: build run test lint

build:
	go build -o bin/nava-core ./cmd/nava-core

# Loads .env when present (copy .env.example). Unlike task run, values in .env
# override variables already set in the shell.
run:
	@set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/nava-core

test:
	go test -race ./...

lint:
	$(GOLANGCI_LINT) run ./...
