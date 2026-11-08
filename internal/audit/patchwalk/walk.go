package patchwalk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

// MaxDepth is the deepest nesting Walk accepts, counting the patch array and
// the op object. A diff is capped at 64 KB (ADR-004 R1), so a legitimate one
// never comes close.
const MaxDepth = 256

// ErrMalformed is wrapped by every error Walk returns for a diff that is not
// a JSON Patch it can walk completely. The message names the op index and
// member, never a value.
var ErrMalformed = errors.New("patchwalk: malformed JSON Patch")

// Kind is a JSON Patch operation name.
type Kind string

// The six RFC 6902 operations.
const (
	KindAdd     Kind = "add"
	KindRemove  Kind = "remove"
	KindReplace Kind = "replace"
	KindMove    Kind = "move"
	KindCopy    Kind = "copy"
	KindTest    Kind = "test"
)

// Field says which part of an op a visit is for.
type Field int

// Visit fields.
const (
	// FieldPath is the op's "path"; v is nil.
	FieldPath Field = iota + 1
	// FieldFrom is a copy or move op's "from"; v is nil.
	FieldFrom
	// FieldValue is the op's "value" at "path" or any node nested in it; v is
	// that node's raw JSON.
	FieldValue
)

func (f Field) String() string {
	switch f {
	case FieldPath:
		return "path"
	case FieldFrom:
		return "from"
	case FieldValue:
		return "value"
	default:
		return "unknown"
	}
}

// Op identifies the operation a visit belongs to.
type Op struct {
	// Index is the op's position in the patch array.
	Index int
	// Kind is the op's operation name.
	Kind Kind
	// Field is the part of the op this visit is for.
	Field Field
}

