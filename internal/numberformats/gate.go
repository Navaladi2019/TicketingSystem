package numberformats

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// GateTTL is how long a Gate keeps an answer (ADR-005 §3.7 [TA-5]): a
// process started before M3 opens within this time of M3 running, without a
// restart.
const GateTTL = 60 * time.Second

// ErrTemplatesUnavailable means record_counters_pkey does not yet cover
// scope_key (M3, TICK-670), so a format other than the default cannot be
// published: its counters would collide on the old key.
var ErrTemplatesUnavailable = errors.New("numberformats: numbering templates are unavailable until M3 has run")

// scopeKeyInPK reads the catalog: does record_counters_pkey include
// scope_key. A primary key on the old two columns, or no such constraint,
// is false.
const scopeKeyInPK = `
SELECT EXISTS (
  SELECT 1
  FROM pg_constraint c
  JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
  WHERE c.conname = 'record_counters_pkey'
    AND c.conrelid = 'public.record_counters'::regclass
    AND a.attname = 'scope_key')`

// Probe reports whether record_counters_pkey covers scope_key.
type Probe func(ctx context.Context, q Querier) (bool, error)

// Querier is the part of a pool or transaction the probe needs.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// CatalogProbe is the production Probe: one catalog query.
func CatalogProbe(ctx context.Context, q Querier) (bool, error) {
	var open bool
	if err := q.QueryRow(ctx, scopeKeyInPK).Scan(&open); err != nil {
		return false, fmt.Errorf("numberformats: read record_counters_pkey: %w", err)
	}
	return open, nil
}

// Gate is the publish gate (ADR-005 §3.2): it refuses a non-default format
// while M3 has not run. An "open" answer is final for the process, since a
// primary key does not lose a column; a "closed" answer is re-read after
// GateTTL, so the process notices M3 without a restart. Safe for concurrent
// use.
type Gate struct {
	probe Probe
	now   func() time.Time
	ttl   time.Duration

	mu        sync.Mutex
	open      bool
	checkedAt time.Time
	checked   bool
}

// NewGate returns a gate reading the catalog. now is time.Now when nil.
func NewGate(probe Probe, now func() time.Time) *Gate {
	if probe == nil {
		probe = CatalogProbe
	}
	if now == nil {
		now = time.Now
	}
	return &Gate{probe: probe, now: now, ttl: GateTTL}
}

// Check returns ErrTemplatesUnavailable while the key does not cover
// scope_key. A probe failure is returned as is and cached nowhere.
func (g *Gate) Check(ctx context.Context, q Querier) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open {
		return nil
	}
	if g.checked && g.now().Sub(g.checkedAt) < g.ttl {
		return ErrTemplatesUnavailable
	}
	open, err := g.probe(ctx, q)
	if err != nil {
		return err
	}
	g.checked, g.checkedAt, g.open = true, g.now(), open
	if !open {
		return ErrTemplatesUnavailable
	}
	return nil
}
