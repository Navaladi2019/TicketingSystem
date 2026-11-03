package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixturePins(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("testdata", "pins.md"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// checkpins is a gate: it exits 1 naming each problem on a tree out of step
// with docs/pins.md, 2 when it cannot check, 0 with "ok" otherwise.
func TestCLI(t *testing.T) {
	pins := fixturePins(t)
	data, err := os.ReadFile(pins)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		args    func(t *testing.T) []string
		code    int
		stdout  string
		wantErr []string
	}{
		{
			name:   "in step",
			args:   func(t *testing.T) []string { return []string{"-root", fixture(t, "ok"), "-pins", pins} },
			code:   0,
			stdout: "check-pins: ok\n",
		},
		{
			name: "pins relative to -root",
			args: func(t *testing.T) []string {
				dir := fixture(t, "ok")
				if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o700); err != nil {
					t.Fatal(err)
				}
				writeFile(t, dir, "docs/pins.md", string(data))
				return []string{"-root", dir}
			},
			code:   0,
			stdout: "check-pins: ok\n",
		},
		{
			name: "a forbidden require fails",
			args: func(t *testing.T) []string { return []string{"-root", fixture(t, "forbidden"), "-pins", pins} },
			code: 1,
			wantErr: []string{
				"check-pins: go.mod: requires github.com/coreos/go-oidc/v3 v3.17.0, which is forbidden (github.com/coreos/go-oidc)\n",
			},
		},
		{
			name: "drifted image tags fail, one line each",
			args: func(t *testing.T) []string { return []string{"-root", fixture(t, "image"), "-pins", pins} },
			code: 1,
			wantErr: []string{
				"check-pins: compose.yaml:",
				"image postgres:16.9, but docs/pins.md pins",
				"image alpine:latest has no exact tag",
			},
		},
		{
			name:    "pins file missing",
			args:    func(t *testing.T) []string { return []string{"-root", t.TempDir()} },
			code:    2,
			wantErr: []string{"check-pins: ", "pins.md"},
		},
		{
			name:    "unknown flag",
			args:    func(*testing.T) []string { return []string{"-bogus"} },
			code:    2,
			wantErr: []string{"flag provided but not defined: -bogus"},
		},
		{
			name:    "help",
			args:    func(*testing.T) []string { return []string{"-h"} },
			code:    0,
			wantErr: []string{"-pins string"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := cli(tc.args(t), &stdout, &stderr)
			if code != tc.code {
				t.Errorf("exit %d, want %d\nstdout:\n%s\nstderr:\n%s", code, tc.code, stdout.String(), stderr.String())
			}
			if stdout.String() != tc.stdout {
				t.Errorf("stdout %q, want %q", stdout.String(), tc.stdout)
			}
			for _, w := range tc.wantErr {
				if !strings.Contains(stderr.String(), w) {
					t.Errorf("stderr lacks %q\n%s", w, stderr.String())
				}
			}
			if len(tc.wantErr) == 0 && stderr.Len() != 0 {
				t.Errorf("stderr %q, want nothing", stderr.String())
			}
		})
	}
}

