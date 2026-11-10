package tenancy

import pgx "github.com/jackc/pgx/v5"

// internal/tenancy owns the transaction and may reach its connection.
func conn(tx pgx.Tx) *pgx.Conn { return tx.Conn() }
