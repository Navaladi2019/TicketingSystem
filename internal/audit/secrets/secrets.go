// Package secrets is the write-time secret check of the audit log (ADR-004
// R6, PLAN-005 P5-12a). audit.Append and every *_versions writer call Check on
// a diff before it is stored.
//
// Check walks the diff with patchwalk and rejects, with *ErrSecretInDiff:
//
//   - any value at or below a secret path (a schema's x-nava-secret or
//     writeOnly pointer), however it gets there: a top-level value, a nested
//     member of an add, a replace or a test of a parent object;
//   - a copy or move whose from overlaps a secret path (equal to it, under it,
//     or a parent of it), because it relocates the secret to a path that is not
//     marked secret;
//   - a URL with userinfo (scheme://user[:pass]@host, or the WHATWG-lax
//     spellings of the special schemes such as https:user@host or
//     https:\user@host) in any string value or object key, or spelled across
//     the tokens of a pointer (an op path, a from, or the nested keys of a
//     value), in every domain, ignoring tab, LF and CR and allowing space and
//     form feed in the userinfo, as WHATWG URL parsing does;
//   - on DomainConfig only, a secret-shaped string (AWS access key id, GitHub
//     token, Slack token, PEM block, JWT) in any string value or object key.
//
// The error, its message and its slog value name the pointer, the op index and
// a fixed reason, never the value. When the offending text is an object key,
// the pointer named is the key's parent, because the key itself is the secret.
//
// Secret paths match token by token; there is no wildcard, so a schema with a
// secret under an array item must list each concrete index the caller wants
// covered.
package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

// Code is the error code a rejected audit_log write maps to.
const Code = "audit_secret_in_diff"

// Domain is the audit_log domain (ADR-004 R1) of the diff being checked.
type Domain string

// The three audit_log domains.
const (
	DomainRecord Domain = "record"
	DomainConfig Domain = "config"
	DomainAudit  Domain = "audit"
)

// Reason says which rule rejected the diff. It is a fixed vocabulary and safe
// to log.
type Reason string

// Rejection reasons.
const (
	ReasonSecretPath  Reason = "secret_path"
	ReasonSecretFrom  Reason = "copy_from_secret_path"
	ReasonURLUserinfo Reason = "url_userinfo"
	ReasonSecretShape Reason = "secret_shape"
)

// ErrSecretInDiff is the rejection. It never holds the secret.
type ErrSecretInDiff struct {
	// Path is the pointer the secret was found at (for a secret object key,
	// the key's parent).
	Path patchwalk.Pointer
	// Op is the index of the op in the patch.
	Op int
	// Reason is the rule that matched.
	Reason Reason
}

func (e *ErrSecretInDiff) Error() string {
	return fmt.Sprintf("%s: %s at %q (op %d)", Code, e.Reason, string(e.Path), e.Op)
}

// Code returns the API error code, audit_secret_in_diff.
func (e *ErrSecretInDiff) Code() string { return Code }

// LogValue makes a logged error carry the code, path, op and reason only.
func (e *ErrSecretInDiff) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("code", Code),
		slog.String("path", string(e.Path)),
		slog.Int("op", e.Op),
		slog.String("reason", string(e.Reason)),
	)
}

// ErrInvalidInput is wrapped when Check is called with an unknown domain or an
// invalid secret path: a caller bug, not a client error.
var ErrInvalidInput = errors.New("secrets: invalid input")

// userinfoURL matches a URL with userinfo: a scheme, then the text before the
// host, then userinfo with no "/", "?" or "#" before the "@", so
// "https://host/@user" does not match.
//
// For any scheme the text before the host is "://". The special schemes of the
// WHATWG URL standard (http, https, ws, wss, ftp, file) are parsed laxly by
// browsers and Node: after the ":" any run of "/" or "\", none included, comes
// before the authority, so "https:u:p@h", "https:/u:p@h", "https:\u:p@h" and
// "https:///u:p@h" all carry userinfo. Percent-encoded slashes ("%2F", "%5C")
// count too. For them the scheme must not continue a longer scheme name
// ("xhttps:" is not special), and the lax form applies to them only, so
// "mailto:a@b" and "urn:x:a@b" still pass.
//
// Tab, LF and CR may sit anywhere in either form, because the WHATWG URL
// parser strips them anywhere in a URL: "ht\ttps:/\t/u:p@h" is
// "https://u:p@h" and "s\t3://u:p@h" is "s3://u:p@h" (TICK-495). Inside a
// generic scheme name they matter when the run after one starts with a digit,
// "+", "." or "-", which cannot start a scheme on its own. Space and form feed
// may sit in the userinfo, which WHATWG keeps percent-encoded:
// "https://u:p x@h" carries userinfo too.
// The pattern runs on the text as given rather than on the text with tab, LF
// and CR removed, so a tab, LF or CR before a special scheme still ends the
// word before it: "note\nhttps:u:p@h" is caught, though removing the LF would
// make the scheme "notehttps:".
var userinfoURL = regexp.MustCompile(`(?i)[a-z](?:` + urlWS + `[a-z0-9+.\-])*` + urlWS + `:` + urlWS + `/` + urlWS + `/[^/?#@]*@` +
	`|(?:^|[^a-z0-9+.\-])(?:` + spaced("https") + `|` + spaced("http") + `|` + spaced("wss") + `|` + spaced("ws") +
	`|` + spaced("ftp") + `|` + spaced("file") + `)` + urlWS +
	`:(?:[/\\\t\r\n]|` + spaced("%2f") + `|` + spaced("%5c") + `)*[^/?#@]*@`)