func TestCheckErrors(t *testing.T) {
	pins := fixturePins(t)
	t.Run("root missing", func(t *testing.T) {
		if _, err := check(filepath.Join(t.TempDir(), "gone"), pins); err == nil {
			t.Fatal("no error")
		}
	})
	t.Run(".env.example unreadable", func(t *testing.T) {
		dir := fixture(t, "ok")
		if err := os.Mkdir(filepath.Join(dir, ".env.example"), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := check(dir, pins); err == nil || !strings.Contains(err.Error(), ".env.example") {
			t.Fatalf("err %v, want the .env.example read error", err)
		}
	})
	t.Run("pins missing", func(t *testing.T) {
		if _, err := check(t.TempDir(), filepath.Join(t.TempDir(), "pins.md")); err == nil {
			t.Fatal("no error")
		}
	})
	t.Run("a Go file that cannot be read", func(t *testing.T) {
		dir := fixture(t, "ok")
		if err := os.Symlink(filepath.Join(dir, "gone.go"), filepath.Join(dir, "dangling.go")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, err := check(dir, pins); err == nil || !strings.Contains(err.Error(), "dangling.go") {
			t.Fatalf("err %v, want the dangling.go read error", err)
		}
	})
}

// Each reader fails on a file it cannot read, rather than passing it.
func TestReadErrors(t *testing.T) {
	p, err := parsePins(fixturePins(t))
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	for name, f := range map[string]func() error{
		"parseGoMod":     func() error { _, err := parseGoMod(missing); return err },
		"checkGoMod":     func() error { _, err := checkGoMod(missing, "go.mod", p); return err },
		"sumModules":     func() error { _, err := sumModules(missing); return err },
		"checkForbidden": func() error { _, err := checkForbidden(missing, "go.sum", p, sumModules); return err },
		"checkImages":    func() error { _, err := checkImages(missing, "Dockerfile", p, fromRe, false); return err },
		"checkLiterals":  func() error { _, err := checkLiterals(missing, "x.go", p); return err },
		"parsePins":      func() error { _, err := parsePins(missing); return err },
	} {
		if err := f(); err == nil {
			t.Errorf("%s on a missing file: no error", name)
		}
	}
}

func TestParsePins(t *testing.T) {
	const modules = "## Go modules\n\n| Component | Version |\n| --- | --- |\n| `a.com/m` | v1.0.0 |\n"
	const images = "## Images\n\n| Component | Version |\n| --- | --- |\n| `postgres` | 16.10 |\n| `short` |\n"
	const forbidden = "## Forbidden\n\n| Module | Ruling |\n| --- | --- |\n| `bad.com/x` | no |\n"
	for _, tc := range []struct {
		name, src, wantErr string
	}{
		{name: "every table", src: "# Pins\n\n" + images + "\n## Notes\n\n| `ignored` | x |\n\n" + modules + forbidden},
		{name: "no Images table", src: modules + forbidden, wantErr: "no Go modules or Images table found"},
		{name: "no Go modules table", src: images + forbidden, wantErr: "no Go modules or Images table found"},
		{name: "no Forbidden table", src: images + modules, wantErr: "no ## Forbidden table found"},
		{name: "a line past the scanner limit", src: images + modules + forbidden + strings.Repeat("x", 70*1024) + "\n", wantErr: "token too long"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pins.md")
			if err := os.WriteFile(path, []byte(tc.src), 0o600); err != nil {
				t.Fatal(err)
			}
			p, err := parsePins(path)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(p.images) != 1 || strings.TrimSpace(p.images["postgres"]) != "16.10" {
				t.Errorf("images = %q, want only postgres (a one-cell row is not a pin)", p.images)
			}
			if len(p.modules) != 1 || strings.TrimSpace(p.modules["a.com/m"]) != "v1.0.0" {
				t.Errorf("modules = %q", p.modules)
			}
			if strings.Join(p.forbidden, ",") != "bad.com/x" {
				t.Errorf("forbidden = %q", p.forbidden)
			}
			if !p.literalRe.MatchString("run postgres:16.10") || p.literalRe.MatchString("host:5432") {
				t.Errorf("literalRe %s", p.literalRe)
			}
		})
	}
}

func TestParseEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.example")
	src := "# comment\n\nexport A=1\nB = \"two\"\nC='three'\nNOEQUALS\nD=\"unbalanced'\nE=\n"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	env, err := parseEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"A": "1", "B": "two", "C": "three", "D": `"unbalanced'`, "E": ""}
	if len(env) != len(want) {
		t.Errorf("env = %q, want %q", env, want)
	}
	for k, v := range want {
		if got, ok := env[k]; !ok || got != v {
			t.Errorf("env[%s] = %q (set %v), want %q", k, got, ok, v)
		}
	}
	if env, err := parseEnv(filepath.Join(t.TempDir(), "missing")); err != nil || len(env) != 0 {
		t.Errorf("missing file: env %q, err %v; want empty, nil", env, err)
	}
}

func TestResolveRefDollarEscape(t *testing.T) {
	if got, err := resolveRef("nava/x:a$$b", nil); err != nil || got != "nava/x:a$b" {
		t.Errorf("resolveRef($$) = %q, %v; want nava/x:a$b", got, err)
	}
}

// A Dockerfile FROM that names a build ARG is skipped; the next FROM is still
// checked.
func TestDockerfileArgFromSkipped(t *testing.T) {
	p, err := parsePins(fixturePins(t))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "Dockerfile")
	if err := os.WriteFile(path, []byte("ARG BASE=postgres:1\nFROM ${BASE}\nFROM postgres:16.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := checkImages(path, "Dockerfile", p, fromRe, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := `Dockerfile:3: image postgres:16.9, but docs/pins.md pins "16.10"`; strings.Join(got, "\n") != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCompareSemver(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"v1.2.3", "v1.2.3", 0},
		{"v1.2.3-rc1", "v1.2.3", 0},
		{"1.2.3+build", "v1.2.3", 0},
		{"v1.2.3", "v1.2.4", -1},
		{"v1.10.0", "v1.9.9", 1},
		{"v2", "v1.99.99", 1},
	} {
		if got := compareSemver(tc.a, tc.b); got != tc.want {
			t.Errorf("compareSemver(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
