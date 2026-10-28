// Package compose holds checks on the compose files themselves: invariants
// that span services, so no one binary's tests own them.
package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// tokenValidators are the services known to validate access tokens. Each one
// must set NAVA_OIDC_ISSUERS in compose.yaml: without it a service either
// refuses every token or, worse, falls back to something other than the pin.
// nava-worker is not one: it calls config.Load, which parses the OIDC vars, but
// never builds a verifier.
func tokenValidators() []string { return []string{"nava-core", "nava-onboard"} }

// issuerVars are the env vars that name an issuer, each with how to read the
// issuers out of its value (internal/platform/config). Every issuer they name
// must be the pinned one.
func issuerVars() map[string]func(string) []string {
	return map[string]func(string) []string{
		"NAVA_OIDC_ISSUERS":   splitList,
		"NAVA_ONBOARD_ISSUER": func(v string) []string { return []string{v} },
		// "issuer|subject" pairs.
		"NAVA_OPERATORS": func(v string) []string {
			var out []string
			for _, p := range splitList(v) {
				iss, _, _ := strings.Cut(p, "|")
				out = append(out, iss)
			}
			return out
		},
	}
}

// keyVars point a service at a key source. A service that sets one validates
// tokens, so it must also set NAVA_OIDC_ISSUERS.
func keyVars() []string { return []string{"NAVA_OIDC_FETCH_BASE_URL"} }

// realmPath is appended to KC_HOSTNAME to make the realm's iss.
const realmPath = "/realms/nava"

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// service is one compose service as far as this check needs it. Env keeps the
// raw template (${VAR:-default}), so an issuer that resolves to the pin only
// by its default, through a different variable, still fails.
type service struct {
	Env     map[string]string
	EnvFile bool
}

// composeFile is a compose file's services by name.
type composeFile map[string]service

