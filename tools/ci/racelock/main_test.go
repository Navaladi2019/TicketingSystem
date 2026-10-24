package main

import (
	"net"
	"testing"
	"time"
)

// A second holder waits until the first releases the lock, and reports
// while it waits.
func TestAcquireWaitsForHolder(t *testing.T) {
	first, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := first.Addr().String()
	got := make(chan net.Listener)
	reports := make(chan time.Duration, 100)
	go func() { got <- acquire(addr, 100*time.Millisecond, func(d time.Duration) { reports <- d }) }()

	select {
	case l := <-got:
		_ = l.Close()
		t.Fatal("acquired while the first holder had the lock")
	case <-time.After(time.Second):
	}
	if len(reports) == 0 {
		t.Fatal("no waiting report after a second")
	}
	_ = first.Close()
	select {
	case l := <-got:
		_ = l.Close()
	case <-time.After(5 * time.Second):
		t.Fatal("not acquired within 5s of the release")
	}
}

func TestRunExitStatus(t *testing.T) {
	if code := run([]string{"go", "version"}); code != 0 {
		t.Fatalf("go version: exit %d", code)
	}
	if code := run([]string{"go", "no-such-command"}); code == 0 {
		t.Fatal("go no-such-command: exit 0")
	}
	if code := run([]string{"no-such-binary-racelock"}); code != 1 {
		t.Fatalf("missing binary: exit %d, want 1", code)
	}
}
