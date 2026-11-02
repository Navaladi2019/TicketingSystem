package verify

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// The CSV form of nava.audit.export/v1 (audit-export-v1.md §5, ADR-004 R22):
// the same lines, one RFC 4180 row each, after a column row. Each row is
// turned back into the JSONL line it stands for and checked by the same
// code, so the two encodings share every check and reason code.
//
// The reader is written here rather than taken from encoding/csv because
// encoding/csv turns CRLF inside a quoted cell into LF, and the diff cell's
// bytes are hashed: §5 forbids normalising them.

// CSVColumns is the column row of a v1 CSV export, in order (§5).
//
//nolint:gochecknoglobals // read-only column table (audit-export-v1.md §5)
var CSVColumns = []string{
	"kind", "seq", "prev_hash", "entry_hash", "envelope_sha256", "genesis", "sealed", "in_verified_range", "sealed_at",
	"tenant_id", "id", "occurred_at", "domain", "action", "subject_type", "subject_id", "entity_definition_id",
	"version_before", "version_after", "actor_principal_id", "actor_system", "source", "source_ref",
	"change_set_id", "causation_id", "causation_depth", "request_id", "backfilled", "content_sha256",
	"diff_sha256", "diff_truncated", "diff", "diff_salt", "diff_redacted", "erased_at", "erasure_ref", "meta",
}

// csvSniff is how Verify tells a CSV file from a JSONL one: a JSONL line
// starts with '{'.
const csvSniff = "kind,"

// The envelope members are the CSV columns tenant_id..diff_truncated.
const envFirst, envLast = 9, 30

//nolint:gochecknoglobals // read-only column tables (audit-export-v1.md §3, §5)
var (
	csvInts  = []string{"seq", "version_before", "version_after", "causation_depth"}
	csvBools = []string{"genesis", "sealed", "in_verified_range", "backfilled", "diff_truncated", "diff_redacted"}
	// csvKindColumns lists, by kind, the columns a row of that kind uses,
	// in the member order of its JSONL line. Every other column is empty.
	csvKindColumns = map[string][]string{
		"anchor": {"kind", "genesis", "seq", "prev_hash", "entry_hash", "envelope_sha256"},
		"stub":   {"kind", "seq", "prev_hash", "entry_hash", "envelope_sha256"},
		"entry": {
			"kind", "sealed", "in_verified_range", "seq", "prev_hash", "entry_hash", "envelope_sha256", "sealed_at",
			"envelope", "diff", "diff_salt", "diff_redacted", "erased_at", "erasure_ref",
		},
		"header":  {"kind", "meta"},
		"trailer": {"kind", "meta"},
	}
	// csvInteger is a JSON integer literal; anything else in an integer
	// cell could change the line's structure.
	csvInteger = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)
)

func columnIndex(name string) int { return slices.Index(CSVColumns, name) }

func verifyCSV(r *bufio.Reader) (*Report, error) {
	v := newVerifier(EncodingCSV)
	cr := &csvReader{r: r}
	for !v.done() {
		cells, err := cr.next()
		if errors.Is(err, io.EOF) {
			break
		}
		var se csvSyntaxError
		if errors.As(err, &se) {
			v.lineNo++
			if v.lineNo == 1 {
				v.fail(nil, ReasonHeaderInvalid, "column row: "+se.msg)
			} else {
				v.fail(nil, ReasonMalformedLine, se.msg)
			}
			return v.rep, nil
		}
		if err != nil {
			return nil, err
		}
		v.lineNo++
		if v.lineNo == 1 {
			if !slices.Equal(cells, CSVColumns) {
				v.fail(nil, ReasonHeaderInvalid, "the first row is not the v1 column row")
			}
			continue
		}
		v.row(cells)
	}
	if !v.done() && v.phase != afterTrailer {
		v.finish()
	}
	return v.rep, nil
}

// row checks one data row: it undoes the formula prefix, rebuilds the JSONL
// line and hands it to line.
func (v *verifier) row(cells []string) {
	if v.phase == afterTrailer {
		v.fail(nil, ReasonLineOrder, "row after the trailer")
		return
	}
	reason := ReasonMalformedLine
	if v.phase == wantHeader {
		reason = ReasonHeaderInvalid
	}
	if len(cells) != len(CSVColumns) {
		v.fail(nil, reason, fmt.Sprintf("row has %d cells, want %d", len(cells), len(CSVColumns)))
		return
	}
	for i, c := range cells {
		if !utf8.ValidString(c) {
			v.fail(nil, ReasonMalformedLine, "invalid UTF-8")
			return
		}
		// §5: the writer prefixes ' to a cell beginning with = + - @ tab CR
		// or ', so the reader drops exactly one leading '.
		cells[i] = strings.TrimPrefix(c, "'")
	}
	b, err := rowLine(cells)
	if err != nil {
		v.fail(nil, reason, err.Error())
		return
	}
	v.line(b)
}

