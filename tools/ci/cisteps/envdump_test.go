package main

import (
	"os"
	"strings"
	"testing"
)

// TestMain makes the test binary the environment dump when make runs it as
// SHELL, as main does for the cisteps binary: checkCopy gives make
// os.Executable().
func TestMain(m *testing.M) {
	if envDumpRequested() {
		os.Exit(envDump(os.Stdout))
	}
	os.Exit(m.Run())
}

func TestParseEnvDump(t *testing.T) {
	out := "make: something\n" + envDumpPrefix + `{"args":["-c","x"],"env":["A=1","B=a=b"]}` + "\n\n"
	recs, other, err := parseEnvDump(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || strings.Join(recs[0].Args, " ") != "-c x" || envMap(recs[0].Env)["B"] != "a=b" {
		t.Fatalf("records %+v", recs)
	}
	if len(other) != 1 || other[0] != "make: something" {
		t.Fatalf("other %q", other)
	}
	if _, _, err := parseEnvDump(envDumpPrefix + "{"); err == nil {
		t.Fatal("a garbled record parsed")
	}
}
