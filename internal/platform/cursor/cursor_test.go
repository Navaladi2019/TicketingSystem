package cursor_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
)

func secret(c string) []byte { return []byte(strings.Repeat(c, 32)) }

func ring(t *testing.T, active string, keys ...cursor.Key) *cursor.Keyring {
	t.Helper()
	kr, err := cursor.NewKeyring(active, keys...)
	if err != nil {
		t.Fatal(err)
	}
	return kr
}

func base() cursor.Claims {
	return cursor.Claims{
		cursor.ClaimTenant:    "tenant-a",
		cursor.ClaimPrincipal: "user-1",
		cursor.ClaimQuery:     cursor.QueryHash(url.Values{"status": {"open"}}),
	}
}

// with returns a copy of c with k set to v, or removed when v is empty.
func with(c cursor.Claims, k, v string) cursor.Claims {
	out := cursor.Claims{}
	for kk, vv := range c {
		out[kk] = vv
	}
	if v == "" {
		delete(out, k)
	} else {
		out[k] = v
	}
	return out
}

func TestRoundTrip(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	for _, pos := range [][]byte{[]byte(`{"after":"42"}`), {}, {0, 255, '.'}} {
		tok, err := kr.Encode(base(), pos)
		if err != nil {
			t.Fatal(err)
		}
		got, err := kr.Decode(base(), tok)
		if err != nil || string(got) != string(pos) {
			t.Fatalf("pos %q: got %q, %v", pos, got, err)
		}
	}
}

func TestBindingMismatchIsTheSameError(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	tok, _ := kr.Encode(base(), []byte("p"))
	cases := map[string]cursor.Claims{
		"cross-tenant":      with(base(), cursor.ClaimTenant, "tenant-b"),
		"other principal":   with(base(), cursor.ClaimPrincipal, "user-2"),
		"changed filter":    with(base(), cursor.ClaimQuery, cursor.QueryHash(url.Values{"status": {"closed"}})),
		"workflow added":    with(base(), cursor.ClaimWorkflow, "wf-1"),
		"claim dropped":     with(base(), cursor.ClaimPrincipal, ""),
		"no claims":         nil,
		"empty claim value": {cursor.ClaimTenant: ""},
	}
	var first string
	for name, c := range cases {
		_, err := kr.Decode(c, tok)
		if !errors.Is(err, cursor.ErrInvalid) {
			t.Fatalf("%s: got %v", name, err)
		}
		if first == "" {
			first = err.Error()
		} else if err.Error() != first {
			t.Fatalf("%s: error text differs: %q vs %q", name, err.Error(), first)
		}
	}
}

func TestWorkflowBinding(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	c := with(base(), cursor.ClaimWorkflow, "wf-1")
	tok, err := kr.Encode(c, []byte("p"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kr.Decode(c, tok); err != nil {
		t.Fatal(err)
	}
	for _, other := range []cursor.Claims{with(c, cursor.ClaimWorkflow, "wf-2"), base()} {
		if _, err := kr.Decode(other, tok); !errors.Is(err, cursor.ErrInvalid) {
			t.Fatalf("got %v", err)
		}
	}
}

func TestClaimEncodingHasNoCollisions(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	tok, _ := kr.Encode(cursor.Claims{"tenant": "t", "a": "bc", "d": "e"}, []byte("p"))
	for _, c := range []cursor.Claims{
		{"tenant": "t", "a": "b", "cd": "e"}, {"tenant": "t", "ab": "c", "d": "e"}, {"tenant": "t", "a": "bcde"},
	} {
		if _, err := kr.Decode(c, tok); !errors.Is(err, cursor.ErrInvalid) {
			t.Fatalf("%v decoded", c)
		}
	}
}

func TestTamperedTokenIsInvalid(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	tok, _ := kr.Encode(base(), []byte("position"))
	for i := 0; i < len(tok); i++ {
		b := []byte(tok)
		b[i] ^= 1
		if _, err := kr.Decode(base(), string(b)); !errors.Is(err, cursor.ErrInvalid) {
			t.Fatalf("byte %d flipped: got %v", i, err)
		}
	}
}

func TestMalformedTokens(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	tok, _ := kr.Encode(base(), []byte("p"))
	parts := strings.Split(tok, ".")
	for name, bad := range map[string]string{
		"empty":               "",
		"one part":            "k1",
		"two parts":           parts[0] + "." + parts[1],
		"four parts":          tok + ".x",
		"not base64":          "k1.!!!.!!!",
		"unknown key":         "k9." + parts[1] + "." + parts[2],
		"empty key id":        "." + parts[1] + "." + parts[2],
		"truncated mac":       tok[:len(tok)-2],
		"newline in position": parts[0] + "." + parts[1][:1] + "\n" + parts[1][1:] + "." + parts[2],
		"newline in mac":      parts[0] + "." + parts[1] + "." + parts[2][:1] + "\n" + parts[2][1:],
		"trailing crlf":       tok + "\r\n",
		"huge":                strings.Repeat("a", 5000),
	} {
		if _, err := kr.Decode(base(), bad); !errors.Is(err, cursor.ErrInvalid) {
			t.Fatalf("%s: got %v", name, err)
		}
	}
}

func TestKeyRotation(t *testing.T) {
	old := cursor.Key{ID: "k1", Secret: secret("a")}
	next := cursor.Key{ID: "k2", Secret: secret("b")}
	oldTok, _ := ring(t, "k1", old).Encode(base(), []byte("p"))

	rotated := ring(t, "k2", old, next)
	if got, err := rotated.Decode(base(), oldTok); err != nil || string(got) != "p" {
		t.Fatalf("old cursor after rotation: %q, %v", got, err)
	}
	newTok, _ := rotated.Encode(base(), []byte("q"))
	if !strings.HasPrefix(newTok, "k2.") {
		t.Fatalf("new cursor not signed by the active key: %q", newTok)
	}
	// Once the old key is retired its cursors stop decoding.
	if _, err := ring(t, "k2", next).Decode(base(), oldTok); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("retired key: %v", err)
	}
	// A key id swapped onto a token signed by another key does not verify.
	swapped := "k2." + strings.SplitN(oldTok, ".", 2)[1]
	if _, err := rotated.Decode(base(), swapped); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("swapped key id: %v", err)
	}
}

