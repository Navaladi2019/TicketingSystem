package read

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

// fakeTx answers the one query DefinitionMasker makes with canned schemas.
type fakeTx struct {
	pgx.Tx
	schemas []string
}

func (f fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &fakeRows{schemas: f.schemas, i: -1}, nil
}

type fakeRows struct {
	pgx.Rows
	schemas []string
	i       int
}

func (r *fakeRows) Next() bool                    { r.i++; return r.i < len(r.schemas) }
func (r *fakeRows) Close()                        {}
func (r *fakeRows) Err() error                    { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *fakeRows) Scan(dest ...any) error {
	*(dest[0].(*[]byte)) = []byte(r.schemas[r.i])
	return nil
}

// A definition none of whose version rows is visible fails closed (ADR-004-A1 A1-4).
func TestDefinitionMaskerFailsClosedWithoutVersions(t *testing.T) {
	def := uuid.New()
	paths, err := DefinitionMasker{}.MaskedPaths(t.Context(), fakeTx{}, Subject{Domain: "record", SubjectType: "record", EntityDefinitionID: &def})
	if !errors.Is(err, ErrNoDefinitionVersions) || paths != nil {
		t.Fatalf("MaskedPaths = %v, %v; want nil, ErrNoDefinitionVersions", paths, err)
	}
	var ne *NoDefinitionVersionsError
	if !errors.As(err, &ne) || ne.DefinitionID != def {
		t.Errorf("error = %v; want it to carry definition %s", err, def)
	}
}

func TestMaskReturnsNoEntryWhenDefinitionHasNoVersions(t *testing.T) {
	def := uuid.New()
	r := &Reader{Masker: DefinitionMasker{}}
	entries := []Entry{{Domain: "record", SubjectType: "record", EntityDefinitionID: &def,
		Diff: []byte(`[{"op":"add","path":"/token","value":"hunter2"}]`), DiffSalt: []byte{1, 2, 3}}}
	err := r.mask(t.Context(), fakeTx{}, entries)
	if !errors.Is(err, ErrNoDefinitionVersions) {
		t.Fatalf("mask err = %v; want ErrNoDefinitionVersions", err)
	}
}

func TestDefinitionMaskerNilDefinitionHasNoPaths(t *testing.T) {
	paths, err := DefinitionMasker{}.MaskedPaths(t.Context(), fakeTx{}, Subject{Domain: "config", SubjectType: "tenant"})
	if err != nil || len(paths) != 0 {
		t.Fatalf("MaskedPaths = %v, %v; want none, nil", paths, err)
	}
}

func TestDefinitionMaskerWithVersionsMasksAsBefore(t *testing.T) {
	def := uuid.New()
	tx := fakeTx{schemas: []string{
		`{"properties":{"token":{"x-nava-secret":true},"n":{"type":"integer"}}}`,
		`{"properties":{"pw":{"writeOnly":true},"token":{"x-nava-secret":true}}}`,
	}}
	paths, err := DefinitionMasker{}.MaskedPaths(t.Context(), tx, Subject{EntityDefinitionID: &def})
	if err != nil {
		t.Fatal(err)
	}
	want := []patchwalk.Pointer{"/pw", "/token"}
	if len(paths) != 2 || paths[0] != want[0] || paths[1] != want[1] {
		t.Errorf("paths = %v; want %v", paths, want)
	}
}
