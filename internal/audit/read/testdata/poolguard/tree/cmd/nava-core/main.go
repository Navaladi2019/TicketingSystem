package main

import (
	"context"
	"log"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func run(ctx context.Context, lookup config.LookupFunc) error {
	cfg, err := config.Load(lookup)
	if err != nil {
		return err
	}
	readerCfg, err := config.LoadAuditReader(lookup)
	if err != nil {
		return err
	}
	// Look-alike: the app's DSN field has the same name on another type.
	log.Print(cfg.Deps.DatabaseURL != "")
	log.Print(readerCfg.DatabaseURL) // want: dsnfield
	dsn := readerCfg.DatabaseURL     // want: dsnfield
	_ = dsn
	// The one allowed call, with the DSN field as its argument.
	p, err := read.OpenPool(ctx, readerCfg.DatabaseURL)
	if err != nil {
		return err
	}
	p.Close()
	return nil
}
