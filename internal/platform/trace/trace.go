// Package trace carries W3C Trace Context (the traceparent header) through a
// request. There is no tracing backend yet: the trace id is propagated and
// logged so that one can be adopted later without touching call sites.
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// Header is the W3C Trace Context request header.
const Header = "traceparent"

// SpanContext identifies the current span within a trace.
type SpanContext struct {
	TraceID [16]byte
	SpanID  [8]byte
	Flags   byte
}

// TraceIDString returns the trace id as 32 lowercase hex characters.
func (sc SpanContext) TraceIDString() string { return hex.EncodeToString(sc.TraceID[:]) }

// SpanIDString returns the span id as 16 lowercase hex characters.
func (sc SpanContext) SpanIDString() string { return hex.EncodeToString(sc.SpanID[:]) }

// Traceparent formats sc as a version-00 traceparent value, for outbound calls.
func (sc SpanContext) Traceparent() string {
	return "00-" + sc.TraceIDString() + "-" + sc.SpanIDString() + "-" + hex.EncodeToString([]byte{sc.Flags})
}

// Parse decodes a traceparent header value. It returns false for anything the
// W3C spec says to ignore: wrong shape, uppercase hex, version ff, or an
// all-zero trace or parent id. Versions above 00 are read by their 00 prefix,
// as the spec requires.
func Parse(v string) (SpanContext, bool) {
	var sc SpanContext
	const v00Len = 55 // 2 + 1 + 32 + 1 + 16 + 1 + 2
	if len(v) < v00Len || v[2] != '-' || v[35] != '-' || v[52] != '-' {
		return sc, false
	}
	version, ok := decodeHex(v[0:2])
	if !ok || version[0] == 0xff {
		return sc, false
	}
	if version[0] == 0 && len(v) != v00Len {
		return sc, false
	}
	if len(v) > v00Len && v[v00Len] != '-' {
		return sc, false
	}
	traceID, ok := decodeHex(v[3:35])
	if !ok || isZero(traceID) {
		return sc, false
	}
	spanID, ok := decodeHex(v[36:52])
	if !ok || isZero(spanID) {
		return sc, false
	}
	flags, ok := decodeHex(v[53:55])
	if !ok {
		return sc, false
	}
	copy(sc.TraceID[:], traceID)
	copy(sc.SpanID[:], spanID)
	sc.Flags = flags[0]
	return sc, true
}

// FromRequest returns the span context for a server span: it continues the
// inbound trace when the header parses, otherwise it starts a new trace. The
// span id is always new, because this process is a new span.
func FromRequest(header string) SpanContext {
	sc, ok := Parse(header)
	if !ok {
		sc = SpanContext{}
		randomNonZero(sc.TraceID[:])
	}
	randomNonZero(sc.SpanID[:])
	return sc
}

type ctxKey struct{}

// WithSpanContext returns ctx carrying sc.
func WithSpanContext(ctx context.Context, sc SpanContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, sc)
}

// FromContext returns the span context in ctx, if any.
func FromContext(ctx context.Context) (SpanContext, bool) {
	sc, ok := ctx.Value(ctxKey{}).(SpanContext)
	return sc, ok
}

// decodeHex decodes lowercase hex only; the spec forbids uppercase.
func decodeHex(s string) ([]byte, bool) {
	for i := range len(s) {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return nil, false
		}
	}
	b, err := hex.DecodeString(s)
	return b, err == nil
}

func isZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

func randomNonZero(b []byte) {
	for {
		_, _ = rand.Read(b) // crypto/rand.Read never returns an error (Go ≥ 1.24)
		if !isZero(b) {
			return
		}
	}
}
