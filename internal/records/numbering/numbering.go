// Package numbering is the compiled form of a record number format (ADR-005
// §3.1): what a create reads to know which counter it takes (the scope key)
// and how the counter value becomes the record's number.
//
// Compile accepts every template of the ADR-005 §3.1 grammar except
// {field:…} (S19) and refuses the rest with a *FormatError. Disjoint is the
// §3.3 rule that two epochs of one type never render the same number. A type
// with no linked format uses Default, {prefix}{seq:7} with overflow grow:
// epoch 0, scope key ”; a linked one the snapshot on its definition version
// (TICK-673). Every record
// number, on the `gaps` and the `gapless` path, is rendered from a Compiled
// (Render in Go, SQL in PostgreSQL), so there is one definition of the
// format, not two. At allocation PostgreSQL computes the scope key, the date
// parts and period_end from now() in the numbering time zone (Scope, A7).
package numbering

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
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

// EpochUnassigned is the Epoch of a compiled non-default format before the
// publish takes its epoch from the registry (ADR-005 §3.3, S18).
const EpochUnassigned = -1

// MaxScopeKey is the longest scope key record_counters accepts (ADR-005
// §4.1, CHECK (length(scope_key) <= 128)).
const MaxScopeKey = 128

// ErrTemplatesUnavailable means a Compiled cannot be rendered on this path:
// SQL was given no local-time expression for a format with date parts.
var ErrTemplatesUnavailable = errors.New("numbering: this number format cannot be rendered here yet")

// ErrNumberSpaceExhausted means a counter value does not fit a Refuse
// format's width (ADR-005 §3.5).
var ErrNumberSpaceExhausted = errors.New("numbering: the counter value does not fit the number format")

// ErrInvalidCompiled means a Compiled is not what Compile makes of its own
// Head, Width, Tail and Overflow: a corrupt or edited snapshot. Render, SQL,
// ScopeKeyAt and LiteralHead refuse it, so text that never passed the
// compiler never reaches a number or a SQL statement.
var ErrInvalidCompiled = errors.New("numbering: the compiled number format does not match its template")

// Compiled is a number format as compiled at publish and snapshotted on the
// definition version (ADR-005 §3.1). The field names are fixed (P6-02):
// P6-03 may add fields, never rename them.
type Compiled struct {
	// ScopeParts are the tokens the counter's scope key is built from, in
	// template order: the date parts ({yyyy}, {yy}, {MM}, {dd}). None for
	// the default.
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
	// 0 for the default, EpochUnassigned for any other format until the
	// publish takes its epoch.
	Epoch int
	// CompiledSHA256 is the hex SHA-256 of the compiled template (scope
	// parts, head, width, tail, overflow): equal hashes are the same epoch.
	CompiledSHA256 string
	// TimeZone is the IANA zone date parts are read in (ADR-005 §3.6); ""
	// means the tenant's time_zone setting.
	TimeZone string
}

// Compile compiles template with overflow and the tenant's time zone. See
// CompileFormat.
func Compile(template string, overflow Overflow) (Compiled, error) {
	return CompileFormat(Format{Template: template, Overflow: overflow})
}

// Default is Compile(DefaultTemplate, Grow): the format of every type until
// S18. It is compiled per call (one small SHA-256), so there is no package
// state.
func Default() Compiled {
	c := Compiled{Head: prefixToken, Width: defaultWidth, Overflow: Grow, Epoch: 0, TimeZone: "UTC"}
	c.CompiledSHA256 = c.digest()
	return c
}

// fromTemplate fills a Compiled from a compiled template. The default gets
// epoch 0; every other format EpochUnassigned.
func fromTemplate(template string, overflow Overflow, ct compiled) Compiled {
	s := ct.tokens[ct.seq]
	c := Compiled{
		ScopeParts: ct.scopeParts(),
		Head:       template[:s.pos],
		Width:      s.width,
		Tail:       template[s.pos+len(s.text):],
		Overflow:   overflow,
		Epoch:      EpochUnassigned,
	}
	c.CompiledSHA256 = c.digest()
	if c.CompiledSHA256 == Default().CompiledSHA256 {
		c.Epoch = 0
	}
	return c
}

