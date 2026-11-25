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
	"time"

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
// openobserve service's image in compose.yaml. The variable covers the
// repository only, so no value of it can replace the tag or digest (TICK-1131).
var ooPin = regexp.MustCompile(`^\$\{OPENOBSERVE_REPO:-([^:}]+)\}:([^@}]+)@(sha256:[0-9a-f]{64})$`)

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
		t.Fatalf("compose.yaml openobserve image %q; want ${OPENOBSERVE_REPO:-<repo>}:<tag>@sha256:<64 hex>", c.Services["openobserve"].Image)
	}
	return m[1], m[2], m[3]
}

// fakeDocker logs its arguments to $FAKE_LOG. `pull` fails for a ref that
// contains a word of $FAKE_REFUSE and sleeps 60 s, ignoring SIGTERM when
// $FAKE_STUBBORN is set, for one that contains a word of $FAKE_HANG; `image inspect` reports $FAKE_DIGEST, when
// set, for the refs that contain $FAKE_DIGEST_FOR, and otherwise the digest
// pulled; `run` prints $FAKE_VERSION; `compose config` prints three services.
const fakeDocker = `#!/bin/sh
printf '%s\n' "$*" >>"$FAKE_LOG"
for ref; do :; done
case $1 in
pull)
    for r in $FAKE_REFUSE; do
        case $ref in *"$r"*) echo "refused-by-$r: Data limit exceeded" >&2; exit 1 ;; esac
    done
    for r in $FAKE_HANG; do
        case $ref in *"$r"*) [ -z "$FAKE_STUBBORN" ] || trap '' TERM; exec sleep 60 ;; esac
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

// fakeTimeout logs its first three arguments and runs the real timeout, the
// next one on PATH.
const fakeTimeout = `#!/bin/sh
echo "timeout $1 $2 $3" >>"$FAKE_LOG"
real=$(PATH=${PATH#*:}; command -v timeout) || exit 127
exec "$real" "$@"
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

// runFake runs name with args from the repo root with the fake docker and
// timeout first on PATH.
func runFake(t *testing.T, env []string, name string, args ...string) fakeRun {
	t.Helper()
	bin, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("no %s on PATH", name)
	}
	dir := t.TempDir()
	for name, body := range map[string]string{"docker": fakeDocker, "timeout": fakeTimeout} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	log, gh := filepath.Join(dir, "calls"), filepath.Join(dir, "github_env")
	cmd := exec.CommandContext(t.Context(), bin, args...)
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(),
		"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_LOG="+filepath.ToSlash(log), "GITHUB_ENV="+filepath.ToSlash(gh),
		"FAKE_REFUSE=", "FAKE_DIGEST=", "FAKE_DIGEST_FOR=", "FAKE_VERSION=", "FAKE_HANG=", "FAKE_STUBBORN=",
		"OPENOBSERVE_REPO=", "PULL_TIMEOUT=", "COMPOSE_YAML=")
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
		b, err := os.ReadFile(p)
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

// checkOrder fails unless every pull is time limited, and the image runs only
// after its digest was read and only as ref with no network. ref "" is a run
// in which no source served: nothing runs.
func (r fakeRun) checkOrder(t *testing.T, limit, ref string) {
	t.Helper()
	inspected, runs := false, 0
	for i, c := range r.calls {
		switch {
		case strings.HasPrefix(c, "pull "):
			if want := "timeout -k 10 " + limit; i == 0 || r.calls[i-1] != want {
				t.Errorf("docker %s does not run under %q", c, want)
			}
		case strings.HasPrefix(c, "image inspect ") && strings.HasSuffix(c, " "+ref):
			inspected = true
		case strings.HasPrefix(c, "run "):
			runs++
			if want := "run --rm --network none --entrypoint /openobserve " + ref + " --version"; c != want || !inspected {
				t.Errorf("docker %s (digest checked before it: %t); want only %q, after the check", c, inspected, want)
			}
		}
	}
	if want := min(len(ref), 1); runs != want {
		t.Errorf("%d docker run calls, want %d: %q", runs, want, r.calls)
	}
}

func TestOpenObservePull(t *testing.T) {
	_, tag, digest := openobservePin(t)
	version := "FAKE_VERSION=" + strings.TrimSuffix(tag, "-debug")
	first, second := ooFirst+"@"+digest, ooSecond+"@"+digest
	for _, tc := range []struct {
		name  string
		env   []string
		pulls []string
		using string // the source that serves; "": the step must fail
		want  []string
	}{
		{"the first source serves", nil, []string{first}, ooFirst, nil},
		{
			"the first refuses, the second serves",
			[]string{"FAKE_REFUSE=public.ecr.aws"},
			[]string{first, second}, ooSecond, []string{"refused-by-public.ecr.aws"},
		},
		{
			"a wrong digest from the first is refused, the second serves",
			[]string{"FAKE_DIGEST=" + ooOther, "FAKE_DIGEST_FOR=public.ecr.aws"},
			[]string{first, second}, ooSecond, []string{ooOther},
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
				r.checkOrder(t, "180", "")
			} else {
				ref := tc.using + "@" + digest
				if r.code != 0 {
					t.Errorf("exit %d", r.code)
				}
				// The repository only: compose.yaml adds the tag and the
				// digest, so it names the digest the script verified and ran.
				if want := "OPENOBSERVE_REPO=" + tc.using; r.ghEnv != want {
					t.Errorf("GITHUB_ENV %q, want %q", r.ghEnv, want)
				}
				if !strings.Contains(r.out, "using "+ref) {
					t.Errorf("output does not name the source and digest used, %s", ref)
				}
				r.checkOrder(t, "180", ref)
			}
			if t.Failed() {
				t.Logf("exit %d, output:\n%s", r.code, r.out)
			}
		})
	}
}

