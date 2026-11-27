package fieldindex

import (
	"math/rand/v2"
	"testing"
	"time"
)

// pauses feeds ds to a fresh pacer and returns the pause after each.
func pauses(p *Pacer, ds ...time.Duration) []time.Duration {
	out := make([]time.Duration, len(ds))
	for i, d := range ds {
		out[i] = p.Next(d)
	}
	return out
}

func flat(d time.Duration, n int) []time.Duration {
	out := make([]time.Duration, n)
	for i := range out {
		out[i] = d
	}
	return out
}

// A noisy but uncontended series, ±30% around a flat mean, stays at the
// floor: the case that ratcheted the old pacer to its cap (TICK-991).
func TestPacerNoiseStaysAtTheFloor(t *testing.T) {
	for seed := range uint64(10) {
		r := rand.New(rand.NewPCG(seed, 991))
		ds := make([]time.Duration, 2000)
		for i := range ds {
			ds[i] = time.Duration(float64(400*time.Millisecond) * (0.7 + 0.6*r.Float64()))
		}
		atFloor := 0
		for _, got := range pauses(&Pacer{}, ds...) {
			if got == InitialPause {
				atFloor++
			}
		}
		if atFloor < len(ds)*99/100 {
			t.Errorf("seed %d: %d of %d pauses at the floor, want at least 99%%", seed, atFloor, len(ds))
		}
	}
}

// Slow drift, +50% over 300 batches with no contention, is followed, not
// treated as contention.
func TestPacerFollowsDrift(t *testing.T) {
	ds := make([]time.Duration, 300)
	for i := range ds {
		ds[i] = 400*time.Millisecond + time.Duration(i)*200*time.Millisecond/300
	}
	for i, got := range pauses(&Pacer{}, ds...) {
		if got != InitialPause {
			t.Fatalf("batch %d: pause %v under slow drift, want the floor", i, got)
		}
	}
}

// Contention doubles the batch: the pause backs off to the cap; when it
// ends, the pause comes back to the floor.
func TestPacerBacksOffAndRecovers(t *testing.T) {
	var p Pacer
	pauses(&p, flat(400*time.Millisecond, 50)...)
	if p.pause != InitialPause {
		t.Fatalf("pause %v before contention, want the floor", p.pause)
	}
	got := pauses(&p, flat(800*time.Millisecond, 15)...)
	if got[1] != InitialPause {
		t.Errorf("pause %v at the second contended batch, want the floor: two slow batches are not contention", got[1])
	}
	if got[len(got)-1] != MaxPause {
		t.Errorf("pauses %v under contention, want them to reach the cap %v", got, MaxPause)
	}
	got = pauses(&p, flat(400*time.Millisecond, 10)...)
	if got[len(got)-1] != InitialPause {
		t.Errorf("pauses %v after contention ended, want the floor", got)
	}
	// A zero duration (no clock movement) changes nothing.
	p.pause = 2 * time.Second
	if got := p.Next(0); got != 2*time.Second {
		t.Fatalf("pause %v for a zero duration, want it held", got)
	}
}

// The pause never leaves [InitialPause, MaxPause], and the cap is well below
// the 10 s that made a 400k-record field take an hour.
func TestPacerBounds(t *testing.T) {
	if MaxPause > 5*time.Second {
		t.Fatalf("MaxPause %v, want at most 5 s (TICK-991)", MaxPause)
	}
	var p Pacer
	for i, got := range pauses(&p, append(flat(time.Millisecond, 10), flat(time.Hour, 100)...)...) {
		if got < InitialPause || got > MaxPause {
			t.Fatalf("batch %d: pause %v out of bounds", i, got)
		}
	}
}
