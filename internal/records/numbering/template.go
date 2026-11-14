package numbering

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Error codes of a refused number format (ADR-005 §3.1), both 422.
const (
	// CodeInvalid is any template, overflow or time zone the grammar
	// refuses: an unknown token, a {field:…} token (S19), a bad literal, no
	// or two {seq:N}, an ambiguous grow, a rendering longer than 64.
	CodeInvalid = "number_format_invalid"
	// CodePrefixRequired is a template that does not start with {prefix},
	// or whose next token produces letters (the leading-{prefix} rule).
	CodePrefixRequired = "number_format_prefix_required"
)

// Sentinels for errors.Is on a *FormatError.
var (
	ErrFormatInvalid  = errors.New("numbering: " + CodeInvalid)
	ErrPrefixRequired = errors.New("numbering: " + CodePrefixRequired)
)

// Grammar bounds (ADR-005 §3.1).
const (
	// MaxWidth is the largest N in {seq:N}.
	MaxWidth = 12
	// MaxLiteralRun is the longest run of literal characters.
	MaxLiteralRun = 16
	// MaxRendered is the longest number a format may render.
	MaxRendered = 64
	// maxPrefix is the longest number_prefix (^[A-Z]{2,6}$).
	maxPrefix = 6
	// maxCounterDigits is the most digits an int64 counter renders in, the
	// width a growing {seq:N} is measured at.
	maxCounterDigits = 19
)

// FormatError is a number format the compiler refuses. Reason is fixed text:
// it never echoes the template, so it is safe in a problem+json detail.
type FormatError struct {
	// Code is CodeInvalid or CodePrefixRequired.
	Code string
	// Field is "template", "overflow", "time_zone" or, for a document the
	// JSON Schema refuses, the JSON pointer of the failing value.
	Field string
	// Position is the 0-based byte offset in the template of the offending
	// token or character, or -1 when the error is not about one position.
	Position int
	// Reason says what is wrong, in fixed text.
	Reason string
}

func (e *FormatError) Error() string {
	if e.Position < 0 {
		return fmt.Sprintf("numbering: %s: %s: %s", e.Code, e.Field, e.Reason)
	}
	return fmt.Sprintf("numbering: %s: %s at %d: %s", e.Code, e.Field, e.Position, e.Reason)
}

// Is matches ErrFormatInvalid or ErrPrefixRequired by Code.
func (e *FormatError) Is(target error) bool {
	return (target == ErrFormatInvalid && e.Code == CodeInvalid) ||
		(target == ErrPrefixRequired && e.Code == CodePrefixRequired)
}

// Status is the HTTP status of every format error: 422.
func (e *FormatError) Status() int { return 422 }

func invalid(field string, pos int, reason string) *FormatError {
	return &FormatError{Code: CodeInvalid, Field: field, Position: pos, Reason: reason}
}

type tokenKind int

const (
	tokLiteral tokenKind = iota
	tokPrefix
	tokDate
	tokSeq
	tokField
)

// token is one lexed piece of a template.
type token struct {
	kind tokenKind
	// text is the token as written ({yyyy}, {seq:4}) or the literal run.
	text string
	// pos is the byte offset of text in the template.
	pos int
	// width is a date part's digits or N of {seq:N}.
	width int
	// layout is a date part's time.Format layout.
	layout string
}

// digits reports whether the token always renders decimal digits.
func (t token) digits() bool { return t.kind == tokDate || t.kind == tokSeq }

// datePart is a date token's width and Go layout; ok is false for any
// other text.
func datePart(text string) (width int, layout string, ok bool) {
	switch text {
	case "{yyyy}":
		return 4, "2006", true
	case "{yy}":
		return 2, "06", true
	case "{MM}":
		return 2, "01", true
	case "{dd}":
		return 2, "02", true
	}
	return 0, "", false
}

