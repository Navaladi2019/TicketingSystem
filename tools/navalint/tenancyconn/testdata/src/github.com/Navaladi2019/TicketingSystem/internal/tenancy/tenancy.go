package tenancy

import pgx "github.com/jackc/pgx/v5"

// internal/tenancy owns the transaction and may reach its connection.
func conn(tx pgx.Tx) *pgx.Conn { return tx.Conn() }

func largeObjects(tx pgx.Tx) func() pgx.LargeObjects { _ = tx.LargeObjects(); return tx.LargeObjects }
