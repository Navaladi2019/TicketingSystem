package catalog

import (
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// The shipped db/global-tables.yaml parses and is empty in M1 (D-P16).
func TestShippedGlobalTablesEmpty(t *testing.T) {
	g, err := ParseGlobalTables(db.GlobalTables())
	if err != nil {
		t.Fatal(err)
	}
	if n := g.Names(); len(n) != 0 {
		t.Fatalf("db/global-tables.yaml lists %v; it is empty in M1", n)
	}
}

func TestParseGlobalTables(t *testing.T) {
	g, err := ParseGlobalTables([]byte("tables:\n  codes: { reason: shared lookup }\n  ref.countries: { reason: shared lookup }\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := g.Names(), []string{"public.codes", "ref.countries"}; !slices.Equal(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for _, bad := range []string{
		"tables:\n  codes: {}\n",                                    // no reason
		"tables:\n  codes: { reason: \" \" }\n",                     // blank reason
		"tables:\n  Codes: { reason: x }\n",                         // not lower-case
		"tables:\n  a.b.c: { reason: x }\n",                         // three parts
		"tables:\n  codes: { reason: x, owner: y }\n",               // unknown key
		"tables: {}\nextra: 1\n",                                    // unknown top-level key
		"tables:\n  codes: { reason: x }\n  codes: { reason: y }\n", // duplicate key
	} {
		if _, err := ParseGlobalTables([]byte(bad)); err == nil {
			t.Errorf("ParseGlobalTables accepted %q", bad)
		}
	}
}
