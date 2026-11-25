package authz

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// The format-2 access document (ADR-010 D2, D3; PLAN-011 C2, TICK-936 P11-04).
// acl_versions.rules holds either a format-1 array of Rule (S10) or a format-2
// object:
//
//	{"format": 2,
//	 "conditions": [{"name": "...", "expr": <node>}],
//	 "policies":   [{"name": "...", "ops": ["read"], "when": <node>}]}
//
// Decode reads either and always returns a format-2 Document: a format-1 array
// is converted on read (convert.go, D12). Masks are not part of the document;
// they stay in their own column, unchanged.
//
// Decode checks the grammar only: the document shape, one known key per node,
// names, unique names and child counts, plus today's Cond rules for a cmp node.
// The checks that need the whole document or the database (every ref resolves,
// no cycle, depth and leaf caps, foreign ids, `under` not available yet, the
// byte limit) belong to the publish route (TICK-939 P11-07) and the database
// CHECK (TICK-940 P11-08), which extend the same Problem list and the shared
// fixture testdata/format2_invalid.json.

// FormatV2 is the value of a format-2 document's "format" key.
const FormatV2 = 2

// The problem codes of contract C1 (PLAN-011). They extend the Rule codes of
// the format-1 check (policycheck.go), which a format-2 document reuses for
// plain shape errors (type, required, value, empty, condition). Decode reports
// unknown_key, invalid_name, duplicate_name and child_count; the rest are
// declared here so the route and the CHECK share one list.
const (
	RuleInvalidName          = "invalid_name"
	RuleDuplicateName        = "duplicate_name"
	RuleUnresolvedRef        = "unresolved_ref"
	RuleRefCycle             = "ref_cycle"
	RuleChildCount           = "child_count"
	RuleDepthExceeded        = "depth_exceeded"
	RulePolicyLeavesExceeded = "policy_leaves_exceeded"
	RulePoliciesExceeded     = "policies_exceeded"
	RuleDocLeavesExceeded    = "document_leaves_exceeded"
	RuleForeignID            = "foreign_id"
	RuleNotAvailable         = "not_available"
	RuleDocumentTooLarge     = "document_too_large"
	// RuleUnderSetsExceeded: a policy names more distinct under start sets
	// than MaxUnderSets (SP-1d change 5 on TICK-946).
	RuleUnderSetsExceeded = "under_sets_exceeded"
	// RuleUnderNested: an under leaf lists a unit and one of its ancestors
	// (SP-1e, beside change 5 on TICK-946): the closure already gives the
	// descendant through its ancestor, so the pair is refused at publish.
	RuleUnderNested = "under_nested"
	// RuleUnderDuplicateID: an under leaf lists the same unit twice
	// (ADR-010-A2 A2-8, R2-N1), reported at the second occurrence.
	RuleUnderDuplicateID = "under_duplicate_id"
)

// MaxUnderSets is the most distinct under start sets U one policy may name,
// after refs expand (SP-1d change 5, ADR-010-A2): {"of": "caller"} is one U,
// and a listed {"of": [ids]} is one U per set of ids, in any order. It bounds
// the region terms a list's count drives through the closure.
const MaxUnderSets = 4

// underSets is the number of distinct under start sets reachable from n,
// refs expanded through conds; each condition is walked once, so a cycle
// (refused elsewhere) ends. It saturates just above MaxUnderSets.
func underSets(n Node, conds map[string]*Node) int {
	seen := map[string]bool{}
	visited := map[string]bool{}
	var walk func(Node)
	walk = func(n Node) {
		if len(seen) > MaxUnderSets {
			return
		}
		switch n.Kind() {
		case NodeAll:
			for _, k := range n.All {
				walk(k)
			}
		case NodeAny:
			for _, k := range n.Any {
				walk(k)
			}
		case NodeNot:
			walk(*n.Not)
		case NodeRef:
			if c, ok := conds[n.Ref]; ok && !visited[n.Ref] {
				visited[n.Ref] = true
				walk(*c)
			}
		case NodeUnder:
			key := "caller"
			if !n.Under.Caller {
				key = strings.Join(UUIDTexts(sortedIDs(n.Under.Units)), ",")
			}
			seen[key] = true
		}
	}
	walk(n)
	return len(seen)
}

