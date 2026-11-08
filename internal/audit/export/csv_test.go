package export_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/export"
)

func runCSV(t *testing.T, l *fakeLog, pageSize int, req export.Request) ([]byte, error) {
	t.Helper()
	var buf bytes.Buffer
	ex := &export.Exporter{Source: l, PageSize: pageSize}
	s := l.head()
	s.Request = req
	err := ex.WriteCSV(t.Context(), &buf, nil, s)
	return buf.Bytes(), err
}

// runBoth exports the same session as JSONL and as CSV.
func runBoth(t *testing.T, l *fakeLog, pageSize int, req export.Request) (jsonl, csvb []byte) {
	t.Helper()
	ex := &export.Exporter{Source: l, PageSize: pageSize}
	s := l.head()
	s.Request = req
	var j, c bytes.Buffer
	if err := ex.Write(t.Context(), &j, nil, s); err != nil {
		t.Fatal(err)
	}
	if err := ex.WriteCSV(t.Context(), &c, nil, s); err != nil {
		t.Fatal(err)
	}
	return j.Bytes(), c.Bytes()
}

func readCSV(t *testing.T, b []byte) [][]string {
	t.Helper()
	recs, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
	if err != nil {
		t.Fatalf("not RFC 4180: %v\n%s", err, b)
	}
	return recs
}

// jsonl cell: the string a JSONL member would give the CSV cell.
func cellOf(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case json.Number:
		return x.String()
	}
	return "?"
}

// CI-A15: each of the six leading characters, and a leading ', is prefixed;
// every other cell is byte-identical to the JSONL value.
func TestCSVFormulaInjectionPrefixing(t *testing.T) {
	t.Parallel()
	leads := []string{"=SUM(1)", "+1", "-1", "@cmd", "\tx", "\rx", "'quoted", "plain"}
	l := newFakeLog(t, len(leads))
	for i := range leads {
		l.rows[i].Entry.ActorSystem = &leads[i]
	}
	req := export.Request{Stubs: true}
	jb, cb := runBoth(t, l, 3, req)
	recs := readCSV(t, cb)
	col := map[string]int{}
	for i, c := range recs[0] {
		col[c] = i
	}
	jl := lines(jb)
	if len(recs)-1 != len(jl) {
		t.Fatalf("csv rows %d, jsonl lines %d", len(recs)-1, len(jl))
	}
	entries := 0
	for i, ln := range jl {
		dec := json.NewDecoder(strings.NewReader(ln))
		dec.UseNumber()
		var o map[string]any
		if err := dec.Decode(&o); err != nil {
			t.Fatal(err)
		}
		row := recs[i+1]
		if o["kind"] != row[0] {
			t.Fatalf("row %d kind %q, want %v", i, row[0], o["kind"])
		}
		if o["kind"] == "header" || o["kind"] == "trailer" {
			if row[col["meta"]] != ln {
				t.Errorf("row %d meta %q != %q", i, row[col["meta"]], ln)
			}
			continue
		}
		if o["kind"] != "entry" {
			continue
		}
		entries++
		env := o["envelope"].(map[string]any)
		for name, idx := range col {
			var want any
			if v, ok := env[name]; ok {
				want = v
			} else if v, ok := o[name]; ok {
				want = v
			} else {
				continue
			}
			w := cellOf(want)
			if w != "" && strings.ContainsAny(w[:1], "=+-@\t\r'") {
				w = "'" + w
			}
			if row[idx] != w {
				t.Errorf("row %d column %s = %q, want %q", i, name, row[idx], w)
			}
		}
	}
	if entries != len(leads) {
		t.Fatalf("entries = %d", entries)
	}
	for i, v := range leads {
		cell := ""
		n := 0
		for _, r := range recs[1:] {
			if r[0] == "entry" {
				if n == i {
					cell = r[col["actor_system"]]
				}
				n++
			}
		}
		want := v
		if strings.ContainsAny(v[:1], "=+-@\t\r'") {
			want = "'" + v
		}
		if cell != want {
			t.Errorf("actor_system %q -> %q, want %q", v, cell, want)
		}
	}
}

// The CSV has the JSONL's lines in order, and its trailer is the same object.
func TestCSVRowsAndTrailerMatchJSONL(t *testing.T) {
	t.Parallel()
	for _, stubs := range []bool{true, false} {
		l := newFakeLog(t, 7)
		l.unsealed = nil
		req := export.Request{Stubs: stubs}
		jb, cb := runBoth(t, l, 2, req)
		recs := readCSV(t, cb)
		jl := lines(jb)
		if len(recs)-1 != len(jl) {
			t.Fatalf("stubs=%v: csv rows %d, jsonl lines %d", stubs, len(recs)-1, len(jl))
		}
		meta := len(recs[0]) - 1
		if recs[0][meta] != "meta" {
			t.Fatalf("last column %q", recs[0][meta])
		}
		if last := recs[len(recs)-1]; last[0] != "trailer" || last[meta] != jl[len(jl)-1] {
			t.Errorf("stubs=%v: trailer row %q, want %q", stubs, last, jl[len(jl)-1])
		}
		if recs[1][0] != "header" || recs[1][meta] != jl[0] {
			t.Errorf("header row %q", recs[1])
		}
	}
}

func TestCSVCRLFRowEndsAndDiffKeepsLF(t *testing.T) {
	t.Parallel()
	l := newFakeLog(t, 2)
	l.rows[0].Entry.Diff = []byte("[\n{\"op\":\"add\",\"path\":\"/a,b\",\"value\":\"x\"}\n]")
	cb, err := runCSV(t, l, 10, export.Request{Stubs: true})
	if err != nil {
		t.Fatal(err)
	}
	// 1 column row + header + anchor + 2 entries + trailer.
	if n := bytes.Count(cb, []byte("\r\n")); n != 6 {
		t.Errorf("CRLF count = %d, want 6\n%q", n, cb)
	}
	if !bytes.Contains(cb, []byte("[\n{\"\"op\"\"")) {
		t.Errorf("diff LF not kept verbatim: %q", cb)
	}
	recs := readCSV(t, cb)
	for _, r := range recs {
		if r[0] == "entry" && strings.HasPrefix(r[len(r)-6], "[") {
			return
		}
	}
	t.Error("no diff cell found")
}

func TestCSVPageFailureLeavesNoTrailer(t *testing.T) {
	t.Parallel()
	l := newFakeLog(t, 6)
	l.failAt = 2
	cb, err := runCSV(t, l, 2, export.Request{Stubs: true})
	if err == nil {
		t.Fatal("want an error")
	}
	if bytes.Contains(cb, []byte("\r\ntrailer,")) {
		t.Errorf("trailer written after a failure: %q", cb)
	}
}
