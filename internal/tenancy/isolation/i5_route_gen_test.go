package isolation

// The I5 route generator (TICK-900): the world and the cases of the two I5
// rows that drive the real list route, GET /api/v1/records/{entity}, "I5 data
// filters" and "I5 list route", and of the masked-field oracle. Like
// i5_gen_test.go it draws everything from one seeded math/rand/v2 source and
// has no build tag, so task test checks the generator; the database half is
// i5_route_isolation_test.go.
//
// One entity type, iso_et_i5r, in tenant A, with three schema versions:
//
//	v1: prio string, tag string, kind integer, cost number (no defaults)
//	v2: v1 plus "default": "low" on prio (NeedsBackfill: old rows start
//	    reading it, TICK-360, and nothing backfills them)
//	v3: v2 with prio, tag, kind and cost flagged x-nava-filterable (ADR-006
//	    D6, D8), published after the records are written, so the real
//	    FieldIndexBackfill copies them to the side tables before any filter
//	    runs (S22 P7-12 removed the JSONB scan these rows used to drive)
//
// Records are written past the API as the superuser, under schema_version 1
// or 2, so each data field takes every shape a row written under an older
// version can have: present, JSON null, absent and defaulted on read (prio),
// absent with no default (tag, kind, cost), and a different JSON type (kind
// stored as a string in a v1 row: the publish route refuses a type change as
// Breaking, so such rows come only from a write that bypassed validation, an
// import or a pre-validation era, and the read path serves them as stored).