// rowLine rebuilds the JSONL line a data row stands for.
func rowLine(cells []string) ([]byte, error) {
	kind := cells[0]
	cols, ok := csvKindColumns[kind]
	if !ok {
		// line reports the unknown kind.
		var b bytes.Buffer
		b.WriteString(`{"kind":`)
		writeJSONString(&b, kind)
		b.WriteByte('}')
		return b.Bytes(), nil
	}
	for i, name := range CSVColumns {
		used := slices.Contains(cols, name) || (kind == "entry" && i >= envFirst && i <= envLast)
		if !used && cells[i] != "" {
			return nil, fmt.Errorf("column %q is not used by a %s row", name, kind)
		}
	}
	if kind == "header" || kind == "trailer" {
		return metaLine(kind, cells[columnIndex("meta")])
	}
	var b bytes.Buffer
	b.WriteByte('{')
	for i, name := range cols {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJSONString(&b, name)
		b.WriteByte(':')
		if name == "envelope" {
			b.WriteByte('{')
			for j := envFirst; j <= envLast; j++ {
				if j > envFirst {
					b.WriteByte(',')
				}
				writeJSONString(&b, CSVColumns[j])
				b.WriteByte(':')
				if err := writeCell(&b, CSVColumns[j], cells[j]); err != nil {
					return nil, err
				}
			}
			b.WriteByte('}')
			continue
		}
		if err := writeCell(&b, name, cells[columnIndex(name)]); err != nil {
			return nil, err
		}
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// metaLine is the JSONL line a header or trailer row carries in meta.
func metaLine(kind, meta string) ([]byte, error) {
	if meta == "" {
		return nil, fmt.Errorf("a %s row has its JSONL line in meta", kind)
	}
	if strings.ContainsAny(meta, "\r\n") {
		return nil, errors.New("meta holds one JSONL line, without line breaks")
	}
	if m, err := members([]byte(meta)); err == nil {
		var k string
		if json.Unmarshal(m["kind"], &k) == nil && k != kind {
			return nil, fmt.Errorf("meta is a %q line in a %s row", k, kind)
		}
	}
	return []byte(meta), nil
}

// writeCell writes one cell as a JSON value: an empty cell is null (§5),
// integer and boolean columns are their literals, the rest are strings.
func writeCell(b *bytes.Buffer, name, cell string) error {
	switch {
	case cell == "":
		b.WriteString("null")
	case slices.Contains(csvInts, name):
		if !csvInteger.MatchString(cell) {
			return fmt.Errorf("column %q is not an integer", name)
		}
		b.WriteString(cell)
	case slices.Contains(csvBools, name):
		if cell != "true" && cell != "false" {
			return fmt.Errorf("column %q is not true or false", name)
		}
		b.WriteString(cell)
	default:
		writeJSONString(b, cell)
	}
	return nil
}

// writeJSONString writes s with the canonical escapes of audit-export-v1.md
// §2 (RFC 8785), which the strict envelope parser requires: only '"', '\\'
// and U+0000..U+001F are escaped, with the short forms where they exist.
func writeJSONString(b *bytes.Buffer, s string) {
	const hexDigits = "0123456789abcdef"
	b.WriteByte('"')
	for i := range len(s) {
		c := s[i]
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if c < 0x20 {
				b.WriteString(`\u00`)
				b.WriteByte(hexDigits[c>>4])
				b.WriteByte(hexDigits[c&0xf])
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
}

// csvSyntaxError is a row that is not RFC 4180 as §5 pins it.
type csvSyntaxError struct{ msg string }

func (e csvSyntaxError) Error() string { return e.msg }

// csvReader reads RFC 4180 rows strictly: comma separators, '"' quoting
// with '""' for a quote, CRLF row ends (a missing CRLF after the last row
// is tolerated). Quoted cells keep their bytes verbatim, CR and LF included.
type csvReader struct {
	r *bufio.Reader
	n int // bytes read in the current row
}

func (c *csvReader) read() (byte, error) {
	b, err := c.r.ReadByte()
	if err == nil {
		c.n++
		if c.n > MaxLineBytes {
			return 0, csvSyntaxError{fmt.Sprintf("row longer than %d bytes", MaxLineBytes)}
		}
	}
	return b, err
}

// next returns the cells of the next row, or io.EOF after the last row.
func (c *csvReader) next() ([]string, error) {
	c.n = 0
	if _, err := c.r.Peek(1); err != nil {
		return nil, err
	}
	var cells []string
	var cell []byte
	for {
		cell = cell[:0]
		b, err := c.read()
		if err == nil && b == '"' {
			for {
				b, err = c.read()
				if errors.Is(err, io.EOF) {
					return nil, csvSyntaxError{"end of file inside a quoted cell"}
				}
				if err != nil {
					return nil, err
				}
				if b == '"' {
					if b, err = c.read(); err == nil && b == '"' {
						cell = append(cell, '"')
						continue
					}
					break // b, err is what follows the closing quote
				}
				cell = append(cell, b)
			}
		} else {
			for err == nil && b != ',' && b != '\r' && b != '\n' {
				if b == '"' {
					return nil, csvSyntaxError{`a '"' inside an unquoted cell`}
				}
				cell = append(cell, b)
				b, err = c.read()
			}
		}
		cells = append(cells, string(cell))
		switch {
		case errors.Is(err, io.EOF):
			return cells, nil
		case err != nil:
			return nil, err
		case b == ',':
		case b == '\r':
			if b, err = c.read(); err != nil && !errors.Is(err, io.EOF) {
				return nil, err
			}
			if err != nil || b != '\n' {
				return nil, csvSyntaxError{"a CR outside quotes not followed by LF"}
			}
			return cells, nil
		case b == '\n':
			return nil, csvSyntaxError{`an LF row end (rows end with CRLF)`}
		default:
			return nil, csvSyntaxError{"data after a closing quote"}
		}
	}
}
