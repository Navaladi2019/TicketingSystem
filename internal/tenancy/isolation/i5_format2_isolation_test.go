//go:build integration

// I5 agreement for format-2 access documents (PLAN-011 P11-10, TICK-942;
// ADR-010 I5, D5, D6, D12). For the format-2 generator's random documents
// (i5_format2_gen_test.go), the fixed cap cases and every format-1 fixture,
// in the same world TestI5Agreement seeds and under RLS as nava_app:
//
//   - SELECT … WHERE <authz.CompileRead> returns exactly {r | authz.Allows};
//   - the UNION ALL of authz.ReadBranches returns the same rows, each once,
//     and when it has no branch ReadPredicate's fallback predicate does;
//   - authz.Allows equals the reference evaluator (format 2) or the S10 rules
//     (format 1), so a planner bug that moves both renderings is seen too.
//
// Each property is a named row of isolation-report.md under I5, and each of
// the five format-2 mutants (i5F2Mutants) is a row that passes only when the
// suite catches it. NAVA_I5_MUTANT=<name> runs the agreement row against
// that mutant instead (it must turn red).

package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

const (
	i5RowF2Agreement = "I5 format-2 agreement: CompileRead, ReadBranches (UNION ALL, each row once) and ReadPredicate's fallback equal {r | Allows(read, r)}, and Allows equals the reference evaluator, for random trees of depth ≤ 4 over every node kind, under included, where authz.ResolveUnder's answers (org_unit_members and the seeded org_unit_closure, live paths only) equal the reference tree's"
	i5RowF2Caps      = "I5 format-2 caps: depth 8 after expansion, 128 leaves in a policy (a shared condition counted at every reach), 64 policies with 128 leaves agree; one past a policy cap voids that policy only (SP-2 caps)"
	i5RowF1Equiv     = "I5 format-1 equivalence (D12): every format-1 fixture gives the same rows through the converted path (CompileRead, ReadBranches) as the S10 rules"
	i5RowF2Mutant    = "I5 format-2 mutant %s: the suite catches it"
)

// TestI5Format2Agreement runs the format-2 properties and adds their rows to
// the report.
func TestI5Format2Agreement(t *testing.T) {
	i5Quiet(t)
	seed, err := i5Seed()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("I5 format-2 seed %d (replay: %s=%d make isolation)", seed, i5SeedEnv, seed)
	w, cases := i5F2Generate(seed)
	compile, compiler := i5F2Compiler(i5F2Real), "authz.CompileRead"
	if m := os.Getenv(i5MutantEnv); m != "" {
		if mc, ok := i5F2Mutants()[m]; ok {
			compile, compiler = mc, "mutant "+m
		}
	}
	add := func(name, status, detail string) {
		suite.addPath(pathResult{Path: Path{Name: name, Case: "I5"}, Status: status, Detail: detail})
		if status == statusFail {
			t.Errorf("%s: %s", name, detail)
		}
	}
	err = i5InWorld(t, w, func(ctx context.Context, tx pgx.Tx) error {
		start := time.Now()
		st, ferr := i5F2AgreeDB(ctx, tx, w, cases, compile, true)
		if ferr == nil {
			ferr = i5F2Discriminates(st, len(cases))
		}
		took := time.Since(start).Round(time.Millisecond)
		switch {
		case ferr != nil:
			add(i5RowF2Agreement, statusFail, fmt.Sprintf("seed %d, %s: %v", seed, compiler, ferr))
		case took > i5Budget:
			add(i5RowF2Agreement, statusFail, fmt.Sprintf("seed %d: %s, over the %s budget", seed, took, i5Budget))
		default:
			add(i5RowF2Agreement, statusPass, fmt.Sprintf("seed %d, %s: %d documents × %d principals × %d records agree (%d checks; %d cases admit a proper subset; %d principal plans split into branches, %d fall back past the SP-2 threshold; %d under expansions match the reference); %s",
				seed, compiler, len(cases), i5Principals, len(w.Records), st.checks, st.proper, st.split, st.fallback, st.expanded, took))
		}

		if detail, cerr := i5F2Caps(ctx, tx, w, cases[0].Principals); cerr != nil {
			add(i5RowF2Caps, statusFail, fmt.Sprintf("seed %d: %v", seed, cerr))
		} else {
			add(i5RowF2Caps, statusPass, detail)
		}

		if detail, eerr := i5F1Equivalence(ctx, tx, w, cases[0].Principals); eerr != nil {
			add(i5RowF1Equiv, statusFail, fmt.Sprintf("seed %d: %v", seed, eerr))
		} else {
			add(i5RowF1Equiv, statusPass, detail)
		}

		for _, name := range i5F2MutantNames() {
			_, merr := i5F2AgreeDB(ctx, tx, w, cases, i5F2Mutants()[name], false)
			var d *i5F2Disagreement
			switch {
			case errors.As(merr, &d):
				add(fmt.Sprintf(i5RowF2Mutant, name), statusPass, "caught: "+d.Error())
			case merr != nil:
				add(fmt.Sprintf(i5RowF2Mutant, name), statusFail, "the run broke, which proves nothing: "+merr.Error())
			default:
				add(fmt.Sprintf(i5RowF2Mutant, name), statusFail, fmt.Sprintf("seed %d: the mutant agreed on every case; the suite cannot see it", seed))
			}
		}
		return nil
	})
	if err != nil {
		add(i5RowF2Agreement, statusFail, fmt.Sprintf("seed %d: the run broke: %v", seed, err))
	}
}

