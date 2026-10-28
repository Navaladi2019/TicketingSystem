# Build context is the repo root (see compose.yaml).
FROM golang:1.26.10-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY api ./api
RUN CGO_ENABLED=0 go build -trimpath -o /out/nava-groupsync ./cmd/nava-groupsync

FROM alpine:3.22.1
RUN adduser -D -u 10001 nava
COPY --from=build /out/nava-groupsync /usr/local/bin/nava-groupsync
USER nava
ENTRYPOINT ["/usr/local/bin/nava-groupsync"]
