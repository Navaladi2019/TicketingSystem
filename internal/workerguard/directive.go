package workerguard

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"

	"github.com/google/uuid"
	"go.temporal.io/sdk/converter"
)

// Directive is the one activity result that is not references only (P3-26,
// PLAN-003 D-3-15, groomed on TICK-360 and TICK-363). The workflow
// interpreter holds no definition and no record data (ADR-001 R1, CI-7); a
// Directive tells it the step it is on, what kind of step that is, the next
// step, and, for a wait, how long and for which signal. Every value comes from
// the published definition, never from a record field (P3-20).
//
// The guard matches it by Go type, as it does outbox.Delivery and RunInput,
// and checks the bytes the DataConverter records against the per-kind table
// in directiveOK. It is accepted as the result of an activity only: not as an
// argument, a workflow or update result, a side effect, failure details, or
// the result of a local activity.
type Directive struct {
	Step        uuid.UUID  `json:"step"`
	Kind        Kind       `json:"kind"`
	WaitSeconds int32      `json:"wait_seconds"`
	Signal      *uuid.UUID `json:"signal,omitempty"`
	Next        *uuid.UUID `json:"next,omitempty"`
	OnTimeout   *uuid.UUID `json:"on_timeout,omitempty"`
}

// Kind is the kind of a Directive's step. The constants below are the only
// valid values; directiveRules is keyed by them and is the guard's only list.
type Kind string

// The step kinds.
const (
	KindTimer        Kind = "timer"
	KindWaitSignal   Kind = "wait_signal"
	KindUpdateRecord Kind = "update_record"
	KindEnd          Kind = "end"
)

// MinWaitSeconds and MaxWaitSeconds bound the wait a timer or wait_signal
// Directive carries: 1 second to 30 days (def.MinWaitSeconds and
// def.MaxWaitSeconds must equal them; internal/workflow/interp tests it).
const (
	MinWaitSeconds = 1
	MaxWaitSeconds = 30 * 24 * 60 * 60
)

// directiveRule is one row of the per-kind table: whether each optional
// reference is required (true) or must be omitted (false), and whether the
// wait is MinWaitSeconds..MaxWaitSeconds (true) or 0 (false). Step is always required.
type directiveRule struct {
	next, signal, onTimeout, wait bool
}

//nolint:gochecknoglobals // the fixed per-kind table and the type the result rule matches on
var (
	directiveRules = map[Kind]directiveRule{
		KindTimer:        {next: true, wait: true},
		KindWaitSignal:   {next: true, signal: true, onTimeout: true, wait: true},
		KindUpdateRecord: {next: true},
		KindEnd:          {},
	}
	directiveType = reflect.TypeFor[Directive]()
)

// isDirective reports whether v is a Directive or a pointer to one. Another
// type with the same JSON shape, or a map, is not.
func isDirective(v any) bool {
	t := reflect.TypeOf(v)
	if t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t == directiveType
}

// directiveResultOK reports whether v, a Directive, may be recorded as an
// activity result: its encoding is JSON and passes directiveOK.
func directiveResultOK(v any) bool {
	p, err := DataConverter().ToPayload(v)
	if err != nil || string(p.GetMetadata()[converter.MetadataEncoding]) != converter.MetadataEncodingJSON {
		return false
	}
	return directiveOK(p.GetData())
}

// directiveFields are the keys a Directive encodes to.
const (
	fieldStep        = "step"
	fieldKind        = "kind"
	fieldWaitSeconds = "wait_seconds"
	fieldSignal      = "signal"
	fieldNext        = "next"
	fieldOnTimeout   = "on_timeout"
)

// directiveOK checks the bytes that are recorded, not the Go fields, so a
// custom MarshalJSON cannot add data past it. The encoding must be one JSON
// object with:
//   - only the six Directive keys, each at most once (a repeated key would
//     record a value a decoder that keeps the last one never sees);
//   - step, kind and wait_seconds present;
//   - kind one of the Kind constants, and the other fields as its row of
//     directiveRules says;
//   - every reference a canonical, non-nil UUID string; an optional reference
//     omitted, never null;
//   - wait_seconds an integer literal;
//   - nothing after the object.
func directiveOK(data []byte) bool {
	fields, ok := directiveFields(data)
	if !ok {
		return false
	}
	var kind string
	if !stringField(fields[fieldKind], &kind) {
		return false
	}
	rule, known := directiveRules[Kind(kind)]
	if !known || !refOK(fields[fieldStep]) {
		return false
	}
	for name, required := range map[string]bool{fieldNext: rule.next, fieldSignal: rule.signal, fieldOnTimeout: rule.onTimeout} {
		raw, present := fields[name]
		if present != required || (present && !refOK(raw)) {
			return false
		}
	}
	wait, ok := waitSeconds(fields[fieldWaitSeconds])
	if !ok {
		return false
	}
	if rule.wait {
		return wait >= MinWaitSeconds && wait <= MaxWaitSeconds
	}
	return wait == 0
}

// directiveFields reads the top-level object of data token by token, so a
// repeated or unknown key is refused, and returns each value's raw bytes.
func directiveFields(data []byte) (map[string]json.RawMessage, bool) {
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, false
	}
	fields := make(map[string]json.RawMessage, 6)
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, false
		}
		name, isKey := tok.(string)
		if !isKey || !knownField(name) {
			return nil, false
		}
		if _, dup := fields[name]; dup {
			return nil, false
		}
		var raw json.RawMessage
		if dec.Decode(&raw) != nil {
			return nil, false
		}
		fields[name] = raw
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, false
	}
	for _, name := range []string{fieldStep, fieldKind, fieldWaitSeconds} {
		if _, ok := fields[name]; !ok {
			return nil, false
		}
	}
	return fields, true
}

func knownField(name string) bool {
	switch name {
	case fieldStep, fieldKind, fieldWaitSeconds, fieldSignal, fieldNext, fieldOnTimeout:
		return true
	}
	return false
}

// stringField decodes raw into s only if raw is a JSON string (not null).
func stringField(raw json.RawMessage, s *string) bool {
	return len(raw) > 0 && raw[0] == '"' && json.Unmarshal(raw, s) == nil
}

// refOK: a canonical, non-nil UUID string.
func refOK(raw json.RawMessage) bool {
	var s string
	if !stringField(raw, &s) {
		return false
	}
	u, err := uuid.Parse(s)
	return err == nil && u != uuid.Nil && u.String() == s
}

// waitSeconds decodes an integer literal: no fraction, no exponent, no
// string, no null. Out-of-range values fail to decode.
func waitSeconds(raw json.RawMessage) (int32, bool) {
	if len(raw) == 0 || bytes.ContainsAny(raw, ".eE\"") || bytes.Equal(raw, []byte("null")) {
		return 0, false
	}
	var n int32
	if json.Unmarshal(raw, &n) != nil {
		return 0, false
	}
	return n, true
}
