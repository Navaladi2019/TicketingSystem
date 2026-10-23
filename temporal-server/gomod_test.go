// Package temporalserver_test holds module-level build checks (ADR-001 R7).
package temporalserver_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

type goMod struct {
	Module  struct{ Path string }
	Go      string
	Require []struct {
		Path     string
		Version  string
		Indirect bool
	}
	Replace []struct {
		Old struct{ Path string }
		New struct{ Path string }
	}
}

func readGoMod(t *testing.T) goMod {
	t.Helper()
	out, err := exec.Command("go", "mod", "edit", "-json").Output()
	if err != nil {
		t.Fatalf("go mod edit -json: %v", err)
	}
	var m goMod
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// TestModuleIsSeparateFromSDKModule: this module never depends on, or replaces
// into, the platform's SDK module, and keeps its own Temporal pins.
func TestModuleIsSeparateFromSDKModule(t *testing.T) {
	m := readGoMod(t)
	const repo = "github.com/Navaladi2019/TicketingSystem"
	if m.Module.Path != repo+"/temporal-server" {
		t.Fatalf("module path %q", m.Module.Path)
	}
	if len(m.Replace) != 0 {
		t.Errorf("go.mod must have no replace directives, has %+v", m.Replace)
	}
	// ADR-001 A1 Decision 1 (CI-B).
	pins := map[string]string{"go.temporal.io/server": "v1.29.7", "go.temporal.io/api": "v1.53.0"}
	for _, r := range m.Require {
		if strings.HasPrefix(r.Path, repo) {
			t.Errorf("depends on a platform module: %s %s", r.Path, r.Version)
		}
		if want, ok := pins[r.Path]; ok {
			if r.Version != want || r.Indirect {
				t.Errorf("%s = %s (indirect=%v), want direct %s", r.Path, r.Version, r.Indirect, want)
			}
			delete(pins, r.Path)
		}
		// The server embeds its own SDK for the system worker; the platform's
		// SDK (v1.49.0, which needs api v1.63.5) must never be pulled in here.
		if r.Path == "go.temporal.io/sdk" && (!r.Indirect || r.Version == "v1.49.0") {
			t.Errorf("go.temporal.io/sdk must only arrive indirectly via the server, got %s indirect=%v", r.Version, r.Indirect)
		}
	}
	for p, v := range pins {
		t.Errorf("missing pin %s %s", p, v)
	}
	if !versionAtLeast(m.Go, "1.26.0") {
		t.Errorf("go directive %s, want >= 1.26.0", m.Go)
	}
}

func versionAtLeast(have, want string) bool {
	h, w := strings.Split(have, "."), strings.Split(want, ".")
	for i := range w {
		var hv, wv int
		if i < len(h) {
			hv = atoi(h[i])
		}
		wv = atoi(w[i])
		if hv != wv {
			return hv > wv
		}
	}
	return true
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
