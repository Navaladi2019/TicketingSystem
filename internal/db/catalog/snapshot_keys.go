package catalog

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// contractKeyRow is a row of the key table in
// docs/handbook/src/database/schema.md: "| `tables[].indexes[].include` | ...".
const contractKeyRow = "(?m)^\\|\\s*`((?:version|tables)[a-z_.\\[\\]]*)`\\s*\\|"

// SnapshotContractKeys are the key paths the contract document lists, in
// document order. A path names a key by its object keys joined with ".",
// with "[]" after a key whose value is an array of objects.
func SnapshotContractKeys(doc []byte) []string {
	var out []string
	for _, m := range regexp.MustCompile(contractKeyRow).FindAllSubmatch(doc, -1) {
		out = append(out, string(m[1]))
	}
	return out
}

// SnapshotKeyProblems compares a snapshot file with the contract document:
// every key the document lists must be present in every object that should
// carry it (present, not merely non-empty), at least one such object must
// exist, and the file must carry no key the document does not list. Empty
// means the file meets the contract's shape.
func SnapshotKeyProblems(doc, snapshot []byte) []string {
	listed := SnapshotContractKeys(doc)
	if len(listed) == 0 {
		return []string{"the contract document lists no keys"}
	}
	var root any
	if err := json.Unmarshal(snapshot, &root); err != nil {
		return []string{"snapshot is not JSON: " + err.Error()}
	}
	keys, objects := map[string]int{}, map[string]int{}
	walkKeys(root, "", keys, objects)
	var out []string
	for _, p := range listed {
		parent := ""
		if i := strings.LastIndex(p, "."); i >= 0 {
			parent = p[:i]
		}
		switch n, want := keys[p], objects[parent]; {
		case want == 0:
			out = append(out, fmt.Sprintf("%s: the snapshot has no %s object to carry it", p, parent))
		case n != want:
			out = append(out, fmt.Sprintf("%s: missing from %d of %d objects", p, want-n, want))
		}
	}
	for p := range keys {
		if !slices.Contains(listed, p) {
			out = append(out, fmt.Sprintf("%s: in the snapshot but not listed in the contract document", p))
		}
	}
	slices.Sort(out)
	return out
}

// walkKeys counts, under prefix, each object (objects) and each key path
// (keys) in v.
func walkKeys(v any, prefix string, keys, objects map[string]int) {
	obj, ok := v.(map[string]any)
	if !ok {
		return
	}
	objects[prefix]++
	for k, child := range obj {
		p := k
		if prefix != "" {
			p = prefix + "." + k
		}
		keys[p]++
		switch c := child.(type) {
		case map[string]any:
			walkKeys(c, p, keys, objects)
		case []any:
			for _, e := range c {
				walkKeys(e, p+"[]", keys, objects)
			}
		}
	}
}
