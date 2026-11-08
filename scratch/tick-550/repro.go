//go:build ignore

// TICK-550 reproducer: a Parse-time error inside a transaction, then the same
// SQL in the next transaction on the same pgx connection.
//
//	go run scratch/tick-550/repro.go <url> [mode]
//
// mode: cache-statement (pgx default), cache-describe, exec, simple.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func state(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	if err == nil {
		return "ok"
	}
	return "-"
}

func main() {
	ctx := context.Background()
	cfg, err := pgx.ParseConfig(os.Args[1])
	if err != nil {
		panic(err)
	}
	mode := "cache-statement"
	if len(os.Args) > 2 {
		mode = os.Args[2]
	}
	switch mode {
	case "cache-statement":
		cfg.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
	case "cache-describe":
		cfg.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
	case "exec":
		cfg.DefaultQueryExecMode = pgx.QueryExecModeExec
	case "simple":
		cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	}
	c, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer c.Close(ctx)
	fmt.Printf("server pgbouncer.version=%q mode=%s\n", c.PgConn().ParameterStatus("pgbouncer.version"), mode)

	for _, sc := range []struct {
		name, sql string
	}{
		{"42883 in tx", "SELECT no_such_fn_550($1::int)"},
		{"42P01 in tx", "SELECT x FROM no_such_table_550 WHERE x = $1::int"},
	} {
		fmt.Println("==", sc.name)
		for i := 1; i <= 3; i++ {
			tx, err := c.Begin(ctx)
			if err != nil {
				fmt.Printf("attempt %d: begin err=%v\n", i, err)
				return
			}
			_, _ = tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", "t")
			_, err = tx.Exec(ctx, sc.sql, i)
			fmt.Printf("attempt %d: state=%s err=%v\n", i, state(err), err)
			_ = tx.Rollback(ctx)
		}
	}
	fmt.Println("== healthy query after")
	var n int
	err = c.QueryRow(ctx, "SELECT $1::int + 1", 1).Scan(&n)
	fmt.Printf("state=%s n=%d closed=%v\n", state(err), n, c.IsClosed())
}
