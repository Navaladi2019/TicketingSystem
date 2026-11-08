package read

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

// Change is one create or update of a record rebuilt from its audit entries:
// the record payload before and after, as this reader may see it.
type Change struct {
	Entry Entry
	// Known is false when the payload cannot be rebuilt at this entry: its
	// diff, or an earlier one, is erased, truncated, redacted whole, or does
	// not apply. Before and After are then nil.
	Known bool
	// Before is nil on a create.
	Before, After map[string]any
}

// Replay rebuilds each entry's before and after payload by applying the
// entries' diffs in order, oldest first. The log stores diffs only (R1), so
// the payload at a version is the replay of every diff up to it. A masked
// member replays as {"redacted":"mask"}. Once a step cannot be applied, every
// later step is unknown too, because it would start from a wrong payload;
// a create restarts the replay.
func Replay(entries []Entry) []Change {
	out := make([]Change, len(entries))
	var (
		cur   map[string]any
		known bool
	)
	for i, e := range entries {
		c := Change{Entry: e}
		if e.Action == "create" {
			cur, known = nil, true
		}
		if known && e.Diff != nil && !e.DiffTruncated && e.ErasedAt == nil {
			next, err := applyPatch(cur, e.Diff)
			if err == nil {
				c.Known, c.Before, c.After = true, cur, next
				cur = next
				out[i] = c
				continue
			}
		}
		known = false
		out[i] = c
	}
	return out
}

var errUnsupported = errors.New("unsupported patch")

// applyPatch applies an RFC 6902 patch of add, remove and replace ops (all the
// pipeline's diffs use) to doc, which is nil before a create. It copies doc
// first, so the caller's before stays whole.
func applyPatch(doc map[string]any, diff []byte) (map[string]any, error) {
	if err := patchwalk.Walk(diff, func(patchwalk.Op, patchwalk.Pointer, json.RawMessage) error { return nil }); err != nil {
		return nil, err
	}
	var ops []rawOp
	if err := json.Unmarshal(diff, &ops); err != nil {
		return nil, err
	}
	var root any
	if doc != nil {
		root = deepCopy(doc)
	}
	for i, op := range ops {
		var val any
		if op.Op == "add" || op.Op == "replace" {
			dec := json.NewDecoder(bytes.NewReader(op.Value))
			dec.UseNumber()
			if err := dec.Decode(&val); err != nil {
				return nil, fmt.Errorf("op %d value: %w", i, err)
			}
		}
		var err error
		if root, err = applyOp(root, op.Op, patchwalk.Pointer(op.Path).Segments(), val); err != nil {
			return nil, fmt.Errorf("op %d: %w", i, err)
		}
	}
	m, ok := root.(map[string]any)
	if !ok {
		return nil, errors.New("result is not an object")
	}
	return m, nil
}

func applyOp(node any, op string, segs []string, val any) (any, error) {
	if len(segs) == 0 {
		if op == "add" || op == "replace" {
			return val, nil
		}
		return nil, fmt.Errorf("%w: %s of the root", errUnsupported, op)
	}
	if op != "add" && op != "remove" && op != "replace" {
		return nil, fmt.Errorf("%w: %s", errUnsupported, op)
	}
	tok, last := segs[0], len(segs) == 1
	switch n := node.(type) {
	case map[string]any:
		if !last {
			child, ok := n[tok]
			if !ok {
				return nil, errors.New("missing member")
			}
			c, err := applyOp(child, op, segs[1:], val)
			if err != nil {
				return nil, err
			}
			n[tok] = c
			return n, nil
		}
		_, ok := n[tok]
		switch {
		case op == "add":
			n[tok] = val
		case !ok:
			return nil, errors.New("missing member")
		case op == "replace":
			n[tok] = val
		default:
			delete(n, tok)
		}
		return n, nil
	case []any:
		idx, err := strconv.Atoi(tok)
		if err != nil || idx < 0 || strconv.Itoa(idx) != tok {
			return nil, errors.New("bad array index")
		}
		if !last {
			if idx >= len(n) {
				return nil, errors.New("index out of range")
			}
			c, err := applyOp(n[idx], op, segs[1:], val)
			if err != nil {
				return nil, err
			}
			n[idx] = c
			return n, nil
		}
		switch op {
		case "add":
			if idx > len(n) {
				return nil, errors.New("index out of range")
			}
			n = append(n, nil)
			copy(n[idx+1:], n[idx:])
			n[idx] = val
		case "replace":
			if idx >= len(n) {
				return nil, errors.New("index out of range")
			}
			n[idx] = val
		default:
			if idx >= len(n) {
				return nil, errors.New("index out of range")
			}
			n = append(n[:idx], n[idx+1:]...)
		}
		return n, nil
	}
	return nil, errors.New("path through a scalar")
}

func deepCopy(v any) any {
	switch x := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, e := range x {
			m[k] = deepCopy(e)
		}
		return m
	case []any:
		s := make([]any, len(x))
		for i, e := range x {
			s[i] = deepCopy(e)
		}
		return s
	}
	return v
}
