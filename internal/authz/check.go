package authz

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/acl"
)

// The save-time checks of an access policy (ADR-010 D4, amendment A1; PLAN-011
// C1; TICK-939 P11-07). Prepare is the one check function: the publish route
// calls it on the request body, and PublishIfNone (seed and onboarding) calls
// it on the rules it is given, so nothing can store a document the route would
// refuse. It is pure: the checks that need the database (every group and user
// id is the tenant's, every mask field is a schema property) take the ids and
// the masks it returns.
//
// The database CHECK (db/migrations/20261119100000, TICK-940) is the backstop.
// It names the first problem its walk meets; Prepare names all of them, and the
// pointer and code of the database's problem is always one of Prepare's for the
// same document (docs/data/acl-format2-checks.md; the shared fixture
// testdata/format2_invalid.json asserts it).

// The caps of A1-1: platform limits, not settings. MaxChildren is in
// document.go; MaxDepth and MaxPolicyLeaves are in plan.go. They are the
// database's constants (acl2_problem) as well.
const (
	// MaxPolicies is the most policies a document holds.
	MaxPolicies = 64
	// MaxDocumentLeaves is the most leaves the policies of a document have in
	// all, after refs expand. Conditions no policy uses are not counted.
	MaxDocumentLeaves = 128
)

// DefaultMaxDocumentBytes is the byte limit on rules and on masks when the
// operator sets none (A1-2a); config.DefaultACLMaxDocumentBytes is the same
// number. nava-onboard publishes with it.
const DefaultMaxDocumentBytes = 32768

// WarnNotOverCaller is the code of the lint on a `not` over caller-dependent
// leaves (TICK-937 security review, finding 2). Under D5a a `not` over a leaf
// that depends on the caller folds to true for a principal with no id, which
// would admit every row of the tenant. It warns; it does not refuse.
const WarnNotOverCaller = "not_over_caller_leaf"

// leafSat is where leaf counts stop growing, so a doubling chain of
// conditions cannot overflow.
const leafSat = 1_000_000_000

// Warning is a lint on a document that is accepted. Pointer is into the body.
type Warning struct {
	Pointer string
	Code    string
}

// Prepared is a publish body that passed every check that needs no database.
type Prepared struct {
	// Format is the format of the submitted rules, 1 or 2.
	Format int
	// Document is the format-2 document that is stored.
	Document Document
	// Rules is the stored text of Document: PostgreSQL's jsonb text form
	// (acl.CanonicalJSON), never the client's bytes.
	Rules []byte
	// Masks is the stored text of the masks, in the same form.
	Masks []byte
	// MaskList is the masks, for the field check against the schema.
	MaskList []Mask
	// Refs are the group and user ids of the body, to check against the
	// tenant, with their pointers into the body.
	Refs []TenantRef
	// Warnings are the lints. Sorted by pointer.
	Warnings []Warning
}

