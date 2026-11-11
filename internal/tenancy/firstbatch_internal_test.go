package tenancy

import (
	"testing"

	"github.com/google/uuid"
)

// TestFirstBatchLargeObjectsPanics pins TICK-698: LargeObjects on the
// transaction InTenantTx hands fn with BeginInFirstBatch (a scopedTx over a
// lazyTx, built as InTenantTx builds it) panics with the documented message,
// before anything is sent. navalint tenancyconn keeps callers away from it.
func TestFirstBatchLargeObjectsPanics(t *testing.T) {
	lazy := &lazyTx{tenant: uuid.New()}
	tx := &scopedTx{Tx: lazy, tenant: lazy.tenant, memo: &memo{m: map[any]any{}}}
	defer func() {
		got := recover()
		if got != "tenancy: LargeObjects is not supported on tenant transactions (BeginInFirstBatch)" {
			t.Errorf("LargeObjects panicked with %v; want the documented message", got)
		}
		if lazy.begun || lazy.closed {
			t.Errorf("LargeObjects changed the transaction: begun %v, closed %v", lazy.begun, lazy.closed)
		}
	}()
	tx.LargeObjects()
	t.Error("LargeObjects returned; want a panic")
}
