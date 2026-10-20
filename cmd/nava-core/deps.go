package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
	"os"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// newMonitor builds the dependency checks behind /healthz and /readyz. The
// returned func closes the pool and the client the checks use; call it after
// the monitor has stopped.
func newMonitor(ctx context.Context, log *slog.Logger, tel *telemetry.Telemetry, cfg config.Config) (*health.Monitor, func(), error) {
	pg, closePG, err := health.Postgres(ctx, cfg.Deps.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	tlsConfig, err := temporalTLS(cfg.Deps.TemporalTLS)
	if err != nil {
		closePG()
		return nil, nil, err
	}
	tc, closeTemporal, err := health.Temporal(cfg.Deps.TemporalAddress, tlsConfig, log)
	if err != nil {
		closePG()
		return nil, nil, err
	}
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		closePG()
		closeTemporal()
		return nil, nil, err
	}
	mon, err := health.NewMonitor(log, tel.MeterProvider,
		[]health.Check{pg, health.Keycloak(cfg.Deps.KeycloakManagementURL, out), tc}, health.Options{})
	if err != nil {
		closePG()
		closeTemporal()
		return nil, nil, err
	}
	return mon, func() { closePG(); closeTemporal() }, nil
}

// temporalTLS builds the client TLS config for the Temporal frontend from the
// nava-core certificate, or returns nil (plaintext) when none is configured.
// File errors name the variable, not the path contents.
func temporalTLS(t config.TemporalTLS) (*tls.Config, error) {
	if !t.Enabled() {
		return nil, nil
	}
	pair, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
	if err != nil {
		return nil, errors.New("temporal TLS: cannot load NAVA_TEMPORAL_TLS_CERT_FILE and NAVA_TEMPORAL_TLS_KEY_FILE")
	}
	caPEM, err := os.ReadFile(t.CAFile)
	if err != nil {
		return nil, errors.New("temporal TLS: cannot read NAVA_TEMPORAL_TLS_CA_FILE")
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("temporal TLS: NAVA_TEMPORAL_TLS_CA_FILE holds no certificate")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{pair},
		RootCAs:      pool,
		ServerName:   t.ServerName,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
