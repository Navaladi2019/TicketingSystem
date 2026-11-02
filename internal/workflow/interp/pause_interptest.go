//go:build interptest

package interp

import (
	"context"
	"os"
)

// PauseDirEnv turns the pause-after-commit hook on in an interptest build:
// with it set to a directory, ApplyStep, after RecordWriter.Write returned
// (committed), writes PauseMarker(dir, key) and blocks until
// ReleaseFile(dir, key) exists or the activity's context is done.
const PauseDirEnv = "NAVA_INTERPTEST_PAUSE_DIR"

func afterCommit(ctx context.Context, key string) {
	if dir := os.Getenv(PauseDirEnv); dir != "" {
		pauseInDir(ctx, dir, key)
	}
}
