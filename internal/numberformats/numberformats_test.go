package numberformats

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// The store's happy paths run against PostgreSQL in internal/server
// (numberformats_integration_test.go). These tests drive each error path of
// the store through a stub transaction: a failing statement must come back as
// an error (wrapping the cause) and the code must not go on to the next one.

var errBoom = errors.New("injected database fault")

// stubTx answers the three calls the store makes. call numbers the calls of
// one kind from 1; a func returning an error fails that call.
type stubTx struct {
	pgx.Tx
	execs, rows, queries int
	execErr              func(n int, sql string) error
	rowErr               func(n int, sql string) error
	queryErr             func(n int) error
	scanFail, endFail    bool
	log                  []string
}

func (s *stubTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	s.execs++
	s.log = append(s.log, "exec")
	if s.execErr != nil {
		return pgconn.CommandTag{}, s.execErr(s.execs, sql)
	}
	return pgconn.CommandTag{}, nil
}

func (s *stubTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	s.rows++
	s.log = append(s.log, "row")
	var err error
	if s.rowErr != nil {
		err = s.rowErr(s.rows, sql)
	}
	return fillRow{err}
}

func (s *stubTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	s.queries++
	if s.queryErr != nil {
		if err := s.queryErr(s.queries); err != nil {
			return nil, err
		}
	}
	return &stubRows{scanFail: s.scanFail, endFail: s.endFail}, nil
}

// fillRow scans plausible values into any destination, or fails.
type fillRow struct{ err error }

func (r fillRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for _, d := range dest {
		switch p := d.(type) {
		case *int:
			*p = 1
		case *string:
			*p = "{}"
		case *[]byte:
			*p = make([]byte, 32)
		case *json.RawMessage:
			*p = json.RawMessage("{}")
		default:
			// uuid.UUID, time.Time and the rest keep their zero value.
			_ = reflect.TypeOf(d)
		}
	}
	return nil
}

type stubRows struct {
	pgx.Rows
	scanFail, endFail bool
	given             bool
}

func (r *stubRows) Next() bool {
	if r.endFail || r.given {
		return false
	}
	r.given = true
	return true
}
func (r *stubRows) Scan(dest ...any) error {
	if r.scanFail {
		return errBoom
	}
	return fillRow{}.Scan(dest...)
}
func (r *stubRows) Err() error {
	if r.endFail {
		return errBoom
	}
	return nil
}
func (r *stubRows) Close() {}

func newInput() NewInput {
	return NewInput{Key: "monthly", Document: json.RawMessage(`{"template":"{prefix}{seq:7}","overflow":"grow"}`),
		Compiled: numbering.Default(), CreatedBy: uuid.New()}
}

func pubInput() PublishInput {
	return PublishInput{ID: uuid.New(), Document: json.RawMessage(`{}`), Compiled: numbering.Default(), CreatedBy: uuid.New(), ExpectedVersion: []int{1}}
}

func TestStoreRefusesAWriteWithoutAPrincipal(t *testing.T) {
	t.Parallel()
	in := newInput()
	in.CreatedBy = uuid.Nil
	tx := &stubTx{}
	if _, err := (Store{}).Create(t.Context(), tx, uuid.New(), in); !errors.Is(err, ErrNoPublisher) {
		t.Errorf("Create = %v, want ErrNoPublisher", err)
	}
	p := pubInput()
	p.CreatedBy = uuid.Nil
	if _, err := (Store{}).Publish(t.Context(), tx, uuid.New(), p); !errors.Is(err, ErrNoPublisher) {
		t.Errorf("Publish = %v, want ErrNoPublisher", err)
	}
	if len(tx.log) != 0 {
		t.Errorf("a refused write reached the database: %v", tx.log)
	}
}

func TestCreateErrorPaths(t *testing.T) {
	t.Parallel()
	dup := &pgconn.PgError{Code: "23505", ConstraintName: keyConstraint}
	other23505 := &pgconn.PgError{Code: "23505", ConstraintName: "something_else"}
	cases := []struct {
		name string
		tx   *stubTx
		is   error
		// stopsAfter is the number of statements that ran before the error.
		stopsAfter int
	}{
		{"duplicate key", &stubTx{execErr: func(int, string) error { return dup }}, ErrKeyExists, 1},
		{"another unique violation", &stubTx{execErr: func(int, string) error { return other23505 }}, other23505, 1},
		{"insert head", &stubTx{execErr: func(int, string) error { return errBoom }}, errBoom, 1},
		{"insert version", &stubTx{execErr: func(n int, _ string) error {
			if n == 2 {
				return errBoom
			}
			return nil
		}}, errBoom, 2},
		{"read the document for the audit diff", &stubTx{rowErr: func(int, string) error { return errBoom }}, errBoom, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := Store{}.Create(t.Context(), c.tx, uuid.New(), newInput())
			if !errors.Is(err, c.is) {
				t.Fatalf("err = %v, want %v", err, c.is)
			}
			if c.tx.execs != c.stopsAfter && c.name != "read the document for the audit diff" {
				t.Errorf("%d inserts ran, want %d", c.tx.execs, c.stopsAfter)
			}
		})
	}
}