// MaxChildren is the most children an all or any node has (D3); it has at
// least one.
const MaxChildren = 16

// The node kinds of D3, the keys a node object may hold (one of them).
const (
	NodeAll      = "all"
	NodeAny      = "any"
	NodeNot      = "not"
	NodeRef      = "ref"
	NodeMemberOf = "member_of"
	NodeCmp      = "cmp"
	NodeShared   = "shared"
	NodeUnder    = "under"
)

// nameRE is the pattern of a condition or policy name (D2).
var nameRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// ValidName reports whether s is a valid condition or policy name.
func ValidName(s string) bool { return nameRE.MatchString(s) }

// Document is a format-2 access document.
type Document struct {
	Format     int           `json:"format"`
	Conditions []Condition   `json:"conditions"`
	Policies   []NamedPolicy `json:"policies"`
}

// Condition is a named expression that nodes reach with ref.
type Condition struct {
	Name string `json:"name"`
	Expr Node   `json:"expr"`
}

// NamedPolicy grants Ops on the records where When holds (D1: allow-only).
//
// LegacyID is set only on a policy converted from a format-1 rule whose id is
// not a valid name: Name is then a valid name derived from it (convert.go) and
// LegacyID is the stored rule id, which the explainer shows. A format-2 input
// may not carry it.
type NamedPolicy struct {
	Name     string   `json:"name"`
	LegacyID string   `json:"legacy_id,omitempty"`
	Ops      []string `json:"ops"`
	When     Node     `json:"when"`
}

// Node is one expression node. Exactly one of its fields is set, the one its
// Kind names; the zero Node is not a valid node.
type Node struct {
	All      []Node
	Any      []Node
	Not      *Node
	Ref      string
	MemberOf *MemberOf
	Cmp      *Cond
	Shared   bool
	Under    *Under
}

// MemberOf holds when the caller is one of Groups (by its platform groups) or
// one of Users (by its principal id). At least one id is set.
type MemberOf struct {
	Groups []uuid.UUID `json:"groups,omitempty"`
	Users  []uuid.UUID `json:"users,omitempty"`
}

// Under holds when the record's org unit is in the subtree of the caller's
// live units (Caller) or of the listed units (Units). Exactly one is set. It
// decodes today but is not Available until S31 (registry.go).
type Under struct {
	Caller bool
	Units  []uuid.UUID
}

// Kind is the node's kind, the key it encodes with, or "" for a node with no
// field set.
func (n Node) Kind() string {
	switch {
	case n.All != nil:
		return NodeAll
	case n.Any != nil:
		return NodeAny
	case n.Not != nil:
		return NodeNot
	case n.Ref != "":
		return NodeRef
	case n.MemberOf != nil:
		return NodeMemberOf
	case n.Cmp != nil:
		return NodeCmp
	case n.Shared:
		return NodeShared
	case n.Under != nil:
		return NodeUnder
	}
	return ""
}

// MarshalJSON encodes the node as its one-key object. A node with no kind
// fails: it can only be built by hand, never decoded.
func (n Node) MarshalJSON() ([]byte, error) {
	var v any
	switch n.Kind() {
	case NodeAll:
		v = n.All
	case NodeAny:
		v = n.Any
	case NodeNot:
		v = n.Not
	case NodeRef:
		v = n.Ref
	case NodeMemberOf:
		v = n.MemberOf
	case NodeCmp:
		v = n.Cmp
	case NodeShared:
		v = true
	case NodeUnder:
		v = n.Under
	default:
		return nil, errors.New("authz: node has no kind")
	}
	return json.Marshal(map[string]any{n.Kind(): v})
}

// MarshalJSON encodes Under as {"of": "caller"} or {"of": [ids]}.
func (u Under) MarshalJSON() ([]byte, error) {
	if u.Caller {
		return []byte(`{"of":"caller"}`), nil
	}
	return json.Marshal(map[string]any{"of": u.Units})
}

