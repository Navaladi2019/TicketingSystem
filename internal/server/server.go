// Package server assembles the nava-core HTTP handler and server. Later tasks
// add their routes here (health in T04, the /api/v1 surface from T20).
package server

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// NewHandler returns the root handler. /metrics serves the telemetry
// registry, never the Prometheus default one. It offers OpenMetrics, the only
// format that carries exemplars: histogram buckets then link to a trace_id.
// Scrapes are untraced, so they create no spans and no access lines.
//
// TODO(TODO-001): /metrics shares the public listener and has no auth, so
// anyone who can reach :8080 can read route names, traffic volume, latency,
// error rates and process details. Move it to its own internal listener
// (NAVA_METRICS_ADDR, default :9090), drain both servers on shutdown, and test
// that the public port answers 404. See docs/delivery/todo.md.
func NewHandler(log *slog.Logger, tel *telemetry.Telemetry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(tel.Registry, promhttp.HandlerOpts{
		Registry:          tel.Registry,
		EnableOpenMetrics: true,
	}))
	return httpobs.Middleware(log, tel, mux, "GET /metrics")
}

// New returns an *http.Server with every timeout set from cfg.
func New(cfg config.HTTP, h http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}
