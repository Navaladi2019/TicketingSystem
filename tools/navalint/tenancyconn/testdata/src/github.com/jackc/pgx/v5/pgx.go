// Package pgx is a stub of github.com/jackc/pgx/v5: the Tx interface, its
// Conn() and LargeObjects().
package pgx

import "context"

type Conn struct{}

func (c *Conn) Exec(ctx context.Context, sql string) error { return nil }

type LargeObjects struct{}

func (o *LargeObjects) Unlink(ctx context.Context, oid uint32) error { return nil }

type Tx interface {
	Begin(ctx context.Context) (Tx, error)
	Exec(ctx context.Context, sql string) error
	Conn() *Conn
	LargeObjects() LargeObjects
}
