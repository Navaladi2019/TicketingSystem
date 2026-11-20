package groupsync

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-948: the pure parts of the org-membership write.

func unitIDs() map[string]uuid.UUID {
	return map[string]uuid.UUID{
		"a": uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"),
		"b": uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002"),
		"c": uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000003"),
	}
}

func userUUIDs() map[string]uuid.UUID {
	return map[string]uuid.UUID{
		"u1": uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000001"),
		"u2": uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002"),
	}
}

const testTenant = "11111111-1111-1111-1111-111111111111"

func ou(kc string, members ...string) identity.DirectoryGroup {
	return identity.DirectoryGroup{KeycloakGroupID: kc, Path: "/t/" + testTenant + "/ou/" + kc, Name: kc, MemberKeycloakUserIDs: members}
}

func usr(kc, primaryKC string) identity.DirectoryUser {
	u := identity.DirectoryUser{KeycloakUserID: kc, Enabled: true}
	if primaryKC != "" {
		u.PrimaryOrgUnitPath = "/t/" + testTenant + "/ou/" + primaryKC
	}
	return u
}

// stmts renders a batch as "KIND unit user [primary]" lines, sorted.
func stmts(b *pgx.Batch) []string {
	byID := map[string]string{}
	for k, v := range unitIDs() {
		byID[v.String()] = k
	}
	for k, v := range userUUIDs() {
		byID[v.String()] = k
	}
	var out []string
	for _, q := range b.QueuedQueries {
		kind := strings.Fields(q.SQL)[0]
		switch {
		case strings.Contains(q.SQL, "is_primary = false"):
			kind = "CLEAR"
		case strings.Contains(q.SQL, "is_primary = true"):
			kind = "SET"
		}
		line := fmt.Sprintf("%s %s %s", kind, byID[q.Arguments[1].(string)], byID[q.Arguments[2].(string)])
		if kind == "INSERT" {
			line += fmt.Sprintf(" %v", q.Arguments[3])
		}
		out = append(out, line)
	}
	slices.Sort(out)
	return out
}

func TestBuildOrgMemberPlan(t *testing.T) {
	type have = map[pair]bool
	row := func(unit, user string) pair { return pair{unitIDs()[unit], userUUIDs()[user]} }
	tests := []struct {
		name   string
		have   have
		units  []identity.DirectoryGroup
		users  []identity.DirectoryUser
		before []string
		after  []string
	}{
		{
			name:  "one unit and no attribute: that unit is primary",
			units: []identity.DirectoryGroup{ou("a", "u1")},
			users: []identity.DirectoryUser{usr("u1", "")},
			after: []string{"INSERT a u1 true"},
		},
		{
			name:  "two units and no attribute: no primary",
			units: []identity.DirectoryGroup{ou("a", "u1"), ou("b", "u1")},
			users: []identity.DirectoryUser{usr("u1", "")},
			after: []string{"INSERT a u1 false", "INSERT b u1 false"},
		},
		{
			name:  "the attribute names one of two units",
			units: []identity.DirectoryGroup{ou("a", "u1"), ou("b", "u1")},
			users: []identity.DirectoryUser{usr("u1", "b")},
			after: []string{"INSERT a u1 false", "INSERT b u1 true"},
		},
		{
			name:  "the attribute names the only unit",
			units: []identity.DirectoryGroup{ou("a", "u1")},
			users: []identity.DirectoryUser{usr("u1", "a")},
			after: []string{"INSERT a u1 true"},
		},
		{
			name:  "the attribute names a unit the user is not in: the only unit wins",
			units: []identity.DirectoryGroup{ou("a", "u1"), ou("b")},
			users: []identity.DirectoryUser{usr("u1", "b")},
			after: []string{"INSERT a u1 true"},
		},
		{
			name:  "the attribute names a unit the user is not in, user in two: none",
			units: []identity.DirectoryGroup{ou("a", "u1"), ou("b", "u1"), ou("c")},
			users: []identity.DirectoryUser{usr("u1", "c")},
			after: []string{"INSERT a u1 false", "INSERT b u1 false"},
		},
		{
			name:  "the attribute names a path that is no unit",
			units: []identity.DirectoryGroup{ou("a", "u1"), ou("b", "u1")},
			users: []identity.DirectoryUser{usr("u1", "zzz")},
			after: []string{"INSERT a u1 false", "INSERT b u1 false"},
		},
		{
			name:  "a user in no unit gets no row",
			units: []identity.DirectoryGroup{ou("a", "u2")},
			users: []identity.DirectoryUser{usr("u1", "a"), usr("u2", "")},
			after: []string{"INSERT a u2 true"},
		},
		{
			name:  "a member with no principal is skipped",
			units: []identity.DirectoryGroup{ou("a", "ghost", "u1")},
			users: []identity.DirectoryUser{usr("u1", "")},
			after: []string{"INSERT a u1 true"},
		},
		{
			name:  "a member listed twice is one row",
			units: []identity.DirectoryGroup{ou("a", "u1", "u1")},
			users: []identity.DirectoryUser{usr("u1", "")},
			after: []string{"INSERT a u1 true"},
		},
		{
			name:  "no change: nothing queued",
			have:  have{row("a", "u1"): true, row("b", "u1"): false},
			units: []identity.DirectoryGroup{ou("a", "u1"), ou("b", "u1")},
			users: []identity.DirectoryUser{usr("u1", "a")},
		},
		{
			name:   "primary moves: the old one is cleared before the new is set",
			have:   have{row("a", "u1"): true, row("b", "u1"): false},
			units:  []identity.DirectoryGroup{ou("a", "u1"), ou("b", "u1")},
			users:  []identity.DirectoryUser{usr("u1", "b")},
			before: []string{"CLEAR a u1"},
			after:  []string{"SET b u1"},
		},
		{
			name:   "primary dropped: cleared, nothing set",
			have:   have{row("a", "u1"): true, row("b", "u1"): false},
			units:  []identity.DirectoryGroup{ou("a", "u1"), ou("b", "u1")},
			users:  []identity.DirectoryUser{usr("u1", "zzz")},
			before: []string{"CLEAR a u1"},
		},
		{
			name:   "c vanished from the snapshot and a lost u1: their rows are deleted",
			have:   have{row("c", "u1"): false, row("c", "u2"): true, row("a", "u1"): false},
			units:  []identity.DirectoryGroup{ou("a")},
			users:  []identity.DirectoryUser{usr("u1", ""), usr("u2", "")},
			before: []string{"DELETE a u1", "DELETE c u1", "DELETE c u2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idOf := map[string]uuid.UUID{}
			for _, u := range tc.units {
				idOf[u.KeycloakGroupID] = unitIDs()[u.KeycloakGroupID]
			}
			plan := buildOrgMemberPlan(testTenant, tc.have, tc.units, idOf, tc.users, userUUIDs())
			if got := stmts(plan.before); !slices.Equal(got, tc.before) {
				t.Errorf("before = %v, want %v", got, tc.before)
			}
			if got := stmts(plan.after); !slices.Equal(got, tc.after) {
				t.Errorf("after = %v, want %v", got, tc.after)
			}
		})
	}
}

