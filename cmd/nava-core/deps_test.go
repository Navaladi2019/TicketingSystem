package main

import (
	"crypto/tls"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

func newTel(t *testing.T) *telemetry.Telemetry {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	return tel
}

const validDSN = "postgres://u:pw@127.0.0.1:1/db?sslmode=disable"

func TestNewMonitorFailures(t *testing.T) {
	t.Parallel()
	log := slog.New(slog.DiscardHandler)
	tel := newTel(t)
	tests := []struct {
		name string
		tel  *telemetry.Telemetry
		deps config.Dependencies
		want string
	}{
		{"bad database url", tel, config.Dependencies{DatabaseURL: "not a url \x7f"}, "NAVA_DATABASE_URL"},
		{"bad temporal tls", tel, config.Dependencies{DatabaseURL: validDSN, TemporalTLS: config.TemporalTLS{CertFile: "missing.pem", KeyFile: "missing.key"}}, "temporal TLS"},
	}
	for _, tt := range tests {
		mon, closeFn, err := newMonitor(t.Context(), log, tt.tel, config.Config{Deps: tt.deps}, nil)
		if err == nil || mon != nil || closeFn != nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: mon=%v err=%v", tt.name, mon, err)
		}
		if strings.Contains(fmtErr(err), "pw") {
			t.Errorf("%s: the error quotes the credential: %v", tt.name, err)
		}
	}
}

func fmtErr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestNewMonitorBuildsWithUnconfiguredDependencies(t *testing.T) {
	t.Parallel()
	mon, closeFn, err := newMonitor(t.Context(), slog.New(slog.DiscardHandler), newTel(t), config.Config{}, nil)
	if err != nil || mon == nil || closeFn == nil {
		t.Fatalf("mon=%v err=%v", mon, err)
	}
	closeFn()
}

func TestNewResolverNotConfiguredRefusesEveryToken(t *testing.T) {
	t.Parallel()
	for name, deps := range map[string]config.Dependencies{
		"no issuers":  {DatabaseURL: validDSN},
		"no database": {OIDCIssuers: []string{"https://idp.example/realms/a"}},
	} {
		res, groups, pool, closeFn, err := newResolver(t.Context(), slog.New(slog.DiscardHandler), newTel(t), config.Config{Deps: deps})
		if err != nil || pool != nil || closeFn == nil {
			t.Fatalf("%s: pool=%v err=%v", name, pool, err)
		}
		closeFn()
		if _, err := res.Resolve(t.Context(), "any.jwt.token"); !errors.Is(err, identity.ErrInvalidToken) {
			t.Errorf("%s: Resolve err = %v, want ErrInvalidToken", name, err)
		}
		if paths, err := groups.GroupPaths(t.Context(), uuid.New(), uuid.New()); err == nil || paths != nil {
			t.Errorf("%s: GroupPaths = %v, %v, want an error", name, paths, err)
		}
	}
}

func TestNewResolverFailures(t *testing.T) {
	t.Parallel()
	log := slog.New(slog.DiscardHandler)
	issuers := []string{"https://idp.example/realms/a"}
	tel := newTel(t)
	tests := []struct {
		name string
		tel  *telemetry.Telemetry
		deps config.Dependencies
		want string
	}{
		{"bad database url", tel, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: "postgres://u:pw@host:notaport/db"}, "NAVA_DATABASE_URL"},
		{"no providers", &telemetry.Telemetry{}, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: validDSN}, ""},
		{"adapter refuses an empty issuer", tel, config.Dependencies{OIDCIssuers: []string{""}, DatabaseURL: validDSN}, "identity:"},
	}
	for _, tt := range tests {
		res, groups, pool, closeFn, err := newResolver(t.Context(), log, tt.tel, config.Config{Deps: tt.deps})
		if err == nil || res != nil || groups != nil || pool != nil || closeFn != nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v", tt.name, err)
		}
		if strings.Contains(fmtErr(err), "pw") {
			t.Errorf("%s: the error quotes the credential: %v", tt.name, err)
		}
	}
}

func TestNewResolverBuildsWithoutReachingTheDatabase(t *testing.T) {
	t.Parallel()
	res, groups, pool, closeFn, err := newResolver(t.Context(), slog.New(slog.DiscardHandler), newTel(t),
		config.Config{Deps: config.Dependencies{OIDCIssuers: []string{"https://idp.example/realms/a"}, DatabaseURL: validDSN}})
	if err != nil || res == nil || groups == nil || pool == nil {
		t.Fatalf("res=%v groups=%v pool=%v err=%v", res, groups, pool, err)
	}
	defer closeFn()
	// A token that is not a JWT is refused before any key or database is used.
	if _, err := res.Resolve(t.Context(), "not-a-jwt"); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("Resolve err = %v, want ErrInvalidToken", err)
	}
}

// Not parallel: these swap the package-level constructors.
func TestNewMonitorConstructorFailuresCloseWhatWasOpened(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	tel := newTel(t)
	boom := errors.New("boom")
	cfg := config.Config{Deps: config.Dependencies{DatabaseURL: validDSN, TemporalAddress: "127.0.0.1:1"}}

	t.Run("temporal client", func(t *testing.T) {
		orig := healthTemporal
		healthTemporal = func(string, *tls.Config, *slog.Logger) (health.Check, func(), error) {
			return health.Check{}, nil, boom
		}
		t.Cleanup(func() { healthTemporal = orig })
		if mon, closeFn, err := newMonitor(t.Context(), log, tel, cfg, nil); !errors.Is(err, boom) || mon != nil || closeFn != nil {
			t.Fatalf("mon=%v err=%v", mon, err)
		}
	})
	t.Run("outbound client", func(t *testing.T) {
		orig := newOutboundClient
		newOutboundClient = func(*telemetry.Telemetry, telemetry.OutboundOptions) (*telemetry.OutboundClient, error) {
			return nil, boom
		}
		t.Cleanup(func() { newOutboundClient = orig })
		if mon, closeFn, err := newMonitor(t.Context(), log, tel, cfg, nil); !errors.Is(err, boom) || mon != nil || closeFn != nil {
			t.Fatalf("mon=%v err=%v", mon, err)
		}
		if _, _, pool, closeFn, err := newResolver(t.Context(), log, tel, config.Config{Deps: config.Dependencies{
			OIDCIssuers: []string{"https://idp.example/realms/a"}, DatabaseURL: validDSN,
		}}); !errors.Is(err, boom) || pool != nil || closeFn != nil {
			t.Fatalf("resolver: pool=%v err=%v", pool, err)
		}
	})
	t.Run("monitor", func(t *testing.T) {
		orig := newHealthMonitor
		newHealthMonitor = func(*slog.Logger, metric.MeterProvider, []health.Check, health.Options) (*health.Monitor, error) {
			return nil, boom
		}
		t.Cleanup(func() { newHealthMonitor = orig })
		if mon, closeFn, err := newMonitor(t.Context(), log, tel, cfg, nil); !errors.Is(err, boom) || mon != nil || closeFn != nil {
			t.Fatalf("mon=%v err=%v", mon, err)
		}
	})
}
