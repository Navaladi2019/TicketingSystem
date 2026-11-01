//go:build interptest

package main

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

// TestInterptestBuildWiresTheFakes: the interptest stack worker reads its
// fakes from FixtureDirEnv and writes to WritesDirEnv (default the fixture
// directory); both must be absolute paths.
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

	// The stack mounts the fixtures read-only and gives the writer its own
	// directory.
	writes := t.TempDir()
	env := map[string]string{FixtureDirEnv: dir, WritesDirEnv: writes}
	acts, err = interpreterActivities(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if err != nil {
		t.Fatal(err)
	}
	if w, ok := acts.Writer.(*interptest.FileWriter); !ok || w.Root != writes {
		t.Errorf("Writer %#v, want the file writer over %s", acts.Writer, writes)
	}
	if r, ok := acts.Records.(interptest.FileRecords); !ok || r.Root != dir {
		t.Errorf("Records %#v, want the file records over %s", acts.Records, dir)
	}
	env[WritesDirEnv] = "relative"
	if _, err := interpreterActivities(func(k string) (string, bool) { v, ok := env[k]; return v, ok }); err == nil {
		t.Errorf("%s=relative: accepted", WritesDirEnv)
	}
}