// i5F2Stats says how much the run discriminated and how it split.
type i5F2Stats struct {
	checks, proper, split, fallback, expanded int
}

// i5F2Disagreement is the first case where two of the sides differ.
type i5F2Disagreement struct {
	Doc, Principal int
	Record         uuid.UUID
	What           string
	Pol            *authz.Policy
	P              identity.Principal
	Row            authz.Row
	Predicate      string
}

func (d *i5F2Disagreement) Error() string {
	j := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	return fmt.Sprintf("document #%d, principal #%d, record %s: %s; document %s; principal %s; row %s; predicate %s",
		d.Doc, d.Principal, d.Record, d.What, j(d.Pol), j(d.P), j(d.Row), d.Predicate)
}

// i5F2AgreeDB checks every case: compile's rows against Allows, and, when
// full (the real compiler's run), also the ReadBranches rows against Allows
// and Allows against the reference. A mutant's run compares its rows with
// Allows only, and stops at the first disagreement.
func i5F2AgreeDB(ctx context.Context, tx pgx.Tx, w *i5World, cases []i5F2Case, compile i5F2Compiler, full bool) (i5F2Stats, error) {
	var st i5F2Stats
	if full {
		if err := i5F2WorldSetsMatch(ctx, tx, w, cases, &st); err != nil {
			return st, err
		}
	}
	for ci := range cases {
		c := &cases[ci]
		bySQL, preds, err := i5Select(ctx, tx, w, i5Case{Policy: c.Policy, Principals: c.Principals},
			func(_ *authz.Policy, p identity.Principal, alias string, argBase int) (string, []any) {
				return compile(c, p, alias, argBase)
			})
		if err != nil {
			return st, fmt.Errorf("document #%d: %w", ci, err)
		}
		var byBranch []map[uuid.UUID]int
		if full {
			if byBranch, err = i5F2BranchSelect(ctx, tx, w, c.bound, c.Principals, &st); err != nil {
				return st, fmt.Errorf("document #%d branches: %w", ci, err)
			}
			if err := i5F2ExpandMatches(ctx, tx, c, &st); err != nil {
				return st, fmt.Errorf("document #%d: %w", ci, err)
			}
		}
		for ki, p := range c.Principals {
			admitted := 0
			for _, rec := range w.Records {
				st.checks++
				row := rec.row(p)
				_, _, goOK := authz.Allows(c.bound(p), p, authz.OpRead, row)
				if goOK {
					admitted++
				}
				fail := func(what string) error {
					return &i5F2Disagreement{Doc: ci, Principal: ki, Record: rec.Row.ID, What: what, Pol: c.Policy, P: p, Row: row, Predicate: preds[ki]}
				}
				if sqlOK := bySQL[ki][rec.Row.ID]; sqlOK != goOK {
					return st, fail(fmt.Sprintf("CompileRead admits it %v, Allows %v", sqlOK, goOK))
				}
				if !full {
					continue
				}
				if n := byBranch[ki][rec.Row.ID]; n > 1 || (n == 1) != goOK {
					return st, fail(fmt.Sprintf("the branches return it %d times, Allows %v", n, goOK))
				}
				if ref := i5Reference(c.Policy, p, authz.OpRead, row); ref != goOK {
					return st, fail(fmt.Sprintf("Allows %v, the reference evaluator %v", goOK, ref))
				}
			}
			if admitted > 0 && admitted < len(w.Records) {
				st.proper++
			}
		}
	}
	return st, nil
}

