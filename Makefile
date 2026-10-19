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
# CI entry point (T05). Taskfile.yml mirrors these commands for local use
# (go-task); keep the two in step.

GOLANGCI_LINT ?= golangci-lint

.PHONY: build run test test-integration lint generate check-openapi

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

# Regenerates api/api.gen.go from api/openapi.yaml (oapi-codegen v2.8.0, pinned
# by the go.mod tool directive).
generate:
	go generate ./...

# Additive-only rule for /api/v1: fails when api/openapi.yaml removes or
# narrows anything the base branch (BASE, default origin/main) offers.
BASE ?= origin/main
check-openapi:
	@if git cat-file -e $(BASE):api/openapi.yaml 2>/dev/null; then 		git show $(BASE):api/openapi.yaml > $${TMPDIR:-/tmp}/openapi.base.yaml && 		go run ./cmd/openapi-compat $${TMPDIR:-/tmp}/openapi.base.yaml api/openapi.yaml; 	else echo "no api/openapi.yaml on $(BASE): nothing to compare"; fi

CERT_DIR ?= deploy/compose/certs

## Dev installation CA for the compose stack (ADR-001 D22, T28): the installation
## (frontend client) CA, a separate internode CA, the frontend and internode
## certificates, and the five platform certificates, written to
## deploy/compose/certs/ (git-ignored). Keeps existing certificates; FORCE=1 reissues.
dev-ca:
	@if [ -f $(CERT_DIR)/ca.pem ] && [ -z "$(FORCE)" ]; then echo "$(CERT_DIR) already holds a dev CA (FORCE=1 to reissue)"; 	else cd temporal-server && go run ./cmd/devcerts -installation -out ../$(CERT_DIR); fi

## Explicit operator tunnel to the Temporal UI (profile operator): forwards
## 127.0.0.1:$(UI_PORT) to the UI until interrupted. The UI publishes no port.
UI_PORT ?= 18233
operator-tunnel:
	docker run --rm --name nava-operator-tunnel -p 127.0.0.1:$(UI_PORT):8080 		--network $$(docker network ls -q --filter label=com.docker.compose.project=nava --filter label=com.docker.compose.network=operator) 		--entrypoint /usr/local/bin/temporal-server nava/temporal-server:dev proxy -listen :8080 -to temporal-ui:8080
