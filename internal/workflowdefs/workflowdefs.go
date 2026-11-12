// Package workflowdefs publishes and reads workflow definitions over the P3-40
// tables (PLAN-003 D-3-12, D-3-15; S11b contract D-3-29; TICK-710).
//
// Store is the API side: Publish and Get run in the caller's
// tenancy.InTenantTx, so row-level security scopes every query to the
// caller's tenant and a key of another tenant reads as ErrNotFound. Reader is
// the worker side: interp.DefinitionStore over the same tables, in
// workerguard.InTenantTx.
//
// Publish validates the body with def.Parse, checks every update_record.set
// against the trigger entity's current schema, draws the step ids server-side
// with def.AssignStepIDs (a client never supplies one) and writes one
// immutable version row. It locks the definition row, so concurrent
// publishes queue and number 1..N with no gap or duplicate.
package workflowdefs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
)

var (
	// ErrNotFound means the tenant has no such workflow definition.
	ErrNotFound = errors.New("workflowdefs: not found")
	// ErrNoPublisher means a publish named no principal. published_by is
	// nullable in the table for seeded definitions, so this is the guard.
	ErrNoPublisher = errors.New("workflowdefs: publish needs a principal")
)

// KeyPattern is the shape of a definition key (workflow_definitions_key_check).
var KeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,40}$`)

// stepKeyPattern is the shape of a step key (def.validKey).
var stepKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Rules of an InvalidError field. A fixed vocabulary: no rule carries input.
const (
	RuleInvalid       = "invalid"        // def.Parse refused the position
	RuleUnknownEntity = "unknown_entity" // trigger.entity is not an entity type of the tenant
	RuleUnknownField  = "unknown_field"  // update_record.set names a field the schema lacks
	RuleEntityChange  = "entity_change"  // a new version names another trigger entity
	RuleKey           = "key"            // the definition key has the wrong shape
)

// FieldError is one failing position. Pointer is an RFC 6901 pointer into the
// submitted body in which a step is named by its key, not its position, when
// the step has a valid key ("/steps/approve/set/priority"). It never carries a
// value.
type FieldError struct {
	Pointer string
	Rule    string
}

// InvalidError means the body is not a publishable definition. Nothing is
// written, and the caller's transaction is unusable after it.
type InvalidError struct{ Fields []FieldError }

func (e *InvalidError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = strconv.Quote(f.Pointer) + " " + f.Rule
	}
	return "workflowdefs: invalid definition at " + strings.Join(parts, ", ")
}

// Store reads and publishes workflow definitions. It holds no state.
type Store struct{}

// PublishInput is one publish.
type PublishInput struct {
	Key         string
	PublishedBy uuid.UUID
	// Body is the request body: a workflow definition in format v1. It is
	// stored as these exact bytes.
	Body []byte
}

// Version is one published version.
type Version struct {
	Key string
	// Entity is the key of the trigger entity type.
	Entity  string
	Version int
	// Current is true while the definition's current_version is Version.
	Current bool
	// ID is the version id a workflow is pinned to (D-3-12).
	ID uuid.UUID
	// Body is the exact bytes that were published.
	Body        json.RawMessage
	BodySHA256  [sha256.Size]byte
	StepIDs     map[string]uuid.UUID
	PublishedAt time.Time
}

// SHA256Hex is BodySHA256 in lowercase hex.
func (v *Version) SHA256Hex() string { return hex.EncodeToString(v.BodySHA256[:]) }

const versionSelect = `
SELECT d.key, e.key, v.id, v.version, d.current_version, v.body::text, v.body_sha256, v.step_ids, v.published_at
FROM workflow_definitions d
JOIN workflow_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id
JOIN entity_definitions e ON e.tenant_id = d.tenant_id AND e.id = d.entity_definition_id`

func scanVersion(row pgx.Row) (*Version, error) {
	var (
		v    Version
		body string
		sum  []byte
		ids  map[string]uuid.UUID
		cur  int
	)
	err := row.Scan(&v.Key, &v.Entity, &v.ID, &v.Version, &cur, &body, &sum, &ids, &v.PublishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("workflowdefs: read version: %w", err)
	}
	if len(sum) != sha256.Size {
		return nil, errors.New("workflowdefs: stored body_sha256 has the wrong length")
	}
	v.Body = json.RawMessage(body)
	copy(v.BodySHA256[:], sum)
	v.StepIDs = ids
	v.Current = cur == v.Version
	return &v, nil
}

// GetCurrent returns the current version of the definition, or ErrNotFound.
func (Store) GetCurrent(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string) (*Version, error) {
	if !KeyPattern.MatchString(key) {
		return nil, ErrNotFound // cannot exist, and must not reach the database
	}
	return scanVersion(tx.QueryRow(ctx, versionSelect+` WHERE d.tenant_id = $1 AND d.key = $2 AND v.version = d.current_version`, tenantID, key))
}

// Publish writes the next version of the definition named key and makes it
// current, creating the definition with version 1 when the tenant has none.
// It runs in the caller's transaction and locks the definition row, so
// concurrent publishes queue and number 1..N with no gap or duplicate.
//
// It returns ErrNoPublisher, or *InvalidError when the key or the body is not
// publishable; nothing is written then, and the caller's transaction is
// unusable after any error. Steps and signals keep the id the previous
// version gave their key and a new key draws a new one (def.AssignStepIDs).
func (Store) Publish(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, in PublishInput) (*Version, error) {
	if in.PublishedBy == uuid.Nil {
		return nil, ErrNoPublisher
	}
	if !KeyPattern.MatchString(in.Key) {
		return nil, &InvalidError{Fields: []FieldError{{Pointer: "/key", Rule: RuleKey}}}
	}
	d, err := def.Parse(in.Body)
	if err != nil {
		var inv *def.ErrInvalidDefinition
		if errors.As(err, &inv) {
			return nil, &InvalidError{Fields: parseFields(in.Body, inv.Paths)}
		}
		return nil, err
	}

	entityID, err := entityID(ctx, tx, d.Trigger.Entity)
	if err != nil {
		return nil, err
	}
	compiled, _, err := entitytypes.Store{}.Current(ctx, tx, d.Trigger.Entity)
	if err != nil {
		if errors.Is(err, entitytypes.ErrNotFound) {
			return nil, &InvalidError{Fields: []FieldError{{Pointer: "/trigger/entity", Rule: RuleUnknownEntity}}}
		}
		return nil, fmt.Errorf("workflowdefs: read trigger entity: %w", err)
	}
	if bad := checkSets(d, compiled.PropertyTypes()); len(bad) > 0 {
		return nil, &InvalidError{Fields: bad}
	}

	// Create the definition if it is new. A concurrent first publish of the
	// same key waits on the unique index, does nothing, and queues on the row
	// lock below with every later publish.
	if _, err := tx.Exec(ctx, `
INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version, created_by)
VALUES ($1, $2, $3, $4, 1, $5)
ON CONFLICT (tenant_id, key) DO NOTHING`, tenantID, uuid.New(), in.Key, entityID, in.PublishedBy); err != nil {
		return nil, fmt.Errorf("workflowdefs: create definition: %w", err)
	}
	var (
		defID    uuid.UUID
		defEntry uuid.UUID
	)
	if err := tx.QueryRow(ctx, `
SELECT id, entity_definition_id FROM workflow_definitions WHERE tenant_id = $1 AND key = $2 FOR UPDATE`, tenantID, in.Key).Scan(&defID, &defEntry); err != nil {
		return nil, fmt.Errorf("workflowdefs: lock definition: %w", err)
	}
	if defEntry != entityID {
		// A definition belongs to one entity type for life (its trigger index
		// and the sink read it so); a new type is a new definition.
		return nil, &InvalidError{Fields: []FieldError{{Pointer: "/trigger/entity", Rule: RuleEntityChange}}}
	}

	// The next number follows the highest version, which the row lock makes
	// stable until this transaction ends.
	var (
		number  int
		prevIDs map[string]uuid.UUID
		prevRaw []byte
	)
	if err := tx.QueryRow(ctx, `
SELECT coalesce(max(version), 0) + 1 FROM workflow_definition_versions WHERE definition_id = $1`, defID).Scan(&number); err != nil {
		return nil, fmt.Errorf("workflowdefs: next version: %w", err)
	}
	if number > 1 {
		var prevBody string
		if err := tx.QueryRow(ctx, `
SELECT step_ids, body::jsonb::text FROM workflow_definition_versions WHERE definition_id = $1 AND version = $2`,
			defID, number-1).Scan(&prevIDs, &prevBody); err != nil {
			return nil, fmt.Errorf("workflowdefs: read previous version: %w", err)
		}
		prevRaw = []byte(prevBody)
	}
	stepIDs := def.AssignStepIDs(d, prevIDs)
	ids, err := json.Marshal(stepIDs)
	if err != nil {
		return nil, fmt.Errorf("workflowdefs: encode step ids: %w", err)
	}
	sum := sha256.Sum256(in.Body)
	if _, err := tx.Exec(ctx, `
INSERT INTO workflow_definition_versions
  (tenant_id, id, definition_id, version, body, body_sha256, step_ids, trigger_on, trigger_entity_definition_id, published_by)
