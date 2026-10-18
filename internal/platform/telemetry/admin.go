package telemetry

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// AdminRoutes registers extra operational routes (health and readiness) on
// the admin mux, next to /metrics.
type AdminRoutes func(mux *http.ServeMux)

// NewAdminHandler returns the handler for the admin listener
// (NAVA_ADMIN_ADDR), which must never be reachable from outside the network.
// Every binary serves the same one: /metrics from the telemetry registry, plus
// whatever routes adds (/healthz and /readyz).
//
// The mux is deliberately not wrapped in httpobs.Middleware: a scrape or a
// probe would start a public root span and an access line every few seconds.
// Admin requests are untraced, add no HTTP metrics and carry no request id; a
// handler panic is logged and becomes a 500.
//
// /metrics offers OpenMetrics, the only format that carries exemplars:
// histogram buckets then link to a trace_id.
func NewAdminHandler(log *slog.Logger, tel *Telemetry, routes ...AdminRoutes) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(tel.Registry, promhttp.HandlerOpts{
		Registry:          tel.Registry,
		EnableOpenMetrics: true,
	}))
	for _, r := range routes {
		r(mux)
	}
	return recoverAdmin(log, mux)
}

// NewAdminServer returns an *http.Server on addr serving h with every timeout
// set from cfg.
func NewAdminServer(addr string, cfg config.HTTP, h http.Handler, log *slog.Logger) *http.Server {
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

// recoverAdmin turns a handler panic into a logged 500, so one bad probe does
// not take the process down. http.ErrAbortHandler passes through untouched.
func recoverAdmin(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler { //nolint:errorlint // net/http compares by identity
				panic(v)
			}
			log.ErrorContext(r.Context(), "admin handler panic",
				slog.Any("panic", v), slog.String("stack", string(debug.Stack())))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}()
		next.ServeHTTP(w, r)
	})
}
