package pipeline_test

import (
	"encoding/json"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// The advisory lock key of a unique value (D-3-22) must be the same for
// every pair of values jsonb equality calls equal, or two writers of equal
// values would not queue on one lock; and different for values it tells
// apart, or unrelated writers would wait (harmless, but not by design).
func TestCanonicalValue(t *testing.T) {
	n := func(s string) json.Number { return json.Number(s) }
	same := []struct {
		name string
		a, b any
	}{
		{"integer and decimal", n("1"), n("1.0")},
		{"exponent", n("100"), n("1e2")},
		{"negative zero", n("0"), n("-0.0")},
		{"object key order", map[string]any{"a": n("1"), "b": "x"}, map[string]any{"b": "x", "a": n("1.00")}},
		{"nested arrays", []any{n("2"), []any{"x"}}, []any{n("2.0"), []any{"x"}}},
	}
	for _, c := range same {
		ka, _ := pipeline.CanonicalValue(c.a, false)
		kb, _ := pipeline.CanonicalValue(c.b, false)
		if ka != kb {
			t.Errorf("%s: %q != %q", c.name, ka, kb)
		}
	}
	differ := []struct {
		name string
		a, b any
	}{
		{"string and number", "7", n("7")},
		{"case", "SN-1", "sn-1"},
		{"bool and string", true, "true"},
		{"null and string", nil, "null"},
		{"array order", []any{"a", "b"}, []any{"b", "a"}},
		{"close numbers", n("0.1"), n("0.10000000000000001")},
	}
	for _, c := range differ {
		ka, _ := pipeline.CanonicalValue(c.a, false)
		kb, _ := pipeline.CanonicalValue(c.b, false)
		if ka == kb {
			t.Errorf("%s: both %q", c.name, ka)
		}
	}

	// A fixed uuid column compares the lowercase form the column returns.
	upper, lower := "0198F3C2-7A1B-7C3D-8E4F-0123456789AB", "0198f3c2-7a1b-7c3d-8e4f-0123456789ab"
	ku, vu := pipeline.CanonicalValue(upper, true)
	kl, vl := pipeline.CanonicalValue(lower, true)
	if ku != kl || vu != lower || vl != lower {
		t.Fatalf("uuid column: keys %q %q, values %v %v", ku, kl, vu, vl)
	}
	// Data values are passed through as they are; jsonb compares them.
	if _, v := pipeline.CanonicalValue(upper, false); v != upper {
		t.Fatalf("data value changed to %v", v)
	}
}

func TestP311ErrorMessages(t *testing.T) {
	for _, c := range []struct {
		err  error
		want string
	}{
		{&pipeline.ErrIllegalTransition{From: "new", To: "closed"}, `pipeline: illegal state transition "new" -> "closed"`},
		{&pipeline.ErrIllegalTransition{To: "closed"}, `pipeline: a record is created in its initial state, not "closed"`},
		{&pipeline.ErrUniqueViolation{Field: "serial"}, `pipeline: value of unique property "serial" is taken`},
		{&pipeline.ErrIdempotencyConflict{Concurrent: true}, "pipeline: concurrent write with the same idempotency key, retry"},
		{&pipeline.ErrIdempotencyConflict{}, "pipeline: idempotency key already used for another entity type, op or record"},
	} {
		if got := c.err.Error(); got != c.want {
			t.Errorf("%T: %q, want %q", c.err, got, c.want)
		}
	}
}
