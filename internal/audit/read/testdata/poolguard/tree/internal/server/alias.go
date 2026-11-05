package server

import (
	"context"

	rd "github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	cfgpkg "github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

type holder struct {
	reader cfgpkg.AuditReader
}

func aliased(ctx context.Context, c cfgpkg.AuditReader, h *holder) {
	_, _ = rd.OpenPool(ctx, "postgres://x") // want: openpool
	_ = c.DatabaseURL                       // want: dsnfield
	_ = h.reader.DatabaseURL                // want: dsnfield
	_ = cfgpkg.AuditReaderDatabaseURLKey    // want: key
}
