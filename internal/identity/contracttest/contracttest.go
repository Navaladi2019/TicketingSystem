// Package contracttest is the adapter-agnostic contract suite of the identity
// port (ADR-001 A.4, CI-11). Every adapter must pass it: the adapter's test
// builds a Suite with its implementations and the tokens only it can mint,
// and calls Run.
//
// The suite checks invariants on every case, built-in or supplied:
//   - an error wraps exactly one of identity.ErrInvalidToken and
//     identity.ErrTenantUnresolved, so T20 can map it without matching text;
//   - a refused token yields a zero Principal or Operator;
//   - an accepted Principal has a tenant and a principal id and no nil group;
//   - the resolvers are safe for concurrent use (run under -race).
//
// T17 adds the tenant-resolution cases, T18 the directory cases, and T19 its
// adversarial cases, each by appending to the Suite's case lists.
//
// T19 (ADR-001 Annex B CI-5): a case may carry a catalogue id (Case, such as
// "CI-5.03"). A Suite that lists "CI-5" in Require fails unless every entry of
// CI5Cases is covered by at least one supplied case, so a future adapter
// inherits the whole CI-5 list, not only the cases that are easy to mint. A
// tagged case runs as a subtest named "[CI-5.03] ...", which is what
// tools/ci/ci5report matches.
package contracttest

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// ResolverCase is one token for IdentityResolver.Resolve. Set Want for a token
// the contract accepts, or WantErr (a sentinel) for one it refuses.
type ResolverCase struct {
	Name string
	// Case is a catalogue id ("CI-5.03"), or empty.
	Case    string
	Token   string
	Want    *identity.Principal
	WantErr error
}

// OperatorCase is one token for OperatorResolver.ResolveOperator. Set Want or
// WantErr as for ResolverCase.
type OperatorCase struct {
	Name    string
	Case    string
	Token   string
	Want    *identity.Operator
	WantErr error
}

// DirectoryCase is one DirectoryAdmin scenario.
type DirectoryCase struct {
	Name string
	Case string
	Run  func(t *testing.T, d identity.DirectoryAdmin)
}

// ScenarioCase is an attack the port alone cannot express: one the IdP itself
// must refuse (a brokered login, the organization picker) or the platform
// database must (a duplicate binding). Run fails the test when the attacker
// gets anything a tenant should not have.
type ScenarioCase struct {
	Name string
	Case string
	Run  func(t *testing.T)
}

// Suite is what an adapter brings. A nil implementation skips its part.
type Suite struct {
	Resolver      identity.IdentityResolver
	ResolverCases []ResolverCase

	Operator      identity.OperatorResolver
	OperatorCases []OperatorCase

	Directory      identity.DirectoryAdmin
	DirectoryCases []DirectoryCase

	Scenarios []ScenarioCase

	// Require lists the catalogues ("CI-5") the supplied cases must cover
	// entry by entry.
	Require []string
}

// garbage are tokens no adapter may accept, whatever its token format.
func garbage() []string {
	unsigned := "eyJhbGciOiJub25lIn0." + // {"alg":"none"}
		"eyJpc3MiOiJodHRwczovL2lkcC5pbnZhbGlkIiwic3ViIjoieCIsImF1ZCI6Im5hdmEtYXBpIiwiZXhwIjo0MTAyNDQ0ODAwfQ." // far-future exp
	return []string{
		"", " ", "not-a-token", "a.b.c", "..", "Bearer x", unsigned,
		strings.Repeat("A", 1<<20),
	}
}

