package db

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// ADR-004-E5 step (i), I5 (TICK-534): the migration that adds groups.live
// rewrites groups under ACCESS EXCLUSIVE, so it sets lock_timeout, adds the
// D11 CHECK and FK NOT VALID, and validates nothing; the VALIDATEs are in a
// file of their own that also sets lock_timeout.
func TestE5GroupsTombstoneMigrationsSplit(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		b, err := fs.ReadFile(Migrations(), name)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	lockTimeout := regexp.MustCompile(`(?m)^SET LOCAL lock_timeout = '\d+s';\r?$`)
	expand := read("20261104013400_groups_tombstone.sql")
	validate := read("20261104013500_groups_tombstone_validate.sql")
	for name, body := range map[string]string{"expand": expand, "validate": validate} {
		if !lockTimeout.MatchString(body) {
			t.Errorf("%s migration does not SET LOCAL lock_timeout", name)
		}
	}
	for _, want := range []string{
		"ADD COLUMN live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED",
		"CHECK (live) NOT VALID",
		"REFERENCES groups (tenant_id, id, live) NOT VALID",
	} {
		if !strings.Contains(expand, want) {
			t.Errorf("expand migration lacks %q", want)
		}
	}
	if strings.Contains(expand, "VALIDATE") {
		t.Error("expand migration validates a constraint; I5 puts VALIDATE in its own migration")
	}
	for _, c := range []string{"group_members_live_check", "group_members_group_live_fkey"} {
		if !strings.Contains(validate, "VALIDATE CONSTRAINT "+c) {
			t.Errorf("validate migration does not validate %s", c)
		}
	}
}
