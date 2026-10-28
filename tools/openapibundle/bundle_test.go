package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const rootSpec = `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /api/v1/things:
    $ref: "paths/things.yaml#/paths/~1api~1v1~1things"
components:
  schemas:
    Shared: {type: string}
`

const thingsSpec = `paths:
  /api/v1/things:
    get:
      operationId: listThings
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
components:
  schemas:
    Thing:
      type: object
      properties:
        s:
          $ref: "../openapi.yaml#/components/schemas/Shared"
`

func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "openapi.yaml")
}

func TestBundleMergesGroupFiles(t *testing.T) {
	t.Parallel()
	out, err := Bundle(tree(t, map[string]string{"openapi.yaml": rootSpec, "paths/things.yaml": thingsSpec}))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"operationId: listThings", "Thing:", "Shared:", `$ref: "#/components/schemas/Shared"`} {
		if !strings.Contains(s, want) {
			t.Errorf("bundle lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "../openapi.yaml") {
		t.Error("bundle keeps a ref to the root file")
	}
}

func TestBundleFailures(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		files map[string]string
		want  string
	}{
		"wrong group": {
			map[string]string{"openapi.yaml": strings.Replace(rootSpec, "paths/things.yaml", "paths/other.yaml", 1), "paths/other.yaml": thingsSpec},
			"belongs in paths/things.yaml",
		},
		"duplicate component": {
			map[string]string{"openapi.yaml": strings.Replace(rootSpec, "Shared: {type: string}", "Thing: {type: string}", 1), "paths/things.yaml": strings.Replace(thingsSpec, "../openapi.yaml#/components/schemas/Shared", "#/components/schemas/Thing", 1)},
			"already defined",
		},
		"unreferenced group file": {
			map[string]string{"openapi.yaml": rootSpec, "paths/things.yaml": thingsSpec, "paths/extra.yaml": "paths: {}\n"},
			"paths/extra.yaml is not referenced",
		},
		"dangling ref": {
			map[string]string{"openapi.yaml": rootSpec, "paths/things.yaml": strings.Replace(thingsSpec, "schemas/Thing\"", "schemas/Nope\"", 1)},
			"bundle does not load",
		},
		"inline path in root": {
			map[string]string{"openapi.yaml": strings.Replace(rootSpec, `$ref: "paths/things.yaml#/paths/~1api~1v1~1things"`, "get: {}", 1), "paths/things.yaml": thingsSpec},
			"keeps only a $ref",
		},
		"missing group file": {
			map[string]string{"openapi.yaml": rootSpec},
			"things.yaml",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := Bundle(tree(t, tc.files))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestRunWritesBundle(t *testing.T) {
	t.Parallel()
	root := tree(t, map[string]string{"openapi.yaml": rootSpec, "paths/things.yaml": thingsSpec})
	out := filepath.Join(filepath.Dir(root), "out.yaml")
	if code := run([]string{"-root", root, "-out", out}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if _, err := os.Stat(out); err != nil {
		t.Error(err)
	}
	if code := run([]string{"-root", filepath.Join(filepath.Dir(root), "nope.yaml"), "-out", out}); code != 1 {
		t.Errorf("missing root: exit %d, want 1", code)
	}
}
