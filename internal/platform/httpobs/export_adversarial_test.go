package httpobs_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
)

// exportMarker is a client-chosen value. It is lowercase because a few libraries
// normalise case; the check below folds case anyway.
const exportMarker = "evilmarker"

// exportedLeaks lists every place in an exported span where exportMarker survives,
// including the span name and event names, which the attribute filter does
// not touch.
func exportedLeaks(s tracetest.SpanStub) []string {
	var out []string
	check := func(where, v string) {
		if strings.Contains(strings.ToLower(v), exportMarker) {
			out = append(out, where+" = "+v)
		}
	}
	check("span name", s.Name)
	for _, kv := range s.Attributes {
		check("attr "+string(kv.Key), kv.Value.String())
	}
	for _, ev := range s.Events {
		check("event name", ev.Name)
		for _, kv := range ev.Attributes {
			check("event "+ev.Name+" "+string(kv.Key), kv.Value.String())
		}
	}
	for _, l := range s.Links {
		check("link tracestate", l.SpanContext.TraceState().String())
		for _, kv := range l.Attributes {
			check("link "+string(kv.Key), kv.Value.String())
		}
	}
	check("status", s.Status.Description)
	for _, kv := range s.Resource.Attributes() {
		check("resource "+string(kv.Key), kv.Value.String())
	}
	return out
}

// TestClientValuesNeverReachTheExportedSpan is the TICK-105 adversarial case
// for the HTTP edge: a caller controls every header, the method, the Host,
// the path and the query, and seeds each with exportMarker. The span the exporter
// receives carries exportMarker nowhere: not in an attribute, event, link,
// tracestate, status or the span name.
func TestClientValuesNeverReachTheExportedSpan(t *testing.T) {
	t.Parallel()
	hostile := map[string]string{
		requestid.Header:   exportMarker + "-request-id",
		"X-Correlation-Id": exportMarker,
		"User-Agent":       exportMarker + "/1.0 alice@" + exportMarker + ".test",
		"X-Forwarded-For":  "203.0.113.9, " + exportMarker,
		"X-Forwarded-Host": exportMarker + ".test",
		"Forwarded":        "for=" + exportMarker + ";host=" + exportMarker,
		"Referer":          "https://" + exportMarker + ".test/?email=alice@" + exportMarker + ".test",
		"Authorization":    "Bearer " + exportMarker,
		"Cookie":           "session=" + exportMarker,
		"traceparent":      inbound,
		"tracestate":       "vendor=" + exportMarker,
		"baggage":          "user=" + exportMarker,
	}
	cases := []struct {
		name, method, target string
	}{
		{"matched route with marker in path and query", http.MethodGet, "/items/alice@" + exportMarker + ".test?email=bob@" + exportMarker + ".test"},
		{"unmatched route", http.MethodGet, "/" + exportMarker + "/alice@" + exportMarker + ".test?q=" + exportMarker},
		{"client-chosen method", strings.ToUpper(exportMarker), "/items/1?email=" + exportMarker},
		{"handler panic", http.MethodGet, "/panic?email=" + exportMarker},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, 1)
			req := httptest.NewRequestWithContext(t.Context(), tc.method, tc.target, nil)
			req.Host = exportMarker + ".test:8443"
			for k, v := range hostile {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			h.handler.ServeHTTP(rec, req)

			s := h.onlySpan(t)
			for _, l := range exportedLeaks(s) {
				t.Errorf("exported span leaks a client value: %s", l)
			}
			got := attrMap(s.Attributes)[httpobs.RequestIDAttribute]
			if !hex32.MatchString(got) || got != rec.Header().Get(requestid.Header) {
				t.Errorf("span %s = %q, want the server-generated id %q",
					httpobs.RequestIDAttribute, got, rec.Header().Get(requestid.Header))
			}
		})
	}
}