func isLiteralChar(c byte) bool {
	return ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9') || c == '_' || c == '-' || c == '/' || c == '.'
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

func isUpper(c byte) bool { return 'A' <= c && c <= 'Z' }

// lex splits template into tokens. Every character outside a token must be
// a literal character, so a quote, a semicolon or a space never gets past
// it (Security, TICK-666 review).
func lex(template string) ([]token, *FormatError) {
	var out []token
	for i := 0; i < len(template); {
		c := template[i]
		switch {
		case c == '{':
			end := strings.IndexByte(template[i:], '}')
			if end < 0 {
				return nil, invalid("template", i, "a token is not closed with }")
			}
			t, err := lexToken(template[i:i+end+1], i)
			if err != nil {
				return nil, err
			}
			out = append(out, t)
			i += end + 1
		case isLiteralChar(c):
			start := i
			for i < len(template) && isLiteralChar(template[i]) {
				i++
			}
			if i-start > MaxLiteralRun {
				return nil, invalid("template", start+MaxLiteralRun, "a literal run is longer than 16 characters")
			}
			out = append(out, token{kind: tokLiteral, text: template[start:i], pos: start})
		default:
			return nil, invalid("template", i, "the character is not allowed: literals are A-Z, 0-9 and _ - / .")
		}
	}
	return out, nil
}

func lexToken(text string, pos int) (token, *FormatError) {
	t := token{text: text, pos: pos}
	body := text[1 : len(text)-1]
	if w, layout, ok := datePart(text); ok {
		t.kind, t.width, t.layout = tokDate, w, layout
		return t, nil
	}
	switch {
	case text == prefixToken:
		t.kind = tokPrefix
	case strings.HasPrefix(body, "seq:"):
		n := body[len("seq:"):]
		w, err := strconv.Atoi(n)
		// Canonical N only: no sign, no leading zero, 1..12.
		if err != nil || n[0] < '1' || n[0] > '9' || w > MaxWidth {
			return t, invalid("template", pos, "{seq:N} needs N from 1 to 12")
		}
		t.kind, t.width = tokSeq, w
	case strings.HasPrefix(body, "field:"):
		t.kind = tokField
	default:
		return t, invalid("template", pos, "unknown token")
	}
	return t, nil
}

// compiled is a template's tokens with the index of its {seq:N}.
type compiled struct {
	tokens []token
	seq    int
}

// compileTemplate checks template under overflow against the grammar,
// in this order: overflow, lexing, the leading-{prefix} rule, field and
// repeated-{prefix} tokens, exactly one {seq:N}, grow ambiguity, length.
func compileTemplate(template string, overflow Overflow) (compiled, *FormatError) {
	if overflow != Grow && overflow != Refuse {
		return compiled{}, invalid("overflow", -1, "overflow must be grow or refuse")
	}
	toks, ferr := lex(template)
	if ferr != nil {
		return compiled{}, ferr
	}
	if len(toks) == 0 || toks[0].kind != tokPrefix {
		return compiled{}, &FormatError{Code: CodePrefixRequired, Field: "template", Position: 0,
			Reason: "the template must start with {prefix}"}
	}
	if len(toks) > 1 {
		if n := toks[1]; n.kind == tokPrefix || n.kind == tokField || (n.kind == tokLiteral && isUpper(n.text[0])) {
			return compiled{}, &FormatError{Code: CodePrefixRequired, Field: "template", Position: n.pos,
				Reason: "the token after {prefix} must not produce letters"}
		}
	}
	c := compiled{tokens: toks, seq: -1}
	for i, t := range toks {
		switch {
		case t.kind == tokField:
			return compiled{}, invalid("template", t.pos, "{field:…} tokens are not available yet")
		case t.kind == tokPrefix && i > 0:
			return compiled{}, invalid("template", t.pos, "{prefix} appears only once, at the start")
		case t.kind == tokSeq && c.seq >= 0:
			return compiled{}, invalid("template", t.pos, "the template has more than one {seq:N}")
		case t.kind == tokSeq:
			c.seq = i
		}
	}
	if c.seq < 0 {
		return compiled{}, invalid("template", len(template), "the template has no {seq:N}")
	}
	if overflow == Grow {
		if err := c.unambiguous(); err != nil {
			return compiled{}, err
		}
	}
	if c.maxLen(overflow) > MaxRendered {
		return compiled{}, invalid("template", -1, "the number can render longer than 64 characters")
	}
	return c, nil
}

// unambiguous is the grow rule (ADR-005 §3.5). Under grow {seq:N} has a
// variable width, so a reader can only split the number if the characters
// either side of it are not digits: it must not touch a date part or a
// literal digit before or after. ({prefix} is variable width too, but the
// leading-{prefix} rule already keeps letters out of the token after it.)
func (c compiled) unambiguous() *FormatError {
	s := c.tokens[c.seq]
	if c.seq > 0 {
		p := c.tokens[c.seq-1]
		if p.digits() || (p.kind == tokLiteral && isDigit(p.text[len(p.text)-1])) {
			return invalid("overflow", s.pos, "grow needs a non-digit before {seq:N}; use refuse")
		}
	}
	if c.seq+1 < len(c.tokens) {
		n := c.tokens[c.seq+1]
		if n.digits() || (n.kind == tokLiteral && isDigit(n.text[0])) {
			return invalid("overflow", s.pos, "grow needs a non-digit after {seq:N}; use refuse")
		}
	}
	return nil
}

// maxLen is the longest number the template renders: the longest prefix,
// and a growing {seq:N} at the most digits a counter has.
func (c compiled) maxLen(overflow Overflow) int {
	n := 0
	for _, t := range c.tokens {
		switch t.kind {
		case tokPrefix:
			n += maxPrefix
		case tokLiteral:
			n += len(t.text)
		case tokDate:
			n += t.width
		case tokSeq:
			if overflow == Grow {
				n += max(t.width, maxCounterDigits)
			} else {
				n += t.width
			}
		}
	}
	return n
}

// scopeParts are the date tokens in template order.
func (c compiled) scopeParts() []string {
	var out []string
	for _, t := range c.tokens {
		if t.kind == tokDate {
			out = append(out, t.text)
		}
	}
	return out
}
