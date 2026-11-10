// Package pgx is a stub of github.com/jackc/pgx/v5: the Tx interface and its
// Conn().
package pgx

import "context"

type Conn struct{}

func (c *Conn) Exec(ctx context.Context, sql string) error { return nil }

type Tx interface {
	Begin(ctx context.Context) (Tx, error)
	Exec(ctx context.Context, sql string) error
	Conn() *Conn
}