// Run runs the suite.
func (s Suite) Run(t *testing.T) {
	t.Helper()
	if len(s.Require) > 0 {
		t.Run("coverage", func(t *testing.T) {
			for _, missing := range s.Uncovered() {
				t.Errorf("no case covers %s", missing)
			}
		})
	}
	t.Run("IdentityResolver", func(t *testing.T) {
		if s.Resolver == nil {
			t.Skip("adapter has no IdentityResolver")
		}
		cases := slices.Concat(builtinResolverCases(), s.ResolverCases)
		for _, c := range cases {
			t.Run(caseName(c.Case, c.Name), func(t *testing.T) { checkResolve(t, s.Resolver, c) })
		}
		t.Run("concurrent", func(t *testing.T) {
			var wg sync.WaitGroup
			for range 4 {
				for _, c := range cases {
					wg.Go(func() { checkResolve(t, s.Resolver, c) })
				}
			}
			wg.Wait()
		})
	})
	t.Run("OperatorResolver", func(t *testing.T) {
		if s.Operator == nil {
			t.Skip("adapter has no OperatorResolver")
		}
		cases := s.OperatorCases
		for _, g := range garbage() {
			cases = append(cases, OperatorCase{Name: "garbage " + clip(g), Token: g, WantErr: identity.ErrInvalidToken})
		}
		for _, c := range cases {
			t.Run(caseName(c.Case, c.Name), func(t *testing.T) { checkOperator(t, s.Operator, c) })
		}
	})
	t.Run("DirectoryAdmin", func(t *testing.T) {
		if s.Directory == nil {
			t.Skip("adapter has no DirectoryAdmin")
		}
		for _, c := range s.DirectoryCases {
			t.Run(caseName(c.Case, c.Name), func(t *testing.T) { c.Run(t, s.Directory) })
		}
	})
	if len(s.Scenarios) > 0 {
		t.Run("Scenarios", func(t *testing.T) {
			for _, c := range s.Scenarios {
				t.Run(caseName(c.Case, c.Name), c.Run)
			}
		})
	}
}

// caseName prefixes a tagged case with its id: "[CI-5.03] name".
func caseName(id, name string) string {
	if id == "" {
		return name
	}
	return "[" + id + "] " + name
}

// Uncovered lists the entries of the Require catalogues that no supplied case
// carries, as "CI-5.03 an unknown issuer".
func (s Suite) Uncovered() []string {
	have := map[string]bool{}
	for _, c := range s.ResolverCases {
		have[c.Case] = true
	}
	for _, c := range s.OperatorCases {
		have[c.Case] = true
	}
	for _, c := range s.DirectoryCases {
		have[c.Case] = true
	}
	for _, c := range s.Scenarios {
		have[c.Case] = true
	}
	var out []string
	for _, req := range s.Require {
		cat, ok := Catalogues()[req]
		if !ok {
			out = append(out, req+" (no such catalogue)")
			continue
		}
		for _, c := range cat {
			if !have[c.ID] {
				out = append(out, c.ID+" "+c.Title)
			}
		}
	}
	return out
}

func builtinResolverCases() []ResolverCase {
	var cases []ResolverCase
	for _, g := range garbage() {
		cases = append(cases, ResolverCase{Name: "garbage " + clip(g), Token: g, WantErr: identity.ErrInvalidToken})
	}
	return cases
}

func checkResolve(t *testing.T, r identity.IdentityResolver, c ResolverCase) {
	t.Helper()
	if (c.Want == nil) == (c.WantErr == nil) {
		t.Fatalf("case %q: set exactly one of Want and WantErr", c.Name)
	}
	got, err := r.Resolve(context.Background(), c.Token)
	if err != nil {
		checkSentinel(t, err)
		if !reflect.DeepEqual(got, identity.Principal{}) {
			t.Errorf("refused token returned a principal: %+v", got)
		}
		if c.WantErr == nil {
			t.Errorf("err = %v, want principal %+v", err, *c.Want)
		} else if !errors.Is(err, c.WantErr) {
			t.Errorf("err = %v, want %v", err, c.WantErr)
		}
		return
	}
	if c.WantErr != nil {
		t.Fatalf("accepted a token the contract refuses (%v): %+v", c.WantErr, got)
	}
	if got.TenantID == uuid.Nil || got.PrincipalID == uuid.Nil {
		t.Errorf("accepted principal has a nil tenant or principal id: %+v", got)
	}
	for _, g := range got.Groups {
		if g == uuid.Nil {
			t.Errorf("accepted principal has a nil group: %+v", got)
		}
	}
	if !reflect.DeepEqual(got, *c.Want) {
		t.Errorf("principal = %+v, want %+v", got, *c.Want)
	}
}

func checkOperator(t *testing.T, r identity.OperatorResolver, c OperatorCase) {
	t.Helper()
	if (c.Want == nil) == (c.WantErr == nil) {
		t.Fatalf("case %q: set exactly one of Want and WantErr", c.Name)
	}
	got, err := r.ResolveOperator(context.Background(), c.Token)
	if err != nil {
		checkSentinel(t, err)
		if got != (identity.Operator{}) {
			t.Errorf("refused token returned an operator: %+v", got)
		}
		if c.WantErr == nil || !errors.Is(err, c.WantErr) {
			t.Errorf("err = %v, want %v", err, c.WantErr)
		}
		return
	}
	if c.WantErr != nil {
		t.Fatalf("accepted a token the contract refuses (%v): %+v", c.WantErr, got)
	}
	if got.PrincipalID == "" {
		t.Errorf("accepted operator has no principal id")
	}
	if got != *c.Want {
		t.Errorf("operator = %+v, want %+v", got, *c.Want)
	}
}

