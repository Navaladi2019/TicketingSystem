//go:build interptest

package interp

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestPauseHookReadsItsDir: in an interptest build, PauseDirEnv turns the
// hook on; unset, it returns at once.
func TestPauseHookReadsItsDir(t *testing.T) {
	t.Setenv(PauseDirEnv, "")
	afterCommit(context.Background(), "k")

	dir := t.TempDir()
	t.Setenv(PauseDirEnv, dir)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	afterCommit(ctx, "k")
	if _, err := os.Stat(PauseMarker(dir, "k")); err != nil {
		t.Errorf("no marker: %v", err)
	}
}
