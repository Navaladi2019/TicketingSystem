package def

import (
	"encoding/json"
	"maps"
	"os"
	"testing"

	"github.com/google/uuid"
)

func mustParse(t *testing.T, b []byte) *Definition {
	t.Helper()
	d, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func readIDs(t *testing.T, name string) map[string]uuid.UUID {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".ids.json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]uuid.UUID
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func readDef(t *testing.T, name string) *Definition {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return mustParse(t, b)
}

func TestAssignStepIDsCoversEveryKeyWithV4(t *testing.T) {
	d := mustParse(t, encode(t, base()))
	ids := AssignStepIDs(d, nil)
	if len(ids) != len(d.Steps)+1 {
		t.Fatalf("len = %d, want %d steps + 1 signal", len(ids), len(d.Steps))
	}
	seen := map[uuid.UUID]bool{}
	for i := range d.Steps {
		id, ok := ids[d.Steps[i].Key]
		if !ok {
			t.Fatalf("no id for %s", d.Steps[i].Key)
		}
		seen[id] = true
	}
	sig, ok := ids[SignalKey("approve")]
	if !ok {
		t.Fatal("no id for signal approve")
	}
	seen[sig] = true
	if len(seen) != len(ids) {
		t.Errorf("ids are not distinct: %v", ids)
	}
	for k, id := range ids {
		if id.Version() != 4 || id.Variant() != uuid.RFC4122 {
			t.Errorf("%s: id %s is not a v4 RFC 4122 uuid", k, id)
		}
	}
}

func TestAssignStepIDsStableForUnchangedKey(t *testing.T) {
	d := mustParse(t, encode(t, base()))
	v1 := AssignStepIDs(d, nil)

	// v2 renames "apply" to "apply2" and changes the condition value.
	m := base()
	step(m, 1)["value"] = "critical"
	step(m, 3)["next"] = "apply2"
	step(m, 4)["key"] = "apply2"
	v2 := AssignStepIDs(mustParse(t, encode(t, m)), v1)

	for _, k := range []string{"wait", "check", "route", "approve", "done", SignalKey("approve")} {
		if v2[k] != v1[k] {
			t.Errorf("%s: id changed across versions: %s -> %s", k, v1[k], v2[k])
		}
	}
	if _, ok := v2["apply"]; ok {
		t.Error("removed key kept its id in the new map")
	}
	if id := v2["apply2"]; id == uuid.Nil || id == v1["apply"] {
		t.Errorf("new key apply2 got %s, want a fresh id", id)
	}
}

func TestAssignStepIDsNotDerivedFromKey(t *testing.T) {
	d := mustParse(t, encode(t, base()))
	tenantA := AssignStepIDs(d, nil)
	tenantB := AssignStepIDs(d, nil)
	for k := range tenantA {
		if tenantA[k] == tenantB[k] {
			t.Errorf("%s: two tenants' identical keys got the same id %s", k, tenantA[k])
		}
	}
}

func TestAssignStepIDsIgnoresNilPrev(t *testing.T) {
	d := mustParse(t, encode(t, base()))
	ids := AssignStepIDs(d, map[string]uuid.UUID{"wait": uuid.Nil})
	if ids["wait"] == uuid.Nil {
		t.Fatal("uuid.Nil carried over from prev")
	}
}

// The committed fixture maps are what AssignStepIDs would keep: v1's map
// covers v1 exactly, and v2 (same keys, a changed value after the timer)
// reuses v1's ids, so histories recorded against v1 replay (P3-22a, P3-24).
func TestFixtureStepIDs(t *testing.T) {
	v1, v2 := readDef(t, "fixture-v1"), readDef(t, "fixture-v2")
	ids1, ids2 := readIDs(t, "fixture-v1"), readIDs(t, "fixture-v2")

	for name, c := range map[string]struct {
		d   *Definition
		ids map[string]uuid.UUID
	}{"fixture-v1": {v1, ids1}, "fixture-v2": {v2, ids2}} {
		if got := AssignStepIDs(c.d, c.ids); !maps.Equal(got, c.ids) {
			t.Errorf("%s: committed map is not a fixed point of AssignStepIDs:\n got %v\nwant %v", name, got, c.ids)
		}
		for k, id := range c.ids {
			if id.Version() != 4 {
				t.Errorf("%s: %s id %s is not v4", name, k, id)
			}
		}
	}
	if !maps.Equal(AssignStepIDs(v2, ids1), ids2) {
		t.Error("fixture-v2 ids are not fixture-v1's ids for unchanged keys")
	}

	a, _ := v1.Step("is_high")
	b, _ := v2.Step("is_high")
	if string(a.Value) == string(b.Value) {
		t.Error("fixture-v2 does not change the step after the timer")
	}
}
