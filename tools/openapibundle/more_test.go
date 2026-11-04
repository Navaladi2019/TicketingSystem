package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestBundleRejectsBadTrees(t *testing.T) {
	t.Parallel()
	ref := `$ref: "paths/things.yaml#/paths/~1api~1v1~1things"`
	cases := map[string]struct {
		files map[string]string
		want  string
	}{
		"root not yaml": {
			map[string]string{"openapi.yaml": "paths: [\n"},
			"openapi.yaml: yaml:",
		},
		"root is a list": {
			map[string]string{"openapi.yaml": "- a\n- b\n"},
			"openapi.yaml: not a YAML mapping",
		},
		"root empty": {
			map[string]string{"openapi.yaml": ""},
			"openapi.yaml: not a YAML mapping",
		},
		"root has no paths": {
			map[string]string{"openapi.yaml": "openapi: 3.1.0\ninfo: {title: t, version: \"1\"}\n"},
			"root has no paths mapping",
		},
		"root paths is a list": {
			map[string]string{"openapi.yaml": "openapi: 3.1.0\npaths: [a]\n"},
			"root has no paths mapping",
		},
		"ref outside paths dir": {
			map[string]string{"openapi.yaml": strings.Replace(rootSpec, ref, `$ref: "things.yaml#/paths/~1api~1v1~1things"`, 1), "paths/things.yaml": thingsSpec},
			`path /api/v1/things: $ref "things.yaml#/paths/~1api~1v1~1things" is not paths/<group>.yaml#/pointer`,
		},
		"ref to a non-yaml file": {
			map[string]string{"openapi.yaml": strings.Replace(rootSpec, ref, `$ref: "paths/things.json#/paths/~1api~1v1~1things"`, 1), "paths/things.yaml": thingsSpec},
			"is not paths/<group>.yaml#/pointer",
		},
		"route outside /api/v1": {
			map[string]string{"openapi.yaml": strings.Replace(rootSpec, "/api/v1/things:", "/things:", 1), "paths/things.yaml": thingsSpec},
			"path /things belongs in paths/",
		},
		"group file not yaml": {
			map[string]string{"openapi.yaml": rootSpec, "paths/things.yaml": "paths: [\n"},
			"things.yaml: yaml:",
		},
		"pointer does not resolve": {
			map[string]string{"openapi.yaml": strings.Replace(rootSpec, "~1things\"", "~1nope\"", 1), "paths/things.yaml": thingsSpec},
			`path /api/v1/things: paths/things.yaml: pointer "/paths/~1api~1v1~1nope" does not resolve`,
		},
		"pointer to a scalar": {
			map[string]string{"openapi.yaml": strings.Replace(rootSpec, ref, `$ref: "paths/things.yaml#/paths/~1api~1v1~1things/get/operationId"`, 1), "paths/things.yaml": thingsSpec},
			`pointer "/paths/~1api~1v1~1things/get/operationId" is not a path item`,
		},
		"duplicate across group files": {
			map[string]string{
				"openapi.yaml":      strings.Replace(rootSpec, "components:", "  /api/v1/more:\n    $ref: \"paths/more.yaml#/paths/~1api~1v1~1more\"\ncomponents:", 1),
				"paths/things.yaml": thingsSpec,
				"paths/more.yaml":   strings.ReplaceAll(strings.Replace(thingsSpec, "listThings", "listMore", 1), "/api/v1/things", "/api/v1/more"),
			},
			"paths/things.yaml: components.schemas.Thing is already defined",
		},
		"bundle fails validation": {
			map[string]string{"openapi.yaml": rootSpec, "paths/things.yaml": strings.Replace(thingsSpec, "          description: ok\n", "", 1)},
			"bundle is not valid OpenAPI",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out, err := Bundle(tree(t, tc.files))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to contain %q", err, tc.want)
			}
			if out != nil {
				t.Errorf("got a bundle on error")
			}
		})
	}
}

// A root without components gets one holding the group files' components; a
// group file may add a section the root lacks; a group file with no paths key
// (its item addressed by another pointer) passes the unused-paths check.
func TestBundleCreatesMissingComponentSections(t *testing.T) {
	t.Parallel()
	root := `openapi: 3.1.0
info: {title: t, version: "1"}
paths:
  /api/v1/things:
    $ref: "paths/things.yaml#/items/list"
`
	group := `items:
  list:
    get:
      parameters:
        - $ref: "#/components/parameters/Limit"
      responses:
        "200": {description: ok}
components:
  parameters:
    Limit: {name: limit, in: query, schema: {type: integer}}
`
	out, err := Bundle(tree(t, map[string]string{"openapi.yaml": root, "paths/things.yaml": group}))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths      map[string]map[string]any `yaml:"paths"`
		Components map[string]map[string]any `yaml:"components"`
	}
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Components["parameters"]["Limit"]; !ok {
		t.Errorf("components.parameters.Limit missing:\n%s", out)
	}
	if _, ok := doc.Paths["/api/v1/things"]["get"]; !ok {
		t.Errorf("path item not inlined:\n%s", out)
	}
	if !strings.HasPrefix(string(out), header) {
		t.Errorf("bundle lacks the generated header")
	}
}

func TestBundleRejectsBadGlobDir(t *testing.T) {
	t.Parallel()
	// A '[' in the directory name makes the paths/*.yaml glob malformed; that
	// must be an error, not a silently skipped unreferenced-file check.
	dir := filepath.Join(t.TempDir(), "api[")
	for name, body := range map[string]string{"openapi.yaml": rootSpec, "paths/things.yaml": thingsSpec} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Bundle(filepath.Join(dir, "openapi.yaml")); err == nil || !strings.Contains(err.Error(), "syntax error in pattern") {
		t.Fatalf("err = %v, want a bad-pattern error", err)
	}
}

func TestGetOnNil(t *testing.T) {
	t.Parallel()
	if get(nil, "x") != nil {
		t.Error("get(nil) != nil")
	}
}

func TestRunExitCodes(t *testing.T) {
	t.Parallel()
	root := tree(t, map[string]string{"openapi.yaml": rootSpec, "paths/things.yaml": thingsSpec})
	dir := filepath.Dir(root)
	out := filepath.Join(dir, "out.yaml")
	if code := run([]string{"-root", root, "-out", out}); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	want, err := Bundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(out); err != nil || string(got) != string(want) {
		t.Errorf("written bundle differs from Bundle(): err %v", err)
	}
	cases := map[string]struct {
		args []string
		code int
	}{
		"unknown flag":       {[]string{"-nope"}, 2},
		"invalid tree":       {[]string{"-root", filepath.Join(dir, "nope.yaml"), "-out", out}, 1},
		"unwritable out":     {[]string{"-root", root, "-out", filepath.Join(dir, "no", "such", "dir", "out.yaml")}, 1},
		"out is a directory": {[]string{"-root", root, "-out", dir}, 1},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if code := run(c.args); code != c.code {
				t.Errorf("exit %d, want %d", code, c.code)
			}
		})
	}
}

// A group file without components adds none and keeps the root's.
func TestBundleGroupWithoutComponents(t *testing.T) {
	t.Parallel()
	group := "paths:\n  /api/v1/things:\n    get:\n      responses:\n        \"200\": {description: ok}\n"
	out, err := Bundle(tree(t, map[string]string{"openapi.yaml": rootSpec, "paths/things.yaml": group}))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Components map[string]map[string]any `yaml:"components"`
	}
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Components) != 1 || len(doc.Components["schemas"]) != 1 || doc.Components["schemas"]["Shared"] == nil {
		t.Errorf("components = %v, want only schemas.Shared", doc.Components)
	}
}
