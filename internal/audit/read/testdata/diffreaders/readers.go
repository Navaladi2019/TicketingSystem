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

	// TICK-551: a comma join (1), TABLE after a CTE and as an INSERT source
	// (2), a parenthesised star (5).
	commaSQL      = `SELECT a.diff FROM records r, audit_log a WHERE a.subject_id = r.id` // want: diff
	commaRowSQL   = `SELECT to_jsonb(a) FROM records r, public.audit_log AS a`            // want: row
	cteTableSQL   = `WITH x AS (SELECT 1) TABLE audit_log`                                // want: table
	insertTblSQL  = `INSERT INTO audit_copy TABLE audit_log`                              // want: table
	insertColsSQL = `INSERT INTO audit_copy (id, diff) TABLE public.audit_log`            // want: table diff
	parenStarSQL  = `SELECT (a).* FROM audit_log a`                                       // want: star

	// Review round 1: TABLE fails closed, so prefixes nobody listed are seen.
	overrideSysSQL = `INSERT INTO audit_copy OVERRIDING SYSTEM VALUE TABLE audit_log`    // want: table
	overrideUsrSQL = `INSERT INTO audit_copy (id) OVERRIDING USER VALUE TABLE audit_log` // want: table
	cursorSQL      = `DECLARE c CURSOR FOR TABLE audit_log`                              // want: table
	cursorHoldSQL  = `DECLARE c CURSOR WITH HOLD FOR TABLE public.audit_log`             // want: table

	// Security A, B, C on PR #458: a parenthesised join, spaces around the
	// schema dot, spaces around the star's dot.
	parenJoinSQL  = `SELECT a.diff FROM records r, (audit_log a CROSS JOIN x)`                  // want: diff
	parenJoinRow  = `SELECT to_jsonb(a) FROM records r JOIN (audit_log a CROSS JOIN x) ON true` // want: row
	parenFromSQL  = `SELECT a.diff_salt FROM (audit_log a CROSS JOIN records r)`                // want: diff_salt
	spacedDotSQL  = `SELECT a.diff FROM public . audit_log a`                                   // want: diff
	spacedDotRow  = `SELECT to_jsonb(a) FROM records r, public . audit_log a`                   // want: row
	spacedStarSQL = `SELECT a . * FROM audit_log a`                                             // want: star
	halfStarSQL   = `SELECT a. * FROM audit_log a`                                              // want: star

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

	// TICK-551 look-alikes: a comma join of another table, TABLE of another
	// table after a CTE or into audit_log, a parenthesised column.
	commaV0SQL    = `SELECT r.diff, r.audit_log_id FROM records r, audit_log_v0 v`
	cteOtherSQL   = `WITH x AS (SELECT 1) TABLE audit_log_v0`
	insertFromSQL = `INSERT INTO audit_log TABLE audit_staging`
	parenColSQL   = `SELECT (a).id, (a).seq FROM audit_log a`
	createAsSQL   = `CREATE TABLE audit_copy (LIKE audit_log)`

	// The allowed forms of TABLE audit_log (DDL and ACL), and a word that
	// ends in "as".
	createSQL    = `CREATE TABLE audit_log (id bigint)`
	dropSQL      = `DROP TABLE audit_log`
	truncSQL     = `TRUNCATE TABLE ONLY audit_log`
	grantSQL     = `GRANT SELECT ON TABLE audit_log TO nava_app`
	revokeSQL    = `REVOKE ALL ON TABLE public.audit_log FROM nava_app`
	commentOnSQL = `COMMENT ON TABLE audit_log IS 'append-only'`
	refSQL       = `CREATE TABLE notes (log_id bigint REFERENCES audit_log)`
	astableSQL   = `SELECT x FROM astable audit_log`

	// Security A, B, C look-alikes: another table in a parenthesised join or
	// behind a spaced schema dot, a spaced column, a product.
	parenV0SQL   = `SELECT a.diff FROM records r, (audit_log_v0 a CROSS JOIN x)`
	subquerySQL  = `SELECT s.diff FROM (SELECT diff FROM record_diffs) s`
	spacedV0SQL  = `SELECT diff FROM public . audit_log_v0`
	spacedColSQL = `SELECT a . id, a . seq * 2 FROM audit_log a`
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
	// A const declared inside the function (TICK-551 item 4), and one built
	// from it.
	const localTable = "audit_log"
	const localSQL = "SELECT diff FROM " + localTable // want: diff
	v := "SELECT id, diff_salt FROM " + localTable    // want: diff_salt
	// Look-alike: a local const naming another table.
	const otherTable = "audit_log_v0"
	w := "SELECT diff FROM " + otherTable
	_, err := tx.Exec(ctx, q+r+s+u+v+w)
	if err != nil {
		return err
	}
	// A function literal sees its own consts and the enclosing ones.
	run := func() error {
		const litCols = "diff, diff_salt"
		_, err := tx.Exec(ctx, "SELECT "+litCols+" FROM "+localTable) // want: diff diff_salt
		return err
	}
	return run()
}
