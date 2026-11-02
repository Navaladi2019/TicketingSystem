package main

import (
	"bytes"
	"os"
	"path/filepath"
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
	for _, r := range []string{
		"envelope_sha256_mismatch", "entry_hash_mismatch", "diff_sha256_mismatch", "diff_missing",
		"diff_state_mismatch", "header_tenant_mismatch", "erasure_unlisted", "verifiable_flag_mismatch",
		"anchor_missing", "anchor_invalid", "anchor_mismatch", "seq_duplicate", "seq_out_of_order",
		"seq_missing", "count_mismatch", "head_mismatch", "line_state_mismatch", "line_order",
		"header_invalid", "trailer_missing", "unknown_kind", "malformed_line",
	} {
		if !strings.Contains(help, r) {
			t.Errorf("help lacks reason %s", r)
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
