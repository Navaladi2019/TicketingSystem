# Build context is the repo root (see compose.yaml).
FROM golang:1.26.10-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY api ./api
# Empty for the production image. Only the interptest stack worker
# (nava-worker-interptest in deploy/compose/compose.test.yaml) sets it, to link
# the S11a fakes (TICK-367). TestProductionBuildLinksNoFakes fails on a
# non-empty default.
ARG NAVA_BUILD_TAGS=""
RUN CGO_ENABLED=0 go build -trimpath -tags "${NAVA_BUILD_TAGS}" -o /out/nava-worker ./cmd/nava-worker

FROM alpine:3.22.1
RUN adduser -D -u 10001 nava
COPY --from=build /out/nava-worker /usr/local/bin/nava-worker
USER nava
ENTRYPOINT ["/usr/local/bin/nava-worker"]
