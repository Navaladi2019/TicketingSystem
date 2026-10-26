package workerguard

import (
	"bytes"
	"encoding/json"
	"maps"
	"reflect"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// RunInput is the input of a relay-started workflow that continues as new
// (CI-7, TICK-216): the relay's outbox.Delivery on the first run, plus the
// applied-signal set (I7) and the workflow's own references, carried to the
// next run. Delivery is embedded, so the relay's start input decodes into it.
//
// Refs holds references only, like an outbox payload; omitted, it is {}.
type RunInput struct {
	outbox.Delivery
	Seen AppliedSignals  `json:"seen"`
	Refs json.RawMessage `json:"refs,omitempty"`
}

//nolint:gochecknoglobals // fixed types the argument rule matches on
var (
	deliveryType = reflect.TypeFor[outbox.Delivery]()
	runInputType = reflect.TypeFor[RunInput]()
)

// argsOutcome applies the references-only rule (ADR-001 R1, CI-7) to the
// arguments of an activity, local activity, child, continue-as-new or update:
// outcomeAllow if every one is accepted, outcomeRecordData if not. No
// arguments is allowed.
func argsOutcome(args []any) string {
	for _, a := range args {
		if !argOK(a) {
			return outcomeRecordData
		}
	}
	return outcomeAllow
}

// argOK reports whether arg may be recorded in a history (TICK-216 contract
// decision). The Go type decides the rule, the JSON the default converter
// encodes is what is checked, since that is what the history records:
//   - outbox.Delivery: Payload passes outbox.ValidatePayload (the relay's rule);
//   - RunInput: Payload and Refs pass it, and every Seen key is a UUID and
//     every value an RFC3339 time;
//   - any other value: its encoding is a JSON object that passes it.
//
// A pointer is matched as the type it points to. A struct of another type
// with the same JSON shape as RunInput is "any other value", and fails.
func argOK(arg any) bool {
	p, err := converter.GetDefaultDataConverter().ToPayload(arg)
	if err != nil {
		return false
	}
	t := reflect.TypeOf(arg)
	if t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t {
	case deliveryType:
		return deliveryOK(p.GetData())
	case runInputType:
		return runInputOK(p.GetData())
	}
	return outbox.ValidatePayload(p.GetData()) == nil
}

func deliveryOK(data []byte) bool {
	var d struct {
		Payload json.RawMessage `json:"payload"`
	}
	return json.Unmarshal(data, &d) == nil && outbox.ValidatePayload(d.Payload) == nil
}

func runInputOK(data []byte) bool {
	var in struct {
		Payload json.RawMessage `json:"payload"`
		Seen    json.RawMessage `json:"seen"`
		Refs    json.RawMessage `json:"refs"`
	}
	if json.Unmarshal(data, &in) != nil || outbox.ValidatePayload(in.Payload) != nil || !seenOK(in.Seen) {
		return false
	}
	return in.Refs == nil || outbox.ValidatePayload(in.Refs) == nil
}

// seenOK: the encoded AppliedSignals has only "applied", a map from UUID to
// RFC3339 time, so the set cannot carry data either.
func seenOK(raw json.RawMessage) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return false
	}
	for name, v := range fields {
		if name != "applied" {
			return false
		}
		var applied map[string]string
		if json.Unmarshal(v, &applied) != nil {
			return false
		}
		for id, at := range applied {
			if _, err := uuid.Parse(id); err != nil {
				return false
			}
			if _, err := time.Parse(time.RFC3339Nano, at); err != nil {
				return false
			}
		}
	}
	return true
}

// checkedInputs holds the continue-as-new inputs NewContinueAsNewError
// checked in this run. ContinueAsNewError is plain data, so workflow code can
// replace its Input after the check, or build the error itself; the inbound
// guard accepts only an input that is one of these. A workflow runs on one
// goroutine at a time, so no lock is needed.
type checkedInputs struct{ inputs [][]checkedPayload }

// checkedPayload is a copy of a payload's metadata and data: the error's
// payloads are pointers that workflow code can change after the check.
type checkedPayload struct {
	metadata map[string][]byte
	data     []byte
}

func (c *checkedInputs) add(p *commonpb.Payloads) {
	if len(p.GetPayloads()) == 0 {
		return
	}
	in := make([]checkedPayload, 0, len(p.GetPayloads()))
	for _, q := range p.GetPayloads() {
		md := make(map[string][]byte, len(q.GetMetadata()))
		for k, v := range q.GetMetadata() {
			md[k] = bytes.Clone(v)
		}
		in = append(in, checkedPayload{metadata: md, data: bytes.Clone(q.GetData())})
	}
	c.inputs = append(c.inputs, in)
}

// has reports whether p was checked. An empty input (no arguments) passes.
func (c *checkedInputs) has(p *commonpb.Payloads) bool {
	if len(p.GetPayloads()) == 0 {
		return true
	}
	for _, in := range c.inputs {
		if sameInput(p.GetPayloads(), in) {
			return true
		}
	}
	return false
}

// sameInput compares metadata and data only. A payload has no other field the
// SDK sets; one set by workflow code makes the input differ, and denies.
func sameInput(ps []*commonpb.Payload, in []checkedPayload) bool {
	if len(ps) != len(in) {
		return false
	}
	for i, p := range ps {
		if len(p.GetExternalPayloads()) > 0 || !bytes.Equal(p.GetData(), in[i].data) ||
			!maps.EqualFunc(p.GetMetadata(), in[i].metadata, bytes.Equal) {
			return false
		}
	}
	return true
}
