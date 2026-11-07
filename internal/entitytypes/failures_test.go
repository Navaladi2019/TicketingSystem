package entitytypes

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// step is the outcome of one database call, in call order.
type step struct {
	scan    func(dest []any) error // QueryRow
	execErr error                  // Exec
	tag     string                 // Exec's command tag, e.g. "INSERT 0 1"
	query   *fakeRows              // Query
	qErr    error
}

// fakeTx replays steps; a call past the script fails the test.
type fakeTx struct {
	pgx.Tx
	t     *testing.T
	steps []step
}

func (f *fakeTx) next() step {
	f.t.Helper()
	if len(f.steps) == 0 {
		f.t.Fatal("unexpected database call")
	}
	s := f.steps[0]
	f.steps = f.steps[1:]
	return s
}

type fakeRow struct{ scan func([]any) error }

func (r fakeRow) Scan(dest ...any) error { return r.scan(dest) }

func (f *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row { return fakeRow{f.next().scan} }

func (f *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	s := f.next()
	return pgconn.NewCommandTag(s.tag), s.execErr
}

func (f *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	s := f.next()
	if s.qErr != nil {
		return nil, s.qErr
	}
	return s.query, nil
}

type fakeRows struct {
	pgx.Rows
	rows    int
	scanErr error
	err     error
}

func (r *fakeRows) Close() {}
func (r *fakeRows) Next() bool {
	if r.rows == 0 {
		return false
	}
	r.rows--
	return true
}
func (r *fakeRows) Scan(...any) error { return r.scanErr }
func (r *fakeRows) Err() error        { return r.err }

var errDB = errors.New("db down")

func fail(err error) step   { return step{scan: func([]any) error { return err }} }
func execFail(e error) step { return step{execErr: e} }
func found() step           { return step{scan: func([]any) error { return nil }} }

// versionRow fills the destinations of scanVersion from def.
func versionRow(def metadata.Definition, version, current int, badSum bool) step {
	return step{scan: func(d []any) error {
		*d[0].(*string) = def.Key
		*d[2].(*string) = def.NumberPrefix
		*d[3].(*int) = current
		*d[4].(*int) = version
		*d[5].(*string) = string(def.Schema)
		sum := sha256.Sum256(def.Schema)
		if badSum {
			*d[6].(*[]byte) = sum[:5]
		} else {
			*d[6].(*[]byte) = sum[:]
		}
		*d[7].(*map[string]string) = def.FixedColumns
		*d[8].(*metadata.StateModel) = def.States
		return nil
	}}
}

func template(t *testing.T) metadata.Definition {
	t.Helper()
	d, err := metadata.Template(metadata.TemplateKeys()[0])
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestBreakingErrorNamesThePaths(t *testing.T) {
	t.Parallel()
	got := (&BreakingError{Paths: []string{"/a", "/b"}}).Error()
	if got != "entitytypes: breaking change at /a, /b" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestReadsReportDatabaseFailures(t *testing.T) {
	t.Parallel()
	ctx, d := t.Context(), template(t)
	s := Store{}
	tx := func(steps ...step) *fakeTx { return &fakeTx{t: t, steps: steps} }

	if _, err := s.GetVersion(ctx, tx(), "k", 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("version 0: %v", err)
	}
	if _, err := s.GetVersion(ctx, tx(fail(pgx.ErrNoRows)), "k", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("no rows: %v", err)
	}
	if _, err := s.GetCurrent(ctx, tx(fail(errDB)), "k"); !errors.Is(err, errDB) || errors.Is(err, ErrNotFound) {
		t.Errorf("scan failure must be wrapped, not not-found: %v", err)
	}
	if _, err := s.GetCurrent(ctx, tx(versionRow(d, 1, 1, true)), "k"); err == nil || !strings.Contains(err.Error(), "wrong length") {
		t.Errorf("bad checksum: %v", err)
	}
	v, err := s.GetCurrent(ctx, tx(versionRow(d, 2, 2, false)), "k")
	if err != nil || !v.Current || v.Version != 2 {
		t.Errorf("good row: %+v %v", v, err)
	}
	if v, _ := s.GetVersion(ctx, tx(versionRow(d, 1, 2, false)), "k", 1); v == nil || v.Current {
		t.Errorf("an older version is not current: %+v", v)
	}

	// DefinitionReader.
	if _, err := s.Version(ctx, tx(fail(pgx.ErrNoRows)), "k", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("Version not found: %v", err)
	}
	if c, err := s.Version(ctx, tx(versionRow(d, 1, 1, false)), d.Key, 1); err != nil || c == nil {
		t.Errorf("Version: %v", err)
	}
	if _, _, err := s.Current(ctx, tx(fail(pgx.ErrNoRows)), "k"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Current not found: %v", err)
	}
	broken := d
	broken.Schema = []byte(`{"type":"array"}`)
	if _, _, err := s.Current(ctx, tx(versionRow(broken, 1, 1, false)), "k"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("a stored version that does not compile is a plain error: %v", err)
	}
	if c, n, err := s.Current(ctx, tx(versionRow(d, 4, 4, false)), d.Key); err != nil || c == nil || n != 4 {
		t.Errorf("Current: %v %d", err, n)
	}
}

func TestListReportsDatabaseFailures(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	cases := map[string]step{
		"query": {qErr: errDB},
		"scan":  {query: &fakeRows{rows: 1, scanErr: errDB}},
		"rows":  {query: &fakeRows{err: errDB}},
	}
	for name, st := range cases {
		out, err := Store{}.List(ctx, &fakeTx{t: t, steps: []step{st}}, "", 10)
		if !errors.Is(err, errDB) || out != nil {
			t.Errorf("%s: out=%v err=%v", name, out, err)
		}
	}
	out, err := Store{}.List(ctx, &fakeTx{t: t, steps: []step{{query: &fakeRows{rows: 2}}}}, "", 10)
	if err != nil || len(out) != 2 {
		t.Errorf("good list: %v %v", out, err)
	}
}

func TestCreateRefusals(t *testing.T) {
	t.Parallel()
	ctx, d := t.Context(), template(t)
	in := NewInput{Name: "n", PublishedBy: uuid.New(), Definition: d}
	pg := func(constraint string) error { return &pgconn.PgError{Code: "23505", ConstraintName: constraint} }
	tests := []struct {
		name  string
		steps []step
		want  error
		text  string
	}{
		{"key exists", []step{found()}, ErrKeyExists, ""},
		{"key check fails", []step{fail(errDB)}, errDB, "check key"},
		{"prefix taken", []step{fail(pgx.ErrNoRows), found()}, ErrPrefixTaken, ""},
		{"prefix check fails", []step{fail(pgx.ErrNoRows), fail(errDB)}, errDB, "check prefix"},
		{"race on key", []step{fail(pgx.ErrNoRows), fail(pgx.ErrNoRows), execFail(pg(keyConstraint))}, ErrKeyExists, ""},
		{"race on prefix", []step{fail(pgx.ErrNoRows), fail(pgx.ErrNoRows), execFail(pg(prefixConstraint))}, ErrPrefixTaken, ""},
		{"other unique violation", []step{fail(pgx.ErrNoRows), fail(pgx.ErrNoRows), execFail(pg("other"))}, nil, "insert entity type"},
		{"insert definition fails", []step{fail(pgx.ErrNoRows), fail(pgx.ErrNoRows), execFail(errDB)}, errDB, "insert entity type"},
		{"insert version fails", []step{fail(pgx.ErrNoRows), fail(pgx.ErrNoRows), {}, execFail(errDB)}, errDB, "insert version"},
	}
	for _, tt := range tests {
		_, err := Store{}.Create(ctx, &fakeTx{t: t, steps: tt.steps}, uuid.New(), in)
		if err == nil || (tt.want != nil && !errors.Is(err, tt.want)) || !strings.Contains(err.Error(), tt.text) {
			t.Errorf("%s: err = %v", tt.name, err)
		}
	}
}

func TestSeedReportsDatabaseFailures(t *testing.T) {
	t.Parallel()
	ctx, d := t.Context(), template(t)
	in := SeedInput{DefinitionID: uuid.New(), VersionID: uuid.New(), Name: "n", Definition: d}
	for name, steps := range map[string][]step{
		"definition": {execFail(errDB)},
		"version":    {{}, execFail(errDB)},
		// Both rows inserted, then the entry's tenant read fails.
		"audit": {{tag: "INSERT 0 1"}, {tag: "INSERT 0 1"}, fail(errDB)},
	} {
		inserted, err := Store{}.Seed(ctx, &fakeTx{t: t, steps: steps}, uuid.New(), in)
		if !errors.Is(err, errDB) || inserted || !strings.Contains(err.Error(), "seed "+name) {
			t.Errorf("%s: inserted=%v err=%v", name, inserted, err)
		}
	}
}

func TestPublishRefusalsAndFailures(t *testing.T) {
	t.Parallel()
	ctx, d := t.Context(), template(t)
	lock := step{scan: func(dest []any) error {
		*dest[1].(*string) = d.NumberPrefix
		*dest[2].(*int) = 1
		return nil
	}}
	cur := versionRow(d, 1, 1, false)
	in := PublishInput{Key: d.Key, PublishedBy: uuid.New(), Definition: d}

	if _, err := (Store{}).Publish(ctx, &fakeTx{t: t}, uuid.New(), PublishInput{Key: d.Key}); !errors.Is(err, ErrNoPublisher) {
		t.Errorf("no publisher: %v", err)
	}
	broken := d
	broken.Schema = []byte(`{"type":"array"}`)
	invalid := in
	invalid.Definition = broken

	tests := []struct {
		name  string
		in    PublishInput
		steps []step
		want  error
		text  string
	}{
		{"unknown type", in, []step{fail(pgx.ErrNoRows)}, ErrNotFound, ""},
		{"lock fails", in, []step{fail(errDB)}, errDB, "lock definition"},
		{"current version unreadable", in, []step{lock, fail(errDB)}, errDB, "read version"},
		{"stored version does not compile", in, []step{lock, versionRow(broken, 1, 1, false)}, nil, "does not compile"},
		{"new definition invalid", invalid, []step{lock, cur}, nil, ""},
		{"next number fails", in, []step{lock, cur, fail(errDB)}, errDB, "next version"},
		{"insert fails", in, []step{lock, cur, found(), execFail(errDB)}, errDB, "insert version"},
		{"set current fails", in, []step{lock, cur, found(), {}, execFail(errDB)}, errDB, "set current version"},
	}
	for _, tt := range tests {
		_, err := Store{}.Publish(ctx, &fakeTx{t: t, steps: tt.steps}, uuid.New(), tt.in)
		if err == nil || (tt.want != nil && !errors.Is(err, tt.want)) || !strings.Contains(err.Error(), tt.text) {
			t.Errorf("%s: err = %v", tt.name, err)
		}
	}
}

// Not parallel: it swaps the package-level marshalJSON.
func TestEncodeFailuresWriteNothing(t *testing.T) {
	orig := marshalJSON
	marshalJSON = func(any) ([]byte, error) { return nil, errors.New("cannot encode") }
	t.Cleanup(func() { marshalJSON = orig })
	ctx, d := t.Context(), template(t)
	noCalls := func() *fakeTx { return &fakeTx{t: t} } // any database call fails the test

	_, err := Store{}.Create(ctx, &fakeTx{t: t, steps: []step{fail(pgx.ErrNoRows), fail(pgx.ErrNoRows)}}, uuid.New(),
		NewInput{Name: "n", PublishedBy: uuid.New(), Definition: d})
	if err == nil || !strings.Contains(err.Error(), "encode fixed columns") {
		t.Errorf("Create: %v", err)
	}
	if _, err := (Store{}).Seed(ctx, noCalls(), uuid.New(), SeedInput{Definition: d}); err == nil || !strings.Contains(err.Error(), "encode fixed columns") {
		t.Errorf("Seed: %v", err)
	}
	lock := step{scan: func(dest []any) error {
		*dest[1].(*string) = d.NumberPrefix
		*dest[2].(*int) = 1
		return nil
	}}
	_, err = Store{}.Publish(ctx, &fakeTx{t: t, steps: []step{lock, versionRow(d, 1, 1, false)}}, uuid.New(),
		PublishInput{Key: d.Key, PublishedBy: uuid.New(), Definition: d})
	if err == nil || !strings.Contains(err.Error(), "encode fixed columns") {
		t.Errorf("Publish: %v", err)
	}

	// States fail when the fixed columns encode: let the first call through.
	n := 0
	marshalJSON = func(v any) ([]byte, error) {
		if n++; n == 1 {
			return orig(v)
		}
		return nil, errors.New("cannot encode")
	}
	_, err = Store{}.Create(ctx, &fakeTx{t: t, steps: []step{fail(pgx.ErrNoRows), fail(pgx.ErrNoRows)}}, uuid.New(),
		NewInput{Name: "n", PublishedBy: uuid.New(), Definition: d})
	if err == nil || !strings.Contains(err.Error(), "encode states") {
		t.Errorf("Create states: %v", err)
	}
	n = 0
	if _, err := (Store{}).Seed(ctx, noCalls(), uuid.New(), SeedInput{Definition: d}); err == nil || !strings.Contains(err.Error(), "encode states") {
		t.Errorf("Seed states: %v", err)
	}
	n = 0
	_, err = Store{}.Publish(ctx, &fakeTx{t: t, steps: []step{lock, versionRow(d, 1, 1, false)}}, uuid.New(),
		PublishInput{Key: d.Key, PublishedBy: uuid.New(), Definition: d})
	if err == nil || !strings.Contains(err.Error(), "encode states") {
		t.Errorf("Publish states: %v", err)
	}
}

func TestEncodeFixedColumnsNilIsAnEmptyObject(t *testing.T) {
	t.Parallel()
	got, err := encodeFixedColumns(nil)
	if err != nil || string(got) != "{}" {
		t.Fatalf("got %q, %v", got, err)
	}
}
