//go:build integration

package isolation

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// pathResult is one registered path in the report.
type pathResult struct {
	Path   Path
	Status string
	Detail string
}

// TestRegisteredPaths runs every registered path: SQL as its role inside
// tenancy.InTenantTx(tenant A), aimed at tenant B; checks in process;
// DB paths against the suite's database; pending paths are listed, not run.
func TestRegisteredPaths(t *testing.T) {
	pools := map[string]*pgxpool.Pool{}
	for _, p := range Paths() {
		if _, ok := pools[p.Role]; p.SQL != nil && !ok {
			pools[p.Role] = suite.pool(t, p.Role, suite.m.Roles[p.Role].PgBouncerLogin)
		}
	}
	for _, p := range Paths() {
		res := pathResult{Path: p, Status: statusPass}
		t.Run(p.Name, func(t *testing.T) {
			switch {
			case p.Pending != "":
				res.Status, res.Detail = statusPending, "pending "+p.Pending
				t.Log(res.Detail)
			case p.Check != nil:
				if err := p.Check(t.Context()); err != nil {
					res.Status, res.Detail = statusFail, err.Error()
				}
			case p.DB != nil:
				if err := p.DB(t.Context(), cluster{t: t, e: suite}); err != nil {
					res.Status, res.Detail = statusFail, err.Error()
				}
			case p.SQL != nil:
				res.Status, res.Detail = runSQLPath(t, pools[p.Role], p.SQL)
			default:
				res.Status, res.Detail = statusFail, "no runner for this path; T17 adds the HTTP runner"
			}
			if res.Status == statusFail {
				t.Error(res.Detail)
			}
		})
		suite.report.paths = append(suite.report.paths, res)
	}
}

// runSQLPath runs c inside tenancy.InTenantTx(tenant A) and rolls it back.
func runSQLPath(t *testing.T, pool *pgxpool.Pool, c *SQLCall) (status, detail string) {
	t.Helper()
	ctx := t.Context()
	status = statusPass
	err := tenancy.InTenantTx(ctx, pool, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		switch c.Expect {
		case OwnRowsOnly:
			rows, err := tx.Query(ctx, c.Query, c.Args...)
			if err != nil {
				return err
			}
			n := 0
			for rows.Next() {
				v, err := rows.Values()
				if err != nil {
					return err
				}
				n++
				b, ok := v[0].([16]byte)
				if !ok || uuid.UUID(b).String() != TenantA {
					status = statusFail
					detail = fmt.Sprintf("row %d belongs to %v, not tenant A", n, v[0])
				}
			}
			if err := rows.Err(); err != nil {
				return err
			}
			if status == statusPass {
				detail = fmt.Sprintf("%d rows, all tenant A's", n)
			}
		case Denied:
			tag, err := tx.Exec(ctx, c.Query, c.Args...)
			switch {
			case err != nil:
				code := sqlState(err)
				switch {
				case code == "":
					return err
				case code == "42501" || strings.HasPrefix(code, "23"):
					detail = "denied: SQLSTATE " + code
				default:
					status, detail = statusFail, "failed for another reason, so it proves nothing: "+err.Error()
				}
			case tag.RowsAffected() > 0:
				status, detail = statusFail, fmt.Sprintf("reached %d rows", tag.RowsAffected())
			default:
				detail = "0 rows"
			}
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		// OwnRowsOnly: the path itself is broken, which proves nothing.
		return statusFail, "the path failed: " + err.Error()
	}
	return status, detail
}