// Prepare checks a publish body `{"rules": <format 1 or 2>, "masks": [...]}`
// and returns what to store, or every problem with a pointer into the body.
// maxBytes is the limit on the stored text of rules and of masks (A1-2c); a
// body above it is refused with document_too_large and nothing else is checked.
func Prepare(body []byte, maxBytes int) (*Prepared, []Problem) {
	var top any
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&top); err != nil || dec.More() {
		return nil, []Problem{{"", RuleType}}
	}
	obj, ok := top.(map[string]any)
	if !ok {
		return nil, []Problem{{"", RuleType}}
	}
	var raw struct {
		Rules json.RawMessage `json:"rules"`
		Masks json.RawMessage `json:"masks"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, []Problem{{"", RuleType}}
	}
	c := &policyCheck{}
	for k := range obj {
		if k != "rules" && k != "masks" {
			c.bad(child("", k), RuleUnknownKey)
		}
	}
	if _, present := obj["rules"]; !present {
		c.bad("/rules", RuleRequired)
	}
	var p *Prepared
	var ps []Problem
	switch firstByte(raw.Rules) {
	case '{':
		p, ps = prepareFormat2(c, obj, raw.Rules, raw.Masks, maxBytes)
	case '[':
		p, ps = prepareFormat1(body, raw.Masks, maxBytes)
		ps = append(ps, c.problems...)
	default:
		if _, present := obj["rules"]; present {
			c.bad("/rules", RuleType)
		}
		ps = c.problems
	}
	if len(ps) > 0 {
		return nil, sortProblems(ps)
	}
	if p == nil {
		return nil, []Problem{{"", RuleType}}
	}
	slices.SortFunc(p.Warnings, func(a, b Warning) int { return strings.Compare(a.Pointer, b.Pointer) })
	return p, nil
}

// sortProblems orders problems the way the database's foreign_id is ordered:
// rules before masks, then by pointer, then by code. It drops repeats.
func sortProblems(ps []Problem) []Problem {
	rank := func(p Problem) int {
		if strings.HasPrefix(p.Pointer, "/masks") {
			return 1
		}
		return 0
	}
	slices.SortStableFunc(ps, func(a, b Problem) int {
		if x := rank(a) - rank(b); x != 0 {
			return x
		}
		if x := strings.Compare(a.Pointer, b.Pointer); x != 0 {
			return x
		}
		return strings.Compare(a.Rule, b.Rule)
	})
	return slices.Compact(ps)
}

// storedMasks returns the canonical text of the masks (an empty array when
// the body has none) or the problem that stops it.
func storedMasks(masksRaw json.RawMessage) ([]byte, *Problem) {
	if len(bytes.TrimSpace(masksRaw)) == 0 {
		return []byte("[]"), nil
	}
	b, err := acl.CanonicalJSON(masksRaw)
	if err != nil {
		return nil, &Problem{"/masks", RuleValue}
	}
	return b, nil
}

func prepareFormat1(body []byte, masksRaw json.RawMessage, maxBytes int) (*Prepared, []Problem) {
	rules, masks, refs, ps := ParsePolicy(body)
	if len(ps) > 0 {
		return nil, ps
	}
	// A rule with no subject converts to an empty any, which the format-2
	// grammar refuses (child_count) and which matches nothing; it is refused
	// here, at the rule's who, rather than stored.
	for i, r := range rules {
		if len(r.Who.Groups) == 0 && len(r.Who.Users) == 0 && !r.Who.Requester && !r.Who.Assignee && !r.Who.Shared {
			ps = append(ps, Problem{child(child(child("", "rules"), i), "who"), RuleChildCount})
		}
	}
	if len(ps) > 0 {
		return nil, ps
	}
	doc := ConvertFormat1(rules)
	// A converted policy's LegacyID is not stored: format-2 input refuses it
	// (unknown_key), and so does the database. The derived name is.
	for i := range doc.Policies {
		doc.Policies[i].LegacyID = ""
	}
	enc, err := Encode(doc)
	if err != nil {
		return nil, []Problem{{"/rules", RuleType}}
	}
	return finish(1, doc, enc, masks, masksRaw, refs, maxBytes, func(ptr string) string {
		// The caps are reported at the format-1 rule: /policies/3/when is
		// /rules/3, /policies is /rules.
		if rest, ok := strings.CutPrefix(ptr, "/policies"); ok {
			idx, _, _ := strings.Cut(strings.TrimPrefix(rest, "/"), "/")
			if idx == "" {
				return "/rules"
			}
			return "/rules/" + idx
		}
		return "/rules" + ptr
	})
}

func prepareFormat2(c *policyCheck, obj map[string]any, rulesRaw, masksRaw json.RawMessage, maxBytes int) (*Prepared, []Problem) {
	// The byte limit first, as the database orders its CHECKs: an oversize
	// value is refused before anything walks it.
	var ps []Problem
	for _, f := range []struct {
		ptr string
		raw json.RawMessage
	}{{"/rules", rulesRaw}, {"/masks", masksRaw}} {
		if len(bytes.TrimSpace(f.raw)) == 0 {
			continue
		}
		n, err := acl.CanonicalSize(f.raw)
		switch {
		case err != nil:
			ps = append(ps, Problem{f.ptr, RuleValue})
		case n > maxBytes:
			ps = append(ps, Problem{f.ptr, RuleDocumentTooLarge})
		}
	}
	if len(ps) > 0 {
		return nil, ps
	}
	c.duplicateKeys(masksRawOrNull(masksRaw)) // pointers are relative to masks, prefixed below
	maskProblems := c.problems
	c.problems = nil
	for _, p := range maskProblems {
		if p.Rule == RuleUnknownKey && strings.HasPrefix(p.Pointer, "/") {
			// A duplicate key inside the body's masks.
			c.bad("/masks"+p.Pointer, p.Rule)
			continue
		}
		c.bad(p.Pointer, p.Rule)
	}
	c.masks(obj)
	ps = slices.Clone(c.problems)
	doc, err := Decode(rulesRaw)
	var de *DecodeError
	switch {
	case errors.As(err, &de):
		for _, p := range de.Problems {
			ps = append(ps, Problem{"/rules" + p.Pointer, p.Rule})
		}
		if n := policyCount(rulesRaw); n > MaxPolicies {
			ps = append(ps, Problem{"/rules/policies", RulePoliciesExceeded})
		}
	case err != nil:
		ps = append(ps, Problem{"/rules", RuleType})
	}
	if len(ps) > 0 {
		return nil, ps
	}
	canon, err := acl.CanonicalJSON(rulesRaw)
	if err != nil {
		return nil, []Problem{{"/rules", RuleValue}}
	}
	var masks []Mask
	if len(bytes.TrimSpace(masksRaw)) > 0 {
		if err := decodeStrict(masksRaw, &masks); err != nil {
			return nil, []Problem{{"/masks", RuleType}}
		}
	}
	p, ps := finish(2, doc, nil, masks, masksRaw, c.refs, maxBytes, func(ptr string) string { return "/rules" + ptr })
	if p != nil {
		p.Rules = canon
	}
	return p, ps
}

func masksRawOrNull(m json.RawMessage) []byte {
	if len(bytes.TrimSpace(m)) == 0 {
		return []byte("null")
	}
	return m
}

// policyCount is the length of the policies array of a rules object that did
// not decode, or 0.
func policyCount(rulesRaw []byte) int {
	var v struct {
		Policies []json.RawMessage `json:"policies"`
	}
	_ = json.Unmarshal(rulesRaw, &v)
	return len(v.Policies)
}

// finish runs the document checks (D4) and builds the Prepared. For format 1,
// enc is the encoded converted document, which is also the stored text; the
// size limit applies to it. at maps a document pointer to a body pointer.
func finish(format int, doc Document, enc []byte, masks []Mask, masksRaw json.RawMessage, refs []TenantRef, maxBytes int, at func(string) string) (*Prepared, []Problem) {
	var ps []Problem
	if enc != nil {
		canon, err := acl.CanonicalJSON(enc)
		switch {
		case err != nil:
			ps = append(ps, Problem{"/rules", RuleValue})
		case len(canon) > maxBytes:
			ps = append(ps, Problem{"/rules", RuleDocumentTooLarge})
		}
		enc = canon
	}
	mb, mp := storedMasks(masksRaw)
	switch {
	case mp != nil:
		ps = append(ps, *mp)
	case len(mb) > maxBytes:
		ps = append(ps, Problem{"/masks", RuleDocumentTooLarge})
	}
	if len(ps) > 0 {
		return nil, ps
	}
	ck := checkDocument(doc)
	for _, p := range ck.problems {
		ps = append(ps, Problem{at(p.Pointer), p.Rule})
	}
	if len(ps) > 0 {
		return nil, ps
	}
	if format == 2 {
		for _, r := range ck.refs {
			refs = append(refs, TenantRef{at(r.Pointer), r.Kind, r.ID})
		}
	}
	p := &Prepared{Format: format, Document: doc, Rules: enc, Masks: mb, MaskList: masks, Refs: refs}
	for _, w := range ck.warnings {
		p.Warnings = append(p.Warnings, Warning{at(w.Pointer), w.Code})
	}
	return p, nil
}

// docCheck is the walk over one document.
type docCheck struct {
	problems []Problem
	refs     []TenantRef // group and user ids, pointers into the document
	warnings []Warning
	nots     []notUse
}

// notUse is one outermost `not`: whether a caller-dependent leaf is directly
// under it, and the conditions it refs.
type notUse struct {
	ptr    string
	direct bool
	refs   []string
}

// exprInfo is what one expression tree holds before refs expand.
type exprInfo struct {
	leaves int64
	depth  int
	refs   []refUse
}

type refUse struct {
	name  string
	ptr   string
	depth int
}

func satAdd(a, b int64) int64 { return min(leafSat, a+b) }

// checkDocument runs the D4 checks that need nothing but the document.
// Pointers are into the document.
func checkDocument(d Document) *docCheck {
	k := &docCheck{}
	if len(d.Policies) > MaxPolicies {
		k.problems = append(k.problems, Problem{"/policies", RulePoliciesExceeded})
	}
	idx := make(map[string]int, len(d.Conditions))
	for i, c := range d.Conditions {
		idx[c.Name] = i
	}
	conds := make([]exprInfo, len(d.Conditions))
	for i, c := range d.Conditions {
		k.scan(c.Expr, "/conditions/"+strconv.Itoa(i)+"/expr", 1, &conds[i], -1)
	}
	pols := make([]exprInfo, len(d.Policies))
	for i, p := range d.Policies {
		k.scan(p.When, "/policies/"+strconv.Itoa(i)+"/when", 1, &pols[i], -1)
	}

	// Every ref resolves.
	for _, in := range append(slices.Clone(conds), pols...) {
		for _, r := range in.refs {
			if _, ok := idx[r.name]; !ok {
				k.problems = append(k.problems, Problem{r.ptr, RuleUnresolvedRef})
			}
		}
	}

	// Leaves and depth per condition, in Kahn's order over the reference
	// graph, never expanded, so a doubling chain costs its written size.
	nc := len(conds)
	type edge struct{ from, to, depth int }
	var edges []edge
	var edgePtr []string
	outdeg := make([]int, nc)
	rev := make([][]int, nc)
	for i, in := range conds {
		for _, r := range in.refs {
			if j, ok := idx[r.name]; ok {
				edges = append(edges, edge{i, j, r.depth})
				edgePtr = append(edgePtr, r.ptr)
				rev[j] = append(rev[j], len(edges)-1)
				outdeg[i]++
			}
		}
	}
	tl := make([]int64, nc)
	td := make([]int, nc)
	accL := make([]int64, nc)
	accD := make([]int, nc)
	done := make([]bool, nc)
	var queue []int
	for i := range conds {
		if outdeg[i] == 0 {
			tl[i], td[i], done[i] = conds[i].leaves, conds[i].depth, true
			queue = append(queue, i)
		}
	}
	for h := 0; h < len(queue); h++ {
		j := queue[h]
		for _, e := range rev[j] {
			f := edges[e].from
			accL[f] = satAdd(accL[f], tl[j])
			accD[f] = max(accD[f], edges[e].depth+td[j]-1)
			outdeg[f]--
			if outdeg[f] == 0 {
				tl[f], td[f], done[f] = satAdd(conds[f].leaves, accL[f]), max(conds[f].depth, accD[f]), true
				queue = append(queue, f)
			}
		}
	}
	if len(queue) < nc {
		// A condition still waiting is on a cycle or reaches one. Name the
		// first ref, in document order, from a waiting condition to a waiting
		// one, as the database does.
		for e, ed := range edges {
			if !done[ed.from] && !done[ed.to] {
				k.problems = append(k.problems, Problem{edgePtr[e], RuleRefCycle})
				break
			}
		}
	}

	// The caps, per policy, on the expanded counts.
	var total int64
	totalReported := false
	for i, in := range pols {
		ptr := "/policies/" + strconv.Itoa(i) + "/when"
		pl, pd, ok := in.leaves, in.depth, true
		for _, r := range in.refs {
			j, found := idx[r.name]
			if !found || !done[j] {
				ok = false
				break
			}
			pl = satAdd(pl, tl[j])
			pd = max(pd, r.depth+td[j]-1)
		}
		if !ok {
			continue
		}
		if pd > MaxDepth {
			k.problems = append(k.problems, Problem{ptr, RuleDepthExceeded})
		}
		if pl > MaxPolicyLeaves {
			k.problems = append(k.problems, Problem{ptr, RulePolicyLeavesExceeded})
		}
		total = satAdd(total, pl)
		if total > MaxDocumentLeaves && !totalReported {
			totalReported = true
			k.problems = append(k.problems, Problem{"/policies", RuleDocLeavesExceeded})
		}
	}
	k.lintNots(d, idx)
	return k
}

// scan walks one expression tree: it counts leaves and depth, collects refs,
// the group and user ids, the kinds that are not available yet, and the facts
// the `not` lint needs. d is the depth of n; outer is the index of the
// outermost `not` above n in k.nots, or -1.
func (k *docCheck) scan(n Node, ptr string, d int, in *exprInfo, outer int) {
	kind := n.Kind()
	if nk, ok := LookupKind(kind); ok && !nk.Available {
		k.problems = append(k.problems, Problem{ptr + "/" + kind, RuleNotAvailable})
	}
	kp := ptr + "/" + kind
	leaf := func() {
		in.leaves = satAdd(in.leaves, 1)
		in.depth = max(in.depth, d)
	}
	switch kind {
	case NodeAll, NodeAny:
		kids := n.All
		if kind == NodeAny {
			kids = n.Any
		}
		for i, c := range kids {
			k.scan(c, kp+"/"+strconv.Itoa(i), d+1, in, outer)
		}
	case NodeNot:
		if outer < 0 {
			k.nots = append(k.nots, notUse{ptr: kp})
			outer = len(k.nots) - 1
		}
		k.scan(*n.Not, kp, d+1, in, outer)
	case NodeRef:
		in.refs = append(in.refs, refUse{n.Ref, kp, d})
		if outer >= 0 {
			k.nots[outer].refs = append(k.nots[outer].refs, n.Ref)
		}
	case NodeMemberOf:
		leaf()
		sub := &policyCheck{}
		sub.subject(kp, map[string]any{"groups": uuidTexts(n.MemberOf.Groups), "users": uuidTexts(n.MemberOf.Users)})
		k.refs = append(k.refs, sub.refs...)
		if outer >= 0 {
			k.nots[outer].direct = true // member_of depends on the caller
		}
	case NodeCmp:
		leaf()
		b, _ := json.Marshal(n.Cmp)
		var v any
		_ = json.Unmarshal(b, &v)
		sub := &policyCheck{}
		sub.cond(kp, v)
		k.refs = append(k.refs, sub.refs...)
		if n.Cmp.Caller != "" && outer >= 0 {
			k.nots[outer].direct = true
		}
	case NodeShared:
		leaf()
		if outer >= 0 {
			k.nots[outer].direct = true
		}
	case NodeUnder:
		leaf()
	}
}

func uuidTexts(ids []uuid.UUID) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// lintNots warns on each outermost `not` over a caller-dependent leaf, directly
// or through the conditions it refs.
func (k *docCheck) lintNots(d Document, idx map[string]int) {
	if len(k.nots) == 0 {
		return
	}
	// A condition is caller-dependent when it holds a caller-dependent leaf or
	// refs one that does. A fixed point over the reference graph.
	dep := make([]bool, len(d.Conditions))
	for i, c := range d.Conditions {
		dep[i] = callerLeaf(c.Expr)
	}
	for changed := true; changed; {
		changed = false
		for i, c := range d.Conditions {
			if dep[i] {
				continue
			}
			for _, name := range refNames(c.Expr, nil) {
				if j, ok := idx[name]; ok && dep[j] {
					dep[i], changed = true, true
					break
				}
			}
		}
	}
	for _, u := range k.nots {
		warn := u.direct
		for _, name := range u.refs {
			if j, ok := idx[name]; ok && dep[j] {
				warn = true
			}
		}
		if warn {
			k.warnings = append(k.warnings, Warning{u.ptr, WarnNotOverCaller})
		}
	}
}

// callerLeaf reports whether n holds a leaf whose value depends on the caller:
// a cmp with a caller value (self, my_groups), shared or member_of.
func callerLeaf(n Node) bool {
	switch n.Kind() {
	case NodeAll:
		return slices.ContainsFunc(n.All, callerLeaf)
	case NodeAny:
		return slices.ContainsFunc(n.Any, callerLeaf)
	case NodeNot:
		return callerLeaf(*n.Not)
	case NodeCmp:
		return n.Cmp.Caller != ""
	case NodeShared, NodeMemberOf:
		return true
	}
	return false
}

func refNames(n Node, out []string) []string {
	switch n.Kind() {
	case NodeAll:
		for _, c := range n.All {
			out = refNames(c, out)
		}
	case NodeAny:
		for _, c := range n.Any {
			out = refNames(c, out)
		}
	case NodeNot:
		out = refNames(*n.Not, out)
	case NodeRef:
		out = append(out, n.Ref)
	}
	return out
}

// CheckSize is the byte limit alone (A1-2c), for the route to apply before it
// opens a transaction: the stored text of a format-2 rules value and of the
// masks against maxBytes. A body that is not JSON, or whose format-1 rules
// convert to a larger document, is left to Prepare.
func CheckSize(body []byte, maxBytes int) []Problem {
	var raw struct {
		Rules json.RawMessage `json:"rules"`
		Masks json.RawMessage `json:"masks"`
	}
	if json.Unmarshal(body, &raw) != nil {
		return nil
	}
	var ps []Problem
	for _, f := range []struct {
		ptr string
		raw json.RawMessage
		ok  bool
	}{{"/rules", raw.Rules, firstByte(raw.Rules) == '{'}, {"/masks", raw.Masks, true}} {
		if !f.ok || len(bytes.TrimSpace(f.raw)) == 0 {
			continue
		}
		if n, err := acl.CanonicalSize(f.raw); err == nil && n > maxBytes {
			ps = append(ps, Problem{f.ptr, RuleDocumentTooLarge})
		}
	}
	return ps
}
