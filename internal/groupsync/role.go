package groupsync

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// RowQuerier is the part of a connection AssertRole needs. *pgxpool.Pool and
// *pgx.Conn satisfy it.
type RowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// AssertRole refuses to run unless the connection is nava_groupsync and that
// role has neither BYPASSRLS nor SUPERUSER, so RLS always applies and the
// explicit tenant_id predicates are not the only guard (TICK-222). Errors
// name the role, never the DSN.
func AssertRole(ctx context.Context, db RowQuerier) error {
	var role string
	var ok bool
	err := db.QueryRow(ctx, `SELECT current_user::text,
		current_user = 'nava_groupsync' AND NOT rolbypassrls AND NOT rolsuper
		FROM pg_roles WHERE rolname = current_user`).Scan(&role, &ok)
	if err != nil {
		return errors.New("cannot verify the database role of nava-groupsync")
	}
	if !ok {
		return fmt.Errorf("nava-groupsync must connect as nava_groupsync without BYPASSRLS or SUPERUSER; connected as %q", role)
	}
	return nil
}
