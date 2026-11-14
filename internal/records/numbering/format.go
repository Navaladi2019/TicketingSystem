package numbering

import (
	"bytes"
	_ "embed" // format.schema.json is the shape contract
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // time zones validate the same on every host, with or without a zoneinfo database

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// schemaURL is the $id of format.schema.json. It is never fetched.
const schemaURL = "https://nava.local/schemas/number-format/v1/schema.json"

//go:embed format.schema.json
var schemaJSON []byte

// Format is a number format document as published (ADR-005 §3.1):
// format.schema.json is its shape.
type Format struct {
	// Template is the §3.1 template, for example {prefix}-{yyyy}{MM}{seq:4}.
	Template string `json:"template"`
	// Overflow is grow or refuse (§3.5).
	Overflow Overflow `json:"overflow"`
	// TimeZone is an IANA name; "" means the tenant's setting (§3.6).
	TimeZone string `json:"time_zone,omitempty"`
}

// ParseFormat validates b against format.schema.json, then compiles it with
// CompileFormat. A document the schema refuses is a *FormatError with
// CodeInvalid naming the failing JSON pointer.
func ParseFormat(b []byte) (Compiled, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return Compiled{}, invalid("/", -1, "the document is not JSON")
	}
	sch, err := compileSchema()
	if err != nil {
		return Compiled{}, err
	}
	if err := sch.Validate(doc); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			return Compiled{}, invalid(firstLeaf(ve), -1, "the document does not match the number format schema")
		}
		return Compiled{}, fmt.Errorf("numbering: validate: %w", err)
	}
	var f Format
	if err := json.Unmarshal(b, &f); err != nil {
		// The schema accepted the document, so this is a bug in Format.
		return Compiled{}, fmt.Errorf("numbering: decode: %w", err)
	}
	return CompileFormat(f)
}

// CompileFormat compiles f (ADR-005 §3.1, §3.5, §3.6). It refuses, with a
// *FormatError:
//
//   - CodePrefixRequired: a template that does not start with {prefix}, or
//     whose next token produces letters ({prefix}, {field:…}, or a literal
//     starting with A-Z), at that token's position;
//   - CodeInvalid: an unknown or {field:…} token, a character outside the
//     literal set, a literal run over 16 characters, no or a second
//     {seq:N}, N outside 1..12, a second {prefix}, grow when a digit touches
//     {seq:N}, a rendering that can exceed 64 characters, an overflow other
//     than grow or refuse, or a time zone time.LoadLocation does not know.
//
// The default ({prefix}{seq:7}, grow) is epoch 0; every other format is
// EpochUnassigned until the publish takes its epoch (S18).
func CompileFormat(f Format) (Compiled, error) {
	ct, ferr := compileTemplate(f.Template, f.Overflow)
	if ferr != nil {
		return Compiled{}, ferr
	}
	if err := checkTimeZone(f.TimeZone); err != nil {
		return Compiled{}, err
	}
	c := fromTemplate(f.Template, f.Overflow, ct)
	c.TimeZone = f.TimeZone
	if c.Epoch == 0 && c.TimeZone == "" {
		// The default has no date part; its zone is pinned to UTC (P6-02).
		c.TimeZone = Default().TimeZone
	}
	return c, nil
}

// checkTimeZone accepts "" (the tenant's setting) or an IANA name.
// "Local" is refused: it is the server's zone, not a tenant's.
func checkTimeZone(name string) error {
	if name == "" {
		return nil
	}
	if name == "Local" || strings.TrimSpace(name) != name {
		return invalid("time_zone", -1, "the time zone is not an IANA name")
	}
	if _, err := time.LoadLocation(name); err != nil {
		return invalid("time_zone", -1, "the time zone is not an IANA name")
	}
	return nil
}

// compileSchema compiles the embedded schema with a loader that refuses every
// URL (D-3-18), as internal/workflow/def does. A new Compiler per call keeps
// the package free of shared state; publishing a format is rare.
func compileSchema() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("numbering: embedded schema: %w", err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.UseLoader(denyAll{})
	if err := c.AddResource(schemaURL, doc); err != nil {
		return nil, fmt.Errorf("numbering: embedded schema: %w", err)
	}
	sch, err := c.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("numbering: embedded schema: %w", err)
	}
	return sch, nil
}

type denyAll struct{}

func (denyAll) Load(url string) (any, error) {
	return nil, fmt.Errorf("numbering: schema loading is disabled (%s)", url)
}

// firstLeaf is the JSON pointer of the first leaf validation error.
func firstLeaf(ve *jsonschema.ValidationError) string {
	for len(ve.Causes) > 0 {
		ve = ve.Causes[0]
	}
	base := "/" + strings.Join(ve.InstanceLocation, "/")
	switch k := ve.ErrorKind.(type) {
	case *kind.Required:
		return strings.TrimSuffix(base, "/") + "/" + k.Missing[0]
	case *kind.AdditionalProperties:
		return strings.TrimSuffix(base, "/") + "/" + k.Properties[0]
	}
	return base
}
