package verify_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	vt "github.com/Navaladi2019/TicketingSystem/internal/audit/verify/verifytest"
)

//nolint:gochecknoglobals // the standard golden-file -update flag
var update = flag.Bool("update", false, "rewrite the testdata/csv fixtures")

// encoding is one way of writing the same export: the JSONL tests run on
// both, so the CSV reader is held to every JSONL check (ADR-004 R22).
type encoding struct {
	name  string
	file  func(*vt.File) []byte
	lines func([]string) []byte
	// before is the number of rows before the export's header line.
	before int
}

func encodings() []encoding {
	return []encoding{
		{"jsonl", (*vt.File).Bytes, vt.JSONL, 1},
		{"csv", (*vt.File).CSV, vt.CSV, 2},
	}
}

func eachEncoding(t *testing.T, f func(*testing.T, encoding)) {
	t.Helper()
	for _, e := range encodings() {
		t.Run(e.name, func(t *testing.T) { f(t, e) })
	}
}

func encodingOf(b []byte) verify.Encoding {
	if bytes.HasPrefix(b, []byte("kind,")) {
		return verify.EncodingCSV
	}
	return verify.EncodingJSONL
}

// rows splits a CSV fixture into its rows, each with its CRLF. The fixtures
// built here quote any cell holding a line break, and none of the clean ones
// has one, so splitting on CRLF is exact for them.
func rows(t *testing.T, b []byte) []string {
	t.Helper()
	rs := strings.SplitAfter(string(b), "\r\n")
	if rs[len(rs)-1] != "" {
		t.Fatal("fixture does not end with CRLF")
	}
	return rs[:len(rs)-1]
}

