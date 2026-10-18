// Package yamlstrict is the one loader for the platform's checked-in YAML
// manifests (db/grants.yaml, api/errors.yaml). A manifest that does not say
// exactly what the struct expects is rejected, never half-read: an unknown
// (typo'd) key, a duplicate key and a second document are all errors.
package yamlstrict

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// Decode reads exactly one YAML document from r into v.
func Decode(r io.Reader, v any) error {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("yaml: empty document")
		}
		return err
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return fmt.Errorf("yaml: after the first document: %w", err)
		}
		return errors.New("yaml: more than one document")
	}
	return nil
}

// DecodeBytes is Decode over b.
func DecodeBytes(b []byte, v any) error { return Decode(bytes.NewReader(b), v) }
