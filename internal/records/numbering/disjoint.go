package numbering

import "strings"

// LiteralHead is the literal text of c before its first variable token, with
// {prefix} resolved to prefix (ADR-005 §3.3): "INC-" for {prefix}-{yyyy}…
// under INC, "INC" for the default. It is what numbering_epochs stores per
// epoch (S18).
func (c Compiled) LiteralHead(prefix string) (string, error) {
	ct, err := c.check()
	if err != nil {
		return "", err
	}
	return ct.literalHead(prefix), nil
}

// literalHead is LiteralHead on checked tokens. The leading-{prefix} rule
// puts {prefix} first; the first variable token is a date part or {seq:N},
// and there is always a {seq:N}, so a literal head is always followed by a
// token that renders digits only.
func (ct compiled) literalHead(prefix string) string {
	var b strings.Builder
	for _, t := range ct.tokens {
		switch t.kind {
		case tokPrefix:
			b.WriteString(prefix)
		case tokLiteral:
			b.WriteString(t.text)
		default:
			return b.String()
		}
	}
	return b.String()
}

// Disjoint reports whether two epochs a and b of one type, whose prefix is
// prefix, can never render the same number (ADR-005 §3.3, TA-1); a publish
// is refused unless the new epoch is disjoint from every other epoch of the
// type, epoch 0 included.
//
// Let s be the shorter literal head and l the longer. They are disjoint when
// s is not a prefix of l (the numbers differ inside both heads), or when the
// character at len(s) in l is outside the character set of the token after
// s, which is a date part or {seq:N}: digits. Equal heads are never
// disjoint, nor are a and b with the same compiled template. Disjoint is
// sound, not complete: it refuses some pairs that could not clash (TA-1
// trades that for a rule a tenant admin can read). A Compiled that fails
// its own check is never disjoint.
func Disjoint(a, b Compiled, prefix string) bool {
	ca, err := a.check()
	if err != nil {
		return false
	}
	cb, err := b.check()
	if err != nil {
		return false
	}
	s, l := ca.literalHead(prefix), cb.literalHead(prefix)
	if len(s) > len(l) {
		s, l = l, s
	}
	if !strings.HasPrefix(l, s) {
		return true
	}
	if len(l) == len(s) {
		return false
	}
	// The shorter template continues with digits only.
	return !isDigit(l[len(s)])
}