// §5 and §4 steps 1 and 10 for CSV: rows are RFC 4180 and strict, and each
// row is one strict line of its kind.
func TestCSVStrictStructure(t *testing.T) {
	clean := vt.Build(vt.Options{}).CSV()
	rs := rows(t, clean)
	n := len(rs)
	cols := len(verify.CSVColumns)
	join := func(ls ...string) []byte { return []byte(strings.Join(ls, "")) }
	rest := func(i int) string { return strings.Join(rs[i:], "") }
	// Row 4 is the seq 1 entry (column row, header, anchor, entry).
	if !strings.HasPrefix(rs[3], "entry,1,") {
		t.Fatalf("row 4 is not the seq 1 entry")
	}
	// edited is the clean file with the seq 1 entry line changed before it
	// is written, so the writer still quotes and prefixes it.
	edited := func(edit func(vt.Line)) []byte {
		f := vt.Build(vt.Options{})
		edit(f.Body[f.Index(1)])
		return f.CSV()
	}
	// replaced changes the first occurrence of old, which is in row 4.
	replaced := func(old, repl string) []byte {
		i := strings.Index(string(clean), old)
		if i < len(strings.Join(rs[:3], "")) || i >= len(strings.Join(rs[:4], "")) {
			t.Fatalf("%q is not first found in row 4", old)
		}
		return []byte(strings.Replace(string(clean), old, repl, 1))
	}
	trailer := strings.Replace(rs[n-1], `"{""anchor_seq""`, `"{`+"\n"+`""anchor_seq""`, 1)
	if trailer == rs[n-1] {
		t.Fatal("trailer meta not found")
	}
	cases := []struct {
		name, reason string
		file         []byte
		line         int
	}{
		{"column-row-only", verify.ReasonHeaderInvalid, join(rs[0]), 2},
		{"column-row-reordered", verify.ReasonHeaderInvalid,
			join(strings.Replace(rs[0], "kind,seq,prev_hash", "kind,prev_hash,seq", 1), rest(1)), 1},
		{"column-row-extra", verify.ReasonHeaderInvalid, join(strings.Replace(rs[0], ",meta", ",meta,x", 1), rest(1)), 1},
		{"lf-row-ends", verify.ReasonHeaderInvalid, []byte(strings.ReplaceAll(string(clean), "\r\n", "\n")), 1},
		{"no-header-row", verify.ReasonHeaderInvalid, join(rs[0], rest(2)), 2},
		{"header-row-with-a-seq", verify.ReasonHeaderInvalid, join(rs[0], strings.Replace(rs[1], "header,", "header,1", 1), rest(2)), 2},
		{"header-meta-is-a-trailer", verify.ReasonHeaderInvalid,
			join(rs[0], "header"+strings.Repeat(",", cols-1)+`"{""kind"":""trailer""}"`+"\r\n", rest(2)), 2},
		{"no-trailer", verify.ReasonTrailerMissing, join(rs[:n-1]...), n},
		{"row-after-trailer", verify.ReasonLineOrder, join(rest(0), rs[2]), n + 1},
		{"blank-row", verify.ReasonMalformedLine, join(rs[0], rs[1], "\r\n", rest(2)), 3},
		{"unknown-kind", verify.ReasonUnknownKind, join(rs[0], rs[1], "note"+strings.Repeat(",", cols-1)+"\r\n", rest(2)), 3},
		{"bare-cr", verify.ReasonMalformedLine, replaced(",config,", ",con\rfig,"), 4},
		{"quote-in-unquoted-cell", verify.ReasonMalformedLine, replaced(",config,", `,con"fig,`), 4},
		{"data-after-closing-quote", verify.ReasonMalformedLine, replaced(",config,", `,"config"x,`), 4},
		{"too-few-cells", verify.ReasonMalformedLine, replaced(",config,", ","), 4},
		{"invalid-utf8", verify.ReasonMalformedLine, replaced(",config,", ",con\xfffig,"), 4},
		{"envelope-uppercase-uuid", verify.ReasonMalformedLine,
			replaced("0190f3a0-0000-7000-8000-0000000000d1", "0190F3A0-0000-7000-8000-0000000000D1"), 4},
		{"eof-in-quoted-cell", verify.ReasonMalformedLine, join(rs[0], rs[1], rs[2], `entry,"1`), 4},
		{"unused-column-set", verify.ReasonMalformedLine, edited(func(l vt.Line) { l["genesis"] = false }), 4},
		{"meta-on-an-entry", verify.ReasonMalformedLine, edited(func(l vt.Line) { l["meta"] = "x" }), 4},
		{"boolean-not-lowercase", verify.ReasonMalformedLine, edited(func(l vt.Line) { l["sealed"] = "TRUE" }), 4},
		{"integer-with-fraction", verify.ReasonMalformedLine, edited(func(l vt.Line) { l["seq"] = "1.0" }), 4},
		{"integer-injection", verify.ReasonMalformedLine, edited(func(l vt.Line) { l["seq"] = `1,"x":2` }), 4},
		{"non-null-made-empty", verify.ReasonMalformedLine, edited(func(l vt.Line) { l["diff_redacted"] = nil }), 4},
		{"meta-with-a-line-break", verify.ReasonMalformedLine, join(append(slicesClone(rs[:n-1]), trailer)...), n},
		{"row-too-long", verify.ReasonMalformedLine, join(rs[0], `header,"`, strings.Repeat("a", verify.MaxLineBytes)), 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := run(t, c.file)
			if reason(r) != c.reason || r.Failure.Line != c.line {
				t.Errorf("got %s, want %s at row %d", describe(r), c.reason, c.line)
			}
		})
	}
	wantOutcome(t, run(t, clean), verify.OutcomePass)
	// A missing final CRLF is tolerated, as a missing final "\n" is in JSONL.
	wantOutcome(t, run(t, bytes.TrimSuffix(clean, []byte("\r\n"))), verify.OutcomePass)
}

// rewriteHead edits the envelope of the file's last range line and rehashes
// it, then moves the trailer's head_hash to match: a clean file whose head
// entry carries the edited values.
func rewriteHead(f *vt.File, edit func(*chain.Envelope, vt.Line)) {
	i := len(f.Body) - 1
	l := f.Body[i]
	env := l["envelope"].(chain.Envelope) //nolint:forcetypeassert // the builder's own entry line
	edit(&env, l)
	l["envelope"] = env
	f.Rehash(i)
	f.Trailer["head_hash"] = l["entry_hash"]
}

