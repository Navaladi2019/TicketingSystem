package main

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// TestEntityTypeTakesTheDemoWrites: the seeded type compiles, and each of
// the three writes the demo makes validates against it, as the pipeline will
// check it. A schema change that broke the demo fails here, not on the stack.
func TestEntityTypeTakesTheDemoWrites(t *testing.T) {
	c, err := metadata.Compile(entityDefinition())
	if err != nil {
		t.Fatalf("demo entity type does not compile: %v", err)
	}
	data := map[string]any{}
	for i, w := range demoWrites() {
		maps.Copy(data, w.data) // a create, then merge patches
		after := c.ApplyDefaults(maps.Clone(data))
		if i == 0 {
			after[c.StateProperty()] = c.InitialState()
			data[c.StateProperty()] = c.InitialState()
		}
		if fe := c.Validate(after); fe != nil {
			t.Fatalf("write %s refused: %v", w.suffix, fe)
		}
	}
}

// TestSeedSQL: the seed is idempotent, keeps the schema's exact bytes with
// the hash the table CHECK recomputes, names tenant-a, and seeds the workflow
// definition the start key names (P3-40).
func TestSeedSQL(t *testing.T) {
	s, err := seedSQL()
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(s, "ON CONFLICT DO NOTHING"); n != 3 {
		t.Errorf("%d ON CONFLICT DO NOTHING, want 3 (one per INSERT)", n)
	}
	sum := sha256.Sum256([]byte(entitySchema))
	for _, want := range []string{"$nava$" + entitySchema + "$nava$::json", `'\x` + hex.EncodeToString(sum[:]) + `'::bytea`,
		"'" + tenantA + "'", "'" + entityKey + "'", "'" + entityPrefix + "'", "'initial', NULL",
		"'" + definition + "', 'demo_start', '" + entityDefID + "', 1, NULL"} {
		if !strings.Contains(s, want) {
			t.Errorf("seed SQL lacks %s:\n%s", want, s)
		}
	}
}

// TestDemoRows: the rows go to one interpreter run, the start through
// EnqueueStart (the only path for nava.interpreter) and the signals to it.
func TestDemoRows(t *testing.T) {
	w := demoWrites()
	if len(w) != 3 {
		t.Fatalf("%d writes, want a create and two updates", len(w))
	}
	if st, ok := w[0].target.(outbox.TargetStart); !ok || st.Kind != outbox.InterpreterKind || w[0].kind != outbox.InterpreterKind {
		t.Errorf("first write's row %+v, want a start of %s", w[0], outbox.InterpreterKind)
	}
	for _, s := range w[1:] {
		if sg, ok := s.target.(outbox.TargetSignal); !ok || sg.Kind != workflowType || sg.SignalName != s.kind {
			t.Errorf("row %+v, want a %s signal to the run", s, signalName)
		}
	}
}
