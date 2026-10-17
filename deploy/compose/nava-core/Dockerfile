# Build context is the repo root (see compose.yaml).
FROM golang:1.26.9-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/nava-core ./cmd/nava-core

FROM alpine:3.22.1
RUN adduser -D -u 10001 nava
COPY --from=build /out/nava-core /usr/local/bin/nava-core
USER nava
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/nava-core"]
