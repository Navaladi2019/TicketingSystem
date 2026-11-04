package main

import (
	"context"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// Two OpenPool calls in main.go: neither is the one allowed call.
func run(ctx context.Context, lookup config.LookupFunc) {
	readerCfg, _ := config.LoadAuditReader(lookup)
	_, _ = read.OpenPool(ctx, readerCfg.DatabaseURL) // want: openpool dsnfield
	_, _ = read.OpenPool(ctx, readerCfg.DatabaseURL) // want: openpool dsnfield
}
