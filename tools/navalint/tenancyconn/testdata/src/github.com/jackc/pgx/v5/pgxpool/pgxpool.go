// Package pgxpool is a stub: a pooled connection's Conn() is not a
// transaction's.
package pgxpool

import pgx "github.com/jackc/pgx/v5"

type Conn struct{}

func (c *Conn) Conn() *pgx.Conn { return nil }