func chain(n int) (units []identity.DirectoryGroup, leaf string) {
	parent := ""
	for i := range n {
		k := fmt.Sprintf("n%d", i)
		u := ou(k)
		u.ParentKeycloakGroupID = parent
		units = append(units, u)
		parent, leaf = k, k
	}
	return units, leaf
}

func TestOrgSubtreeMax(t *testing.T) {
	deep, deepLeaf := chain(50)
	wide := []identity.DirectoryGroup{ou("root")}
	for i := range 200 {
		c := ou(fmt.Sprintf("leaf%d", i))
		c.ParentKeycloakGroupID = "root"
		wide = append(wide, c)
	}
	cycle := []identity.DirectoryGroup{
		{KeycloakGroupID: "x", ParentKeycloakGroupID: "y"},
		{KeycloakGroupID: "y", ParentKeycloakGroupID: "x"},
	}
	tests := []struct {
		name     string
		units    []identity.DirectoryGroup
		members  []string // units that have a member
		maxDepth int
		want     int
	}{
		{"no units", nil, nil, 8, 0},
		{"no members anywhere", wide, nil, 8, 0},
		{"deep tree, member at the root: every unit", deep, []string{"n0"}, 64, 50},
		{"deep tree, member in the middle", deep, []string{"n40"}, 64, 10},
		{"deep tree, member at the leaf: itself only", deep, []string{deepLeaf}, 64, 1},
		{"wide tree, member at the root", wide, []string{"root"}, 8, 201},
		{"wide tree, member at a leaf", wide, []string{"leaf7"}, 8, 1},
		{"the largest of several units wins", wide, []string{"leaf1", "root", "leaf2"}, 8, 201},
		{"a cycle terminates at the depth bound", cycle, []string{"x"}, 4, 5},
		{"a member of an unlisted unit counts nothing", wide, []string{"ghost"}, 8, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unitsOf := map[string][]string{}
			for i, m := range tc.members {
				unitsOf[fmt.Sprintf("user%d", i)] = []string{m}
			}
			if got := orgSubtreeMax(tc.units, unitsOf, tc.maxDepth); got != tc.want {
				t.Errorf("orgSubtreeMax = %d, want %d", got, tc.want)
			}
		})
	}
}
