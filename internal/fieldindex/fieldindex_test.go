package fieldindex_test

import (
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
)

// The strings are the values of record_index_fields' value_type and state
// CHECKs (ADR-006 D2); TestP701RegistryChecksMatchFieldindex in
// internal/db/dbtest pins them against the migrated catalog.
func TestValueTypes(t *testing.T) {
	want := []string{"text", "num", "ts", "bool"}
	var got []string
	for _, v := range fieldindex.ValueTypes() {
		got = append(got, string(v))
	}
	if !slices.Equal(got, want) {
		t.Errorf("ValueTypes() = %v, want %v", got, want)
	}
	for _, c := range []struct {
		v    fieldindex.ValueType
		want string
	}{
		{fieldindex.TypeText, "text"}, {fieldindex.TypeNum, "num"}, {fieldindex.TypeTS, "ts"}, {fieldindex.TypeBool, "bool"},
	} {
		if string(c.v) != c.want {
			t.Errorf("%q, want %q", c.v, c.want)
		}
	}
}

func TestStates(t *testing.T) {
	want := []string{"backfilling", "ready", "removing"}
	var got []string
	for _, s := range fieldindex.States() {
		got = append(got, string(s))
	}
	if !slices.Equal(got, want) {
		t.Errorf("States() = %v, want %v", got, want)
	}
}

func TestSideTable(t *testing.T) {
	for v, want := range map[fieldindex.ValueType]string{
		fieldindex.TypeText: "rfv_text", fieldindex.TypeNum: "rfv_num", fieldindex.TypeTS: "rfv_ts", fieldindex.TypeBool: "rfv_bool",
	} {
		if got, ok := v.SideTable(); !ok || got != want {
			t.Errorf("%s.SideTable() = %q, %v; want %q, true", v, got, ok, want)
		}
	}
	// An unknown type names no table, so a caller cannot build SQL from it.
	for _, v := range []fieldindex.ValueType{"", "numeric", "rfv_text", "text; DROP TABLE records"} {
		if got, ok := v.SideTable(); ok || got != "" {
			t.Errorf("%q.SideTable() = %q, %v; want \"\", false", v, got, ok)
		}
	}
}
