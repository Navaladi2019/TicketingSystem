package main

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Image build modes for the stack job (TICK-628), printed by -images.
const (
	// imagesReal: make image-ci builds the real Dockerfiles, as main does.
	imagesReal = "real"
	// imagesThin: make ci-binaries, then make image-ci IMAGE_THIN=1
	// (deploy/compose/Dockerfile.ci over the host-built binaries).
	imagesThin = "thin"
)

// fileReader reads a file at one side of the change; a nil slice and no
// error is a file that does not exist there.
type fileReader func(file string) ([]byte, error)

// classifyImages returns real when a change touches what the real image
// build depends on and the thin CI images skip: a Dockerfile under deploy/,
// Dockerfile.ci itself, .dockerignore, go.mod or go.sum, or the build: block
// of a compose file service. Anything else keeps the thin images: the Go
// sources the binaries come from are built either way. AR-TICK-628-01 is the
// risk this rule bounds.
func classifyImages(files []string, base, head fileReader) (string, []string) {
	mode := imagesThin
	var reasons []string
	for _, f := range files {
		real, why := fileNeedsRealImages(f, base, head)
		if real {
			mode = imagesReal
			reasons = append(reasons, fmt.Sprintf("%s: images real (%s)", f, why))
		}
	}
	if mode == imagesThin {
		reasons = append(reasons, "images thin: no change to a Dockerfile, .dockerignore, go.mod, go.sum or a compose build: block")
	}
	return mode, reasons
}

func fileNeedsRealImages(f string, base, head fileReader) (bool, string) {
	switch {
	case f == "go.mod" || f == "go.sum":
		return true, "the module the real build stage downloads"
	case f == ".dockerignore":
		return true, "the build context of every image"
	case f == "deploy/compose/Dockerfile.ci":
		return true, "the thin CI images' own Dockerfile"
	case strings.HasPrefix(f, "deploy/") && path.Base(f) == "Dockerfile":
		return true, "a real Dockerfile"
	case isComposeFile(f):
		changed, err := buildBlocksDiffer(f, base, head)
		if err != nil {
			return true, "compose file not readable as YAML (" + err.Error() + "), so assume its build: blocks changed"
		}
		if changed {
			return true, "a compose build: block changed"
		}
	}
	return false, ""
}

// isComposeFile is compose.yaml or a deploy/compose/*.yaml overlay.
func isComposeFile(f string) bool {
	return f == "compose.yaml" || (path.Dir(f) == "deploy/compose" && path.Ext(f) == ".yaml")
}

// buildBlocksDiffer compares services.<name>.build between the two sides. A
// file missing on one side counts as having no build blocks there.
func buildBlocksDiffer(f string, base, head fileReader) (bool, error) {
	a, err := buildBlocks(f, base)
	if err != nil {
		return false, err
	}
	b, err := buildBlocks(f, head)
	if err != nil {
		return false, err
	}
	return !reflect.DeepEqual(a, b), nil
}

func buildBlocks(f string, read fileReader) (map[string]any, error) {
	data, err := read(f)
	if err != nil || data == nil {
		return map[string]any{}, err
	}
	var doc struct {
		Services map[string]map[string]any `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for name, svc := range doc.Services {
		if b, ok := svc["build"]; ok {
			out[name] = b
		}
	}
	return out, nil
}
