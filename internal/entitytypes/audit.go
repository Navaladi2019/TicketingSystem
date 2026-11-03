package entitytypes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/secrets"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
)

// SecretInContentError is returned by Create and Publish when the schema
// carries a value where a secret belongs (ADR-004 R6, code
// config_secret_in_content). It names the pointer into the schema, never the
// value. Nothing is written once it is returned; the caller's transaction is
// unusable after it.
type SecretInContentError struct {
	// Path is a JSON pointer into the schema document.
	Path string
	// Reason is the fixed secrets.Reason vocabulary.
	Reason string
}

func (e *SecretInContentError) Error() string {
	return "entitytypes: config_secret_in_content: " + e.Reason + " at " + strconv.Quote(e.Path)
}

// SecretPaths returns the pointers, into schema, at which a value would be a
// secret: default, const, enum and examples of every top-level property marked
// x-nava-secret or writeOnly. metadata.Compile allows the marker nowhere else
// (no applicator, $defs, items or $ref), so no walk of the schema and no array
// wildcard is needed. A nested or array secret needs an ADR amendment first.
func SecretPaths(schema json.RawMessage) ([]patchwalk.Pointer, error) {
	var doc struct {
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil, fmt.Errorf("entitytypes: secret paths: %w", err)
	}
	var out []patchwalk.Pointer
	for name, p := range doc.Properties {
		secret, _ := p["x-nava-secret"].(bool)
		wo, _ := p["writeOnly"].(bool)
		if !secret && !wo {
			continue
		}
		for _, kw := range []string{"default", "const", "enum", "examples"} {
			out = append(out, patchwalk.FromSegments([]string{"properties", name, kw}))
		}
	}
	slices.Sort(out)
	return out, nil
}

// CheckSchemaSecrets refuses a schema that holds a value at a secret path (R6).
// It checks the whole schema, not the diff against the previous version: a
// publish that only adds the marker to a property with an existing default
// has a diff that never shows the default. It runs before any version row is
// written, and before metadata.Compile, so a value at a marked property is
// config_secret_in_content and not a generic invalid_definition.
func CheckSchemaSecrets(schema json.RawMessage) error {
	paths, err := SecretPaths(schema)
	if err != nil {
		return nil // not JSON of the expected shape: metadata.Compile reports it
	}
	if len(paths) == 0 {
		return nil
	}
	whole, err := jsondiff.Diff(nil, schema)
	if err != nil {
		return fmt.Errorf("entitytypes: diff schema: %w", err)
	}
	if err := secrets.Check(whole, paths, secrets.DomainConfig); err != nil {
		var sec *secrets.ErrSecretInDiff
		if errors.As(err, &sec) {
			return &SecretInContentError{Path: string(sec.Path), Reason: string(sec.Reason)}
		}
		return fmt.Errorf("entitytypes: check schema secrets: %w", err)
	}
	return nil
}

// schemaJSONB reads a stored schema back as jsonb text. The audit diff is
// built from these values, never from the request body, so diff_sha256 does
// not depend on the request's formatting (TICK-451).
func schemaJSONB(ctx context.Context, tx pgx.Tx, definitionID uuid.UUID, version int) (json.RawMessage, error) {
	var s string
	err := tx.QueryRow(ctx, `
SELECT schema::jsonb::text FROM entity_definition_versions WHERE definition_id = $1 AND version = $2`,
		definitionID, version).Scan(&s)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: read schema as jsonb: %w", err)
	}
	return json.RawMessage(s), nil
}

// auditVersion writes the config-domain entry of a version just inserted in
// tx: create for version 1, publish after. before is the previous version
// number, or 0 for the first.
func auditVersion(ctx context.Context, tx pgx.Tx, definitionID uuid.UUID, version int, schemaSum [sha256.Size]byte, by uuid.UUID) error {
	after, err := schemaJSONB(ctx, tx, definitionID, version)
	if err != nil {
		return err
	}
	var before json.RawMessage
	action := audit.ActionCreate
	var vBefore *int64
	if version > 1 {
		if before, err = schemaJSONB(ctx, tx, definitionID, version-1); err != nil {
			return err
		}
		action = audit.ActionPublish
		vb := int64(version - 1)
		vBefore = &vb
	}
	diff, err := jsondiff.Diff(before, after)
	if err != nil {
		return fmt.Errorf("entitytypes: diff schema: %w", err)
	}
	paths, err := SecretPaths(after)
	if err != nil {
		return err
	}
	va := int64(version)
	e := audit.Entry{
		Domain:        audit.DomainConfig,
		Action:        action,
		SubjectType:   audit.SubjectEntityDefinition,
		SubjectID:     definitionID,
		VersionBefore: vBefore,
		VersionAfter:  &va,
		Actor:         audit.Actor{PrincipalID: &by},
		Source:        audit.SourceUser,
		ContentSHA256: &schemaSum,
		Diff:          diff,
		SecretPaths:   paths,
	}
	if id := requestid.FromContext(ctx); id != "" {
		e.RequestID = &id
	}
	if _, err := audit.Append(ctx, tx, e); err != nil {
		var sec *secrets.ErrSecretInDiff
		if errors.As(err, &sec) {
			return &SecretInContentError{Path: string(sec.Path), Reason: string(sec.Reason)}
		}
		return fmt.Errorf("entitytypes: audit version: %w", err)
	}
	return nil
}
