package isolation

import (
	"strings"
	"testing"
)

// TestComposeDefaults (TICK-552, TICK-436 review finding 2): the compose
// NAVA_DATABASE_URL resolver returns an error, never a panic, on a bad
// variable, so one malformed compose.yaml fails the CI-A10 path only.
func TestComposeDefaults(t *testing.T) {
	for _, c := range []struct {
		name, in, want, err string
	}{
		{"no variable", "postgres://nava_app@pgbouncer:6432/nava", "postgres://nava_app@pgbouncer:6432/nava", ""},
		{"defaults", "postgres://${U:-nava_app}:${P:-x}@pgbouncer:6432/nava", "postgres://nava_app:x@pgbouncer:6432/nava", ""},
		{"empty default", "postgres://nava_app:${P:-}@pgbouncer/nava", "postgres://nava_app:@pgbouncer/nava", ""},
		{"unterminated", "postgres://${U:-nava_app@pgbouncer/nava", "", "unterminated ${"},
		{"unterminated at the end", "postgres://nava_app@pgbouncer/${", "", "unterminated ${"},
		{"unterminated after a resolved one", "postgres://${U:-nava_app}@${H:-pgbouncer", "", "unterminated ${"},
		{"no default", "postgres://${U}@pgbouncer/nava", "", "no default"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := func() (got string, err error) {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("composeDefaults(%q) panicked: %v", c.in, r)
					}
				}()
				return composeDefaults("nava-core", c.in)
			}()
			switch {
			case c.err == "" && (err != nil || got != c.want):
				t.Fatalf("composeDefaults(%q) = %q, %v; want %q", c.in, got, err, c.want)
			case c.err != "" && (err == nil || !strings.Contains(err.Error(), c.err)):
				t.Fatalf("composeDefaults(%q) = %q, %v; want an error naming %q", c.in, got, err, c.err)
			}
		})
	}
}
