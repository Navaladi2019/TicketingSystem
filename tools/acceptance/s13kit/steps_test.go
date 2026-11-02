package s13kit_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// expected is one docs/acceptance/S13/expected/<sample>.txt: the exit code
// `nava audit verify` must give and its exact stdout.
type expected struct {
	exit   int
	stdout string
}

func readExpected(t *testing.T, sampleFile string) expected {
	t.Helper()
	p := filepath.Join(kitDir, "expected", strings.TrimSuffix(sampleFile, ".jsonl")+".txt")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("%s: %v", sampleFile, err)
	}
	first, rest, ok := strings.Cut(string(b), "\n")
	code, err := strconv.Atoi(strings.TrimPrefix(first, "exit "))
	if !ok || !strings.HasPrefix(first, "exit ") || err != nil {
		t.Fatalf("%s: first line %q, want \"exit N\"", p, first)
	}
	return expected{exit: code, stdout: rest}
}

// buildNava builds cmd/nava once, the binary the human runs after
// `task build`.
func buildNava(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "nava")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "./cmd/nava")
	cmd.Dir = "../../.."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/nava: %v\n%s", err, out)
	}
	return bin
}

// TestSteps runs every sample through the real binary, from the kit
// directory as the README does, and compares stdout and the exit code.
func TestSteps(t *testing.T) {
	if testing.Short() {
		t.Skip("builds cmd/nava")
	}
	nava := buildNava(t)
	codes := map[int][]string{}
	for _, s := range samples() {
		t.Run(s.file, func(t *testing.T) {
			want := readExpected(t, s.file)
			cmd := exec.CommandContext(t.Context(), nava, "audit", "verify", s.file)
			cmd.Dir = kitDir
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			code := 0
			if err := cmd.Run(); err != nil {
				var ee *exec.ExitError
				if !errors.As(err, &ee) {
					t.Fatalf("run nava: %v", err)
				}
				code = ee.ExitCode()
			}
			codes[code] = append(codes[code], s.file)
			if code != want.exit {
				t.Errorf("exit %d, want %d\nstdout: %s\nstderr: %s", code, want.exit, stdout.String(), stderr.String())
			}
			if stdout.String() != want.stdout {
				t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), want.stdout)
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr not empty: %s", stderr.String())
			}
		})
	}
	// The kit must show every verdict a human can meet, and "not verifiable:
	// stubs omitted" (4) must not share an exit code with tampering (1).
	for _, c := range []int{0, 1, 3, 4, 5} {
		if len(codes[c]) == 0 {
			t.Errorf("no sample exits %d", c)
		}
	}
}
