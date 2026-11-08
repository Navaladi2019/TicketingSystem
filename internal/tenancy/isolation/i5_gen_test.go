package isolation

// The I5 agreement generator (PLAN-003 P3-32, TICK-617): random worlds,
// policies and principals for the randomized check that the SQL list
// predicate (authz.CompileRead) and the Go single-record check
// (authz.Allows) never disagree. Everything is drawn from one seeded
// math/rand/v2 source, ids included, so a seed replays the same world and the
// same cases. The database half is i5_agreement_isolation_test.go; this file
// has no build tag, so task test checks the generator itself.

import (
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// The I5 run size (P3-32 scope: at least 500 policies × 50 principals) and
// the fixed CI seed. NAVA_I5_SEED replays another seed.
const (
	i5Policies      = 500
	i5Principals    = 50
	i5Records       = 160
	i5DefaultSeed   = uint64(20261109)
	i5SeedEnv       = "NAVA_I5_SEED"
	i5MutantEnv     = "NAVA_I5_MUTANT"
	i5LoaderCallers = 20
)

// i5Seed is the run's seed: NAVA_I5_SEED when set, otherwise the fixed CI
// seed.
func i5Seed() (uint64, error) {
	s := os.Getenv(i5SeedEnv)
	if s == "" {
		return i5DefaultSeed, nil
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w", i5SeedEnv, s, err)
	}
	return n, nil
}

// i5Share is one record_shares row: a user share names a principal, a group
// share a group (TICK-625).
type i5Share struct {
	Kind string // "user" or "group"
	ID   uuid.UUID
}

// i5Record is one generated record: its fixed columns (Row without
// SharedWith, which depends on the caller) and every share it has.
type i5Record struct {
	Row    authz.Row
	Shares []i5Share
}

// i5World is one tenant's principals, groups and records.
//
// Callers are drawn from Users and Groups only. Outsiders and OutsiderGroups
// exist in the tenant and are named by shares, rules and record columns, but
// never call: a compiler that matches a share without its pair, or matches
// any share at all, admits rows through them. DecoyPrincipal is a principal
// whose id is Groups[0]'s, DecoyGroup a group whose id is Users[0]'s, so a
// share that matched on the id alone would cross kinds (AR-S10-01).
type i5World struct {
	Tenant, Definition uuid.UUID
	Users, Outsiders   []uuid.UUID
	Groups             []uuid.UUID
	OutsiderGroups     []uuid.UUID
	DecoyPrincipal     uuid.UUID // a principal with Groups[0]'s id
	DecoyGroup         uuid.UUID // a group with Users[0]'s id
	Records            []i5Record
}

// i5States are the states records take; "pending" is never named by a
// policy literal, "Open" is a literal no record has.
func i5States() []string { return []string{"open", "closed", "pending"} }

func i5UUID(r *rand.Rand) uuid.UUID {
	var u uuid.UUID
	for i := range u {
		u[i] = byte(r.IntN(256))
	}
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // RFC 4122 variant
	return u
}

func i5UUIDs(r *rand.Rand, n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = i5UUID(r)
	}
	return out
}

func i5Pick[T any](r *rand.Rand, xs []T) T { return xs[r.IntN(len(xs))] }

func i5Chance(r *rand.Rand, percent int) bool { return r.IntN(100) < percent }

// allPrincipals is every principals row: the ids a requester, an assignee or
// a user share may name.
func (w *i5World) allPrincipals() []uuid.UUID {
	return append(append(slices.Clone(w.Users), w.Outsiders...), w.DecoyPrincipal)
}

// allGroups is every groups row: the ids an assignment group or a group
// share may name.
func (w *i5World) allGroups() []uuid.UUID {
	return append(append(slices.Clone(w.Groups), w.OutsiderGroups...), w.DecoyGroup)
}

