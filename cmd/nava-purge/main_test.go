package main

import (
	"strings"
	"testing"
)

// Unreachable database: run fails on the role check and never reaches the
// purger, so nothing is deleted.
func TestRunFailsWithoutDatabase(t *testing.T) {
	env := map[string]string{"NAVA_DATABASE_URL": "postgres://nava_purge:s3cret@127.0.0.1:1/nava"}
	err := run(t.Context(), func(k string) (string, bool) { v, ok := env[k]; return v, ok }, &strings.Builder{})
	if err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("run = %v, want an error without the DSN", err)
	}
}