// i5F2ExpandPrincipals is how many of each case's principals the per-case
// ResolveUnder check runs for. Each is a query; the closure itself is checked
// for every principal of the run by i5F2WorldSetsMatch.
const i5F2ExpandPrincipals = 10

// i5F2ExpandMatches checks authz.ResolveUnder end to end on each case: for
// the first i5F2ExpandPrincipals principals of every document holding an
// under leaf, the policy ResolveUnder binds (its probe plan, its query over
// org_unit_members and the seeded org_unit_closure as nava_app under RLS, for
// every record of the world, and the empty U of a leaf the probe folded away)
// compiles to exactly the predicate and arguments of the reference's binding
// (c.bound), and Allows gives the reference's answer on every record. A
// policy ResolveUnder voids that the reference does not changes both
// (TICK-950 review: a leaf folded away beside a deciding constant voided its
// policy). Principals whose plan keeps no under leaf are checked too:
// ResolveUnder runs no query for them, and its binding must still not void
// anything.
func i5F2ExpandMatches(ctx context.Context, tx pgx.Tx, c *i5F2Case, st *i5F2Stats) error {
	if c.Policy == nil || len(i5UnderLeaves(c.Policy.Document)) == 0 {
		return nil
	}
	for ki, p := range c.Principals[:min(len(c.Principals), i5F2ExpandPrincipals)] {
		resolved, err := authz.ResolveUnder(ctx, tx, c.Policy, p, i5WorldRows(c.World, p)...)
		if err != nil {
			return fmt.Errorf("principal #%d: ResolveUnder: %w", ki, err)
		}
		ref := c.bound(p)
		gotSQL, gotArgs := authz.CompileRead(resolved, p, "r", 2)
		wantSQL, wantArgs := authz.CompileRead(ref, p, "r", 2)
		if gotSQL != wantSQL || !reflect.DeepEqual(gotArgs, wantArgs) {
			return fmt.Errorf("principal #%d: ResolveUnder's binding differs from the reference's: %s %v, want %s %v", ki, gotSQL, gotArgs, wantSQL, wantArgs)
		}
		for _, rec := range c.World.Records {
			row := rec.row(p)
			for _, op := range []string{authz.OpRead, authz.OpWrite} {
				_, _, got := authz.Allows(resolved, p, op, row)
				if _, _, want := authz.Allows(ref, p, op, row); got != want {
					return fmt.Errorf("principal #%d, record %s, %s: Allows with ResolveUnder's closure = %v, the reference's = %v", ki, row.ID, op, got, want)
				}
			}
		}
		if authz.UnderNeeded(c.Policy, p) {
			st.expanded++
		}
	}
	return nil
}

// i5WorldRows is every record of w as p's check sees it.
func i5WorldRows(w *i5World, p identity.Principal) []authz.Row {
	if w == nil {
		return nil
	}
	rows := make([]authz.Row, len(w.Records))
	for i, rec := range w.Records {
		rows[i] = rec.row(p)
	}
	return rows
}

