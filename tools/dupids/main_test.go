package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const (
	todoHead = "# TODO\n\n| ID | Status |\n| --- | --- |\n"
	arHead   = "# AR\n\n| ID | Sev |\n| --- | --- |\n"
)

func repo(t *testing.T, todoRows, arRows string) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "docs/delivery/todo.md", todoHead+todoRows)
	write(t, root, "docs/security/accepted-risks.md", arHead+arRows)
	return root
}

func TestCleanTables(t *testing.T) {
	root := repo(t, "| TODO-001 | Open |\n| TODO-S5-01 | Open |\n| TODO-S7-01 | Open |\n",
		"| AR-001 (R8) | Low |\n| AR-002 | Low |\n| AR-S5-01 | Low |\n")
	got, err := check(root)
	if err != nil || len(got) != 0 {
		t.Fatalf("check = %v, %v; want clean", got, err)
	}
}

func TestDuplicateIDs(t *testing.T) {
	for _, tc := range []struct {
		name, todo, ar, want string
	}{
		{"todo same row edited", "| TODO-001 | Open |\n| TODO-001 | Fixed |\n", "", "todo.md: id TODO-001 appears 2 times (lines 5, 6)"},
		{"two tasks one story", "| TODO-S5-01 | Open |\n| TODO-S5-01 | Open |\n", "", "id TODO-S5-01 appears 2 times"},
		{"ar same row edited", "", "| AR-007 | Low |\n| AR-007 (R1) | Info |\n", "accepted-risks.md: id AR-007 appears 2 times (lines 5, 6)"},
		{"ar story id", "", "| AR-S5-01 | Low |\n| AR-S5-01 | Low |\n", "id AR-S5-01 appears 2 times"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := check(repo(t, tc.todo, tc.ar))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("check = %v; want one problem containing %q", got, tc.want)
			}
		})
	}
}

func TestMalformedIDCellFails(t *testing.T) {
	for _, tc := range []struct {
		name, todo, ar, want string
	}{
		{"ar backticks", "", "| AR-001 | Low |\n| `AR-033` | Low |\n", "accepted-risks.md: line 6: malformed id cell \"`AR-033`\""},
		{"ar bold", "", "| **AR-033** | Low |\n", "malformed id cell \"**AR-033**\""},
		{"todo lowercase", "| todo-001 | Open |\n", "", "todo.md: line 5: malformed id cell"},
		{"todo wrong prefix", "| AR-001 | Open |\n", "", "malformed id cell \"AR-001\""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := check(repo(t, tc.todo, tc.ar))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("check = %v; want one problem containing %q", got, tc.want)
			}
		})
	}
}

func TestOtherTablesAreNotIDTables(t *testing.T) {
	// A table not headed "ID" (a legend, say) may have any first cell.
	root := repo(t, "| TODO-001 | Open |\n\n| Flag | Meaning |\n| --- | --- |\n| High | bad |\n",
		"| AR-001 | Low |\n\n| Flag | Meaning |\n| --- | --- |\n| `x` | y |\n")
	got, err := check(root)
	if err != nil || len(got) != 0 {
		t.Fatalf("check = %v, %v; want clean", got, err)
	}
}

func TestNoIDTableFails(t *testing.T) {
	// A header renamed to "Id" must not turn the check into a silent no-op.
	root := t.TempDir()
	write(t, root, "docs/delivery/todo.md", "| Id | Status |\n| --- | --- |\n| TODO-001 | Open |\n| TODO-001 | Open |\n")
	write(t, root, "docs/security/accepted-risks.md", arHead+"| AR-001 | Low |\n")
	got, err := check(root)
	if err != nil {
		t.Fatal(err)
	}
	want := `todo.md: no table headed "ID"; dupids checks nothing`
	if len(got) != 1 || !strings.Contains(got[0], want) {
		t.Fatalf("check = %v; want one problem containing %q", got, want)
	}
}

func TestSameIDInOtherFileIsFine(t *testing.T) {
	// Prose and other tables may repeat an id; only first-column cells count.
	root := repo(t, "| TODO-001 | See TODO-001 |\nSee TODO-001 again.\n", "| AR-001 | TODO-001 |\n")
	got, err := check(root)
	if err != nil || len(got) != 0 {
		t.Fatalf("check = %v, %v; want clean", got, err)
	}
}

func TestMissingFile(t *testing.T) {
	if _, err := check(t.TempDir()); err == nil {
		t.Fatal("want an error when a table is missing")
	}
}

func TestRealTablesAreClean(t *testing.T) {
	got, err := check(filepath.Join("..", ".."))
	if err != nil || len(got) != 0 {
		t.Fatalf("check = %v, %v; want clean", got, err)
	}
}
