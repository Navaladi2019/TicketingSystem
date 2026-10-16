// Command nava-core is the platform's core API service. main only wires
// configuration, telemetry, logging and the HTTP server together.
package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.LookupEnv, os.Stdout)
	stop()
	if err != nil {
		slog.ErrorContext(ctx, "nava-core stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

// run serves until ctx is cancelled, then shuts the server down and flushes
// telemetry.
func run(ctx context.Context, lookup config.LookupFunc, stdout io.Writer) error {
	cfg, err := config.Load(lookup)
	if err != nil {
		return err
	}
	log := logging.New(stdout, cfg.Log)

	tel, err := telemetry.New(ctx, cfg.Telemetry)
	if err != nil {
		return err
	}
	srv := server.New(cfg.HTTP, server.NewHandler(log, tel), log)

	errc := make(chan error, 1)
	go func() {
		log.InfoContext(ctx, "nava-core listening",
			slog.String("addr", cfg.HTTP.Addr), slog.Bool("trace_export", tel.Exporting()))
		errc <- srv.ListenAndServe()
	}()

	var serveErr error
	select {
	case serveErr = <-errc:
	case <-ctx.Done():
		log.InfoContext(ctx, "nava-core shutting down")
	}

	// ctx is already cancelled here; shutdown gets its own deadline.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if serveErr == nil {
		serveErr = srv.Shutdown(shutdownCtx)
		if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
			serveErr = errors.Join(serveErr, err)
		}
	}
	telErr := tel.Shutdown(shutdownCtx)
	log.InfoContext(shutdownCtx, "telemetry shut down", slog.Bool("ok", telErr == nil))
	return errors.Join(serveErr, telErr)
}