// digest is the hex SHA-256 of the fields that make the compiled template's
// identity: the JSON of an object with the fixed keys below, in that order,
// and no HTML-escapable characters (the grammar has none), so the bytes do
// not depend on the Go version (golden tests pin it). Epoch is derived from
// it, and TimeZone changes future scope keys only (§3.6), so neither is
// part of it.
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

// check recompiles c's own template and returns its tokens when c is
// exactly what Compile makes of it, else ErrInvalidCompiled.
func (c Compiled) check() (compiled, error) {
	template := c.Head + "{seq:" + strconv.Itoa(c.Width) + "}" + c.Tail
	ct, ferr := compileTemplate(template, c.Overflow)
	if ferr != nil {
		return compiled{}, fmt.Errorf("%w: %w", ErrInvalidCompiled, ferr)
	}
	want := fromTemplate(template, c.Overflow, ct)
	if want.Head != c.Head || want.Width != c.Width || want.Tail != c.Tail ||
		!slices.Equal(want.ScopeParts, c.ScopeParts) || want.CompiledSHA256 != c.CompiledSHA256 {
		return compiled{}, fmt.Errorf("%w: scope parts, head, width, tail or hash differ", ErrInvalidCompiled)
	}
	if c.Epoch < EpochUnassigned || (c.Epoch == 0) != (want.Epoch == 0) {
		return compiled{}, fmt.Errorf("%w: epoch %d: only the default is epoch 0", ErrInvalidCompiled, c.Epoch)
	}
	return ct, nil
}

// ScopeKeyAt is the counter's scope key, the third column of
// record_counters' key (ADR-005 §4.1), for datePart, the format's date
// parts at the allocation instant concatenated in ScopeParts order (see
// DatePart; at allocation PostgreSQL computes it, A7, and Scope is the
// same key as SQL builds it):
//
//   - the default (epoch 0): ”, and datePart must be empty;
//   - any other epoch: e<epoch>, then |d:<datePart> when the format has date
//     parts, for example e1|d:202611.
//
// The key holds only the epoch number and datePart's digits, never tenant
// text, so it cannot carry another tenant's identifier, and it is at most
// MaxScopeKey characters.
func (c Compiled) ScopeKeyAt(datePart string) (string, error) {
	ct, err := c.check()
	if err != nil {
		return "", err
	}
	if err := checkDatePart(ct, datePart); err != nil {
		return "", err
	}
	switch c.Epoch {
	case 0:
		return "", nil
	case EpochUnassigned:
		return "", errors.New("numbering: the format has no epoch yet")
	}
	key := "e" + strconv.Itoa(c.Epoch)
	if datePart != "" {
		key += "|d:" + datePart
	}
	if len(key) > MaxScopeKey {
		return "", fmt.Errorf("numbering: the scope key is longer than %d characters", MaxScopeKey)
	}
	return key, nil
}

// DatePart is the format's date parts at at, read in at's location,
// concatenated in ScopeParts order: "202611" for {yyyy}{MM} in November
// 2026. It is "" for a format with no date part.
func (c Compiled) DatePart(at time.Time) (string, error) {
	ct, err := c.check()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, t := range ct.tokens {
		if t.kind == tokDate {
			v, err := renderDate(t, at)
			if err != nil {
				return "", err
			}
			b.WriteString(v)
		}
	}
	return b.String(), nil
}

func renderDate(t token, at time.Time) (string, error) {
	v := at.Format(t.layout)
	if len(v) != t.width {
		return "", fmt.Errorf("numbering: year %d does not fit %s", at.Year(), t.text)
	}
	return v, nil
}

// Render is RenderAt for a format with no date part.
func (c Compiled) Render(prefix string, n int64) (string, error) {
	if len(c.ScopeParts) > 0 {
		return "", errors.New("numbering: the number format has date parts: use RenderAt")
	}
	return c.RenderAt(prefix, time.Time{}, n)
}

// RenderAt is the record number of counter value n under prefix at the
// allocation instant at, whose date parts are read in at's location (the
// caller converts it to the numbering time zone). {seq:N} is n zero padded
// to Width digits. Past Width digits Grow renders every digit (INC12345678;
// the text then no longer sorts in counter order, so lists sort on
// created_at) and Refuse fails with ErrNumberSpaceExhausted.
func (c Compiled) RenderAt(prefix string, at time.Time, n int64) (string, error) {
	ct, err := c.check()
	if err != nil {
		return "", err
	}
	return c.render(ct, prefix, n, func(t token) (string, error) { return renderDate(t, at) })
}

