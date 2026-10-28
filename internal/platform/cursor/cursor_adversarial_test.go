package cursor_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
)

// Adversarial cases for the cursor binding. The attacker holds a valid token
// for tenant A and a cursor minted for it, and controls every byte of the
// cursor string they send.

// A cursor minted for tenant A does not decode for tenant B even when every
// other claim matches, including a principal id that exists in both tenants.
func TestAdversarialCrossTenantReplaySamePrincipalID(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	a := base()
	tok, err := kr.Encode(a, []byte(`{"after":"42"}`))
	if err != nil {
		t.Fatal(err)
	}
	b := with(a, cursor.ClaimTenant, "tenant-b")
	if _, err := kr.Decode(b, tok); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("tenant-a cursor decoded for tenant-b: %v", err)
	}
}

// Shifting bytes between claim values (or between a value and the next name)
// does not keep the encoding. Covers separator-smuggling values such as a
// tenant value that embeds the principal claim.
func TestAdversarialClaimValueSmuggling(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	minted := cursor.Claims{"principal": "user-1", "tenant": "tenant-a"}
	tok, _ := kr.Encode(minted, []byte("p"))
	lp := func(s string) string {
		return string(binary.BigEndian.AppendUint64(nil, uint64(len(s)))) + s
	}
	for _, c := range []cursor.Claims{
		// Same tenant as the minted claims, so only the smuggled principal
		// differs. A tenant claim is present, or the tenant guard would
		// reject it before the encoding is compared.
		{"principal": "user-1" + lp("tenant") + lp("tenant-a"), "tenant": "tenant-a"},
		{"principal": "user-1", "tenant": "tenant-a\x00"},
		{"principal": "user-1tenant", "tenant": "-a"},
		{"Principal": "user-1", "tenant": "tenant-a"},
	} {
		if _, err := kr.Decode(c, tok); !errors.Is(err, cursor.ErrInvalid) {
			t.Fatalf("%q decoded a cursor minted for %q", c, minted)
		}
	}
}

// Swapping the key id is rejected for an unknown id and for every known id
// other than the signer: the key id is inside the MAC, and an unknown id is
// never accepted even though Decode does the MAC work with the active key.
func TestAdversarialKeyIDSwap(t *testing.T) {
	k1 := cursor.Key{ID: "k1", Secret: secret("a")}
	k2 := cursor.Key{ID: "k2", Secret: secret("b")}
	kr := ring(t, "k1", k1, k2)
	tok, _ := kr.Encode(base(), []byte("p"))
	rest := strings.SplitN(tok, ".", 2)[1]
	for _, kid := range []string{"k2", "k9", "K1", "k1 ", "k1\x00"} {
		if _, err := kr.Decode(base(), kid+"."+rest); !errors.Is(err, cursor.ErrInvalid) {
			t.Fatalf("key id %q accepted: %v", kid, err)
		}
	}
}

// An unknown key id must fail even if its MAC is exactly what the active key
// would compute for that id. This pins the `known` check in Decode: without
// it the fallback to the active secret would let an attacker who can mint
// cursors under one id present them under any id.
func TestAdversarialUnknownKeyIDWithActiveKeyMAC(t *testing.T) {
	s := secret("a")
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: s})
	claims := base()
	pos := []byte("p")
	mac := testSign(s, "k9", claims, pos)
	enc := base64.RawURLEncoding
	forged := "k9." + enc.EncodeToString(pos) + "." + enc.EncodeToString(mac)
	if _, err := kr.Decode(claims, forged); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("unknown key id accepted via active-key fallback: %v", err)
	}
}

// A key removed from the ring stops verifying, even when it is still the key
// id prefix of a structurally perfect token.
func TestAdversarialRetiredKeyCannotVerify(t *testing.T) {
	old := cursor.Key{ID: "k1", Secret: secret("a")}
	next := cursor.Key{ID: "k2", Secret: secret("b")}
	tok, _ := ring(t, "k1", old).Encode(base(), []byte("p"))
	retired := ring(t, "k2", next)
	if _, err := retired.Decode(base(), tok); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("retired key verified: %v", err)
	}
}

// Every failure path, including ones a probing client can reach, yields the
// very same error value: no wrapping that a handler could unwrap into detail.
func TestAdversarialEveryFailureIsTheSameValue(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	tok, _ := kr.Encode(base(), []byte("p"))
	parts := strings.Split(tok, ".")
	for name, in := range map[string]struct {
		c   cursor.Claims
		tok string
	}{
		"other tenant": {with(base(), cursor.ClaimTenant, "tenant-b"), tok},
		"unknown key":  {base(), "k9." + parts[1] + "." + parts[2]},
		"bad base64":   {base(), "k1.%%%." + parts[2]},
		"padded b64":   {base(), "k1." + parts[1] + "=." + parts[2]},
		"no claims":    {nil, tok},
		"too long":     {base(), tok + strings.Repeat("A", 5000)},
		"extra dot":    {base(), tok + "."},
	} {
		if _, err := kr.Decode(in.c, in.tok); err != cursor.ErrInvalid { //nolint:errorlint // identity is the point
			t.Fatalf("%s: got %#v, want the ErrInvalid value itself", name, err)
		}
	}
}

// testSign mirrors the package's MAC construction so a test can forge a
// token with a chosen key id. If the construction changes, update this too.
func testSign(secret []byte, keyID string, claims cursor.Claims, position []byte) []byte {
	lp := func(b []byte, s string) []byte {
		b = binary.BigEndian.AppendUint64(b, uint64(len(s)))
		return append(b, s...)
	}
	names := make([]string, 0, len(claims))
	for n := range claims {
		names = append(names, n)
	}
	sort.Strings(names)
	var canon []byte
	for _, n := range names {
		canon = lp(canon, n)
		canon = lp(canon, claims[n])
	}
	m := hmac.New(sha256.New, secret)
	m.Write([]byte("nava-cursor-v1"))
	m.Write(lp(nil, keyID))
	m.Write(lp(nil, string(canon)))
	m.Write(position)
	return m.Sum(nil)
}
