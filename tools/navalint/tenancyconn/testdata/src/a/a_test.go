package a

import pgx "github.com/jackc/pgx/v5"

// A _test.go harness may reach the connection.
func harness(tx pgx.Tx) *pgx.Conn { return tx.Conn() }