// A source that hangs is abandoned at the time limit and the next one is
// tried, also when the hung pull ignores SIGTERM.
func TestOpenObservePullTimeLimit(t *testing.T) {
	_, tag, digest := openobservePin(t)
	for _, tc := range []struct {
		name, hang, stubborn, using string
	}{
		{"the first hangs, the second serves", "public.ecr.aws", "", ooSecond},
		{"the first hangs and ignores SIGTERM", "public.ecr.aws", "1", ooSecond},
		{"both hang", "public.ecr.aws docker.io", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			r := runFake(t, []string{
				"FAKE_VERSION=" + strings.TrimSuffix(tag, "-debug"), "PULL_TIMEOUT=1",
				"FAKE_HANG=" + tc.hang, "FAKE_STUBBORN=" + tc.stubborn,
			}, "sh", ooScript)
			if d := time.Since(start); d > 30*time.Second {
				t.Errorf("took %s, want under 30 s", d)
			}
			if want := []string{ooFirst + "@" + digest, ooSecond + "@" + digest}; !slices.Equal(r.pulls(), want) {
				t.Errorf("pulled %v, want one attempt per source: %v", r.pulls(), want)
			}
			ref, ghEnv, hung := "", "", []string{ooFirst, ooSecond}
			if tc.using != "" {
				ref, ghEnv, hung = tc.using+"@"+digest, "OPENOBSERVE_REPO="+tc.using, hung[:1]
			}
			if (r.code == 0) != (tc.using != "") || r.ghEnv != ghEnv {
				t.Errorf("exit %d, GITHUB_ENV %q; want success %t, GITHUB_ENV %q", r.code, r.ghEnv, tc.using != "", ghEnv)
			}
			for _, repo := range hung {
				if want := repo + " failed: no answer in 1 s\n"; !strings.Contains(r.out, want) {
					t.Errorf("output lacks %q", want)
				}
			}
			r.checkOrder(t, "1", ref)
			if t.Failed() {
				t.Logf("exit %d, output:\n%s", r.code, r.out)
			}
		})
	}
}

// Setting the variable moves the pull to another source and nothing else: the
// real docker compose keeps compose.yaml's tag and digest (TICK-1131).
func TestOpenObserveOverrideChoosesOnlyTheRepository(t *testing.T) {
	_, tag, digest := openobservePin(t)
	if out, err := exec.CommandContext(t.Context(), "docker", "compose", "version").CombinedOutput(); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("CI needs docker compose: %v\n%s", err, out)
		}
		t.Skipf("no docker compose: %v", err)
	}
	// Not the developer's own .env, which compose would read.
	env := filepath.Join(t.TempDir(), "env")
	if err := os.WriteFile(env, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for value, repo := range map[string]string{"": ooFirst, ooSecond: ooSecond} {
		cmd := exec.CommandContext(t.Context(), "docker", "compose", "--env-file", env, "-f", "compose.yaml",
			"--profile", "operator", "config", "--images", "openobserve")
		cmd.Dir = filepath.Join("..", "..")
		cmd.Env = append(os.Environ(), "OPENOBSERVE_REPO="+value)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("docker compose config: %v\n%s", err, stderr.String())
		}
		if got, want := strings.TrimSpace(string(out)), repo+":"+tag+"@"+digest; got != want {
			t.Errorf("OPENOBSERVE_REPO=%q: image %s, want %s", value, got, want)
		}
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
			b, err := os.ReadFile(f)
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
		"${OPENOBSERVE_REPO:-" + ooFirst + "}:v1.1.2-debug",
		"${OPENOBSERVE_REPO:-" + ooFirst + "}:v1.1.2-debug@sha256:16d9",
		"${OPENOBSERVE_REPO:-" + ooFirst + "}@sha256:" + strings.Repeat("a", 64),
		// The whole reference inside the braces: a value would replace the digest.
		"${OPENOBSERVE_REPO:-" + ooFirst + ":v1.1.2-debug@sha256:" + strings.Repeat("a", 64) + "}",
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

// Once OPENOBSERVE_REPO is set, image-ci's registry pull leaves the
// service out; without it (a laptop) it pulls every service as before.
func TestImageCILeavesVerifiedOpenObserveOut(t *testing.T) {
	const pull = "compose pull --quiet --ignore-buildable --ignore-pull-failures"
	for repo, want := range map[string]string{
		ooSecond: pull + " postgres prometheus",
		"":       pull,
	} {
		r := runFake(t, []string{"OPENOBSERVE_REPO=" + repo}, "make", "--no-print-directory", "image-ci",
			"COMPOSE_PROFILES=operator", "IMAGE_CACHE=", "IMAGE_THIN=", "PREBUILT_SERVICES=")
		if r.code != 0 || !slices.Contains(r.calls, want) {
			t.Errorf("OPENOBSERVE_REPO=%q: exit %d, docker calls %q; want %q\n%s", repo, r.code, r.calls, want, r.out)
		}
	}
}

// The stack job runs the verified pull once, before image-ci and every stack
// start, so OPENOBSERVE_REPO is set for all of them.
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
