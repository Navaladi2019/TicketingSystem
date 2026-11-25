package isolation

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestOrgUnitFKRefused (TICK-1126): the org-unit INSERT paths accept only
// 23503 from records_org_unit_fkey, so a lost INSERT grant or another
// constraint cannot pass for the FK.
func TestOrgUnitFKRefused(t *testing.T) {
	const fkey = "records_org_unit_fkey"
	pg := func(code, constraint string) error {
		return &pgconn.PgError{Code: code, ConstraintName: constraint}
	}
	for _, c := range []struct {
		name string
		err  error
		ok   bool
	}{
		{"23503 from the FK", pg("23503", fkey), true},
		{"23503 from the FK, wrapped", fmt.Errorf("insert: %w", pg("23503", fkey)), true},
		{"nil: the row was written", nil, false},
		{"42501: no grant, or a policy", pg("42501", ""), false},
		{"42501 that names the FK", pg("42501", fkey), false},
		{"23503 from another FK", pg("23503", "records_assignment_group_fkey"), false},
		{"23503 with no constraint name", pg("23503", ""), false},
		{"23505 from the primary key", pg("23505", "records_pkey"), false},
		{"23505 that names the FK", pg("23505", fkey), false},
		{"23514 from a check", pg("23514", "records_pipeline_all_or_none_check"), false},
		{"not a PostgreSQL error", errors.New("connection reset"), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := orgUnitFKRefused("nava_app", c.err)
			if c.ok != (err == nil) {
				t.Fatalf("orgUnitFKRefused(%v) = %v, want accepted = %v", c.err, err, c.ok)
			}
			if c.err != nil && err != nil && !errors.Is(err, c.err) {
				t.Errorf("orgUnitFKRefused(%v) = %v, which does not wrap the cause", c.err, err)
			}
		})
	}
}
