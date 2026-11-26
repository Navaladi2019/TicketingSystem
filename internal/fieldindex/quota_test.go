package fieldindex_test

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
)

func TestQuotasAreTheD12Defaults(t *testing.T) {
	if fieldindex.DefaultPerTypeLimit != 5 {
		t.Fatalf("DefaultPerTypeLimit = %d, want 5", fieldindex.DefaultPerTypeLimit)
	}
	if fieldindex.TenantLimit != 50 {
		t.Fatalf("TenantLimit = %d, want 50", fieldindex.TenantLimit)
	}
}
