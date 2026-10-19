// Command reviewpaths flags a change that needs the D-P10 Security review or
// CTO sign-off (PLAN-001, T05), from docs/delivery/review-paths.yaml and the
// diff against a base ref. It prints the two summary lines, then the files
// behind each "yes", and appends the same to $GITHUB_STEP_SUMMARY when set.
// It never fails the build: it only flags.
//
// Usage: reviewpaths [-base origin/main] [-paths docs/delivery/review-paths.yaml]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictyaml"
)

func main() {
	base := flag.String("base", "origin/main", "base ref of the change")
	pathsFile := flag.String("paths", "docs/delivery/review-paths.yaml", "review path list")
	flag.Parse()

	if err := run(*base, *pathsFile); err != nil {
		fmt.Fprintln(os.Stderr, "review-paths:", err)
		os.Exit(2)
	}
}

func run(base, pathsFile string) error {
	data, err := os.ReadFile(pathsFile) //nolint:gosec // a path the caller chose
	if err != nil {
		return err
	}
	var cfg Config
	if err := strictyaml.Decode(data, &cfg); err != nil {
		return fmt.Errorf("%s: %w", pathsFile, err)
	}
	nameStatus, err := git("diff", "--no-renames", "--name-status", base+"...HEAD")
	if err != nil {
		return err
	}
	changes := parseNameStatus(nameStatus)
	added := map[string][]string{}
	for _, c := range changes {
		if c.Status != "D" && cfg.isIsolationTest(c.Path) {
			diff, err := git("diff", "-U0", base+"...HEAD", "--", c.Path)
			if err != nil {
				return err
			}
			added[c.Path] = addedLines(diff)
		}
	}
	r := Evaluate(cfg, changes, added)
	out := r.String()
	fmt.Print(out)
	if summary := os.Getenv("GITHUB_STEP_SUMMARY"); summary != "" {
		f, err := os.OpenFile(summary, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600) //nolint:gosec // set by GitHub Actions
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		if _, err := fmt.Fprintf(f, "### Review paths (D-P10)\n\n```\n%s```\n", out); err != nil {
			return err
		}
	}
	return nil
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...) //nolint:gosec,noctx // fixed git subcommands, run to completion
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return string(out), nil
}

// Config is docs/delivery/review-paths.yaml.
type Config struct {
	Security   []string `yaml:"security"`
	CTOSignoff struct {
		Paths          []string `yaml:"paths"`
		IsolationTests []string `yaml:"isolation_tests"`
	} `yaml:"cto_signoff"`
}

func (c Config) isIsolationTest(p string) bool { return matchAny(c.CTOSignoff.IsolationTests, p) }

// Change is one line of `git diff --name-status`.
type Change struct {
	Status string // A, M, D, ...
	Path   string
}

func parseNameStatus(s string) []Change {
	var out []Change
	for line := range strings.SplitSeq(s, "\n") {
		f := strings.Split(strings.TrimSpace(line), "\t")
		if len(f) >= 2 {
			out = append(out, Change{Status: f[0][:1], Path: f[len(f)-1]})
		}
	}
	return out
}

// addedLines returns the "+" lines of a unified diff, without the marker.
func addedLines(diff string) []string {
	var out []string
	for line := range strings.SplitSeq(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			out = append(out, line[1:])
		}
	}
	return out
}

// Result is the verdict and the files behind it.
type Result struct {
	Security []string
	CTO      []string
}

// String renders the two summary lines, then the reasons.
func (r Result) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "security review required: %s\n", yesNo(len(r.Security) > 0))
	fmt.Fprintf(&b, "CTO sign-off required: %s\n", yesNo(len(r.CTO) > 0))
	for _, p := range r.Security {
		fmt.Fprintf(&b, "  security: %s\n", p)
	}
	for _, p := range r.CTO {
		fmt.Fprintf(&b, "  CTO: %s\n", p)
	}
	return b.String()
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// skipRe matches an added line that can take a test out of the run: a skip,
// a -short guard, or a build constraint (`//go:build ignore` and the like).
var skipRe = regexp.MustCompile(`\bt\.(Skip|SkipNow|Skipf)\(|testing\.Short\(\)|^\s*//\s*(go:build|\+build)\b`)

// Evaluate applies the path list to a change. added holds the added lines of
// each isolation test file that was not deleted.
func Evaluate(cfg Config, changes []Change, added map[string][]string) Result {
	var r Result
	for _, c := range changes {
		if matchAny(cfg.Security, c.Path) {
			r.Security = append(r.Security, c.Path)
		}
		switch {
		case matchAny(cfg.CTOSignoff.Paths, c.Path):
			r.CTO = append(r.CTO, c.Path+" (changed)")
		case cfg.isIsolationTest(c.Path) && c.Status == "D":
			r.CTO = append(r.CTO, c.Path+" (isolation test removed)")
		case cfg.isIsolationTest(c.Path):
			for _, l := range added[c.Path] {
				if skipRe.MatchString(l) {
					r.CTO = append(r.CTO, c.Path+" (isolation test skipped or excluded)")
					break
				}
			}
		}
	}
	return r
}

func matchAny(patterns []string, p string) bool {
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
