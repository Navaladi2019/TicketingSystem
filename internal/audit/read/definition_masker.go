package read

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

// ErrNoDefinitionVersions means an entry names an entity definition and none
// of the definition's version rows is visible to the reader, so the secret
// properties cannot be known. The masker fails closed (ADR-004-A1 A1-4): the
// read is refused, never served unmasked.
var ErrNoDefinitionVersions = errors.New("audit/read: entry names a definition with no visible version")

// NoDefinitionVersionsError carries the definition id of an
// ErrNoDefinitionVersions for the caller's log. It never carries entry data.
type NoDefinitionVersionsError struct {
	DefinitionID uuid.UUID
}

func (e *NoDefinitionVersionsError) Error() string {
	return fmt.Sprintf("%v: definition %s", ErrNoDefinitionVersions, e.DefinitionID)
}

func (e *NoDefinitionVersionsError) Unwrap() error { return ErrNoDefinitionVersions }

// DefinitionMasker is the production Masker (PLAN-005 D-5-5): for an entry
// about an entity definition's records or versions it returns the pointers of
// the definition's x-nava-secret and writeOnly properties, from every version
// the definition has published, because an old entry was written under an
// older schema. Secret markers are top-level only (decision 2026-11-03), so
// each pointer is "/<property>". S10 adds per-field masks to this port.
type DefinitionMasker struct{}

// MaskedPaths reads the schemas of the definition's versions inside tx, so
// row-level security scopes them to the tenant. An entry with no definition
// has no masked path. A definition with no visible version row is an error
// (ErrNoDefinitionVersions), because an empty answer would mean "nothing is
// secret".
func (DefinitionMasker) MaskedPaths(ctx context.Context, tx pgx.Tx, s Subject) ([]patchwalk.Pointer, error) {
	if s.EntityDefinitionID == nil {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `SELECT schema::text FROM entity_definition_versions WHERE definition_id = $1`, *s.EntityDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("read definition schemas: %w", err)
	}
	defer rows.Close()
	var names []string
	versions := 0
	for rows.Next() {
		versions++
		var schema []byte
		if err := rows.Scan(&schema); err != nil {
			return nil, fmt.Errorf("scan definition schema: %w", err)
		}
		var doc struct {
			Properties map[string]struct {
				Secret    bool `json:"x-nava-secret"`
				WriteOnly bool `json:"writeOnly"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(schema, &doc); err != nil {
			return nil, fmt.Errorf("decode definition schema: %w", err)
		}
		for name, p := range doc.Properties {
			if p.Secret || p.WriteOnly {
				names = append(names, name)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read definition schemas: %w", err)
	}
	if versions == 0 {
		return nil, &NoDefinitionVersionsError{DefinitionID: *s.EntityDefinitionID}
	}
	slices.Sort(names)
	names = slices.Compact(names)
	out := make([]patchwalk.Pointer, len(names))
	for i, n := range names {
		out[i] = patchwalk.FromSegments([]string{n})
	}
	return out, nil
}
