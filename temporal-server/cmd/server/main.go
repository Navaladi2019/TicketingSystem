// Command server runs Temporal 1.28.4 with nava's deny-by-default Authorizer and
// certificate ClaimMapper (ADR-001 D22, Annex A.2).
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/metrics"
	_ "go.temporal.io/server/common/persistence/sql/sqlplugin/postgresql"
	_ "go.temporal.io/server/common/persistence/sql/sqlplugin/sqlite"
	"go.temporal.io/server/temporal"

	"github.com/Navaladi2019/TicketingSystem/temporal-server/authz"
)

func main() {
	configDir := flag.String("config", "/etc/temporal/config", "config directory")
	env := flag.String("env", "development", "config environment (file <env>.yaml)")
	zone := flag.String("zone", "", "config zone")
	flag.Parse()

	if err := run(*configDir, *env, *zone); err != nil {
		fmt.Fprintln(os.Stderr, "temporal-server:", err)
		os.Exit(1)
	}
}

func run(configDir, env, zone string) error {
	cfg, err := config.LoadConfig(env, configDir, zone)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger := log.NewZapLogger(log.BuildZapLogger(cfg.Log))

	var dc dynamicconfig.Client = dynamicconfig.NewNoopClient()
	if cfg.DynamicConfigClient != nil {
		if !filepath.IsAbs(cfg.DynamicConfigClient.Filepath) {
			cfg.DynamicConfigClient.Filepath = filepath.Join(configDir, cfg.DynamicConfigClient.Filepath)
		}
		dc, err = dynamicconfig.NewFileBasedClient(cfg.DynamicConfigClient, logger, temporal.InterruptCh())
		if err != nil {
			return fmt.Errorf("dynamic config: %w", err)
		}
	}

	s, err := newServer(cfg, logger, dc, temporal.InterruptOn(temporal.InterruptCh()))
	if err != nil {
		return err
	}
	logger.Info("Starting Temporal server with nava Authorizer", tag.NewStringTag("namespace", authz.Namespace))
	// Start blocks until interrupted.
	return s.Start()
}

// newServer builds the server with the nava Authorizer and ClaimMapper. It
// refuses a config without frontend mTLS: the ClaimMapper's only input is the
// client certificate.
func newServer(cfg *config.Config, logger log.Logger, dc dynamicconfig.Client, extra ...temporal.ServerOption) (temporal.Server, error) {
	if err := checkTLS(cfg); err != nil {
		return nil, err
	}
	mh, err := metrics.MetricsHandlerFromConfig(logger, cfg.Global.Metrics)
	if err != nil {
		return nil, fmt.Errorf("metrics: %w", err)
	}
	opts := append([]temporal.ServerOption{
		temporal.ForServices(temporal.DefaultServices),
		temporal.WithConfig(cfg),
		temporal.WithLogger(logger),
		temporal.WithCustomMetricsHandler(mh),
		temporal.WithDynamicConfigClient(dc),
		temporal.WithAuthorizer(authz.NewAuthorizer(logger, mh)),
		temporal.WithClaimMapper(func(*config.Config) authorization.ClaimMapper { return authz.ClaimMapper{} }),
	}, extra...)
	s, err := temporal.NewServer(opts...)
	if err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}
	return s, nil
}

func checkTLS(cfg *config.Config) error {
	t := cfg.Global.TLS
	fe := t.Frontend.Server
	var errs []error
	if !fe.RequireClientAuth {
		errs = append(errs, errors.New("global.tls.frontend.server.requireClientAuth must be true"))
	}
	if len(fe.ClientCAFiles) == 0 && len(fe.ClientCAData) == 0 {
		errs = append(errs, errors.New("global.tls.frontend.server needs the installation CA (clientCaFiles)"))
	}
	if fe.CertFile == "" && fe.CertData == "" {
		errs = append(errs, errors.New("global.tls.frontend.server needs a certificate"))
	}
	if !t.Internode.Server.RequireClientAuth {
		errs = append(errs, errors.New("global.tls.internode.server.requireClientAuth must be true"))
	}
	if t.SystemWorker.CertFile == "" && t.SystemWorker.CertData == "" {
		errs = append(errs, errors.New("global.tls.systemWorker needs the temporal-system certificate"))
	}
	if cfg.Global.Authorization.Authorizer != "" || cfg.Global.Authorization.ClaimMapper != "" {
		errs = append(errs, errors.New("global.authorization must be empty: the nava Authorizer and ClaimMapper are compiled in"))
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("refusing to start: %w", err)
	}
	return nil
}
