package jsondiff_test

// A minimal RFC 6902 applier for add, remove and replace, used only to check
// that Diff's output turns before into after. It is written from the RFCs,
// independently of diff.go, so a bug in Diff does not cancel out here.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type patchOp struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value"`
}

// decodeDoc parses one JSON value with numbers kept as literals.
func decodeDoc(raw []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// apply applies patch to doc (nil doc means "no document", for a create).
func apply(doc any, patch []byte) (any, error) {
	var ops []patchOp
	if err := json.Unmarshal(patch, &ops); err != nil {
		return nil, fmt.Errorf("patch: %w", err)
	}
	for i, op := range ops {
		var val any
		if op.Op == "add" || op.Op == "replace" {
			if op.Value == nil {
				return nil, fmt.Errorf("op %d: %s without value", i, op.Op)
			}
			v, err := decodeDoc(op.Value)
			if err != nil {
				return nil, fmt.Errorf("op %d value: %w", i, err)
			}
			val = v
		}
		var err error
		doc, err = applyOp(doc, op.Op, op.Path, val)
		if err != nil {
			return nil, fmt.Errorf("op %d (%s %q): %w", i, op.Op, op.Path, err)
		}
	}
	return doc, nil
}

func applyOp(doc any, op, path string, val any) (any, error) {
	if path == "" {
		switch op {
		case "add", "replace":
			return val, nil
		default:
			return nil, fmt.Errorf("cannot %s the root", op)
		}
	}
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("pointer must start with /")
	}
	tokens := strings.Split(path[1:], "/")
	for i, t := range tokens {
		tokens[i] = strings.ReplaceAll(strings.ReplaceAll(t, "~1", "/"), "~0", "~")
	}
	return applyAt(doc, op, tokens, val)
}

func applyAt(node any, op string, tokens []string, val any) (any, error) {
	tok, last := tokens[0], len(tokens) == 1
	switch n := node.(type) {
	case map[string]any:
		if !last {
			child, ok := n[tok]
			if !ok {
				return nil, fmt.Errorf("member %q missing", tok)
			}
			c, err := applyAt(child, op, tokens[1:], val)
			if err != nil {
				return nil, err
			}
			n[tok] = c
			return n, nil
		}
		_, ok := n[tok]
		switch op {
		case "add":
			n[tok] = val
		case "replace":
			if !ok {
				return nil, fmt.Errorf("replace of missing member %q", tok)
			}
			n[tok] = val
		case "remove":
			if !ok {
				return nil, fmt.Errorf("remove of missing member %q", tok)
			}
			delete(n, tok)
		default:
			return nil, fmt.Errorf("op %q not allowed", op)
		}
		return n, nil
	case []any:
		idx, err := strconv.Atoi(tok)
		if err != nil || idx < 0 || strconv.Itoa(idx) != tok {
			return nil, fmt.Errorf("bad array index %q", tok)
		}
		if !last {
			if idx >= len(n) {
				return nil, fmt.Errorf("index %d out of range", idx)
			}
			c, err := applyAt(n[idx], op, tokens[1:], val)
			if err != nil {
				return nil, err
			}
			n[idx] = c
			return n, nil
		}
		switch op {
		case "add":
			if idx > len(n) {
				return nil, fmt.Errorf("add index %d past end %d", idx, len(n))
			}
			n = append(n, nil)
			copy(n[idx+1:], n[idx:])
			n[idx] = val
		case "replace":
			if idx >= len(n) {
				return nil, fmt.Errorf("replace index %d out of range", idx)
			}
			n[idx] = val
		case "remove":
			if idx >= len(n) {
				return nil, fmt.Errorf("remove index %d out of range", idx)
			}
			n = append(n[:idx], n[idx+1:]...)
		default:
			return nil, fmt.Errorf("op %q not allowed", op)
		}
		return n, nil
	default:
		return nil, fmt.Errorf("cannot index into %T", node)
	}
}

// sameValue compares decoded JSON values; numbers compare by literal, as
// Diff does.
func sameValue(a, b any) bool { return reflect.DeepEqual(a, b) }
