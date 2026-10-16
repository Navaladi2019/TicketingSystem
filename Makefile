# CI entry point (T05). Taskfile.yml mirrors these commands for local use
# (go-task); keep the two in step.

GOLANGCI_LINT ?= golangci-lint

.PHONY: build test lint

build:
	go build -o bin/nava-core ./cmd/nava-core

test:
	go test -race ./...

lint:
	$(GOLANGCI_LINT) run ./...
