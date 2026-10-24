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
	Name    string
	Token   string
	Want    *identity.Principal
	WantErr error
}

// OperatorCase is one token for OperatorResolver.ResolveOperator. Set Want or
// WantErr as for ResolverCase.
type OperatorCase struct {
	Name    string
	Token   string
	Want    *identity.Operator
	WantErr error
}

// DirectoryCase is one DirectoryAdmin scenario.
type DirectoryCase struct {
	Name string
	Run  func(t *testing.T, d identity.DirectoryAdmin)
}

// Suite is what an adapter brings. A nil implementation skips its part.
type Suite struct {
	Resolver      identity.IdentityResolver
	ResolverCases []ResolverCase

	Operator      identity.OperatorResolver
	OperatorCases []OperatorCase

	Directory      identity.DirectoryAdmin
	DirectoryCases []DirectoryCase
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
	t.Run("IdentityResolver", func(t *testing.T) {
		if s.Resolver == nil {
			t.Skip("adapter has no IdentityResolver")
		}
		cases := slices.Concat(builtinResolverCases(), s.ResolverCases)
		for _, c := range cases {
			t.Run(c.Name, func(t *testing.T) { checkResolve(t, s.Resolver, c) })
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
			t.Run(c.Name, func(t *testing.T) { checkOperator(t, s.Operator, c) })
		}
	})
	t.Run("DirectoryAdmin", func(t *testing.T) {
		if s.Directory == nil {
			t.Skip("adapter has no DirectoryAdmin yet (T18)")
		}
		for _, c := range s.DirectoryCases {
			t.Run(c.Name, func(t *testing.T) { c.Run(t, s.Directory) })
		}
	})
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
