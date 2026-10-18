# Build context is the repo root (see compose.yaml). One image for the two
# one-shot database jobs: `nava-db roles` (db-roles) and `nava-db migrate`.
FROM golang:1.26.9-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY db ./db
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/nava-db ./cmd/nava-db

FROM alpine:3.22.1
RUN adduser -D -u 10001 nava
COPY --from=build /out/nava-db /usr/local/bin/nava-db
USER nava
ENTRYPOINT ["/usr/local/bin/nava-db"]
