//go:build interptest

package main

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

// TestInterptestBuildWiresTheFakes: the interptest stack worker reads its
// fakes from FixtureDirEnv, which must be an absolute path.
func TestInterptestBuildWiresTheFakes(t *testing.T) {
	for _, v := range []string{"", "relative/dir"} {
		if _, err := interpreterActivities(func(string) (string, bool) { return v, v != "" }); err == nil {
			t.Errorf("%s=%q: accepted", FixtureDirEnv, v)
		}
	}
	dir := t.TempDir()
	acts, err := interpreterActivities(func(k string) (string, bool) { return dir, k == FixtureDirEnv })
	if err != nil || acts == nil {
		t.Fatalf("interpreterActivities = (%v, %v)", acts, err)
	}
	if s, ok := acts.Definitions.(interptest.FixtureStore); !ok || s.Root != dir {
		t.Errorf("Definitions %#v, want the fixture store over %s", acts.Definitions, dir)
	}
	if w, ok := acts.Writer.(*interptest.FileWriter); !ok || w.Root != dir {
		t.Errorf("Writer %#v, want the file writer over %s", acts.Writer, dir)
	}
}
