package chain_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
)

// escapeProne are the characters whose escaping encoding/json has changed
// between releases (U+2028, U+2029: TICK-970) or that HTML escaping touches
// (< > &). RFC 8785 JCS keeps every one raw.
const escapeProne = "\u2028\u2029<>&"

// TestVectorsGeneratorKeepsEnvelopesCanonical is the untagged guard for the
// vectors generator (QA on PR #736, TICK-964): the generator's own inputs
// carry every escape-prone character inside an envelope, and each chain
// envelope in Generate's output is its canonical bytes, only re-indented.
// Before this, only the integration-tagged TestSealGoldenVectors caught the
// Go 1.27 escape.
func TestVectorsGeneratorKeepsEnvelopesCanonical(t *testing.T) {
	t.Parallel()
	out, err := vectors.Generate()
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Chain []struct {
			Name     string          `json:"name"`
			Envelope json.RawMessage `json:"envelope"`
		} `json:"chain"`
	}
	if err := json.Unmarshal(out, &f); err != nil {
		t.Fatal(err)
	}
	cases := loadVectors(t).Chain
	if len(f.Chain) != len(cases) || len(cases) == 0 {
		t.Fatalf("generated %d chain cases, committed file has %d", len(f.Chain), len(cases))
	}
	covered := map[rune]bool{}
	for i, c := range cases {
		canon, err := chain.CanonicalEnvelope(c.Envelope)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		for _, r := range escapeProne {
			if bytes.ContainsRune(canon, r) {
				covered[r] = true
			}
		}
		if got := stripJSONWhitespace(f.Chain[i].Envelope); !bytes.Equal(got, canon) {
			t.Errorf("%s: generated envelope\n got %q\nwant %q", c.Name, got, canon)
		}
	}
	// The vector inputs carry U+2028 but no U+2029. Adding one would change
	// the committed vectors, a breaking change (PLAN-005 D-5-7), so U+2029
	// is fed through Generate's writer (MarshalIndentJSONL) by
	// TestEnvelopeWritersKeepEscapeProneTextRaw instead.
	for _, r := range escapeProne {
		if r == 0x2029 {
			continue
		}
		if !covered[r] {
			t.Errorf("no chain vector's canonical envelope holds %U raw; the generator no longer exercises it", r)
		}
	}
}

// TestEnvelopeWritersKeepEscapeProneTextRaw feeds the escape-prone
// characters through every in-package envelope writer and asserts the
// canonical bytes come out unchanged.
func TestEnvelopeWritersKeepEscapeProneTextRaw(t *testing.T) {
	t.Parallel()
	env := loadVectors(t).Chain[0].Envelope
	text := "a" + escapeProne + "z"
	env.ActorSystem = &text
	canon, err := chain.CanonicalEnvelope(env)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(canon, []byte(text)) {
		t.Fatalf("canonical envelope does not hold the text raw: %q", canon)
	}
	type line struct {
		Kind     string         `json:"kind"`
		Envelope chain.Envelope `json:"envelope"`
	}
	writers := map[string]func() ([]byte, error){
		"MarshalJSON":        func() ([]byte, error) { return env.MarshalJSON() },
		"MarshalJSONL":       func() ([]byte, error) { return chain.MarshalJSONL(line{"entry", env}) },
		"MarshalIndentJSONL": func() ([]byte, error) { return chain.MarshalIndentJSONL(line{"entry", env}, "", "  ") },
	}
	for name, write := range writers {
		b, err := write()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !bytes.Contains(stripJSONWhitespace(b), canon) {
			t.Errorf("%s does not carry the canonical bytes:\n%q\nwant within it %q", name, b, canon)
		}
		if strings.Contains(string(b), `\u2028`) || strings.Contains(string(b), `\u003c`) {
			t.Errorf("%s escaped the envelope text: %q", name, b)
		}
	}
}

// stripJSONWhitespace drops the whitespace outside strings, undoing only
// what an indenter adds.
func stripJSONWhitespace(b []byte) []byte {
	out := make([]byte, 0, len(b))
	inString, escaped := false, false
	for _, c := range b {
		switch {
		case inString:
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
		case c == '"':
			inString = true
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			continue
		}
		out = append(out, c)
	}
	return out
}

// TestEnvelopeWritersAreDeterministic: the same input gives the same bytes on
// every run (TICK-964). Go randomises map iteration, so a writer that leaked
// map order into its output would differ between runs.
func TestEnvelopeWritersAreDeterministic(t *testing.T) {
	t.Parallel()
	first, err := vectors.Generate()
	if err != nil {
		t.Fatal(err)
	}
	framing := map[string]any{}
	for i := range 64 {
		framing[string(rune('a'+i%26))+strings.Repeat("k", i/26)] = i
	}
	framing["envelope"] = loadVectors(t).Chain[0].Envelope
	line, err := chain.MarshalJSONL(framing)
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		again, err := vectors.Generate()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(again, first) {
			t.Fatal("vectors.Generate gave different bytes on a repeat run")
		}
		l, err := chain.MarshalJSONL(framing)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(l, line) {
			t.Fatalf("MarshalJSONL gave different bytes on a repeat run:\n%s\n%s", l, line)
		}
	}
}
