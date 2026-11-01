package metadata

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// TestCompilerLoaderDeniesEverything is the backstop behind walkSchema: even
// a reference walkSchema missed cannot load anything. The file below exists
// and holds a valid schema, so v6's default FileLoader would load it; the
// compiler newSchemaCompiler returns refuses it, and refuses http too
// without dialling.
func TestCompilerLoaderDeniesEverything(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ext.json")
	if err := os.WriteFile(path, []byte(`{"type":"string"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fileURL := "file:///" + filepath.ToSlash(path)
	if filepath.IsAbs(path) && path[0] == '/' {
		fileURL = "file://" + path
	}

	// Control: the library default really does read the file.
	def := jsonschema.NewCompiler()
	if err := def.AddResource(schemaURL, map[string]any{"$ref": fileURL}); err != nil {
		t.Fatal(err)
	}
	if _, err := def.Compile(schemaURL); err != nil {
		t.Fatalf("control: default compiler did not load the file: %v", err)
	}

	for _, ref := range []string{fileURL, "http://127.0.0.1:1/s.json", "urn:x"} {
		c := newSchemaCompiler()
		if err := c.AddResource(schemaURL, map[string]any{"$ref": ref}); err != nil {
			t.Fatal(err)
		}
		_, err := c.Compile(schemaURL)
		if !errors.Is(err, errLoadDenied) {
			var lue *jsonschema.LoadURLError
			if !errors.As(err, &lue) || !errors.Is(lue.Err, errLoadDenied) {
				t.Fatalf("%s: err = %v, want the deny-all loader's refusal", ref, err)
			}
		}
		var bad problems
		schemaErrorPaths(err, &bad)
		if len(bad) != 1 || bad[0] != "/schema" {
			t.Fatalf("%s: paths = %q, want [/schema]", ref, bad)
		}
	}
}

// TestSharedCompilerWouldLeak shows why Compile never shares a compiler. A
// compiler caches every resource and compiled schema it has seen: if tenant
// A's schema went in first, a shared compiler refuses B's under the same URL
// and Compile(url) hands B tenant A's schema. (v6.0.3 resolves $id only
// within the root that declares it, so the "$id":"urn:x" route is closed in
// this version too; the per-call compiler does not depend on that detail.)
// A fresh compiler per call has nothing from A.
func TestSharedCompilerWouldLeak(t *testing.T) {
	a := map[string]any{"$id": "urn:x", "type": "string"}
	b := map[string]any{"$ref": "urn:x"}

	shared := newSchemaCompiler()
	if err := shared.AddResource(schemaURL, a); err != nil {
		t.Fatal(err)
	}
	schA, err := shared.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.AddResource(schemaURL, b); err == nil {
		t.Fatal("expected the shared compiler to refuse B's resource under A's URL")
	}
	if schB, err := shared.Compile(schemaURL); err != nil || schB != schA {
		t.Fatalf("expected the shared compiler to return A's schema for B (the hazard), got %v, %v", schB, err)
	}

	fresh := newSchemaCompiler()
	if err := fresh.AddResource(schemaURL, b); err != nil {
		t.Fatal(err)
	}
	if _, err := fresh.Compile(schemaURL); err == nil {
		t.Fatal("a fresh compiler resolved B's ref: state leaked between compilers")
	}
}

func TestFragmentEscaping(t *testing.T) {
	for name, want := range map[string]string{
		"plain":  "/properties/plain",
		"a/b":    "/properties/a~1b",
		"t~x":    "/properties/t~0x",
		"sp ace": "/properties/sp%20ace",
		"pct%":   "/properties/pct%25",
		"hash#":  "/properties/hash%23",
	} {
		if got := fragment("properties", name); got != want {
			t.Fatalf("fragment(%q) = %q, want %q", name, got, want)
		}
	}
}
