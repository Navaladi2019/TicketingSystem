package entitytypes

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RegisterSeedFilterable refuses, before it writes anything, every type that
// is not an onboarding seed at version 1.
func TestRegisterSeedFilterableRefusesWhatIsNotAnOnboardingSeed(t *testing.T) {
	t.Parallel()
	system, other := "onboarding", "someone-else"
	principal := uuid.New()
	lock := func(current int) step {
		return step{scan: func(d []any) error {
			*d[0].(*uuid.UUID), *d[1].(*int) = uuid.New(), current
			return nil
		}}
	}
	version := func(sys *string, by *uuid.UUID) step {
		return step{scan: func(d []any) error {
			*d[0].(**string), *d[1].(**uuid.UUID) = sys, by
			return nil
		}}
	}
	for name, c := range map[string]struct {
		steps []step
		want  error
	}{
		"unknown key":                  {[]step{{scan: func([]any) error { return pgx.ErrNoRows }}}, ErrNotFound},
		"a published v2":               {[]step{lock(2)}, ErrNotOnboardingSeed},
		"v1 published by a principal":  {[]step{lock(1), version(nil, &principal)}, ErrNotOnboardingSeed},
		"v1 with another system word":  {[]step{lock(1), version(&other, nil)}, ErrNotOnboardingSeed},
		"v1 system word and principal": {[]step{lock(1), version(&system, &principal)}, ErrNotOnboardingSeed},
	} {
		err := Store{}.RegisterSeedFilterable(context.Background(), &fakeTx{t: t, steps: c.steps}, uuid.New(), "incident")
		if !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", name, err, c.want)
		}
	}
}

// A version 1 with no publisher at all was seeded before TICK-1172, flags
// nothing and writes nothing: a retried onboarding must go on to Activate.
func TestRegisterSeedFilterableIgnoresAPreColumnSeed(t *testing.T) {
	t.Parallel()
	steps := []step{
		{scan: func(d []any) error { *d[0].(*uuid.UUID), *d[1].(*int) = uuid.New(), 1; return nil }},
		{scan: func(d []any) error { *d[0].(**string), *d[1].(**uuid.UUID) = nil, nil; return nil }},
	}
	if err := (Store{}).RegisterSeedFilterable(context.Background(), &fakeTx{t: t, steps: steps}, uuid.New(), "incident"); err != nil {
		t.Fatalf("a v1 with no publisher = %v, want nil", err)
	}
}
