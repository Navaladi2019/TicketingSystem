//go:build integration

// I5 agreement (PLAN-003 P3-32, TICK-617; ADR-001 I5: one authorization
// layer, inside the query). For random policies, principals and records,
//
//	SELECT r.id FROM records r WHERE <authz.CompileRead(pol, p)>
//
// run as nava_app under RLS returns exactly {r | authz.Allows(pol, p, "read", r)},
// with Row.SharedWith built by A1's pair-matched rule. Shares that name
// principals other than the caller (outsiders, and decoys that carry a
// caller's id under the other kind) are in the database too, so a compiler
// that ignores the pair, or the caller, admits rows the Go check does not.
//
// Each property is a named row in isolation-report.md under I5. The world is
// seeded inside one superuser transaction that switches to nava_app for the
// reads and is rolled back, so the deferred commit checks never run and
// tenants A and B are left as the seed made them.
//
// The fixed CI seed is i5DefaultSeed; NAVA_I5_SEED=<n> make isolation replays
// another. NAVA_I5_MUTANT=<name> runs the agreement row against a mutant
// compiler instead of the real one, which must turn it red (the PR's
// mutation run).

package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

// i5Budget is the P3-32 time budget for the agreement row.
const i5Budget = 60 * time.Second

// The I5 row names (the report's Path column).
const (
	i5RowAgreement = "I5 agreement: SELECT … WHERE CompileRead equals {r | Allows(read, r)} for random policies × principals × records"
	i5RowLoader    = "I5 loader: authz.SharesFor fills Row.SharedWith pair-matched, as the generator does (A1, AR-S10-01)"
	i5RowMutant    = "I5 mutant %s: the suite catches a broken compiler"
	i5RowFilters   = "I5 data filters: eq and in on data fields agree with the Go check on null, absent, defaulted and type-changed fields"
	i5RowListPage  = "I5 list route: a record the caller cannot read is absent from a list page (PR #523 security finding 2)"
)

// TestI5Agreement runs the I5 properties and adds their rows to the report.
func TestI5Agreement(t *testing.T) {
	seed, err := i5Seed()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("I5 seed %d (replay: %s=%d make isolation)", seed, i5SeedEnv, seed)
	w, cases := i5Generate(seed)
	compile, compiler := i5Compiler(authz.CompileRead), "authz.CompileRead"
	if m := os.Getenv(i5MutantEnv); m != "" {
		mc, ok := i5Mutants()[m]
		if !ok {
			t.Fatalf("%s=%q: not one of %v", i5MutantEnv, m, i5MutantNames())
		}
		compile, compiler = mc, "mutant "+m
	}

	add := func(name, status, detail string) {
		suite.report.paths = append(suite.report.paths, pathResult{Path: Path{Name: name, Case: "I5"}, Status: status, Detail: detail})
		if status == statusFail {
			t.Errorf("%s: %s", name, detail)
		}
	}
	err = i5InWorld(t, w, func(ctx context.Context, tx pgx.Tx) error {
		start := time.Now()
		st, ferr := i5Agree(ctx, tx, w, cases, compile)
		took := time.Since(start).Round(time.Millisecond)
		switch {
		case ferr != nil:
			add(i5RowAgreement, statusFail, fmt.Sprintf("seed %d, %s: %v", seed, compiler, ferr))
		case took > i5Budget:
			add(i5RowAgreement, statusFail, fmt.Sprintf("seed %d: %s, over the %s budget", seed, took, i5Budget))
		default:
			add(i5RowAgreement, statusPass, fmt.Sprintf("seed %d, %s: %d policies × %d principals × %d records agree (%d checks; %d cases admit a proper subset, %d via a share); %s",
				seed, compiler, len(cases), i5Principals, len(w.Records), st.checks, st.proper, st.viaShare, took))
		}

		if detail, lerr := i5Loader(ctx, tx, w, cases); lerr != nil {
			add(i5RowLoader, statusFail, fmt.Sprintf("seed %d: %v", seed, lerr))
		} else {
			add(i5RowLoader, statusPass, detail)
		}

		for _, name := range i5MutantNames() {
			_, merr := i5Agree(ctx, tx, w, cases, i5Mutants()[name])
			var d *i5Disagreement
			switch {
			case errors.As(merr, &d):
				add(fmt.Sprintf(i5RowMutant, name), statusPass, "caught: "+d.Error())
			case merr != nil:
				add(fmt.Sprintf(i5RowMutant, name), statusFail, "the run broke, which proves nothing: "+merr.Error())
			default:
				add(fmt.Sprintf(i5RowMutant, name), statusFail, fmt.Sprintf("seed %d: the mutant agreed on every case; the suite cannot see it", seed))
			}
		}
		return nil
	})
	if err != nil {
		add(i5RowAgreement, statusFail, fmt.Sprintf("seed %d: the run broke: %v", seed, err))
	}
	// P3-34 (TICK-619) adds the data filter compiler and the list route; it
	// turns these two rows into checks (P3-38 waits on the first).
	add(i5RowFilters, statusPending, "pending P3-34 (TICK-619): no data filter compiler yet; P3-38 starts only when this row is a PASS")
	add(i5RowListPage, statusPending, "pending P3-34 (TICK-619): no list route yet")
}

