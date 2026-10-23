// Command server runs Temporal 1.29.7 with nava's deny-by-default Authorizer and
// certificate ClaimMapper (ADR-001 D22, Annex A.2).
//
// Two helper subcommands run from the same image: `health` (a probe that dials
// with a client certificate) and `proxy` (the TCP gateway that is the only way
// from the platform network to the frontend).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	if len(os.Args) > 1 {
		var err error
		switch os.Args[1] {
		case "health":
			err = runHealth(os.Args[2:], os.Stdout)
		case "proxy":
			err = runProxy(os.Args[2:])
		default:
			serve()
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "temporal-server:", err)
			os.Exit(1)
		}
		return
	}
	serve()
}

func serve() {
	configDir := flag.String("config", "/etc/temporal/config", "config directory")
	env := flag.String("env", "development", "config environment (file <env>.yaml)")
	zone := flag.String("zone", "", "config zone")
	boot := flag.Bool("bootstrap", false, "register the nava namespace and the NavaTenantId search attribute once the frontend serves")
	flag.Parse()

	if err := run(*configDir, *env, *zone, *boot); err != nil {
		fmt.Fprintln(os.Stderr, "temporal-server:", err)
		os.Exit(1)
	}
}

func run(configDir, env, zone string, boot bool) error {
	cfg, err := config.LoadConfig(env, configDir, zone)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger := log.NewZapLogger(log.BuildZapLogger(cfg.Log))

	dc, err := dynamicConfigClient(cfg, configDir, logger, temporal.InterruptCh())
	if err != nil {
		return err
	}

	s, err := newServer(cfg, logger, dc, temporal.InterruptOn(temporal.InterruptCh()))
	if err != nil {
		return err
	}
	if boot {
		// The temporal-system certificate lives only in this container, so the
		// server registers its own namespace and search attribute.
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := bootstrap(ctx, cfg, logger); err != nil {
				logger.Error("bootstrap failed", tag.Error(err))
			}
		}()
	}
	logger.Info("Starting Temporal server with nava Authorizer", tag.NewStringTag("namespace", authz.Namespace))
	// Start blocks until interrupted.
	return s.Start()
}

// dynamicConfigClient reads the dynamic config file the config names, relative
// to configDir. With none named there are no values, and checkSecurityConfig
// refuses to start (D6-K needs explicit values).
func dynamicConfigClient(cfg *config.Config, configDir string, logger log.Logger, done <-chan interface{}) (dynamicconfig.Client, error) {
	if cfg.DynamicConfigClient == nil {
		return dynamicconfig.NewNoopClient(), nil
	}
	if !filepath.IsAbs(cfg.DynamicConfigClient.Filepath) {
		cfg.DynamicConfigClient.Filepath = filepath.Join(configDir, cfg.DynamicConfigClient.Filepath)
	}
	dc, err := dynamicconfig.NewFileBasedClient(cfg.DynamicConfigClient, logger, done)
	if err != nil {
		return nil, fmt.Errorf("dynamic config: %w", err)
	}
	return dc, nil
}

// newServer builds the server with the nava Authorizer and ClaimMapper. It
// refuses a config that weakens them (see checkSecurityConfig).
func newServer(cfg *config.Config, logger log.Logger, dc dynamicconfig.Client, extra ...temporal.ServerOption) (temporal.Server, error) {
	if err := checkSecurityConfig(cfg, dc); err != nil {
		return nil, err
	}
	dc = guardedClient{dc}
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
