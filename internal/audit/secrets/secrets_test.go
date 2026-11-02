package secrets

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

const canary = "CANARY-9b1e-do-not-log"

func secretPaths() []patchwalk.Pointer { return []patchwalk.Pointer{"/auth/password", "/token"} }

// assertRejected checks Check rejects diff with the reason and path given,
// and that neither the error string nor a logged line carries the canary.
func assertRejected(t *testing.T, diff string, domain Domain, wantReason Reason, wantPath patchwalk.Pointer) {
	t.Helper()
	err := Check([]byte(diff), secretPaths(), domain)
	var se *ErrSecretInDiff
	if !errors.As(err, &se) {
		t.Fatalf("Check = %v, want *ErrSecretInDiff", err)
	}
	if se.Reason != wantReason || se.Path != wantPath {
		t.Errorf("rejected %s at %q, want %s at %q", se.Reason, se.Path, wantReason, wantPath)
	}
	if se.Code() != "audit_secret_in_diff" {
		t.Errorf("code = %q", se.Code())
	}
	msg := err.Error()
	if !strings.Contains(msg, string(wantPath)) || !strings.Contains(msg, Code) {
		t.Errorf("error %q does not name the code and path", msg)
	}

	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	log.LogAttrs(context.Background(), slog.LevelWarn, "audit write rejected", slog.Any("err", err))
	line := buf.String()
	if !strings.Contains(line, `"path":"`+string(wantPath)+`"`) {
		t.Errorf("log line does not name the path: %s", line)
	}
	for _, leak := range []string{canary, "ghp_", "AKIA", "u:p@"} {
		if strings.Contains(msg, leak) || strings.Contains(line, leak) {
			t.Errorf("secret %q leaked: error %q, log %s", leak, msg, line)
		}
	}
}

// CI-A5, walker half.
func TestCheckRejectsCanaryAtSecretPath(t *testing.T) {
	cases := map[string]struct {
		diff string
		path patchwalk.Pointer
	}{
		"top-level value": {
			`[{"op":"replace","path":"/token","value":"` + canary + `"}]`, "/token"},
		"nested add": {
			`[{"op":"add","path":"/auth","value":{"user":"u","password":"` + canary + `"}}]`, "/auth/password"},
		"parent replace": {
			`[{"op":"replace","path":"","value":{"name":"n","auth":{"password":"` + canary + `"}}}]`, "/auth/password"},
		"test op": {
			`[{"op":"test","path":"/auth/password","value":"` + canary + `"}]`, "/auth/password"},
		"test of parent": {
			`[{"op":"test","path":"/auth","value":{"password":"` + canary + `"}}]`, "/auth/password"},
		"value under a secret path": {
			`[{"op":"add","path":"/token/inner","value":"` + canary + `"}]`, "/token/inner"},
		"secret after a clean op": {
			`[{"op":"add","path":"/name","value":"ok"},{"op":"add","path":"/token","value":"` + canary + `"}]`, "/token"},
		"non-string secret": {
			`[{"op":"add","path":"/auth","value":{"password":{"v":"` + canary + `"}}}]`, "/auth/password"},
	}
	for name, c := range cases {
		for _, d := range []Domain{DomainRecord, DomainConfig, DomainAudit} {
			t.Run(name+"/"+string(d), func(t *testing.T) {
				assertRejected(t, c.diff, d, ReasonSecretPath, c.path)
			})
		}
	}
}

func TestCheckRejectsCopyAndMoveFromSecretPath(t *testing.T) {
	cases := map[string]struct {
		diff string
		path patchwalk.Pointer
	}{
		"copy from secret":       {`[{"op":"copy","from":"/token","path":"/public"}]`, "/token"},
		"move from secret":       {`[{"op":"move","from":"/auth/password","path":"/note"}]`, "/auth/password"},
		"copy from parent":       {`[{"op":"copy","from":"/auth","path":"/backup"}]`, "/auth"},
		"copy from root":         {`[{"op":"copy","from":"","path":"/snapshot"}]`, ""},
		"copy from under secret": {`[{"op":"copy","from":"/token/x","path":"/y"}]`, "/token/x"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assertRejected(t, c.diff, DomainRecord, ReasonSecretFrom, c.path)
		})
	}
}

