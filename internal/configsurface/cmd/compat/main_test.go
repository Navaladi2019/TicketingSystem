package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	baseV1 = `{"major": 1,
		"workflow": {"step_types": ["end", "timer"], "step_fields": {"timer": ["duration", "next"]}},
		"type_definition": {"keywords": ["type"]}, "script": [], "deprecated": []}`
	headNoTimer = `{"major": 1,
		"workflow": {"step_types": ["end"], "step_fields": {}},
		"type_definition": {"keywords": ["type"]}, "script": [], "deprecated": []}`
	headNoTimerV2 = `{"major": 2,
		"workflow": {"step_types": ["end"], "step_fields": {}},
		"type_definition": {"keywords": ["type"]}, "script": [], "deprecated": []}`
)

// fakeRepo is an injected git ref and working tree.
type fakeRepo struct {
	base  map[string]string // path -> content on the base; nil: the ref does not exist
	files map[string]string // working-tree files
}

func (r fakeRepo) show(_, path string) ([]byte, error) {
	if r.base == nil {
		return nil, errors.New("base origin/main not found")
	}
	s, ok := r.base[path]
	if !ok {
		return nil, errNoFile
	}
	return []byte(s), nil
}

func (r fakeRepo) read(path string) ([]byte, error) {
	s, ok := r.files[filepath.ToSlash(path)]
	if !ok {
		return nil, os.ErrNotExist
	}
	return []byte(s), nil
}

func runFake(t *testing.T, r fakeRepo, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr, r.show, r.read)
	return code, stdout.String(), stderr.String()
}

// AC-5, first case: an item on the base missing on head is reported (the
// "removed from the schema and the inventory" case).
func TestRemovedFromBothIsReported(t *testing.T) {
	code, _, stderr := runFake(t, fakeRepo{
		base:  map[string]string{defaultInventory: baseV1},
		files: map[string]string{defaultInventory: headNoTimer},
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1; stderr: %s", code, stderr)
	}
	for _, want := range []string{"removed: workflow.step_types/timer", "removed: workflow.step_fields/timer.duration", "removed: workflow.step_fields/timer.next"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	}
}

// AC-5: major 2 with no docs/upgrade/v2.md is still reported.
func TestMajorBumpWithoutNoteIsReported(t *testing.T) {
	code, _, stderr := runFake(t, fakeRepo{
		base:  map[string]string{defaultInventory: baseV1},
		files: map[string]string{defaultInventory: headNoTimerV2},
	})
	if code != 1 || !strings.Contains(stderr, "removed: workflow.step_types/timer (major 1 -> 2, but there is no docs/upgrade/v2.md)") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}

func TestMajorBumpWithNoteMissingAnItemIsReported(t *testing.T) {
	code, _, stderr := runFake(t, fakeRepo{
		base: map[string]string{defaultInventory: baseV1},
		files: map[string]string{
			defaultInventory:     headNoTimerV2,
			"docs/upgrade/v2.md": "Removed: workflow.step_types/timer, workflow.step_fields/timer.duration",
		},
	})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if want := "removed: workflow.step_fields/timer.next (major 1 -> 2, but docs/upgrade/v2.md does not name it)"; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
	if strings.Contains(stderr, "timer.duration") || strings.Contains(stderr, "step_types/timer") {
		t.Errorf("stderr = %q, names an item the note covers", stderr)
	}
}

// AC-5: with the note naming every removed item the check is clean.
func TestMajorBumpWithNoteIsClean(t *testing.T) {
	code, stdout, stderr := runFake(t, fakeRepo{
		base: map[string]string{defaultInventory: baseV1},
		files: map[string]string{
			defaultInventory:     headNoTimerV2,
			"docs/upgrade/v2.md": "## Removed\n\n- workflow.step_types/timer\n- workflow.step_fields/timer.duration\n- workflow.step_fields/timer.next\n",
		},
	})
	if code != 0 || stderr != "" || !strings.Contains(stdout, "compat: ok") {
		t.Errorf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
}

// AC-5: no base file means nothing to compare.
func TestNoBaseFileIsNothingToCompare(t *testing.T) {
	code, stdout, _ := runFake(t, fakeRepo{base: map[string]string{}, files: map[string]string{}})
	if code != 0 || !strings.Contains(stdout, "nothing to compare") {
		t.Errorf("code = %d, stdout = %q", code, stdout)
	}
}

func TestAdditionIsClean(t *testing.T) {
	head := strings.Replace(baseV1, `["end", "timer"]`, `["end", "noop", "timer"]`, 1)
	code, _, stderr := runFake(t, fakeRepo{
		base:  map[string]string{defaultInventory: baseV1},
		files: map[string]string{defaultInventory: head},
	})
	if code != 0 {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}

func TestBadInput(t *testing.T) {
	cases := map[string]fakeRepo{
		"missing base ref": {files: map[string]string{defaultInventory: baseV1}},
		"missing head":     {base: map[string]string{defaultInventory: baseV1}, files: map[string]string{}},
		"bad base":         {base: map[string]string{defaultInventory: `{`}, files: map[string]string{defaultInventory: baseV1}},
		"bad head":         {base: map[string]string{defaultInventory: baseV1}, files: map[string]string{defaultInventory: `{"major": 0}`}},
	}
	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			if code, _, _ := runFake(t, r); code != 2 {
				t.Errorf("code = %d, want 2", code)
			}
		})
	}
	if code, _, _ := runFake(t, fakeRepo{}, "-nope"); code != 2 {
		t.Errorf("unknown flag: code = %d, want 2", code)
	}
	if code, _, _ := runFake(t, fakeRepo{}, "-h"); code != 0 {
		t.Errorf("-h: code = %d, want 0", code)
	}
}

