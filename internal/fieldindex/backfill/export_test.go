package backfill

import "context"

// SetAfterSnapshot installs the test-only hook that runs between the
// barrier's snapshot and its fence.
func SetAfterSnapshot(a *Activities, f func(context.Context)) { a.afterSnapshot = f }
