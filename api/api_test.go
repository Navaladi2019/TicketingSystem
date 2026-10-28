package api_test

import (
	"os"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/Navaladi2019/TicketingSystem/api"
)

// The split must lose nothing. These tests compare the bundle with the
// single-file contract it replaced (testdata/openapi.presplit.yaml) as a
// subset: whatever the bundle adds (a later story's endpoint) is ignored, so
// adding an endpoint never edits this file. Additive-only against main is
// enforced by check-openapi.

func load(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func specs(t *testing.T) (pre, bundle map[string]any) {
	t.Helper()
	data, err := api.Files.ReadFile(api.SpecFile)
	if err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile("testdata/openapi.presplit.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return load(t, old), load(t, data)
}

func operationIDs(t *testing.T, spec map[string]any) map[string]bool {
	t.Helper()
	ids := map[string]bool{}
	for _, item := range spec["paths"].(map[string]any) {
		for _, op := range item.(map[string]any) {
			if m, ok := op.(map[string]any); ok {
				if id, ok := m["operationId"].(string); ok {
					ids[id] = true
				}
			}
		}
	}
	return ids
}

// TestBundleKeepsEveryPreSplitOperationID: no operation was dropped or renamed.
func TestBundleKeepsEveryPreSplitOperationID(t *testing.T) {
	t.Parallel()
	pre, bundle := specs(t)
	want, got := operationIDs(t, pre), operationIDs(t, bundle)
	if len(want) == 0 {
		t.Fatal("pre-split snapshot has no operations")
	}
	for id := range want {
		if !got[id] {
			t.Errorf("operation id %q is missing from the bundle", id)
		}
	}
}

// TestBundleKeepsPreSplitPathsAndComponents: every pre-split path item and
// component is in the bundle, equal in meaning (key order and the file it
// came from do not matter).
func TestBundleKeepsPreSplitPathsAndComponents(t *testing.T) {
	t.Parallel()
	pre, bundle := specs(t)
	prePaths, bundlePaths := pre["paths"].(map[string]any), bundle["paths"].(map[string]any)
	for p, item := range prePaths {
		if !reflect.DeepEqual(bundlePaths[p], item) {
			t.Errorf("path %s differs from the pre-split spec", p)
		}
	}
	preC, bundleC := pre["components"].(map[string]any), bundle["components"].(map[string]any)
	for section, entries := range preC {
		got, _ := bundleC[section].(map[string]any)
		for name, def := range entries.(map[string]any) {
			if !reflect.DeepEqual(got[name], def) {
				t.Errorf("components.%s.%s differs from the pre-split spec", section, name)
			}
		}
	}
}
