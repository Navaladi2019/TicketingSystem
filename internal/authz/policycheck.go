package authz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// Publish validation (PLAN-003 S10, P3-35). ParsePolicy checks a request body
// for the ACL publish route and returns the policy it holds, or every problem
// with a JSON pointer into the body. It runs the shape rules of the database
// CHECK (acl_rules_valid and acl_masks_valid,
// docs/data/acl-versions-shape-check.md) first, so a user gets a 422 with
// pointers and never a raw constraint error, and then the rules only a publish
// can check: unique rule ids, masks that are row-independent (A2) and whose
// write set is within the read set (R3). What needs the database or the
// schema, the tenant of every id and the mask fields, is checked by the
// caller with the ids this returns (TenantRefs) and CheckMaskFields.
//
// Everything is judged on the decoded JSON tree, not on the Go types: the
// strict decoder (decodeStrict) and the CHECK see the stored bytes, and a
// value the typed decode would silently coerce (a null, a number for a bool)
// must be refused here, as they refuse it.

// The rules of a Problem.
const (
	RuleRequired   = "required"
	RuleType       = "type"
	RuleUnknownKey = "unknown_key"
	RuleValue      = "value"
	RuleEmpty      = "empty"
	RuleDuplicate  = "duplicate"
	RuleCondition  = "condition"
	RuleMaskSubj   = "mask_subject"
	RuleMaskWrite  = "mask_write"
	RuleField      = "field"
	RuleNotTenant  = "not_in_tenant"
)

// Problem is one thing wrong with a publish body. Pointer is a JSON pointer
// into the body; Rule is one of the Rule constants. Neither carries a value
// from the request.
type Problem struct {
	Pointer string
	Rule    string
}

// Kind says which table a TenantRef id must be found in.
type Kind int

// The kinds of id a policy names.
const (
	KindGroup Kind = iota + 1
	KindUser
)

// TenantRef is an id of a policy, with the pointer it was read at, that must
// belong to the publishing tenant.
type TenantRef struct {
	Pointer string
	Kind    Kind
	ID      uuid.UUID
}

var uuidRE = regexp.MustCompile(`^[0-9A-Fa-f]{8}-([0-9A-Fa-f]{4}-){3}[0-9A-Fa-f]{12}$`)

type policyCheck struct {
	problems []Problem
	refs     []TenantRef
}

func (c *policyCheck) bad(ptr, rule string) { c.problems = append(c.problems, Problem{ptr, rule}) }

func child(ptr string, seg any) string {
	var s string
	switch v := seg.(type) {
	case int:
		s = strconv.Itoa(v)
	case string:
		s = v
	}
	// RFC 6901: ~ and / in a name are escaped. Names here are fixed keys or
	// indexes; an unknown key from the request is escaped too.
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '~':
			out = append(out, '~', '0')
		case '/':
			out = append(out, '~', '1')
		default:
			out = append(out, s[i])
		}
	}
	return ptr + "/" + string(out)
}

