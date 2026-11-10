package compose

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TICK-641, TICK-693: the stack job pulls a published image for each service
// in PUBLISHED_IMAGES (make/stack.mk) whose tag is a content key of the
// paths in that service's inputs= line of scripts/ci/prebuilt-image.sh. If
// the Dockerfile copies a file the key does not hash, a change to that file
// keeps the old key, CI pulls the stale image, and the stack cases test an
// image that is not the one the change builds. These tests fail first.

const prebuiltScript = "scripts/ci/prebuilt-image.sh" // from the repo root

// prebuiltInputs reads the key inputs of svc from the script's case entry.
func prebuiltInputs(t *testing.T, svc string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", prebuiltScript))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(svc) + `\)\s*\n\s*inputs='([^']*)'`).FindSubmatch(b)
	if m == nil {
		t.Fatalf("%s has no %s) entry starting with an inputs='...' line", prebuiltScript, svc)
	}
	return strings.Fields(string(m[1]))
}

// composeBuild is a compose.yaml build: in its long or short form.
type composeBuild struct {
	Context, Dockerfile string
}

func composeBuilds(t *testing.T) map[string]composeBuild {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Services map[string]struct {
			Build yaml.Node `yaml:"build"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	out := map[string]composeBuild{}
	for svc, s := range c.Services {
		switch s.Build.Kind {
		case 0:
			continue
		case yaml.ScalarNode:
			out[svc] = composeBuild{Context: s.Build.Value}
		default:
			var cb composeBuild
			if err := s.Build.Decode(&cb); err != nil {
				t.Fatalf("compose.yaml %s build: %v", svc, err)
			}
			out[svc] = cb
		}
	}
	for svc, cb := range out {
		cb.Context = path.Clean(strings.TrimPrefix(cb.Context, "./"))
		if cb.Dockerfile == "" {
			cb.Dockerfile = "Dockerfile"
		}
		// compose resolves dockerfile relative to the context.
		cb.Dockerfile = path.Join(cb.Context, cb.Dockerfile)
		out[svc] = cb
	}
	return out
}

// copySources returns the repo-relative source paths of every COPY and ADD
// in lines (parseDockerfile's joined instructions) that reads the build
// context: not COPY --from=, not a URL.
func copySources(lines []string, context string) []string {
	var srcs []string
	for _, l := range lines {
		f := strings.Fields(l)
		if len(f) < 3 || (!strings.EqualFold(f[0], "COPY") && !strings.EqualFold(f[0], "ADD")) {
			continue
		}
		args := f[1:]
		fromStage := false
		for len(args) > 0 && strings.HasPrefix(args[0], "--") {
			if strings.HasPrefix(args[0], "--from=") {
				fromStage = true
			}
			args = args[1:]
		}
		if fromStage || len(args) < 2 {
			continue
		}
		for _, s := range args[:len(args)-1] {
			if strings.Contains(s, "://") {
				continue
			}
			srcs = append(srcs, path.Join(context, s))
		}
	}
	return srcs
}

// uncovered returns the paths that no input equals or contains.
func uncovered(paths, inputs []string) []string {
	var out []string
	for _, p := range paths {
		p = path.Clean(p)
		if !slices.ContainsFunc(inputs, func(in string) bool {
			in = path.Clean(in)
			return in == "." || p == in || strings.HasPrefix(p, in+"/")
		}) {
			out = append(out, p)
		}
	}
	return out
}

func TestPublishedImageKeyCoversBuild(t *testing.T) {
	builds := composeBuilds(t)
	published := makeVar(t, "PUBLISHED_IMAGES")
	if !slices.Contains(published, "keycloak") || !slices.Contains(published, "temporal-server") {
		t.Errorf("PUBLISHED_IMAGES = %v; want temporal-server and keycloak", published)
	}
	targets := makeVar(t, "BAKE_CACHED")
	for _, svc := range published {
		t.Run(svc, func(t *testing.T) {
			if !slices.Contains(targets, svc) {
				t.Errorf("%s is in PUBLISHED_IMAGES but not in BAKE_CACHED: make image-ci would never build it from source on a miss", svc)
			}
			cb, ok := builds[svc]
			if !ok {
				t.Fatalf("compose.yaml does not build %s", svc)
			}
			inputs := prebuiltInputs(t, svc)
			for _, in := range inputs {
				if _, err := os.Stat(filepath.Join("..", "..", filepath.FromSlash(in))); err != nil {
					t.Errorf("key input %s: %v", in, err)
				}
			}
			stages := parseDockerfile(t, filepath.Join("..", "..", filepath.FromSlash(cb.Dockerfile)))
			need := []string{cb.Dockerfile}
			for _, st := range stages {
				need = append(need, copySources(st.lines, cb.Context)...)
			}
			for _, p := range uncovered(need, inputs) {
				t.Errorf("%s's build reads %s, which its key (%s inputs=%v) does not hash: a change to it would pull a stale image", svc, p, prebuiltScript, inputs)
			}
		})
	}
}

// The guard itself: a COPY the key misses is reported, the stage copies and
// the covered paths are not.
func TestUncoveredCopySources(t *testing.T) {
	lines := []string{
		"ENV KC_DB=postgres",
		"COPY deploy/compose/keycloak/quarkus.properties /opt/keycloak/conf/quarkus.properties",
		"COPY --chown=1000 deploy/compose/keycloak/import/ /opt/keycloak/data/import/",
		"ADD deploy/compose/keycloak/a.jar deploy/compose/keycloak/b.jar /opt/keycloak/providers/",
		"COPY --from=build /out/app /app",
		"ADD https://example.invalid/x.tgz /tmp/",
		"RUN /opt/keycloak/bin/kc.sh build",
	}
	inputs := []string{"deploy/compose/keycloak/Dockerfile", "deploy/compose/keycloak/quarkus.properties"}
	got := uncovered(copySources(lines, "."), inputs)
	want := []string{"deploy/compose/keycloak/import", "deploy/compose/keycloak/a.jar", "deploy/compose/keycloak/b.jar"}
	if !slices.Equal(got, want) {
		t.Errorf("uncovered = %v, want %v", got, want)
	}
	// A directory input (temporal-server) covers everything under it.
	if got := uncovered(copySources([]string{"COPY go.mod go.sum ./", "COPY . ."}, "temporal-server"), []string{"temporal-server"}); len(got) != 0 {
		t.Errorf("temporal-server context: uncovered = %v, want none", got)
	}
}

// The publish workflow's matrix is PUBLISHED_IMAGES: a service missing from
// it would never be published and always miss; one extra would publish an
// image the stack job never pulls.
func TestPrebuiltWorkflowMatrix(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "prebuilt-images.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Strategy struct {
				Matrix struct {
					Service []string `yaml:"service"`
				} `yaml:"matrix"`
			} `yaml:"strategy"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(b, &wf); err != nil {
		t.Fatal(err)
	}
	got := slices.Clone(wf.Jobs["publish"].Strategy.Matrix.Service)
	want := slices.Clone(makeVar(t, "PUBLISHED_IMAGES"))
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("prebuilt-images.yml publish matrix %v, PUBLISHED_IMAGES %v", got, want)
	}
}
