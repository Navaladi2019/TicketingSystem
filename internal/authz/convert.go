package authz

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// The format-1 converter (ADR-010 D12, PLAN-011 §Migration, TICK-936 P11-04).
//
// Stored acl_versions rows are never rewritten, so a format-1 rules array is
// converted on every read, including history read by the explainer and the
// audit. DO NOT DELETE this converter or its tests while any stored format-1
// row exists: the removal gate is the query in docs/delivery/plan-adr-010.md
// §Migration (D12),
//
//	SELECT count(*) FROM acl_versions WHERE jsonb_typeof(rules) = 'array';
//
// returning zero on every environment.

// ConvertFormat1 converts S10 rules to a format-2 document, in rule order.
// Each Rule becomes a policy with the rule's ops and
//
//	when = all[ any[subjects...], cmp(where[0]), cmp(where[1]), ... ]
//
// where the subjects, in this order and only those set, are one member_of for
// Groups and Users, cmp(requester_id eq caller self) for Requester,
// cmp(assignee_id eq caller self) for Assignee, and shared. An all with one
// child is kept as is: the planner folds it.
//
// A rule with no subject converts to an empty any, which Decode would refuse
// (child_count) and the planner voids; today that rule matches nothing too.
//
// The policy is named Rule.ID when that is a valid name. Otherwise the name is
// derived from it (format1Names) and Rule.ID is kept in LegacyID.
func ConvertFormat1(rules []Rule) Document {
	d := Document{Format: FormatV2, Conditions: []Condition{}, Policies: make([]NamedPolicy, 0, len(rules))}
	names := format1Names(rules)
	for i, r := range rules {
		p := NamedPolicy{Name: names[i], Ops: slices.Clone(r.Ops), When: format1When(r)}
		if p.Ops == nil {
			p.Ops = []string{}
		}
		if p.Name != r.ID {
			p.LegacyID = r.ID
		}
		d.Policies = append(d.Policies, p)
	}
	return d
}

func format1When(r Rule) Node {
	subjects := []Node{}
	if len(r.Who.Groups) > 0 || len(r.Who.Users) > 0 {
		subjects = append(subjects, Node{MemberOf: &MemberOf{Groups: slices.Clone(r.Who.Groups), Users: slices.Clone(r.Who.Users)}})
	}
	if r.Who.Requester {
		subjects = append(subjects, callerSelf(metadata.ColumnRequesterID))
	}
	if r.Who.Assignee {
		subjects = append(subjects, callerSelf(metadata.ColumnAssigneeID))
	}
	if r.Who.Shared {
		subjects = append(subjects, Node{Shared: true})
	}
	all := []Node{{Any: subjects}}
	for _, c := range r.Where {
		c.Values = slices.Clone(c.Values)
		all = append(all, Node{Cmp: &c})
	}
	return Node{All: all}
}

func callerSelf(column string) Node {
	return Node{Cmp: &Cond{Column: column, Op: CondEq, Caller: CallerSelf}}
}

// format1Names returns a valid, unique policy name for each rule, by position.
// A rule id that is already a valid name keeps it. Any other id is lowercased,
// every character outside [a-z0-9_] becomes "_", an "r_" prefix is added when
// it does not start with a letter, and it is cut to 63 characters; a name that
// is still taken (by a valid id anywhere in the list, or an earlier derived
// name) gets the smallest suffix "_2", "_3", ... that frees it. Stored rows
// are immutable, so the same row always gets the same names.
func format1Names(rules []Rule) []string {
	taken := map[string]bool{}
	for _, r := range rules {
		if ValidName(r.ID) && !taken[r.ID] {
			taken[r.ID] = true
		}
	}
	out := make([]string, len(rules))
	kept := map[string]bool{}
	for i, r := range rules {
		if ValidName(r.ID) && !kept[r.ID] {
			kept[r.ID] = true
			out[i] = r.ID
			continue
		}
		base := sanitizeName(r.ID)
		name := base
		for n := 2; taken[name]; n++ {
			sfx := "_" + strconv.Itoa(n)
			name = truncate(base, 63-len(sfx)) + sfx
		}
		taken[name] = true
		out[i] = name
	}
	return out
}

func sanitizeName(id string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(id) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	s := b.String()
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		s = "r_" + s
	}
	return truncate(s, 63)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
