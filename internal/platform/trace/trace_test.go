package trace_test

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/trace"
)

const valid = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     string
		ok     bool
		trace  string
		parent string
	}{
		{"valid v00", valid, true, "4bf92f3577b34da6a3ce929d0e0e4736", "00f067aa0ba902b7"},
		{"future version with suffix", "01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-extra", true, "4bf92f3577b34da6a3ce929d0e0e4736", "00f067aa0ba902b7"},
		{"empty", "", false, "", ""},
		{"v00 with suffix", valid + "-extra", false, "", ""},
		{"version ff", "ff-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", false, "", ""},
		{"uppercase hex", "00-4BF92F3577B34DA6A3CE929D0E0E4736-00f067aa0ba902b7-01", false, "", ""},
		{"zero trace id", "00-00000000000000000000000000000000-00f067aa0ba902b7-01", false, "", ""},
		{"zero parent id", "00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01", false, "", ""},
		{"bad separator", "00_4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", false, "", ""},
		{"non-hex", "00-4bf92f3577b34da6a3ce929d0e0e47zz-00f067aa0ba902b7-01", false, "", ""},
		{"short", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7", false, "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sc, ok := trace.Parse(tc.in)
			if ok != tc.ok {
				t.Fatalf("Parse(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			}
			if !ok {
				return
			}
			if sc.TraceIDString() != tc.trace || sc.SpanIDString() != tc.parent {
				t.Errorf("Parse(%q) = %s/%s, want %s/%s", tc.in, sc.TraceIDString(), sc.SpanIDString(), tc.trace, tc.parent)
			}
		})
	}
}

func TestFromRequestContinuesInboundTrace(t *testing.T) {
	t.Parallel()
	sc := trace.FromRequest(valid)
	if got := sc.TraceIDString(); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("trace id = %s, want the inbound one", got)
	}
	if sc.SpanIDString() == "00f067aa0ba902b7" {
		t.Error("span id reused the caller's parent id; a server span needs its own")
	}
	if sc.Flags != 0x01 {
		t.Errorf("flags = %#x, want the inbound 0x01", sc.Flags)
	}
}

func TestFromRequestStartsNewTraceOnInvalidHeader(t *testing.T) {
	t.Parallel()
	a := trace.FromRequest("garbage")
	b := trace.FromRequest("")
	if a.TraceIDString() == b.TraceIDString() {
		t.Error("two new traces share a trace id")
	}
	if _, ok := trace.Parse(a.Traceparent()); !ok {
		t.Errorf("generated traceparent %q does not parse", a.Traceparent())
	}
}