func TestCheckRejectsURLUserinfoInEveryDomain(t *testing.T) {
	for _, d := range []Domain{DomainRecord, DomainConfig, DomainAudit} {
		t.Run(string(d), func(t *testing.T) {
			assertRejected(t, `[{"op":"add","path":"/hook","value":"https://u:p@hooks.example.org/x"}]`, d, ReasonURLUserinfo, "/hook")
			assertRejected(t, `[{"op":"add","path":"/n","value":{"d":"see postgres://u:p@db/x"}}]`, d, ReasonURLUserinfo, "/n/d")
			assertRejected(t, `[{"op":"add","path":"/m","value":{"https://u:p@h":1}}]`, d, ReasonURLUserinfo, "/m")
			assertRejected(t, `[{"op":"add","path":"/m/https:~1~1u:p@h","value":1}]`, d, ReasonURLUserinfo, "/m")
			assertRejected(t, `[{"op":"copy","from":"/https:~1~1u:p@h","path":"/z"}]`, d, ReasonURLUserinfo, "")
		})
	}
}

// TICK-490: userinfo split across pointer tokens or nested keys. Each token
// is clean on its own; only the joined pointer spells the URL.
func TestCheckRejectsURLUserinfoSplitAcrossTokens(t *testing.T) {
	userinfo := "u:pass-" + canary + "@h"
	cases := map[string]struct {
		diff string
		path patchwalk.Pointer
	}{
		"op path":    {`[{"op":"add","path":"/https://` + userinfo + `","value":1}]`, "/https:/"},
		"from":       {`[{"op":"copy","from":"/https://` + userinfo + `","path":"/z"}]`, "/https:/"},
		"nested key": {`[{"op":"add","path":"/m","value":{"https:":{"":{"` + userinfo + `":1}}}}]`, "/m/https:/"},
		"path and key": {
			`[{"op":"add","path":"/https:","value":{"":{"` + userinfo + `":1}}}]`, "/https:/"},
		"deep after scheme": {
			`[{"op":"add","path":"/a","value":{"see https:":{"":{"` + userinfo + `":{"b":1}}}}}]`, "/a/see https:/"},
	}
	for name, c := range cases {
		for _, d := range []Domain{DomainRecord, DomainConfig, DomainAudit} {
			t.Run(name+"/"+string(d), func(t *testing.T) {
				assertRejected(t, c.diff, d, ReasonURLUserinfo, c.path)
				assertNoLeak(t, Check([]byte(c.diff), secretPaths(), d), "pass")
			})
		}
	}
}

// assertNoLeak checks neither err's string nor its JSON slog line holds s.
func assertNoLeak(t *testing.T, err error, s string) {
	t.Helper()
	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).WarnContext(context.Background(), "audit write rejected", slog.Any("err", err))
	if strings.Contains(err.Error(), s) || strings.Contains(buf.String(), s) {
		t.Errorf("%q leaked: error %q, log %s", s, err.Error(), buf.String())
	}
}

func TestCheckSecretShapesOnConfigOnly(t *testing.T) {
	shapes := map[string]string{
		"aws":        "AKIA" + "ABCDEFGHIJKLMNOP",
		"github":     "ghp_" + strings.Repeat("a", 36),
		"github pat": "github_pat_" + strings.Repeat("b", 30),
		"slack":      "xoxb-" + "1234567890-abc",
		"pem":        "-----BEGIN RSA PRIVATE KEY-----\nMII...",
		"jwt":        "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.sig",
	}
	for name, s := range shapes {
		t.Run(name, func(t *testing.T) {
			diff := `[{"op":"add","path":"/adapter","value":{"note":` + jsonQuote(s) + `}}]`
			assertRejected(t, diff, DomainConfig, ReasonSecretShape, "/adapter/note")
			for _, d := range []Domain{DomainRecord, DomainAudit} {
				if err := Check([]byte(diff), secretPaths(), d); err != nil {
					t.Errorf("%s domain: shape scanner ran: %v", d, err)
				}
			}
		})
	}
}

func TestCheckAllowsCleanDiffs(t *testing.T) {
	clean := []string{
		`[]`,
		`[{"op":"remove","path":"/token"}]`,
		`[{"op":"remove","path":"/auth/password"}]`,
		`[{"op":"replace","path":"/auth/user","value":"alice"}]`,
		`[{"op":"copy","from":"/name","path":"/token"}]`,
		`[{"op":"move","from":"/authx","path":"/auth2"}]`,
		`[{"op":"add","path":"/link","value":"https://example.org/@alice?q=a@b#x@y"}]`,
		`[{"op":"add","path":"/email","value":"alice@example.org"}]`,
		`[{"op":"test","path":"/n","value":{"auth":{"pw":"x"}}}]`,
		`[{"op":"add","path":"/https:/example.org/@alice","value":1}]`,
		`[{"op":"add","path":"/m","value":{"https:":{"":{"example.org":{"@alice":1}}}}}]`,
	}
	for _, d := range clean {
		for _, dom := range []Domain{DomainRecord, DomainConfig, DomainAudit} {
			if err := Check([]byte(d), secretPaths(), dom); err != nil {
				t.Errorf("%s %s: %v", dom, d, err)
			}
		}
	}
}