// RenderParts is RenderAt with the date parts given as datePart, the digits
// PostgreSQL computed for the counter's scope key (DatePartOf), instead of
// an instant: the number then carries exactly the date its counter was
// scoped by (ADR-005 A7).
func (c Compiled) RenderParts(prefix, datePart string, n int64) (string, error) {
	ct, err := c.check()
	if err != nil {
		return "", err
	}
	if err := checkDatePart(ct, datePart); err != nil {
		return "", err
	}
	rest := datePart
	return c.render(ct, prefix, n, func(t token) (string, error) {
		v := rest[:t.width]
		rest = rest[t.width:]
		return v, nil
	})
}

// render renders ct, c's tokens, with date(t) for each date token.
func (c Compiled) render(ct compiled, prefix string, n int64, date func(token) (string, error)) (string, error) {
	if n < 1 {
		return "", fmt.Errorf("numbering: counter value %d: values start at 1", n)
	}
	if !c.Fits(n) {
		return "", fmt.Errorf("%w: %d has more than %d digits", ErrNumberSpaceExhausted, n, c.Width)
	}
	digits := strconv.FormatInt(n, 10)
	if pad := c.Width - len(digits); pad > 0 {
		digits = strings.Repeat("0", pad) + digits
	}
	var b strings.Builder
	for _, t := range ct.tokens {
		switch t.kind {
		case tokPrefix:
			b.WriteString(prefix)
		case tokLiteral:
			b.WriteString(t.text)
		case tokSeq:
			b.WriteString(digits)
		case tokDate:
			v, err := date(t)
			if err != nil {
				return "", err
			}
			b.WriteString(v)
		}
	}
	return b.String(), nil
}

// Fits reports whether counter value n renders under c: always under Grow,
// and under Refuse only within Width digits (ADR-005 §3.5).
func (c Compiled) Fits(n int64) bool {
	return c.Overflow != Refuse || len(strconv.FormatInt(n, 10)) <= c.Width
}

// SQL is Render as a PostgreSQL text expression, for a number PostgreSQL
// builds in the statement that inserts the record (the `gapless` path):
// prefix and value are SQL expressions for the type's prefix (text) and the
// counter value (bigint), and local, for a format with date parts, one for
// the allocation instant's wall time in the numbering time zone (timestamp
// without time zone, ADR-005 A7); "" for a format without. Literal template
// text never enters the SQL text: each literal run, and each date part's
// to_char pattern, is appended to args and referenced as $<len(args)>::text,
// and SQL returns the extended args (Security, TICK-666 review). Width is an
// int. Under Refuse a value past Width digits renders NULL, which the
// records.number NOT NULL constraint refuses: the caller, which knows the
// value, reports ErrNumberSpaceExhausted (Fits).
func (c Compiled) SQL(prefix, value, local string, args []any) (string, []any, error) {
	ct, err := c.check()
	if err != nil {
		return "", args, err
	}
	if len(c.ScopeParts) > 0 && local == "" {
		return "", args, fmt.Errorf("%w: date parts in SQL need the local time", ErrTemplatesUnavailable)
	}
	// lpad alone would cut a value longer than Width.
	w := strconv.Itoa(c.Width)
	fits := value + " < 1" + strings.Repeat("0", c.Width)
	seq := "CASE WHEN " + fits + " THEN lpad(" + value + "::text, " + w + ", '0') ELSE " + value + "::text END"
	if c.Overflow == Refuse {
		seq = "CASE WHEN " + fits + " THEN lpad(" + value + "::text, " + w + ", '0') END"
	}
	parts := make([]string, 0, len(ct.tokens))
	for _, t := range ct.tokens {
		switch t.kind {
		case tokPrefix:
			parts = append(parts, prefix)
		case tokSeq:
			parts = append(parts, seq)
		case tokLiteral:
			args = append(args, t.text)
			parts = append(parts, "$"+strconv.Itoa(len(args))+"::text")
		case tokDate:
			args = append(args, sqlPattern(t))
			parts = append(parts, "to_char("+local+", $"+strconv.Itoa(len(args))+"::text)")
		}
	}
	return "(" + strings.Join(parts, " || ") + ")", args, nil
}
