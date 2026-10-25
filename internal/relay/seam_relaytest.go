//go:build relaytest

package relay

import (
	"context"
	"os"
)

// PauseEnv turns the crash seam on in a relaytest build: with it set to 1,
// every start pauses at StdinPause on the process's stdin and stderr.
const PauseEnv = "NAVA_RELAYTEST_PAUSE"

// crashSeam is the T33 crash seam (relaytest builds only). The production
// image is built without the tag, so it has no pause and reads no variable.
//
//nolint:gochecknoglobals // set once at start, like the default build's func
var crashSeam = func() func(context.Context, Claimed) {
	if os.Getenv(PauseEnv) != "1" {
		return func(context.Context, Claimed) {}
	}
	return StdinPause(os.Stdin, os.Stderr)
}()
