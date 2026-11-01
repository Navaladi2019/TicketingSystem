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

// The incident template's state model, as the pipeline reads it (P3-11):
// exactly the four listed transitions are allowed, no reverse and no
// self-transition.
func TestAllowsTransition(t *testing.T) {
	d, err := metadata.Template("incident")
	if err != nil {
		t.Fatal(err)
	}
	c := mustCompile(t, d)
	allowed := map[metadata.Transition]bool{}
	for _, tr := range d.States.Transitions {
		allowed[tr] = true
	}
	if len(allowed) != 4 {
		t.Fatalf("incident template lists %d transitions, want 4", len(allowed))
	}
	states := c.States()
	if len(states) != 4 {
		t.Fatalf("States() = %v", states)
	}
	for _, from := range states {
		for _, to := range states {
			if got := c.AllowsTransition(from, to); got != allowed[metadata.Transition{From: from, To: to}] {
				t.Errorf("AllowsTransition(%q, %q) = %v", from, to, got)
			}
		}
	}
	states[0] = "changed"
	if c.States()[0] == "changed" {
		t.Fatal("States returned the compiled slice itself")
	}
}
