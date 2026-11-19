package backfill

import "context"

// SetAfterSnapshot installs the test-only hook that runs between the
// barrier's snapshot and its fence.
func SetAfterSnapshot(a *Activities, f func(context.Context)) { a.afterSnapshot = f }

// NextIDsSQL is the batch's keyset step (b), for the plan test (TICK-990).
const NextIDsSQL = nextIDsSQL
