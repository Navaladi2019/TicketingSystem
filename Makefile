TEMPORAL_SERVER_IMAGE ?= nava/temporal-server:dev

.PHONY: temporal-server-test temporal-server-image temporal-server-dev-certs temporal-server-run

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
