package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A bad closed.yaml entry stops the tool with exit status 2 (an error that is
// not a findings count), before it touches git or gh (TICK-379).
func TestBadClosedExitsTwo(t *testing.T) {
	dir := t.TempDir()
	closed := filepath.Join(dir, "closed.yaml")
	for name, bad := range map[string]string{
		"unknown status":     "closed:\n  - branch: TICK-48/x\n    status: merged\n    issue: TICK-48\n",
		"not a story branch": "closed:\n  - branch: agent/x/1\n    status: done\n    issue: TICK-48\n",
	} {
		if err := os.WriteFile(closed, []byte(bad), 0o600); err != nil {
			t.Fatal(err)
		}
		err := run("no-such-remote", "main", "", filepath.Join(dir, "none.yaml"), closed, "")
		var f failed
		if err == nil || errors.As(err, &f) || !strings.Contains(err.Error(), closed) {
			t.Errorf("%s: want an error that exits 2, got %v", name, err)
		}
	}
	// The checked-in file is valid.
	if _, err := readClosed("closed.yaml"); err != nil {
		t.Fatalf("closed.yaml: %v", err)
	}
}
