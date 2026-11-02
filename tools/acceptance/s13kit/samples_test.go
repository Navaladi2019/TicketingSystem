package s13kit_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
	vt "github.com/Navaladi2019/TicketingSystem/internal/audit/verify/verifytest"
)

//nolint:gochecknoglobals // the standard golden-file -update flag
var update = flag.Bool("update", false, "rewrite the docs/acceptance/S13 sample files")

// kitDir is the acceptance kit, relative to this package.
const kitDir = "../../../docs/acceptance/S13"

// sample is one file in the kit and how it is built.
type sample struct {
	file  string
	build func() ([]byte, error)
}

// samples lists the kit in README order. Every file is either a whole export
// from the P5-02 generator's "exports" vectors, or built by verifytest from
// the generator's chain and tampered vectors. A tampering is applied to the
// clean file's bytes or lines, never by writing a hash.
func samples() []sample {
	return []sample{
		// Step 1.
		{"export-ok.jsonl", exportVector("unfiltered")},
		// Step 2.
		{"export-byte-edited.jsonl", byteEdited},
		{"export-line-deleted.jsonl", lineDeleted},
		{"export-first-entry-deleted.jsonl", firstEntryDeleted},
		{"export-first-entry-deleted-reanchored.jsonl", firstEntryDeletedReanchored},
		// Step 3.
		{"export-no-anchor.jsonl", noAnchor},
		// Step 4.
		{"export-filtered.jsonl", exportVector("filtered-subject-type")},
		// Step 5.
		{"export-redacted.jsonl", exportVector("diff-redacted-replaced")},
		{"export-erased-without-erase-entry.jsonl", exportVector("erasure-not-listed")},
		// Beyond the story's five steps: the other outcomes the human meets.
		{"export-erased-pending.jsonl", erasedPending},
		{"export-truncated-honest.jsonl", truncatedHonest},
		{"export-truncated-flag-cleared.jsonl", truncatedFlagCleared},
		{"export-stubs-omitted.jsonl", exportVector("stubs-false-honest")},
		{"export-empty.jsonl", exportVector("empty-range")},
	}
}

// exportVector is a whole export file exactly as the generator emits it.
func exportVector(name string) func() ([]byte, error) {
	return func() ([]byte, error) {
		for _, e := range vt.Vectors().Exports {
			if e.Name == name {
				return vt.JSONL(e.JSONL), nil
			}
		}
		return nil, fmt.Errorf("no export vector %q", name)
	}
}

func chainCase(name string) (vectors.ChainCase, error) {
	for _, c := range vt.Chain() {
		if c.Name == name {
			return c, nil
		}
	}
	return vectors.ChainCase{}, fmt.Errorf("no chain vector %q", name)
}

// byteEdited is export-ok.jsonl with one byte of seq 2's request_id changed.
func byteEdited() ([]byte, error) {
	ok, err := exportVector("unfiltered")()
	if err != nil {
		return nil, err
	}
	from, to := []byte(`"request_id":"req-01J9ZZ"`), []byte(`"request_id":"req-01J9ZY"`)
	if n := bytes.Count(ok, from); n != 1 {
		return nil, fmt.Errorf("byte edit: %d matches for %s, want 1", n, from)
	}
	return bytes.Replace(ok, from, to, 1), nil
}

