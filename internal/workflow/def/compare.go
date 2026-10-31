package def

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
)

// Holds evaluates the condition against a decoded record, the comparison
// contract of the NextStep activity (P3-22a):
//
//   - Field is a JSON Pointer (RFC 6901) into the record.
//   - eq compares decoded JSON values: numbers by value (1 equals 1.0),
//     objects and arrays element by element, no type coercion ("1" is not 1).
//   - ne is the negation of eq.
//   - in holds when the field equals (as eq) any element of Value.
//   - A missing field equals nothing: eq and in are false, ne is true.
//     A field present with JSON null equals null.
//
// record is what encoding/json decodes into map[string]any; json.Number and
// float64 numbers are both accepted. A Value that does not decode (never the
// case after Parse) makes every operator false.
func (c Condition) Holds(record map[string]any) bool {
	var want any
	dec := json.NewDecoder(bytes.NewReader(c.Value))
	dec.UseNumber()
	if err := dec.Decode(&want); err != nil {
		return false
	}
	got, found := lookup(record, c.Field)
	switch c.Op {
	case OpEq:
		return found && equal(got, want)
	case OpNe:
		return !found || !equal(got, want)
	case OpIn:
		list, ok := want.([]any)
		if !found || !ok {
			return false
		}
		for _, w := range list {
			if equal(got, w) {
				return true
			}
		}
		return false
	}
	return false
}

// lookup resolves an RFC 6901 pointer. "" is the whole document.
func lookup(doc map[string]any, ptr string) (any, bool) {
	if ptr == "" {
		return doc, true
	}
	if !strings.HasPrefix(ptr, "/") {
		return nil, false
	}
	var cur any = doc
	for _, tok := range strings.Split(ptr[1:], "/") {
		tok = strings.ReplaceAll(strings.ReplaceAll(tok, "~1", "/"), "~0", "~")
		switch v := cur.(type) {
		case map[string]any:
			next, ok := v[tok]
			if !ok {
				return nil, false
			}
			cur = next
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= len(v) || (len(tok) > 1 && tok[0] == '0') {
				return nil, false
			}
			cur = v[i]
		default:
			return nil, false
		}
	}
	return cur, true
}

func equal(a, b any) bool {
	if ra, ok := number(a); ok {
		rb, ok := number(b)
		return ok && ra.Cmp(rb) == 0
	}
	switch av := a.(type) {
	case nil:
		return b == nil
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !equal(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, x := range av {
			y, ok := bv[k]
			if !ok || !equal(x, y) {
				return false
			}
		}
		return true
	}
	return false
}

// number returns v as an exact rational when it is a JSON number.
func number(v any) (*big.Rat, bool) {
	switch n := v.(type) {
	case json.Number:
		s := string(n)
		if !strings.ContainsAny(s, "eE") {
			r, ok := new(big.Rat).SetString(s)
			return r, ok
		}
		// An exponent goes through float64: big.Rat would expand "1e999999999"
		// into a billion-digit integer.
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, false
		}
		return number(f)
	case float64:
		r := new(big.Rat)
		if r.SetFloat64(n) == nil {
			return nil, false
		}
		return r, true
	}
	return nil, false
}