func newI5World(r *rand.Rand) *i5World {
	w := &i5World{Tenant: i5UUID(r), Definition: i5UUID(r)}
	w.Users, w.Outsiders = i5UUIDs(r, 6), i5UUIDs(r, 3)
	w.Groups, w.OutsiderGroups = i5UUIDs(r, 5), i5UUIDs(r, 2)
	w.DecoyPrincipal, w.DecoyGroup = w.Groups[0], w.Users[0]
	principals, groups := w.allPrincipals(), w.allGroups()
	states := i5States()
	for range i5Records {
		rec := i5Record{Row: authz.Row{ID: i5UUID(r), State: i5Pick(r, states)}}
		if !i5Chance(r, 25) {
			id := i5Pick(r, principals)
			rec.Row.RequesterID = &id
		}
		if !i5Chance(r, 40) {
			id := i5Pick(r, principals)
			rec.Row.AssigneeID = &id
		}
		if !i5Chance(r, 30) {
			id := i5Pick(r, groups)
			rec.Row.AssignmentGroupID = &id
		}
		for range r.IntN(4) {
			s := i5Share{Kind: "user", ID: i5Pick(r, principals)}
			if i5Chance(r, 50) {
				s = i5Share{Kind: "group", ID: i5Pick(r, groups)}
			}
			if !slices.Contains(rec.Shares, s) { // the primary key
				rec.Shares = append(rec.Shares, s)
			}
		}
		w.Records = append(w.Records, rec)
	}
	return w
}

// principal draws a caller of the tenant: usually one of Users, sometimes the
// decoy principal (whose id is a group's), sometimes no id at all (a system
// writer); in any subset of Groups, sometimes the decoy group too.
func (w *i5World) principal(r *rand.Rand) identity.Principal {
	p := identity.Principal{TenantID: w.Tenant}
	switch n := r.IntN(100); {
	case n < 85:
		p.PrincipalID = i5Pick(r, w.Users)
	case n < 92:
		p.PrincipalID = w.DecoyPrincipal
	}
	for _, g := range w.Groups {
		if i5Chance(r, 30) {
			p.Groups = append(p.Groups, g)
		}
	}
	if i5Chance(r, 10) {
		p.Groups = append(p.Groups, w.DecoyGroup)
	}
	return p
}

