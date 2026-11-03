package api_test

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/Navaladi2019/TicketingSystem/api"
)

// TICK-510: every operation in the bundle, and every status it documents,
// needs a //go:build integration test that calls it through the real handler
// against the throwaway PostgreSQL of scripts/ci/test-integration.sh, or a
// reason in integration-coverage-exempt.json. The link is a marker in the
// test's doc comment, one operation and one or more statuses per line:
//
//	// covers: createRecord 201 409
//	func TestCreateRecordRetryWithTheSameKeyCreatesOne(t *testing.T) {
//
// The gate is mechanical; whether the test asserts the rows written or read
// (and that a refused request writes nothing) is the reviewer's check
// (docs/delivery/review-standard-core.md, Tests). It runs in the unit stage,
// so a new endpoint or status without a test fails the PR.

const (
	integrationExemptFile = "integration-coverage-exempt.json"
	coversPrefix          = "covers:"
)

type coverMarker struct {
	op, status, where string
}

// scanCoverMarkers returns the covers: markers of one Go test file and the
// problems with them: a marker outside the doc comment of a func TestXxx, or
// in a file that is not built with the integration tag, does not count.
func scanCoverMarkers(path string, src []byte) ([]coverMarker, []string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	var markers []coverMarker
	var problems []string
	inDoc := map[*ast.Comment]string{}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Doc == nil || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		for _, c := range fn.Doc.List {
			inDoc[c] = fn.Name.Name
		}
	}
	integration := isIntegrationFile(f)
	for _, g := range f.Comments {
		for _, c := range g.List {
			text, ok := strings.CutPrefix(c.Text, "//")
			if !ok {
				continue
			}
			rest, ok := strings.CutPrefix(strings.TrimSpace(text), coversPrefix)
			if !ok {
				continue
			}
			pos := fset.Position(c.Pos())
			where := fmt.Sprintf("%s:%d", filepath.ToSlash(pos.Filename), pos.Line)
			test, ok := inDoc[c]
			switch {
			case !integration:
				problems = append(problems, where+": covers: marker in a file without //go:build integration; it never runs against PostgreSQL")
				continue
			case !ok:
				problems = append(problems, where+": covers: marker outside the doc comment of a func TestXxx(t *testing.T)")
				continue
			}
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				problems = append(problems, where+": covers: wants <operationId> <status> [<status>...]")
				continue
			}
			for _, s := range fields[1:] {
				markers = append(markers, coverMarker{op: fields[0], status: s, where: where + " " + test})
			}
		}
	}
	return markers, problems
}

func isIntegrationFile(f *ast.File) bool {
	for _, g := range f.Comments {
		if g.Pos() >= f.Package {
			break
		}
		for _, c := range g.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				return false
			}
			return expr.Eval(func(tag string) bool { return tag == "integration" })
		}
	}
	return false
}

func integrationCoverMarkers(t *testing.T) ([]coverMarker, []string) {
	t.Helper()
	var markers []coverMarker
	var problems []string
	// The trees test-integration.sh searches for //go:build integration
	// tests, relative to this package.
	for _, root := range []string{"../cmd", "../internal", "../tools"} {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if n := d.Name(); n == "testdata" || n == "vendor" || n == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, "_test.go") {
				return nil
			}
			src, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if !strings.Contains(string(src), coversPrefix) {
				return nil
			}
			m, pr := scanCoverMarkers(p, src)
			markers, problems = append(markers, m...), append(problems, pr...)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	return markers, problems
}

func TestIntegrationTestsCoverEveryOperationAndStatus(t *testing.T) {
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
	// documented[op][status] = "METHOD /path"
	documented := map[string]map[string]string{}
	for path, item := range spec.Paths {
		for method, op := range item {
			method = strings.ToUpper(method)
			if method == "PARAMETERS" || method == "SUMMARY" || method == "DESCRIPTION" {
				continue
			}
			documented[op.OperationID] = map[string]string{}
			for s := range op.Responses {
				documented[op.OperationID][s] = method + " " + path
			}
		}
	}
	if len(documented) == 0 {
		t.Fatal("no operations in the bundle")
	}

	raw, err := os.ReadFile(integrationExemptFile)
	if err != nil {
		t.Fatal(err)
	}
	var exemptRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw, &exemptRaw); err != nil {
		t.Fatalf("%s: %v", integrationExemptFile, err)
	}
	exempt := map[string]map[string]string{}
	for op, m := range exemptRaw {
		if op == "_doc" {
			continue
		}
		reasons := map[string]string{}
		if err := json.Unmarshal(m, &reasons); err != nil {
			t.Fatalf("%s %s: %v", integrationExemptFile, op, err)
		}
		exempt[op] = reasons
	}

	markers, problems := integrationCoverMarkers(t)
	for _, p := range problems {
		t.Error(p)
	}
	for _, p := range checkCoverage(documented, markers, exempt) {
		t.Error(p)
	}
}

// checkCoverage returns, sorted, every documented operation and status with
// neither a marker nor a reason, every marker or exemption naming an
// operation and status the bundle does not document, and every exemption a
// marker already satisfies. documented maps operation, then status, to
// "METHOD /path"; exempt maps operation, then status, to the reason.
func checkCoverage(documented map[string]map[string]string, markers []coverMarker, exempt map[string]map[string]string) []string {
	var problems []string
	covered := map[string]map[string][]string{}
	for _, m := range markers {
		if _, ok := documented[m.op][m.status]; !ok {
			problems = append(problems, fmt.Sprintf("%s: covers: %s %s, but the bundle documents no such operation and status", m.where, m.op, m.status))
			continue
		}
		if covered[m.op] == nil {
			covered[m.op] = map[string][]string{}
		}
		covered[m.op][m.status] = append(covered[m.op][m.status], m.where)
	}
	for op, statuses := range documented {
		for s, route := range statuses {
			tests, reason := covered[op][s], strings.TrimSpace(exempt[op][s])
			switch {
			case len(tests) == 0 && reason == "":
				problems = append(problems, fmt.Sprintf("%s (%s) %s: no //go:build integration test has `// covers: %s %s` and %s gives no reason",
					op, route, s, op, s, integrationExemptFile))
			case len(tests) > 0 && reason != "":
				problems = append(problems, fmt.Sprintf("%s %s: covered by %s; remove its exemption from %s", op, s, tests[0], integrationExemptFile))
			}
		}
	}
	for op, reasons := range exempt {
		for s := range reasons {
			if _, ok := documented[op][s]; !ok {
				problems = append(problems, fmt.Sprintf("%s exempts %s %s, which the bundle does not document; remove it", integrationExemptFile, op, s))
			}
		}
	}
	sort.Strings(problems)
	return problems
}
