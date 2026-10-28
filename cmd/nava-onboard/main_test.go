package main

import (
	"io"
	"testing"
)

func TestRunRefusesMissingConfig(t *testing.T) {
	lookup := func(string) (string, bool) { return "", false }
	if err := run(t.Context(), lookup, io.Discard); err == nil {
		t.Fatal("run succeeded with no configuration")
	}
}
