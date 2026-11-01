package chain

import (
	"bytes"
	"slices"
	"unicode/utf16"
)

const hexDigits = "0123456789abcdef"

// writeJCSString writes s as an RFC 8785 §3.2.2.2 string (the ECMAScript
// JSON.stringify form): only '"', '\\' and U+0000..U+001F are escaped; the
// short forms \b \t \n \f \r are used where they exist, \u00xx with
// lowercase hex otherwise; everything else, '<', '>', '&', U+2028, U+2029
// and non-ASCII included, is written as raw UTF-8. Callers validate UTF-8
// first.
func writeJCSString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			b.WriteString(`\"`)
		case c == '\\':
			b.WriteString(`\\`)
		case c == '\b':
			b.WriteString(`\b`)
		case c == '\t':
			b.WriteString(`\t`)
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\f':
			b.WriteString(`\f`)
		case c == '\r':
			b.WriteString(`\r`)
		case c < 0x20:
			b.WriteString(`\u00`)
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0xf])
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
}

// sortByUTF16 sorts members by their keys' UTF-16 code units, as RFC 8785
// §3.2.3 requires. (Go's string order is by UTF-8 bytes, which differs above
// U+FFFF.)
func sortByUTF16[T any](ms []T, key func(T) string) {
	slices.SortStableFunc(ms, func(a, b T) int {
		return slices.Compare(utf16.Encode([]rune(key(a))), utf16.Encode([]rune(key(b))))
	})
}
