package tenancy_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// firstBatch is the batch a BeginInFirstBatch transaction sends first, with
// one statement of the caller's.
func firstBatch(tenant any) *pgx.Batch {
	b := &pgx.Batch{}
	b.Queue("BEGIN")
	b.Queue("SELECT set_config('app.tenant_id', $1, true)", tenant)
	b.Queue("SELECT 1")
	return b
}

// FirstBatchTenant recognises only a first batch with a canonical tenant,
// never an AllocTx batch (which the Guard scopes as an allocation).
func TestFirstBatchTenant(t *testing.T) {
	id := uuid.New()
	if got, ok := tenancy.FirstBatchTenant(firstBatch(id.String())); !ok || got != id {
		t.Errorf("FirstBatchTenant = %s, %v; want %s, true", got, ok, id)
	}
	alloc := allocBatch(id.String())
	for name, b := range map[string]*pgx.Batch{
		"nil tenant":       firstBatch(uuid.Nil.String()),
		"uppercase tenant": firstBatch(strings.ToUpper(id.String())),
		"not a string":     firstBatch(id),
		"no tenant":        {QueuedQueries: firstBatch(id.String()).QueuedQueries[:1]},
		"allocation":       alloc,
		"empty":            {},
	} {
		if got, ok := tenancy.FirstBatchTenant(b); ok {
			t.Errorf("%s: FirstBatchTenant = %s, true; want false", name, got)
		}
	}
	other := firstBatch(id.String())
	other.QueuedQueries[0].SQL = "begin"
	if _, ok := tenancy.FirstBatchTenant(other); ok {
		t.Error("a batch opened by another BEGIN text was recognised")
	}
}
