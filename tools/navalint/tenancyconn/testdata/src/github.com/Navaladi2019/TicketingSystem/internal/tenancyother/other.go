package tenancyother

import pgx "github.com/jackc/pgx/v5"

func conn(tx pgx.Tx) *pgx.Conn {
	return tx.Conn() // want `\.Conn\(\) on a pgx\.Tx outside internal/tenancy`
}
