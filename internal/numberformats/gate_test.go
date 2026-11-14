package numberformats

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGateCachesClosedForTTLThenReprobes(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	open, calls := false, 0
	g := NewGate(func(context.Context, Querier) (bool, error) { calls++; return open, nil }, func() time.Time { return now })
	ctx := context.Background()

	if err := g.Check(ctx, nil); !errors.Is(err, ErrTemplatesUnavailable) {
		t.Fatalf("before M3 = %v, want ErrTemplatesUnavailable", err)
	}
	open = true // M3 runs
	now = now.Add(GateTTL - time.Second)
	if err := g.Check(ctx, nil); !errors.Is(err, ErrTemplatesUnavailable) || calls != 1 {
		t.Fatalf("inside the TTL = %v after %d probes, want refused from the cache", err, calls)
	}
	now = now.Add(time.Second)
	if err := g.Check(ctx, nil); err != nil || calls != 2 {
		t.Fatalf("at the TTL = %v after %d probes, want open after a re-read", err, calls)
	}
	open = false
	if err := g.Check(ctx, nil); err != nil || calls != 2 {
		t.Fatalf("an open gate stays open without a probe: %v, %d probes", err, calls)
	}
}

func TestGateProbeFailureIsNotCached(t *testing.T) {
	t.Parallel()
	boom := errors.New("catalog down")
	fail := true
	g := NewGate(func(context.Context, Querier) (bool, error) {
		if fail {
			return false, boom
		}
		return true, nil
	}, nil)
	if err := g.Check(context.Background(), nil); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the probe's", err)
	}
	fail = false
	if err := g.Check(context.Background(), nil); err != nil {
		t.Fatalf("after recovery = %v, want open", err)
	}
}

func TestNewGateDefaultsToTheCatalogAndTheClock(t *testing.T) {
	t.Parallel()
	g := NewGate(nil, nil)
	if g.probe == nil || g.now == nil || g.ttl != GateTTL {
		t.Errorf("defaults not set: %+v", g)
	}
}