func parseCompose(b []byte) (composeFile, error) {
	var doc struct {
		Services map[string]struct {
			Environment yaml.Node `yaml:"environment"`
			EnvFile     yaml.Node `yaml:"env_file"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	out := composeFile{}
	for name, s := range doc.Services {
		env := map[string]string{}
		switch s.Environment.Kind {
		case 0:
		case yaml.MappingNode:
			var m map[string]any
			if err := s.Environment.Decode(&m); err != nil {
				return nil, fmt.Errorf("%s: environment: %w", name, err)
			}
			for k, v := range m {
				if v == nil {
					v = ""
				}
				env[k] = fmt.Sprint(v)
			}
		case yaml.SequenceNode:
			var l []string
			if err := s.Environment.Decode(&l); err != nil {
				return nil, fmt.Errorf("%s: environment: %w", name, err)
			}
			for _, kv := range l {
				k, v, _ := strings.Cut(kv, "=")
				env[k] = v
			}
		default:
			return nil, fmt.Errorf("%s: environment is neither a map nor a list", name)
		}
		out[name] = service{Env: env, EnvFile: s.EnvFile.Kind != 0}
	}
	return out, nil
}

var composeDefault = regexp.MustCompile(`\$\{[A-Za-z_][A-Za-z0-9_]*:?-([^}]*)\}`)

// resolve replaces ${VAR:-default} and ${VAR-default} with the default.
func resolve(s string) string { return composeDefault.ReplaceAllString(s, "$1") }

// checkIssuers returns one message per violation, each naming the file and
// service. files maps a file name to its services; base is the name of the
// root file (compose.yaml), which holds Keycloak and every validator. The
// others are overlays: what they set must not move any issuer off the pin.
func checkIssuers(files map[string]composeFile, base string) []string {
	var errs []string
	kc, ok := files[base]["keycloak"]
	host := kc.Env["KC_HOSTNAME"]
	if !ok || resolve(host) == "" {
		return []string{base + ": keycloak sets no KC_HOSTNAME, so there is no pinned issuer"}
	}
	pinRaw, pin := host+realmPath, resolve(host)+realmPath

	names := make([]string, 0, len(files))
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	for _, f := range names {
		svcs := make([]string, 0, len(files[f]))
		for s := range files[f] {
			svcs = append(svcs, s)
		}
		sort.Strings(svcs)
		for _, name := range svcs {
			s := files[f][name]
			if v, ok := s.Env["KC_HOSTNAME"]; ok && f != base && v != host {
				errs = append(errs, fmt.Sprintf("%s: %s overrides KC_HOSTNAME = %q, which moves the pinned issuer %q", f, name, v, pin))
			}
			validates := false
			for _, k := range keyVars() {
				if _, ok := s.Env[k]; ok {
					validates = true
				}
			}
			ivars := issuerVars()
			vars := make([]string, 0, len(ivars))
			for k := range ivars {
				vars = append(vars, k)
			}
			sort.Strings(vars)
			for _, k := range vars {
				v, ok := s.Env[k]
				if !ok {
					continue
				}
				validates = true
				issuers := ivars[k](v)
				if len(issuers) == 0 {
					errs = append(errs, fmt.Sprintf("%s: %s sets %s empty, want the pinned issuer %q", f, name, k, pin))
				}
				for _, iss := range issuers {
					if resolve(iss) != pin {
						errs = append(errs, fmt.Sprintf("%s: %s %s trusts %q, want the pinned issuer %q", f, name, k, resolve(iss), pin))
					} else if iss != pinRaw {
						errs = append(errs, fmt.Sprintf("%s: %s %s is %q, want it written as %q so it follows KC_HOSTNAME", f, name, k, iss, pinRaw))
					}
				}
			}
			if f != base {
				continue
			}
			if validates || isValidator(name) {
				if _, ok := s.Env["NAVA_OIDC_ISSUERS"]; !ok {
					errs = append(errs, fmt.Sprintf("%s: %s validates tokens but sets no NAVA_OIDC_ISSUERS", f, name))
				}
				if s.EnvFile {
					errs = append(errs, fmt.Sprintf("%s: %s validates tokens and reads an env_file this check cannot see; set its issuer in environment", f, name))
				}
			}
		}
	}
	for _, v := range tokenValidators() {
		if _, ok := files[base][v]; !ok {
			errs = append(errs, fmt.Sprintf("%s: token-validating service %s is missing; update tokenValidators if it was renamed", base, v))
		}
	}
	return errs
}

func isValidator(name string) bool {
	for _, v := range tokenValidators() {
		if v == name {
			return true
		}
	}
	return false
}

// loadStackFiles reads compose.yaml and every overlay in deploy/compose, the
// files the stack is brought up from.
func loadStackFiles(t *testing.T) (map[string]composeFile, string) {
	t.Helper()
	const base = "compose.yaml"
	overlays, err := filepath.Glob("compose.*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]composeFile{}
	for _, f := range append([]string{"../../" + base}, overlays...) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		c, err := parseCompose(b)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		name := strings.TrimPrefix(f, "../../")
		if name != base {
			name = "deploy/compose/" + name
		}
		files[name] = c
	}
	return files, base
}

// TestEveryTokenValidatorTrustsPinnedIssuer (TICK-291, merge-safety D9): every
// service that validates tokens trusts exactly the issuer Keycloak is pinned
// to (KC_HOSTNAME + /realms/nava, TICK-160). TICK-278 was nava-onboard still
// trusting keycloak:8080 after nava-core was pinned.
func TestEveryTokenValidatorTrustsPinnedIssuer(t *testing.T) {
	t.Parallel()
	files, base := loadStackFiles(t)
	for _, e := range checkIssuers(files, base) {
		t.Error(e)
	}
}

// TestCheckIssuersCatchesDrift: the check fails, naming the service, on each
// way a service can drift off the pin.
func TestCheckIssuersCatchesDrift(t *testing.T) {
	t.Parallel()
	const host = "${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}"
	const pin = host + realmPath
	good := func() map[string]composeFile {
		return map[string]composeFile{"compose.yaml": {
			"keycloak":  {Env: map[string]string{"KC_HOSTNAME": host}},
			"nava-core": {Env: map[string]string{"NAVA_OIDC_ISSUERS": pin, "NAVA_OIDC_FETCH_BASE_URL": "http://keycloak:8080"}},
			"nava-onboard": {Env: map[string]string{
				"NAVA_OIDC_ISSUERS": pin, "NAVA_OIDC_FETCH_BASE_URL": "http://keycloak:8080",
				"NAVA_ONBOARD_ISSUER": pin, "NAVA_OPERATORS": pin + "|00000000-0000-4000-8000-00000000f001",
			}},
			"nava-worker": {Env: map[string]string{"NAVA_LOG_FORMAT": "json"}},
		}}
	}
	if errs := checkIssuers(good(), "compose.yaml"); len(errs) != 0 {
		t.Fatalf("good stack: %v", errs)
	}
	cases := []struct {
		name   string
		mutate func(map[string]composeFile)
		want   string
	}{
		{"onboard trusts the back channel", func(f map[string]composeFile) {
			f["compose.yaml"]["nava-onboard"].Env["NAVA_OIDC_ISSUERS"] = "http://keycloak:8080/realms/nava"
		}, `nava-onboard NAVA_OIDC_ISSUERS trusts "http://keycloak:8080/realms/nava"`},
		{"second issuer on the allow-list", func(f map[string]composeFile) {
			f["compose.yaml"]["nava-core"].Env["NAVA_OIDC_ISSUERS"] = pin + ",http://keycloak:8080/realms/nava"
		}, "nava-core NAVA_OIDC_ISSUERS trusts"},
		{"onboard binding issuer differs", func(f map[string]composeFile) {
			f["compose.yaml"]["nava-onboard"].Env["NAVA_ONBOARD_ISSUER"] = "http://keycloak:8080/realms/nava"
		}, "nava-onboard NAVA_ONBOARD_ISSUER trusts"},
		{"operator issuer differs", func(f map[string]composeFile) {
			f["compose.yaml"]["nava-onboard"].Env["NAVA_OPERATORS"] = "http://keycloak:8080/realms/nava|x"
		}, "nava-onboard NAVA_OPERATORS trusts"},
		{"issuer removed from a known validator", func(f map[string]composeFile) {
			delete(f["compose.yaml"]["nava-onboard"].Env, "NAVA_OIDC_ISSUERS")
		}, "nava-onboard validates tokens but sets no NAVA_OIDC_ISSUERS"},
		{"all OIDC vars removed from a known validator", func(f map[string]composeFile) {
			f["compose.yaml"]["nava-core"] = service{Env: map[string]string{}}
		}, "nava-core validates tokens but sets no NAVA_OIDC_ISSUERS"},
		{"new service fetches keys without an issuer", func(f map[string]composeFile) {
			f["compose.yaml"]["nava-worker"].Env["NAVA_OIDC_FETCH_BASE_URL"] = "http://keycloak:8080"
		}, "nava-worker validates tokens but sets no NAVA_OIDC_ISSUERS"},
		{"empty issuer", func(f map[string]composeFile) {
			f["compose.yaml"]["nava-core"].Env["NAVA_OIDC_ISSUERS"] = ""
		}, "nava-core sets NAVA_OIDC_ISSUERS empty"},
		{"pin by default only, through another variable", func(f map[string]composeFile) {
			f["compose.yaml"]["nava-core"].Env["NAVA_OIDC_ISSUERS"] = "${CORE_ISSUER:-http://localhost:18081}/realms/nava"
		}, "nava-core NAVA_OIDC_ISSUERS is"},
		{"validator reads an env_file", func(f map[string]composeFile) {
			s := f["compose.yaml"]["nava-core"]
			s.EnvFile = true
			f["compose.yaml"]["nava-core"] = s
		}, "nava-core validates tokens and reads an env_file"},
		{"known validator missing", func(f map[string]composeFile) {
			delete(f["compose.yaml"], "nava-onboard")
		}, "token-validating service nava-onboard is missing"},
		{"overlay moves an issuer", func(f map[string]composeFile) {
			f["deploy/compose/compose.x.yaml"] = composeFile{"nava-onboard": {Env: map[string]string{"NAVA_OIDC_ISSUERS": "http://keycloak:8080/realms/nava"}}}
		}, "deploy/compose/compose.x.yaml: nava-onboard NAVA_OIDC_ISSUERS trusts"},
		{"overlay moves KC_HOSTNAME", func(f map[string]composeFile) {
			f["deploy/compose/compose.x.yaml"] = composeFile{"keycloak": {Env: map[string]string{"KC_HOSTNAME": "http://keycloak:8080"}}}
		}, "deploy/compose/compose.x.yaml: keycloak overrides KC_HOSTNAME"},
		{"no pin", func(f map[string]composeFile) {
			f["compose.yaml"]["keycloak"] = service{Env: map[string]string{}}
		}, "keycloak sets no KC_HOSTNAME"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := good()
			c.mutate(f)
			errs := checkIssuers(f, "compose.yaml")
			if !strings.Contains(strings.Join(errs, "\n"), c.want) {
				t.Errorf("errors %q, want one containing %q", errs, c.want)
			}
		})
	}
}

// TestParseComposeListEnvironment: the list form of environment is read too,
// so writing it that way does not hide a service from the check.
func TestParseComposeListEnvironment(t *testing.T) {
	t.Parallel()
	c, err := parseCompose([]byte("services:\n  a:\n    environment:\n      - NAVA_OIDC_ISSUERS=http://x/realms/nava\n    env_file: [.env]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := c["a"].Env["NAVA_OIDC_ISSUERS"]; got != "http://x/realms/nava" || !c["a"].EnvFile {
		t.Errorf("parsed %+v", c["a"])
	}
}
