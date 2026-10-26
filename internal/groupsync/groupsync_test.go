package groupsync

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

func TestPathRules(t *testing.T) {
	tid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	other := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	p := func(id uuid.UUID, rest string) string { return "/t/" + id.String() + rest }
	for name, tc := range map[string]struct {
		path        string
		group, unit bool
	}{
		"group":                    {p(tid, "/agents"), true, false},
		"nested group":             {p(tid, "/agents/tier1"), true, false},
		"org unit":                 {p(tid, "/ou/emea"), false, true},
		"nested org unit":          {p(tid, "/ou/emea/de"), false, true},
		"ou container":             {p(tid, "/ou"), false, false},
		"ou container slash":       {p(tid, "/ou/"), false, false},
		"tenant root":              {p(tid, ""), false, false},
		"tenant root slash":        {p(tid, "/"), false, false},
		"other tenant group":       {p(other, "/agents"), false, false},
		"other tenant org unit":    {p(other, "/ou/emea"), false, false},
		"prefix lookalike":         {p(tid, "x/agents"), false, false},
		"ou lookalike is a group":  {p(tid, "/outside/x"), true, false},
		"no tenant":                {"/agents", false, false},
		"empty":                    {"", false, false},
		"other tenant id prefixed": {"/t/" + tid.String()[:8], false, false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := isGroupPath(tid, tc.path); got != tc.group {
				t.Errorf("isGroupPath = %v, want %v", got, tc.group)
			}
			if got := isOrgUnitPath(tid, tc.path); got != tc.unit {
				t.Errorf("isOrgUnitPath = %v, want %v", got, tc.unit)
			}
		})
	}
}

func TestLinksJSON(t *testing.T) {
	got, err := linksJSON(nil, nil)
	if err != nil || string(got) != "[]" {
		t.Fatalf("no links = %s, %v; want [] (NULL would mean never synced)", got, err)
	}
	in := []identity.FederatedLink{{IdPAlias: "idp-a", ExternalUserID: "x1"}, {IdPAlias: "idp-b", ExternalUserID: "x2"}}
	got, err = linksJSON(in, map[string]bool{"idp-a": true})
	if err != nil || string(got) != `[{"identityProvider":"idp-a","userId":"x1"}]` {
		t.Fatalf("links = %s, %v; want only the bound alias", got, err)
	}
	got, err = linksJSON(in, nil)
	if err != nil || string(got) != "[]" {
		t.Fatalf("unbound tenant links = %s, %v; want []", got, err)
	}
	if !sameJSON([]byte(`[ ]`), []byte(`[]`)) || sameJSON([]byte(`[]`), nil) {
		t.Error("sameJSON compares values, and NULL is never equal")
	}
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

// The lag histogram gets one observation per tenant per cycle with no
// attributes, and a tenant over the limit is named in a log line.
func TestObserveLag(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	clk := &clock{t: time.Unix(1_000_000, 0)}
	h, err := mp.Meter("test").Float64Histogram("groupsync_lag_seconds")
	if err != nil {
		t.Fatal(err)
	}
	// observe touches neither the pool nor the directory.
	s := &Syncer{log: log, opts: Options{Now: clk.now}, lag: h,
		cursors: map[uuid.UUID]string{}, lastOK: map[uuid.UUID]time.Time{}, started: clk.t}

	fresh, stale := uuid.New(), uuid.New()
	s.lastOK[fresh] = clk.t.Add(-20 * time.Second)
	s.lastOK[stale] = clk.t.Add(-11 * time.Minute)
	s.observe(context.Background(), Result{TenantID: fresh})
	s.observe(context.Background(), Result{TenantID: stale, Err: errors.New("boom")})
	s.observe(context.Background(), Result{TenantID: uuid.New(), Skipped: true})

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "groupsync_lag_seconds" {
				continue
			}
			found = true
			h := m.Data.(metricdata.Histogram[float64])
			if len(h.DataPoints) != 1 {
				t.Fatalf("data points = %d, want 1 (no tenant attribute)", len(h.DataPoints))
			}
			dp := h.DataPoints[0]
			if dp.Attributes.Len() != 0 {
				t.Errorf("attributes = %v, want none", dp.Attributes)
			}
			if dp.Count != 2 {
				t.Errorf("observations = %d, want 2 (a skipped tenant records none)", dp.Count)
			}
		}
	}
	if !found {
		t.Fatal("groupsync_lag_seconds not recorded")
	}
	out := buf.String()
	if !strings.Contains(out, "lag over 10 minutes") || !strings.Contains(out, stale.String()) {
		t.Errorf("log does not name the stale tenant: %s", out)
	}
	if strings.Contains(out, fresh.String()) {
		t.Errorf("log names a tenant within the limit: %s", out)
	}
}

func TestPathsNeverEnterErrors(t *testing.T) {
	for _, e := range []error{ErrOrgUnitCycle, ErrOrgUnitDepth, ErrBadSnapshot} {
		if strings.Contains(e.Error(), "/t/") {
			t.Errorf("%v carries a path", e)
		}
	}
}
