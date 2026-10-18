GOLANGCI_LINT ?= golangci-lint

build:
	go build -o bin/nava-core ./cmd/nava-core

test:
	go test -race ./...

lint:
	$(GOLANGCI_LINT) run ./...
