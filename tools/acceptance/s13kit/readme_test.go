package s13kit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadme fails if the README does not show, for every sample, the
// command, the exact stdout and the exit code, in one console block, as
// expected/<sample>.txt has them. The README cannot drift from what
// TestSteps checks.
func TestReadme(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(kitDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	readme := string(b)
	for _, s := range samples() {
		want := readExpected(t, s.file)
		block := fmt.Sprintf("```console\n$ ../../../bin/nava audit verify %s; echo \"exit $?\"\n%sexit %d\n```\n",
			s.file, want.stdout, want.exit)
		if !strings.Contains(readme, block) {
			t.Errorf("README.md has no block for %s; want:\n%s", s.file, block)
		}
		row := fmt.Sprintf("| `%s` |", s.file)
		if !strings.Contains(readme, row) {
			t.Errorf("README.md sample table has no row for %s", s.file)
		}
	}
}
