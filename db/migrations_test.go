package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/pressly/goose/v3"
)

// TICK-290 (docs/delivery/merge-safety.md item 3): two branches that each
// took the next sequential version merged cleanly in git, and goose then
// refused the directory in every migrating suite. These checks run in
// `task test`, with no database: migration names, versions, and goose's own
// parse of every file.

// migrationName is NNNNN_<name>.sql (00001 to 00007 and the numbers TICK-284
// and TICK-285 assign) or YYYYMMDDHHMMSS_<name>.sql (every later migration;
// `goose create <name> sql`).
var migrationName = regexp.MustCompile(`^([0-9]{5}|[0-9]{14})_[a-z0-9_]+\.sql$`)

const timestampLayout = "20060102150405"

func TestMigrationVersions(t *testing.T) {
	if err := checkMigrationVersions(Migrations()); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationsParseWithGoose(t *testing.T) {
	if err := gooseParse(Migrations()); err != nil {
		t.Fatal(err)
	}
}

// checkMigrationVersions reports every naming and versioning problem in fsys
// at once, each naming the file or files involved.
func checkMigrationVersions(fsys fs.FS) error {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("no migrations found")
	}
	var problems []string
	byVersion := map[int64]string{}
	var sequential []int64
	var maxSequential int64
	type stamped struct {
		name    string
		version int64
	}
	var timestamps []stamped
	for _, name := range names {
		m := migrationName.FindStringSubmatch(name)
		if m == nil {
			problems = append(problems, fmt.Sprintf("%s: name must be NNNNN_<name>.sql or YYYYMMDDHHMMSS_<name>.sql "+
				"(lower-case name; new migrations: goose create <name> sql)", name))
			continue
		}
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: version %q: %v", name, m[1], err))
			continue
		}
		if prev, ok := byVersion[v]; ok {
			problems = append(problems, fmt.Sprintf("duplicate migration version %s: %s and %s", m[1], prev, name))
			continue
		}
		byVersion[v] = name
		if len(m[1]) == 5 {
			sequential = append(sequential, v)
			maxSequential = max(maxSequential, v)
			continue
		}
		if _, err := time.Parse(timestampLayout, m[1]); err != nil {
			problems = append(problems, fmt.Sprintf("%s: version %s is not a valid UTC timestamp YYYYMMDDHHMMSS: %v", name, m[1], err))
			continue
		}
		timestamps = append(timestamps, stamped{name, v})
	}
	sort.Slice(sequential, func(i, j int) bool { return sequential[i] < sequential[j] })
	for i, v := range sequential {
		if want := int64(i + 1); v != want {
			problems = append(problems, fmt.Sprintf("sequential versions are not contiguous from 00001: %s has version %05d, want %05d "+
				"(a gap or a renumbering; new migrations use a timestamp version)", byVersion[v], v, want))
			break
		}
	}
	for _, s := range timestamps {
		if s.version <= maxSequential {
			problems = append(problems, fmt.Sprintf("%s: timestamp version is not later than the highest sequential version %05d", s.name, maxSequential))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("db/migrations: %d problems:\n%s", len(problems), strings.Join(problems, "\n"))
	}
	return nil
}

// gooseParse loads fsys with goose's provider, which refuses a duplicate or
// unparsable version, then runs every migration up against a driver that
// accepts any statement: goose parses each file before it runs any, so a
// missing "-- +goose Up", an unterminated StatementBegin or a bad annotation
// fails here exactly as it would in the migrate job.
func gooseParse(fsys fs.FS) error {
	sqlDB := sql.OpenDB(nopConnector{})
	defer func() { _ = sqlDB.Close() }()
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fsys,
		goose.WithDisableVersioning(true),
		goose.WithDisableGlobalRegistry(true))
	if err != nil {
		return fmt.Errorf("goose: %w", err)
	}
	if _, err := p.Up(context.Background()); err != nil {
		return fmt.Errorf("goose: %w", err)
	}
	return nil
}

// nopConnector is a database/sql driver whose every statement succeeds and
// does nothing.
type nopConnector struct{}

func (nopConnector) Connect(context.Context) (driver.Conn, error) { return nopConn{}, nil }
func (nopConnector) Driver() driver.Driver                        { return nopDriver{} }

type nopDriver struct{}