// i5F2WorldSetsMatch checks the closure directly, for every distinct
// principal of the run (and a caller with no id): a document with one read
// policy per leaf, under caller and under each unit of the world plus an id
// that is no unit, so no leaf folds away. ResolveUnder binds it once per
// principal, for every record of the world, and each leaf alone must then
// admit exactly the records the reference's tree puts under it (i5Subtree,
// i5CallerSet: live units, a tombstoned unit and everything below it out),
// with the reference's SQL and arguments.
func i5F2WorldSetsMatch(ctx context.Context, tx pgx.Tx, w *i5World, cases []i5F2Case, st *i5F2Stats) error {
	leaves := []authz.Node{{Under: &authz.Under{Caller: true}}}
	for _, u := range w.Units {
		leaves = append(leaves, authz.Node{Under: &authz.Under{Units: []uuid.UUID{u.ID}}})
	}
	leaves = append(leaves, authz.Node{Under: &authz.Under{Units: []uuid.UUID{uuid.MustParse("0f000000-0000-4000-8000-0000000000aa")}}})
	doc := authz.Document{Format: authz.FormatV2, Conditions: []authz.Condition{}}
	for i, l := range leaves {
		doc.Policies = append(doc.Policies, authz.NamedPolicy{Name: "s" + strconv.Itoa(i), Ops: []string{authz.OpRead}, When: l})
	}
	pol := &authz.Policy{EntityDefinitionID: w.Definition, Version: 1, Document: doc}
	seen := map[uuid.UUID]bool{}
	principals := []identity.Principal{{TenantID: w.Tenant}}
	for _, c := range cases {
		for _, p := range c.Principals {
			if !seen[p.PrincipalID] {
				seen[p.PrincipalID] = true
				principals = append(principals, identity.Principal{TenantID: w.Tenant, PrincipalID: p.PrincipalID})
			}
		}
	}
	for _, p := range principals {
		resolved, err := authz.ResolveUnder(ctx, tx, pol, p, i5WorldRows(w, p)...)
		if err != nil {
			return fmt.Errorf("principal %s: ResolveUnder: %w", p.PrincipalID, err)
		}
		gotSQL, gotArgs := authz.CompileRead(resolved, p, "r", 2)
		wantSQL, wantArgs := authz.CompileRead(i5Bind(pol, p, nil), p, "r", 2)
		if gotSQL != wantSQL || !reflect.DeepEqual(gotArgs, wantArgs) {
			return fmt.Errorf("principal %s: U differs from the reference's: %s %v, want %s %v", p.PrincipalID, gotSQL, gotArgs, wantSQL, wantArgs)
		}
		for i, l := range leaves {
			one := *resolved // keeps ResolveUnder's binding
			one.Document = authz.Document{Format: authz.FormatV2, Conditions: []authz.Condition{}, Policies: []authz.NamedPolicy{doc.Policies[i]}}
			var set []uuid.UUID
			if l.Under.Caller {
				set = i5CallerSet(p)
			} else {
				set = i5Subtree(w.Tenant, l.Under.Units[0])
			}
			for _, rec := range w.Records {
				row := rec.row(p)
				want := row.OrgUnitID != nil && slices.Contains(set, *row.OrgUnitID)
				if _, _, got := authz.Allows(&one, p, authz.OpRead, row); got != want {
					return fmt.Errorf("principal %s, leaf %s, record %s: under = %v from the closure, %v from the reference", p.PrincipalID, doc.Policies[i].Name, row.ID, got, want)
				}
			}
		}
		st.expanded++
	}
	return nil
}

// i5F2Discriminates is the random run's guard: agreement on "nothing" or
// "everything" alone proves little, and a run with no split proves nothing
// about the branches.
func i5F2Discriminates(st i5F2Stats, cases int) error {
	if total := cases * i5Principals; st.proper < total/5 {
		return fmt.Errorf("only %d of %d cases admit a proper subset of the records", st.proper, total)
	}
	if st.expanded == 0 {
		return errors.New("no principal's plan kept an under leaf; the ResolveUnder check proves nothing")
	}
	if st.split == 0 {
		return errors.New("no case split into branches; the branch half proves nothing")
	}
	return nil
}

// i5F2BranchSelect runs, for one policy, one query with every principal's
// list predicate as the list route builds it: the ReadBranches branches as
// UNION ALL arms when there are any (each must also be ReadPredicate's when
// there are two or more), else ReadPredicate's single predicate (false, one
// branch's statement, or the fallback past the SP-2 threshold). It returns,
// per principal, how many times each record came back.
func i5F2BranchSelect(ctx context.Context, tx pgx.Tx, w *i5World, bound func(identity.Principal) *authz.Policy, principals []identity.Principal, st *i5F2Stats) ([]map[uuid.UUID]int, error) {
	args := []any{w.Tenant, w.Definition}
	var arms []string
	for k, p := range principals {
		pol := bound(p)
		branches, bargs := authz.ReadBranches(pol, p, "r", len(args))
		pb, where, pargs := authz.ReadPredicate(pol, p, "r", len(args))
		switch {
		case len(branches) >= 2:
			if !slices.Equal(pb, branches) {
				return nil, fmt.Errorf("principal #%d: ReadPredicate's branches differ from ReadBranches'", k)
			}
			st.split++
		case len(branches) == 0 && where != "false":
			st.fallback++
		}
		if len(branches) == 0 {
			branches, bargs = []string{where}, pargs
		}
		args = append(args, bargs...)
		for _, b := range branches {
			arms = append(arms, fmt.Sprintf("SELECT %d AS k, r.id FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND %s", k, b))
		}
	}
	out := make([]map[uuid.UUID]int, len(principals))
	for k := range out {
		out[k] = map[uuid.UUID]int{}
	}
	rows, err := tx.Query(ctx, strings.Join(arms, "\nUNION ALL\n"), args...)
	if err != nil {
		return nil, err
	}
	var (
		k  int
		id uuid.UUID
	)
	_, err = pgx.ForEachRow(rows, []any{&k, &id}, func() error { out[k][id]++; return nil })
	return out, err
}

