package chain_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
)

const vectorsPath = "testdata/vectors.json"

func loadVectors(t *testing.T) vectors.File {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(vectorsPath))
	if err != nil {
		t.Fatal(err)
	}
	var f vectors.File
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		t.Fatalf("decode %s: %v", vectorsPath, err)
	}
	if f.Format != vectors.Format {
		t.Fatalf("format = %q, want %q", f.Format, vectors.Format)
	}
	return f
}

// TestVectorsFileIsCurrent fails when the committed vectors drift from the
// generator: rerun go run ./internal/audit/chain/cmd/genvectors and review the
// diff (a changed vector is a breaking change, PLAN-005 D-5-7).
func TestVectorsFileIsCurrent(t *testing.T) {
	want, err := vectors.Generate()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.FromSlash(vectorsPath))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale or was edited by hand; run go run ./internal/audit/chain/cmd/genvectors from the module root", vectorsPath)
	}
}

// TestGoldenVectorsChain recomputes every chain vector from its envelope
// object: canonical bytes, envelope_sha256, entry_hash, the link to the entry
// before, and the diff hash.
func TestGoldenVectorsChain(t *testing.T) {
	f := loadVectors(t)
	if f.Genesis != chain.Hex(chain.Genesis[:]) {
		t.Fatalf("genesis = %s", f.Genesis)
	}
	if len(f.Chain) < 5 {
		t.Fatalf("only %d chain vectors", len(f.Chain))
	}
	prev := f.Genesis
	var sawTruncated, sawErased bool
	for i, c := range f.Chain {
		t.Run(c.Name, func(t *testing.T) {
			if c.Seq != int64(i+1) {
				t.Errorf("seq = %d, want %d", c.Seq, i+1)
			}
			if c.PrevHash != prev {
				t.Errorf("prev_hash = %s, want the previous entry_hash %s", c.PrevHash, prev)
			}
			canon, err := chain.CanonicalEnvelope(c.Envelope)
			if err != nil {
				t.Fatal(err)
			}
			if string(canon) != c.Canonical {
				t.Errorf("canonical bytes differ\n got: %s\nwant: %s", canon, c.Canonical)
			}
			if sum := sha256.Sum256([]byte(c.Canonical)); chain.Hex(sum[:]) != c.EnvelopeSHA256 {
				t.Errorf("sha256(canonical) = %x, vector says %s", sum, c.EnvelopeSHA256)
			}
			esha, err := chain.EnvelopeSHA256(c.Envelope)
			if err != nil {
				t.Fatal(err)
			}
			if chain.Hex(esha[:]) != c.EnvelopeSHA256 {
				t.Errorf("envelope_sha256 = %x, want %s", esha, c.EnvelopeSHA256)
			}
			p, err := chain.ParseHash32(c.PrevHash)
			if err != nil {
				t.Fatal(err)
			}
			if eh := chain.EntryHash(p, esha); chain.Hex(eh[:]) != c.EntryHash {
				t.Errorf("entry_hash = %x, want %s", eh, c.EntryHash)
			}
			if c.Diff != nil {
				salt, err := chain.ParseSalt16(*c.DiffSalt)
				if err != nil {
					t.Fatal(err)
				}
				d := chain.DiffSHA256(salt, []byte(*c.Diff))
				if c.Envelope.DiffSHA256 == nil || d != *c.Envelope.DiffSHA256 {
					t.Errorf("diff_sha256 does not cover salt ‖ stored diff bytes")
				}
			}
			if c.Envelope.DiffTruncated && c.Diff != nil {
				sawTruncated = true
			}
			if c.ErasedAt != nil {
				sawErased = true
				if c.Diff != nil || c.DiffSalt != nil || c.Envelope.DiffSHA256 == nil {
					t.Errorf("an erased vector has no diff or salt but keeps diff_sha256")
				}
			}
		})
		prev = c.EntryHash
	}
	if !sawTruncated {
		t.Error("no truncated-honest vector (TICK-449)")
	}
	if !sawErased {
		t.Error("no erased-entry vector")
	}
}

// TestGoldenVectorsTampered checks that every tampered line fails, and fails
// for the reason the vector names.
func TestGoldenVectorsTampered(t *testing.T) {
	f := loadVectors(t)
	var sawFlag bool
	for _, c := range f.Tampered {
		t.Run(c.Name, func(t *testing.T) {
			got := verifyLine(t, c)
			if got != c.Expect {
				t.Fatalf("verification result %q, want %q", got, c.Expect)
			}
			if c.Expect == vectors.ExpectEnvelopeMismatch && c.BasedOn == "truncated-honest" {
				sawFlag = true
			}
		})
	}
	if !sawFlag {
		t.Error("no tampered diff_truncated vector (TICK-449)")
	}
}

