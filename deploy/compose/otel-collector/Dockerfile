# The contrib image is scratch-based, so it has no shell or wget for a container
# health check. Copy its binary onto a pinned Alpine.
FROM otel/opentelemetry-collector-contrib:0.139.0 AS collector

FROM alpine:3.22.1
COPY --from=collector /otelcol-contrib /otelcol-contrib
COPY config.yaml /etc/otelcol/config.yaml
USER 10001:10001
EXPOSE 4317 4318
ENTRYPOINT ["/otelcol-contrib"]
CMD ["--config=/etc/otelcol/config.yaml"]