func TestPublishErrorPaths(t *testing.T) {
	t.Parallel()
	t.Run("unknown format", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{rowErr: func(int, string) error { return pgx.ErrNoRows }}
		if _, err := (Store{}).Publish(t.Context(), tx, uuid.New(), pubInput()); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
		if tx.execs != 0 {
			t.Error("wrote for an unknown format")
		}
	})
	t.Run("lock fails", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{rowErr: func(int, string) error { return errBoom }}
		if _, err := (Store{}).Publish(t.Context(), tx, uuid.New(), pubInput()); !errors.Is(err, errBoom) || errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want the cause, not ErrNotFound", err)
		}
	})
	t.Run("stale version on the locked row", func(t *testing.T) {
		t.Parallel()
		// The stub's locked row is at version 1; the caller named 5.
		tx := &stubTx{}
		p := pubInput()
		p.ExpectedVersion = []int{5, 6}
		_, err := (Store{}).Publish(t.Context(), tx, uuid.New(), p)
		var c *ConflictError
		if !errors.As(err, &c) || c.Current != 1 || c.Error() == "" {
			t.Fatalf("err = %v, want *ConflictError at version 1", err)
		}
		if tx.execs != 0 {
			t.Error("a stale publish wrote")
		}
	})
	t.Run("insert version fails", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{execErr: func(int, string) error { return errBoom }}
		if _, err := (Store{}).Publish(t.Context(), tx, uuid.New(), pubInput()); !errors.Is(err, errBoom) || tx.execs != 1 {
			t.Errorf("err = %v after %d statements", err, tx.execs)
		}
	})
	t.Run("move the current version fails", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{execErr: func(n int, _ string) error {
			if n == 2 {
				return errBoom
			}
			return nil
		}}
		if _, err := (Store{}).Publish(t.Context(), tx, uuid.New(), pubInput()); !errors.Is(err, errBoom) || tx.execs != 2 {
			t.Errorf("err = %v after %d statements", err, tx.execs)
		}
	})
	t.Run("read the document for the audit diff", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{rowErr: func(n int, _ string) error {
			if n == 2 { // the lock is call 1
				return errBoom
			}
			return nil
		}}
		if _, err := (Store{}).Publish(t.Context(), tx, uuid.New(), pubInput()); !errors.Is(err, errBoom) {
			t.Errorf("err = %v, want the cause", err)
		}
	})
}

func TestReadErrorPaths(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	t.Run("get version: not found", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{rowErr: func(int, string) error { return pgx.ErrNoRows }}
		if _, err := (Store{}).GetVersion(t.Context(), tx, id, 3); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v", err)
		}
		if _, err := (Store{}).Current(t.Context(), tx, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("current err = %v", err)
		}
	})
	t.Run("get version: database error", func(t *testing.T) {
		t.Parallel()
		tx := &stubTx{rowErr: func(int, string) error { return errBoom }}
		if _, err := (Store{}).GetVersion(t.Context(), tx, id, 0); !errors.Is(err, errBoom) || errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("get version: ok", func(t *testing.T) {
		t.Parallel()
		v, err := (Store{}).GetVersion(t.Context(), &stubTx{}, id, 1)
		if err != nil || len(v.CompiledSHA256) != 64 {
			t.Errorf("v = %+v, err = %v", v, err)
		}
	})
	t.Run("current version", func(t *testing.T) {
		t.Parallel()
		if n, err := (Store{}).CurrentVersion(t.Context(), &stubTx{}, id); err != nil || n != 1 {
			t.Errorf("ok: %d, %v", n, err)
		}
		if _, err := (Store{}).CurrentVersion(t.Context(), &stubTx{rowErr: func(int, string) error { return pgx.ErrNoRows }}, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("not found: %v", err)
		}
		if _, err := (Store{}).CurrentVersion(t.Context(), &stubTx{rowErr: func(int, string) error { return errBoom }}, id); !errors.Is(err, errBoom) {
			t.Errorf("fault: %v", err)
		}
	})
	t.Run("list", func(t *testing.T) {
		t.Parallel()
		if got, err := (Store{}).List(t.Context(), &stubTx{}, "", 10); err != nil || len(got) != 1 {
			t.Errorf("ok: %v, %v", got, err)
		}
		for name, tx := range map[string]*stubTx{
			"query": {queryErr: func(int) error { return errBoom }},
			"scan":  {scanFail: true},
			"end":   {endFail: true},
		} {
			if _, err := (Store{}).List(t.Context(), tx, "", 10); name != "end" && !errors.Is(err, errBoom) {
				t.Errorf("%s: err = %v", name, err)
			} else if name == "end" && !errors.Is(err, errBoom) {
				t.Errorf("end: err = %v", err)
			}
		}
	})
}

func TestCatalogProbeError(t *testing.T) {
	t.Parallel()
	open, err := CatalogProbe(t.Context(), &stubTx{rowErr: func(int, string) error { return errBoom }})
	if open || !errors.Is(err, errBoom) {
		t.Errorf("= %v, %v; want closed and the cause", open, err)
	}
}

func TestInsertVersionRefusesACompiledFormWithoutASHA256(t *testing.T) {
	t.Parallel()
	in := newInput()
	in.Compiled.CompiledSHA256 = "not-hex"
	tx := &stubTx{}
	if _, err := (Store{}).Create(t.Context(), tx, uuid.New(), in); err == nil {
		t.Fatal("created a version with no usable hash")
	}
	if tx.execs != 1 { // the head row only; the version insert never ran
		t.Errorf("%d inserts ran, want only the head", tx.execs)
	}
}
