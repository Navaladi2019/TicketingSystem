package entitytypes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/secrets"
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

// secretDataKeywords are the keywords that hold a value a schema author
// supplies. A secret-marked schema may not set any of them.
var secretDataKeywords = []string{"default", "const", "enum", "examples"}

// maxRefHops bounds how far a local $ref chain is followed.
const maxRefHops = 8

// SecretPaths returns the pointers, into schema, at which a value would be a
// secret: the data keywords of every subschema marked x-nava-secret or
// writeOnly, directly or through a local $ref. Pointers match token by token
// in secrets.Check. Keywords whose subschemas sit in arrays (allOf, anyOf,
// oneOf, prefixItems) get their concrete index, so a marker under one of them
// is covered at the index it has in this document.
func SecretPaths(schema json.RawMessage) ([]patchwalk.Pointer, error) {
	var doc any
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil, fmt.Errorf("entitytypes: secret paths: %w", err)
	}
	var out []patchwalk.Pointer
	walkSecrets(doc, doc, nil, &out)
	slices.Sort(out)
	return slices.Compact(out), nil
}

func isSecretNode(root any, n map[string]any, hops int) bool {
	for _, k := range []string{"x-nava-secret", "writeOnly"} {
		if v, _ := n[k].(bool); v {
			return true
		}
	}
	ref, _ := n["$ref"].(string)
	if hops >= maxRefHops || !strings.HasPrefix(ref, "#/") {
		return false
	}
	cur := root
	for _, tok := range strings.Split(ref[2:], "/") {
		tok = strings.ReplaceAll(strings.ReplaceAll(tok, "~1", "/"), "~0", "~")
		switch c := cur.(type) {
		case map[string]any:
			cur = c[tok]
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= len(c) {
				return false
			}
			cur = c[i]
		default:
			return false
		}
	}
	m, ok := cur.(map[string]any)
	return ok && isSecretNode(root, m, hops+1)
}

func walkSecrets(root, node any, at []string, out *[]patchwalk.Pointer) {
	n, ok := node.(map[string]any)
	if !ok {
		return
	}
	if isSecretNode(root, n, 0) {
		for _, kw := range secretDataKeywords {
			*out = append(*out, patchwalk.FromSegments(append(slices.Clone(at), kw)))
		}
	}
	for kw, v := range n {
		switch kw {
		case "additionalProperties", "contains", "else", "if", "items", "not", "propertyNames",
			"then", "unevaluatedItems", "unevaluatedProperties", "contentSchema":
			walkSecrets(root, v, append(slices.Clone(at), kw), out)
		case "allOf", "anyOf", "oneOf", "prefixItems":
			for i, e := range asSlice(v) {
				walkSecrets(root, e, append(slices.Clone(at), kw, strconv.Itoa(i)), out)
			}
		case "$defs", "dependentSchemas", "patternProperties", "properties":
			if m, ok := v.(map[string]any); ok {
				for name, e := range m {
					walkSecrets(root, e, append(slices.Clone(at), kw, name), out)
				}
			}
		}
	}
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
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
func auditVersion(ctx context.Context, tx pgx.Tx, definitionID uuid.UUID, version int, schemaSum [sha256.Size]byte, by uuid.UUID, requestID string) error {
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
	if requestID != "" {
		e.RequestID = &requestID
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