func TestMajorDown(t *testing.T) {
	base := strings.Replace(baseV1, `"major": 1`, `"major": 2`, 1)
	code, _, stderr := runFake(t, fakeRepo{
		base:  map[string]string{defaultInventory: base},
		files: map[string]string{defaultInventory: baseV1},
	})
	if code != 1 || !strings.Contains(stderr, "major went down from 2 to 1") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}

func TestParse(t *testing.T) {
	inv, err := parse([]byte(baseV1))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"type_definition.keywords/type",
		"workflow.step_fields/timer.duration",
		"workflow.step_fields/timer.next",
		"workflow.step_types/end",
		"workflow.step_types/timer",
	}
	if inv.major != 1 || !slices.Equal(inv.items, want) {
		t.Errorf("parse = %d %q, want 1 %q", inv.major, inv.items, want)
	}
	inv, err = parse([]byte(`{"major": 1, "script": ["run"]}`))
	if err != nil || !slices.Equal(inv.items, []string{"script/run"}) {
		t.Errorf("script: %q, %v", inv.items, err)
	}
	// A reserved item is metadata: Compile refuses it, so it is not surface.
	inv, err = parse([]byte(`{"major": 1, "reserved": [{"item": "type_definition.extensions/x-nava-searchable", "reason": "r"}]}`))
	if err != nil || len(inv.items) != 0 {
		t.Errorf("reserved: %q, %v", inv.items, err)
	}
	for _, bad := range []string{`[]`, `{"major": "1"}`, `{"major": 1, "workflow": 3}`, `{"major": 1, "workflow": {"step_fields": {"timer": 3}}}`} {
		if _, err := parse([]byte(bad)); err == nil {
			t.Errorf("parse(%s) accepted", bad)
		}
	}
}

// The real inventory parses into the item names the suite uses.
func TestParseRealInventory(t *testing.T) {
	b, err := os.ReadFile("../../testdata/inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := parse(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"workflow.step_types/timer", "workflow.step_fields/wait_signal.on_timeout", "type_definition.extensions/x-nava-secret", "type_definition.fields/states.initial", "workflow.formats/nava.workflow/v1"} {
		if !slices.Contains(inv.items, want) {
			t.Errorf("items lack %q", want)
		}
	}
}

// gitShow against a throwaway repository: a missing ref fails, a ref without
// the file is errNoFile, a ref with it returns its content.
func TestGitShow(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	mustGit := func(args ...string) {
		t.Helper()
		if _, err := git(args...); err != nil {
			t.Fatal(err)
		}
	}
	mustGit("init", "-q")
	mustGit("config", "user.email", "compat@example.invalid")
	mustGit("config", "user.name", "compat test")
	mustGit("config", "commit.gpgsign", "false")
	if err := os.WriteFile("a.txt", []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustGit("add", "a.txt")
	mustGit("commit", "-q", "-m", "a")

	if _, err := gitShow("no-such-ref", "a.txt"); err == nil || errors.Is(err, errNoFile) {
		t.Errorf("missing ref: err = %v, want a not-found error", err)
	}
	if _, err := gitShow("HEAD", "b.txt"); !errors.Is(err, errNoFile) {
		t.Errorf("missing file: err = %v, want errNoFile", err)
	}
	b, err := gitShow("HEAD", "a.txt")
	if err != nil || string(b) != "a" {
		t.Errorf("gitShow = %q, %v", b, err)
	}
}
