// Package server assembles the nava-core HTTP handlers and servers: the public
// API listener and the internal one for /metrics. Later tasks add their routes
// here (health on the internal listener in T04, the /api/v1 surface from T20).
package server

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// NewHandler returns the public root handler, served on NAVA_HTTP_ADDR. It
// carries no operational endpoints: /metrics lives on NewInternalHandler.
func NewHandler(log *slog.Logger, tel *telemetry.Telemetry) http.Handler {
	mux := http.NewServeMux()
	return httpobs.Middleware(log, tel, mux)
}

// NewInternalHandler returns the handler for the internal listener
// (NAVA_METRICS_ADDR), which must not be reachable from outside the network.
// T04 adds /healthz and /readyz here.
//
// /metrics serves the telemetry registry, never the Prometheus default one. It
// offers OpenMetrics, the only format that carries exemplars: histogram
// buckets then link to a trace_id. Scrapes are untraced, so they create no
// spans and no access lines.
func NewInternalHandler(log *slog.Logger, tel *telemetry.Telemetry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(tel.Registry, promhttp.HandlerOpts{
		Registry:          tel.Registry,
		EnableOpenMetrics: true,
	}))
	return httpobs.Middleware(log, tel, mux, "GET /metrics")
}

// New returns an *http.Server on addr with every timeout set from cfg.
func New(addr string, cfg config.HTTP, h http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}
