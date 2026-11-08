package db

import (
	"bytes"
	"io/fs"
	"reflect"
	"sort"
	"testing"
	"testing/fstest"
)

// TICK-547: a Windows checkout with core.autocrlf=true wrote the migrations
// with CRLF line endings, and every check that matches a statement end with
// `(?m);$` stopped matching. .gitattributes pins *.sql to LF; this test fails
// any checkout (or new migration) where that pin does not hold.

func TestMigrationsHaveLFLineEndings(t *testing.T) {
	bad, err := filesWithCR(Migrations())
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) > 0 {
		t.Fatalf("migrations contain carriage returns (CRLF line endings); .gitattributes must pin *.sql to eol=lf, then `git rm --cached -r -q db/migrations && git checkout HEAD -- db/migrations`: %v", bad)
	}
}

func TestFilesWithCRFindsCRLFFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"00001_lf.sql":   {Data: []byte("-- +goose Up\nSELECT 1;\n")},
		"00002_crlf.sql": {Data: []byte("-- +goose Up\r\nSELECT 1;\r\n")},
		"00003_cr.sql":   {Data: []byte("SELECT 1;\r")},
		"sub/x.sql":      {Data: []byte("SELECT 1;\r\n")},
	}
	got, err := filesWithCR(fsys)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"00002_crlf.sql", "00003_cr.sql", "sub/x.sql"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filesWithCR = %v, want %v", got, want)
	}
}

// filesWithCR returns, sorted, every regular file in fsys containing a '\r'.
func filesWithCR(fsys fs.FS) ([]string, error) {
	var bad []string
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		if bytes.IndexByte(b, '\r') >= 0 {
			bad = append(bad, path)
		}
		return nil
	})
	sort.Strings(bad)
	return bad, err
}