func TestDifferentSecretsDoNotInterchange(t *testing.T) {
	tok, _ := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")}).Encode(base(), []byte("p"))
	other := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("b")})
	if _, err := other.Decode(base(), tok); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestEncodeRefusals(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	if _, err := kr.Encode(nil, []byte("p")); err == nil {
		t.Fatal("encoded with no claims")
	}
	if _, err := kr.Encode(cursor.Claims{"tenant": ""}, []byte("p")); err == nil {
		t.Fatal("encoded an empty claim")
	}
	if _, err := kr.Encode(base(), make([]byte, 2000)); err == nil {
		t.Fatal("encoded an oversize position")
	}
}

func TestNewKeyringRefusals(t *testing.T) {
	good := cursor.Key{ID: "k1", Secret: secret("a")}
	for name, tc := range map[string]struct {
		active string
		keys   []cursor.Key
	}{
		"no keys":        {"k1", nil},
		"active missing": {"k2", []cursor.Key{good}},
		"short secret":   {"k1", []cursor.Key{{ID: "k1", Secret: []byte("short")}}},
		"dot in id":      {"a.b", []cursor.Key{{ID: "a.b", Secret: secret("a")}}},
		"empty id":       {"", []cursor.Key{{ID: "", Secret: secret("a")}}},
		"long id":        {"k", []cursor.Key{{ID: strings.Repeat("k", 33), Secret: secret("a")}}},
		"duplicate id":   {"k1", []cursor.Key{good, good}},
	} {
		if _, err := cursor.NewKeyring(tc.active, tc.keys...); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestQueryHash(t *testing.T) {
	a := cursor.QueryHash(url.Values{"status": {"open"}, "q": {"x"}})
	b := cursor.QueryHash(url.Values{"q": {"x"}, "status": {"open"}})
	if a != b {
		t.Fatal("hash depends on key order")
	}
	if a == cursor.QueryHash(url.Values{"status": {"open"}, "q": {"y"}}) {
		t.Fatal("hash ignores a changed filter")
	}
}

func TestTenantClaimIsRequired(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	noTenant := with(base(), cursor.ClaimTenant, "")
	if _, err := kr.Encode(noTenant, []byte("p")); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("Encode without tenant: got %v", err)
	}
	if _, err := kr.Encode(cursor.Claims{cursor.ClaimPrincipal: "u"}, []byte("p")); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("Encode with only a principal: got %v", err)
	}
	tok, _ := kr.Encode(base(), []byte("p"))
	if _, err := kr.Decode(noTenant, tok); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("Decode without tenant: got %v", err)
	}
	if _, err := kr.Decode(cursor.Claims{cursor.ClaimPrincipal: "u"}, tok); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("Decode with only a principal: got %v", err)
	}
}
