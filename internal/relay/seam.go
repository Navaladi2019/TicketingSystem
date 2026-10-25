package relay

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
)

// PauseMarker starts the line StdinPause writes when it pauses: the marker, a
// space and the outbox id. T33's crash test waits for it on the relay's
// stderr, then kills the process.
const PauseMarker = "relaytest: paused before StartWorkflow"

// StdinPause returns a crash seam that pauses each start between its claim and
// its StartWorkflow call. It writes PauseMarker and the row's outbox id as one
// line to w, then blocks until a line arrives on r (one line releases one
// paused start) or ctx is done. Only a binary built with the relaytest tag
// installs it (see crashSeam); docs/architecture/outbox-relay.md shows how to
// drive it.
func StdinPause(r io.Reader, w io.Writer) func(context.Context, Claimed) {
	release := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	return func(ctx context.Context, c Claimed) {
		once.Do(func() {
			go func() {
				s := bufio.NewScanner(r)
				for s.Scan() {
					release <- struct{}{}
				}
			}()
		})
		mu.Lock()
		_, _ = fmt.Fprintf(w, "%s %s\n", PauseMarker, c.ID)
		mu.Unlock()
		select {
		case <-release:
		case <-ctx.Done():
		}
	}
}
