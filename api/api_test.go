package api_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"go.yaml.in/yaml/v3"

	"github.com/Navaladi2019/TicketingSystem/api"
)

func bundle(t *testing.T) []byte {
	t.Helper()
	data, err := api.Files.ReadFile(api.SpecFile)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestBundleOperationIDsMatchPreSplit: splitting the source must not add,
// drop or rename an operation. The ids below are the pre-split set.
func TestBundleOperationIDsMatchPreSplit(t *testing.T) {
	t.Parallel()
	spec, err := openapi3.NewLoader().LoadFromData(bundle(t))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, item := range spec.Paths.Map() {
		for _, op := range item.Operations() {
			got = append(got, op.OperationID)
		}
	}
	slices.Sort(got)
	want := []string{"getMe", "onboardTenant"}
	if !slices.Equal(got, want) {
		t.Errorf("operation ids = %v, want %v", got, want)
	}
}

// TestBundleIsSemanticallyThePreSplitSpec compares the bundle with the single
// file contract it replaced (testdata/openapi.presplit.yaml): same document,
// whatever the key order or the file it came from. The only allowed
// difference is the description, which now explains the source tree.
func TestBundleIsSemanticallyThePreSplitSpec(t *testing.T) {
	t.Parallel()
	load := func(data []byte) map[string]any {
		var m map[string]any
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}
		delete(m["info"].(map[string]any), "description")
		return m
	}
	pre, err := readTestdata()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(load(bundle(t)), load(pre)) {
		t.Error("bundle differs in meaning from testdata/openapi.presplit.yaml")
	}
}
