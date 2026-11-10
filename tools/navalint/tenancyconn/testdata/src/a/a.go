package a

import (
	"context"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func called(ctx context.Context, tx pgx.Tx) error {
	return tx.Conn().Exec(ctx, "SELECT 1") // want `\.Conn\(\) on a pgx\.Tx outside internal/tenancy`
}

func methodValue(tx pgx.Tx) func() *pgx.Conn {
	return tx.Conn // want `\.Conn\(\) on a pgx\.Tx outside internal/tenancy`
}

func savepoint(ctx context.Context, tx pgx.Tx) error {
	sp, err := tx.Begin(ctx)
	if err != nil {
		return err
	}
	_ = sp.Conn() // want `\.Conn\(\) on a pgx\.Tx outside internal/tenancy`
	return nil
}

func inClosure(run func(func(pgx.Tx) error) error) error {
	return run(func(tx pgx.Tx) error {
		_ = tx.Conn() // want `\.Conn\(\) on a pgx\.Tx outside internal/tenancy`
		return nil
	})
}

// The transaction's own methods are fine.
func own(ctx context.Context, tx pgx.Tx) error { return tx.Exec(ctx, "SELECT 1") }

// A pooled connection's Conn() is not a transaction's.
func pooled(c *pgxpool.Conn) *pgx.Conn { return c.Conn() }

type other struct{ tx pgx.Tx }

func (other) Conn() *pgx.Conn { return nil }

// Nor is any other type's, even one holding a pgx.Tx.
func otherConn(o other) *pgx.Conn { return o.Conn() }
