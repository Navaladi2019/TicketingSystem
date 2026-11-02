//go:build !interptest

package interp

import "context"

// afterCommit does nothing in a normal build. The interptest build replaces
// it with the pause-after-commit hook (pause_interptest.go).
func afterCommit(context.Context, string) {}
