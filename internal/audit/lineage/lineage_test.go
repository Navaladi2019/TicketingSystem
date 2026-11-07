package lineage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// R17 says 10 minutes. Changing DeadAfter or the interval in Predicate fails
// here; changing PolicyQual or the migration fails the catalog guard and
// TestPolicyIsThePredicate (integration).
func TestPredicateIsR17(t *testing.T) {
	if DeadAfter != 10*time.Minute {
		t.Fatalf("DeadAfter = %s, want ADR-004 R17's 10m", DeadAfter)
	}
	want := fmt.Sprintf("next_seal_at < now() - interval '%d minutes'", int(DeadAfter/time.Minute))
	if Predicate != want {
		t.Fatalf("Predicate = %q, want %q (DeadAfter)", Predicate, want)
	}
	if want := fmt.Sprintf("'%s'::interval", "00:10:00"); !strings.Contains(PolicyQual, want) {
		t.Fatalf("PolicyQual = %q, want it to hold %s", PolicyQual, want)
	}
}

// Both readers evaluate Predicate itself, and only it: no second interval.
func TestBothReadersUseThePredicate(t *testing.T) {
	for name, sql := range map[string]string{"ReseedGateSQL": ReseedGateSQL, "DeadLineagesSQL": DeadLineagesSQL} {
		if strings.Count(sql, Predicate) != 1 {
			t.Errorf("%s = %q, want Predicate exactly once", name, sql)
		}
		if n := strings.Count(sql, "interval"); n != 1 {
			t.Errorf("%s has %d intervals, want only Predicate's", name, n)
		}
	}
	if !strings.HasSuffix(ReseedGateSQL, "FOR UPDATE") {
		t.Errorf("ReseedGateSQL = %q, want it under the head lock", ReseedGateSQL)
	}
}

// nextSealAtCompare finds a comparison on next_seal_at written out by hand.
var nextSealAtCompare = regexp.MustCompile(`next_seal_at\s*(<|>|<=|>=)`)

// No production Go or SQL file outside this package restates R17, except the
// migration whose policy the catalog guard pins to PolicyQual. A reseed gate
// or a second gauge that writes its own predicate fails here.
func TestNoOtherCopyOfThePredicate(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	allowed := map[string]bool{
		filepath.Join("db", "migrations", "20261107100000_audit_chain_head_dead_lineage.sql"): true,
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			switch rel {
			case ".git", "docs", "scratch", "bench", filepath.Join("internal", "audit", "lineage"):
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || (!strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".sql")) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if nextSealAtCompare.Match(b) && !allowed[rel] {
			t.Errorf("%s compares next_seal_at itself; use lineage.Predicate (R17)", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

type failingQuerier struct{}

func (failingQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("connection refused")
}

// A failed read withdraws the gauge rather than leaving the last value.
func TestMonitorWithdrawsTheGaugeOnError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m, err := NewMonitor(failingQuerier{}, mp, slog.New(slog.DiscardHandler), 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.interval != DefaultInterval {
		t.Errorf("interval = %s, want DefaultInterval", m.interval)
	}
	m.count, m.ok = 3, true
	if err := m.Once(context.Background()); err == nil {
		t.Fatal("Once: nil error, want the read's")
	}
	if _, ok := m.Value(); ok {
		t.Fatal("Value ok after a failed read, want withdrawn")
	}
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, md := range sm.Metrics {
			if g, ok := md.Data.(metricdata.Gauge[int64]); ok && md.Name == DeadLineagesMetric && len(g.DataPoints) > 0 {
				t.Fatalf("%s exported %v after a failed read", DeadLineagesMetric, g.DataPoints)
			}
		}
	}
}

func TestMonitorRunStopsOnCancel(t *testing.T) {
	m, err := NewMonitor(failingQuerier{}, sdkmetric.NewMeterProvider(), slog.New(slog.DiscardHandler), time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := m.Run(ctx); err != nil {
		t.Fatalf("Run = %v, want nil on cancel", err)
	}
}
