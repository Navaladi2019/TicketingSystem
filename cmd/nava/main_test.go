package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
	vt "github.com/Navaladi2019/TicketingSystem/internal/audit/verify/verifytest"
)

func chainCase(t *testing.T, name string) vectors.ChainCase {
	t.Helper()
	for _, c := range vt.Chain() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no chain vector %s", name)
	return vectors.ChainCase{}
}

func exportVector(t *testing.T, name string) []byte {
	t.Helper()
	for _, x := range vt.Vectors().Exports {
		if x.Name == name {
			return vt.JSONL(x.JSONL)
		}
	}
	t.Fatalf("no export vector %s", name)
	return nil
}

func writeFile(t *testing.T, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "export.jsonl")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// Each outcome has its own exit code and its own first word of output.
func TestExitCodes(t *testing.T) {
	erased := chainCase(t, "erased-entry")
	pending := vt.Build(vt.Options{HeadSeq: erased.Seq, Unsealed: []vectors.ChainCase{chainCase(t, "audit-erase")}}).Bytes()

	cases := []struct {
		name   string
		file   []byte
		code   int
		prefix string
		output string
	}{
		{"ok", exportVector(t, "unfiltered"), ExitOK, "ok: ", "seq 1..7 from genesis"},
		{"ok-redacted", exportVector(t, "diff-redacted-replaced"), ExitOK, "ok: ", "seq 2: diff not verifiable: redacted"},
		{"tampered", exportVector(t, "diff-dropped-not-erased"), ExitTampered, "tampered: ", "(seq 2): diff_missing"},
		{"tampered-erasure", exportVector(t, "erasure-not-listed"), ExitTampered, "tampered: ", "erasure_unlisted"},
		{"pending", pending, ExitPending, "pending: ", "seq " + strconv.FormatInt(erased.Seq, 10) + ": pending"},
		{"stubs-omitted", exportVector(t, "stubs-false-honest"), ExitUnverifiable, "not verifiable: stubs omitted", "gaps"},
		{"empty", exportVector(t, "empty-range"), ExitEmpty, "empty: ", "no sealed entry"},
	}
	seen := map[int]string{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := run([]string{"audit", "verify", writeFile(t, c.file)}, nil, &out, &errOut)
			if code != c.code {
				t.Fatalf("exit %d, want %d; stdout %q stderr %q", code, c.code, out.String(), errOut.String())
			}
			if !strings.HasPrefix(out.String(), c.prefix) || !strings.Contains(out.String(), c.output) {
				t.Errorf("stdout %q, want prefix %q containing %q", out.String(), c.prefix, c.output)
			}
		})
		if prev, ok := seen[c.code]; ok && !strings.HasPrefix(c.name, prev) {
			t.Errorf("exit %d shared by %s and %s", c.code, prev, c.name)
		}
		seen[c.code] = c.name
	}
	codes := []int{ExitOK, ExitTampered, ExitUsage, ExitPending, ExitUnverifiable, ExitEmpty}
	for i, a := range codes {
		for _, b := range codes[i+1:] {
			if a == b {
				t.Errorf("exit code %d used twice", a)
			}
		}
	}
}

// A CSV export gets the same outcome, exit code and report as its JSONL
// twin (R22), and a failure names the CSV row.
func TestCSVExitCodes(t *testing.T) {
	for _, c := range []struct {
		name, output string
		code         int
	}{
		{"unfiltered", "ok: seq 1..7 from genesis", ExitOK},
		{"diff-redacted-replaced", "seq 2: diff not verifiable: redacted", ExitOK},
		{"diff-dropped-not-erased", "tampered: row 5 (seq 2): diff_missing", ExitTampered},
		{"erasure-not-listed", "erasure_unlisted", ExitTampered},
		{"stubs-false-honest", "not verifiable: stubs omitted", ExitUnverifiable},
		{"empty-range", "empty: ", ExitEmpty},
	} {
		t.Run(c.name, func(t *testing.T) {
			var x vectors.Export
			for _, e := range vt.Vectors().Exports {
				if e.Name == c.name {
					x = e
				}
			}
			var jsonl, csv, errOut bytes.Buffer
			jcode := run([]string{"audit", "verify", writeFile(t, vt.JSONL(x.JSONL))}, nil, &jsonl, &errOut)
			code := run([]string{"audit", "verify", writeFile(t, vt.CSV(x.JSONL))}, nil, &csv, &errOut)
			if code != c.code || jcode != c.code {
				t.Fatalf("exit csv %d jsonl %d, want %d; stdout %q stderr %q", code, jcode, c.code, csv.String(), errOut.String())
			}
			if !strings.Contains(csv.String(), c.output) {
				t.Errorf("stdout %q, want %q", csv.String(), c.output)
			}
			if code != ExitTampered && csv.String() != jsonl.String() {
				t.Errorf("csv report %q differs from jsonl %q", csv.String(), jsonl.String())
			}
		})
	}
	var out bytes.Buffer
	pending := vt.Build(vt.Options{HeadSeq: chainCase(t, "erased-entry").Seq, Unsealed: []vectors.ChainCase{chainCase(t, "audit-erase")}})
	if code := run([]string{"audit", "verify", "-"}, bytes.NewReader(pending.CSV()), &out, &out); code != ExitPending {
		t.Errorf("pending csv on stdin: exit %d, want %d: %s", code, ExitPending, out.String())
	}
}

