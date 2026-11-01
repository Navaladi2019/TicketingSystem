package metadata_test

import (
	"maps"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// The record write pipeline reads the mapping, state property, initial state
// and prefix through these; FixedColumns hands out a copy.
func TestCompiledAccessors(t *testing.T) {
	d := baseDefinition(t, baseSchema())
	c := mustCompile(t, d)
	if c.NumberPrefix() != "TKT" || c.StateProperty() != "state" || c.InitialState() != "open" {
		t.Fatalf("prefix %q, state property %q, initial %q", c.NumberPrefix(), c.StateProperty(), c.InitialState())
	}
	got := c.FixedColumns()
	if !maps.Equal(got, d.FixedColumns) {
		t.Fatalf("fixed columns %v, want %v", got, d.FixedColumns)
	}
	got["title"] = metadata.ColumnAssigneeID
	if _, leaked := c.FixedColumns()["title"]; leaked {
		t.Fatal("FixedColumns returned the compiled map itself")
	}
}
