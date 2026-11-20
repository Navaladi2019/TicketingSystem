package metadata

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"io"
)

// invalidUTF8At reports the JSON pointer of the first string or name in raw
// that holds invalid UTF-8 or a lone surrogate escape, the check
// encoding/json/v2 makes by default (TICK-964). It is the same rule as
// internal/platform/strictjson, restated here because this package imports
// no other platform package (P3-03). A duplicate key or a syntax error is
// not reported; scanSchema reports those.
func invalidUTF8At(raw []byte) (string, bool) {
	ptr, offset, err := firstJSONError(raw)
	if err == nil || errors.Is(err, jsontext.ErrDuplicateName) {
		return "", false
	}
	// jsontext exports no sentinel for invalid UTF-8: with only that check
	// off, a read that gets past the same offset proves it was the cause.
	_, permOffset, permErr := firstJSONError(raw, jsontext.AllowInvalidUTF8(true))
	if permErr == nil || permOffset != offset {
		return ptr, true
	}
	return "", false
}

func firstJSONError(raw []byte, opts ...jsontext.Options) (string, int64, error) {
	dec := jsontext.NewDecoder(bytes.NewReader(raw), opts...)
	for {
		_, err := dec.ReadToken()
		if errors.Is(err, io.EOF) {
			return "", 0, nil
		}
		if err != nil {
			var se *jsontext.SyntacticError
			if errors.As(err, &se) {
				return string(se.JSONPointer), se.ByteOffset, err
			}
			return "", 0, err
		}
		if dec.StackDepth() == 0 {
			return "", 0, nil
		}
	}
}