func (nopDriver) Open(string) (driver.Conn, error) { return nopConn{}, nil }

type nopConn struct{}

func (nopConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("nopConn: Prepare not supported")
}
func (nopConn) Close() error              { return nil }
func (nopConn) Begin() (driver.Tx, error) { return nopTx{}, nil }
func (nopConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

type nopTx struct{}

func (nopTx) Commit() error   { return nil }
func (nopTx) Rollback() error { return nil }

func TestCheckMigrationVersionsRejects(t *testing.T) {
	up := []byte("-- +goose Up\nSELECT 1;\n")
	base := func() fstest.MapFS {
		return fstest.MapFS{
			"00001_a.sql": {Data: up},
			"00002_b.sql": {Data: up},
		}
	}
	cases := []struct {
		name  string
		add   map[string]string
		drop  string
		wants []string // substrings of the error; nil means no error
	}{
		{name: "sequential and timestamp", add: map[string]string{"20261028000000_x.sql": ""}},
		{name: "duplicate sequential", add: map[string]string{"00002_dup.sql": ""},
			wants: []string{"duplicate migration version 00002", "00002_b.sql", "00002_dup.sql"}},
		{name: "duplicate timestamp", add: map[string]string{"20261028000000_x.sql": "", "20261028000000_y.sql": ""},
			wants: []string{"20261028000000_x.sql and 20261028000000_y.sql"}},
		{name: "gap", add: map[string]string{"00004_d.sql": ""}, wants: []string{"00004_d.sql has version 00004, want 00003"}},
		{name: "not from 00001", drop: "00001_a.sql", wants: []string{"00002_b.sql has version 00002, want 00001"}},
		{name: "six digits", add: map[string]string{"000003_c.sql": ""}, wants: []string{"000003_c.sql: name must be"}},
		{name: "no name", add: map[string]string{"00003.sql": ""}, wants: []string{"00003.sql: name must be"}},
		{name: "upper case", add: map[string]string{"00003_Add.sql": ""}, wants: []string{"00003_Add.sql: name must be"}},
		{name: "bad date", add: map[string]string{"20261328000000_x.sql": ""}, wants: []string{"20261328000000_x.sql: version 20261328000000 is not a valid UTC timestamp"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fsys := base()
			delete(fsys, c.drop)
			for n, body := range c.add {
				if body == "" {
					body = string(up)
				}
				fsys[n] = &fstest.MapFile{Data: []byte(body)}
			}
			err := checkMigrationVersions(fsys)
			if c.wants == nil {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want an error containing %q, got none", c.wants)
			}
			for _, w := range c.wants {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
		})
	}
}

func TestGooseParseRejects(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string // substring of the error; empty means no error
	}{
		{name: "valid", body: "-- +goose Up\nCREATE TABLE x (id int);\n\n-- +goose Down\nDROP TABLE x;\n"},
		{name: "no transaction", body: "-- +goose NO TRANSACTION\n-- +goose Up\nCREATE INDEX CONCURRENTLY IF NOT EXISTS i ON x (id);\n"},
		{name: "missing Up", body: "CREATE TABLE x (id int);\n", want: "failed to parse 20261028000000_x.sql"},
		{name: "comments only", body: "-- nothing yet\n", want: "must start with '-- +goose Up' annotation"},
		{name: "unterminated StatementBegin", body: "-- +goose Up\n-- +goose StatementBegin\nSELECT 1;\n", want: "missing '-- +goose StatementEnd'"},
		{name: "unknown annotation", body: "-- +goose Upp\nSELECT 1;\n", want: "20261028000000_x.sql"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"00001_a.sql":          {Data: []byte("-- +goose Up\nSELECT 1;\n")},
				"20261028000000_x.sql": {Data: []byte(c.body)},
			}
			err := gooseParse(fsys)
			if c.want == "" {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error containing %q, got %v", c.want, err)
			}
		})
	}
	t.Run("duplicate version", func(t *testing.T) {
		fsys := fstest.MapFS{
			"00001_a.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n")},
			"00001_b.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n")},
		}
		if err := gooseParse(fsys); err == nil || !strings.Contains(err.Error(), "duplicate migration version 1") {
			t.Fatalf("want goose's duplicate version error, got %v", err)
		}
	})
}
