package chain

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// MarshalJSONL returns v as one JSON line ending in "\n", the bytes a
// json.Encoder with SetEscapeHTML(false) writes, except that every Envelope
// inside v is written as its canonical bytes (CanonicalEnvelope) verbatim.
//
// encoding/json may re-escape the bytes a MarshalJSON method returns: Go 1.27
// turns a raw U+2028 or U+2029 there into \u2028 or \u2029, and Go 1.26 did
// not. RFC 8785 keeps both raw, and the verifier rejects any other spelling
// (decString), so an export or vectors file must never let the standard
// library rewrite the envelope (TICK-970). Everything that is not an
// Envelope still goes through encoding/json, so the framing members keep
// their usual encoding.
//
// v may nest structs, pointers, interfaces, slices, arrays and string-keyed
// maps around its envelopes. Struct fields take their json tag name; a tag
// option (omitempty, string) or an embedded field is refused rather than
// half-supported. A value with its own MarshalJSON or MarshalText is handed
// to encoding/json whole.
func MarshalJSONL(v any) ([]byte, error) {
	b, err := appendJSON(nil, reflect.ValueOf(v))
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// MarshalIndentJSONL is MarshalJSONL indented like a json.Encoder with
// SetIndent(prefix, indent). json.Indent only moves whitespace, so the
// envelope bytes stay as they are.
func MarshalIndentJSONL(v any, prefix, indent string) ([]byte, error) {
	b, err := appendJSON(nil, reflect.ValueOf(v))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, b, prefix, indent); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

//nolint:gochecknoglobals // reflect types, set once
var (
	envelopeType      = reflect.TypeFor[Envelope]()
	jsonMarshalerType = reflect.TypeFor[json.Marshaler]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
)

var errUnsupportedJSON = errors.New("chain: MarshalJSONL: unsupported")

func appendJSON(dst []byte, v reflect.Value) ([]byte, error) {
	if !v.IsValid() {
		return append(dst, "null"...), nil
	}
	t := v.Type()
	if t == envelopeType {
		canon, err := CanonicalEnvelope(v.Interface().(Envelope)) //nolint:forcetypeassert // checked by type
		if err != nil {
			return nil, err
		}
		return append(dst, canon...), nil
	}
	switch t.Kind() { //nolint:exhaustive // every other kind is a plain value for encoding/json
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return append(dst, "null"...), nil
		}
		if t.Kind() == reflect.Pointer && hasOwnMarshaler(t) && !hasOwnMarshaler(t.Elem()) {
			return appendStd(dst, v)
		}
		return appendJSON(dst, v.Elem())
	}
	if hasOwnMarshaler(t) {
		return appendStd(dst, v)
	}
	switch t.Kind() { //nolint:exhaustive // see above
	case reflect.Struct:
		return appendStruct(dst, v)
	case reflect.Map:
		return appendMap(dst, v)
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 || v.IsNil() {
			return appendStd(dst, v) // base64 or null, as encoding/json does
		}
		return appendList(dst, v)
	case reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return appendStd(dst, v)
		}
		return appendList(dst, v)
	default:
		return appendStd(dst, v)
	}
}

func hasOwnMarshaler(t reflect.Type) bool {
	return t.Implements(jsonMarshalerType) || t.Implements(textMarshalerType)
}

// appendStd encodes v with encoding/json, HTML escaping off.
func appendStd(dst []byte, v reflect.Value) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v.Interface()); err != nil {
		return nil, err
	}
	return append(dst, bytes.TrimSuffix(b.Bytes(), []byte("\n"))...), nil
}

func appendStruct(dst []byte, v reflect.Value) ([]byte, error) {
	t := v.Type()
	dst = append(dst, '{')
	first := true
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" && opts == "" {
			continue
		}
		if f.Anonymous || opts != "" {
			return nil, fmt.Errorf("%w: %s.%s: embedded field or json tag option", errUnsupportedJSON, t, f.Name)
		}
		if name == "" {
			name = f.Name
		}
		if !first {
			dst = append(dst, ',')
		}
		first = false
		var err error
		if dst, err = appendKey(dst, name); err != nil {
			return nil, err
		}
		if dst, err = appendJSON(dst, v.Field(i)); err != nil {
			return nil, err
		}
	}
	return append(dst, '}'), nil
}

func appendMap(dst []byte, v reflect.Value) ([]byte, error) {
	if v.IsNil() {
		return append(dst, "null"...), nil
	}
	if v.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("%w: map key type %s", errUnsupportedJSON, v.Type().Key())
	}
	keys := v.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(a.String(), b.String()) })
	dst = append(dst, '{')
	for i, k := range keys {
		if i > 0 {
			dst = append(dst, ',')
		}
		var err error
		if dst, err = appendKey(dst, k.String()); err != nil {
			return nil, err
		}
		if dst, err = appendJSON(dst, v.MapIndex(k)); err != nil {
			return nil, err
		}
	}
	return append(dst, '}'), nil
}

func appendList(dst []byte, v reflect.Value) ([]byte, error) {
	dst = append(dst, '[')
	for i := range v.Len() {
		if i > 0 {
			dst = append(dst, ',')
		}
		var err error
		if dst, err = appendJSON(dst, v.Index(i)); err != nil {
			return nil, err
		}
	}
	return append(dst, ']'), nil
}

func appendKey(dst []byte, k string) ([]byte, error) {
	dst, err := appendStd(dst, reflect.ValueOf(k))
	if err != nil {
		return nil, err
	}
	return append(dst, ':'), nil
}
