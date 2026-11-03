package secrets

// Adversarial probe for PR #400 (TICK-492), Security review. WHATWG URL
// parsing strips ASCII tab, LF and CR anywhere in the input and keeps space
// and form feed inside userinfo (percent-encoded). Node 22 `new URL(...)`
// reports a password for every case below. Each should be rejected in every
// domain; all pass through at 7c30b82.

import (
	"errors"
	"testing"
)

func TestCheckRejectsWhitespaceUserinfo(t *testing.T) {
	const c = "CANARY-9b1e-do-not-log"
	cases := map[string]string{
		"tab after scheme":    `[{"op":"add","path":"/a","value":"https:\t//u:` + c + `@h"}]`,
		"tab in scheme":       `[{"op":"add","path":"/a","value":"ht\ttps://u:` + c + `@h"}]`,
		"tab between slashes": `[{"op":"add","path":"/a","value":"https:/\t/u:` + c + `@h"}]`,
		"tab in password":     `[{"op":"add","path":"/a","value":"https://u:` + c + `\tx@h"}]`,
		"LF in password":      `[{"op":"add","path":"/a","value":"https://u:` + c + `\nx@h"}]`,
		"CR in password":      `[{"op":"add","path":"/a","value":"https://u:` + c + `\rx@h"}]`,
		"space in password":   `[{"op":"add","path":"/a","value":"https://u:` + c + ` x@h"}]`,
		"FF in password":      `[{"op":"add","path":"/a","value":"https://u:` + c + `\fx@h"}]`,
		"tab lax no slash":    `[{"op":"add","path":"/a","value":"https:u:` + c + `\tx@h"}]`,
		"tab in key":          `[{"op":"add","path":"/a","value":{"https:\t//u:` + c + `@h":1}}]`,
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
