//go:build integration

package groupsync_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-1072 (ADR-010-A2 A2-7, A2-R6): groupsync_org_units_live_max is the
// largest live org-unit count over the tenants a cycle visited. It is only
// reported; no sync is ever refused, skipped or delayed on it.

// flatUnits is a root and n-1 children: n live units in one tree.
func flatUnits(tid uuid.UUID, n int) []identity.DirectoryGroup {
	out := make([]identity.DirectoryGroup, 0, n)
	out = append(out, unitG("r", "", path(tid, "/ou/r"), "r"))
	for i := 1; i < n; i++ {
		id := fmt.Sprintf("c%d", i)
		out = append(out, unitG(id, "r", path(tid, "/ou/r/"+id), id))
	}
	return out
}

func TestOrgUnitsLiveMax(t *testing.T) {
	super, pool := cluster(t)
	a, b := uuid.New(), uuid.New()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	reader := metric.NewManualReader()
	s, err := groupsync.New(pool, dir, metric.NewMeterProvider(metric.WithReader(reader)), slog.New(slog.DiscardHandler), groupsync.Options{MaxOrgUnitDepth: 8})
	if err != nil {
		t.Fatal(err)
	}
	gauge := func() int64 {
		t.Helper()
		var rm metricdata.ResourceMetrics
		if err := reader.Collect(context.Background(), &rm); err != nil {
			t.Fatal(err)
		}
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name != "groupsync_org_units_live_max" {
					continue
				}
				g, ok := m.Data.(metricdata.Gauge[int64])
				if !ok {
					t.Fatalf("groupsync_org_units_live_max is %T, want an int64 gauge", m.Data)
				}
				if len(g.DataPoints) != 1 {
					t.Fatalf("%d data points, want 1 (no tenant label)", len(g.DataPoints))
				}
				if g.DataPoints[0].Attributes.Len() != 0 {
					t.Error("the gauge carries attributes; a tenant id must never be one")
				}
				return g.DataPoints[0].Value
			}
		}
		t.Fatal("groupsync_org_units_live_max not reported")
		return 0
	}
	cycleAll := func() {
		t.Helper()
		rs, err := s.Once(context.Background())
		if err != nil || len(rs) != 2 {
			t.Fatalf("Once = %v, %v", rs, err)
		}
		for _, r := range rs {
			if r.Err != nil {
				t.Fatal(r.Err)
			}
		}
	}

	// AC1: 12 and 30 live units, the second with 5 tombstoned.
	dir.set(a, identity.DirectoryChanges{OrgUnits: flatUnits(a, 12)})
	dir.set(b, identity.DirectoryChanges{OrgUnits: flatUnits(b, 35)})
	cycleAll()
	dir.set(b, identity.DirectoryChanges{OrgUnits: flatUnits(b, 30)})
	cycleAll()
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1 AND deleted_at IS NOT NULL", b); n != 5 {
		t.Fatalf("%d tombstoned units, want 5", n)
	}
	if got := gauge(); got != 30 {
		t.Fatalf("gauge = %d, want 30 (tombstoned units not counted)", got)
	}

	// AC5: a cycle that changes only the 12-unit tenant still reads 30.
	dir.set(a, identity.DirectoryChanges{OrgUnits: flatUnits(a, 13)})
	cycleAll()
	if got := gauge(); got != 30 {
		t.Errorf("gauge = %d after a cycle that changed only the small tenant, want 30", got)
	}
	// A fully quiet cycle (both unchanged) still reports.
	cycleAll()
	if got := gauge(); got != 30 {
		t.Errorf("gauge = %d after a quiet cycle, want 30", got)
	}
}

// AC2: a tenant above the certified 10,111 units syncs, its closure is
// rebuilt and the gauge reads its size: nothing refuses on tree size.
func TestOrgUnitsAboveCertifiedStillSync(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	reader := metric.NewManualReader()
	s, err := groupsync.New(pool, dir, metric.NewMeterProvider(metric.WithReader(reader)), slog.New(slog.DiscardHandler), groupsync.Options{MaxOrgUnitDepth: 8})
	if err != nil {
		t.Fatal(err)
	}
	const n = 10112
	dir.set(tid, identity.DirectoryChanges{OrgUnits: flatUnits(tid, n)})
	if r := once(t, s); r.Err != nil || r.Skipped {
		t.Fatalf("sync of %d units = %+v, want success", n, r)
	}
	if got := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1 AND deleted_at IS NULL", tid); got != n {
		t.Fatalf("%d live units stored, want %d", got, n)
	}
	if got := count(t, super, "SELECT count(*) FROM org_unit_closure WHERE tenant_id = $1", tid); got < n-1 {
		t.Errorf("closure has %d rows, want it rebuilt for %d units", got, n)
	}
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "groupsync_org_units_live_max" {
				if v := m.Data.(metricdata.Gauge[int64]).DataPoints[0].Value; v != n {
					t.Errorf("gauge = %d, want %d", v, n)
				}
				return
			}
		}
	}
	t.Fatal("groupsync_org_units_live_max not reported")
}
