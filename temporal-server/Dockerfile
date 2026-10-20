# nava/temporal-server: Temporal 1.28.4 with the nava Authorizer and ClaimMapper.
# Build: make temporal-server-image (from the repository root).
FROM golang:1.25.8-bookworm AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
# Compile the Temporal server dependencies in their own layer, keyed only on
# go.mod/go.sum, so a source change rebuilds just this module's packages.
# The module's tests run in CI's temporal-server job, not in the image build.
RUN go build -trimpath \
      go.temporal.io/server/temporal \
      go.temporal.io/server/common/persistence/sql/sqlplugin/postgresql \
      go.temporal.io/server/common/persistence/sql/sqlplugin/sqlite \
      go.temporal.io/server/cmd/tools/sql
COPY . .
RUN go build -trimpath -o /out/temporal-server ./cmd/server \
 && go build -trimpath -o /out/temporal-sql-tool go.temporal.io/server/cmd/tools/sql \
 && cp -r "$(go list -m -f '{{.Dir}}' go.temporal.io/server)/schema/postgresql/v12" /out/schema \
 && chmod -R a+rX /out/schema

FROM debian:bookworm-slim
RUN useradd --system --uid 10001 --no-create-home temporal
COPY --from=build /out/temporal-server /usr/local/bin/temporal-server
COPY --from=build /out/temporal-sql-tool /usr/local/bin/temporal-sql-tool
COPY --from=build /out/schema /etc/temporal/schema
COPY scripts/setup-schema.sh /usr/local/bin/setup-schema.sh
COPY config /etc/temporal/config
USER 10001
# Frontend gRPC and Prometheus metrics. Internal ports bind to loopback only.
EXPOSE 7233 9090
# Certificates are mounted at /etc/temporal/certs (see config/development.yaml).
# `temporal-server health` and `temporal-server proxy` are helper subcommands.
ENTRYPOINT ["/usr/local/bin/temporal-server", "--config", "/etc/temporal/config"]
CMD ["--env", "development"]
