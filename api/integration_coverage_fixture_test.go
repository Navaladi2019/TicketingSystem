package api_test

import (
	"strings"
	"testing"
)

// The fixtures of the TICK-510 gate (integration_coverage_test.go), written
// before it: each case is a bundle, the markers found and the exemptions, and
// the problems the gate must report.

func TestCheckCoverage(t *testing.T) {
	t.Parallel()
	documented := map[string]map[string]string{
		"getRecord": {"200": "GET /r/{id}", "404": "GET /r/{id}", "501": "GET /r/{id}"},
	}
	mark := func(op, status string) coverMarker {
		return coverMarker{op: op, status: status, where: "x_test.go:1 TestX"}
	}
	full := []coverMarker{mark("getRecord", "200"), mark("getRecord", "404")}
	ok501 := map[string]map[string]string{"getRecord": {"501": "generated default"}}
	cases := []struct {
		name    string
		markers []coverMarker
		exempt  map[string]map[string]string
		want    []string // substrings, one per problem, in sorted order
	}{
		{"missing marker is reported with operation and status", []coverMarker{mark("getRecord", "200")}, ok501,
			[]string{"getRecord (GET /r/{id}) 404: no //go:build integration test has `// covers: getRecord 404`"}},
		{"every status covered or exempt", full, ok501, nil},
		{"empty reason is no exemption", full, map[string]map[string]string{"getRecord": {"501": "  "}},
			[]string{"getRecord (GET /r/{id}) 501: no //go:build integration test"}},
		{"marker for an undocumented status", append(full, mark("getRecord", "418")), ok501,
			[]string{"covers: getRecord 418, but the bundle documents no such operation and status"}},
		{"marker for an unknown operation", append(full, mark("getRecrod", "200")), ok501,
			[]string{"covers: getRecrod 200, but the bundle documents no such operation and status"}},
		{"exemption already covered", append(full, mark("getRecord", "501")), ok501,
			[]string{"getRecord 501: covered by x_test.go:1 TestX; remove its exemption"}},
		{"exemption for an undocumented status", full, map[string]map[string]string{"getRecord": {"501": "r", "502": "r"}},
			[]string{"exempts getRecord 502, which the bundle does not document"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := checkCoverage(documented, tc.markers, tc.exempt)
			if len(got) != len(tc.want) {
				t.Fatalf("problems %q; want %d matching %q", got, len(tc.want), tc.want)
			}
			for i, w := range tc.want {
				if !strings.Contains(got[i], w) {
					t.Errorf("problem %d = %q; want it to contain %q", i, got[i], w)
				}
			}
		})
	}
}

// TestScanCoverMarkers proves the scanner: only a marker in a TestXxx doc
// comment of an integration-tagged file counts.
func TestScanCoverMarkers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, src    string
		want         []string // op status
		wantProblems int
	}{
		{"counted", "//go:build integration\n\npackage x\n\n// Doc.\n// covers: getRecord 200 404\nfunc TestA(t *testing.T) {}\n",
			[]string{"getRecord 200", "getRecord 404"}, 0},
		{"two lines", "//go:build integration && !race\n\npackage x\n\n// covers: getRecord 200\n// covers: patchRecord 412\nfunc TestA(t *testing.T) {}\n",
			[]string{"getRecord 200", "patchRecord 412"}, 0},
		{"unit file", "package x\n\n// covers: getRecord 200\nfunc TestA(t *testing.T) {}\n", nil, 1},
		{"excluded by tag", "//go:build !integration\n\npackage x\n\n// covers: getRecord 200\nfunc TestA(t *testing.T) {}\n", nil, 1},
		{"helper", "//go:build integration\n\npackage x\n\n// covers: getRecord 200\nfunc helper(t *testing.T) {}\n", nil, 1},
		{"in body", "//go:build integration\n\npackage x\n\nfunc TestA(t *testing.T) {\n\t// covers: getRecord 200\n}\n", nil, 1},
		{"no status", "//go:build integration\n\npackage x\n\n// covers: getRecord\nfunc TestA(t *testing.T) {}\n", nil, 1},
		{"detached", "//go:build integration\n\npackage x\n\n// covers: getRecord 200\n\nfunc TestA(t *testing.T) {}\n", nil, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			markers, problems := scanCoverMarkers("x_test.go", []byte(tc.src))
			var got []string
			for _, m := range markers {
				got = append(got, m.op+" "+m.status)
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") || len(problems) != tc.wantProblems {
				t.Errorf("markers %v problems %v; want %v and %d problems", got, problems, tc.want, tc.wantProblems)
			}
		})
	}
}
