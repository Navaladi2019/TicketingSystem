package server

import (
	"encoding/json"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictjson"
)

// decodeBody decodes a request body's top-level object with encoding/json/v2
// (strictjson.Unmarshal), which refuses a duplicate member name or invalid
// UTF-8 anywhere in the body, nested raw members included. On failure it
// returns the one field error to report: the strict-JSON violation with its
// pointer and rule (duplicate, invalid_utf8), or ("", "type") when the body
// is not a JSON object. The boundary middleware (apihttp.StrictBody) refuses
// the same bodies first; this keeps the parsers safe on their own (TICK-964).
func decodeBody(raw []byte) (map[string]json.RawMessage, *problem.FieldError) {
	var fields map[string]json.RawMessage
	if err := strictjson.Unmarshal(raw, &fields); err != nil || fields == nil {
		fe := invalid("", "type")
		if v, bad := strictjson.Check(raw); bad {
			fe = invalid(v.Pointer, v.Rule)
		}
		return nil, &fe
	}
	return fields, nil
}
