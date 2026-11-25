// Package repairtest runs the operator SQL of
// docs/data/entity-versions-org-unit-id-reserved.md (TICK-1125) from that
// file, the way psql runs a paste: one statement after another, going on
// after an error. Only test code imports this package.
package repairtest

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The info strings of the fenced blocks, after "sql ".
const (
	Check     = "tick-1125-check"
	ARS3001   = "tick-1125-ar-s30-01"
	PasteOne  = "tick-1125-paste-1"
	PasteTwo  = "tick-1125-paste-2"
	dollarTag = "$nava_repair$"
)

var (
	errNoBlock  = errors.New("repairtest: no such fenced block")
	errNoInput  = errors.New("repairtest: not exactly one line sets the input")
	errNotPaste = errors.New("repairtest: statement with no end")
)

// Block returns the body of the one fenced block of markdown whose info
// string is "sql <name>".
func Block(markdown, name string) (string, error) {
	_, rest, ok := strings.Cut(markdown, "```sql "+name+"\n")
	if !ok {
		return "", fmt.Errorf("%w: %s", errNoBlock, name)
	}
	body, _, ok := strings.Cut(rest, "\n```\n")
	if !ok {
		return "", fmt.Errorf("%w: %s has no end", errNoBlock, name)
	}
	return body + "\n", nil
}

// Inputs are the operator's five inputs.
type Inputs struct {
	TenantID, TypeID, NewName, SourceRef, RecordLimit string
}

// WithInputs replaces the five inputs in paste 1 and nothing else.
func WithInputs(sql string, in Inputs) (string, error) {
	for key, value := range map[string]string{
		"tenant_id": in.TenantID, "type_id": in.TypeID, "new_name": in.NewName,
		"source_ref": in.SourceRef, "record_limit": in.RecordLimit,
	} {
		re := regexp.MustCompile(`(?m)^(SELECT set_config\('nava_repair\.` + key + `', ')[^']*(', true\);)$`)
		if len(re.FindAllString(sql, -1)) != 1 {
			return "", fmt.Errorf("%w: %s", errNoInput, key)
		}
		sql = re.ReplaceAllString(sql, "${1}"+strings.ReplaceAll(value, "$", "$$")+"${2}")
	}
	return sql, nil
}

// Statements splits a block: a statement ends at a line that ends in ";",
// outside the DO body.
func Statements(sql string) ([]string, error) {
	var (
		out  []string
		cur  strings.Builder
		body bool
	)
	for line := range strings.Lines(sql) {
		cur.WriteString(line)
		if strings.Count(line, dollarTag)%2 == 1 {
			body = !body
		}
		if !body && strings.HasSuffix(strings.TrimSpace(line), ";") {
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		}
	}
	if strings.TrimSpace(cur.String()) != "" {
		return nil, errNotPaste
	}
	return out, nil
}

// Output is what a paste answered: the first error, and the rows of every
// statement that returned some, as text.
type Output struct {
	Err  error
	Rows [][][]string
}

// Paste runs sql on conn as psql does with ON_ERROR_STOP off. With
// onErrorRollback, each statement in a transaction has its own savepoint, as
// with ON_ERROR_ROLLBACK=on.
func Paste(ctx context.Context, conn *pgx.Conn, sql string, onErrorRollback bool) (Output, error) {
	var out Output
	stmts, err := Statements(sql)
	if err != nil {
		return out, err
	}
	pc := conn.PgConn()
	run := func(q string) ([]*pgconn.Result, error) { return pc.Exec(ctx, q).ReadAll() }
	for _, s := range stmts {
		save := onErrorRollback && pc.TxStatus() == 'T' && !strings.HasPrefix(s, "COMMIT")
		if save {
			if _, err := run("SAVEPOINT psql"); err != nil {
				return out, err
			}
		}
		res, err := run(s)
		if err != nil {
			var pe *pgconn.PgError
			if !errors.As(err, &pe) {
				return out, err
			}
			if out.Err == nil {
				out.Err = err
			}
		}
		if save && pc.TxStatus() != 'I' {
			q := "RELEASE SAVEPOINT psql"
			if err != nil {
				q = "ROLLBACK TO SAVEPOINT psql"
			}
			if _, err := run(q); err != nil {
				return out, err
			}
		}
		for _, r := range res {
			if !r.CommandTag.Select() {
				continue
			}
			rows := make([][]string, len(r.Rows))
			for i, row := range r.Rows {
				for _, v := range row {
					rows[i] = append(rows[i], string(v))
				}
			}
			out.Rows = append(out.Rows, rows)
		}
	}
	return out, nil
}
