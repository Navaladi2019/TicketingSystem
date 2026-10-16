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
// registry, never the Prometheus default one.
func NewHandler(log *slog.Logger, tel *telemetry.Telemetry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(tel.Registry, promhttp.HandlerOpts{Registry: tel.Registry}))
	return httpobs.Middleware(log, tel, mux)
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
