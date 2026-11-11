package a

import pgx "github.com/jackc/pgx/v5"

// A _test.go harness may reach the connection.
func harness(tx pgx.Tx) *pgx.Conn { return tx.Conn() }

// And may call LargeObjects, as the tenancy tests that pin its panic do.
func harnessLargeObjects(tx pgx.Tx) func() pgx.LargeObjects {
	_ = tx.LargeObjects()
	return tx.LargeObjects
}
