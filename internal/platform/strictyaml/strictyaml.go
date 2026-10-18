// Package strictyaml decodes one YAML document and rejects anything the
// target type does not name. Registry files (api/errors.yaml) are read through
// it so a misspelt key or a second document fails loudly instead of being
// ignored.
package strictyaml

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// Decode decodes data into v. Unknown fields are an error, and so is a second
// document after the first (a second Decode must return io.EOF).
func Decode(data []byte, v any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode yaml: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("decode yaml: more than one document")
	}
	return nil
}
