package configsurface

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func realInventory(t *testing.T) inventory {
	t.Helper()
	inv, err := decodeInventory(readFile(t, inventoryPath))
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func realSurface(t *testing.T) itemSet {
	t.Helper()
	w, err := workflowSurface(readFile(t, workflowSchema))
	if err != nil {
		t.Fatal(err)
	}
	td, err := typedefSurface(metadataSrcDir)
	if err != nil {
		t.Fatal(err)
	}
	for it := range td {
		w[it] = true
	}
	return w
}

func realFixtures(t *testing.T) []fixtureResult {
	t.Helper()
	f, err := loadFixtures(fixturesDir)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func reportAll(t *testing.T, problems []string) {
	t.Helper()
	for _, p := range problems {
		t.Error(p)
	}
}

// AC-6: the real repository passes every check.

func TestRemoval(t *testing.T) {
	reportAll(t, removed(realInventory(t).items(), realSurface(t)))
}

func TestUnlistedAddition(t *testing.T) {
	reportAll(t, unlisted(realInventory(t).listed(), realSurface(t)))
}

func TestFixturesStillAccepted(t *testing.T) {
	reportAll(t, stillAccepted(realInventory(t), realFixtures(t)))
}

func TestDeprecatedStillAccepted(t *testing.T) {
	reportAll(t, deprecatedStillAccepted(realInventory(t), realFixtures(t)))
}

// A reserved item is still refused by Compile: its fixture under
// testdata/fixtures/reserved names the item and is not accepted. The day
// Compile accepts it, it is surface and moves into the inventory proper.
func TestReservedStillRefused(t *testing.T) {
	t.Parallel()
	for _, r := range realInventory(t).Reserved {
		_, name, _ := strings.Cut(r.Item, "/")
		path := filepath.Join(fixturesDir, "reserved", name+".json")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reserved %s: %v; add %s", r.Item, err, path)
			continue
		}
		f := typedefFixture("reserved/"+name+".json", b)
		if !f.uses[r.Item] {
			t.Errorf("reserved fixture %s does not use %s", f.name, r.Item)
		}
		if f.problem == "" {
			t.Errorf("reserved %s is accepted by Compile; move it into inventory.json (%s)", r.Item, r.Reason)
		}
	}
}

// No script surface exists in v1 (PLAN-009 D-9-6). The day the schema gains a
// script step type, this test and check 2 both fail until it is listed.
func TestNoScriptSurface(t *testing.T) {
	inv := realInventory(t)
	surface, err := workflowSurface(readFile(t, workflowSchema))
	if err != nil {
		t.Fatal(err)
	}
	if surface[item("workflow.step_types", scriptStepType)] && len(inv.Script) == 0 {
		t.Error("unlisted: the workflow schema has a script step type; list the script surface in inventory.json")
	}
	if len(inv.Script) != 0 {
		t.Errorf("inventory lists a script surface %q, but v1 has none (PLAN-009 D-9-6)", inv.Script)
	}
}

// --- Injected inputs ---------------------------------------------------------

// mutateSchema returns schema.json with edit applied to its decoded form.
func mutateSchema(t *testing.T, edit func(root map[string]any)) []byte {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(readFile(t, workflowSchema), &root); err != nil {
		t.Fatal(err)
	}
	edit(root)
	b, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func stepTypeEnum(root map[string]any) map[string]any {
	defs := root["$defs"].(map[string]any)
	step := defs["step"].(map[string]any)
	return step["properties"].(map[string]any)["type"].(map[string]any)
}

// AC-1: a schema.json copy without timer gives "removed: workflow.step_types/timer".
func TestRemovalReportsDroppedStepType(t *testing.T) {
	schema := mutateSchema(t, func(root map[string]any) {
		e := stepTypeEnum(root)
		e["enum"] = slices.DeleteFunc(e["enum"].([]any), func(v any) bool { return v == "timer" })
		delete(root["$defs"].(map[string]any), "timer_step")
	})
	src, err := workflowSurface(schema)
	if err != nil {
		t.Fatal(err)
	}
	got := removed(realInventory(t).items(), src)
	if !slices.Contains(got, "removed: workflow.step_types/timer") {
		t.Errorf("removed = %q, want it to name workflow.step_types/timer", got)
	}
	if !slices.Contains(got, "removed: workflow.step_fields/timer.duration") {
		t.Errorf("removed = %q, want it to name workflow.step_fields/timer.duration", got)
	}
}

func TestRemovalReportsDroppedEnumValues(t *testing.T) {
	schema := mutateSchema(t, func(root map[string]any) {
		props := root["properties"].(map[string]any)
		props["trigger"].(map[string]any)["properties"].(map[string]any)["on"] = map[string]any{"enum": []any{"record.created"}}
		cond := root["$defs"].(map[string]any)["condition"].(map[string]any)
		cond["properties"].(map[string]any)["op"] = map[string]any{"enum": []any{"eq", "ne"}}
		ts := root["$defs"].(map[string]any)["wait_signal_step"].(map[string]any)
		delete(ts["properties"].(map[string]any), "on_timeout")
	})
	src, err := workflowSurface(schema)
	if err != nil {
		t.Fatal(err)
	}
	got := removed(realInventory(t).items(), src)
	for _, want := range []string{
		"removed: workflow.trigger_on/record.updated",
		"removed: workflow.condition_ops/in",
		"removed: workflow.step_fields/wait_signal.on_timeout",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("removed = %q, want %q", got, want)
		}
	}
}

func TestRemovalReportsDroppedExtension(t *testing.T) {
	src := realSurface(t)
	delete(src, item("type_definition.extensions", "x-nava-unique"))
	got := removed(realInventory(t).items(), src)
	if !slices.Equal(got, []string{"removed: type_definition.extensions/x-nava-unique"}) {
		t.Errorf("removed = %q", got)
	}
}

// Keywords are hand-curated, so checks 1 and 2 leave them to check 3.
func TestRemovalIgnoresCuratedKeywords(t *testing.T) {
	inv := itemSet{}
	inv.add("type_definition.keywords", "minLength")
	if got := removed(inv, itemSet{}); len(got) != 0 {
		t.Errorf("removed = %q, want none for a curated keyword", got)
	}
}

// AC-2: an injected step type noop gives "unlisted".
func TestUnlistedReportsNewStepType(t *testing.T) {
	schema := mutateSchema(t, func(root map[string]any) {
		e := stepTypeEnum(root)
		e["enum"] = append(e["enum"].([]any), "noop")
		root["$defs"].(map[string]any)["noop_step"] = map[string]any{
			"properties": map[string]any{"key": true, "type": true, "next": map[string]any{}},
		}
	})
	src, err := workflowSurface(schema)
	if err != nil {
		t.Fatal(err)
	}
	got := unlisted(realInventory(t).listed(), src)
	for _, want := range []string{
		"unlisted: workflow.step_types/noop; add it to inventory.json",
		"unlisted: workflow.step_fields/noop.next; add it to inventory.json",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("unlisted = %q, want %q", got, want)
		}
	}
}