// i5F2Caps runs the fixed cap cases through the full agreement, and checks
// that each policy is valid or void as the caps say (a void policy alone
// grants no row).
func i5F2Caps(ctx context.Context, tx pgx.Tx, w *i5World, principals []identity.Principal) (string, error) {
	var st i5F2Stats
	var names []string
	for _, cc := range w.i5CapCases() {
		cases := []i5F2Case{{Policy: cc.Policy, Principals: principals}}
		s, err := i5F2AgreeDB(ctx, tx, w, cases, i5F2Real, true)
		if err != nil {
			return "", fmt.Errorf("%s: %w", cc.Name, err)
		}
		st.checks, st.split, st.fallback = st.checks+s.checks, st.split+s.split, st.fallback+s.fallback
		for _, np := range cc.Policy.Document.Policies {
			only := i5WithDocument(cc.Policy, authz.Document{Format: authz.FormatV2, Conditions: cc.Policy.Document.Conditions, Policies: []authz.NamedPolicy{np}})
			sql, _ := authz.CompileRead(only, principals[0], "r", 2)
			if void := sql == "false"; void == cc.Valid[np.Name] {
				return "", fmt.Errorf("%s: policy %s compiles to %q, want valid = %v", cc.Name, np.Name, sql, cc.Valid[np.Name])
			}
		}
		names = append(names, cc.Name)
	}
	return fmt.Sprintf("%s: %d checks agree; %d principal plans split into branches, %d fall back to CompileRead", strings.Join(names, "; "), st.checks, st.split, st.fallback), nil
}

// i5F1Equivalence (D12): each format-1 fixture, remapped onto the world and
// converted as a stored row is on read, gives through CompileRead and
// through ReadBranches exactly the rows the S10 rules grant, for every
// principal. Branch count and SQL text are TestReadBranchesFormat1Golden's
// (TICK-938 P11-06, CI-6).
func i5F1Equivalence(ctx context.Context, tx pgx.Tx, w *i5World, principals []identity.Principal) (string, error) {
	fixtures, err := w.i5Format1Fixtures()
	if err != nil {
		return "", err
	}
	checks, granted := 0, 0
	for _, f := range fixtures {
		pol := &authz.Policy{EntityDefinitionID: w.Definition, Version: 1, Document: authz.ConvertFormat1(f.Rules)}
		bySQL, preds, err := i5Select(ctx, tx, w, i5Case{Policy: pol, Principals: principals}, authz.CompileRead)
		if err != nil {
			return "", fmt.Errorf("%s: %w", f.Name, err)
		}
		var st i5F2Stats
		byBranch, err := i5F2BranchSelect(ctx, tx, w, func(identity.Principal) *authz.Policy { return pol }, principals, &st)
		if err != nil {
			return "", fmt.Errorf("%s branches: %w", f.Name, err)
		}
		for ki, p := range principals {
			for _, rec := range w.Records {
				checks++
				row := rec.row(p)
				want := i5S10Allows(f.Rules, p, authz.OpRead, row)
				if want {
					granted++
				}
				if got := bySQL[ki][rec.Row.ID]; got != want {
					return "", fmt.Errorf("%s principal #%d record %s: CompileRead %v, S10 %v; predicate %s", f.Name, ki, rec.Row.ID, got, want, preds[ki])
				}
				if n := byBranch[ki][rec.Row.ID]; n > 1 || (n == 1) != want {
					return "", fmt.Errorf("%s principal #%d record %s: branches return it %d times, S10 %v", f.Name, ki, rec.Row.ID, n, want)
				}
			}
		}
	}
	if granted == 0 {
		return "", errors.New("no fixture grants any row; the check proves nothing")
	}
	return fmt.Sprintf("%d fixtures × %d principals × %d records agree (%d checks, %d granted); random S10 rule sets: TestI5Format1ConvertedMatchesS10 and the I5 agreement row",
		len(fixtures), len(principals), len(w.Records), checks, granted), nil
}
