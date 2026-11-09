package compose

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TICK-628: pull request stack runs build the Go services' images from
// Dockerfile.ci over binaries built on the runner (make image-ci
// IMAGE_THIN=1). These tests keep those images the same as the real ones
// where it matters at run time, so the thin image cannot pass a case the real
// image would fail. What they cannot see, the real build stage itself, is
// AR-TICK-628-01; main builds the real Dockerfiles on every push.

const (
	ciDockerfile = "Dockerfile.ci" // in this directory
	ciBinDir     = ".ci-bin"       // make/stack.mk CI_BIN_DIR
)

// finalStage is what the run time sees of an image: base image, the RUN lines
// (the nava user and its UID), the binary's path, the user, the exposed ports
// and the entrypoint.
type finalStage struct {
	From       string
	Runs       []string
	Binary     string // COPY destination
	User       string
	Expose     []string
	Entrypoint string
}

type stage struct {
	name, from string
	lines      []string // instructions after FROM, joined continuations
}

// parseDockerfile splits a Dockerfile into stages.
func parseDockerfile(t *testing.T, file string) []stage {
	t.Helper()
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var stages []stage
	var cont string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if strings.HasSuffix(l, `\`) {
			cont += strings.TrimSuffix(l, `\`) + " "
			continue
		}
		l, cont = cont+l, ""
		if fields := strings.Fields(l); strings.EqualFold(fields[0], "FROM") {
			st := stage{from: fields[len(fields)-1]}
			if len(fields) >= 4 && strings.EqualFold(fields[len(fields)-2], "AS") {
				st.from, st.name = fields[len(fields)-3], fields[len(fields)-1]
			}
			stages = append(stages, st)
			continue
		}
		if len(stages) == 0 {
			continue // ARG before the first FROM
		}
		stages[len(stages)-1].lines = append(stages[len(stages)-1].lines, l)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return stages
}

// flatten resolves FROM <earlier stage> chains into one final stage.
func flatten(t *testing.T, stages []stage, i int) finalStage {
	t.Helper()
	var chain []stage
	for st := stages[i]; ; {
		chain = append([]stage{st}, chain...)
		j := slices.IndexFunc(stages, func(s stage) bool { return s.name != "" && s.name == st.from })
		if j < 0 {
			break
		}
		st = stages[j]
	}
	fs := finalStage{From: chain[0].from}
	for _, st := range chain {
		for _, l := range st.lines {
			op, rest, _ := strings.Cut(l, " ")
			rest = strings.TrimSpace(rest)
			switch strings.ToUpper(op) {
			case "RUN":
				fs.Runs = append(fs.Runs, rest)
			case "COPY":
				parts := strings.Fields(rest)
				fs.Binary = parts[len(parts)-1]
			case "USER":
				fs.User = rest
			case "EXPOSE":
				fs.Expose = append(fs.Expose, strings.Fields(rest)...)
			case "ENTRYPOINT":
				fs.Entrypoint = rest
			case "ARG", "ENV", "WORKDIR", "CMD", "LABEL", "HEALTHCHECK", "VOLUME", "STOPSIGNAL", "SHELL", "ADD", "ONBUILD":
				t.Errorf("final stage has %s, which this check does not compare: add it to finalStage: %q", op, l)
			}
		}
	}
	slices.Sort(fs.Expose)
	return fs
}

// realGoServices are the deploy/compose/<dir>/Dockerfile files that build a
// Go binary (a build stage running go build) and the binary each copies.
func realGoServices(t *testing.T) map[string]finalStage {
	t.Helper()
	files, err := filepath.Glob("*/Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	copyRe := regexp.MustCompile(`^COPY\s+--from=\S+\s+/out/(\S+)\s`)
	out := map[string]finalStage{}
	for _, f := range files {
		stages := parseDockerfile(t, f)
		goBuild := false
		for _, st := range stages[:len(stages)-1] {
			for _, l := range st.lines {
				if strings.Contains(l, "go build") {
					goBuild = true
				}
			}
		}
		if !goBuild {
			continue
		}
		var bin string
		for _, l := range stages[len(stages)-1].lines {
			if m := copyRe.FindStringSubmatch(l); m != nil {
				bin = m[1]
			}
		}
		if bin == "" || bin != filepath.Base(filepath.Dir(f)) {
			t.Errorf("%s: final stage copies %q from the build stage, want the binary named after its directory", f, bin)
			continue
		}
		out[bin] = flatten(t, stages, len(stages)-1)
	}
	if len(out) == 0 {
		t.Fatal("no Go service Dockerfile under deploy/compose")
	}
	return out
}

// variantOf maps a Dockerfile.ci target that is a build variant of a real
// Dockerfile (same final stage, other build tags) to that Dockerfile's
// directory: nava-worker-interptest is deploy/compose/nava-worker/Dockerfile
// with NAVA_BUILD_TAGS=interptest (compose.test.yaml).
func variantOf() map[string]string { return map[string]string{"nava-worker-interptest": "nava-worker"} }

func TestDockerfileCIMatchesRealFinalStages(t *testing.T) {
	real := realGoServices(t)
	for variant, base := range variantOf() {
		if _, ok := real[base]; !ok {
			t.Fatalf("variantOf %s: no real deploy/compose/%s/Dockerfile", variant, base)
		}
	}
	stages := parseDockerfile(t, ciDockerfile)
	ci := map[string]finalStage{}
	srcRe := regexp.MustCompile(`^COPY\s+(\S+)\s`)
	for i, st := range stages {
		if st.name == "" || st.name == "base" {
			continue
		}
		for _, l := range st.lines {
			if m := srcRe.FindStringSubmatch(l); m != nil && m[1] != ciBinDir+"/"+st.name {
				t.Errorf("%s target %s copies %s, want %s/%s (make ci-binaries)", ciDockerfile, st.name, m[1], ciBinDir, st.name)
			}
		}
		ci[st.name] = flatten(t, stages, i)
	}
	targets := map[string]string{} // target -> real Dockerfile directory
	for bin := range real {
		targets[bin] = bin
	}
	for variant, base := range variantOf() {
		targets[variant] = base
	}
	for bin, dir := range targets {
		want := real[dir]
		got, ok := ci[bin]
		if !ok {
			t.Errorf("%s has no target %s for deploy/compose/%s/Dockerfile", ciDockerfile, bin, dir)
			continue
		}
		for _, d := range []struct{ what, got, want string }{
			{"base image", got.From, want.From},
			{"RUN lines (user and UID)", strings.Join(got.Runs, "; "), strings.Join(want.Runs, "; ")},
			{"binary path", got.Binary, want.Binary},
			{"user", got.User, want.User},
			{"exposed ports", strings.Join(got.Expose, " "), strings.Join(want.Expose, " ")},
			{"entrypoint", got.Entrypoint, want.Entrypoint},
		} {
			if d.got != d.want {
				t.Errorf("%s target %s: %s %q, real Dockerfile %q", ciDockerfile, bin, d.what, d.got, d.want)
			}
		}
	}
	for bin := range ci {
		if _, ok := targets[bin]; !ok {
			t.Errorf("%s target %s has no real deploy/compose/%s/Dockerfile", ciDockerfile, bin, bin)
		}
	}
}

// makeVar reads one `NAME := value` line of make/stack.mk.
func makeVar(t *testing.T, name string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "make", "stack.mk"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + ` := (.*)$`).FindSubmatch(b)
	if m == nil {
		t.Fatalf("make/stack.mk has no %s := line", name)
	}
	return strings.Fields(string(m[1]))
}

// Every compose service built from a Go service's real Dockerfile is in
// CI_THIN_SERVICES with that service's target, and make ci-binaries builds
// every binary a target copies, so no stack image is left on a stale or
// missing binary, and none is built the real way unnoticed.
func TestThinServicesCoverCompose(t *testing.T) {
	real := realGoServices(t)
	if got := makeVar(t, "CI_BIN_DIR"); len(got) != 1 || got[0] != ciBinDir {
		t.Errorf("CI_BIN_DIR %q, want %s (the COPY source in %s)", got, ciBinDir, ciDockerfile)
	}
	bins := makeVar(t, "CI_BINARIES")
	for bin := range real {
		if !slices.Contains(bins, bin) {
			t.Errorf("CI_BINARIES lacks %s", bin)
		}
	}
	thin := map[string]string{}
	for _, m := range makeVar(t, "CI_THIN_SERVICES") {
		svc, target, ok := strings.Cut(m, "=")
		if !ok {
			t.Fatalf("CI_THIN_SERVICES entry %q is not service=target", m)
		}
		thin[svc] = target
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Services map[string]struct {
			Build *struct {
				Dockerfile string `yaml:"dockerfile"`
			} `yaml:"build"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	built := map[string]bool{}
	for svc, s := range c.Services {
		if s.Build == nil {
			continue
		}
		bin := path.Base(path.Dir(s.Build.Dockerfile))
		if _, ok := real[bin]; !ok || s.Build.Dockerfile != "deploy/compose/"+bin+"/Dockerfile" {
			continue
		}
		built[svc] = true
		if thin[svc] != bin {
			t.Errorf("compose.yaml service %s builds deploy/compose/%s/Dockerfile; CI_THIN_SERVICES maps it to %q, want %s", svc, bin, thin[svc], bin)
		}
	}
	for svc, target := range thin {
		if !built[svc] {
			t.Errorf("CI_THIN_SERVICES lists %s=%s, which compose.yaml does not build from a Go service Dockerfile", svc, target)
		}
		if !slices.Contains(bins, target) {
			t.Errorf("CI_THIN_SERVICES target %s is not in CI_BINARIES", target)
		}
	}
}

// TICK-641: with the published temporal-server image, make image-ci bakes
// BAKE_TARGETS minus temporal-server instead of bake's default group (every
// compose service with a build:). A service missing from BAKE_TARGETS would
// then not be built at all, and the stack would start on a stale image or
// none. BAKE_TARGETS must be exactly the services compose.yaml builds.
func TestBakeTargetsCoverCompose(t *testing.T) {
	var targets []string
	for _, f := range makeVar(t, "BAKE_TARGETS") {
		if f == "$(BAKE_CACHED)" {
			targets = append(targets, makeVar(t, "BAKE_CACHED")...)
			continue
		}
		targets = append(targets, f)
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Services map[string]struct {
			Build any `yaml:"build"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	var built []string
	for svc, s := range c.Services {
		if s.Build != nil {
			built = append(built, svc)
		}
	}
	slices.Sort(targets)
	slices.Sort(built)
	if !slices.Equal(targets, built) {
		t.Errorf("BAKE_TARGETS (make/stack.mk) = %v; compose.yaml builds %v", targets, built)
	}
	if !slices.Contains(targets, "temporal-server") {
		t.Errorf("BAKE_TARGETS lacks temporal-server, which TEMPORAL_SERVER_PREBUILT=1 filters out")
	}
}

// .dockerignore must not drop the binaries, or the thin build fails with a
// missing COPY source (or, worse, keeps a stale layer).
func TestDockerignoreKeepsCIBinaries(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", ".dockerignore"))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(string(b), "\n") {
		p := strings.TrimSpace(l)
		if p == "" || strings.HasPrefix(p, "#") || strings.HasPrefix(p, "!") {
			continue
		}
		p = strings.TrimPrefix(strings.TrimSuffix(p, "/"), "/")
		for _, target := range []string{ciBinDir, ciBinDir + "/nava-core"} {
			if ok, _ := path.Match(p, target); ok || p == "**" || strings.HasPrefix(target, p+"/") {
				t.Errorf(".dockerignore line %q excludes %s, which Dockerfile.ci copies from", l, target)
			}
		}
	}
}