// Encode is the stored form of a format-2 document: its JSON, with empty
// arrays rather than nulls.
func Encode(d Document) ([]byte, error) {
	if d.Conditions == nil {
		d.Conditions = []Condition{}
	}
	if d.Policies == nil {
		d.Policies = []NamedPolicy{}
	}
	return json.Marshal(d)
}

// DecodeError is a document that does not decode: every problem found, sorted
// by pointer then code. Each pointer is a JSON pointer into the document (the
// stored rules value); the publish route prefixes it with "/rules".
type DecodeError struct {
	Problems []Problem
}

func (e *DecodeError) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		parts[i] = fmt.Sprintf("%s at %q", p.Rule, p.Pointer)
	}
	return "authz: invalid access document: " + strings.Join(parts, ", ")
}

// Decode returns the format-2 document of a stored or submitted rules value:
// a format-2 object is checked and decoded, a format-1 array is checked as
// today's publish check does and converted (D12). On any problem it returns a
// *DecodeError and a zero Document.
func Decode(raw []byte) (Document, error) {
	switch firstByte(raw) {
	case '[':
		return decodeFormat1(raw)
	case '{':
		return decodeFormat2(raw)
	}
	return Document{}, &DecodeError{Problems: []Problem{{"", RuleType}}}
}

func firstByte(raw []byte) byte {
	t := bytes.TrimLeft(raw, " \t\r\n")
	if len(t) == 0 {
		return 0
	}
	return t[0]
}

// decodeFormat1 checks a format-1 rules array with the format-1 publish
// checks, then converts it.
func decodeFormat1(raw []byte) (Document, error) {
	c := &policyCheck{}
	c.duplicateKeys(raw)
	var arr []any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return Document{}, &DecodeError{Problems: []Problem{{"", RuleType}}}
	}
	seen := map[string]bool{}
	for i, r := range arr {
		c.rule(child("", i), r, seen)
	}
	if len(c.problems) > 0 {
		return Document{}, c.decodeError()
	}
	var rules []Rule
	if err := decodeStrict(raw, &rules); err != nil {
		return Document{}, &DecodeError{Problems: []Problem{{"", RuleType}}}
	}
	return ConvertFormat1(rules), nil
}

// decodeFormat2 checks a format-2 object on its decoded JSON tree (so a null
// or a number for a bool is refused, as the CHECK refuses it), then builds the
// Document from the same tree.
func decodeFormat2(raw []byte) (Document, error) {
	c := &policyCheck{}
	c.duplicateKeys(raw)
	var top any
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&top); err != nil || dec.More() {
		return Document{}, &DecodeError{Problems: []Problem{{"", RuleType}}}
	}
	obj, _ := top.(map[string]any) // firstByte was '{'
	for k := range obj {
		if k != "format" && k != "conditions" && k != "policies" {
			c.bad(child("", k), RuleUnknownKey)
		}
	}
	switch f, present := obj["format"]; {
	case !present:
		c.bad("/format", RuleRequired)
	default:
		n, isNum := f.(float64)
		switch {
		case !isNum:
			c.bad("/format", RuleType)
		case n != FormatV2:
			c.bad("/format", RuleValue)
		}
	}
	d := Document{Format: FormatV2, Conditions: []Condition{}, Policies: []NamedPolicy{}}
	if arr, ok := c.array(obj, "", "conditions"); ok {
		names := map[string]bool{}
		for i, e := range arr {
			ptr := child("/conditions", i)
			o, ok := c.object(ptr, e, "name", "expr")
			if !ok {
				continue
			}
			cd := Condition{Name: c.name(o, ptr, names)}
			if x, present := o["expr"]; present {
				cd.Expr = c.node(child(ptr, "expr"), x)
			} else {
				c.bad(child(ptr, "expr"), RuleRequired)
			}
			d.Conditions = append(d.Conditions, cd)
		}
	}
	if arr, ok := c.array(obj, "", "policies"); ok {
		names := map[string]bool{}
		for i, e := range arr {
			ptr := child("/policies", i)
			o, ok := c.object(ptr, e, "name", "ops", "when")
			if !ok {
				continue
			}
			p := NamedPolicy{Name: c.name(o, ptr, names)}
			p.Ops = c.ops(o, ptr)
			if x, present := o["when"]; present {
				p.When = c.node(child(ptr, "when"), x)
			} else {
				c.bad(child(ptr, "when"), RuleRequired)
			}
			d.Policies = append(d.Policies, p)
		}
	}
	if len(c.problems) > 0 {
		return Document{}, c.decodeError()
	}
	return d, nil
}

