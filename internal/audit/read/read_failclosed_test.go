package read

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Security review of PR #457 (ADR-004-A1 A1-4): when the masker fails, the
// Reader's public reads return the error and no entry. Before the fix, List
// and RecordEntries returned `entries, r.mask(...)`: the page came back next
// to the error, with the entries before the failing one masked and the
// failing one and every later one still holding the raw diff and the salt.
// No handler reads entries on error today. This test keeps it that way for
// every future caller (exports, jobs, webhooks).

// auditRow is one canned audit_log row. Only the columns mask reads are set.
type auditRow struct {
	def  *uuid.UUID
	diff string
	salt []byte
}

// auditTx answers Reader's queries: the audit_log reads with rows, the
// RecordEntries count with len(rows), and the masker's version query with
// schemas only for the definitions in versions.
type auditTx struct {
	pgx.Tx
	rows     []auditRow
	versions map[uuid.UUID]string
}

func (f auditTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	if strings.Contains(sql, "entity_definition_versions") {
		var schemas []string
		if s, ok := f.versions[args[0].(uuid.UUID)]; ok {
			schemas = []string{s}
		}
		return &fakeRows{schemas: schemas, i: -1}, nil
	}
	return &auditRows{rows: f.rows, i: -1}, nil
}

func (f auditTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return countRow(len(f.rows))
}

type countRow int64

func (c countRow) Scan(dest ...any) error { *(dest[0].(*int64)) = int64(c); return nil }

type auditRows struct {
	pgx.Rows
	rows []auditRow
	i    int
}

func (r *auditRows) Next() bool                    { r.i++; return r.i < len(r.rows) }
func (r *auditRows) Close()                        {}
func (r *auditRows) Err() error                    { return nil }
func (r *auditRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

// Scan fills the columns of read.go's column list that mask uses: domain (2),
// subject_type (4), entity_definition_id (6), diff (24), diff_salt (25).
func (r *auditRows) Scan(dest ...any) error {
	row := r.rows[r.i]
	*(dest[2].(*string)) = "record"
	*(dest[4].(*string)) = "record"
	*(dest[6].(**uuid.UUID)) = row.def
	d := row.diff
	*(dest[24].(**string)) = &d
	*(dest[25].(*[]byte)) = row.salt
	return nil
}

func TestReaderReturnsNoEntryWhenMaskerFails(t *testing.T) {
	known, unknown := uuid.New(), uuid.New()
	tx := auditTx{
		rows: []auditRow{
			{def: &known, diff: `[{"op":"add","path":"/token","value":"masked-one"}]`, salt: []byte{1}},
			{def: &unknown, diff: `[{"op":"add","path":"/token","value":"hunter2"}]`, salt: []byte{2}},
		},
		versions: map[uuid.UUID]string{known: `{"properties":{"token":{"x-nava-secret":true}}}`},
	}
	r := &Reader{Masker: DefinitionMasker{}}
	tenant := uuid.New()

	for name, read := range map[string]func() ([]Entry, error){
		"List":          func() ([]Entry, error) { return r.List(t.Context(), tx, tenant, Filter{}, nil, 10) },
		"RecordEntries": func() ([]Entry, error) { return r.RecordEntries(t.Context(), tx, tenant, uuid.New()) },
	} {
		t.Run(name, func(t *testing.T) {
			entries, err := read()
			if !errors.Is(err, ErrNoDefinitionVersions) {
				t.Fatalf("err = %v; want ErrNoDefinitionVersions", err)
			}
			if entries != nil {
				t.Errorf("%s returned %d entries with the masker error; want none (entry 2 raw diff %s, salt %v)",
					name, len(entries), entries[len(entries)-1].Diff, entries[len(entries)-1].DiffSalt)
			}
		})
	}
}
