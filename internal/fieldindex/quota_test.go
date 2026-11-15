package fieldindex_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
)

func TestPerTypeLimitIsTheD12Default(t *testing.T) {
	got, err := fieldindex.PerTypeLimit(context.Background(), nil, uuid.New())
	if err != nil || got != 5 || fieldindex.DefaultPerTypeLimit != 5 {
		t.Fatalf("PerTypeLimit = %d, %v, want 5", got, err)
	}
	if fieldindex.TenantLimit != 50 {
		t.Fatalf("TenantLimit = %d, want 50", fieldindex.TenantLimit)
	}
}