// checkSentinel fails unless err wraps exactly one sentinel.
func checkSentinel(t *testing.T, err error) {
	t.Helper()
	invalid, unresolved := errors.Is(err, identity.ErrInvalidToken), errors.Is(err, identity.ErrTenantUnresolved)
	if invalid == unresolved {
		t.Errorf("err %v must wrap exactly one of ErrInvalidToken and ErrTenantUnresolved", err)
	}
}

func clip(s string) string {
	if len(s) > 12 {
		return s[:12] + "…"
	}
	return s
}

// CommonDirectoryCases are the DirectoryAdmin invariants any adapter must keep
// on a live directory, whatever it stores:
//   - every method refuses uuid.Nil as the tenant with ErrInvalidInput, before
//     any call to the IdP;
//   - ChangesSince is a poll and diff: its cursor, fed back, returns
//     Unchanged and nothing else, and a stale cursor returns a full snapshot
//     under the same digest;
//   - a snapshot of a tenant that has no directory is empty, not an error;
//   - an unknown principal or group is ErrNotFound.
func CommonDirectoryCases() []DirectoryCase {
	ctx := context.Background()
	return []DirectoryCase{
		{Name: "nil tenant is refused", Run: func(t *testing.T, d identity.DirectoryAdmin) {
			if _, err := d.CreateTenantOrg(ctx, uuid.Nil, "x", nil); !errors.Is(err, identity.ErrInvalidInput) {
				t.Errorf("CreateTenantOrg: %v", err)
			}
			if err := d.InviteUser(ctx, uuid.Nil, "a@b.test"); !errors.Is(err, identity.ErrInvalidInput) {
				t.Errorf("InviteUser: %v", err)
			}
			if err := d.DisableUser(ctx, uuid.Nil, uuid.New()); !errors.Is(err, identity.ErrInvalidInput) {
				t.Errorf("DisableUser: %v", err)
			}
			if err := d.SetGroupMembers(ctx, uuid.Nil, uuid.New(), nil); !errors.Is(err, identity.ErrInvalidInput) {
				t.Errorf("SetGroupMembers: %v", err)
			}
			if _, _, err := d.ChangesSince(ctx, uuid.Nil, ""); !errors.Is(err, identity.ErrInvalidInput) {
				t.Errorf("ChangesSince: %v", err)
			}
		}},
		{Name: "cursor round trip", Run: func(t *testing.T, d identity.DirectoryAdmin) {
			tenant := uuid.New()
			first, cur, err := d.ChangesSince(ctx, tenant, "")
			if err != nil {
				t.Fatal(err)
			}
			if first.Unchanged || cur == "" {
				t.Fatalf("first snapshot: unchanged=%v cursor=%q", first.Unchanged, cur)
			}
			again, cur2, err := d.ChangesSince(ctx, tenant, cur)
			if err != nil {
				t.Fatal(err)
			}
			if !again.Unchanged || cur2 != cur || len(again.Users)+len(again.Groups)+len(again.OrgUnits) != 0 {
				t.Errorf("same cursor: %+v %q", again, cur2)
			}
			stale, cur3, err := d.ChangesSince(ctx, tenant, "stale")
			if err != nil || stale.Unchanged || cur3 != cur {
				t.Errorf("stale cursor: unchanged=%v cursor=%q err=%v", stale.Unchanged, cur3, err)
			}
		}},
		{Name: "unknown principal and group", Run: func(t *testing.T, d identity.DirectoryAdmin) {
			tenant := uuid.New()
			if err := d.DisableUser(ctx, tenant, uuid.New()); !errors.Is(err, identity.ErrNotFound) {
				t.Errorf("DisableUser: %v", err)
			}
			if err := d.SetGroupMembers(ctx, tenant, uuid.New(), nil); !errors.Is(err, identity.ErrNotFound) {
				t.Errorf("SetGroupMembers: %v", err)
			}
		}},
	}
}