VALUES ($1, $2, $3, $4, $5::json, $6, $7::jsonb, $8, $9, $10)`,
		tenantID, uuid.New(), defID, number, string(in.Body), sum[:], string(ids), d.Trigger.On, entityID, in.PublishedBy); err != nil {
		return nil, fmt.Errorf("workflowdefs: insert version: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE workflow_definitions SET current_version = $1, updated_at = now() WHERE id = $2`, number, defID); err != nil {
		return nil, fmt.Errorf("workflowdefs: set current version: %w", err)
	}
	if err := auditVersion(ctx, tx, defID, number, prevRaw, sum, in.PublishedBy); err != nil {
		return nil, err
	}
	return scanVersion(tx.QueryRow(ctx, versionSelect+` WHERE d.tenant_id = $1 AND d.key = $2 AND v.version = $3`, tenantID, in.Key, number))
}

// entityID is the id of the tenant's entity type key; an unknown key is a 422
// at /trigger/entity.
func entityID(ctx context.Context, tx pgx.Tx, key string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE key = $1`, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, &InvalidError{Fields: []FieldError{{Pointer: "/trigger/entity", Rule: RuleUnknownEntity}}}
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("workflowdefs: read trigger entity id: %w", err)
	}
	return id, nil
}

// checkSets reports every update_record.set field the trigger entity's schema
// has no top-level property for. Positions name the step by its key.
func checkSets(d *def.Definition, props map[string]string) []FieldError {
	var bad []FieldError
	for i := range d.Steps {
		s := &d.Steps[i]
		if s.Type != def.TypeUpdateRecord {
			continue
		}
		fields := make([]string, 0, len(s.Set))
		for f := range s.Set {
			fields = append(fields, f)
		}
		slices.Sort(fields)
		for _, f := range fields {
			if _, ok := props[f]; !ok {
				bad = append(bad, FieldError{Pointer: "/steps/" + s.Key + "/set/" + escape(f), Rule: RuleUnknownField})
			}
		}
	}
	return bad
}

// parseFields turns def.Parse's index pointers into FieldErrors, naming each
// step by its key when the submitted step has a well-formed one.
func parseFields(body []byte, paths []string) []FieldError {
	var doc struct {
		Steps []struct {
			Key any `json:"key"`
		} `json:"steps"`
	}
	_ = json.Unmarshal(body, &doc) // Parse accepted nothing here, so a failure leaves no keys
	out := make([]FieldError, len(paths))
	for i, p := range paths {
		out[i] = FieldError{Pointer: p, Rule: RuleInvalid}
		rest, ok := strings.CutPrefix(p, "/steps/")
		if !ok {
			continue
		}
		idx, tail, _ := strings.Cut(rest, "/")
		n, err := strconv.Atoi(idx)
		if err != nil || n < 0 || n >= len(doc.Steps) {
			continue
		}
		if k, ok := doc.Steps[n].Key.(string); ok && stepKeyPattern.MatchString(k) {
			out[i].Pointer = "/steps/" + k
			if tail != "" {
				out[i].Pointer += "/" + tail
			}
		}
	}
	return out
}

// escape is RFC 6901 escaping of one token.
func escape(token string) string {
	return strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
}

// auditVersion writes the config-domain entry of the version just inserted in
// tx: create for version 1, publish after. Its diff is built from the stored
// jsonb forms, never from the request bytes.
func auditVersion(ctx context.Context, tx pgx.Tx, defID uuid.UUID, version int, prev []byte, sum [sha256.Size]byte, by uuid.UUID) error {
	var after string
	if err := tx.QueryRow(ctx, `
SELECT body::jsonb::text FROM workflow_definition_versions WHERE definition_id = $1 AND version = $2`,
		defID, version).Scan(&after); err != nil {
		return fmt.Errorf("workflowdefs: read body as jsonb: %w", err)
	}
	diff, err := jsondiff.Diff(prev, json.RawMessage(after))
	if err != nil {
		return fmt.Errorf("workflowdefs: diff body: %w", err)
	}
	action := audit.ActionCreate
	var vBefore *int64
	if version > 1 {
		action = audit.ActionPublish
		vb := int64(version - 1)
		vBefore = &vb
	}
	va := int64(version)
	e := audit.Entry{
		Domain:        audit.DomainConfig,
		Action:        action,
		SubjectType:   audit.SubjectWorkflowDefinition,
		SubjectID:     defID,
		VersionBefore: vBefore,
		VersionAfter:  &va,
		Actor:         audit.Actor{PrincipalID: &by},
		Source:        audit.SourceUser,
		ContentSHA256: &sum,
		Diff:          diff,
	}
	if id := requestid.FromContext(ctx); id != "" {
		e.RequestID = &id
	}
	if _, err := audit.Append(ctx, tx, e); err != nil {
		return fmt.Errorf("workflowdefs: audit: %w", err)
	}
	return nil
}
