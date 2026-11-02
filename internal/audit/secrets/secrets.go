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
//   - a URL with userinfo (scheme://user[:pass]@host) in any string value or
//     object key, in every domain;
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

// userinfoURL matches scheme://userinfo@ with no "/", "?", "#" or whitespace
// before the "@", so "https://host/@user" does not match.
var userinfoURL = regexp.MustCompile(`(?i)[a-z][a-z0-9+.\-]*://[^/?#@\s]*@`)

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

func scan(s string, shapes bool) (Reason, bool) {
	if userinfoURL.MatchString(s) {
		return ReasonURLUserinfo, true
	}
	if shapes && secretShape.MatchString(s) {
		return ReasonSecretShape, true
	}
	return "", false
}
