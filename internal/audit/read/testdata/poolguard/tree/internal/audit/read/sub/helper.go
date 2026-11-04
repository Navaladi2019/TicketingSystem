package sub

import (
	"context"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
)

// Inside internal/audit/read the pool may be opened, but the env name is
// still the loader's.
func open(ctx context.Context) {
	_, _ = read.OpenPool(ctx, "postgres://x")
	_ = "NAVA_AUDIT_READER_DATABASE_URL" // want: env
}
