package schemabackfill

import (
	"context"

	"github.com/google/uuid"
)

// SetAfterRecord installs the test-only hook that runs after each record's
// transaction has committed.
func SetAfterRecord(a *Activities, f func(ctx context.Context, record uuid.UUID) error) {
	a.afterRecord = f
}

// SetBeforeAdmit installs the test-only hook that runs before the pending →
// filling UPDATE.
func SetBeforeAdmit(a *Activities, f func(ctx context.Context)) { a.beforeAdmit = f }

// The job's reads, for the CI-3 (5) no-tenant check.
const (
	LockJobSQL = lockJobSQL
	PageSQL    = pageSQL
	FillSQL    = fillSQL
)
