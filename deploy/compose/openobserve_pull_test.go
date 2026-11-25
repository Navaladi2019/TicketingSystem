package compose

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TICK-1128: a full CI run pulls OpenObserve by the digest compose.yaml pins,
// from ECR Public and, when that fails, from Docker Hub (make
// openobserve-pull, scripts/ci/pull-openobserve.sh). These tests run the real
// script and the real make targets against a fake docker on PATH: no daemon
// and no network.

const (
	ooScript = "scripts/ci/pull-openobserve.sh"
	ooFirst  = "public.ecr.aws/zinclabs/openobserve"
	ooSecond = "docker.io/openobserve/openobserve"
	ooOther  = "sha256:0000000000000000000000000000000000000000000000000000000000000001"
)

// ooPin is the one place the OpenObserve version and digest are written: the
// default of the openobserve service's image in compose.yaml.
var ooPin = regexp.MustCompile(`^\$\{OPENOBSERVE_IMAGE:-([^:}]+):([^@}]+)@(sha256:[0-9a-f]{64})\}$`)

func openobservePin(t *testing.T) (repo, tag, digest string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Services map[string]struct {
			Image string `yaml:"image"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	m := ooPin.FindStringSubmatch(c.Services["openobserve"].Image)
	if m == nil {
		t.Fatalf("compose.yaml openobserve image %q; want ${OPENOBSERVE_IMAGE:-<repo>:<tag>@sha256:<64 hex>}", c.Services["openobserve"].Image)
	}
	return m[1], m[2], m[3]
}

// fakeDocker logs its arguments to $FAKE_LOG. `pull` fails for a ref that
// contains a word of $FAKE_REFUSE; `image inspect` reports $FAKE_DIGEST, when
// set, for the refs that contain $FAKE_DIGEST_FOR, and otherwise the digest
// pulled; `run` prints $FAKE_VERSION; `compose config` prints three services.
const fakeDocker = `#!/bin/sh
echo "$*" >>"$FAKE_LOG"
for ref; do :; done
case $1 in
pull)
    for r in $FAKE_REFUSE; do
        case $ref in *"$r"*) echo "refused-by-$r: Data limit exceeded" >&2; exit 1 ;; esac
    done
    echo "$ref" ;;
image)
    case ${FAKE_DIGEST:+$ref} in
    *"${FAKE_DIGEST_FOR:-@}"*) ref=${ref%@*}@$FAKE_DIGEST ;;
    esac
    echo "${ref#docker.io/}" ;;
run) echo "openobserve $FAKE_VERSION" ;;
compose)
    case " $* " in *" config "*) printf 'postgres\nopenobserve\nprometheus\n' ;; esac ;;
