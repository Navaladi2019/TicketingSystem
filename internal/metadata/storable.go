package metadata

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"io"
	"slices"
	"strings"
)

// CheckStorable refuses a schema that PostgreSQL could store as json but not
// read back as jsonb: a string or name holding U+0000 (jsonb refuses the
// \u0000 escape, 22P05). The audit entry of a create or publish reads the
// stored schema as jsonb, so such a schema gave a 500 (TICK-1121). It gives
// *ErrInvalidDefinition with the pointer, under /schema, of every string and
// name that holds one.
//
// It runs on the write path only (create and publish), never in Compile:
// Compile also loads stored versions, and one stored before 2026-11-03 may
// hold a NUL and must keep loading. A schema that is not well-formed JSON,
// or holds invalid UTF-8, passes: Compile reports those.
func CheckStorable(raw json.RawMessage) error {
	dec := jsontext.NewDecoder(bytes.NewReader(raw))
	var paths []string
	for {
		tok, err := dec.ReadToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil // Compile reports it
		}
		if tok.Kind() == '"' && strings.IndexByte(tok.String(), 0) >= 0 {
			// For a name, the pointer names the member it starts.
			paths = append(paths, "/schema"+string(dec.StackPointer()))
		}
		if dec.StackDepth() == 0 {
			break
		}
	}
	if len(paths) == 0 {
		return nil
	}
	slices.Sort(paths)
	return &ErrInvalidDefinition{Paths: slices.Compact(paths)}
}
