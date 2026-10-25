package relay

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

func TestIsStart(t *testing.T) {
	id := uuid.MustParse("01920000-0000-7000-8000-0000000000ab")
	other := uuid.MustParse("01920000-0000-7000-8000-0000000000cd")
	for _, tc := range []struct {
		name      string
		kind, key string
		want      bool
	}{
		{"own id", "ticket.approval", "ticket.approval:" + id.String(), true},
		{"another row's id", "approve", "ticket.approval:" + other.String(), false},
		{"signal named like the workflow", "ticket.approval", "ticket.approval:" + other.String(), false},
		{"uppercase id is not canonical", "ticket.approval", "ticket.approval:" + strings.ToUpper(id.String()), false},
		{"other kind, own id", "approve", "ticket.approval:" + id.String(), false},
	} {
		if got := (Claimed{ID: id, Kind: tc.kind, TargetKey: tc.key}).IsStart(); got != tc.want {
			t.Errorf("%s: IsStart = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDelay(t *testing.T) {
	r := &Relay{opts: Options{BackoffBase: time.Second, BackoffMax: 5 * time.Minute}}
	for attempts, want := range map[int32]time.Duration{
		0: time.Second, 1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second,
		9: 256 * time.Second, 10: 5 * time.Minute, 1 << 30: 5 * time.Minute,
	} {
		if got := r.delay(attempts); got != want {
			t.Errorf("delay(%d) = %s, want %s", attempts, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short"); got != "short" {
		t.Errorf("truncate(short) = %q", got)
	}
	long := strings.Repeat("a", MaxErrorLen-1) + "é" // é is 2 bytes and straddles the cut
	if got := truncate(long); len(got) != MaxErrorLen-1 || !strings.HasSuffix(got, "a") {
		t.Errorf("truncate cut %d bytes, want %d on a rune boundary", len(got), MaxErrorLen-1)
	}
	// PostgreSQL rejects both in text; either would fail the whole Backoff.
	if got := truncate("bad \xff\xfe byte and \x00 nul"); got != "bad � byte and � nul" {
		t.Errorf("truncate kept unstorable bytes: %q", got)
	}
	if got := truncate(strings.Repeat("\xff", 2*MaxErrorLen)); len(got) > MaxErrorLen || !utf8.ValidString(got) {
		t.Errorf("truncate of invalid input: %d bytes, valid %v", len(got), utf8.ValidString(got))
	}
}

// D-P8 and A.2: the worker's applied-signal window outlives every redelivery.
func TestMaxRedeliveryAgeWithinDedupeWindow(t *testing.T) {
	if MaxRedeliveryAge >= workerguard.SignalDedupeWindow {
		t.Fatalf("MaxRedeliveryAge %s is not shorter than SignalDedupeWindow %s",
			MaxRedeliveryAge, workerguard.SignalDedupeWindow)
	}
}

// TICK-94 item 7: buckets are upper-inclusive at the cap, and the histogram
// carries no attributes.
func TestScannedBuckets(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	r, err := New(nil, sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)), slog.New(slog.DiscardHandler), Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []int64{3, 200, 201} {
		r.scanned.Record(context.Background(), v)
	}
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	h := scannedHistogram(t, rm)
	if h.Count != 3 || h.Attributes.Len() != 0 {
		t.Fatalf("count %d, attributes %v; want 3, none", h.Count, h.Attributes)
	}
	if le200, le250 := cumulativeAt(h, 200), cumulativeAt(h, 250); le200 != 2 || le250 != 3 {
		t.Fatalf(`le="200" %d, le="250" %d; want 2, 3`, le200, le250)
	}
}
