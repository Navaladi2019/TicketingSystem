package server

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// Every 23514 on acl_versions is a 422 and never a 500 (A1-2d, TICK-940's
// DETAIL contract).
func TestPgCheckProblemMapsEvery23514(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		want string // "pointer rule"
	}{
		{"bytes rules", &pgconn.PgError{Code: "23514", ConstraintName: "acl_versions_bytes_rules_check"}, "/rules document_too_large"},
		{"bytes masks", &pgconn.PgError{Code: "23514", ConstraintName: "acl_versions_bytes_masks_check"}, "/masks document_too_large"},
		{"rules check with detail", &pgconn.PgError{Code: "23514", ConstraintName: "acl_versions_rules_check", Detail: `{"column": "rules", "code": "ref_cycle", "pointer": "/conditions/0/expr/ref"}`}, "/rules/conditions/0/expr/ref ref_cycle"},
		{"trigger foreign id in masks", &pgconn.PgError{Code: "23514", Detail: `{"column": "masks", "code": "foreign_id", "pointer": "/0/read/groups/0"}`}, "/masks/0/read/groups/0 foreign_id"},
		{"trigger foreign id without a pointer", &pgconn.PgError{Code: "23514", Detail: `{"column": "rules", "code": "foreign_id"}`}, "/rules foreign_id"},
		{"another check without detail", &pgconn.PgError{Code: "23514", ConstraintName: "acl_versions_masks_check"}, " value"},
		{"detail that is not JSON", &pgconn.PgError{Code: "23514", Detail: "nope"}, " value"},
		{"detail of another column", &pgconn.PgError{Code: "23514", Detail: `{"column": "x", "code": "c", "pointer": "/p"}`}, " value"},
		{"wrapped", fmt.Errorf("authz: insert acl version: %w", &pgconn.PgError{Code: "23514", ConstraintName: "acl_versions_bytes_rules_check"}), "/rules document_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ps, ok := pgCheckProblem(tc.err)
			if !ok || len(ps) != 1 || ps[0].Pointer+" "+ps[0].Rule != tc.want {
				t.Errorf("pgCheckProblem = %+v, %v, want %q", ps, ok, tc.want)
			}
		})
	}
	for _, err := range []error{nil, errors.New("x"), &pgconn.PgError{Code: "42501"}, &pgconn.PgError{Code: "23505"}} {
		if ps, ok := pgCheckProblem(err); ok {
			t.Errorf("pgCheckProblem(%v) = %+v, want no mapping", err, ps)
		}
	}
}