// i5InWorld seeds w as the superuser in one transaction, switches it to
// nava_app with w's tenant set, runs fn, and rolls everything back.
func i5InWorld(t *testing.T, w *i5World, fn func(context.Context, pgx.Tx) error) error {
	t.Helper()
	ctx := t.Context()
	conn := suite.superConn(t)
	errDone := errors.New("i5: done")
	err := pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if err := i5SeedWorld(ctx, tx, w); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
		if _, err := tx.Exec(ctx, "SET LOCAL ROLE nava_app"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", w.Tenant.String()); err != nil {
			return err
		}
		if err := fn(ctx, tx); err != nil {
			return err
		}
		return errDone
	})
	if errors.Is(err, errDone) {
		return nil
	}
	return err
}

func i5SeedWorld(ctx context.Context, tx pgx.Tx, w *i5World) error {
	b := &pgx.Batch{}
	b.Queue("INSERT INTO tenants (id, state) VALUES ($1, 'active')", w.Tenant)
	for _, id := range w.allPrincipals() {
		b.Queue(`INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, $2, $3, true, now())`,
			w.Tenant, id, "kc-i5-"+id.String())
	}
	for i, id := range w.allGroups() {
		b.Queue(`INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)`,
			w.Tenant, id, "kc-i5-"+id.String(), "/t/"+w.Tenant.String()+"/i5-"+strconv.Itoa(i))
	}
	b.Queue(`INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version) VALUES ($1, $2, 'i5case', 'I5', 'IFV', 1)`,
		w.Tenant, w.Definition)
	for i, rec := range w.Records {
		b.Queue(`INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, requester_id, assignee_id, assignment_group_id)
			VALUES ($1, $2, $3, 1, $4, $5, $6, $7, $8)`,
			w.Tenant, rec.Row.ID, w.Definition, fmt.Sprintf("IFV%07d", i+1), rec.Row.State, rec.Row.RequesterID, rec.Row.AssigneeID, rec.Row.AssignmentGroupID)
		for _, s := range rec.Shares {
			column := "user_id"
			if s.Kind == "group" {
				column = "group_id"
			}
			b.Queue(`INSERT INTO record_shares (tenant_id, record_id, `+column+`, principal_kind) VALUES ($1, $2, $3, $4)`,
				w.Tenant, rec.Row.ID, s.ID, s.Kind)
		}
	}
	return tx.SendBatch(ctx, b).Close()
}

// i5Stats says how much the run discriminated: agreement on "nothing" or
// "everything" alone proves little.
type i5Stats struct {
	checks, proper, viaShare int
}

// i5Disagreement is the first case where SQL and Go differ, with what
// replays it.
type i5Disagreement struct {
	Policy, Principal int
	Record            uuid.UUID
	SQLAdmits         bool
	Pol               *authz.Policy
	P                 identity.Principal
	Row               authz.Row
	Predicate         string
}

func (d *i5Disagreement) Error() string {
	side := "Allows admits it and the SQL does not"
	if d.SQLAdmits {
		side = "the SQL admits it and Allows does not"
	}
	j := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	return fmt.Sprintf("policy #%d, principal #%d, record %s: %s; policy %s; principal %s; row %s; predicate %s",
		d.Policy, d.Principal, d.Record, side, j(d.Pol), j(d.P), j(d.Row), d.Predicate)
}

