package chain_test

// Adversarial tamper tests for the hash-chain contract (Security review of
// PR #365, TICK-424). Each test plays an attacker who can edit an export file
// or a stored row after sealing and checks that the edit does not verify.

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
)

// otherTenant is a tenant that does not appear in the golden vectors.
const otherTenant = "0190f3a0-0000-7000-8000-0000000000b2"

// TestAdversarialTenantRelabelBreaksChain: a sealed entry of one tenant
// re-labelled with another tenant_id, even with its link fields kept, must
// not reproduce the committed envelope_sha256 or entry_hash. Without this,
// one tenant's sealed entry could be passed off inside another tenant's chain.
func TestAdversarialTenantRelabelBreaksChain(t *testing.T) {
	f := loadVectors(t)
	for _, c := range f.Chain {
		t.Run(c.Name, func(t *testing.T) {
			e := c.Envelope
			e.TenantID = uuid.MustParse(otherTenant)
			env, err := chain.EnvelopeSHA256(e)
			if err != nil {
				t.Fatal(err)
			}
			if chain.Hex(env[:]) == c.EnvelopeSHA256 {
				t.Fatal("tenant_id is not covered by envelope_sha256")
			}
			prev, err := chain.ParseHash32(c.PrevHash)
			if err != nil {
				t.Fatal(err)
			}
			link := chain.EntryHash(prev, env)
			if chain.Hex(link[:]) == c.EntryHash {
				t.Fatal("a re-labelled entry reproduces the sealed entry_hash")
			}
		})
	}
}

// TestAdversarialEscapedKeyCannotSmuggleDuplicate: a member name written with
// a \u escape decodes to the same name, so it must count as a duplicate. If
// it did not, "source" could appear twice and readers would disagree on which
// value was hashed.
func TestAdversarialEscapedKeyCannotSmuggleDuplicate(t *testing.T) {
	f := loadVectors(t)
	canon := f.Chain[0].Canonical
	in := strings.Replace(canon, `{`, `{"source":"forged",`, 1)
	var e chain.Envelope
	err := json.Unmarshal([]byte(in), &e)
	if !errors.Is(err, chain.ErrInvalidEnvelope) {
		t.Fatalf("escaped duplicate member accepted (err = %v, source = %q)", err, e.Source)
	}
}

// TestAdversarialParserRejectsLossyStrings: the parser must reject string
// bytes it cannot represent exactly (spec section 2: "Invalid UTF-8 is an
// error"). encoding/json silently turns invalid UTF-8 and lone surrogate
// escapes into U+FFFD. Several different files would then verify as the same
// entry, and a verifier in another language could reach a different verdict
// on the same file.
func TestAdversarialParserRejectsLossyStrings(t *testing.T) {
	f := loadVectors(t)
	canon := f.Chain[0].Canonical
	const orig = `"source":"user"`
	if !strings.Contains(canon, orig) {
		t.Fatalf("vector %s no longer contains %s", f.Chain[0].Name, orig)
	}
	cases := map[string]string{
		"invalid-utf8-byte":     "\"source\":\"us\xffer\"",
		"lone-high-surrogate":   `"source":"us\ud800er"`,
		"lone-low-surrogate":    `"source":"us\udc00er"`,
		"reversed-surrogates":   `"source":"us\udc00\ud800er"`,
		"overlong-utf8-slash":   "\"source\":\"us\xc0\xafer\"",
		"invalid-utf8-in-a-key": "\"sou\xffrce\":\"user\"",
	}
	for name, repl := range cases {
		t.Run(name, func(t *testing.T) {
			in := strings.Replace(canon, orig, repl, 1)
			var e chain.Envelope
			if err := json.Unmarshal([]byte(in), &e); !errors.Is(err, chain.ErrInvalidEnvelope) {
				t.Fatalf("lossy string accepted (err = %v, source = %q)", err, e.Source)
			}
		})
	}
}
