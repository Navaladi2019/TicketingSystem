# T27: the custom Temporal server module and its dev certificates.

TEMPORAL_SERVER_IMAGE ?= nava/temporal-server:$(NAVA_IMAGE_TAG)

.PHONY: temporal-server-test temporal-server-image temporal-server-dev-certs temporal-server-run dev-ca operator-tunnel

ci-unit: temporal-server-test

## Unit, policy and end-to-end tests of the custom Temporal server module.
temporal-server-test:
	cd temporal-server && go vet ./...
	go run ./tools/ci/suiteran -- go -C temporal-server test -json ./...

## Build the custom Temporal server image (ADR-001 D22, task T27a).
temporal-server-image:
	docker build -t $(TEMPORAL_SERVER_IMAGE) temporal-server

## Write throwaway development certificates to temporal-server/.devcerts.
temporal-server-dev-certs:
	cd temporal-server && go run ./cmd/devcerts -out .devcerts

## Run the image with the dev config and dev certificates (foreground).
temporal-server-run: temporal-server-dev-certs
	MSYS_NO_PATHCONV=1 docker run --rm --name nava-temporal-dev -p 7233:7233 -p 9090:9090 \
		-v "$(CURDIR)/temporal-server/.devcerts:/etc/temporal/certs:ro" $(TEMPORAL_SERVER_IMAGE)

CERT_DIR ?= deploy/compose/certs

## Dev installation CA for the compose stack (ADR-001 D22, T28): the installation
## (frontend client) CA, a separate internode CA, the frontend and internode
## certificates, the five platform certificates and nava-unknown (a CN that is
## no identity, for the CI-6 suite; never mounted), written to
## deploy/compose/certs/ (git-ignored). Keeps existing certificates; FORCE=1 reissues.
dev-ca:
	@if [ -f $(CERT_DIR)/ca.pem ] && [ -z "$(FORCE)" ]; then echo "$(CERT_DIR) already holds a dev CA (FORCE=1 to reissue)"; 	else cd temporal-server && go run ./cmd/devcerts -installation -unlisted -out ../$(CERT_DIR); fi

## Explicit operator tunnel to the Temporal UI (profile operator): forwards
## 127.0.0.1:$(UI_PORT) to the UI until interrupted. The UI publishes no port.
# TODO(TODO-006): while the tunnel is open, any container on a non-internal
# network reaches the UI through host.docker.internal and acts as nava-operator.
UI_PORT ?= 18233
operator-tunnel:
	MSYS_NO_PATHCONV=1 docker run --rm --name $(COMPOSE_PROJECT)-operator-tunnel -p 127.0.0.1:$(UI_PORT):8080 		--network $$(docker network ls -q --filter label=com.docker.compose.project=$(COMPOSE_PROJECT) --filter label=com.docker.compose.network=operator) 		--entrypoint /usr/local/bin/temporal-server $(TEMPORAL_SERVER_IMAGE) proxy -listen :8080 -to temporal-ui:8080

