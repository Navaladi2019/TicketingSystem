package migrate

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
)

// Two files with one version: goose refuses the set before it touches the
// database, so the run fails with nothing applied. The zero Options take the
// production lock settings. The database is never dialled.
func TestUpRefusesDuplicateVersions(t *testing.T) {
	t.Parallel()
	cfg, err := pgx.ParseConfig("postgres://nava_migrator@127.0.0.1:1/nava?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{
		"00001_a.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n")},
		"00001_b.sql": {Data: []byte("-- +goose Up\nSELECT 2;\n")},
	}
	applied, err := Up(context.Background(), cfg, fsys, slog.New(slog.DiscardHandler), Options{})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("Up = %v, %v; want the duplicate version refused", applied, err)
	}
	if applied != nil {
		t.Fatalf("applied = %v with an error, want nil", applied)
	}
}