func TestStdin(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"audit", "verify", "-"}, bytes.NewReader(exportVector(t, "unfiltered")), &out, &out); code != ExitOK {
		t.Fatalf("exit %d: %s", code, out.String())
	}
}

func TestUsage(t *testing.T) {
	for _, args := range [][]string{
		nil, {"audit"}, {"audit", "verify"}, {"audit", "verify", "a", "b"}, {"frobnicate"},
		{"audit", "verify", filepath.Join(t.TempDir(), "missing.jsonl")},
	} {
		var out, errOut bytes.Buffer
		if code := run(args, nil, &out, &errOut); code != ExitUsage {
			t.Errorf("%q: exit %d, want %d", args, code, ExitUsage)
		}
		if errOut.Len() == 0 {
			t.Errorf("%q: nothing on stderr", args)
		}
	}
}

// The help documents every exit code and every reason code.
func TestHelpDocumentsExitCodes(t *testing.T) {
	var out bytes.Buffer
	if code := run([]string{"audit", "verify", "-h"}, nil, &out, &out); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	help := out.String()
	for code, word := range map[int]string{
		ExitOK: "ok", ExitTampered: "tampered", ExitUsage: "usage", ExitPending: "pending",
		ExitUnverifiable: "not verifiable: stubs omitted", ExitEmpty: "empty",
	} {
		if want := "  " + strconv.Itoa(code) + "  " + word; !strings.Contains(help, want) {
			t.Errorf("help lacks %q", want)
		}
	}
	// The help's reason list is exactly the audit-export-v1.md §4.1 table, so
	// a code added to §4.1 (and to internal/audit/verify, which
	// TestReasonCodesListedInSpec enforces) must be added to the help too.
	spec := specReasonCodes(t)
	listed := helpReasonCodes(t, help)
	for code := range spec {
		if !listed[code] {
			t.Errorf("help lacks reason %s (audit-export-v1.md §4.1)", code)
		}
	}
	for code := range listed {
		if !spec[code] {
			t.Errorf("help lists reason %s, which audit-export-v1.md §4.1 does not", code)
		}
	}
	if !strings.Contains(help, "does not prove") {
		t.Error("help does not state what a pass does not prove (§4 step 9)")
	}
	out.Reset()
	if code := run([]string{"help"}, nil, &out, &out); code != ExitOK || !strings.Contains(out.String(), "nava audit verify") {
		t.Errorf("nava help: exit %d, %q", code, out.String())
	}
}

var (
	specRow  = regexp.MustCompile("^\\| `([a-z0-9_]+)` \\|")
	helpCode = regexp.MustCompile(`[a-z0-9]+(?:_[a-z0-9]+)+`)
)

// specReasonCodes reads the first column of the audit-export-v1.md §4.1 table.
func specReasonCodes(t *testing.T) map[string]bool {
	t.Helper()
	b, err := os.ReadFile("../../docs/api/audit-export-v1.md")
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	in := false
	for _, l := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(l, "### 4.1 Reason codes"):
			in = true
		case in && strings.HasPrefix(l, "#"):
			in = false
		case in:
			if m := specRow.FindStringSubmatch(l); m != nil {
				codes[m[1]] = true
			}
		}
	}
	if len(codes) == 0 {
		t.Fatal("no reason codes read from audit-export-v1.md §4.1")
	}
	return codes
}

// helpReasonCodes reads the codes in the help's "Reason codes" paragraph.
func helpReasonCodes(t *testing.T, help string) map[string]bool {
	t.Helper()
	i := strings.Index(help, "Reason codes (exit 1)")
	if i < 0 {
		t.Fatal("help has no Reason codes paragraph")
	}
	para, _, _ := strings.Cut(help[i:], "\n\n")
	_, list, _ := strings.Cut(para, ":\n")
	codes := map[string]bool{}
	for _, c := range helpCode.FindAllString(list, -1) {
		codes[c] = true
	}
	return codes
}
