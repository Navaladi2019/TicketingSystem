package db

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TICK-299 (follow-up to TICK-290, plan TICK-288): a fresh database migrates
// in version order, a deployed one in merge order (out-of-order is allowed).
// A migration a pull request adds must sort after every migration on main,
// so CI applies migrations in the order deployed databases will.
// The rule: re-stamp the branch's migration before merging (README.md).
//
// TICK-309: the reference is main, what has shipped, even for a pull request
// into a story branch. A migration is new when main lacks it, so main's own
// migrations are never new, and a story may renumber its unshipped ones.
//
// BASE names main's commit (origin/main in the ci workflow's unit job, for
// every event). Without it the test skips: a laptop's origin/main may be stale.
func TestNewMigrationsLaterThanBase(t *testing.T) {
	base := os.Getenv("BASE")
	if base == "" {
		t.Skip("BASE not set: set BASE=origin/main (after git fetch origin main) to check new migrations against main")
	}
	// --full-tree: go test runs in db/, and ls-tree otherwise lists only the
	// paths under the working directory, which would be none.
	out, err := exec.CommandContext(t.Context(), "git", "ls-tree", "--full-tree", "--name-only", base+":db/migrations").Output()
	if err != nil {
		var stderr string
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		t.Fatalf("git ls-tree %s:db/migrations: %v %s (fetch main first; the checkout needs its history)", base, err, stderr)
	}
	baseNames := strings.Fields(string(out))
	if !slices.ContainsFunc(baseNames, func(n string) bool { _, ok := migrationVersion(n); return ok }) {
		t.Fatalf("git ls-tree %s:db/migrations listed no migrations (%q); the check would compare against nothing", base, baseNames)
	}
	head, err := fs.Glob(Migrations(), "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkLaterThanBase(base, baseNames, head); err != nil {
		t.Fatal(err)
	}
}

// checkLaterThanBase reports every migration in head that is not in baseNames
// and whose version is not later than the highest version in baseNames.
// Names that are not migrations are ignored; TestMigrationVersions reports
// them.
func checkLaterThanBase(base string, baseNames, head []string) error {
	inBase := map[string]bool{}
	var newest int64
	var newestName string
	for _, n := range baseNames {
		n = path.Base(n)
		inBase[n] = true
		if v, ok := migrationVersion(n); ok && v > newest {
			newest, newestName = v, n
		}
	}
	var problems []string
	for _, n := range head {
		if inBase[n] {
			continue
		}
		v, ok := migrationVersion(n)
		if !ok || v > newest {
			continue
		}
		problems = append(problems, fmt.Sprintf("db/migrations/%s: version %d is not later than %s's newest migration version %d (%s); "+
			"re-stamp it: rename it to a current timestamp, goose create <name> sql (db/migrations/README.md)",
			n, v, base, newest, newestName))
	}
	if len(problems) > 0 {
		return fmt.Errorf("%d new migrations would apply out of order:\n%s", len(problems), strings.Join(problems, "\n"))
	}
	return nil
}

func migrationVersion(name string) (int64, bool) {
	m := migrationName.FindStringSubmatch(name)
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseInt(m[1], 10, 64)
	return v, err == nil
}

func TestCheckLaterThanBase(t *testing.T) {
	base := []string{"00001_a.sql", "00002_b.sql", "20261028100000_c.sql", "README.md"}
	cases := []struct {
		name  string
		head  []string
		wants []string // substrings of the error; nil means no error
	}{
		{name: "nothing new", head: []string{"00001_a.sql", "00002_b.sql", "20261028100000_c.sql"}},
		{name: "later timestamp", head: []string{"00001_a.sql", "20261028100000_c.sql", "20261028100001_d.sql"}},
		{name: "older timestamp", head: []string{"20261028100000_c.sql", "20261027000000_d.sql"},
			wants: []string{"db/migrations/20261027000000_d.sql: version 20261027000000 is not later than origin/main's newest migration version 20261028100000 (20261028100000_c.sql)", "re-stamp"}},
		{name: "same timestamp", head: []string{"20261028100000_c.sql", "20261028100000_d.sql"},
			wants: []string{"20261028100000_d.sql: version 20261028100000 is not later"}},
		{name: "new sequential", head: []string{"00003_x.sql"}, wants: []string{"00003_x.sql: version 3 is not later"}},
		{name: "every offender", head: []string{"20261027000000_d.sql", "20261026000000_e.sql", "20261029000000_f.sql"},
			wants: []string{"2 new migrations", "20261027000000_d.sql", "20261026000000_e.sql"}},
		{name: "not a migration", head: []string{"notes.sql"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkLaterThanBase("origin/main", base, c.head)
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
	t.Run("base without migrations", func(t *testing.T) {
		if err := checkLaterThanBase("origin/main", nil, []string{"00001_a.sql"}); err != nil {
			t.Fatalf("want no error, got %v", err)
		}
	})
}

// TestCheckLaterThanMainForStoryBranch is the TICK-309 case: a pull request
// into a story branch is checked against main. main has shipped 00008 and
// 00009 (S7); the story branch has merged main and renumbered its own
// unshipped 00008 to 00010 into 00010 to 00012 (S5).
func TestCheckLaterThanMainForStoryBranch(t *testing.T) {
	main := []string{"00001_a.sql", "00002_b.sql", "00007_c.sql", "00008_outbox_refs.sql", "00009_outbox_keys.sql", "README.md"}
	shipped := []string{"00001_a.sql", "00002_b.sql", "00007_c.sql", "00008_outbox_refs.sql", "00009_outbox_keys.sql"}
	t.Run("main's migrations are not new", func(t *testing.T) {
		if err := checkLaterThanBase("origin/main", main, shipped); err != nil {
			t.Fatalf("want no error, got %v", err)
		}
	})
	t.Run("renumbered story migrations later than main's newest", func(t *testing.T) {
		head := append(slices.Clone(shipped), "00010_group_a.sql", "00011_group_b.sql", "00012_group_c.sql")
		if err := checkLaterThanBase("origin/main", main, head); err != nil {
			t.Fatalf("want no error, got %v", err)
		}
	})
	t.Run("new migration not later than main's newest", func(t *testing.T) {
		head := append(slices.Clone(shipped), "00009_group_a.sql", "00010_group_b.sql")
		err := checkLaterThanBase("origin/main", main, head)
		if err == nil {
			t.Fatal("want an error for 00009_group_a.sql, got none")
		}
		for _, w := range []string{"1 new migrations", "db/migrations/00009_group_a.sql: version 9 is not later than origin/main's newest migration version 9 (00009_outbox_keys.sql)"} {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("error %q does not contain %q", err, w)
			}
		}
		if strings.Contains(err.Error(), "00010_group_b.sql") {
			t.Errorf("error %q names 00010_group_b.sql, which is later than main's newest", err)
		}
	})
}