import (
	"encoding/json"
	"fmt"
	"maps"
	"math/rand/v2"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// The route run size. Every list is a real handler call, so the run is
// smaller than the SQL agreement row's 500 × 50, and every case still checks
// every record.
const (
	i5rKey        = "iso_et_i5r"
	i5rPrefix     = "IFR"
	i5rPolicies   = 24
	i5rPrincipals = 6
	i5rRecords    = 48
	i5rFilters    = 5 // filter requests per principal and policy
	i5rLists      = 1 // paged unfiltered lists per principal and policy
	// i5rDefaultPrio is v2's default of prio.
	i5rDefaultPrio = "low"
	// i5rFlaggedVersion is the version that flags i5rFlagged.
	i5rFlaggedVersion = 3
)

// i5rFlagged are the data fields v3 flags x-nava-filterable, sorted: every
// data field the route rows filter on (4 of the per-type quota of 5, D12).
func i5rFlagged() []string { return []string{"cost", "kind", "prio", "tag"} }

// The data fields' shapes (scope item 1 of TICK-900).
const (
	i5rPresent     = "present with a value"
	i5rNull        = "JSON null"
	i5rDefaulted   = "absent and defaulted on read"
	i5rNoDefault   = "absent with no default"
	i5rTypeChanged = "a different JSON type in the older version"
)

func i5rShapes() []string {
	return []string{i5rPresent, i5rNull, i5rDefaulted, i5rNoDefault, i5rTypeChanged}
}

// i5rExtraUsers and i5rExtraGroups are the tenant A principals and groups
// the route run inserts (and deletes) beside the seed's two of each.
func i5rExtraUsers() []uuid.UUID {
	return s10UUIDs("0a000000-0000-4000-8000-0000000095a1", "0a000000-0000-4000-8000-0000000095a2", "0a000000-0000-4000-8000-0000000095a3", "0a000000-0000-4000-8000-0000000095a4")
}

func i5rExtraGroups() []uuid.UUID {
	return s10UUIDs("0a000000-0000-4000-8000-0000000096a1", "0a000000-0000-4000-8000-0000000096a2")
}

// i5rSchema is version v's schema (1, 2 or 3). The definition maps state,
// group, assignee and requester to the fixed columns.
func i5rSchema(v int) string {
	flag := ""
	if v >= i5rFlaggedVersion {
		flag = `,"x-nava-filterable":true`
	}
	prio := `{"type":"string"` + flag + `}`
	if v >= 2 {
		prio = `{"type":"string","default":"` + i5rDefaultPrio + `"` + flag + `}`
	}
	return `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,` +
		`"required":["state"],"properties":{"state":{"type":"string","enum":["open","closed"],"default":"open"},` +
		`"prio":` + prio + `,"tag":{"type":"string"` + flag + `},"kind":{"type":"integer"` + flag + `},"cost":{"type":"number"` + flag + `},` +
		`"group":{"type":"string","format":"uuid"},"assignee":{"type":"string","format":"uuid"},"requester":{"type":"string","format":"uuid"}}}`
}

func i5rDefinition(v int) string {
	return fmt.Sprintf(`{"schema": %s,
		"fixed_columns": {"state": "state", "group": "assignment_group_id", "assignee": "assignee_id", "requester": "requester_id"},
		"states": {"initial": "open", "states": ["open", "closed"], "transitions": [{"from": "open", "to": "closed"}]}}`, i5rSchema(v))
}

// i5rRecord is one generated record: its fixed columns, its data as stored
// and the shape each data field takes.
type i5rRecord struct {
	ID            uuid.UUID
	SchemaVersion int
	State         string
	Requester     *uuid.UUID
	Assignee      *uuid.UUID
	Group         *uuid.UUID
	Data          map[string]any // as stored: absent keys are absent
	Shapes        map[string]string
	Shares        []i5Share
	// Age orders created_at and updated_at: now() minus Age seconds.
	Age int
}

// i5rWorld is the route run's world: the tenant A pool and the records.
type i5rWorld struct {
	acl     *i5World // the pool as the policy generator sees it
	Records []i5rRecord
}

// i5rDataFields are the data fields and the JSON Schema type of each.
func i5rDataFields() map[string]string {
	return map[string]string{"prio": "string", "tag": "string", "kind": "integer", "cost": "number"}
}

func newI5rWorld(r *rand.Rand) *i5rWorld {
	users := append(s10UUIDs(principalA1, principalA2), i5rExtraUsers()...)
	groups := append(s10UUIDs(adminGroupA, auditorGroupA), i5rExtraGroups()...)
	w := &i5rWorld{acl: &i5World{Tenant: uuid.MustParse(TenantA), Users: users, Groups: groups,
		DecoyPrincipal: groups[0], DecoyGroup: users[0]}}
	ages := r.Perm(i5rRecords * 4)
	for i := range i5rRecords {
		rec := i5rRecord{ID: i5UUID(r), SchemaVersion: 1 + r.IntN(2), State: i5Pick(r, []string{"open", "closed"}),
			Data: map[string]any{}, Shapes: map[string]string{}, Age: ages[i] + 1}
		pick := func(pct int, pool []uuid.UUID) *uuid.UUID {
			if i5Chance(r, pct) {
				return nil
			}
			id := i5Pick(r, pool)
			return &id
		}
		rec.Requester, rec.Assignee, rec.Group = pick(20, users), pick(35, users), pick(25, groups)
		for range r.IntN(3) {
			s := i5Share{Kind: "user", ID: i5Pick(r, users)}
			if i5Chance(r, 50) {
				s = i5Share{Kind: "group", ID: i5Pick(r, groups)}
			}
			if !slices.Contains(rec.Shares, s) {
				rec.Shares = append(rec.Shares, s)
			}
		}
		w.shape(r, &rec, "prio", []any{"low", "high"}, i5rDefaulted)
		w.shape(r, &rec, "tag", []any{"x", "y"}, i5rNoDefault)
		w.shape(r, &rec, "kind", []any{1, 2}, i5rNoDefault)
		w.shape(r, &rec, "cost", []any{1, 2, 3}, i5rNoDefault)
		w.Records = append(w.Records, rec)
	}
	return w
}

// shape gives field one shape: a value, null, absent (which reads as the
// default when absent is i5rDefaulted) or, for kind in a v1 row, the value as
// a JSON string.
func (w *i5rWorld) shape(r *rand.Rand, rec *i5rRecord, field string, values []any, absent string) {
	n := r.IntN(100)
	switch {
	case n < 45:
		rec.Data[field], rec.Shapes[field] = i5Pick(r, values), i5rPresent
	case n < 60:
		rec.Data[field], rec.Shapes[field] = nil, i5rNull
	case n < 80 || field != "kind" || rec.SchemaVersion != 1:
		rec.Shapes[field] = absent
	default:
		rec.Data[field], rec.Shapes[field] = strconv.Itoa(i5Pick(r, values).(int)), i5rTypeChanged
	}
}

// sharedWith is the record's pair-matched shares for p (A1), as SharesFor
// returns them.
func (rec i5rRecord) sharedWith(p identity.Principal) []uuid.UUID {
	return i5Record{Shares: rec.Shares}.sharedWith(p)
}

// row is the record as p's Allows sees it.
func (rec i5rRecord) row(p identity.Principal) authz.Row {
	return authz.Row{ID: rec.ID, State: rec.State, RequesterID: rec.Requester, AssigneeID: rec.Assignee,
		AssignmentGroupID: rec.Group, SharedWith: rec.sharedWith(p)}
}

// defaulted is the payload the read path should serve for the record: the
// stored data, v2's prio default when prio is absent, and the fixed columns.
// The route row checks the read path against it before it trusts GET as the
// oracle.
func (rec i5rRecord) defaulted() map[string]any {
	out := map[string]any{"state": rec.State}
	maps.Copy(out, rec.Data)
	if _, ok := out["prio"]; !ok {
		out["prio"] = i5rDefaultPrio
	}
	for prop, id := range map[string]*uuid.UUID{"requester": rec.Requester, "assignee": rec.Assignee, "group": rec.Group} {
		if id != nil {
			out[prop] = id.String()
		}
	}
	return out
}

// i5rFilter is one filter of a list request: filter[Prop]=v or
// filter[Prop][in]=v1,v2.
type i5rFilter struct {
	Prop   string
	In     bool
	Values []string
}

func (f i5rFilter) key() string {
	if f.In {
		return "filter[" + f.Prop + "][in]"
	}
	return "filter[" + f.Prop + "]"
}

// i5rQuery is one list request's filters.
type i5rQuery []i5rFilter

func (q i5rQuery) encode() string {
	v := url.Values{}
	for _, f := range q {
		v.Set(f.key(), strings.Join(f.Values, ","))
	}
	return v.Encode()
}

func (q i5rQuery) names() []string {
	out := make([]string, len(q))
	for i, f := range q {
		out[i] = f.Prop
	}
	return out
}

func (w *i5rWorld) filterValues(prop string) []string {
	ids := func(xs []uuid.UUID) []string {
		out := make([]string, len(xs))
		for i, x := range xs {
			out[i] = x.String()
		}
		return out
	}
	switch prop {
	case "prio":
		return []string{i5rDefaultPrio, "high", "none"}
	case "tag":
		return []string{"x", "y", "z"}
	case "kind":
		return []string{"1", "2", "3"}
	case "cost":
		return []string{"1", "2", "3", "4"}
	case "state":
		return []string{"open", "closed"}
	case "assignee":
		return ids(w.acl.Users)
	default: // group
		return ids(w.acl.Groups)
	}
}

// query draws one or two filters on distinct properties. Data fields come up
// most: they are what the row is about.
func (w *i5rWorld) query(r *rand.Rand) i5rQuery {
	props := []string{"prio", "prio", "tag", "kind", "kind", "cost", "state", "assignee", "group"}
	var q i5rQuery
	for range 1 + r.IntN(2) {
		prop := i5Pick(r, props)
		if slices.ContainsFunc(q, func(f i5rFilter) bool { return f.Prop == prop }) {
			continue
		}
		f := i5rFilter{Prop: prop, In: i5Chance(r, 40)}
		pool := w.filterValues(prop)
		k := 1
		if f.In {
			k = 1 + r.IntN(len(pool))
		}
		for _, i := range r.Perm(len(pool))[:k] {
			f.Values = append(f.Values, pool[i])
		}
		q = append(q, f)
	}
	return q
}

// i5rIndexed is the side-table oracle of one stored value (ADR-006 D5, D7):
// the value record_field_coerce copies for field, read by the field's value
// type (text for a string property, num for integer and number), and whether
// it copies one at all. An absent key, a JSON null and a value of another
// JSON type (kind stored as "1") copy nothing, so no filter matches them.
func i5rIndexed(field string, v any) (any, bool) {
	switch i5rDataFields()[field] {
	case "string":
		s, ok := v.(string)
		return s, ok
	case "integer", "number":
		switch n := v.(type) {
		case int:
			return float64(n), true
		case int64:
			return float64(n), true
		case float64:
			return n, true
		case json.Number:
			f, err := n.Float64()
			return f, err == nil
		}
	}
	return nil, false
}

// matches is the Go side of a filter, the side-table oracle: the record's
// value of the property as stored (filterView), not as the read path serves
// it, is indexed and equals one of the filter's values read by the field's
// value type. A key the read path only fills with v2's default has no side
// row. One exception, ADR-009 D12 (TICK-1192): v2 gives prio a default and its
// schema backfill never finishes in this run, so the window is open the whole
// time, and a filter that the default satisfies also matches a record whose
// stored payload has no prio key (a stored null is not absent: GET serves it
// as null, so it must not match).
func (q i5rQuery) matches(stored map[string]any) bool {
	for _, f := range q {
		v, present := stored[f.Prop]
		if f.Prop == "prio" && !present && slices.Contains(f.Values, i5rDefaultPrio) {
			continue // in the D12 window: the default satisfies this filter
		}
		var got any
		ok := false
		switch f.Prop {
		case "state", "assignee", "group":
			got, ok = v.(string) // a fixed column, as text
		default:
			if present {
				got, ok = i5rIndexed(f.Prop, v)
			}
		}
		if !ok {
			return false
		}
		hit := false
		for _, raw := range f.Values {
			var want any = raw
			if _, num := got.(float64); num {
				n, err := strconv.ParseFloat(raw, 64)
				if err != nil {
					return false
				}
				want = n
			}
			hit = hit || got == want
		}
		if !hit {
			return false
		}
	}
	return true
}

// filterView is what matches reads for the record: the data as stored and
// the fixed columns as text, without the read path's defaults.
func (rec i5rRecord) filterView() map[string]any {
	out := map[string]any{"state": rec.State}
	maps.Copy(out, rec.Data)
	for prop, id := range map[string]*uuid.UUID{"requester": rec.Requester, "assignee": rec.Assignee, "group": rec.Group} {
		if id != nil {
			out[prop] = id.String()
		}
	}
	return out
}

// i5rMask is the case's mask on cost: read and write by one group or one
// user of the pool, so the cases mix callers who may name cost and callers
// who may not.
func (w *i5rWorld) mask(r *rand.Rand) []authz.Mask {
	s := authz.Subject{Groups: []uuid.UUID{i5Pick(r, w.acl.Groups)}}
	if i5Chance(r, 40) {
		s = authz.Subject{Users: []uuid.UUID{i5Pick(r, w.acl.Users)}}
	}
	return []authz.Mask{{Field: "cost", Read: s, Write: s}}
}

// i5rCase is one policy and mask, the callers it is checked for, and each
// caller's filter requests and paged-list shapes.
type i5rCase struct {
	Rules      []authz.Rule
	Masks      []authz.Mask
	Principals []identity.Principal
	Queries    [][]i5rQuery // per principal
	Pages      [][]i5rPage  // per principal
}

// i5rPage is one paged unfiltered list: the sort and the page size.
type i5rPage struct {
	Sort  string
	Limit int
}

// principal draws a caller: a user of the pool in any subset of its groups.
func (w *i5rWorld) principal(r *rand.Rand) identity.Principal {
	p := identity.Principal{TenantID: w.acl.Tenant, PrincipalID: i5Pick(r, w.acl.Users)}
	for _, g := range w.acl.Groups {
		if i5Chance(r, 35) {
			p.Groups = append(p.Groups, g)
		}
	}
	return p
}

// i5rStorable reports whether acl_versions' shape CHECK
// (acl_rules_valid, migration 20261109100000) admits rule: the route can only
// ever read a stored policy, so the malformed rules the SQL agreement row
// draws are dropped here. That row covers them.
func i5rStorable(rule authz.Rule) bool {
	if rule.ID == "" || len(rule.Ops) == 0 || slices.ContainsFunc(rule.Ops, func(op string) bool { return op != authz.OpRead && op != authz.OpWrite }) {
		return false
	}
	for _, c := range rule.Where {
		if !slices.Contains([]string{"state", "requester_id", "assignee_id", "assignment_group_id"}, c.Column) ||
			(c.Op != authz.CondEq && c.Op != authz.CondIn) {
			return false
		}
		switch c.Caller {
		case "":
			if c.Op == authz.CondEq && len(c.Values) != 1 {
				return false
			}
			if c.Column != "state" && slices.ContainsFunc(c.Values, func(v string) bool { _, err := uuid.Parse(v); return err != nil || len(v) != 36 }) {
				return false
			}
		case authz.CallerSelf:
			if len(c.Values) > 0 || (c.Column != "requester_id" && c.Column != "assignee_id") {
				return false
			}
		case authz.CallerMyGroups:
			if len(c.Values) > 0 || c.Column != "assignment_group_id" {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// i5rGenerate is the whole route run for a seed. The policies come from the
// agreement row's rule generator over the pool, so every Subject kind and
// Cond form, malformed ones included, reaches the route too.
func i5rGenerate(seed uint64) (*i5rWorld, []i5rCase) {
	r := rand.New(rand.NewPCG(seed^0x5bd1e995, seed))
	w := newI5rWorld(r)
	sorts := []string{"", "-updated_at", "created_at", "state", "-state", "assignee", "-group", "requester"}
	cases := make([]i5rCase, i5rPolicies)
	for i := range cases {
		c := &cases[i]
		c.Rules = []authz.Rule{} // stored as [], never null
		if pol, rules := w.acl.policy(r); pol != nil {
			c.Rules = append(c.Rules, slices.DeleteFunc(rules, func(rule authz.Rule) bool { return !i5rStorable(rule) })...)
		}
		c.Masks = w.mask(r)
		for range i5rPrincipals {
			c.Principals = append(c.Principals, w.principal(r))
			var qs []i5rQuery
			for range i5rFilters {
				qs = append(qs, w.query(r))
			}
			c.Queries = append(c.Queries, qs)
			var ps []i5rPage
			for range i5rLists {
				ps = append(ps, i5rPage{Sort: i5Pick(r, sorts), Limit: 4 + r.IntN(12)})
			}
			c.Pages = append(c.Pages, ps)
		}
	}
	return w, cases
}

// i5rTwins is the masked-field oracle's pair: two records equal in every
// column and field but cost, which the case's mask hides from some callers.
func i5rTwins(w *i5rWorld) (a, b i5rRecord) {
	base := i5rRecord{SchemaVersion: 2, State: "open", Data: map[string]any{"prio": "high", "tag": "twin"},
		Shapes: map[string]string{}, Group: &w.acl.Groups[1], Requester: &w.acl.Users[1]}
	a, b = base, base
	a.ID, a.Data = uuid.MustParse("0a000000-0000-4000-8000-0000000097a1"), map[string]any{"prio": "high", "tag": "twin", "cost": 1}
	b.ID, b.Data = uuid.MustParse("0a000000-0000-4000-8000-0000000097a2"), map[string]any{"prio": "high", "tag": "twin", "cost": 900}
	a.Age, b.Age = 10_000, 10_000
	return a, b
}

// i5rMaskCase is the masked-field oracle's fixed case: a rule that lets every
// user of the pool read every record, cost readable by Users[0] only, and
// every user as a caller, so the oracle always has callers on both sides.
func i5rMaskCase(w *i5rWorld) i5rCase {
	only := authz.Subject{Users: []uuid.UUID{w.acl.Users[0]}}
	c := i5rCase{
		Rules: []authz.Rule{{ID: "all", Ops: []string{authz.OpRead}, Who: authz.Subject{Users: w.acl.Users}}},
		Masks: []authz.Mask{{Field: "cost", Read: only, Write: only}},
	}
	for _, u := range w.acl.Users {
		c.Principals = append(c.Principals, identity.Principal{TenantID: w.acl.Tenant, PrincipalID: u})
	}
	return c
}

// TestI5RouteGeneratorCoversTheRiskyShapes: the default seed's route run has
// every data-field shape of TICK-900 (and the rows the oracle needs to tell
// them apart): each shape on some field, a filter on each data field naming
// a value some record holds, eq and in, both schema versions, callers who
// may and may not name cost, and the defaulted read of a v1 row.
func TestI5RouteGeneratorCoversTheRiskyShapes(t *testing.T) {
	w, cases := i5rGenerate(i5DefaultSeed)
	has := map[string]bool{}
	for _, rec := range w.Records {
		has[fmt.Sprintf("schema_version %d", rec.SchemaVersion)] = true
		for field, shape := range rec.Shapes {
			has[shape] = true
			has[field+": "+shape] = true
		}
		if rec.Shapes["prio"] == i5rDefaulted && rec.SchemaVersion == 1 {
			has["a v1 row that reads v2's default"] = true
		}
		for _, field := range i5rFlagged() {
			if _, ok := i5rIndexed(field, rec.Data[field]); ok {
				has["an indexed "+field] = true
			}
		}
	}
	for _, c := range cases {
		pol := &authz.Policy{Document: authz.ConvertFormat1(c.Rules), Masks: c.Masks}
		for k, p := range c.Principals {
			has[fmt.Sprintf("caller who may name cost: %v", authz.Readable(pol, p, "cost"))] = true
			for _, q := range c.Queries[k] {
				for _, f := range q {
					has[fmt.Sprintf("in: %v", f.In)] = true
					if slices.Contains(f.Values, i5rDefaultPrio) && f.Prop == "prio" {
						has["a filter on prio's default"] = true
					}
				}
			}
		}
	}
	want := append(i5rShapes(), "prio: "+i5rDefaulted, "kind: "+i5rTypeChanged, "tag: "+i5rNull, "tag: "+i5rNoDefault,
		"schema_version 1", "schema_version 2", "a v1 row that reads v2's default", "a filter on prio's default",
		"in: true", "in: false", "caller who may name cost: true", "caller who may name cost: false",
		"an indexed cost", "an indexed kind", "an indexed prio", "an indexed tag")
	for _, s := range want {
		if !has[s] {
			t.Errorf("the default seed's route run has no %s", s)
		}
	}
}

// TestI5RouteMatchesTheSideTable: the Go filter is the side-table oracle. A
// number matches by value (1 and 1.0 are one double), the string "1" does
// not match the integer filter 1, a null and a missing field match nothing,
// and a prio absent as stored matches a filter on the default only through
// the D12 window (a stored null does not), while a stored "low" always does.
func TestI5RouteMatchesTheSideTable(t *testing.T) {
	kind1 := i5rQuery{{Prop: "kind", Values: []string{"1"}}}
	prioLow := i5rQuery{{Prop: "prio", Values: []string{i5rDefaultPrio}}}
	for _, c := range []struct {
		view map[string]any
		q    i5rQuery
		want bool
	}{
		{map[string]any{"kind": 1}, kind1, true},
		{map[string]any{"kind": json.Number("1.0")}, kind1, true},
		{map[string]any{"kind": "1"}, kind1, false},
		{map[string]any{"kind": nil}, kind1, false},
		{map[string]any{}, kind1, false},
		// ADR-009 D12 window: a stored payload with no prio matches the default
		// (the run's v2 backfill never finishes); a stored null does not.
		{i5rRecord{State: "open", Data: map[string]any{}}.filterView(), prioLow, true},
		{i5rRecord{State: "open", Data: map[string]any{"prio": nil}}.filterView(), prioLow, false},
		{i5rRecord{State: "open", Data: map[string]any{}}.filterView(), i5rQuery{{Prop: "prio", Values: []string{"high"}}}, false},
		{i5rRecord{State: "open", Data: map[string]any{}}.defaulted(), prioLow, true},
		{i5rRecord{State: "open", Data: map[string]any{"prio": i5rDefaultPrio}}.filterView(), prioLow, true},
		{map[string]any{"tag": "x", "kind": 2}, i5rQuery{{Prop: "tag", In: true, Values: []string{"y", "x"}}, {Prop: "kind", Values: []string{"2"}}}, true},
		{map[string]any{"cost": 3}, i5rQuery{{Prop: "cost", In: true, Values: []string{"1", "2"}}}, false},
		{map[string]any{"tag": 1}, i5rQuery{{Prop: "tag", Values: []string{"1"}}}, false},
		{i5rRecord{State: "closed", Data: map[string]any{}}.filterView(), i5rQuery{{Prop: "state", In: true, Values: []string{"open", "closed"}}}, true},
	} {
		if got := c.q.matches(c.view); got != c.want {
			t.Errorf("%v matches %v = %v, want %v", c.q, c.view, got, c.want)
		}
	}
}

// TestI5RouteGeneratorReplaysItsSeed: one seed is one route run.
func TestI5RouteGeneratorReplaysItsSeed(t *testing.T) {
	w1, c1 := i5rGenerate(42)
	w2, c2 := i5rGenerate(42)
	j := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	if j(w1.Records) != j(w2.Records) || j(c1) != j(c2) {
		t.Fatal("seed 42 generated two different route runs")
	}
	if w3, _ := i5rGenerate(43); j(w1.Records) == j(w3.Records) {
		t.Fatal("seeds 42 and 43 generated the same records")
	}
}

// TestI5RouteVersionsPublish: v1, v2 and v3 compile; v2 is NeedsBackfill
// (prio's new default), never Breaking, so the route run's publish succeeds
// and old rows start reading the default (TICK-360); v3 flags exactly
// i5rFlagged, as text and num, and is not Breaking.
func TestI5RouteVersionsPublish(t *testing.T) {
	compile := func(v int) *metadata.Compiled {
		var d metadata.Definition
		if err := json.Unmarshal([]byte(i5rDefinition(v)), &d); err != nil {
			t.Fatal(err)
		}
		d.Key, d.NumberPrefix = i5rKey, i5rPrefix
		c, err := metadata.Compile(d)
		if err != nil {
			t.Fatalf("v%d: %v", v, err)
		}
		return c
	}
	v1, v2, v3 := compile(1), compile(2), compile(i5rFlaggedVersion)
	class, paths, err := metadata.Classify(v1, v2)
	if err != nil || class != metadata.NeedsBackfill {
		t.Fatalf("v1 -> v2 = %v %v %v, want NeedsBackfill", class, paths, err)
	}
	if class, paths, err := metadata.Classify(v2, v3); err != nil || class == metadata.Breaking {
		t.Fatalf("v2 -> v3 = %v %v %v, want not Breaking", class, paths, err)
	}
	if len(v2.Filterable()) != 0 {
		t.Fatalf("v2 flags %v, want none", v2.Filterable())
	}
	var flagged []string
	for _, f := range v3.Filterable() {
		want := fieldindex.TypeText
		if k := i5rDataFields()[f.Key]; k == "integer" || k == "number" {
			want = fieldindex.TypeNum
		}
		if f.ValueType != want {
			t.Errorf("v3 indexes %s as %s, want %s", f.Key, f.ValueType, want)
		}
		flagged = append(flagged, f.Key)
	}
	if !slices.Equal(flagged, i5rFlagged()) {
		t.Fatalf("v3 flags %v, want %v", flagged, i5rFlagged())
	}
	if got := v2.ApplyDefaults(map[string]any{})["prio"]; got != i5rDefaultPrio {
		t.Fatalf("v2 reads an absent prio as %v, want %q", got, i5rDefaultPrio)
	}
	if _, ok := v1.ApplyDefaults(map[string]any{})["prio"]; ok {
		t.Fatal("v1 has a prio default")
	}
}

// TestI5RouteQueryEncodes: a query is the route's filter[...] form.
func TestI5RouteQueryEncodes(t *testing.T) {
	q := i5rQuery{{Prop: "prio", Values: []string{"low"}}, {Prop: "kind", In: true, Values: []string{"1", "2"}}}
	v, err := url.ParseQuery(q.encode())
	if err != nil || v.Get("filter[prio]") != "low" || v.Get("filter[kind][in]") != "1,2" || len(v) != 2 {
		t.Fatalf("encode = %q (%v), want filter[prio]=low and filter[kind][in]=1,2", q.encode(), err)
	}
	if !slices.Equal(q.names(), []string{"prio", "kind"}) {
		t.Fatalf("names = %v", q.names())
	}
}

// TestI5RouteRowIsPairMatched: the route world's Row carries the caller's
// pair-matched shares only (A1), as the SQL agreement row's does.
func TestI5RouteRowIsPairMatched(t *testing.T) {
	u, g := uuid.MustParse("0e000000-0000-4000-8000-0000000000e1"), uuid.MustParse("0e000000-0000-4000-8000-0000000000e2")
	rec := i5rRecord{ID: u, State: "open", Shares: []i5Share{{"group", u}, {"user", u}, {"group", g}}}
	row := rec.row(identity.Principal{PrincipalID: u, Groups: []uuid.UUID{g}})
	if row.ID != u || row.State != "open" || !slices.Equal(row.SharedWith, []uuid.UUID{u, g}) {
		t.Fatalf("row = %+v, want the user share of u and the group share of g", row)
	}
	if got := rec.row(identity.Principal{PrincipalID: g}).SharedWith; len(got) != 0 {
		t.Fatalf("a caller no share names gets SharedWith %v", got)
	}
}

// TestI5RouteTwinsDifferOnlyInCost: the masked-field oracle's twins are equal
// in every column and field but cost, and the fixed mask case has callers on
// both sides of the mask, all of whom read every record.
func TestI5RouteTwinsDifferOnlyInCost(t *testing.T) {
	w, _ := i5rGenerate(i5DefaultSeed)
	a, b := i5rTwins(w)
	if a.ID == b.ID || a.Data["cost"] == b.Data["cost"] {
		t.Fatal("the twins share an id or a cost")
	}
	strip := func(r i5rRecord) string {
		r.ID, r.Data = uuid.Nil, mapsWithout(r.Data, "cost")
		b, _ := json.Marshal(r)
		return string(b)
	}
	if strip(a) != strip(b) {
		t.Fatalf("the twins differ beyond cost:\n%s\n%s", strip(a), strip(b))
	}
	c := i5rMaskCase(w)
	pol := &authz.Policy{Document: authz.ConvertFormat1(c.Rules), Masks: c.Masks}
	sides := map[bool]int{}
	for _, p := range c.Principals {
		if _, _, ok := authz.Allows(pol, p, authz.OpRead, a.row(p)); !ok {
			t.Fatalf("caller %s cannot read twin a under the mask case", p.PrincipalID)
		}
		sides[authz.Readable(pol, p, "cost")]++
	}
	if sides[true] == 0 || sides[false] == 0 {
		t.Fatalf("the mask case's callers: %v by whether they read cost, want both", sides)
	}
	if !i5rStorable(c.Rules[0]) {
		t.Fatal("the mask case's rule is not storable")
	}
}

func mapsWithout(m map[string]any, key string) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		if k != key {
			out[k] = v
		}
	}
	return out
}