// lines splits a JSONL file into its lines, without line ends.
func lines(b []byte) []string {
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func join(ls []string) []byte { return []byte(strings.Join(ls, "\n") + "\n") }

// lineDeleted is export-ok.jsonl with the seq 3 line removed. Nothing else
// changes, so the trailer still counts it.
func lineDeleted() ([]byte, error) {
	ok, err := exportVector("unfiltered")()
	if err != nil {
		return nil, err
	}
	ls := lines(ok)
	// header, anchor, seq 1, seq 2, seq 3: seq 3 is line 5, index 4.
	if !strings.Contains(ls[4], `"seq":3,`) {
		return nil, fmt.Errorf("line 5 is not seq 3: %.80s", ls[4])
	}
	return join(slices.Delete(ls, 4, 5)), nil
}

// firstEntryDeleted drops seq 1, the first entry after the anchor, and
// recounts the trailer as a careful forger would.
func firstEntryDeleted() ([]byte, error) {
	f := vt.Build(vt.Options{})
	f.Body = slices.Delete(f.Body, 1, 2)
	f.Recount()
	return f.Bytes(), nil
}

// firstEntryDeletedReanchored drops seq 1 and turns it into the anchor, so
// every remaining link holds. Only the header filter (no from, no after_seq:
// the range starts at genesis) gives it away.
func firstEntryDeletedReanchored() ([]byte, error) {
	f := vt.Build(vt.Options{})
	first := f.Body[1]
	f.Body[1] = vt.Line{
		"kind": "anchor", "genesis": false, "seq": first["seq"], "prev_hash": first["prev_hash"],
		"entry_hash": first["entry_hash"], "envelope_sha256": first["envelope_sha256"],
	}
	f.Body = f.Body[1:]
	f.Trailer["anchor_seq"] = first["seq"]
	f.Recount()
	return f.Bytes(), nil
}

// noAnchor is export-ok.jsonl without its anchor line: the first chain line
// is seq 1.
func noAnchor() ([]byte, error) {
	ok, err := exportVector("unfiltered")()
	if err != nil {
		return nil, err
	}
	ls := lines(ok)
	if !strings.Contains(ls[1], `"kind":"anchor"`) {
		return nil, fmt.Errorf("line 2 is not the anchor: %.80s", ls[1])
	}
	return join(slices.Delete(ls, 1, 2)), nil
}

// erasedPending ends the range at the erased entry; its audit/erase entry
// was written but not sealed yet, so it is an unsealed line (R24).
func erasedPending() ([]byte, error) {
	erased, err := chainCase("erased-entry")
	if err != nil {
		return nil, err
	}
	erase, err := chainCase("audit-erase")
	if err != nil {
		return nil, err
	}
	return vt.Build(vt.Options{HeadSeq: erased.Seq, Unsealed: []vectors.ChainCase{erase}}).Bytes(), nil
}

// truncatedOnly exports the truncated-honest entry in full and the rest as
// stubs.
func truncatedOnly() (*vt.File, int64, error) {
	c, err := chainCase("truncated-honest")
	if err != nil {
		return nil, 0, err
	}
	if !c.Envelope.DiffTruncated {
		return nil, 0, fmt.Errorf("truncated-honest vector has diff_truncated false")
	}
	return vt.Build(vt.Options{Match: func(x vectors.ChainCase) bool { return x.Seq == c.Seq }}), c.Seq, nil
}

// truncatedHonest: a diff cut at 64 KB with diff_truncated true (ADR-004-E1)
// verifies.
func truncatedHonest() ([]byte, error) {
	f, _, err := truncatedOnly()
	if err != nil {
		return nil, err
	}
	return f.Bytes(), nil
}

// truncatedFlagCleared is the same file with the generator's
// diff-truncated-flag-cleared tampered vector in place of the honest line:
// the truncation is hidden, so the envelope hash breaks.
func truncatedFlagCleared() ([]byte, error) {
	f, seq, err := truncatedOnly()
	if err != nil {
		return nil, err
	}
	for _, tv := range vt.Vectors().Tampered {
		if tv.Name == "diff-truncated-flag-cleared" {
			if tv.Seq != seq {
				return nil, fmt.Errorf("tampered vector at seq %d, want %d", tv.Seq, seq)
			}
			f.Body[f.Index(seq)] = vt.Tampered(tv)
			return f.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("no tampered vector diff-truncated-flag-cleared")
}

func TestSamplesAreCurrent(t *testing.T) {
	want := map[string]bool{}
	for _, s := range samples() {
		want[s.file] = true
		b, err := s.build()
		if err != nil {
			t.Fatalf("%s: %v", s.file, err)
		}
		p := filepath.Join(kitDir, s.file)
		if *update {
			if err := os.WriteFile(p, b, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("%s: %v (rebuild with -update)", s.file, err)
		}
		if !bytes.Equal(got, b) {
			t.Errorf("%s differs from what the P5-02 generator builds; never edit a sample by hand, rebuild with -update", s.file)
		}
	}
	// No stray sample: every .jsonl in the kit comes from the generator.
	files, err := filepath.Glob(filepath.Join(kitDir, "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if !want[filepath.Base(f)] {
			t.Errorf("%s is in the kit but not built by samples()", filepath.Base(f))
		}
	}
}
