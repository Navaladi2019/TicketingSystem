package main

// TICK-909: -images and -areas end to end through cli, against a throwaway
// repository, so the git plumbing the stack job's image mode and area list
// depend on (merge base, a file's base side, a file missing on one side) is
// exercised, not only the classifiers.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	cliCompose = "services:\n  api:\n    build: ./api\n    environment:\n      A: \"1\"\n"
	cliOverlay = "services:\n  worker:\n    build:\n      context: ./worker\n"
)

func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut strings.Builder
	code = cli(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func writeFile(t *testing.T, name, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A compose change outside every build: block, and a new overlay with none
// (absent on the base side), keep the thin images; -v says why.
func TestCLIImagesThin(t *testing.T) {
	tempRepo(t, map[string]string{"compose.yaml": cliCompose, "deploy/compose/old.yaml": cliOverlay})
	writeFile(t, "compose.yaml", strings.Replace(cliCompose, `A: "1"`, `A: "2"`, 1))
	writeFile(t, "deploy/compose/new.yaml", "services:\n  extra:\n    image: busybox\n")
	code, stdout, stderr := runCLI(t, "-base", "HEAD", "-images", "-v")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if stdout != "thin\n" {
		t.Errorf("stdout %q, want thin", stdout)
	}
	if !strings.HasPrefix(stderr, "cichanged: images thin: ") {
		t.Errorf("stderr %q, want the thin reason", stderr)
	}
}

// A changed build: block, and an overlay deleted from the tree whose base
// side had one, both need the real images.
func TestCLIImagesReal(t *testing.T) {
	for _, c := range []struct {
		name, file string
		change     func(t *testing.T)
	}{
		{"build block changed", "compose.yaml", func(t *testing.T) {
			writeFile(t, "compose.yaml", strings.Replace(cliCompose, "./api", "./api2", 1))
		}},
		{"overlay deleted", "deploy/compose/old.yaml", func(t *testing.T) {
			if err := os.Remove(filepath.FromSlash("deploy/compose/old.yaml")); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tempRepo(t, map[string]string{"compose.yaml": cliCompose, "deploy/compose/old.yaml": cliOverlay})
			c.change(t)
			code, stdout, stderr := runCLI(t, "-base", "HEAD", "-images", "-v")
			if code != 0 {
				t.Fatalf("exit %d, stderr %q", code, stderr)
			}
			if stdout != "real\n" {
				t.Errorf("stdout %q, want real", stdout)
			}
			if want := "cichanged: " + c.file + ": images real (a compose build: block changed)\n"; stderr != want {
				t.Errorf("stderr %q, want %q", stderr, want)
			}
		})
	}
}

// -areas prints the areas of the change from a saved graph, and with -v one
// reason per file.
func TestCLIAreas(t *testing.T) {
	tempRepo(t, map[string]string{"README.md": "x\n"})
	g := filepath.Join(t.TempDir(), "graph.json")
	writeFile(t, g, `{"packages":["a"]}`)
	t.Setenv(graphEnv, g)
	writeFile(t, "db/migrations/0001.sql", "SELECT 1;\n")
	code, stdout, stderr := runCLI(t, "-base", "HEAD", "-areas", "-v")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if stdout != "db\n" {
		t.Errorf("stdout %q, want db", stdout)
	}
	if !strings.HasPrefix(stderr, "cichanged: db/migrations/0001.sql: areas db (") || strings.Count(stderr, "\n") != 1 {
		t.Errorf("stderr %q, want one reason, for the migration", stderr)
	}
}

// -areas and -images cannot be combined, and -images with a base git does
// not know fails like every other mode: exit 2, nothing on stdout.
func TestCLIModeRefusals(t *testing.T) {
	tempRepo(t, map[string]string{"README.md": "x\n"})
	for _, c := range []struct {
		name   string
		args   []string
		stderr string
	}{
		{"exclusive modes", []string{"-base", "HEAD", "-areas", "-images"}, "cichanged: -areas and -images are exclusive\n"},
		{"images unknown base", []string{"-base", "no-such-ref", "-images"}, "cichanged: "},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, c.args...)
			if code != 2 {
				t.Errorf("exit %d, want 2", code)
			}
			if stdout != "" {
				t.Errorf("stdout %q, want nothing", stdout)
			}
			if !strings.HasPrefix(stderr, c.stderr) {
				t.Errorf("stderr %q, want prefix %q", stderr, c.stderr)
			}
		})
	}
}
