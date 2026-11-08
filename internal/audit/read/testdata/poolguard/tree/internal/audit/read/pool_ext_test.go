package read_test

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
)

func TestOpen(t *testing.T) {
	_, _ = read.OpenPool(t.Context(), "postgres://x")
}
