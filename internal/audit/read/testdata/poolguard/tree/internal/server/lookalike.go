package server

import (
	"context"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/pgutil"
)

// Look-alikes: another package's OpenPool, a local type's DatabaseURL, the
// env name in a comment (NAVA_AUDIT_READER_DATABASE_URL), a method named
// OpenPool.

type appConfig struct{ DatabaseURL string }

type opener struct{}

func (opener) OpenPool() {}

func lookalikes(ctx context.Context, c appConfig) {
	pgutil.OpenPool(ctx)
	_ = c.DatabaseURL
	opener{}.OpenPool()
	_ = struct{ OpenPool int }{OpenPool: 1}
	_ = "NAVA_AUDIT_READER_URL"
}
