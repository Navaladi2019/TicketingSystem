// Command nava-core is the platform's core API service. main only wires
// configuration, telemetry, logging and the HTTP servers together.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

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

// run serves the public and admin listeners until ctx is cancelled or
// either one fails, then drains both and flushes telemetry.
func run(ctx context.Context, lookup config.LookupFunc, stdout io.Writer) error {
	cfg, err := config.Load(lookup)
	if err != nil {
		return err
	}
	log := logging.New(stdout, cfg.Log)
	telemetry.RouteErrors(log)

	tel, err := telemetry.New(ctx, cfg.Telemetry)
	if err != nil {
		return err
	}
	mon, closeDeps, err := newMonitor(ctx, log, tel, cfg)
	if err != nil {
		_ = tel.Shutdown(context.WithoutCancel(ctx))
		return err
	}
	defer closeDeps()
	listeners := []struct {
		name string
		srv  *http.Server
	}{
		{"public", server.New(cfg.HTTP.Addr, cfg.HTTP, server.NewHandler(log, tel), log)},
		{"admin", telemetry.NewAdminServer(cfg.HTTP.AdminAddr, cfg.HTTP, telemetry.NewAdminHandler(log, tel, mon.Routes), log)},
	}

	log.InfoContext(ctx, "nava-core listening",
		slog.String("addr", cfg.HTTP.Addr), slog.String("admin_addr", cfg.HTTP.AdminAddr),
		slog.Bool("trace_export", tel.Exporting()))
	// Either server stopping (a listen failure, a recovered panic) cancels
	// stopCtx, so the other drains too. Both drain before telemetry flushes.
	stopCtx, stopAll := context.WithCancel(ctx)
	defer stopAll()
	serveErrs := make([]error, len(listeners))
	var wg sync.WaitGroup
	wg.Go(func() { mon.Run(stopCtx) })
	for i, l := range listeners {
		wg.Go(func() {
			llog := log.With(slog.String("listener", l.name))
			serveErrs[i] = serve(stopCtx, llog, l.srv, l.srv.ListenAndServe, cfg.HTTP.ShutdownTimeout)
			stopAll()
		})
	}
	wg.Wait()
	serveErr := errors.Join(serveErrs...)

	// The telemetry flush has its own deadline, separate from the drain, so a
	// slow drain cannot use up the time the last requests' spans need to leave.
	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.Telemetry.ShutdownTimeout)
	defer cancel()
	telErr := tel.Shutdown(flushCtx)
	if telErr != nil {
		log.WarnContext(flushCtx, "telemetry shutdown failed", slog.Any("error", telErr))
	} else {
		log.InfoContext(flushCtx, "telemetry shut down")
	}
	return errors.Join(serveErr, telErr)
}

// serve runs listen until it returns or ctx is done, then drains srv. A
// signal and listen ending (an error, or a panic recovered into one) take the
// same path: Shutdown also runs when serving failed, because connections
// accepted before a panic in the serve loop are still in flight.
func serve(ctx context.Context, log *slog.Logger, srv *http.Server, listen func() error, drainTimeout time.Duration) error {
	errc := make(chan error, 1)
	go func() { errc <- recovered(listen) }()

	var serveErr error
	served := false
	select {
	case serveErr = <-errc:
		served = true
		if !errors.Is(serveErr, http.ErrServerClosed) {
			log.ErrorContext(ctx, "nava-core serve failed", slog.Any("error", serveErr))
		}
	case <-ctx.Done():
	}
	log.InfoContext(ctx, "nava-core shutting down")

	// ctx may not be cancelled yet, and must not cut the drain short.
	drainCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), drainTimeout)
	defer cancel()
	shutErr := srv.Shutdown(drainCtx)
	if !served {
		serveErr = <-errc
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, shutErr)
}

// recovered runs fn and turns a panic into an error carrying the stack, so a
// panicking goroutine leads to a graceful shutdown instead of killing the
// process. Every long-lived goroutine run starts goes through it. Request
// handlers do not need it: httpobs recovers them per request.
func recovered(fn func() error) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("panic: %v\n%s", v, debug.Stack())
		}
	}()
	return fn()
}
