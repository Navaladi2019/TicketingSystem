// Package server assembles the nava-core HTTP handler and server. Later tasks
// add their routes here (health in T04, the /api/v1 surface from T20).
package server

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
)

// NewHandler returns the root handler. It uses its own registry, never the
// Prometheus default one, so there is no package-level metric state.
func NewHandler(log *slog.Logger) (http.Handler, error) {
	reg := prometheus.NewRegistry()
	if err := reg.Register(collectors.NewGoCollector()); err != nil {
		return nil, err
	}
	if err := reg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return nil, err
	}
	m, err := httpobs.NewMetrics(reg)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	return httpobs.Middleware(log, m, mux), nil
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
