// Package numbering is the compiled form of a record number format (ADR-005
// §3.1): what a create reads to know which counter it takes (the scope key)
// and how the counter value becomes the record's number.
//
// Until number formats ship (S18), Compile accepts only today's format,
// {prefix}{seq:7} with overflow grow: epoch 0, scope key ”. Every record
// number, on the `gaps` and the `gapless` path, is rendered from a Compiled
// (Render in Go, SQL in PostgreSQL), so there is one definition of the
// format, not two.
package numbering

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Overflow is what happens when the counter outgrows {seq:N} (ADR-005 §3.5).
type Overflow string

const (
	// Grow renders the counter with more than N digits.
	Grow Overflow = "grow"
	// Refuse fails the create with number_space_exhausted.
	Refuse Overflow = "refuse"
)

// DefaultTemplate is the format every type has until S18: the type's prefix,
// then the counter zero padded to 7 digits.
const DefaultTemplate = "{prefix}{seq:7}"

// prefixToken is the {prefix} token: the type's number_prefix.
const prefixToken = "{prefix}"

// defaultWidth is the {seq:N} width of DefaultTemplate.
const defaultWidth = 7

// ErrTemplatesUnavailable means Compile was given a template other than
// DefaultTemplate with Grow: number formats are not available yet (S18,
// P6-03 extends Compile).
var ErrTemplatesUnavailable = errors.New("numbering: only the default number format {prefix}{seq:7} with overflow grow is available")

// ErrNumberSpaceExhausted means a counter value does not fit a Refuse
// format's width (ADR-005 §3.5).
var ErrNumberSpaceExhausted = errors.New("numbering: the counter value does not fit the number format")

// Compiled is a number format as compiled at publish and snapshotted on the
// definition version (ADR-005 §3.1). The field names are fixed (P6-02):
// P6-03 may add fields, never rename them.
type Compiled struct {
	// ScopeParts are the tokens the counter's scope key is built from, in
	// template order (date parts, field tokens). None for the default.
	ScopeParts []string
	// Head is the template text before {seq:N}; {prefix} stands for the
	// type's number_prefix.
	Head string
	// Width is N in {seq:N}: the counter's zero-padded width.
	Width int
	// Tail is the template text after {seq:N}.
	Tail string
	// Overflow is what happens past Width digits.
	Overflow Overflow
	// Epoch identifies the compiled template within a type (ADR-005 §3.3):
	// 0 for the default.
	Epoch int
	// CompiledSHA256 is the hex SHA-256 of the compiled template (scope
	// parts, head, width, tail, overflow): equal hashes are the same epoch.
	CompiledSHA256 string
	// TimeZone is the IANA zone date parts are read in (ADR-005 §3.6).
	TimeZone string
}

// Compile compiles template with overflow. It accepts only DefaultTemplate
// with Grow and returns ErrTemplatesUnavailable for anything else.
func Compile(template string, overflow Overflow) (Compiled, error) {
	if template != DefaultTemplate || overflow != Grow {
		return Compiled{}, fmt.Errorf("%w: got %q with overflow %q", ErrTemplatesUnavailable, template, overflow)
	}
	return Default(), nil
}

// Default is Compile(DefaultTemplate, Grow): the format of every type until
// S18. It is compiled per call (one small SHA-256), so there is no package
// state.
func Default() Compiled {
	c := Compiled{Head: prefixToken, Width: defaultWidth, Overflow: Grow, Epoch: 0, TimeZone: "UTC"}
	c.CompiledSHA256 = c.digest()
	return c
}

// digest is the hex SHA-256 of the fields that make the compiled template's
// identity. Epoch is derived from it, and TimeZone changes future scope keys
// only (§3.6), so neither is part of it.
func (c Compiled) digest() string {
	parts := c.ScopeParts
	if parts == nil {
		parts = []string{}
	}
	b, err := json.Marshal(struct {
		ScopeParts []string `json:"scope_parts"`
		Head       string   `json:"head"`
		Width      int      `json:"width"`
		Tail       string   `json:"tail"`
		Overflow   Overflow `json:"overflow"`
	}{parts, c.Head, c.Width, c.Tail, c.Overflow})
	if err != nil {
		panic(err) // strings and an int always encode
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ScopeKey is the counter's scope key, the third column of record_counters'
// key: ” for a format with no scope parts, the default's.
func (c Compiled) ScopeKey() string {
	return ""
}

// supported reports whether c is a form Render and SQL can render: a head of
// {prefix} alone, no tail and no scope parts (the default's shape).
func (c Compiled) supported() error {
	if c.Head != prefixToken || c.Tail != "" || len(c.ScopeParts) != 0 || c.Width < 1 || c.Width > 12 {
		return fmt.Errorf("%w: cannot render head %q, width %d, tail %q, %d scope parts",
			ErrTemplatesUnavailable, c.Head, c.Width, c.Tail, len(c.ScopeParts))
	}
	if c.Overflow != Grow && c.Overflow != Refuse {
		return fmt.Errorf("%w: overflow %q", ErrTemplatesUnavailable, c.Overflow)
	}
	return nil
}

// Render is the record number of counter value n under prefix: the prefix,
// then n zero padded to Width digits. Past Width digits Grow renders every
// digit (INC12345678; the text then no longer sorts in counter order, so
// lists sort on created_at) and Refuse fails with ErrNumberSpaceExhausted.
func (c Compiled) Render(prefix string, n int64) (string, error) {
	if err := c.supported(); err != nil {
		return "", err
	}
	if n < 1 {
		return "", fmt.Errorf("numbering: counter value %d: values start at 1", n)
	}
	digits := strconv.FormatInt(n, 10)
	if len(digits) > c.Width && c.Overflow == Refuse {
		return "", fmt.Errorf("%w: %d has more than %d digits", ErrNumberSpaceExhausted, n, c.Width)
	}
	if pad := c.Width - len(digits); pad > 0 {
		digits = strings.Repeat("0", pad) + digits
	}
	return prefix + digits, nil
}

// SQL is Render as a PostgreSQL text expression, for a number PostgreSQL
// builds in the statement that inserts the record (the `gapless` path):
// prefix and value are SQL expressions for the type's prefix (text) and the
// counter value (bigint). Width is an int, so the expression carries no
// caller text but prefix and value. Only Grow is rendered in SQL: Refuse
// needs an error path SQL has not got.
func (c Compiled) SQL(prefix, value string) (string, error) {
	if err := c.supported(); err != nil {
		return "", err
	}
	if c.Overflow != Grow {
		return "", fmt.Errorf("%w: overflow %q in SQL", ErrTemplatesUnavailable, c.Overflow)
	}
	// lpad alone would cut a value longer than Width.
	w := strconv.Itoa(c.Width)
	return "(" + prefix + " || CASE WHEN " + value + " < 1" + strings.Repeat("0", c.Width) +
		" THEN lpad(" + value + "::text, " + w + ", '0') ELSE " + value + "::text END)", nil
}
