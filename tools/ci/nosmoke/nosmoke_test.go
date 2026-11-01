// Package nosmoke fails `task test` if code, a kit or a current doc still
// names a removed M1 smoke workflow type (TICK-374, Security review of PR #337
// and Technical Lead review of PR #340). The D9 allow-list admits only
// nava.interpreter, so anything that still starts a smoke type is denied on
// the stack, and anything that documents one is stale.
//
// The check is scoped, not repo-wide: historical records (acceptance-run
// evidence, ADRs and RFCs, reviews, the delivery plan and decision log) name
// the old types on purpose and must not be rewritten.
package nosmoke

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const repoRoot = "../../.."

// prefix is the removed types' common prefix, split so this file does not
// match itself.
const prefix = "nava" + ".smoke."

// scopePaths lists what is checked, relative to the repository root: every Go file
// anywhere, plus every text file under these paths.
func scopePaths() []string {
	return []string{
		"cmd", "internal", "tools", "scripts", "deploy", "make", "api", "temporal-server", ".github",
		"Makefile", "Taskfile.yml",
		"docs/acceptance/guide", "docs/acceptance/outbox-row-runs-one-workflow.md",
		"docs/architecture", "docs/temporal-server.md", "docs/security/temporal-ci6.md",
	}
}

// exemptFiles may name the old types, each for the reason given.
func exemptFiles() map[string]string {
	return map[string]string{
		// The negative test: the old types must be denied.
		"temporal-server/authz/workload_smoke_removed_test.go": "asserts the removed types are denied",
		// A dated record of the acceptance run before TICK-374.
		"docs/acceptance/guide/acceptance-run.md": "historical acceptance-run record",
	}
}

// binaryExts are skipped.
func binaryExts() map[string]bool {
	return map[string]bool{".zip": true, ".png": true, ".jpg": true, ".gif": true, ".ico": true, ".pdf": true}
}

func TestNoSmokeWorkflowTypeLeft(t *testing.T) {
	exempt, binary := exemptFiles(), binaryExts()
	checked := map[string]bool{}
	var hits []string
	check := func(rel string) {
		if checked[rel] || binary[strings.ToLower(filepath.Ext(rel))] {
			return
		}
		checked[rel] = true
		if _, ok := exempt[rel]; ok {
			return
		}
		f, err := os.Open(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = f.Close() }()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 4<<20)
		for n := 1; sc.Scan(); n++ {
			if strings.Contains(sc.Text(), prefix) {
				hits = append(hits, rel+":"+strconv.Itoa(n))
			}
		}
		if err := sc.Err(); err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
	}
	walk := func(root string, keep func(rel string) bool) {
		err := filepath.WalkDir(filepath.Join(repoRoot, root), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(repoRoot, p)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if keep(rel) {
				check(rel)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	walk(".", func(rel string) bool { return strings.HasSuffix(rel, ".go") })
	for _, s := range scopePaths() {
		if _, err := os.Stat(filepath.Join(repoRoot, s)); err != nil {
			t.Fatalf("scope entry %s: %v", s, err)
		}
		walk(s, func(string) bool { return true })
	}
	// Not vacuous: the exempt negative test is reached and does name them.
	if !checked["temporal-server/authz/workload_smoke_removed_test.go"] {
		t.Fatal("the walk never reached temporal-server/authz/workload_smoke_removed_test.go")
	}
	for _, h := range hits {
		t.Errorf("%s names a removed smoke workflow type (%s*); use nava.interpreter, or add a reasoned exemption", h, prefix)
	}
}
