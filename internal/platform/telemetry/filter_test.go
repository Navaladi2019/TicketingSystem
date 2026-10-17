package telemetry_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// newExported builds the providers with an in-memory exporter behind the real
// filteringExporter, so every assertion reads spans as they are exported.
func newExported(t *testing.T) (*telemetry.Telemetry, *tracetest.InMemoryExporter) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), load(t, nil), telemetry.WithSpanExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	return tel, exp
}

// TestSpikeFilterBypass is the TICK-94 case: client and IdP values written
// into a span by each route a library uses (attributes, RecordError, the
// status, an event, a link) never reach the exported span, while the platform
// ids and exception.type survive.
func TestSpikeFilterBypass(t *testing.T) {
	t.Parallel()
	tel, exp := newExported(t)

	_, span := tel.TracerProvider.Tracer("spike").Start(t.Context(), "op",
		trace.WithLinks(trace.Link{
			SpanContext: span0(),
			Attributes:  []attribute.KeyValue{attribute.String("url.full", "https://kc/users?email=bob@x")},
		}),
	)
	span.SetAttributes(
		attribute.String("url.full", "https://api.test/users?email=alice@tenant-a.test"),
		attribute.String("temporalWorkflowID", "t:tenant-a:ticket-1"),
	)
	// What otelpgx records for a PostgreSQL 22P02 error.
	span.RecordError(fmt.Errorf("invalid input syntax for type uuid: %q", "alice@tenant-a.test"))
	span.SetStatus(codes.Error, "lookup failed for alice@tenant-a.test")
	span.AddEvent("idp call", trace.WithAttributes(attribute.String("url.full", "https://kc/users?email=bob@x")))
	span.End()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("exported %d spans, want 1", len(spans))
	}
	s := spans[0]
	for _, l := range leaks(s, "alice@", "bob@") {
		t.Errorf("exported span leaks %s", l)
	}
	if got := attrMap(s.Attributes)["temporalWorkflowID"]; got != "t:tenant-a:ticket-1" {
		t.Errorf("temporalWorkflowID = %q, want it kept", got)
	}
	var exceptionType string
	for _, ev := range s.Events {
		if ev.Name == "exception" {
			exceptionType = attrMap(ev.Attributes)["exception.type"]
		}
	}
	if exceptionType == "" {
		t.Error("the exception event lost exception.type")
	}
}

// TestUnlistedKeysAreDropped: a key not on the allow-list, such as one a new
// library emits, is dropped from span, event and link attributes; the platform
// ids are kept in all three.
func TestUnlistedKeysAreDropped(t *testing.T) {
	t.Parallel()
	tel, exp := newExported(t)

	attrs := []attribute.KeyValue{
		attribute.String("foo.bar", "x"),
		attribute.String("db.system.name", "postgresql"),
		attribute.String("nava.principal_id", "p-1"), // nava.* but not declared: dropped
		telemetry.TenantIDKey.String("tenant-a"),
		attribute.String("temporalWorkflowID", "t:tenant-a:ticket-1"),
		attribute.String("temporalRunID", "run-1"),
	}
	want := []string{"nava.tenant_id", "temporalRunID", "temporalWorkflowID"}

	_, span := tel.TracerProvider.Tracer("lib").Start(t.Context(), "op",
		trace.WithAttributes(attrs...),
		trace.WithLinks(trace.Link{SpanContext: span0(), Attributes: attrs}),
	)
	span.AddEvent("ev", trace.WithAttributes(attrs...))
	span.End()

	s := exp.GetSpans()[0]
	if len(s.Events) != 1 || len(s.Links) != 1 {
		t.Fatalf("exported %d events and %d links, want 1 and 1", len(s.Events), len(s.Links))
	}
	for where, kvs := range map[string][]attribute.KeyValue{
		"span":  s.Attributes,
		"event": s.Events[0].Attributes,
		"link":  s.Links[0].Attributes,
	} {
		if got := sortedKeys(kvs); !slices.Equal(got, want) {
			t.Errorf("%s attribute keys = %v, want %v", where, got, want)
		}
	}
}

// TestStatusDescriptionIsErrorType: the exported description is the span's
// error.type, or empty without one; the status code is kept either way.
func TestStatusDescriptionIsErrorType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		attrs []attribute.KeyValue
		want  string
	}{
		{"with error.type", []attribute.KeyValue{attribute.String("error.type", "*pgconn.PgError")}, "*pgconn.PgError"},
		{"without error.type", nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tel, exp := newExported(t)
			_, span := tel.TracerProvider.Tracer("test").Start(t.Context(), "op", trace.WithAttributes(tc.attrs...))
			span.SetStatus(codes.Error, "lookup failed for alice@tenant-a.test")
			span.End()

			st := exp.GetSpans()[0].Status
			if st.Description != tc.want {
				t.Errorf("status description = %q, want %q", st.Description, tc.want)
			}
			if st.Code != codes.Error {
				t.Errorf("status code = %v, want Error", st.Code)
			}
		})
	}
}

// leaks lists every place in an exported span where a marker survives.
func leaks(s tracetest.SpanStub, markers ...string) []string {
	var out []string
	check := func(where, v string) {
		for _, m := range markers {
			if strings.Contains(v, m) {
				out = append(out, fmt.Sprintf("%s = %q", where, v))
			}
		}
	}
	for _, kv := range s.Attributes {
		check("attr "+string(kv.Key), kv.Value.String())
	}
	for _, ev := range s.Events {
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
	return out
}

func attrMap(kvs []attribute.KeyValue) map[attribute.Key]string {
	m := make(map[attribute.Key]string, len(kvs))
	for _, kv := range kvs {
		m[kv.Key] = kv.Value.String()
	}
	return m
}

func sortedKeys(kvs []attribute.KeyValue) []string {
	keys := make([]string, 0, len(kvs))
	for _, kv := range kvs {
		keys = append(keys, string(kv.Key))
	}
	slices.Sort(keys)
	return keys
}
