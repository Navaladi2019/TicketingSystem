package interp

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// waitFor polls until ok or the deadline; the hook runs on another goroutine.
func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestPauseAfterCommit: the hook writes its marker and blocks until the
// release file exists, or its context ends.
func TestPauseAfterCommit(t *testing.T) {
	dir := t.TempDir()
	afterCommit := func(ctx context.Context, key string) { pauseInDir(ctx, dir, key) }
	const key = "wf:t:0192f3a0-0000-7000-8000-00000000000a:interp:x:step"

	done := make(chan struct{})
	go func() { afterCommit(context.Background(), key); close(done) }()
	waitFor(t, func() bool { _, err := os.Stat(PauseMarker(dir, key)); return err == nil })
	if b, err := os.ReadFile(PauseMarker(dir, key)); err != nil || string(b) != key {
		t.Fatalf("marker %q (%v), want the key", b, err)
	}
	select {
	case <-done:
		t.Fatal("the hook returned before its release")
	case <-time.After(100 * time.Millisecond):
	}
	if err := os.WriteFile(ReleaseFile(dir, key), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	other := make(chan struct{})
	go func() { afterCommit(ctx, key+"2"); close(other) }()
	waitFor(t, func() bool { _, err := os.Stat(PauseMarker(dir, key+"2")); return err == nil })
	cancel()
	waitFor(t, func() bool {
		select {
		case <-other:
			return true
		default:
			return false
		}
	})
}

// TestPauseWithoutMarkerDoesNotBlock: when the marker cannot be written (its
// directory is gone), the hook returns at once instead of waiting for a
// release nobody can see it needs.
func TestPauseWithoutMarkerDoesNotBlock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	const key = "wf:t:0192f3a0-0000-7000-8000-00000000000a:interp:x:step"
	done := make(chan struct{})
	go func() { pauseInDir(context.Background(), dir, key); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the hook blocked without a marker")
	}
	if _, err := os.Stat(PauseMarker(dir, key)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("marker stat: %v, want not exist", err)
	}
}