func TestUnlistedReportsScriptStepType(t *testing.T) {
	schema := mutateSchema(t, func(root map[string]any) {
		e := stepTypeEnum(root)
		e["enum"] = append(e["enum"].([]any), scriptStepType)
		root["$defs"].(map[string]any)["script_step"] = map[string]any{"properties": map[string]any{"key": true}}
	})
	src, err := workflowSurface(schema)
	if err != nil {
		t.Fatal(err)
	}
	if got := unlisted(realInventory(t).listed(), src); !slices.Contains(got, "unlisted: script/script; add it to inventory.json") {
		t.Errorf("unlisted = %q, want the script surface", got)
	}
}

func TestUnlistedReportsNewExtensionAndField(t *testing.T) {
	dir := t.TempDir()
	src := "package metadata\n\nconst k = \"x-nava-pii\"\n"
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	// A test file's literal is not surface.
	if err := os.WriteFile(filepath.Join(dir, "x_test.go"), []byte("package metadata\n\nconst j = \"x-nava-test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := typedefSurface(dir)
	if err != nil {
		t.Fatal(err)
	}
	inv := realInventory(t).listed()
	got := unlisted(inv, s)
	if !slices.Contains(got, "unlisted: type_definition.extensions/x-nava-pii; add it to inventory.json") {
		t.Errorf("unlisted = %q, want x-nava-pii", got)
	}
	if slices.ContainsFunc(got, func(p string) bool { return strings.Contains(p, "x-nava-test") }) {
		t.Errorf("unlisted = %q, a _test.go literal must not count", got)
	}

	delete(inv, item("type_definition.fields", "states.transitions.to"))
	if got := unlisted(inv, realSurface(t)); !slices.Equal(got, []string{"unlisted: type_definition.fields/states.transitions.to; add it to inventory.json"}) {
		t.Errorf("unlisted = %q", got)
	}
}

func TestSurfaceErrors(t *testing.T) {
	if _, err := workflowSurface([]byte("{")); err == nil {
		t.Error("workflowSurface accepted invalid JSON")
	}
	noDef := mutateSchema(t, func(root map[string]any) { delete(root["$defs"].(map[string]any), "end_step") })
	if _, err := workflowSurface(noDef); err == nil || !strings.Contains(err.Error(), "end_step") {
		t.Errorf("workflowSurface without end_step: err = %v", err)
	}
	if _, err := typedefSurface(t.TempDir()); err == nil {
		t.Error("typedefSurface accepted a directory with no Go files")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte("package"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := typedefSurface(dir); err == nil {
		t.Error("typedefSurface accepted a file that does not parse")
	}
	if _, err := typedefSurface("["); err == nil {
		t.Error("typedefSurface accepted a bad glob")
	}
	if _, err := decodeInventory([]byte(`{"major": 1, "extra": true}`)); err == nil {
		t.Error("decodeInventory accepted an unknown field")
	}
}

// AC-3: a fixture that no longer parses is reported with its file name and
// def.Parse's problem paths.
func TestStillAcceptedReportsWorkflowFixtureThatNoLongerParses(t *testing.T) {
	b := []byte(`{"format":"nava.workflow/v1","trigger":{"on":"record.created","entity":"incident"},"start":"wait",
		"steps":[{"key":"wait","type":"timer","duration":"PT5M","delay":"PT1M","next":"done"},{"key":"done","type":"end"}]}`)
	fixtures := append(realFixtures(t), workflowFixture("workflow/broken.json", b))
	got := stillAccepted(realInventory(t), fixtures)
	want := `not accepted: workflow/broken.json: workflow definition: invalid at "/steps/0/delay" (paths ["/steps/0/delay"])`
	if !slices.Contains(got, want) {
		t.Errorf("stillAccepted = %q, want %q", got, want)
	}
}

func TestStillAcceptedReportsTypedefFixtureThatNoLongerCompiles(t *testing.T) {
	b := readFile(t, filepath.Join(fixturesDir, "typedef", "minLength.json"))
	b = []byte(strings.Replace(string(b), `"minLength": 1`, `"minLength": "one"`, 1))
	r := typedefFixture("typedef/minLength.json", b)
	if !strings.HasPrefix(r.problem, "metadata: invalid entity definition at /schema") || !strings.Contains(r.problem, `paths ["/schema`) {
		t.Errorf("problem = %q, want metadata's error and its paths", r.problem)
	}

	fixtures := slices.DeleteFunc(realFixtures(t), func(f fixtureResult) bool { return f.name == r.name })
	got := stillAccepted(realInventory(t), append(fixtures, r))
	if !slices.ContainsFunc(got, func(p string) bool { return strings.HasPrefix(p, "not accepted: typedef/minLength.json: ") }) {
		t.Errorf("stillAccepted = %q, want typedef/minLength.json reported", got)
	}
	if !slices.Contains(got, "unused: type_definition.keywords/minLength is used by no accepted fixture") {
		t.Errorf("stillAccepted = %q, want minLength reported unused", got)
	}
}

func TestTypedefFixtureRefusesUnknownField(t *testing.T) {
	r := typedefFixture("typedef/x.json", []byte(`{"key":"x","colour":"red"}`))
	if !strings.Contains(r.problem, "colour") {
		t.Errorf("problem = %q, want the unknown field named", r.problem)
	}
	r = workflowFixture("workflow/x.json", []byte(`[`))
	if r.problem == "" {
		t.Error("workflowFixture accepted invalid JSON")
	}
	r = workflowFixture("workflow/x.json", []byte(`{"format": 1}`))
	if r.problem == "" {
		t.Error("workflowFixture accepted a mistyped format")
	}
}

// AC-4: an inventory item with no fixture is reported.
func TestStillAcceptedReportsItemWithNoFixture(t *testing.T) {
	inv := realInventory(t)
	inv.Workflow.StepTypes = append(inv.Workflow.StepTypes, "noop")
	inv.Workflow.StepFields["noop"] = []string{"key", "type"}
	inv.TypeDefinition.Keywords = append(inv.TypeDefinition.Keywords, "pattern")
	inv.TypeDefinition.Fields = append(inv.TypeDefinition.Fields, "labels")
	got := stillAccepted(inv, realFixtures(t))
	for _, want := range []string{
		"no fixture: workflow.step_types/noop; add testdata/fixtures/workflow/noop.json",
		"no fixture: type_definition.keywords/pattern; add testdata/fixtures/typedef/pattern.json",
		"unused: workflow.step_fields/noop.key is used by no accepted fixture",
		"unused: type_definition.fields/labels is used by no accepted fixture",
	} {
		if !slices.Contains(got, want) {
			t.Errorf("stillAccepted = %q, want %q", got, want)
		}
	}
}

func TestStillAcceptedReportsFixtureNotUsingItsItem(t *testing.T) {
	fixtures := realFixtures(t)
	for i, f := range fixtures {
		if f.name == "workflow/timer.json" {
			fixtures[i] = workflowFixture(f.name, readFile(t, filepath.Join(fixturesDir, "workflow", "end.json")))
		}
	}
	got := stillAccepted(realInventory(t), fixtures)
	if !slices.Contains(got, "fixture workflow/timer.json does not use workflow.step_types/timer") {
		t.Errorf("stillAccepted = %q", got)
	}
}

func TestLoadFixturesErrors(t *testing.T) {
	if _, err := loadFixtures("["); err == nil {
		t.Error("loadFixtures accepted a bad glob")
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "workflow", "d.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := loadFixtures(dir); err == nil {
		t.Error("loadFixtures read a directory as a fixture")
	}
}

// Check 4 on an injected inventory: v1 deprecates nothing (PLAN-009 D-9-5),
// so the path is proven with an existing item marked deprecated.
func TestDeprecatedItem(t *testing.T) {
	inv := realInventory(t)
	fixtures := realFixtures(t)
	inv.Deprecated = []deprecated{{Item: "workflow.step_types/timer", Since: "1.4"}}
	if got := deprecatedStillAccepted(inv, fixtures); len(got) != 0 {
		t.Errorf("deprecated timer, listed and accepted: %q", got)
	}

	broken := slices.Clone(fixtures)
	for i, f := range broken {
		if f.uses["workflow.step_types/timer"] {
			broken[i].problem = "invalid"
		}
	}
	if got := deprecatedStillAccepted(inv, broken); !slices.Equal(got, []string{"deprecated: workflow.step_types/timer (since 1.4) has no accepted fixture"}) {
		t.Errorf("deprecated timer, fixture broken: %q", got)
	}

	inv.Workflow.StepTypes = slices.DeleteFunc(slices.Clone(inv.Workflow.StepTypes), func(s string) bool { return s == "timer" })
	if got := deprecatedStillAccepted(inv, fixtures); !slices.Equal(got, []string{"deprecated: workflow.step_types/timer (since 1.4) is no longer in the inventory"}) {
		t.Errorf("deprecated timer, unlisted: %q", got)
	}
}
