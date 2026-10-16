// Command nava-core is the platform's core API service. main only wires
// configuration, logging and the HTTP server together.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("nava-core stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return err
	}
	log := logging.New(os.Stdout, cfg.Log)

	h, err := server.NewHandler(log)
	if err != nil {
		return err
	}
	srv := server.New(cfg.HTTP, h, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		log.Info("nava-core listening", slog.String("addr", cfg.HTTP.Addr))
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	log.Info("nava-core shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