// verifyLine is the minimal per-line check the verifier (P5-03) builds on.
func verifyLine(t *testing.T, c vectors.Tampered) string {
	t.Helper()
	esha, err := chain.EnvelopeSHA256(c.Envelope)
	if err != nil {
		t.Fatal(err)
	}
	if chain.Hex(esha[:]) != c.EnvelopeSHA256 {
		return vectors.ExpectEnvelopeMismatch
	}
	if c.Diff != nil {
		salt, err := chain.ParseSalt16(*c.DiffSalt)
		if err != nil {
			t.Fatal(err)
		}
		if c.Envelope.DiffSHA256 == nil || chain.DiffSHA256(salt, []byte(*c.Diff)) != *c.Envelope.DiffSHA256 {
			return vectors.ExpectDiffMismatch
		}
	}
	prev, err := chain.ParseHash32(c.PrevHash)
	if err != nil {
		t.Fatal(err)
	}
	if eh := chain.EntryHash(prev, esha); chain.Hex(eh[:]) != c.EntryHash {
		return vectors.ExpectEntryHashMismatch
	}
	return "ok"
}

func TestGoldenVectorsDiffs(t *testing.T) {
	f := loadVectors(t)
	seen := map[string]string{}
	for _, c := range f.Diffs {
		salt, err := chain.ParseSalt16(c.DiffSalt)
		if err != nil {
			t.Fatal(err)
		}
		got := chain.DiffSHA256(salt, []byte(c.Diff))
		if chain.Hex(got[:]) != c.DiffSHA256 {
			t.Errorf("%s: diff_sha256 = %x, want %s", c.Name, got, c.DiffSHA256)
		}
		if other, dup := seen[c.DiffSHA256]; dup {
			t.Errorf("%s and %s hash alike", c.Name, other)
		}
		seen[c.DiffSHA256] = c.Name
	}
}

func TestGoldenVectorsInvalid(t *testing.T) {
	f := loadVectors(t)
	if len(f.Invalid) == 0 {
		t.Fatal("no invalid vectors")
	}
	for _, c := range f.Invalid {
		in, err := c.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		var e chain.Envelope
		if err := e.UnmarshalJSON(in); !errors.Is(err, chain.ErrInvalidEnvelope) {
			t.Errorf("%s: err = %v, want ErrInvalidEnvelope", c.Name, err)
		}
	}
}

// TestGoldenVectorPinned pins one vector's hashes as literals, so the hashing
// code and the generator cannot drift together unnoticed. The values were
// cross-checked with an independent implementation (Python json.dumps with
// sort_keys, compact separators and ensure_ascii=False, which equals JCS for
// strings, integers, booleans and null).
func TestGoldenVectorPinned(t *testing.T) {
	f := loadVectors(t)
	pinned := []struct{ name, envelopeSHA, entryHash string }{
		{"config-create-minimal",
			"75b2549dd7927ea6cce470bc916705586507745318b1acb72a9113f369f92991",
			"b0663332aec5e8640abcdbb12a5e9b18d4777128f8ad716f00bc8d313a1dad72"},
		// The escape rules are the riskiest part of the canonical form.
		{"system-actor-string-escapes",
			"a7de43f16880270cd35ae9c15fa609f704a2cff7f3148d5e36fd3dcc21e225bf",
			"9dd1bed60c016c1aa704a6bc9c583cccd2a3be05f59da0558ea9f5475cc3625c"},
	}
	for _, p := range pinned {
		var c *vectors.ChainCase
		for i := range f.Chain {
			if f.Chain[i].Name == p.name {
				c = &f.Chain[i]
			}
		}
		if c == nil {
			t.Fatalf("no vector %s", p.name)
		}
		esha, err := chain.EnvelopeSHA256(c.Envelope)
		if err != nil {
			t.Fatal(err)
		}
		if chain.Hex(esha[:]) != p.envelopeSHA {
			t.Errorf("%s: envelope_sha256 = %x, pinned %s", p.name, esha, p.envelopeSHA)
		}
		prev, err := chain.ParseHash32(c.PrevHash)
		if err != nil {
			t.Fatal(err)
		}
		if eh := chain.EntryHash(prev, esha); chain.Hex(eh[:]) != p.entryHash {
			t.Errorf("%s: entry_hash = %x, pinned %s", p.name, eh, p.entryHash)
		}
	}
}
