package health

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
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

// Keycloak returns a check on Keycloak: GET <managementURL>/health/ready must
// answer 200 (management port 9000). client is the outbound client from
// telemetry.NewOutboundClient. managementURL is the base, such as
// http://keycloak:9000; empty means not configured. The probe runs every
// round, so it is not traced.
//
// /health/ready stayed 200 while Keycloak held dead database connections after
// a PostgreSQL restart and refused every token (TICK-328, R4). With tokenURL
// set, the check also POSTs a client_credentials request for a client that does
// not exist to the realm's token endpoint. Keycloak does not cache a missing
// client, so it must read the database to answer, and a healthy one answers
// 401 invalid_client. Any other answer fails the check. The request carries no
// credential.
func Keycloak(managementURL, tokenURL string, client *telemetry.OutboundClient) Check {
	c := Check{Name: "keycloak"}
	if managementURL == "" {
		c.Probe = unconfigured
		return c
	}
	c.Probe = func(ctx context.Context) error {
		if err := keycloakGet(ctx, client, managementURL+"/health/ready", "/health/ready", http.StatusOK); err != nil {
			return err
		}
		if tokenURL == "" {
			return nil
		}
		return keycloakTokenProbe(ctx, client, tokenURL)
	}
	return c
}

func keycloakGet(ctx context.Context, client *telemetry.OutboundClient, url, route string, want int) error {
	ctx = telemetry.WithoutRedirects(telemetry.WithoutTrace(telemetry.WithRoute(ctx, route)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.New("health: bad Keycloak management URL")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	drainClose(resp)
	if resp.StatusCode != want {
		return fmt.Errorf("keycloak %s: status %d", route, resp.StatusCode)
	}
	return nil
}

func keycloakTokenProbe(ctx context.Context, client *telemetry.OutboundClient, tokenURL string) error {
	const route = "/protocol/openid-connect/token"
	ctx = telemetry.WithoutRedirects(telemetry.WithoutTrace(telemetry.WithRoute(ctx, route)))
	form := strings.NewReader("grant_type=client_credentials&client_id=nava-health-probe")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, form)
	if err != nil {
		return errors.New("health: bad Keycloak token URL")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	drainClose(resp)
	if resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("keycloak token endpoint: status %d, want 401 invalid_client", resp.StatusCode)
	}
	return nil
}

func drainClose(resp *http.Response) {
	// Drain a little so the connection is reused on the next round.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	_ = resp.Body.Close()
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