// CI-A15, reader half (R22): a cell beginning with = + - @ tab CR or ' is
// written with one ' prefixed, and the reader drops exactly that one, so the
// hashes still verify. A prefix the reader did not drop, or dropped twice,
// would change the envelope and fail.
func TestCSVFormulaPrefix(t *testing.T) {
	for _, v := range []string{"=1+1", "+1", "-1", "@SUM(A1)", "\tx", "\rx", "'quoted", "''twice"} {
		t.Run(strings.NewReplacer("\t", "tab", "\r", "cr").Replace(v), func(t *testing.T) {
			f := vt.Build(vt.Options{})
			rewriteHead(f, func(env *chain.Envelope, _ vt.Line) { env.RequestID = &v })
			b := f.CSV()
			if !bytes.Contains(b, []byte("'"+v)) {
				t.Fatalf("the writer did not prefix %q", v)
			}
			wantOutcome(t, run(t, b), verify.OutcomePass)
			// Without the prefix the reader drops a ' that is part of the
			// value; the other values are read as written.
			r := run(t, bytes.Replace(b, []byte("'"+v), []byte(v), 1))
			if strings.HasPrefix(v, "'") {
				if reason(r) != verify.ReasonEnvelopeMismatch {
					t.Errorf("unprefixed %q: got %s, want %s", v, describe(r), verify.ReasonEnvelopeMismatch)
				}
			} else {
				wantOutcome(t, r, verify.OutcomePass)
			}
		})
	}
}

// §5: the diff cell holds the stored bytes verbatim. A stored diff with LF
// and CRLF inside verifies, so the reader normalises neither.
func TestCSVDiffLineEndsKept(t *testing.T) {
	f := vt.Build(vt.Options{})
	var stored string
	rewriteHead(f, func(env *chain.Envelope, l vt.Line) {
		d := *l["diff"].(*string) //nolint:forcetypeassert // the builder's own entry line
		stored = strings.Replace(strings.Replace(d, `,"path"`, ",\r\n\"path\"", 1), `,"value"`, ",\n\"value\"", 1)
		salt, err := chain.ParseSalt16(*l["diff_salt"].(*string)) //nolint:forcetypeassert // as above
		if err != nil {
			t.Fatal(err)
		}
		h := chain.DiffSHA256(salt, []byte(stored))
		env.DiffSHA256 = &h
		l["diff"] = &stored
	})
	if !strings.Contains(stored, "\r\n") || !strings.Contains(stored, "\n\"value\"") {
		t.Fatalf("diff %q lacks the line ends under test", stored)
	}
	b := f.CSV()
	wantOutcome(t, run(t, b), verify.OutcomePass)
	// A reader that turned CRLF into LF would see this file.
	norm := bytes.Replace(b, []byte(",\r\n\"\"path"), []byte(",\n\"\"path"), 1)
	if bytes.Equal(norm, b) {
		t.Fatal("CRLF not found in the diff cell")
	}
	if r := run(t, norm); reason(r) != verify.ReasonDiffMismatch {
		t.Errorf("normalised diff: got %s, want %s", describe(r), verify.ReasonDiffMismatch)
	}
}

// A file is read as CSV exactly when it starts with the column row.
func TestEncodingDetected(t *testing.T) {
	f := vt.Build(vt.Options{})
	for _, e := range encodings() {
		r := run(t, e.file(f))
		if string(r.Encoding) != e.name {
			t.Errorf("%s file read as %s", e.name, r.Encoding)
		}
	}
}

const csvDir = "testdata/csv"

// TestCSVFixturesAreCurrent keeps one CSV file per golden export vector in
// testdata/csv, built by vt.CSV from the P5-02 generator, and checks that
// each, read from disk, gives the vector's outcome. Rebuild with
// go test ./internal/audit/verify -run TestCSVFixturesAreCurrent -update.
func TestCSVFixturesAreCurrent(t *testing.T) {
	want := map[string]bool{}
	for _, x := range vt.Vectors().Exports {
		name := x.Name + ".csv"
		want[name] = true
		b := vt.CSV(x.JSONL)
		p := filepath.Join(csvDir, name)
		if *update {
			if err := os.MkdirAll(csvDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, b, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("%s: %v (rebuild with -update)", name, err)
		}
		if !bytes.Equal(got, b) {
			t.Errorf("%s differs from what the P5-02 generator builds; never edit a fixture by hand, rebuild with -update", name)
			continue
		}
		if r := run(t, got); string(r.Outcome) != x.Expect {
			t.Errorf("%s: outcome %s (%s), want %s", name, r.Outcome, describe(r), x.Expect)
		}
	}
	files, err := filepath.Glob(filepath.Join(csvDir, "*.csv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if !want[filepath.Base(f)] {
			t.Errorf("%s is in testdata/csv but no export vector builds it", filepath.Base(f))
		}
	}
}
