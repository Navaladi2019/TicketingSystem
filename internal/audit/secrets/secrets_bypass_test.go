package secrets

// Adversarial probe for PR #398 (TICK-490), Security review. Each case should
// be rejected with *ErrSecretInDiff in every domain; today all pass through.

import (
	"errors"
	"testing"
)

func TestCheckRejectsUserinfoBypasses(t *testing.T) {
	const c = "CANARY-9b1e-do-not-log"
	cases := map[string]string{
		// F1: split across tokens where a key holds an escaped "/" (~1).
		// Escaped join "/https:~1/u:...@h" has no "://"; unescaped it is "/https://u:...@h".
		"tilde1 op path":    `[{"op":"add","path":"/https:~1/u:` + c + `@h","value":1}]`,
		"tilde1 nested key": `[{"op":"add","path":"/a","value":{"https:/":{"u:` + c + `@h":1}}}]`,
		"slashes in key":    `[{"op":"add","path":"/a","value":{"https://":{"u:` + c + `@h":1}}}]`,
		// F2 (pre-existing since #392): WHATWG URL parsing treats these as userinfo for special schemes.
		"no slashes":    `[{"op":"add","path":"/a","value":"https:u:` + c + `@h"}]`,
		"single slash":  `[{"op":"add","path":"/a","value":"https:/u:` + c + `@h"}]`,
		"backslashes":   `[{"op":"add","path":"/a","value":"https:\\u:` + c + `@h"}]`,
		"pct slashes":   `[{"op":"add","path":"/a","value":"https:%2F%2Fu:` + c + `@h"}]`,
		"path no slash": `[{"op":"add","path":"/https:u:` + c + `@h","value":1}]`,
	}
	for name, d := range cases {
		for _, dom := range []Domain{DomainRecord, DomainConfig, DomainAudit} {
			var se *ErrSecretInDiff
			if err := Check([]byte(d), nil, dom); !errors.As(err, &se) {
				t.Errorf("%s/%s: Check = %v, want *ErrSecretInDiff", name, dom, err)
			}
		}
	}
}