esac
`

type fakeRun struct {
	out   string
	code  int
	calls []string // the docker calls, in order
	ghEnv string   // what was written to GITHUB_ENV
}

// pulls returns the refs the docker calls pulled, in order.
func (r fakeRun) pulls() []string {
	var refs []string
	for _, c := range r.calls {
		if f := strings.Fields(c); len(f) > 1 && f[0] == "pull" {
			refs = append(refs, f[len(f)-1])
		}
	}
	return refs
}

// runFake runs name with args from the repo root with the fake docker first
// on PATH.
func runFake(t *testing.T, env []string, name string, args ...string) fakeRun {
	t.Helper()
	bin, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("no %s on PATH", name)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(fakeDocker), 0o755); err != nil { //nolint:gosec // a test executable
		t.Fatal(err)
	}
	log, gh := filepath.Join(dir, "calls"), filepath.Join(dir, "github_env")
	cmd := exec.CommandContext(t.Context(), bin, args...)
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(),
		"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_LOG="+filepath.ToSlash(log), "GITHUB_ENV="+filepath.ToSlash(gh),
		"FAKE_REFUSE=", "FAKE_DIGEST=", "FAKE_DIGEST_FOR=", "FAKE_VERSION=",
		"OPENOBSERVE_IMAGE=", "COMPOSE_YAML=")
	cmd.Env = append(cmd.Env, env...)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	r := fakeRun{}
	err = cmd.Run()
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		r.code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	r.out = buf.String()
	read := func(p string) string {
		b, err := os.ReadFile(p) //nolint:gosec // this test's own files
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(b))
	}
	if s := read(log); s != "" {
		r.calls = strings.Split(s, "\n")
	}
	r.ghEnv = read(gh)
	return r
}

func TestOpenObservePull(t *testing.T) {
	_, tag, digest := openobservePin(t)
	version := "FAKE_VERSION=" + strings.TrimSuffix(tag, "-debug")
	first, second := ooFirst+"@"+digest, ooSecond+"@"+digest
	for _, tc := range []struct {
		name  string
		env   []string
		pulls []string
		using string // "": the step must fail
		want  []string
	}{
		{"the first source serves", nil, []string{first}, first, nil},
		{
			"the first refuses, the second serves",
			[]string{"FAKE_REFUSE=public.ecr.aws"},
			[]string{first, second}, second, []string{"refused-by-public.ecr.aws"},
		},
		{
			"a wrong digest from the first is refused, the second serves",
			[]string{"FAKE_DIGEST=" + ooOther, "FAKE_DIGEST_FOR=public.ecr.aws"},
			[]string{first, second}, second, []string{ooOther},
		},
		{
			"a wrong digest from both fails",
			[]string{"FAKE_DIGEST=" + ooOther},
			[]string{first, second}, "", []string{ooFirst, ooSecond, ooOther},
		},
		{
			"both refuse, both errors are printed",
			[]string{"FAKE_REFUSE=public.ecr.aws docker.io"},
			[]string{first, second}, "",
			[]string{"refused-by-public.ecr.aws", "refused-by-docker.io", ooFirst, ooSecond},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := runFake(t, append([]string{version}, tc.env...), "sh", ooScript)
			if got := r.pulls(); !slices.Equal(got, tc.pulls) {
				t.Errorf("pulled %v, want %v", got, tc.pulls)
			}
			for _, w := range tc.want {
				if !strings.Contains(r.out, w) {
					t.Errorf("output lacks %q", w)
				}
			}
			if tc.using == "" {
				if r.code == 0 || r.ghEnv != "" {
					t.Errorf("exit %d, GITHUB_ENV %q; want a failure that exports nothing", r.code, r.ghEnv)
				}
			} else {
				if r.code != 0 {
					t.Errorf("exit %d", r.code)
				}
				if want := "OPENOBSERVE_IMAGE=" + tc.using; r.ghEnv != want {
					t.Errorf("GITHUB_ENV %q, want %q", r.ghEnv, want)
				}
				if !strings.Contains(r.out, "using "+tc.using) {
					t.Errorf("output does not name the source and digest used, %s", tc.using)
				}
			}
			if t.Failed() {
				t.Logf("exit %d, output:\n%s", r.code, r.out)
			}
		})
	}
}

// The drift test: the version CI pulls is the one compose.yaml names. Docker
// ignores the tag of a tag@digest ref, so a tag bumped without its digest
// would run the old version unnoticed; the script asks the pulled binary.
func TestOpenObserveVersionMatchesCompose(t *testing.T) {
	repo, tag, digest := openobservePin(t)
	if repo != ooFirst {
		t.Errorf("compose.yaml pulls openobserve from %s, want %s (TICK-634)", repo, ooFirst)
	}
	r := runFake(t, []string{"FAKE_VERSION=v0.0.1"}, "sh", ooScript)
	if r.code == 0 || r.ghEnv != "" || !strings.Contains(r.out, "v0.0.1") || !strings.Contains(r.out, strings.TrimSuffix(tag, "-debug")) {
		t.Errorf("image says v0.0.1, compose.yaml %s: exit %d, GITHUB_ENV %q; want a failure naming both\n%s", tag, r.code, r.ghEnv, r.out)
	}
	// Stated once: no script, make file, workflow or overlay repeats them.
	for _, glob := range []string{"make/*.mk", "scripts/ci/*.sh", ".github/workflows/*.yml", "deploy/compose/*.yaml"} {
		files, err := filepath.Glob(filepath.Join("..", "..", glob))
		if err != nil || len(files) == 0 {
			t.Fatalf("%s: %v, %d files", glob, err, len(files))
		}
		for _, f := range files {
			b, err := os.ReadFile(f) //nolint:gosec // a repo file
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), digest) || strings.Contains(string(b), tag) {
				t.Errorf("%s repeats the OpenObserve digest or version of compose.yaml", f)
			}
		}
	}
}

// A pin without a full digest fails the step before any pull.
func TestOpenObservePullNeedsAPinnedDigest(t *testing.T) {
	for _, image := range []string{
		ooFirst + ":v1.1.2-debug",
		"${OPENOBSERVE_IMAGE:-" + ooFirst + ":v1.1.2-debug}",
		"${OPENOBSERVE_IMAGE:-" + ooFirst + ":v1.1.2-debug@sha256:16d9}",
		"${OPENOBSERVE_IMAGE:-" + ooFirst + "@sha256:" + strings.Repeat("a", 64) + "}",
	} {
		f := filepath.Join(t.TempDir(), "compose.yaml")
		if err := os.WriteFile(f, []byte("services:\n  openobserve:\n    image: "+image+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		r := runFake(t, []string{"COMPOSE_YAML=" + filepath.ToSlash(f)}, "sh", ooScript)
		if r.code == 0 || len(r.calls) != 0 || !strings.Contains(r.out, "pin") {
			t.Errorf("image %s: exit %d, docker calls %v; want a failure about the pin with no call\n%s", image, r.code, r.calls, r.out)
		}
	}
}

// make openobserve-pull pulls only when the operator profile starts; a fast
// run starts no OpenObserve.
func TestOpenObservePullFollowsTheProfile(t *testing.T) {
	_, tag, digest := openobservePin(t)
	env := []string{"FAKE_VERSION=" + strings.TrimSuffix(tag, "-debug")}
	for profiles, want := range map[string][]string{"operator": {ooFirst + "@" + digest}, "": nil} {
		r := runFake(t, env, "make", "--no-print-directory", "openobserve-pull", "COMPOSE_PROFILES="+profiles)
		if got := r.pulls(); r.code != 0 || !slices.Equal(got, want) {
			t.Errorf("COMPOSE_PROFILES=%q: exit %d, pulled %v, want %v\n%s", profiles, r.code, got, want, r.out)
		}
	}
}

// Once OPENOBSERVE_IMAGE is verified, image-ci's registry pull leaves the
// service out; without it (a laptop) it pulls every service as before.
func TestImageCILeavesVerifiedOpenObserveOut(t *testing.T) {
	_, _, digest := openobservePin(t)
	const pull = "compose pull --quiet --ignore-buildable --ignore-pull-failures"
	for image, want := range map[string]string{
		ooSecond + "@" + digest: pull + " postgres prometheus",
		"":                      pull,
	} {
		r := runFake(t, []string{"OPENOBSERVE_IMAGE=" + image}, "make", "--no-print-directory", "image-ci",
			"COMPOSE_PROFILES=operator", "IMAGE_CACHE=", "IMAGE_THIN=", "PREBUILT_SERVICES=")
		if r.code != 0 || !slices.Contains(r.calls, want) {
			t.Errorf("OPENOBSERVE_IMAGE=%q: exit %d, docker calls %q; want %q\n%s", image, r.code, r.calls, want, r.out)
		}
	}
}

// The stack job runs the verified pull once, before image-ci and every stack
// start, so OPENOBSERVE_IMAGE is set for all of them.
func TestStackJobPullsOpenObserveFirst(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Jobs map[string]struct {
			Steps []struct {
				Run string `yaml:"run"`
				If  string `yaml:"if"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(b, &wf); err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, s := range wf.Jobs["stack"].Steps {
		run := strings.TrimSpace(s.Run)
		if run == "make openobserve-pull" && s.If != "" {
			t.Errorf("make openobserve-pull has if: %s; the make target decides from COMPOSE_PROFILES", s.If)
		}
		if slices.Contains([]string{"make openobserve-pull", "make image-ci", "make stack-up"}, run) {
			order = append(order, run)
		}
	}
	if want := []string{"make openobserve-pull", "make image-ci", "make stack-up", "make stack-up"}; !slices.Equal(order, want) {
		t.Errorf("stack job steps %v, want %v", order, want)
	}
}
