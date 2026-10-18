# nava/temporal-server: Temporal 1.28.4 with the nava Authorizer and ClaimMapper.
# Build: make temporal-server-image (from the repository root).
FROM golang:1.25.8-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test -short ./... \
 && CGO_ENABLED=0 go build -trimpath -o /out/temporal-server ./cmd/server

FROM debian:bookworm-slim
RUN useradd --system --uid 10001 --no-create-home temporal
COPY --from=build /out/temporal-server /usr/local/bin/temporal-server
COPY config /etc/temporal/config
USER 10001
# Frontend gRPC and Prometheus metrics. Internal ports bind to loopback only.
EXPOSE 7233 9090
# Certificates are mounted at /etc/temporal/certs (see config/development.yaml).
ENTRYPOINT ["/usr/local/bin/temporal-server", "--config", "/etc/temporal/config"]
CMD ["--env", "development"]
