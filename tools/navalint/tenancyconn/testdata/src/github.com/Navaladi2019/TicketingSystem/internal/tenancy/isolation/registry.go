package isolation

import pgx "github.com/jackc/pgx/v5"

// The CI-2 harness copies with the connection (COPY ... TO STDOUT).
func copyTo(tx pgx.Tx) *pgx.Conn { return tx.Conn() }
