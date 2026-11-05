package secrets

// Adversarial probe for PR #404 (TICK-495), Security review. The WHATWG URL
// parser strips tab, LF and CR inside the scheme too. The scheme class
// [a-z][a-z0-9+.\-]* allows no whitespace, and the generic arm is unanchored,
// so it only matches when the text between the last stripped character and
// the ":" starts with a letter. "s\t3://..." is "s3://..." to WHATWG (Node 22:
// protocol "s3:", password set) but the "3" cannot start a scheme. Match-after-
// strip sees "s3://AKIA...:secret@bucket" and rejects it. At 491b84f every
// case passes Check in every domain.

import (
	"errors"
	"testing"
)

func TestCheckRejectsWhitespaceInSchemeTail(t *testing.T) {
	const c = "CANARY-9b1e-do-not-log"
	cases := map[string]string{
		"tab before digit in s3": `[{"op":"add","path":"/a","value":"s\t3://AKIAEXAMPLE:` + c + `@bucket/k"}]`,
		"LF before digit":        `[{"op":"add","path":"/a","value":"a\n1://u:` + c + `@h"}]`,
		"CR before dot-digit":    `[{"op":"add","path":"/a","value":"svn+\r.1://u:` + c + `@h"}]`,
		"tab in key":             `[{"op":"add","path":"/a","value":{"s\t3://u:` + c + `@h":1}}]`,
		"tab in path token":      `[{"op":"add","path":"/s\t3:~1~1u:` + c + `@h","value":1}]`,
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
