// Package reviewcfg is docs/delivery/review-paths.yaml and its pattern
// rules, shared by the CI tools that read it (reviewpaths, suiteran), so they
// agree on which files a pattern names.
package reviewcfg

import (
	"fmt"
	"os"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictyaml"
)

// Load reads and strictly decodes the path list at file.
func Load(file string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(file) //nolint:gosec // a path the caller chose
	if err != nil {
		return cfg, err
	}
	if err := strictyaml.Decode(data, &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", file, err)
	}
	return cfg, nil
}

// Config is docs/delivery/review-paths.yaml.
type Config struct {
	Security   Entries `yaml:"security"`
	CTOSignoff struct {
		Paths          Entries  `yaml:"paths"`
		IsolationTests []string `yaml:"isolation_tests"`
	} `yaml:"cto_signoff"`
}

// IsIsolationTest reports whether p is a registered isolation test file.
func (c Config) IsIsolationTest(p string) bool { return MatchAny(c.CTOSignoff.IsolationTests, p) }

// Entry is one pattern of a path list: a bare string, or a mapping
//
//   - path: cmd/nava-relay/
//     planned: "PLAN-001 T30b"
//
// for a pattern that matches no tracked file yet. Planned names the task that
// creates it (TestPathsMatchTrackedFiles holds both forms to that).
type Entry struct {
	Path    string
	Planned string
}

// UnmarshalYAML accepts both forms. node.Decode does not inherit the
// decoder's KnownFields, so the mapping keys are checked here.
func (e *Entry) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		return node.Decode(&e.Path)
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: a path entry is a string or a {path, planned} mapping", node.Line)
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		var dst *string
		switch k.Value {
		case "path":
			dst = &e.Path
		case "planned":
			dst = &e.Planned
		default:
			return fmt.Errorf("line %d: field %s not found in path entry", k.Line, k.Value)
		}
		if err := v.Decode(dst); err != nil {
			return err
		}
	}
	if e.Path == "" {
		return fmt.Errorf("line %d: path entry has no path", node.Line)
	}
	return nil
}

// Entries is a path list.
type Entries []Entry

// Patterns returns the patterns, planned or not: a planned path is flagged
// from the PR that creates it.
func (es Entries) Patterns() []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Path
	}
	return out
}

// MatchAny reports whether the repo-relative, slash-separated path p matches
// one of the patterns, by the rules at the top of review-paths.yaml.
func MatchAny(patterns []string, p string) bool {
	for _, pat := range patterns {
		if rest, ok := strings.CutPrefix(pat, "**/"); ok {
			if matchAnyDepth(rest, p) {
				return true
			}
			continue
		}
		if strings.HasSuffix(pat, "/") {
			if strings.HasPrefix(p, pat) {
				return true
			}
			continue
		}
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
	}
	return false
}

// matchAnyDepth matches pat against p and every trailing run of p's segments,
// so "*isolation*_test.go" matches the file name at any depth.
func matchAnyDepth(pat, p string) bool {
	for {
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
		i := strings.Index(p, "/")
		if i < 0 {
			return false
		}
		p = p[i+1:]
	}
}
