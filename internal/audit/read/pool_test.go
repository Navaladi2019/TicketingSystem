package read

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ADR-004-A1 A1-5: the reader's pool holds at most five connections, as
// PgBouncer's pool_size=5 for the role (A1-8). OpenPool does not connect.
func TestOpenPoolMaxConns(t *testing.T) {
	p, err := OpenPool(t.Context(), "postgres://nava_audit_reader:pw@127.0.0.1:1/nava?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if got := p.pool.Config().MaxConns; got != 5 || PoolMaxConns != 5 {
		t.Errorf("MaxConns = %d (PoolMaxConns %d), want 5", got, PoolMaxConns)
	}
	if got := p.pool.Config().ConnConfig.User; got != "nava_audit_reader" {
		t.Errorf("user = %q, want the DSN's role (no role switch)", got)
	}
}

func TestOpenPoolRejects(t *testing.T) {
	for name, dsn := range map[string]string{
		"empty":  "",
		"no url": "postgres://nava_audit_reader:s3cret@[::1:5432/nava",
	} {
		t.Run(name, func(t *testing.T) {
			p, err := OpenPool(t.Context(), dsn)
			if err == nil {
				p.Close()
				t.Fatal("OpenPool accepted a bad DSN")
			}
			if strings.Contains(err.Error(), "s3cret") {
				t.Errorf("error echoes the password: %v", err)
			}
		})
	}
}

// A Reader with no pool fails closed: no read, no entry.
func TestReaderWithoutPool(t *testing.T) {
	r := &Reader{Masker: DefinitionMasker{}}
	if got, err := r.List(t.Context(), uuid.New(), Filter{}, nil, 10); !errors.Is(err, ErrNoPool) || got != nil {
		t.Errorf("List = %v, %v; want ErrNoPool", got, err)
	}
	if got, err := r.RecordEntries(t.Context(), uuid.New(), uuid.New()); !errors.Is(err, ErrNoPool) || got != nil {
		t.Errorf("RecordEntries = %v, %v; want ErrNoPool", got, err)
	}
}
