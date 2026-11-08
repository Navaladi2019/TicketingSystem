//go:build integration

package dbtest

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// faultTracer fails one statement on a real connection: the nth statement
// (from 1) that match accepts runs with an already-cancelled context, so it
// fails with context.Canceled the way a dropped connection or a cancelled
// job would, while every other statement runs normally. It is a pgx
// QueryTracer set on the connection config, so the code under test runs
// unchanged against a real PostgreSQL.
//
// Where the error surfaces depends on pgx's statement cache. The first time
// a connection runs a statement, pgx prepares it, so the prepare (and the
// Exec, Query or QueryRow) fails. A statement the connection has already
// prepared is sent at once, so Query returns rows and the failure comes from
// rows.Err (pgx.CollectRows). With the config's DefaultQueryExecMode set to
// pgx.QueryExecModeExec nothing is prepared and every Query fails that way.
type faultTracer struct {
	match func(sql string) bool
	nth   int

	mu   sync.Mutex
	seen int
	hits []string
}

func (f *faultTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if !f.match(d.SQL) {
		return ctx
	}
	f.mu.Lock()
	f.seen++
	hit := f.seen == f.nth
	if hit {
		f.hits = append(f.hits, d.SQL)
	}
	f.mu.Unlock()
	if !hit {
		return ctx
	}
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	return cctx
}

func (f *faultTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// fired reports whether the tracer failed a statement.
func (f *faultTracer) fired() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.hits) > 0
}

// sqlHas matches a statement containing every part.
func sqlHas(parts ...string) func(string) bool {
	return func(sql string) bool {
		for _, p := range parts {
			if !strings.Contains(sql, p) {
				return false
			}
		}
		return true
	}
}

// sqlIs matches a statement equal to s once surrounding space is trimmed.
func sqlIs(s string) func(string) bool {
	return func(sql string) bool { return strings.TrimSpace(sql) == s }
}

// withFault is cfg with a faultTracer failing the nth statement match accepts.
func withFault(cfg *pgx.ConnConfig, match func(string) bool, nth int) (*pgx.ConnConfig, *faultTracer) {
	f := &faultTracer{match: match, nth: nth}
	cfg = cfg.Copy()
	cfg.Tracer = f
	return cfg, f
}

// failNthConnect is cfg whose nth connection (from 1) fails after the
// server accepted it, as a server at max_connections or a restart would.
func failNthConnect(cfg *pgx.ConnConfig, nth int) *pgx.ConnConfig {
	cfg = cfg.Copy()
	var mu sync.Mutex
	n := 0
	cfg.ValidateConnect = func(context.Context, *pgconn.PgConn) error {
		mu.Lock()
		defer mu.Unlock()
		n++
		if n == nth {
			return errConnectRefused
		}
		return nil
	}
	return cfg
}

var errConnectRefused = errors.New("injected: connection refused")

// wantFault fails t unless err is the tracer's injected cancellation, wrapped
// with every one of wrap, and the tracer fired.
func wantFault(t *testing.T, err error, f *faultTracer, wrap ...string) {
	t.Helper()
	if !f.fired() {
		t.Fatalf("the fault never fired (no statement matched %d times); err = %v", f.nth, err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the injected context.Canceled", err)
	}
	for _, w := range wrap {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("err = %q, want it to contain %q", err, w)
		}
	}
}
