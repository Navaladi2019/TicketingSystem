package health

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	tlog "go.temporal.io/sdk/log"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// Postgres returns a critical check that pings the database at dsn, and the
// function that closes its pool. The pool connects lazily, so an unreachable
// database does not fail startup: it fails readiness.
//
// dsn is a credential. Neither the returned errors nor the probe's own errors
// repeat it.
func Postgres(ctx context.Context, dsn string) (Check, func(), error) {
	if dsn == "" {
		return Check{Name: "postgres", Critical: true, Probe: unconfigured}, func() {}, nil
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return Check{}, nil, errors.New("health: NAVA_DATABASE_URL is not a valid PostgreSQL connection string")
	}
	// The pool exists to answer a ping: one connection is enough, and a
	// short connect timeout keeps a probe inside its own deadline.
	cfg.MaxConns = 1
	cfg.MinConns = 0
	cfg.ConnConfig.ConnectTimeout = 2 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return Check{}, nil, errors.New("health: cannot create the PostgreSQL pool")
	}
	return Check{Name: "postgres", Critical: true, Probe: pool.Ping}, pool.Close, nil
}

// Keycloak returns a check on Keycloak's management port (9000):
// GET <managementURL>/health/ready must answer 200. client is the outbound
// client from telemetry.NewOutboundClient. managementURL is the base, such as
// http://keycloak:9000; empty means not configured. The probe runs every
// round, so it is not traced.
func Keycloak(managementURL string, client *telemetry.OutboundClient) Check {
	c := Check{Name: "keycloak"}
	if managementURL == "" {
		c.Probe = unconfigured
		return c
	}
	c.Probe = func(ctx context.Context) error {
		ctx = telemetry.WithoutTrace(telemetry.WithRoute(ctx, "/health/ready"))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, managementURL+"/health/ready", nil)
		if err != nil {
			return errors.New("health: bad Keycloak management URL")
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("keycloak /health/ready: status %d", resp.StatusCode)
		}
		return nil
	}
	return c
}

// Temporal returns a check that calls GetSystemInfo on the frontend at
// address. tlsConfig is nil for plaintext; once T28 lands it carries the
// nava-core client certificate. The client connects lazily. Empty address means
// not configured.
func Temporal(address string, tlsConfig *tls.Config, log *slog.Logger) (Check, func(), error) {
	c := Check{Name: "temporal"}
	if address == "" {
		c.Probe = unconfigured
		return c, func() {}, nil
	}
	tc, err := client.NewLazyClient(client.Options{
		HostPort:          address,
		Logger:            tlog.NewStructuredLogger(log),
		ConnectionOptions: client.ConnectionOptions{TLS: tlsConfig},
	})
	if err != nil {
		return Check{}, nil, errors.New("health: cannot create the Temporal client")
	}
	c.Probe = func(ctx context.Context) error {
		_, err := tc.WorkflowService().GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
		return err
	}
	return c, tc.Close, nil
}

func unconfigured(context.Context) error { return ErrUnconfigured }