// i5Agree checks every case with compile on the SQL side and authz.Allows on
// the Go side. It returns the first disagreement as an *i5Disagreement, so a
// mutant's run stops at the first policy it breaks.
func i5Agree(ctx context.Context, tx pgx.Tx, w *i5World, cases []i5Case, compile i5Compiler) (i5Stats, error) {
	var st i5Stats
	for pi, c := range cases {
		bySQL, preds, err := i5Select(ctx, tx, w, c, compile)
		if err != nil {
			return st, fmt.Errorf("policy #%d: %w", pi, err)
		}
		var first *i5Disagreement
		for ki, p := range c.Principals {
			admitted, shared := 0, false
			for _, rec := range w.Records {
				st.checks++
				row := rec.row(p)
				_, goOK := authz.Allows(c.Policy, p, authz.OpRead, row)
				sqlOK := bySQL[ki][rec.Row.ID]
				if goOK {
					admitted++
					shared = shared || len(row.SharedWith) > 0
				}
				if goOK != sqlOK && first == nil {
					first = &i5Disagreement{Policy: pi, Principal: ki, Record: rec.Row.ID, SQLAdmits: sqlOK,
						Pol: c.Policy, P: p, Row: row, Predicate: preds[ki]}
				}
			}
			if admitted > 0 && admitted < len(w.Records) {
				st.proper++
				if shared {
					st.viaShare++
				}
			}
		}
		if first != nil {
			return st, first
		}
	}
	// The run must discriminate, or agreement proves nothing.
	if total := len(cases) * i5Principals; st.proper < total/5 {
		return st, fmt.Errorf("only %d of %d cases admit a proper subset of the records", st.proper, total)
	}
	return st, nil
}

// i5Select runs one query for a policy: one UNION ALL arm per principal,
// each with its own CompileRead predicate numbered after the arms before it,
// and returns, per principal, the set of record ids it admits.
func i5Select(ctx context.Context, tx pgx.Tx, w *i5World, c i5Case, compile i5Compiler) ([]map[uuid.UUID]bool, []string, error) {
	args := []any{w.Tenant, w.Definition}
	arms := make([]string, len(c.Principals))
	preds := make([]string, len(c.Principals))
	for k, p := range c.Principals {
		pred, pargs := compile(c.Policy, p, "r", len(args))
		preds[k] = pred
		args = append(args, pargs...)
		arms[k] = fmt.Sprintf("SELECT %d AS k, r.id FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND %s", k, pred)
	}
	rows, err := tx.Query(ctx, strings.Join(arms, "\nUNION ALL\n"), args...)
	if err != nil {
		return nil, preds, err
	}
	out := make([]map[uuid.UUID]bool, len(c.Principals))
	for k := range out {
		out[k] = map[uuid.UUID]bool{}
	}
	var (
		k  int
		id uuid.UUID
	)
	_, err = pgx.ForEachRow(rows, []any{&k, &id}, func() error {
		out[k][id] = true
		return nil
	})
	return out, preds, err
}

// i5Loader checks that authz.SharesFor, the loader the pipeline builds
// Row.SharedWith with, returns what the generator's pair-matched rule does,
// for the first callers of the run against every record.
func i5Loader(ctx context.Context, tx pgx.Tx, w *i5World, cases []i5Case) (string, error) {
	n, nonEmpty := 0, 0
	for _, p := range cases[0].Principals[:i5LoaderCallers] {
		for _, rec := range w.Records {
			got, err := authz.SharesFor(ctx, tx, p, rec.Row.ID)
			if err != nil {
				return "", err
			}
			want := rec.sharedWith(p)
			slices.SortFunc(got, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
			if !slices.Equal(got, want) {
				return "", fmt.Errorf("record %s, principal %s in %v: SharesFor %v, pair-matched shares %v (all shares %v)",
					rec.Row.ID, p.PrincipalID, p.Groups, got, want, rec.Shares)
			}
			n++
			if len(want) > 0 {
				nonEmpty++
			}
		}
	}
	if nonEmpty == 0 {
		return "", errors.New("no caller has a share, so the check proves nothing")
	}
	return fmt.Sprintf("%d caller × record pairs agree, %d with a share", n, nonEmpty), nil
}