// Walk parses diff as an RFC 6902 JSON Patch and calls fn for the path, the
// from and every value node of every op (see the package doc for the order).
// The whole diff is parsed and checked before the first call, so a malformed
// diff never produces a partial walk. The first error fn returns stops the
// walk and is returned unchanged.
func Walk(diff []byte, fn func(op Op, ptr Pointer, v json.RawMessage) error) error {
	ops, err := parsePatch(diff)
	if err != nil {
		return err
	}
	for _, o := range ops {
		if err := fn(Op{Index: o.index, Kind: o.kind, Field: FieldPath}, o.path, nil); err != nil {
			return err
		}
		if o.kind == KindCopy || o.kind == KindMove {
			if err := fn(Op{Index: o.index, Kind: o.kind, Field: FieldFrom}, o.from, nil); err != nil {
				return err
			}
		}
		if o.value != nil {
			vop := Op{Index: o.index, Kind: o.kind, Field: FieldValue}
			if err := visit(diff, vop, o.path, o.value, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

func visit(diff []byte, op Op, ptr Pointer, n *node, fn func(Op, Pointer, json.RawMessage) error) error {
	if err := fn(op, ptr, json.RawMessage(diff[n.start:n.end])); err != nil {
		return err
	}
	for _, m := range n.members {
		if err := visit(diff, op, ptr.Child(m.key), m.val, fn); err != nil {
			return err
		}
	}
	for i, e := range n.elems {
		if err := visit(diff, op, ptr.Child(strconv.Itoa(i)), e, fn); err != nil {
			return err
		}
	}
	return nil
}

// patchOp is one parsed and checked op.
type patchOp struct {
	index int
	kind  Kind
	path  Pointer
	from  Pointer
	value *node // nil unless add, replace or test
}

// allowed lists, per op, the members it may and must carry besides "op" and
// "path".
func allowed(k Kind) (from, value, ok bool) {
	switch k {
	case KindAdd, KindReplace, KindTest:
		return false, true, true
	case KindRemove:
		return false, false, true
	case KindMove, KindCopy:
		return true, false, true
	default:
		return false, false, false
	}
}

func parsePatch(diff []byte) ([]patchOp, error) {
	if !utf8.Valid(diff) {
		return nil, fmt.Errorf("%w: invalid UTF-8", ErrMalformed)
	}
	p := &parser{data: diff, dec: json.NewDecoder(bytes.NewReader(diff))}
	p.dec.UseNumber()
	root, err := p.parse(1)
	if err != nil {
		return nil, err
	}
	if _, err := p.dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: trailing data after the patch array", ErrMalformed)
	}
	if root.kind != kindArray {
		return nil, fmt.Errorf("%w: a patch is a JSON array", ErrMalformed)
	}
	ops := make([]patchOp, 0, len(root.elems))
	for i, e := range root.elems {
		o, err := checkOp(i, e)
		if err != nil {
			return nil, err
		}
		ops = append(ops, o)
	}
	return ops, nil
}

func checkOp(i int, n *node) (patchOp, error) {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: op %d: %s", ErrMalformed, i, fmt.Sprintf(format, args...))
	}
	if n.kind != kindObject {
		return patchOp{}, bad("an op is a JSON object")
	}
	members := make(map[string]*node, len(n.members))
	for _, m := range n.members {
		members[m.key] = m.val
	}
	opNode, ok := members["op"]
	if !ok || opNode.kind != kindString {
		return patchOp{}, bad(`"op" is required and is a string`)
	}
	o := patchOp{index: i, kind: Kind(opNode.str)}
	wantFrom, wantValue, ok := allowed(o.kind)
	if !ok {
		return patchOp{}, bad(`unknown "op"`)
	}
	for _, m := range n.members {
		switch {
		case m.key == "op", m.key == "path":
		case m.key == "from" && wantFrom:
		case m.key == "value" && wantValue:
		default:
			// The key is not named: it is client data and could be anything.
			return patchOp{}, bad("member %d is not allowed on a %s op", memberIndex(n, m.key), o.kind)
		}
	}
	var err error
	if o.path, err = pointerMember(members, "path"); err != nil {
		return patchOp{}, bad("%v", err)
	}
	if wantFrom {
		if o.from, err = pointerMember(members, "from"); err != nil {
			return patchOp{}, bad("%v", err)
		}
	}
	if wantValue {
		if o.value = members["value"]; o.value == nil {
			return patchOp{}, bad(`"value" is required on a %s op`, o.kind)
		}
	}
	return o, nil
}

func memberIndex(n *node, key string) int {
	for i, m := range n.members {
		if m.key == key {
			return i
		}
	}
	return -1
}

func pointerMember(members map[string]*node, name string) (Pointer, error) {
	v, ok := members[name]
	if !ok || v.kind != kindString {
		return "", fmt.Errorf("%q is required and is a string", name)
	}
	p := Pointer(v.str)
	if err := p.Validate(); err != nil {
		return "", fmt.Errorf("%q: %w", name, err)
	}
	return p, nil
}

type nodeKind int

const (
	kindScalar nodeKind = iota
	kindString
	kindObject
	kindArray
)

// node is one parsed JSON value with its byte span in the diff.
type node struct {
	start, end int
	kind       nodeKind
	str        string // kindString only
	members    []member
	elems      []*node
}

type member struct {
	key string
	val *node
}

// parser reads the diff token by token so it sees every object member,
// duplicates included, and records each value's byte span.
type parser struct {
	data []byte
	dec  *json.Decoder
}

func (p *parser) syntaxErr(off int) error {
	// The decoder's own message quotes input bytes; report the offset only.
	return fmt.Errorf("%w: invalid JSON at byte %d", ErrMalformed, off)
}

// valueStart returns the offset of the next value's first byte.
func (p *parser) valueStart() int {
	i := int(p.dec.InputOffset())
	for i < len(p.data) {
		switch p.data[i] {
		case ' ', '\t', '\n', '\r', ',', ':':
			i++
		default:
			return i
		}
	}
	return i
}

func (p *parser) parse(depth int) (*node, error) {
	if depth > MaxDepth {
		return nil, fmt.Errorf("%w: nested deeper than %d", ErrMalformed, MaxDepth)
	}
	start := p.valueStart()
	tok, err := p.dec.Token()
	if err != nil {
		return nil, p.syntaxErr(start)
	}
	n := &node{start: start}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			if err := p.object(n, depth); err != nil {
				return nil, err
			}
		case '[':
			if err := p.array(n, depth); err != nil {
				return nil, err
			}
		default:
			return nil, p.syntaxErr(start)
		}
	case string:
		n.kind, n.str = kindString, t
	default:
		n.kind = kindScalar
	}
	n.end = int(p.dec.InputOffset())
	return n, nil
}

func (p *parser) object(n *node, depth int) error {
	n.kind = kindObject
	seen := make(map[string]struct{})
	for p.dec.More() {
		off := p.valueStart()
		tok, err := p.dec.Token()
		if err != nil {
			return p.syntaxErr(off)
		}
		key, ok := tok.(string)
		if !ok {
			return p.syntaxErr(off)
		}
		if _, dup := seen[key]; dup {
			return fmt.Errorf("%w: duplicate object key at byte %d", ErrMalformed, off)
		}
		seen[key] = struct{}{}
		v, err := p.parse(depth + 1)
		if err != nil {
			return err
		}
		n.members = append(n.members, member{key: key, val: v})
	}
	return p.closing('}')
}

func (p *parser) array(n *node, depth int) error {
	n.kind = kindArray
	for p.dec.More() {
		v, err := p.parse(depth + 1)
		if err != nil {
			return err
		}
		n.elems = append(n.elems, v)
	}
	return p.closing(']')
}

func (p *parser) closing(want json.Delim) error {
	off := p.valueStart()
	tok, err := p.dec.Token()
	if err != nil || tok != want {
		return p.syntaxErr(off)
	}
	return nil
}
