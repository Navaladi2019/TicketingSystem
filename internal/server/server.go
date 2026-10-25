// Package server assembles the nava-core public API handler and server. The
// admin listener (/metrics, /healthz, /readyz) lives in internal/platform/telemetry,
// shared by every binary. Later tasks add their routes here (the /api/v1
// surface from T20).
package server

import (
	"log/slog"
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// NewHandler returns the public root handler, served on NAVA_HTTP_ADDR. It
// serves /api/v1/* only: /metrics, /healthz and /readyz live on the admin
// listener (telemetry.NewAdminHandler). Every contract operation authenticates
// through resolver (internal/platform/auth).
func NewHandler(log *slog.Logger, tel *telemetry.Telemetry, resolver identity.IdentityResolver) (http.Handler, error) {
	mux := http.NewServeMux()
	a, err := newAPI(log, tel, resolver, mux)
	if err != nil {
		return nil, err
	}
	return httpobs.Middleware(log, tel, a.Wrap(mux)), nil
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
