package main

import (
	"errors"
	"strings"
	"testing"
)

// mapReader serves files from a map; a missing key does not exist.
func mapReader(m map[string]string) fileReader {
	return func(f string) ([]byte, error) {
		s, ok := m[f]
		if !ok {
			return nil, nil
		}
		return []byte(s), nil
	}
}

const composeBase = `services:
  nava-core:
    build:
      context: .
      dockerfile: deploy/compose/nava-core/Dockerfile
    image: nava/nava-core:dev
    environment:
      A: "1"
  postgres:
    image: example.invalid/db
`

// TICK-628: a change that touches the real image build runs the real
// Dockerfiles; every other change keeps the thin CI images.
func TestClassifyImages(t *testing.T) {
	for _, tc := range []struct {
		name       string
		files      []string
		base, head map[string]string
		want       string
		why        string
	}{
		{name: "Go source only", files: []string{"internal/identity/x.go", "cmd/nava-core/main.go"}, want: imagesThin},
		{name: "docs only", files: []string{"docs/quality/ci-pipeline.md"}, want: imagesThin},
		{name: "no change", want: imagesThin},
		{name: "a real Dockerfile", files: []string{"deploy/compose/nava-core/Dockerfile"}, want: imagesReal, why: "a real Dockerfile"},
		{name: "a nested real Dockerfile", files: []string{"deploy/compose/keycloak/Dockerfile"}, want: imagesReal},
		{name: "Dockerfile.ci", files: []string{"deploy/compose/Dockerfile.ci"}, want: imagesReal},
		{name: ".dockerignore", files: []string{".dockerignore"}, want: imagesReal},
		{name: "go.mod", files: []string{"go.mod"}, want: imagesReal},
		{name: "go.sum among Go files", files: []string{"cmd/nava-core/main.go", "go.sum"}, want: imagesReal},
		{
			name:  "compose build: block changed",
			files: []string{"compose.yaml"},
			base:  map[string]string{"compose.yaml": composeBase},
			head:  map[string]string{"compose.yaml": strings.Replace(composeBase, "nava-core/Dockerfile", "nava-core/Dockerfile.x", 1)},
			want:  imagesReal, why: "build: block changed",
		},
		{
			name:  "compose environment changed, build: blocks the same",
			files: []string{"compose.yaml"},
			base:  map[string]string{"compose.yaml": composeBase},
			head:  map[string]string{"compose.yaml": strings.Replace(composeBase, `A: "1"`, `A: "2"`, 1)},
			want:  imagesThin,
		},
		{
			name:  "overlay gains a build: block",
			files: []string{"deploy/compose/compose.test.yaml"},
			base:  map[string]string{"deploy/compose/compose.test.yaml": "services:\n  x:\n    image: a\n"},
			head:  map[string]string{"deploy/compose/compose.test.yaml": "services:\n  x:\n    image: a\n    build:\n      context: .\n"},
			want:  imagesReal,
		},
		{
			name:  "new overlay without a build: block",
			files: []string{"deploy/compose/compose.new.yaml"},
			head:  map[string]string{"deploy/compose/compose.new.yaml": "services:\n  x:\n    image: a\n"},
			want:  imagesThin,
		},
		{
			name:  "compose file not YAML",
			files: []string{"compose.yaml"},
			base:  map[string]string{"compose.yaml": composeBase},
			head:  map[string]string{"compose.yaml": "services: [\n"},
			want:  imagesReal, why: "not readable as YAML",
		},
		{
			// A YAML file under a service's own directory is not a compose file.
			name:  "a service config file",
			files: []string{"deploy/compose/prometheus/prometheus.yaml"},
			want:  imagesThin,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, reasons := classifyImages(tc.files, mapReader(tc.base), mapReader(tc.head))
			if got != tc.want {
				t.Errorf("classifyImages(%q) = %s, want %s; reasons %q", tc.files, got, tc.want, reasons)
			}
			if tc.why != "" && !strings.Contains(strings.Join(reasons, "\n"), tc.why) {
				t.Errorf("reasons %q lack %q", reasons, tc.why)
			}
		})
	}
}

// A compose file that cannot be read on one side is treated as changed.
func TestClassifyImagesReadError(t *testing.T) {
	fail := func(string) ([]byte, error) { return nil, errors.New("boom") }
	if got, _ := classifyImages([]string{"compose.yaml"}, fail, mapReader(nil)); got != imagesReal {
		t.Errorf("got %s, want %s", got, imagesReal)
	}
}

func TestCLIImagesAndAreasExclusive(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := cli([]string{"-images", "-areas"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "exclusive") {
		t.Errorf("stderr %q", stderr.String())
	}
}
