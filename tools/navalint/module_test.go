package main

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

const xtools = "golang.org/x/tools"

// TestXToolsStaysOutOfProduction: navalint's golang.org/x/tools is not a
// direct require of the production go.mod, is not the version that module
// builds with, and no production package or test imports it (TICK-94 item
// 15). The production go.mod may hold an older x/tools as an indirect
// require: the oapi-codegen tool directive and goose bring it (docs/pins.md).
func TestXToolsStaysOutOfProduction(t *testing.T) {
	own, ok := requires(t, "go.mod")[xtools]
	if !ok {
		t.Fatalf("navalint's go.mod does not require %s", xtools)
	}
	if r, ok := requires(t, "../../go.mod")[xtools]; ok {
		if !r.Indirect {
			t.Errorf("the production go.mod requires %s directly", xtools)
		}
		if r.Version == own.Version {
			t.Errorf("the production go.mod has navalint's %s %s", xtools, own.Version)
		}
	}

	cmd := exec.Command("go", "list", "-deps", "-test", "./...")
	cmd.Dir = "../.."
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if pkg == xtools || strings.HasPrefix(pkg, xtools+"/") {
			t.Errorf("a production package or test imports %s", pkg)
		}
	}
}

type require struct {
	Path, Version string
	Indirect      bool
}

// requires reads a go.mod with go mod edit -json, keyed by module path.
func requires(t *testing.T, gomod string) map[string]require {
	t.Helper()
	out, err := exec.Command("go", "mod", "edit", "-json", gomod).Output()
	if err != nil {
		t.Fatalf("go mod edit -json %s: %v", gomod, err)
	}
	var f struct{ Require []require }
	if err := json.Unmarshal(out, &f); err != nil {
		t.Fatal(err)
	}
	m := make(map[string]require, len(f.Require))
	for _, r := range f.Require {
		m[r.Path] = r
	}
	return m
}
