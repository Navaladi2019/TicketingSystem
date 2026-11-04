// Package diffreaders is the permanent red fixture of the TICK-541 diff
// reader test (internal/audit/read/diffreaders_test.go). Every line marked
// `// want: <what>...` plants a read of audit_log.diff or diff_salt that the
// test must report on that line; every unmarked statement is a look-alike it
// must not report. It lives under testdata, so it is never built.
package diffreaders

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
)

const (
	logTable      = "audit_log"
	logTableAlias = audited.LogTable
	cols          = "id, diff, diff_salt"
)

const (
	plainSQL   = `SELECT id, diff FROM audit_log WHERE tenant_id = $1`                        // want: diff
	saltSQL    = "select DIFF_SALT\n\tfrom public.\"audit_log\" where id = $1"                // want: diff_salt
	bothSQL    = `SELECT diff, diff_salt FROM audit_log`                                      // want: diff diff_salt
	joinSQL    = `SELECT r.id, a.diff FROM records r JOIN audit_log a ON a.subject_id = r.id` // want: diff
	starSQL    = `SELECT * FROM audit_log WHERE id = $1`                                      // want: star
	aliasStar  = `SELECT a.* FROM audit_log a`                                                // want: star
	rowSQL     = `SELECT row_to_json(a) FROM audit_log a`                                     // want: row
	partSQL    = `SELECT diff FROM ONLY audit_log_default`                                    // want: diff
	returnSQL  = `UPDATE audit_log SET erased_at = now() WHERE id = $1 RETURNING diff_salt`   // want: diff_salt
	fromConsts = "SELECT " + cols + " FROM " + logTable                                       // want: diff diff_salt

	// The whole row: aggregates, a cast, a bare select-list item, a bare COPY.
	jsonAggSQL  = `SELECT json_agg(a) FROM audit_log a WHERE a.tenant_id = $1` // want: row
	jsonbAggSQL = `SELECT jsonb_agg(l ORDER BY l.seq) FROM audit_log AS l`     // want: row
	arrayAggSQL = `SELECT array_agg(audit_log) FROM audit_log`                 // want: row
	castSQL     = `SELECT a::text FROM audit_log a WHERE a.id = $1`            // want: row
	wholeRowSQL = `SELECT audit_log FROM audit_log WHERE id = $1`              // want: row
	aliasRowSQL = `SELECT a.id, a FROM audit_log a`                            // want: row
	copyAllSQL  = `COPY audit_log TO STDOUT`                                   // want: copy
	copyColsSQL = `COPY public.audit_log (id, diff) TO STDOUT (FORMAT csv)`    // want: diff

	// Security findings on PR #443: another package's constant, directly
	// and through a local constant (1), COPY of the protected columns (3),
	// TABLE (4), a bare alias as the row (5). a::text (6) is castSQL above.
	pkgConstSQL  = "SELECT id, diff FROM " + audited.LogTable         // want: diff
	viaChainSQL  = "SELECT diff_salt FROM " + logTableAlias           // want: diff_salt
	copyBothSQL  = `COPY audit_log (diff, diff_salt) TO STDOUT`       // want: diff diff_salt
	tableSQL     = `TABLE audit_log`                                  // want: table
	copyTableSQL = `COPY (TABLE public.audit_log) TO STDOUT`          // want: table
	bareAliasSQL = `SELECT a FROM audit_log a WHERE a.tenant_id = $1` // want: row

	// Look-alikes: DDL that names the table, other columns, other tables, a pure insert, comments,
	// aggregates and casts of a column, COPY in, COPY out of safe columns.
	arrayIDSQL  = `SELECT array_agg(id) FROM audit_log`
	colCastSQL  = `SELECT a.id::text, tenant_id::text FROM audit_log a`
	copyInSQL   = `COPY audit_log (id) FROM STDIN`
	alterSQL    = `ALTER TABLE audit_log ADD COLUMN note text`
	lockSQL     = `LOCK TABLE audit_log IN SHARE MODE`
	copySafeSQL = `COPY audit_log (id, diff_sha256) TO STDOUT`
	hashSQL     = `SELECT diff_sha256, diff_truncated, diff_redacted FROM audit_log`
	countSQL    = `SELECT count(*) FROM audit_log WHERE tenant_id = $1`
	v0SQL       = `SELECT diff FROM audit_log_v0`
	otherSQL    = `SELECT diff FROM record_diffs`
	insertSQL   = `INSERT INTO audit_log (tenant_id, diff, diff_salt) VALUES ($1, $2::json, $3)`
	commentSQL  = "-- SELECT diff FROM audit_log\nSELECT 1"
	blockSQL    = "SELECT 1 /* diff FROM audit_log */"
)

func plant(ctx context.Context, tx pgx.Tx, where string) error {
	// A fmt.Sprintf with the table in the format.
	q := fmt.Sprintf("SELECT diff FROM audit_log WHERE %s", where) // want: diff
	// And with the table as an argument, from a constant.
	r := fmt.Sprintf("SELECT %s FROM %s", cols, logTable) // want: diff diff_salt
	// A concatenation with an unknown middle.
	s := "SELECT diff_salt, " + where + " FROM audit_log" // want: diff_salt
	// Another package's constant as a Sprintf argument.
	u := fmt.Sprintf("SELECT id, diff FROM %s WHERE %s", audited.LogTable, where) // want: diff
	_, err := tx.Exec(ctx, q+r+s+u)
	return err
}
