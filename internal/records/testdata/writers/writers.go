// Package writers is the permanent red fixture of the D-3-6 single-writer
// test (internal/records/arch_test.go, TICK-356). Every line marked
// `// want: <op> <table>` plants a record-table write that the test must
// report on that line; every unmarked statement is a look-alike it must not
// report. It lives under testdata, so it is never built.
package writers

import (
	"context"
	"fmt"

	pg "github.com/jackc/pgx/v5"
)

const auditTable = "audit_log"

const (
	updateRecordsSQL = `UPDATE records SET state = $2 WHERE id = $1`           // want: update records
	mixedCaseSQL     = "uPdAtE    public.Records\n\t SET x = 1"                // want: update records
	quotedSQL        = `INSERT INTO "records" (tenant_id, id) VALUES ($1, $2)` // want: insert records
	schemaQuotedSQL  = `DELETE FROM public."record_links" WHERE id = $1`       // want: delete record_links
	truncateSQL      = "TRUNCATE outbox, record_counters"                      // want: truncate record_counters
	commentsSQL      = "insert\ninto\tcomments (id) values ($1)"               // want: insert comments
	copySQL          = "COPY audit_log (id) FROM STDIN"                        // want: copy audit_log

	// Look-alikes: reads, other tables, ON CONFLICT DO UPDATE and SQL comments.
	lockSQL     = `SELECT version FROM records WHERE id = $1 FOR UPDATE`
	outboxSQL   = `INSERT INTO outbox (id) VALUES ($1) ON CONFLICT (id) DO UPDATE SET id = $1`
	archiveSQL  = `INSERT INTO records_archive (id) VALUES ($1)`
	commentSQL  = "-- UPDATE records\nSELECT 1"
	blockSQL    = "SELECT 1 /* DELETE FROM comments */"
	readOnlySQL = `SELECT count(*) FROM comments`
)

func plant(ctx context.Context, tx pg.Tx, cols, vals string, rows pg.CopyFromSource) error {
	// A fmt.Sprintf with the table in the format.
	q := fmt.Sprintf("INSERT INTO audit_log (%s) VALUES (%s)", cols, vals) // want: insert audit_log
	// And with the table as an argument, from a constant.
	r := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", auditTable, cols, vals) // want: insert audit_log
	// A concatenation.
	s := "DELETE FROM " + "records WHERE id = $1" // want: delete records
	if _, err := tx.Exec(ctx, q+r+s); err != nil {
		return err
	}
	// pgx's COPY, through an aliased import.
	_, err := tx.CopyFrom(ctx, pg.Identifier{"records"}, []string{"id"}, rows) // want: identifier records
	if err != nil {
		return err
	}
	_, err = tx.CopyFrom(ctx, pg.Identifier{"public", "comments"}, []string{"id"}, rows) // want: identifier comments
	if err != nil {
		return err
	}
	// Not a record table.
	_, err = tx.CopyFrom(ctx, pg.Identifier{"outbox"}, []string{"id"}, rows)
	return err
}
