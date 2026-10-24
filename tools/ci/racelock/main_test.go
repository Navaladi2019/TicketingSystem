package main

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"
)

// freePorts returns the first of n consecutive loopback ports that are free
// now.
func freePorts(t *testing.T, n int) int {
	t.Helper()
	for base := 47700; base < 48700; base += n {
		ok := true
		for i := range n {
			l, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:"+strconv.Itoa(base+i))
			if err != nil {
				ok = false
				break
			}
			_ = l.Close()
		}
		if ok {
			return base
		}
	}
	t.Fatalf("no %d free consecutive ports", n)
	return 0
}

func hold(t *testing.T, port int) net.Listener {
	t.Helper()
	l, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// acquireAsync runs acquire in the background and returns its result channel
// and the reports it made.
func acquireAsync(port, slots, take int) (<-chan []net.Listener, chan time.Duration) {
	got := make(chan []net.Listener, 1)
	reports := make(chan time.Duration, 100)
	go func() {
		got <- acquire(port, slots, take, 100*time.Millisecond, func(d time.Duration) { reports <- d })
	}()
	return got, reports
}

func release(ls []net.Listener) {
	for _, l := range ls {
		_ = l.Close()
	}
}

// The race lock: a second holder waits until the first releases it, and
// reports while it waits.
func TestAcquireWaitsForHolder(t *testing.T) {
	port := freePorts(t, 1)
	first := hold(t, port)
	got, reports := acquireAsync(port, 1, 1)
	select {
	case ls := <-got:
		release(ls)
		t.Fatal("acquired while the first holder had the lock")
	case <-time.After(time.Second):
	}
	if len(reports) == 0 {
		t.Fatal("no waiting report after a second")
	}
	_ = first.Close()
	select {
	case ls := <-got:
		release(ls)
	case <-time.After(5 * time.Second):
		t.Fatal("not acquired within 5s of the release")
	}
}

// Stack slots: a free slot is taken at once, a run that needs two waits while
// only one is free and holds none of them meanwhile, and gets both once a
// second one frees.
func TestAcquireSlots(t *testing.T) {
	port := freePorts(t, 3)
	busy := hold(t, port)
	one := acquire(port, 3, 1, time.Second, func(time.Duration) {})
	if len(one) != 1 || one[0].Addr().(*net.TCPAddr).Port != port+1 {
		t.Fatalf("one slot: got %v, want port %d", one, port+1)
	}

	got, _ := acquireAsync(port, 3, 2)
	select {
	case ls := <-got:
		release(ls)
		t.Fatal("took two slots while only one was free")
	case <-time.After(time.Second):
	}
	// The waiting run must not sit on the one free slot. It binds and releases
	// it for a moment on every retry, so one failed bind is not proof: a slot
	// it sat on would refuse every attempt.
	free := false
	for range 20 {
		if l, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:"+strconv.Itoa(port+2)); err == nil {
			_ = l.Close()
			free = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !free {
		t.Fatal("the waiting run holds the one free slot")
	}

	_ = busy.Close()
	select {
	case ls := <-got:
		if len(ls) != 2 {
			t.Fatalf("got %d slots, want 2", len(ls))
		}
		release(ls)
	case <-time.After(5 * time.Second):
		t.Fatal("two slots not acquired within 5s of one freeing")
	}
	release(one)
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
