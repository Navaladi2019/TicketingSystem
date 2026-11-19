package fieldindex_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Security condition 1 on TICK-745 (PR #734 review): only internal/fieldindex
// registers and names the field-index workflow types, so no other code (the
// interpreter, a future package) can start one as a child or continue as
// one. The relay may name the coordinator, for its SignalWithStart, and
// nothing else. The Authorizer (temporal-server, its own module) is not
// scanned: it holds the allow-list itself.
//
// The test walks every non-test Go file of the module and fails on a
// field-index workflow type outside those places, as a string literal or as
// the fieldindex constant.
func TestOnlyFieldindexNamesItsWorkflowTypes(t *testing.T) {
	root := filepath.Join("..", "..")
	types := []string{
		`"field_index.coordinator"`, `"field_index.backfill.run"`, `"field_index.remove.run"`,
		"fieldindex.CoordinatorWorkflowType", "fieldindex.BackfillWorkflowType", "fieldindex.RemoveWorkflowType",
	}
	// allowed[file] is what that file outside internal/fieldindex may name.
	allowed := map[string][]string{
		"internal/relay/dispatch.go": {"fieldindex.CoordinatorWorkflowType"},
	}
	skipDirs := map[string]bool{".git": true, "node_modules": true, "docs": true, "scratch": true, "sdk": true,
		"bench": true, "temporal-server": true, "testdata": true}
	var scanned int
	found := map[string]int{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		scanned++
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		inside := strings.HasPrefix(rel, "internal/fieldindex/")
		for _, ty := range types {
			if !strings.Contains(string(b), ty) {
				continue
			}
			found[ty]++
			if inside || contains(allowed[rel], ty) {
				continue
			}
			t.Errorf("%s names %s: only internal/fieldindex may name a field-index workflow type", rel, ty)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The scan must see the real names, or it proves nothing.
	if scanned < 100 || found[`"field_index.coordinator"`] == 0 || found["fieldindex.BackfillWorkflowType"] == 0 ||
		found["fieldindex.CoordinatorWorkflowType"] < 2 {
		t.Fatalf("scanned %d files, found %v: the scan no longer sees the field-index names", scanned, found)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
