package api_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/Navaladi2019/TicketingSystem/api"
)

// The acceptance kit's Bruno collection is the API test suite (TICK-331).
// This test keeps it complete: every operation in the bundle needs at least
// one Bruno request, and every status the operation documents needs a request
// whose tests assert it, or a reason in coverage-exempt.json. A new endpoint
// fails here until the suite (docs/acceptance/guide/build/collection.py) tests
// it. build.py runs the same check when it regenerates the kit.

const (
	brunoDir   = "../docs/acceptance/guide/bruno"
	exemptFile = "../docs/acceptance/guide/build/coverage-exempt.json"
)

type brunoRequest struct {
	file, method, path string
	statuses           map[string]bool
}

var (
	bruMethod = regexp.MustCompile(`(?m)^(get|put|post|delete|patch|head|options) \{\n  url: (\S+)`)
	bruTests  = regexp.MustCompile(`(?ms)^tests \{\n(.*?)\n\}`)
	// N.problem('code', 401), N.status(201), N.me(...) (a 200).
	bruStatus = regexp.MustCompile(`N\.problem\('[a-z_]+', (\d{3})\)|N\.status\((\d{3})\)|N\.(me)\(`)
	bruVar    = regexp.MustCompile(`^\{\{[a-z_]+\}\}`)
	pathParam = regexp.MustCompile(`\\\{[^}]+\\\}`)
)

func brunoRequests(t *testing.T) []brunoRequest {
	t.Helper()
	var out []brunoRequest
	err := filepath.WalkDir(brunoDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".bru") {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		m := bruMethod.FindStringSubmatch(text)
		if m == nil {
			return nil // folder.bru, collection.bru, environments
		}
		path, _, _ := strings.Cut(bruVar.ReplaceAllString(m[2], ""), "?")
		r := brunoRequest{file: p, method: strings.ToUpper(m[1]), path: path, statuses: map[string]bool{}}
		if tm := bruTests.FindStringSubmatch(text); tm != nil {
			for _, s := range bruStatus.FindAllStringSubmatch(tm[1], -1) {
				switch {
				case s[3] != "":
					r.statuses["200"] = true
				case s[1] != "":
					r.statuses[s[1]] = true
				default:
					r.statuses[s[2]] = true
				}
			}
		}
		out = append(out, r)
		return nil
	})
	if err != nil {
		t.Fatalf("read the Bruno collection %s: %v", brunoDir, err)
	}
	if len(out) == 0 {
		t.Fatalf("no Bruno requests under %s", brunoDir)
	}
	return out
}

func TestBrunoCoversEveryOperation(t *testing.T) {
	data, err := api.Files.ReadFile(api.SpecFile)
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string               `yaml:"operationId"`
			Responses   map[string]yaml.Node `yaml:"responses"`
		} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(exemptFile)
	if err != nil {
		t.Fatal(err)
	}
	var exempt map[string]json.RawMessage
	if err := json.Unmarshal(raw, &exempt); err != nil {
		t.Fatal(err)
	}
	reqs := brunoRequests(t)

	ops := 0
	for path, item := range spec.Paths {
		rx := regexp.MustCompile("^" + pathParam.ReplaceAllString(regexp.QuoteMeta(path), "[^/]+") + "$")
		for method, op := range item {
			method = strings.ToUpper(method)
			if method == "PARAMETERS" || method == "SUMMARY" || method == "DESCRIPTION" {
				continue
			}
			ops++
			asserted := map[string]bool{}
			hits := 0
			for _, r := range reqs {
				if r.method == method && rx.MatchString(r.path) {
					hits++
					for s := range r.statuses {
						asserted[s] = true
					}
				}
			}
			if hits == 0 {
				t.Errorf("%s (%s %s): no Bruno request; add one to docs/acceptance/guide/build/collection.py and run build.py", op.OperationID, method, path)
				continue
			}
			reasons := map[string]string{}
			if m, ok := exempt[op.OperationID]; ok {
				if err := json.Unmarshal(m, &reasons); err != nil {
					t.Fatalf("%s: %v", exemptFile, err)
				}
			}
			statuses := make([]string, 0, len(op.Responses))
			for s := range op.Responses {
				statuses = append(statuses, s)
			}
			sort.Strings(statuses)
			for _, s := range statuses {
				if !asserted[s] && reasons[s] == "" {
					t.Errorf("%s (%s %s) %s: no Bruno request asserts it and %s gives no reason", op.OperationID, method, path, s, exemptFile)
				}
			}
		}
	}
	if ops == 0 {
		t.Fatal("no operations in the bundle")
	}
}