// urlWS matches what the WHATWG URL parser strips anywhere in a URL: ASCII
// tab, LF and CR.
const urlWS = `[\t\r\n]*`

// spaced returns a pattern for lit with urlWS allowed between its characters.
// lit must hold no regexp metacharacters.
func spaced(lit string) string {
	return strings.Join(strings.Split(lit, ""), urlWS)
}

// secretShape matches the secret shapes ADR-004 R6 names (RFC-008 D7).
var secretShape = regexp.MustCompile(
	`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b` + // AWS access key id
		`|\bgh[pousr]_[A-Za-z0-9]{36,}` + // GitHub token
		`|\bgithub_pat_[A-Za-z0-9_]{22,}` + // GitHub fine-grained token
		`|\bxox[abposr]-[A-Za-z0-9-]{10,}` + // Slack token
		`|-----BEGIN [A-Z0-9 ]*-----` + // PEM block
		`|\beyJ[A-Za-z0-9_-]{2,}\.[A-Za-z0-9_-]{2,}\.[A-Za-z0-9_-]*`, // JWT
)

// Check rejects diff when it carries a secret (see the package doc). It
// returns *ErrSecretInDiff for a secret, an error wrapping
// patchwalk.ErrMalformed for a diff it cannot walk completely, and one
// wrapping ErrInvalidInput for a bad domain or secret path.
func Check(diff []byte, secretPaths []patchwalk.Pointer, domain Domain) error {
	switch domain {
	case DomainRecord, DomainConfig, DomainAudit:
	default:
		return fmt.Errorf("%w: unknown domain", ErrInvalidInput)
	}
	for i, p := range secretPaths {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("%w: secret path %d: %w", ErrInvalidInput, i, err)
		}
	}
	shapes := domain == DomainConfig
	return patchwalk.Walk(diff, func(op patchwalk.Op, ptr patchwalk.Pointer, v json.RawMessage) error {
		reject := func(at patchwalk.Pointer, r Reason) error {
			return &ErrSecretInDiff{Path: at, Op: op.Index, Reason: r}
		}
		switch op.Field {
		case patchwalk.FieldPath, patchwalk.FieldFrom:
			// Every key the pointer names is client text: scan each one.
			segs := ptr.Segments()
			for i, s := range segs {
				if r, bad := scan(s, shapes); bad {
					return reject(patchwalk.FromSegments(segs[:i]), r)
				}
			}
			if at, r, bad := scanJoined(ptr, shapes); bad {
				return reject(at, r)
			}
			if op.Field == patchwalk.FieldFrom {
				for _, s := range secretPaths {
					if ptr.HasPrefix(s) || s.HasPrefix(ptr) {
						return reject(ptr, ReasonSecretFrom)
					}
				}
			}
		case patchwalk.FieldValue:
			for _, s := range secretPaths {
				if ptr.HasPrefix(s) {
					return reject(ptr, ReasonSecretPath)
				}
			}
			// A member key is the last token of its pointer; the op path's
			// own tokens were scanned on the FieldPath visit.
			if segs := ptr.Segments(); len(segs) > 0 {
				if r, bad := scan(segs[len(segs)-1], shapes); bad {
					return reject(patchwalk.FromSegments(segs[:len(segs)-1]), r)
				}
			}
			// Nested keys can spell a URL between them: {"https:":{"":{"u:p@h":1}}}.
			if at, r, bad := scanJoined(ptr, shapes); bad {
				return reject(at, r)
			}
			if len(v) > 0 && v[0] == '"' {
				var s string
				if err := json.Unmarshal(v, &s); err != nil {
					return fmt.Errorf("%w: op %d: undecodable string", patchwalk.ErrMalformed, op.Index)
				}
				if r, bad := scan(s, shapes); bad {
					return reject(ptr, r)
				}
			}
		}
		return nil
	})
}

// scanJoined scans the pointer text itself, so a URL split across tokens
// ("/https://u:p@h" is "https:", "", "u:p@h") is caught. It scans both the
// escaped pointer and the unescaped join of its tokens, so a key holding a
// "/" ("https:/" then "u:p@h", escaped "/https:~1/u:p@h") is caught too.
//
// On a match it names the parent of the token where the match ends. That
// parent never holds the userinfo: the pattern allows no "/" in the userinfo
// (between the slashes after the scheme and the "@"), and in either text the "/" nearest before the "@" is at or after the
// start of that token. The shapes hold no "/" either, so the same is true of
// them.
func scanJoined(ptr patchwalk.Pointer, shapes bool) (patchwalk.Pointer, Reason, bool) {
	segs := ptr.Segments()
	both := func(segs []string) (Reason, bool) {
		if r, bad := scan(string(patchwalk.FromSegments(segs)), shapes); bad {
			return r, true
		}
		if len(segs) == 0 {
			return "", false
		}
		return scan("/"+strings.Join(segs, "/"), shapes)
	}
	if _, bad := both(segs); !bad {
		return "", "", false
	}
	for i := range segs {
		if r, bad := both(segs[:i+1]); bad {
			return patchwalk.FromSegments(segs[:i]), r, true
		}
	}
	// Unreachable: the full prefix is ptr itself. Fail closed at the root.
	r, _ := both(segs)
	return "", r, true
}

// scan is every match Check makes: on a value, a key, a pointer token and both
// joined pointer texts.
func scan(s string, shapes bool) (Reason, bool) {
	if userinfoURL.MatchString(s) {
		return ReasonURLUserinfo, true
	}
	if shapes && secretShape.MatchString(s) {
		return ReasonSecretShape, true
	}
	return "", false
}
