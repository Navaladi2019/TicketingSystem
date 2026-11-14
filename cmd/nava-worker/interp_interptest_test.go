//go:build interptest

package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
	"github.com/Navaladi2019/TicketingSystem/internal/workflowdefs"
)

// TestInterptestBuildWiresTheFakes: the interptest stack worker reads its
// fakes from FixtureDirEnv and writes to WritesDirEnv (default the fixture
// directory); both must be absolute paths.
func TestInterptestBuildWiresTheFakes(t *testing.T) {
	for _, v := range []string{"", "relative/dir"} {
		if _, err := interpreterActivities(func(string) (string, bool) { return v, v != "" }, nil, nil, true); err == nil {
			t.Errorf("%s=%q: accepted", FixtureDirEnv, v)
		}
	}
	dir := t.TempDir()
	acts, err := interpreterActivities(func(k string) (string, bool) { return dir, k == FixtureDirEnv }, nil, nil, true)
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
	acts, err = interpreterActivities(func(k string) (string, bool) { v, ok := env[k]; return v, ok }, nil, nil, true)
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
	if _, err := interpreterActivities(func(k string) (string, bool) { v, ok := env[k]; return v, ok }, nil, nil, true); err == nil {
		t.Errorf("%s=relative: accepted", WritesDirEnv)
	}
}

// TestInterptestBuildProductionPorts: with PortsEnv=db the interptest worker
// runs the production ports (the P3-44 crash case against the real
// pipeline.Write) and needs no fixture directory; any other value is refused
// rather than read as the fakes.
func TestInterptestBuildProductionPorts(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://nava_worker@127.0.0.1:1/nava")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg) // lazy: never connects here
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	env := func(v string) func(string) (string, bool) {
		return func(k string) (string, bool) { return v, k == PortsEnv }
	}
	acts, err := interpreterActivities(env("db"), pool, pool, true)
	if err != nil || acts == nil {
		t.Fatalf("%s=db: (%v, %v)", PortsEnv, acts, err)
	}
	if _, ok := acts.Definitions.(workflowdefs.Reader); !ok {
		t.Errorf("Definitions %T, want workflowdefs.Reader", acts.Definitions)
	}
	if _, ok := acts.Records.(recordports.Reader); !ok {
		t.Errorf("Records %T, want recordports.Reader", acts.Records)
	}
	if _, ok := acts.Writer.(recordports.Writer); !ok {
		t.Errorf("Writer %T, want recordports.Writer (the real pipeline)", acts.Writer)
	}
	if _, ok := acts.Waits.(workerguard.WaitTracker); !ok {
		t.Errorf("Waits %T, want workerguard.WaitTracker", acts.Waits)
	}
	if _, err := interpreterActivities(env("db"), nil, nil, true); err == nil {
		t.Errorf("%s=db without pools: accepted", PortsEnv)
	}
	if _, err := interpreterActivities(env("fakes"), pool, pool, true); err == nil {
		t.Errorf("%s=fakes: accepted", PortsEnv)
	}
}
