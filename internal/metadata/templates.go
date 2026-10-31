package metadata

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// templateFS holds the seed definitions (P3-06). They are compiled into the
// binary, so reading one is not a file-system call at run time.
//
//go:embed templates/*.json
var templateFS embed.FS

// ErrNoTemplate is returned by Template for a key with no seed definition.
var ErrNoTemplate = errors.New("metadata: no such template")

// TemplateKeys returns the keys of the seed definitions, sorted.
func TemplateKeys() []string {
	entries, _ := fs.ReadDir(templateFS, "templates") // embedded: cannot fail
	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		keys = append(keys, strings.TrimSuffix(e.Name(), ".json"))
	}
	slices.Sort(keys)
	return keys
}

// Template returns the seed definition with the given key. It is decoded
// strictly (unknown fields refused) but not compiled: pass it to Compile.
func Template(key string) (Definition, error) {
	if !keyPattern.MatchString(key) {
		return Definition{}, ErrNoTemplate
	}
	raw, err := templateFS.ReadFile(path.Join("templates", key+".json"))
	if err != nil {
		return Definition{}, ErrNoTemplate
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var d Definition
	if err := dec.Decode(&d); err != nil {
		return Definition{}, fmt.Errorf("metadata: template %s: %w", key, err)
	}
	return d, nil
}
