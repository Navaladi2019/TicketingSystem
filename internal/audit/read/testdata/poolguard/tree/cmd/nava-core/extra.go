package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// NAVA_AUDIT_READER_DATABASE_URL in a comment is a look-alike.

const (
	envPart1 = "NAVA_AUDIT_READER_"
	envPart2 = "DATABASE_URL"
)

// A second OpenPool call in cmd/nava-core, outside main.go.
func second(ctx context.Context) {
	_, _ = read.OpenPool(ctx, "postgres://x") // want: openpool
}

func envNames() {
	_ = os.Getenv("NAVA_AUDIT_READER_DATABASE_URL") // want: env
	const local = "NAVA_AUDIT_READER_DATABASE_URL"  // want: env
	_ = os.Getenv(local)
	_ = os.Getenv(envPart1 + envPart2)                  // want: env
	_ = fmt.Sprintf("%s%s", envPart1, envPart2)         // want: env
	_ = os.Getenv(config.AuditReaderDatabaseURLKey)     // want: key
	_ = config.AuditReader{DatabaseURL: os.Getenv("X")} // want: dsnfield
	_ = os.Getenv("NAVA_DATABASE_URL")
	_ = os.Getenv(envPart1 + "PASSWORD")
}
