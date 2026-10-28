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
// BASE names the pull request's base branch (origin/<base_ref> in the ci
// workflow's unit job). Without it the test skips: a laptop's refs may be
// stale. Whatever the base, migrations are measured against main (see
// mainRef).
func TestNewMigrationsLaterThanBase(t *testing.T) {
	base := os.Getenv("BASE")
	if base == "" {
		t.Skip("BASE not set: set BASE=origin/main (after git fetch) to check new migrations against the base branch")
	}
	head, err := fs.Glob(Migrations(), "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkNewMigrations(base, gitMigrations(t), head); err != nil {
		t.Fatal(err)
	}
}

// mainRef is the ref added migrations are measured against, for a pull
// request into main and for one into a story branch alike. Main's migrations
// reach a story branch by merge, so they are not new there; the story's own
// migrations must sort after main's newest, because that is the order they
// apply in once the story merges. Story-branch databases are not deployed, so
// this rule does not protect them (an accepted risk: rebuild them after a
// renumber). Two tasks of one story that take the same version are
// TestMigrationVersions' job.
const mainRef = "origin/main"

// checkNewMigrations reports every migration in head that is not on mainRef
// and is not later than its newest migration. base is the pull request's base
// branch; list returns the migration file names on a ref.
func checkNewMigrations(base string, list func(ref string) ([]string, error), head []string) error {
	names, err := list(mainRef)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(names, func(n string) bool { _, ok := migrationVersion(n); return ok }) {
		return fmt.Errorf("%s:db/migrations lists no migrations (%q); the check would compare against nothing", mainRef, names)
	}
	if err := checkLaterThanBase(mainRef, names, head); err != nil {
		if base != mainRef {
			return fmt.Errorf("pull request into %s, measured against %s (TICK-299): %w", base, mainRef, err)
		}
		return err
	}
	return nil
}

// gitMigrations lists db/migrations on a ref with git ls-tree.
func gitMigrations(t *testing.T) func(ref string) ([]string, error) {
	return func(ref string) ([]string, error) {
		// --full-tree: go test runs in db/, and ls-tree otherwise lists only
		// the paths under the working directory, which would be none.
		out, err := exec.CommandContext(t.Context(), "git", "ls-tree", "--full-tree", "--name-only", ref+":db/migrations").Output()
		if err != nil {
			var stderr string
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				stderr = strings.TrimSpace(string(ee.Stderr))
			}
			return nil, fmt.Errorf("git ls-tree %s:db/migrations: %w %s (fetch %s first; the checkout needs its history)", ref, err, stderr, ref)
		}
		return strings.Fields(string(out)), nil
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

// TestCheckNewMigrationsMeasuresAgainstMain: a pull request into a story
// branch is measured against main, not against the story branch; a pull
// request into main keeps the rule it always had.
func TestCheckNewMigrationsMeasuresAgainstMain(t *testing.T) {
	const story = "origin/TICK-47/story"
	refs := map[string][]string{
		// main has S7's 00008/00009.
		mainRef: {"00001_a.sql", "00007_g.sql", "00008_s7.sql", "00009_s7.sql", "README.md"},
		// The story branch still has its own migrations at 00008-00010.
		story: {"00001_a.sql", "00007_g.sql", "00008_s5.sql", "00009_s5.sql", "00010_s5.sql"},
	}
	list := func(ref string) ([]string, error) {
		names, ok := refs[ref]
		if !ok {
			return nil, fmt.Errorf("unknown ref %s", ref)
		}
		return names, nil
	}
	merged := []string{"00001_a.sql", "00007_g.sql", "00008_s7.sql", "00009_s7.sql", "00010_s5.sql", "00011_s5.sql", "00012_s5.sql"}
	cases := []struct {
		name  string
		base  string
		head  []string
		wants []string // substrings of the error; nil means no error
	}{
		{name: "story base: main's migrations are not new, the story's sort after main's", base: story, head: merged},
		{name: "story base: a version not later than main's newest fails", base: story,
			head:  []string{"00001_a.sql", "00007_g.sql", "00008_s7.sql", "00009_s7.sql", "00009_late.sql"},
			wants: []string{"pull request into " + story + ", measured against origin/main", "00009_late.sql: version 9 is not later than origin/main's newest migration version 9"}},
		{name: "story base: an old story-branch version still fails against main", base: story,
			head:  []string{"00001_a.sql", "00007_g.sql", "00008_s5.sql"},
			wants: []string{"00008_s5.sql: version 8 is not later than origin/main's newest migration version 9"}},
		{name: "main base: a later migration passes", base: mainRef, head: append(slices.Clone(refs[mainRef][:4]), "20261028130000_x.sql")},
		{name: "main base: out of order still fails", base: mainRef,
			head:  []string{"00001_a.sql", "00007_g.sql", "00008_s7.sql", "00009_s7.sql", "00005_old.sql"},
			wants: []string{"00005_old.sql: version 5 is not later than origin/main's newest migration version 9"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkNewMigrations(c.base, list, c.head)
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
	t.Run("main base: an error names main only", func(t *testing.T) {
		err := checkNewMigrations(mainRef, list, []string{"00005_old.sql"})
		if err == nil || strings.Contains(err.Error(), "pull request into") {
			t.Fatalf("want the plain main-base error, got %v", err)
		}
	})
	t.Run("ref without migrations", func(t *testing.T) {
		empty := func(string) ([]string, error) { return []string{"README.md"}, nil }
		if err := checkNewMigrations(story, empty, merged); err == nil || !strings.Contains(err.Error(), "lists no migrations") {
			t.Fatalf("want a lists-no-migrations error, got %v", err)
		}
	})
}
