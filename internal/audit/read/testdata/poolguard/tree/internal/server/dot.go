package server

import (
	"context"

	. "github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	. "github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func dotted(ctx context.Context, lookup LookupFunc) {
	_, _ = OpenPool(ctx, "postgres://x") // want: openpool
	_ = AuditReaderDatabaseURLKey        // want: key
	r, _ := LoadAuditReader(lookup)
	_ = r.DatabaseURL // want: dsnfield
}
