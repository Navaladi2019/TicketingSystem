package server_test

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
)

// Test files are covered too; only the listed ones are exempt.
func TestPlanted(t *testing.T) {
	_, _ = read.OpenPool(t.Context(), "postgres://x") // want: openpool
	t.Setenv("NAVA_AUDIT_READER_DATABASE_URL", "x")   // want: env
}