func TestCheckInputErrors(t *testing.T) {
	ok := []byte(`[]`)
	if err := Check(ok, nil, "Config"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("unknown domain: %v", err)
	}
	if err := Check(ok, []patchwalk.Pointer{"token"}, DomainRecord); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("bad secret path: %v", err)
	}
	err := Check([]byte(`[{"op":"add","path":"/a","value":"`+canary+`","value":"x"}]`), secretPaths(), DomainRecord)
	if !errors.Is(err, patchwalk.ErrMalformed) || strings.Contains(err.Error(), canary) {
		t.Errorf("malformed: %v", err)
	}
	// A secret path covering the whole document rejects any value.
	err = Check([]byte(`[{"op":"add","path":"/a","value":1}]`), []patchwalk.Pointer{""}, DomainRecord)
	var se *ErrSecretInDiff
	if !errors.As(err, &se) {
		t.Errorf("root secret path: %v", err)
	}
}

// FuzzCheck: Check never panics, and a canary placed at a secret path is
// never let through nor echoed in the error.
func FuzzCheck(f *testing.F) {
	f.Add("auth", "password", true)
	f.Add("a~b", "c/d", false)
	f.Fuzz(func(t *testing.T, parent, key string, wrap bool) {
		if !utf8.ValidString(parent) || !utf8.ValidString(key) {
			t.Skip("a diff is valid UTF-8, so no diff can name this path")
		}
		secret := patchwalk.FromSegments([]string{parent, key})
		value := `"` + canary + `"`
		path := secret
		if wrap {
			value = `{` + jsonQuote(key) + `:"` + canary + `"}`
			path = patchwalk.FromSegments([]string{parent})
		}
		diff := `[{"op":"add","path":` + jsonQuote(string(path)) + `,"value":` + value + `}]`
		err := Check([]byte(diff), []patchwalk.Pointer{secret}, DomainRecord)
		if err == nil {
			t.Fatalf("secret at %q let through: %s", secret, diff)
		}
		if strings.Contains(err.Error(), canary) {
			t.Fatalf("canary in error: %v", err)
		}
	})
}

// FuzzCheckJoinedUserinfo (TICK-490): a URL with userinfo spelled across
// pointer tokens or nested keys, after any prefix, is rejected in every domain
// and its password is never echoed. The seeds are the TestCheckRejectsURL
// UserinfoSplitAcrossTokens cases.
func FuzzCheckJoinedUserinfo(f *testing.F) {
	f.Add("", 0)       // op path
	f.Add("", 1)       // from
	f.Add("m/", 2)     // nested key
	f.Add("", 3)       // path and key
	f.Add("a/see ", 2) // deep after scheme
	f.Fuzz(func(t *testing.T, prefix string, form int) {
		if !utf8.ValidString(prefix) || strings.Contains(prefix, canary) {
			t.Skip("not a diff, or the canary is in the prefix itself")
		}
		segs := strings.Split(prefix+"https://u:"+canary+"@h", "/")
		var diff string
		switch form & 3 {
		case 0:
			diff = `[{"op":"add","path":` + jsonQuote(string(patchwalk.FromSegments(segs))) + `,"value":1}]`
		case 1:
			diff = `[{"op":"copy","from":` + jsonQuote(string(patchwalk.FromSegments(segs))) + `,"path":"/z"}]`
		default:
			// The first k tokens form the op path, the rest nested keys.
			k := 0
			if form&3 == 3 {
				k = len(segs) - 2
			}
			value := "1"
			for i := len(segs) - 1; i >= k; i-- {
				value = `{` + jsonQuote(segs[i]) + `:` + value + `}`
			}
			diff = `[{"op":"add","path":` + jsonQuote(string(patchwalk.FromSegments(segs[:k]))) + `,"value":` + value + `}]`
		}
		for _, d := range []Domain{DomainRecord, DomainConfig, DomainAudit} {
			err := Check([]byte(diff), nil, d)
			var se *ErrSecretInDiff
			if !errors.As(err, &se) {
				t.Fatalf("%s: userinfo let through: %s: %v", d, diff, err)
			}
			if strings.Contains(err.Error(), canary) {
				t.Fatalf("%s: canary in error: %v", d, err)
			}
		}
	})
}

func jsonQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20:
			b.WriteString(`\u00`)
			b.WriteByte("0123456789abcdef"[r>>4])
			b.WriteByte("0123456789abcdef"[r&0xf])
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
