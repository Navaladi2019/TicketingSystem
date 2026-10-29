.PHONY: build run generate

build:
	go build -o bin/nava-core ./cmd/nava-core

# Loads .env when present (copy .env.example). Unlike task run, values in .env
# override variables already set in the shell.
run:
	@set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/nava-core

# Bundles api/openapi.yaml + api/paths/*.yaml, then regenerates api/api.gen.go (oapi-codegen v2.8.0, pinned
# by the go.mod tool directive).
generate:
	go generate ./...