func (c *policyCheck) decodeError() *DecodeError {
	ps := slices.Clone(c.problems)
	slices.SortFunc(ps, func(a, b Problem) int {
		if x := strings.Compare(a.Pointer, b.Pointer); x != 0 {
			return x
		}
		return strings.Compare(a.Rule, b.Rule)
	})
	return &DecodeError{Problems: slices.Compact(ps)}
}

// array reads the required array obj[key].
func (c *policyCheck) array(obj map[string]any, ptr, key string) ([]any, bool) {
	v, present := obj[key]
	if !present {
		c.bad(child(ptr, key), RuleRequired)
		return nil, false
	}
	arr, ok := v.([]any)
	if !ok {
		c.bad(child(ptr, key), RuleType)
	}
	return arr, ok
}

// name reads obj["name"], a valid name not yet in seen, and adds it.
func (c *policyCheck) name(obj map[string]any, ptr string, seen map[string]bool) string {
	ptr = child(ptr, "name")
	v, present := obj["name"]
	if !present {
		c.bad(ptr, RuleRequired)
		return ""
	}
	s, ok := v.(string)
	switch {
	case !ok:
		c.bad(ptr, RuleType)
	case !ValidName(s):
		c.bad(ptr, RuleInvalidName)
	case seen[s]:
		c.bad(ptr, RuleDuplicateName)
	default:
		seen[s] = true
	}
	return s
}

// ops reads obj["ops"]: a non-empty array of read and write, as a format-1
// rule's ops.
func (c *policyCheck) ops(obj map[string]any, ptr string) []string {
	ptr = child(ptr, "ops")
	v, present := obj["ops"]
	if !present {
		c.bad(ptr, RuleRequired)
		return nil
	}
	arr, ok := v.([]any)
	switch {
	case !ok:
		c.bad(ptr, RuleType)
		return nil
	case len(arr) == 0:
		c.bad(ptr, RuleEmpty)
		return nil
	}
	out := make([]string, 0, len(arr))
	for i, o := range arr {
		s, isStr := o.(string)
		switch {
		case !isStr:
			c.bad(child(ptr, i), RuleType)
		case s != OpRead && s != OpWrite:
			c.bad(child(ptr, i), RuleValue)
		default:
			out = append(out, s)
		}
	}
	return out
}

// node checks one expression node and returns it. A node object has exactly
// one key: none or two is unknown_key at the node, one key outside D3 is
// unknown_key at that key.
func (c *policyCheck) node(ptr string, v any) Node {
	obj, ok := v.(map[string]any)
	if !ok {
		c.bad(ptr, RuleType)
		return Node{}
	}
	if len(obj) != 1 {
		c.bad(ptr, RuleUnknownKey)
		return Node{}
	}
	var n Node
	for k, x := range obj {
		kp := child(ptr, k)
		switch k {
		case NodeAll:
			n.All = c.children(kp, x)
		case NodeAny:
			n.Any = c.children(kp, x)
		case NodeNot:
			inner := c.node(kp, x)
			n.Not = &inner
		case NodeRef:
			s, isStr := x.(string)
			switch {
			case !isStr:
				c.bad(kp, RuleType)
			case !ValidName(s):
				c.bad(kp, RuleInvalidName)
			default:
				n.Ref = s
			}
		case NodeMemberOf:
			n.MemberOf = c.memberOf(kp, x)
		case NodeCmp:
			n.Cmp = c.cmp(kp, x)
		case NodeShared:
			if b, isBool := x.(bool); !isBool {
				c.bad(kp, RuleType)
			} else if !b {
				c.bad(kp, RuleValue)
			}
			n.Shared = true
		case NodeUnder:
			n.Under = c.under(kp, x)
		default:
			c.bad(kp, RuleUnknownKey)
		}
	}
	return n
}

