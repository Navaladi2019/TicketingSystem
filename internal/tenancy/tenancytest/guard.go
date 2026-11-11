// Package tenancytest checks, from the driver's side, that every statement a
// test drives runs inside a transaction whose BEGIN set the tenant (TICK-648).
//
// Guard is a pgx tracer for a test pool. It follows each connection: the
// tenancy BEGIN, alone or opening a batch (tenancy.BeginInFirstBatch), marks
// it scoped to a tenant, COMMIT or ROLLBACK (alone or as
// the last statement of a batch) unmarks it. Any other statement, and any
// batch, sent on a connection that is not scoped is a violation. So a test
// that drives a request through a Guard pool and finds no violation has
// shown that the GUC was set before the first statement of every round trip
// and every batch, and Tenants shows which tenant each one ran under.
//
// Guard also counts round trips (Trips) and statements (Statements) per
// request context, for the statement budgets.
package tenancytest

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Trip is one round trip: a single statement, or a batch.
type Trip struct {
	// SQL is the statement, or a batch's statements in order.
	SQL []string
	// Tenant is the tenant the connection was scoped to when the trip was
	// sent; uuid.Nil when it was not scoped (a violation, unless the trip
	// is the tenancy BEGIN itself).
	Tenant uuid.UUID
	// Batch is true for a pgx.Batch.
	Batch bool
}

type traceKey struct{}

// Recorder collects the trips of one request. Attach it to the request's
// context with Record.
type Recorder struct {
	mu    sync.Mutex
	trips []Trip
}

// Record returns ctx carrying a new Recorder.
func Record(ctx context.Context) (context.Context, *Recorder) {
	r := &Recorder{}
	return context.WithValue(ctx, traceKey{}, r), r
}

// Trips returns the round trips recorded so far.
func (r *Recorder) Trips() []Trip {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Trip(nil), r.trips...)
}

// Statements is the number of statements across every trip, each statement
// of a batch counted, the tenancy BEGIN counted as two.
func (r *Recorder) Statements() int {
	n := 0
	for _, t := range r.Trips() {
		for _, s := range t.SQL {
			n++
			if _, ok := tenancy.BeginTenant(s); ok {
				n++ // BEGIN and the setting
			}
		}
	}
	return n
}

func (r *Recorder) add(t Trip) {
	r.mu.Lock()
	r.trips = append(r.trips, t)
	r.mu.Unlock()
}

// Guard is the tracer. The zero value is ready; set it as a test pool's
// ConnConfig.Tracer.
type Guard struct {
	mu         sync.Mutex
	scoped     map[*pgx.Conn]uuid.UUID
	violations []string
}

var (
	_ pgx.QueryTracer = (*Guard)(nil)
	_ pgx.BatchTracer = (*Guard)(nil)
)

// Violations lists every statement or batch that ran on a connection with no
// tenancy BEGIN behind it.
func (g *Guard) Violations() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.violations...)
}

func (g *Guard) state(conn *pgx.Conn) uuid.UUID {
	if g.scoped == nil {
		g.scoped = map[*pgx.Conn]uuid.UUID{}
	}
	return g.scoped[conn]
}

func ends(sql string) bool {
	s := strings.ToLower(strings.TrimSpace(sql))
	return s == "commit" || s == "rollback"
}

// TraceQueryStart implements pgx.QueryTracer.
func (g *Guard) TraceQueryStart(ctx context.Context, conn *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	g.mu.Lock()
	defer g.mu.Unlock()
	tenant := g.state(conn)
	// The allocator pool's role check (ADR-005 A3) runs once as a connection
	// opens, under no tenant by design: neither a violation nor a request's
	// trip.
	if tenancy.IsAllocatorRoleCheck(d.SQL) {
		return ctx
	}
	if id, ok := tenancy.BeginTenant(d.SQL); ok {
		if tenant != uuid.Nil {
			g.violations = append(g.violations, "a tenancy BEGIN on a connection already scoped to "+tenant.String())
		}
		g.scoped[conn] = id
		record(ctx, Trip{SQL: []string{d.SQL}, Tenant: id})
		return ctx
	}
	// Ending a transaction is never a violation: a batch whose COMMIT was
	// skipped after an error is followed by a ROLLBACK.
	if tenant == uuid.Nil && !ends(d.SQL) {
		g.violations = append(g.violations, "no tenant: "+d.SQL)
	}
	if ends(d.SQL) {
		delete(g.scoped, conn)
	}
	record(ctx, Trip{SQL: []string{d.SQL}, Tenant: tenant})
	return ctx
}

// TraceQueryEnd implements pgx.QueryTracer.
func (g *Guard) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// TraceBatchStart implements pgx.BatchTracer: the whole batch is one trip,
// checked against the connection's state when it is sent.
func (g *Guard) TraceBatchStart(ctx context.Context, conn *pgx.Conn, d pgx.TraceBatchStartData) context.Context {
	g.mu.Lock()
	defer g.mu.Unlock()
	tenant := g.state(conn)
	// An AllocTx batch opens its own transaction and scopes it in its
	// first statements, so it runs under that tenant from the start.
	if id, ok := tenancy.AllocTenant(d.Batch); ok {
		if tenant != uuid.Nil {
			g.violations = append(g.violations, "an allocation BEGIN on a connection already scoped to "+tenant.String())
		}
		tenant = id
		g.scoped[conn] = id
	} else if id, ok := tenancy.FirstBatchTenant(d.Batch); ok {
		// A BeginInFirstBatch transaction opens, and sets its tenant, in
		// the first statements of its first batch (TICK-692).
		if tenant != uuid.Nil {
			g.violations = append(g.violations, "a first-batch BEGIN on a connection already scoped to "+tenant.String())
		}
		tenant = id
		g.scoped[conn] = id
	}
	sqls := make([]string, 0, d.Batch.Len())
	for _, q := range d.Batch.QueuedQueries {
		sqls = append(sqls, q.SQL)
	}
	if tenant == uuid.Nil {
		g.violations = append(g.violations, fmt.Sprintf("no tenant: batch %q", sqls))
	}
	if len(sqls) > 0 && ends(sqls[len(sqls)-1]) {
		delete(g.scoped, conn)
	}
	record(ctx, Trip{SQL: sqls, Tenant: tenant, Batch: true})
	return ctx
}

// TraceBatchQuery implements pgx.BatchTracer.
func (g *Guard) TraceBatchQuery(context.Context, *pgx.Conn, pgx.TraceBatchQueryData) {}

// TraceBatchEnd implements pgx.BatchTracer.
func (g *Guard) TraceBatchEnd(context.Context, *pgx.Conn, pgx.TraceBatchEndData) {}

func record(ctx context.Context, t Trip) {
	if r, ok := ctx.Value(traceKey{}).(*Recorder); ok {
		r.add(t)
	}
}
