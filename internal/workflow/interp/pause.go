package interp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// The pause-after-commit fixture seam (P3-22a; P3-24's crash test uses it).
// Only an interptest build turns it on (pause_interptest.go); a normal build's
// afterCommit does nothing (pause_default.go), so the production worker never
// pauses and reads no variable.

// pausePoll is how often a paused ApplyStep looks for its release file.
const pausePoll = 20 * time.Millisecond

// pauseName names a key's files: the key holds a workflow id, which has ':'.
func pauseName(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:16])
}

// PauseMarker is the file a paused ApplyStep writes in dir; it holds the key.
func PauseMarker(dir, key string) string { return filepath.Join(dir, pauseName(key)+".paused") }

// ReleaseFile is the file a test creates in dir to release the paused
// ApplyStep for key.
func ReleaseFile(dir, key string) string { return filepath.Join(dir, pauseName(key)+".release") }

// pauseInDir writes PauseMarker(dir, key), then blocks until
// ReleaseFile(dir, key) exists or ctx is done.
func pauseInDir(ctx context.Context, dir, key string) {
	if err := writeMarker(dir, key); err != nil {
		return
	}
	t := time.NewTicker(pausePoll)
	defer t.Stop()
	for {
		if _, err := os.Stat(ReleaseFile(dir, key)); err == nil || !errors.Is(err, fs.ErrNotExist) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// writeMarker publishes PauseMarker(dir, key) atomically: it writes the key to
// a temporary file in dir and renames it into place, so a poller that sees the
// marker always reads the whole key. os.WriteFile would create the marker
// empty and fill it afterwards, and a poller could read it in between
// (TICK-532).
func writeMarker(dir, key string) error {
	tmp, err := os.CreateTemp(dir, pauseName(key)+".*.tmp")
	if err != nil {
		return err
	}
	_, werr := tmp.WriteString(key)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), PauseMarker(dir, key)); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}