// children reads the 1 to MaxChildren nodes of an all or any. It never
// returns nil, so the node keeps its kind.
func (c *policyCheck) children(ptr string, v any) []Node {
	arr, ok := v.([]any)
	if !ok {
		c.bad(ptr, RuleType)
		return []Node{}
	}
	if len(arr) == 0 || len(arr) > MaxChildren {
		c.bad(ptr, RuleChildCount)
	}
	out := make([]Node, len(arr))
	for i, e := range arr {
		out[i] = c.node(child(ptr, i), e)
	}
	return out
}

// memberOf reads {"groups": [ids], "users": [ids]} with at least one id,
// recording each id as a tenant reference.
func (c *policyCheck) memberOf(ptr string, v any) *MemberOf {
	before := len(c.problems)
	obj, ok := c.object(ptr, v, "groups", "users")
	if !ok {
		return &MemberOf{}
	}
	c.subject(ptr, obj)
	m := &MemberOf{}
	for _, f := range []struct {
		key string
		dst *[]uuid.UUID
	}{{"groups", &m.Groups}, {"users", &m.Users}} {
		arr, _ := obj[f.key].([]any)
		for _, e := range arr {
			if s, isStr := e.(string); isStr {
				if id, err := uuid.Parse(s); err == nil {
					*f.dst = append(*f.dst, id)
				}
			}
		}
	}
	if len(c.problems) == before && len(m.Groups) == 0 && len(m.Users) == 0 {
		c.bad(ptr, RuleEmpty)
	}
	return m
}

// cmp reads today's Cond, with today's rules (policycheck.go cond).
func (c *policyCheck) cmp(ptr string, v any) *Cond {
	before := len(c.problems)
	c.cond(ptr, v)
	if len(c.problems) > before {
		return &Cond{}
	}
	// The shape is valid, so this cannot fail.
	b, _ := json.Marshal(v)
	var cd Cond
	_ = json.Unmarshal(b, &cd)
	return &cd
}

// under reads {"of": "caller"} or {"of": [ids]} with at least one id.
func (c *policyCheck) under(ptr string, v any) *Under {
	obj, ok := c.object(ptr, v, "of")
	if !ok {
		return &Under{}
	}
	op := child(ptr, "of")
	u := &Under{}
	switch of := obj["of"].(type) {
	case nil:
		if _, present := obj["of"]; present {
			c.bad(op, RuleType)
		} else {
			c.bad(op, RuleRequired)
		}
	case string:
		if of != "caller" {
			c.bad(op, RuleValue)
		}
		u.Caller = true
	case []any:
		if len(of) == 0 {
			c.bad(op, RuleEmpty)
		}
		for i, e := range of {
			s, isStr := e.(string)
			if !isStr {
				c.bad(child(op, i), RuleType)
				continue
			}
			id, err := uuid.Parse(s)
			if !uuidRE.MatchString(s) || err != nil {
				c.bad(child(op, i), RuleValue)
				continue
			}
			u.Units = append(u.Units, id)
		}
	default:
		c.bad(op, RuleType)
	}
	return u
}

// duplicateKeys reports every object key that repeats in its object as
// unknown_key at the repeated key: encoding/json silently keeps the last one,
// and PostgreSQL's jsonb does the same, so a duplicate is refused before
// either can drop it. raw that is not JSON is left to the decode.
func (c *policyCheck) duplicateKeys(raw []byte) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	_ = c.scanValue(dec, "")
}

// scanValue reads one JSON value from dec, tracking its pointer.
func (c *policyCheck) scanValue(dec *json.Decoder, ptr string) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch d {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return err
			}
			k, _ := kt.(string)
			if seen[k] {
				c.bad(child(ptr, k), RuleUnknownKey)
			}
			seen[k] = true
			if err := c.scanValue(dec, child(ptr, k)); err != nil {
				return err
			}
		}
	case '[':
		for i := 0; dec.More(); i++ {
			if err := c.scanValue(dec, child(ptr, i)); err != nil {
				return err
			}
		}
	}
	_, err = dec.Token() // the closing delimiter
	return err
}
