// Package server assembles the nava-core public API handler and server. The
// admin listener (/metrics, /healthz, /readyz) lives in internal/platform/telemetry,
// shared by every binary. Later tasks add their routes here (the /api/v1
// surface from T20).
package server

import (
	"errors"
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
// through resolver (internal/platform/auth); GET /api/v1/me reads the caller's
// group paths through paths, which may be nil only with OperatorOnly.
func NewHandler(log *slog.Logger, tel *telemetry.Telemetry, resolver identity.IdentityResolver, paths identity.GroupPathReader, opts ...Option) (http.Handler, error) {
	var o options
	for _, f := range opts {
		f(&o)
	}
	if paths == nil && !o.operatorOnly {
		return nil, errors.New("server: paths is nil; only an OperatorOnly handler may omit it")
	}
	mux := http.NewServeMux()
	h, err := newAPI(log, tel, resolver, paths, mux, o)
	if err != nil {
		return nil, err
	}
	return httpobs.Middleware(log, tel, h), nil
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