// sharedWith is Row.SharedWith as A1 defines it (AR-S10-01): the ids of the
// record's shares whose (kind, id) pair names p, a user share for p's id or
// a group share for one of p's groups, in principal-id order as SharesFor
// returns them. A caller with no id (a system writer) has no shares, as
// SharesFor's contract says.
func (rec i5Record) sharedWith(p identity.Principal) []uuid.UUID {
	if p.PrincipalID == uuid.Nil {
		return nil
	}
	var out []uuid.UUID
	for _, s := range rec.Shares {
		if (s.Kind == "user" && s.ID == p.PrincipalID) ||
			(s.Kind == "group" && slices.Contains(p.Groups, s.ID)) {
			out = append(out, s.ID)
		}
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return out
}

// row is the record as p's Allows sees it.
func (rec i5Record) row(p identity.Principal) authz.Row {
	row := rec.Row
	row.SharedWith = rec.sharedWith(p)
	return row
}

// policy draws a random policy over every Subject kind and Cond form,
// valid and malformed alike (a malformed rule must match nothing on both
// sides).
func (w *i5World) policy(r *rand.Rand) *authz.Policy {
	switch n := r.IntN(100); {
	case n < 2:
		return nil
	case n < 4:
		return &authz.Policy{Version: 1}
	}
	pol := &authz.Policy{EntityDefinitionID: w.Definition, Version: 1 + r.IntN(3)}
	for i := range 1 + r.IntN(4) {
		pol.Rules = append(pol.Rules, w.rule(r, i))
	}
	return pol
}

func (w *i5World) rule(r *rand.Rand, i int) authz.Rule {
	opSets := [][]string{{authz.OpRead}, {authz.OpRead}, {authz.OpWrite}, {authz.OpRead, authz.OpWrite}, {authz.OpWrite, authz.OpRead}, {}, {"READ"}}
	rule := authz.Rule{ID: "r" + strconv.Itoa(i), Ops: i5Pick(r, opSets)}
	groups := append(w.allGroups(), i5UUID(r)) // and a group the tenant does not have
	users := append(w.allPrincipals(), uuid.Nil)
	if i5Chance(r, 20) {
		for range 1 + r.IntN(2) {
			rule.Who.Groups = append(rule.Who.Groups, i5Pick(r, groups))
		}
	}
	if i5Chance(r, 15) {
		for range 1 + r.IntN(2) {
			rule.Who.Users = append(rule.Who.Users, i5Pick(r, users))
		}
	}
	rule.Who.Requester = i5Chance(r, 35)
	rule.Who.Assignee = i5Chance(r, 35)
	rule.Who.Shared = i5Chance(r, 40)
	for range r.IntN(4) {
		rule.Where = append(rule.Where, w.cond(r))
	}
	return rule
}

func (w *i5World) cond(r *rand.Rand) authz.Cond {
	op := authz.CondIn
	if i5Chance(r, 40) {
		op = authz.CondEq
	}
	uuidCols := []string{"requester_id", "assignee_id", "assignment_group_id"}
	switch n := r.IntN(100); {
	case n < 25: // a state literal; eq has one value, unless it is malformed
		c := authz.Cond{Column: "state", Op: op}
		k := r.IntN(4)
		if op == authz.CondEq {
			k = 1
			if i5Chance(r, 5) {
				k = 2
			}
		}
		for range k {
			c.Values = append(c.Values, i5Pick(r, []string{"open", "closed", "Open"}))
		}
		return c
	case n < 50: // a uuid literal
		c := authz.Cond{Column: i5Pick(r, uuidCols), Op: op}
		pool := w.allPrincipals()
		if c.Column == "assignment_group_id" {
			pool = w.allGroups()
		}
		k := 1 + r.IntN(3)
		if op == authz.CondEq {
			k = 1
		}
		for range k {
			switch m := r.IntN(100); {
			case m < 5:
				c.Values = append(c.Values, "not-a-uuid")
			case m < 15:
				c.Values = append(c.Values, strings.ToUpper(i5Pick(r, pool).String()))
			default:
				c.Values = append(c.Values, i5Pick(r, pool).String())
			}
		}
		return c
	case n < 70: // self: requester_id or assignee_id is the caller
		c := authz.Cond{Column: i5Pick(r, uuidCols[:2]), Op: op, Caller: authz.CallerSelf}
		if i5Chance(r, 10) {
			c.Column = i5Pick(r, []string{"assignment_group_id", "state"})
		}
		if i5Chance(r, 5) {
			c.Values = []string{w.Users[0].String()}
		}
		return c
	case n < 92: // my_groups: assignment_group_id is one of the caller's groups
		c := authz.Cond{Column: "assignment_group_id", Op: op, Caller: authz.CallerMyGroups}
		if i5Chance(r, 10) {
			c.Column = i5Pick(r, []string{"requester_id", "state"})
		}
		if i5Chance(r, 5) {
			c.Values = []string{w.Groups[0].String()}
		}
		return c
	default: // malformed: a column or op the cut does not have
		return i5Pick(r, []authz.Cond{
			{Column: "data", Op: authz.CondEq, Values: []string{"x"}},
			{Column: "title", Op: authz.CondIn, Values: []string{"t"}},
			{Column: "state", Op: "ne", Values: []string{"open"}},
			{Column: "assignee_id", Op: authz.CondEq, Caller: "everyone"},
		})
	}
}

// i5Case is one policy and the principals it is checked for.
type i5Case struct {
	Policy     *authz.Policy
	Principals []identity.Principal
}

// i5Generate is the whole run for a seed: the world, then each policy with
// its principals.
func i5Generate(seed uint64) (*i5World, []i5Case) {
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	w := newI5World(r)
	cases := make([]i5Case, i5Policies)
	for i := range cases {
		cases[i].Policy = w.policy(r)
		for range i5Principals {
			cases[i].Principals = append(cases[i].Principals, w.principal(r))
		}
	}
	return w, cases
}

// i5Compiler is CompileRead's signature: the real compiler or a mutant.
type i5Compiler func(pol *authz.Policy, p identity.Principal, alias string, argBase int) (string, []any)

// i5Mutants are deliberately broken compilers (P3-32: the suite must fail on
// them). Each breaks the SQL side only, so the Go check is the oracle.
func i5Mutants() map[string]i5Compiler {
	return map[string]i5Compiler{
		// drop_my_groups: a my_groups condition is ignored, so a desk rule
		// admits every group's records.
		"drop_my_groups": func(pol *authz.Policy, p identity.Principal, alias string, argBase int) (string, []any) {
			return authz.CompileRead(i5MapRules(pol, func(r authz.Rule) authz.Rule {
				r.Where = slices.DeleteFunc(slices.Clone(r.Where), func(c authz.Cond) bool { return c.Caller == authz.CallerMyGroups })
				return r
			}), p, alias, argBase)
		},
		// share_id_only: a share matches on its id alone, not the
		// (principal_kind, principal_id) pair (AR-S10-01).
		"share_id_only": func(pol *authz.Policy, p identity.Principal, alias string, argBase int) (string, []any) {
			sql, args := authz.CompileRead(pol, p, alias, argBase)
			sql = strings.ReplaceAll(sql, "acl_share.principal_kind = 'user' AND ", "")
			return strings.ReplaceAll(sql, "acl_share.principal_kind = 'group' AND ", ""), args
		},
		// ignore_ops: a write-only rule grants read.
		"ignore_ops": func(pol *authz.Policy, p identity.Principal, alias string, argBase int) (string, []any) {
			return authz.CompileRead(i5MapRules(pol, func(r authz.Rule) authz.Rule {
				r.Ops = []string{authz.OpRead}
				return r
			}), p, alias, argBase)
		},
	}
}

func i5MapRules(pol *authz.Policy, f func(authz.Rule) authz.Rule) *authz.Policy {
	if pol == nil {
		return nil
	}
	out := *pol
	out.Rules = make([]authz.Rule, len(pol.Rules))
	for i, r := range pol.Rules {
		out.Rules[i] = f(r)
	}
	return &out
}

// i5MutantNames is i5Mutants' keys in order, so the report is stable.
func i5MutantNames() []string {
	names := make([]string, 0, len(i5Mutants()))
	for n := range i5Mutants() {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// TestI5GeneratorReplaysItsSeed: one seed is one world and one list of
// cases, so a logged seed replays the failing case; another seed is another
// run.
func TestI5GeneratorReplaysItsSeed(t *testing.T) {
	w1, c1 := i5Generate(42)
	w2, c2 := i5Generate(42)
	if !reflect.DeepEqual(w1, w2) || !reflect.DeepEqual(c1, c2) {
		t.Fatal("seed 42 generated two different runs")
	}
	w3, _ := i5Generate(43)
	if reflect.DeepEqual(w1.Records, w3.Records) {
		t.Fatal("seeds 42 and 43 generated the same records")
	}
	if len(c1) < 500 || len(c1[0].Principals) < 50 {
		t.Fatalf("run size %d policies × %d principals, want at least 500 × 50", len(c1), len(c1[0].Principals))
	}
}

// TestI5GeneratorCoversTheRiskyShapes: the default seed's run holds every
// shape the agreement must hold for, so the database check cannot pass for
// want of one: nil columns, shares of outsiders and of decoys across kinds,
// every Subject kind, self, my_groups, write-only and malformed rules, and
// callers with no id or no group.
func TestI5GeneratorCoversTheRiskyShapes(t *testing.T) {
	w, cases := i5Generate(i5DefaultSeed)
	has := map[string]bool{}
	outsider := append(slices.Clone(w.Outsiders), w.OutsiderGroups...)
	for _, rec := range w.Records {
		has["nil requester"] = has["nil requester"] || rec.Row.RequesterID == nil
		has["nil assignee"] = has["nil assignee"] || rec.Row.AssigneeID == nil
		has["nil group"] = has["nil group"] || rec.Row.AssignmentGroupID == nil
		for _, s := range rec.Shares {
			has["share of an outsider"] = has["share of an outsider"] || slices.Contains(outsider, s.ID)
			has["user share of the decoy principal (a group's id)"] = has["user share of the decoy principal (a group's id)"] || (s.Kind == "user" && s.ID == w.DecoyPrincipal)
			has["group share of the decoy group (a user's id)"] = has["group share of the decoy group (a user's id)"] || (s.Kind == "group" && s.ID == w.DecoyGroup)
		}
	}
	for _, c := range cases {
		for _, p := range c.Principals {
			has["caller with no id"] = has["caller with no id"] || p.PrincipalID == uuid.Nil
			has["caller in no group"] = has["caller in no group"] || len(p.Groups) == 0
		}
		if c.Policy == nil {
			has["nil policy"] = true
			continue
		}
		for _, r := range c.Policy.Rules {
			has["groups subject"] = has["groups subject"] || len(r.Who.Groups) > 0
			has["users subject"] = has["users subject"] || len(r.Who.Users) > 0
			has["requester subject"] = has["requester subject"] || r.Who.Requester
			has["assignee subject"] = has["assignee subject"] || r.Who.Assignee
			has["shared subject"] = has["shared subject"] || r.Who.Shared
			has["write-only rule"] = has["write-only rule"] || slices.Equal(r.Ops, []string{authz.OpWrite})
			for _, c := range r.Where {
				has["self"] = has["self"] || c.Caller == authz.CallerSelf
				has["my_groups"] = has["my_groups"] || c.Caller == authz.CallerMyGroups
				has["malformed cond"] = has["malformed cond"] || c.Column == "data" || c.Op == "ne" || slices.Contains(c.Values, "not-a-uuid")
			}
		}
	}
	for _, want := range []string{"nil requester", "nil assignee", "nil group", "share of an outsider",
		"user share of the decoy principal (a group's id)", "group share of the decoy group (a user's id)",
		"caller with no id", "caller in no group", "nil policy", "groups subject", "users subject", "requester subject",
		"assignee subject", "shared subject", "write-only rule", "self", "my_groups", "malformed cond"} {
		if !has[want] {
			t.Errorf("the default seed's run has no %s", want)
		}
	}
}

// TestI5SharedWithIsPairMatched: Row.SharedWith holds a share only when its
// kind and id name the caller (A1). The decoy shares carry a caller's id
// under the other kind and must stay out.
func TestI5SharedWithIsPairMatched(t *testing.T) {
	u, g := uuid.MustParse("0e000000-0000-4000-8000-0000000000e1"), uuid.MustParse("0e000000-0000-4000-8000-0000000000e2")
	rec := i5Record{Shares: []i5Share{{"user", g}, {"group", u}, {"user", u}, {"group", g}}}
	p := identity.Principal{PrincipalID: u, Groups: []uuid.UUID{g}}
	got := rec.row(p).SharedWith
	if want := []uuid.UUID{u, g}; !slices.Equal(got, want) {
		t.Fatalf("SharedWith = %v, want only the user share of u and the group share of g", got)
	}
	if got := rec.sharedWith(identity.Principal{PrincipalID: g, Groups: []uuid.UUID{u}}); !slices.Equal(got, []uuid.UUID{g, u}) && !slices.Equal(got, []uuid.UUID{u, g}) {
		t.Fatalf("SharedWith for caller g in group u = %v, want the user share of g and the group share of u", got)
	}
	if got := rec.sharedWith(identity.Principal{Groups: []uuid.UUID{u, g}}); len(got) != 0 {
		t.Fatalf("SharedWith for a caller with no id = %v, want none (SharesFor's contract)", got)
	}
}

// TestI5SeedFromEnv: NAVA_I5_SEED replays a seed; unset is the CI seed; a
// malformed one is an error, never a silent default.
func TestI5SeedFromEnv(t *testing.T) {
	t.Setenv(i5SeedEnv, "")
	if s, err := i5Seed(); err != nil || s != i5DefaultSeed {
		t.Fatalf("unset: %d, %v; want %d", s, err, i5DefaultSeed)
	}
	t.Setenv(i5SeedEnv, "7")
	if s, err := i5Seed(); err != nil || s != 7 {
		t.Fatalf("7: %d, %v", s, err)
	}
	t.Setenv(i5SeedEnv, "x")
	if _, err := i5Seed(); err == nil {
		t.Fatal("a malformed seed was accepted")
	}
}

// TestI5MutantsChangeTheSQL: each mutant compiles a different predicate from
// the real compiler for some case of the default run, so a mutant row in the
// report can never pass because the mutant was a no-op. The database run
// (TestI5Agreement) then shows the difference changes which rows match.
func TestI5MutantsChangeTheSQL(t *testing.T) {
	_, cases := i5Generate(i5DefaultSeed)
	for _, name := range i5MutantNames() {
		mutant, changed := i5Mutants()[name], false
		for _, c := range cases {
			for _, p := range c.Principals {
				want, wantArgs := authz.CompileRead(c.Policy, p, "r", 2)
				got, gotArgs := mutant(c.Policy, p, "r", 2)
				if got != want || !reflect.DeepEqual(gotArgs, wantArgs) {
					changed = true
					break
				}
			}
			if changed {
				break
			}
		}
		if !changed {
			t.Errorf("mutant %s compiles every case of the default run as the real compiler does", name)
		}
	}
}
