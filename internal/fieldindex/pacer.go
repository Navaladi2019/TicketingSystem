package fieldindex

import (
	"slices"
	"time"
)

// The bounds of Pacer (D10.4, CTO ruling 5, PLAN-007 D-7-15).
const (
	// InitialPause is the first pause, and the floor: the spike measured
	// +31% live-write p95 at 200 ms.
	InitialPause = 500 * time.Millisecond
	// MaxPause caps the pause. One backfill runs per tenant at a time, so a
	// batch of a few hundred milliseconds then this pause keeps the
	// backfill's share of the database small (TICK-991: a 10 s cap made a
	// 400k-record field take about an hour).
	MaxPause = 4 * time.Second
)

// Pacer is the adaptive pause between pages of background work (ADR-006
// D10.4, D-7-15), built by S21 for the field-index backfill and reused by the
// schema backfill (ADR-009 D7). The zero value is ready to use. Its only input is each
// batch's duration as the workflow measured it, workflow.Now() around the
// activity, so a replay computes the same pauses.
//
// The signal is the median of the last pacerWindow batches, so one slow or
// fast batch moves nothing; until the window is full the pause stays at
// InitialPause. The baseline starts at the first full window's signal and
// then follows the signal by at most 1/pacerCreep a batch, up or down. It is
// not the fastest batch of the run: the first batches are the fastest and
// later ones drift (cache, checkpoints), which ratcheted the pause to its
// cap (TICK-991). A signal more than 1.5× the baseline means the batches
// compete with live writes and the pause doubles, up to MaxPause; a signal
// within 1.25× of it halves the pause, down to InitialPause; in between it
// holds. A sudden doubling backs off for about 18 batches, holds while the
// baseline follows it, and eases after about 30; when the contention ends
// the pause halves at once, the rate it went up.
type Pacer struct {
	pause, baseline time.Duration
	recent          [pacerWindow]time.Duration
	n               int
}

const (
	pacerWindow = 5
	pacerCreep  = 64
)

func (p *Pacer) Next(d time.Duration) time.Duration {
	if p.pause == 0 {
		p.pause = InitialPause
	}
	if d <= 0 {
		return p.pause
	}
	p.recent[p.n%pacerWindow] = d
	p.n++
	if p.n < pacerWindow {
		return p.pause
	}
	s := median(p.recent[:])
	if p.baseline == 0 {
		p.baseline = s
	} else {
		step := p.baseline / pacerCreep
		p.baseline = min(max(s, p.baseline-step), p.baseline+step)
	}
	switch {
	case s*2 > p.baseline*3:
		p.pause = min(p.pause*2, MaxPause)
	case s*4 <= p.baseline*5:
		p.pause = max(p.pause/2, InitialPause)
	}
	return p.pause
}

// median is the middle of ds, the upper middle for an even count. ds is
// not changed.
func median(ds []time.Duration) time.Duration {
	sorted := slices.Clone(ds)
	slices.Sort(sorted)
	return sorted[len(sorted)/2]
}
