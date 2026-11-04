package main

import (
	"context"
	"log/slog"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/temporaltls"
)

// Seams for tests: the constructors that only fail on conditions this
// process cannot produce from configuration.
//
//nolint:gochecknoglobals // test seams, replaced only by tests
var (
	newOutboundClient = telemetry.NewOutboundClient
	healthTemporal    = health.Temporal
	newHealthMonitor  = health.NewMonitor
)

// newMonitor builds the dependency checks behind /healthz and /readyz. The
// returned func closes the pool and the client the checks use; call it after
// the monitor has stopped.
func newMonitor(ctx context.Context, log *slog.Logger, tel *telemetry.Telemetry, cfg config.Config) (*health.Monitor, func(), error) {
	pg, closePG, err := health.Postgres(ctx, cfg.Deps.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	tlsConfig, err := temporaltls.Load(cfg.Deps.TemporalTLS)
	if err != nil {
		closePG()
		return nil, nil, err
	}
	tc, closeTemporal, err := healthTemporal(cfg.Deps.TemporalAddress, tlsConfig, log)
	if err != nil {
		closePG()
		return nil, nil, err
	}
	out, err := newOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		closePG()
		closeTemporal()
		return nil, nil, err
	}
	mon, err := newHealthMonitor(log, tel.MeterProvider,
		[]health.Check{pg, health.Keycloak(cfg.Deps.KeycloakManagementURL, cfg.Deps.KeycloakTokenURL, out), tc}, health.Options{})
	if err != nil {
		closePG()
		closeTemporal()
		return nil, nil, err
	}
	return mon, func() { closePG(); closeTemporal() }, nil
}
