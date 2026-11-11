package export

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// csvEnvelope are the envelope members in column order (audit-export-v1.md
// section 5).
func csvEnvelope() []string {
	return []string{
		"tenant_id", "id", "occurred_at", "domain", "action", "subject_type", "subject_id",
		"entity_definition_id", "version_before", "version_after", "actor_principal_id",
		"actor_system", "source", "source_ref", "change_set_id", "causation_id",
		"causation_depth", "request_id", "backfilled", "content_sha256", "diff_sha256", "diff_truncated",
	}
}

// csvColumns is the first row: the line members, the flattened envelope, then
// the rest and meta.
func csvColumns() []string {
	return append(append([]string{
		"kind", "seq", "prev_hash", "entry_hash", "envelope_sha256", "genesis", "sealed",
		"in_verified_range", "sealed_at",
	}, csvEnvelope()...), "diff", "diff_salt", "diff_redacted", "erased_at", "erasure_ref", "meta")
}

// WriteCSV streams the same lines as Write, in the same order and with the
// same trailer, as RFC 4180 CSV (audit-export-v1.md section 5): CRLF row ends,
// one row per line, NULL as an empty cell. Cells that begin with = + - @ tab
// CR or ' are prefixed with ' (OWASP, R22). The diff cell keeps its bytes
// verbatim, so an LF inside it stays LF.
func (e *Exporter) WriteCSV(ctx context.Context, w io.Writer, flush func(), s Session) error {
	return e.stream(ctx, w, flush, s, func(bw *bufio.Writer) (func(any) error, error) {
		if err := writeCSVRow(bw, csvColumns()); err != nil {
			return nil, err
		}
		return func(v any) error {
			row, err := csvRow(v)
			if err != nil {
				return err
			}
			return writeCSVRow(bw, row)
		}, nil
	})
}

// csvRow flattens one JSONL line into the cells of csvColumns.
func csvRow(v any) ([]string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	obj := map[string]any{}
	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}
	kind, _ := obj["kind"].(string)
	cols, nenv := csvColumns(), len(csvEnvelope())
	row := make([]string, len(cols))
	if kind == "header" || kind == "trailer" {
		row[0] = kind
		row[len(row)-1] = strings.TrimSuffix(buf.String(), "\n")
		return row, nil
	}
	env, _ := obj["envelope"].(map[string]any)
	for i, c := range cols {
		var val any
		if i >= 9 && i < 9+nenv {
			val = env[c]
		} else {
			val = obj[c]
		}
		if kind == "stub" || kind == "anchor" {
			// genesis belongs to the anchor; sealed flags to entries.
			if c == "sealed" || c == "in_verified_range" {
				val = nil
			}
		} else if c == "genesis" {
			val = nil
		}
		cell, err := csvCell(val)
		if err != nil {
			return nil, fmt.Errorf("audit/export: csv column %s: %w", c, err)
		}
		row[i] = cell
	}
	return row, nil
}

func csvCell(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", nil
	case string:
		return x, nil
	case bool:
		if x {
			return "true", nil
		}
		return "false", nil
	case json.Number:
		return x.String(), nil
	}
	return "", fmt.Errorf("unexpected %T", v)
}

// neutralise prefixes a cell that a spreadsheet could read as a formula. A
// value that already begins with ' is prefixed too, so a reader drops exactly
// one leading ' from any cell that has one.
func neutralise(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r', '\'':
		return "'" + s
	}
	return s
}

func writeCSVRow(bw *bufio.Writer, cells []string) error {
	for i, c := range cells {
		if i > 0 {
			if err := bw.WriteByte(','); err != nil {
				return err
			}
		}
		c = neutralise(c)
		if strings.ContainsAny(c, ",\"\r\n") {
			c = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
		}
		if _, err := bw.WriteString(c); err != nil {
			return err
		}
	}
	_, err := bw.WriteString("\r\n")
	return err
}