// ParsePolicy decodes and checks a publish body `{"rules": [...], "masks":
// [...]}`. rules is required, masks defaults to none. On any problem the
// returned policy parts are nil.
func ParsePolicy(raw []byte) (rules []Rule, masks []Mask, refs []TenantRef, problems []Problem) {
	c := &policyCheck{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	var top any
	if err := dec.Decode(&top); err != nil || dec.More() {
		return nil, nil, nil, []Problem{{"", RuleType}}
	}
	obj, ok := top.(map[string]any)
	if !ok {
		return nil, nil, nil, []Problem{{"", RuleType}}
	}
	for k := range obj {
		if k != "rules" && k != "masks" {
			c.bad(child("", k), RuleUnknownKey)
		}
	}
	ruleIDs := map[string]bool{}
	switch v, present := obj["rules"]; {
	case !present:
		c.bad("/rules", RuleRequired)
	default:
		arr, ok := v.([]any)
		if !ok {
			c.bad("/rules", RuleType)
			break
		}
		for i, r := range arr {
			c.rule(child("/rules", i), r, ruleIDs)
		}
	}
	if v, present := obj["masks"]; present {
		arr, ok := v.([]any)
		if !ok {
			c.bad("/masks", RuleType)
		} else {
			for i, m := range arr {
				c.mask(child("/masks", i), m)
			}
		}
	}
	if len(c.problems) > 0 {
		return nil, nil, nil, c.problems
	}
	// The shape is valid, so the strict typed decode cannot fail; it is still
	// the decode PolicyStore.Current does, so a failure is a refusal here
	// and never a stored row that reads as a 500.
	var body struct {
		Rules []Rule `json:"rules"`
		Masks []Mask `json:"masks"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&body); err != nil {
		return nil, nil, nil, []Problem{{"", RuleType}}
	}
	if body.Rules == nil {
		body.Rules = []Rule{}
	}
	if body.Masks == nil {
		body.Masks = []Mask{}
	}
	return body.Rules, body.Masks, c.refs, nil
}

func (c *policyCheck) object(ptr string, v any, keys ...string) (map[string]any, bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		c.bad(ptr, RuleType)
		return nil, false
	}
	for k := range obj {
		if !slices.Contains(keys, k) {
			c.bad(child(ptr, k), RuleUnknownKey)
		}
	}
	return obj, true
}

func (c *policyCheck) rule(ptr string, v any, seen map[string]bool) {
	obj, ok := c.object(ptr, v, "id", "ops", "who", "where")
	if !ok {
		return
	}
	switch id, present := obj["id"]; {
	case !present:
		c.bad(child(ptr, "id"), RuleRequired)
	default:
		s, isStr := id.(string)
		switch {
		case !isStr:
			c.bad(child(ptr, "id"), RuleType)
		case s == "":
			c.bad(child(ptr, "id"), RuleEmpty)
		case seen[s]:
			c.bad(child(ptr, "id"), RuleDuplicate)
		default:
			seen[s] = true
		}
	}
	switch ops, present := obj["ops"]; {
	case !present:
		c.bad(child(ptr, "ops"), RuleRequired)
	default:
		arr, isArr := ops.([]any)
		switch {
		case !isArr:
			c.bad(child(ptr, "ops"), RuleType)
		case len(arr) == 0:
			c.bad(child(ptr, "ops"), RuleEmpty)
		default:
			for i, o := range arr {
				if s, isStr := o.(string); !isStr {
					c.bad(child(child(ptr, "ops"), i), RuleType)
				} else if s != OpRead && s != OpWrite {
					c.bad(child(child(ptr, "ops"), i), RuleValue)
				}
			}
		}
	}
	if who, present := obj["who"]; !present {
		c.bad(child(ptr, "who"), RuleRequired)
	} else {
		c.subject(child(ptr, "who"), who)
	}
	if where, present := obj["where"]; present {
		arr, isArr := where.([]any)
		if !isArr {
			c.bad(child(ptr, "where"), RuleType)
			return
		}
		for i, cd := range arr {
			c.cond(child(child(ptr, "where"), i), cd)
		}
	}
}

// subject checks a Subject and records its ids as tenant references.
func (c *policyCheck) subject(ptr string, v any) {
	obj, ok := c.object(ptr, v, "groups", "users", "requester", "assignee", "shared")
	if !ok {
		return
	}
	for _, f := range []struct {
		key  string
		kind Kind
	}{{"groups", KindGroup}, {"users", KindUser}} {
		raw, present := obj[f.key]
		if !present {
			continue
		}
		arr, isArr := raw.([]any)
		if !isArr {
			c.bad(child(ptr, f.key), RuleType)
			continue
		}
		for i, e := range arr {
			ep := child(child(ptr, f.key), i)
			s, isStr := e.(string)
			if !isStr {
				c.bad(ep, RuleType)
				continue
			}
			id, err := uuid.Parse(s)
			if !uuidRE.MatchString(s) || err != nil {
				c.bad(ep, RuleValue)
				continue
			}
			c.refs = append(c.refs, TenantRef{ep, f.kind, id})
		}
	}
	for _, k := range []string{"requester", "assignee", "shared"} {
		if raw, present := obj[k]; present {
			if _, isBool := raw.(bool); !isBool {
				c.bad(child(ptr, k), RuleType)
			}
		}
	}
}

func (c *policyCheck) cond(ptr string, v any) {
	obj, ok := c.object(ptr, v, "column", "op", "values", "caller")
	if !ok {
		return
	}
	column, colOK := "", false
	switch raw, present := obj["column"]; {
	case !present:
		c.bad(child(ptr, "column"), RuleRequired)
	default:
		s, isStr := raw.(string)
		switch {
		case !isStr:
			c.bad(child(ptr, "column"), RuleType)
		case s != metadata.ColumnState && s != metadata.ColumnRequesterID && s != metadata.ColumnAssigneeID && s != metadata.ColumnAssignmentGroupID:
			c.bad(child(ptr, "column"), RuleValue)
		default:
			column, colOK = s, true
		}
	}
	op, opOK := "", false
	switch raw, present := obj["op"]; {
	case !present:
		c.bad(child(ptr, "op"), RuleRequired)
	default:
		s, isStr := raw.(string)
		switch {
		case !isStr:
			c.bad(child(ptr, "op"), RuleType)
		case s != CondEq && s != CondIn:
			c.bad(child(ptr, "op"), RuleValue)
		default:
			op, opOK = s, true
		}
	}
	caller, callerOK := "", true
	if raw, present := obj["caller"]; present {
		s, isStr := raw.(string)
		switch {
		case !isStr:
			c.bad(child(ptr, "caller"), RuleType)
			callerOK = false
		case s != "" && s != CallerSelf && s != CallerMyGroups:
			c.bad(child(ptr, "caller"), RuleValue)
			callerOK = false
		default:
			caller = s
		}
	}
	var values []any
	valuesOK := true
	if raw, present := obj["values"]; present {
		arr, isArr := raw.([]any)
		if !isArr {
			c.bad(child(ptr, "values"), RuleType)
			valuesOK = false
		} else {
			values = arr
			for i, e := range arr {
				if _, isStr := e.(string); !isStr {
					c.bad(child(child(ptr, "values"), i), RuleType)
					valuesOK = false
				}
			}
		}
	}
	if !colOK || !opOK || !callerOK || !valuesOK {
		return
	}
	switch caller {
	case "":
		if op == CondEq && len(values) != 1 {
			c.bad(child(ptr, "values"), RuleCondition)
		}
		if column != metadata.ColumnState {
			for i, e := range values {
				s, _ := e.(string)
				id, err := uuid.Parse(s)
				if !uuidRE.MatchString(s) || err != nil {
					c.bad(child(child(ptr, "values"), i), RuleValue)
					continue
				}
				// A requester or assignee names a user, an assignment group a
				// group: both must be this tenant's.
				kind := KindUser
				if column == metadata.ColumnAssignmentGroupID {
					kind = KindGroup
				}
				c.refs = append(c.refs, TenantRef{child(child(ptr, "values"), i), kind, id})
			}
		}
	case CallerSelf:
		if len(values) > 0 || (column != metadata.ColumnRequesterID && column != metadata.ColumnAssigneeID) {
			c.bad(child(ptr, "caller"), RuleCondition)
		}
	case CallerMyGroups:
		if len(values) > 0 || column != metadata.ColumnAssignmentGroupID {
			c.bad(child(ptr, "caller"), RuleCondition)
		}
	}
}

func (c *policyCheck) mask(ptr string, v any) {
	obj, ok := c.object(ptr, v, "field", "read", "write")
	if !ok {
		return
	}
	switch raw, present := obj["field"]; {
	case !present:
		c.bad(child(ptr, "field"), RuleRequired)
	default:
		s, isStr := raw.(string)
		switch {
		case !isStr:
			c.bad(child(ptr, "field"), RuleType)
		case s == "":
			c.bad(child(ptr, "field"), RuleEmpty)
		}
	}
	subjects := map[string]map[string]bool{} // "read"/"write" -> the group and user ids it names
	for _, side := range []string{"read", "write"} {
		raw, present := obj[side]
		if !present {
			c.bad(child(ptr, side), RuleRequired)
			continue
		}
		sp := child(ptr, side)
		before := len(c.problems)
		c.subject(sp, raw)
		if len(c.problems) > before {
			continue
		}
		so, _ := raw.(map[string]any)
		// A2: a mask is row-independent, only Groups and Users match.
		for _, flag := range []string{"requester", "assignee", "shared"} {
			if b, _ := so[flag].(bool); b {
				c.bad(child(sp, flag), RuleMaskSubj)
			}
		}
		ids := map[string]bool{}
		for _, key := range []string{"groups", "users"} {
			arr, _ := so[key].([]any)
			for _, e := range arr {
				s, _ := e.(string)
				id, _ := uuid.Parse(s)
				ids[key+":"+id.String()] = true
			}
		}
		subjects[side] = ids
	}
	// R3: write-without-read lets a create or PATCH 409 answer "does another
	// record hold this value?" for a unique field. Write names nobody Read
	// does not.
	if wo, ro := subjects["write"], subjects["read"]; wo != nil && ro != nil {
		w, _ := obj["write"].(map[string]any)
		for _, key := range []string{"groups", "users"} {
			arr, _ := w[key].([]any)
			for i, e := range arr {
				s, _ := e.(string)
				id, _ := uuid.Parse(s)
				if !ro[key+":"+id.String()] {
					c.bad(child(child(child(ptr, "write"), key), i), RuleMaskWrite)
				}
			}
		}
	}
}

// CheckMaskFields returns a problem for every mask whose field is not a
// top-level property of the entity type's schema (the properties).
func CheckMaskFields(masks []Mask, properties []string) []Problem {
	var out []Problem
	for i, m := range masks {
		if !slices.Contains(properties, m.Field) {
			out = append(out, Problem{fmt.Sprintf("/masks/%d/field", i), RuleField})
		}
	}
	return out
}
