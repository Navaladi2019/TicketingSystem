package interp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"
)

// closed reports whether ch is closed, without blocking.
func closed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// TestPauseAfterCommit: the hook writes its marker and blocks until the
// release file exists, or its context ends. It runs in a synctest bubble:
// synctest.Wait returns once the hook is parked on its poll ticker, and the
// fake clock decides exactly which poll sees the release.
func TestPauseAfterCommit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		afterCommit := func(ctx context.Context, key string) { pauseInDir(ctx, dir, key) }
		const key = "wf:t:0192f3a0-0000-7000-8000-00000000000a:interp:x:step"

		done := make(chan struct{})
		go func() { afterCommit(context.Background(), key); close(done) }()
		synctest.Wait()
		if b, err := os.ReadFile(PauseMarker(dir, key)); err != nil || string(b) != key {
			t.Fatalf("marker %q (%v), want the key", b, err)
		}
		// Fifty polls without a release file: still paused.
		advance(50 * pausePoll)
		if closed(done) {
			t.Fatal("the hook returned before its release")
		}
		if err := os.WriteFile(ReleaseFile(dir, key), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		// The release is seen on the next poll, not before.
		advance(pausePoll - time.Nanosecond)
		if closed(done) {
			t.Fatal("the hook returned between polls")
		}
		advance(time.Nanosecond)
		if !closed(done) {
			t.Fatal("the hook did not return on the poll after its release")
		}

		ctx, cancel := context.WithCancel(context.Background())
		other := make(chan struct{})
		go func() { afterCommit(ctx, key+"2"); close(other) }()
		synctest.Wait()
		if _, err := os.Stat(PauseMarker(dir, key+"2")); err != nil {
			t.Fatalf("second marker: %v", err)
		}
		if closed(other) {
			t.Fatal("the second hook returned before its release")
		}
		cancel()
		synctest.Wait()
		if !closed(other) {
			t.Fatal("a cancelled context did not release the hook")
		}
	})
}

// advance moves the bubble's fake clock on by d, then waits until every
// goroutine in the bubble is blocked again. It takes no wall-clock time.
func advance(d time.Duration) {
	<-time.After(d)
	synctest.Wait()
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

// TestPauseMarkerAppearsWhole: a reader that sees the marker sees the whole
// key. The marker is the "committed and paused" signal both this package's
// tests and P3-24's crash test poll for, so it must never be visible empty or
// half-written (TICK-532: os.WriteFile creates the file before it writes it).
func TestPauseMarkerAppearsWhole(t *testing.T) {
	dir := t.TempDir()
	for i := range 200 {
		key := fmt.Sprintf("wf:t:0192f3a0-0000-7000-8000-00000000000a:interp:x:step%d", i)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { pauseInDir(ctx, dir, key); close(done) }()
		deadline := time.Now().Add(10 * time.Second)
		for {
			b, err := os.ReadFile(PauseMarker(dir, key))
			if err == nil {
				if string(b) != key {
					cancel()
					<-done
					t.Fatalf("run %d: marker read %q, want the key", i, b)
				}
				break
			}
			// Not there yet, or (Windows) briefly locked by the rename that
			// is publishing it: either way, no content was observed.
			if time.Now().After(deadline) {
				t.Fatalf("run %d: no readable marker: %v", i, err)
			}
		}
		cancel()
		<-done
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".paused" {
			t.Errorf("stray file %s left in the pause dir", e.Name())
		}
	}
}
