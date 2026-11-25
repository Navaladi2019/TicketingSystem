package authz

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The ACL publish path (P3-35): the one writer of acl_versions that serves
// the API. Every insert writes its audit entry in the same transaction
// (ADR-004 CI-A3: acl_versions has the audit_commit_check trigger).

// ErrNoEntityType means the transaction's tenant has no entity type with the
// key. Another tenant's key is the same error.
var ErrNoEntityType = errors.New("authz: no such entity type")

// ConflictError means the version the caller based its publish on is not the
// newest one. Current is the newest version, 0 when none is published.
type ConflictError struct{ Current int }

func (e *ConflictError) Error() string {
	return fmt.Sprintf("authz: acl version conflict, current is %d", e.Current)
}

// Published is a stored ACL version.
type Published struct {
	Version int
	// Document is the rules as a format-2 document: a stored format-1 array
	// is converted on read (D12).
	Document Document
	// RulesJSON and MasksJSON are the stored values, as read.
	RulesJSON, MasksJSON json.RawMessage
	Masks                []Mask
	PublishedBy          *uuid.UUID
	PublishedAt          time.Time
	DefinitionID         uuid.UUID
}

const publishedSQL = `
SELECT a.version, a.rules, a.masks, a.published_by, a.published_at
FROM acl_versions a
WHERE a.tenant_id = $1 AND a.entity_definition_id = $2
ORDER BY a.version DESC
LIMIT 1`

const definitionSQL = `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`

// Newest returns the entity type's newest ACL version, nil when none is
// published, and ErrNoEntityType when the tenant has no such type. It reads in
// tx, under the transaction's tenant.
func (PolicyStore) Newest(ctx context.Context, tx pgx.Tx, entityKey string) (*Published, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	var def uuid.UUID
	if err := tx.QueryRow(ctx, definitionSQL, tenant, entityKey).Scan(&def); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEntityType
	} else if err != nil {
		return nil, fmt.Errorf("authz: read entity type: %w", err)
	}
	return newest(ctx, tx, tenant, def)
}

func newest(ctx context.Context, tx pgx.Tx, tenant, def uuid.UUID) (*Published, error) {
	var (
		p            = Published{DefinitionID: def}
		rules, masks []byte
	)
	err := tx.QueryRow(ctx, publishedSQL, tenant, def).Scan(&p.Version, &rules, &masks, &p.PublishedBy, &p.PublishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // nil is "none published"
	}
	if err != nil {
		return nil, fmt.Errorf("authz: read acl: %w", err)
	}
	if p.Document, err = Decode(rules); err != nil {
		return nil, fmt.Errorf("authz: decode acl %d rules: %w", p.Version, err)
	}
	p.RulesJSON, p.MasksJSON = rules, masks
	if err := decodeStrict(masks, &p.Masks); err != nil {
		return nil, fmt.Errorf("authz: decode acl %d masks: %w", p.Version, err)
	}
	return &p, nil
}

const lockDefinitionSQL = `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2 FOR UPDATE`

const insertACLSQL = `
INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks, published_by)
VALUES ($1, $2, $3, $4::text::jsonb, $5::text::jsonb, $6)
RETURNING published_at`

// Publisher identifies who publishes: a principal (the API) or a named system
// actor (onboarding).
type Publisher struct {
	Principal *uuid.UUID
	System    string
	Source    audit.Source
}

// Publish inserts rules and masks as version expected+1 of the entity type's
// ACL, and the audit entry for it, in tx.
//
// TEST SEAM: it skips Prepare and every D4 check and stores the rules as
// format 1, as the S10 tests and the D12 read tests need. No production path
// may call it: the publish route uses PublishPrepared, and onboarding uses
// PublishIfNone, which runs the checks. It takes the definition's row lock, so two publishes
// serialize, and returns *ConflictError when expected is not the newest
// version. expected 0 is the first publish.
func (PolicyStore) Publish(ctx context.Context, tx pgx.Tx, entityKey string, expected int, by Publisher, rules []Rule, masks []Mask) (*Published, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	var def uuid.UUID
	if err := tx.QueryRow(ctx, lockDefinitionSQL, tenant, entityKey).Scan(&def); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEntityType
	} else if err != nil {
		return nil, fmt.Errorf("authz: lock entity type: %w", err)
	}
	tenancy.Forget(tx, policyKey{entityKey})
	return publishLocked(ctx, tx, tenant, def, expected, by, rules, masks)
}

// PublishIfNone publishes version 1 for the entity type unless it already has
// a version, and reports whether it did. It is the idempotent step of the
// default ACL (onboarding and the backfill).
func (PolicyStore) PublishIfNone(ctx context.Context, tx pgx.Tx, entityKey string, by Publisher, rules []Rule, masks []Mask) (bool, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return false, fmt.Errorf("authz: %w", err)
	}
	var def uuid.UUID
	if err := tx.QueryRow(ctx, lockDefinitionSQL, tenant, entityKey).Scan(&def); errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNoEntityType
	} else if err != nil {
		return false, fmt.Errorf("authz: lock entity type: %w", err)
	}
	tenancy.Forget(tx, policyKey{entityKey})
	// The route's own check, so nothing is stored that the route would
	// refuse; the policy is stored as format 2.
	body, err := json.Marshal(struct {
		Rules []Rule `json:"rules"`
		Masks []Mask `json:"masks"`
	}{nonNilRules(rules), nonNilMasks(masks)})
	if err != nil {
		return false, fmt.Errorf("authz: encode policy: %w", err)
	}
	prep, problems := Prepare(body, DefaultMaxDocumentBytes)
	if len(problems) > 0 {
		return false, &InvalidPolicyError{Problems: problems}
	}
	// No caller publishes masks here, and the field check against the schema
	// is the route's: refuse them rather than store what the route would
	// check.
	if len(prep.MaskList) > 0 {
		return false, &InvalidPolicyError{Problems: []Problem{{Pointer: "/masks", Rule: RuleValue}}}
	}
	// The ids belong to the tenant, before the insert, so a foreign id is a
	// named problem and not the trigger's 23514.
	scope, err := NotInTenant(ctx, tx, prep.Refs)
	if err != nil {
		return false, err
	}
	for i := range scope {
		scope[i].Rule = RuleForeignID
	}
	nested, err := NestedUnderUnits(ctx, tx, prep.Refs)
	if err != nil {
		return false, err
	}
	if scope = append(scope, nested...); len(scope) > 0 {
		return false, &InvalidPolicyError{Problems: sortProblems(scope)}
	}
	_, err = PublishPrepared(ctx, tx, entityKey, 0, by, prep)
	if _, ok := errors.AsType[*ConflictError](err); ok {
		return false, nil
	}
	return err == nil, err
}

// publishLocked encodes format-1 rules and masks without checking them: the
// test seam behind Publish. See Publish.
func publishLocked(ctx context.Context, tx pgx.Tx, tenant, def uuid.UUID, expected int, by Publisher, rules []Rule, masks []Mask) (*Published, error) {
	if rules == nil {
		rules = []Rule{}
	}
	if masks == nil {
		masks = []Mask{}
	}
	rb, err := json.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("authz: encode rules: %w", err)
	}
	mb, err := json.Marshal(masks)
	if err != nil {
		return nil, fmt.Errorf("authz: encode masks: %w", err)
	}
	return publishRaw(ctx, tx, tenant, def, expected, by, rb, mb)
}

// publishRaw inserts rb and mb, already encoded, as version expected+1.
func publishRaw(ctx context.Context, tx pgx.Tx, tenant, def uuid.UUID, expected int, by Publisher, rb, mb []byte) (*Published, error) {
	cur, err := newest(ctx, tx, tenant, def)
	if err != nil {
		return nil, err
	}
	have := 0
	if cur != nil {
		have = cur.Version
	}
	if have != expected {
		return nil, &ConflictError{Current: have}
	}
	next := have + 1
	out := &Published{Version: next, RulesJSON: rb, MasksJSON: mb, PublishedBy: by.Principal, DefinitionID: def}
	if out.Document, err = Decode(rb); err != nil {
		return nil, fmt.Errorf("authz: decode published rules: %w", err)
	}
	if err := decodeStrict(mb, &out.Masks); err != nil {
		return nil, fmt.Errorf("authz: decode published masks: %w", err)
	}
	if err := tx.QueryRow(ctx, insertACLSQL, tenant, def, next, string(rb), string(mb), by.Principal).Scan(&out.PublishedAt); err != nil {
		return nil, fmt.Errorf("authz: insert acl version: %w", err)
	}
	if err := auditPublish(ctx, tx, def, cur, out, by); err != nil {
		return nil, err
	}
	return out, nil
}

func policyDoc(rules, masks json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(struct {
		Rules json.RawMessage `json:"rules"`
		Masks json.RawMessage `json:"masks"`
	}{rules, masks})
}

// auditPublish writes the config entry of the version just inserted: the
// subject is the entity type, the versions are ACL versions, the content hash
// covers the canonical rules and masks, and the diff is against the previous
// version (nil for the first). The entry is built from the stored values,
// never from the request body.
func auditPublish(ctx context.Context, tx pgx.Tx, def uuid.UUID, before, after *Published, by Publisher) error {
	afterDoc, err := policyDoc(after.RulesJSON, after.MasksJSON)
	if err != nil {
		return fmt.Errorf("authz: encode policy: %w", err)
	}
	var beforeDoc json.RawMessage
	var vBefore *int64
	if before != nil {
		if beforeDoc, err = policyDoc(before.RulesJSON, before.MasksJSON); err != nil {
			return fmt.Errorf("authz: encode policy: %w", err)
		}
		v := int64(before.Version)
		vBefore = &v
	}
	diff, err := jsondiff.Diff(beforeDoc, afterDoc)
	if err != nil {
		return fmt.Errorf("authz: diff policy: %w", err)
	}
	sum := sha256.Sum256(afterDoc)
	va := int64(after.Version)
	e := audit.Entry{
		Domain:        audit.DomainConfig,
		Action:        audit.ActionPublish,
		SubjectType:   audit.SubjectACLPolicy,
		SubjectID:     def,
		VersionBefore: vBefore,
		VersionAfter:  &va,
		Source:        by.Source,
		ContentSHA256: &sum,
		Diff:          diff,
	}
	if by.Principal != nil {
		e.Actor = audit.Actor{PrincipalID: by.Principal}
	} else {
		s := by.System
		e.Actor = audit.Actor{System: &s}
	}
	if id := requestid.FromContext(ctx); id != "" {
		e.RequestID = &id
	}
	if _, err := audit.Append(ctx, tx, e); err != nil {
		return fmt.Errorf("authz: audit acl publish: %w", err)
	}
	return nil
}

func nonNilRules(r []Rule) []Rule {
	if r == nil {
		return []Rule{}
	}
	return r
}

func nonNilMasks(m []Mask) []Mask {
	if m == nil {
		return []Mask{}
	}
	return m
}

// InvalidPolicyError is a policy that Prepare refuses, on a path with no
// request to answer 422 to (onboarding, the backfill).
type InvalidPolicyError struct{ Problems []Problem }

func (e *InvalidPolicyError) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		parts[i] = p.Rule + " at " + p.Pointer
	}
	return "authz: invalid access policy: " + strings.Join(parts, ", ")
}

// PublishPrepared inserts a checked policy (Prepare) as version expected+1,
// with its audit entry, in tx. It takes the definition's row lock, and
// returns *ConflictError when expected is not the newest version. The stored
// values are Prepare's canonical text, never the request body.
func (PolicyStore) PublishPrepared(ctx context.Context, tx pgx.Tx, entityKey string, expected int, by Publisher, p *Prepared) (*Published, error) {
	return PublishPrepared(ctx, tx, entityKey, expected, by, p)
}

// PublishPrepared is PolicyStore.PublishPrepared.
func PublishPrepared(ctx context.Context, tx pgx.Tx, entityKey string, expected int, by Publisher, p *Prepared) (*Published, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	var def uuid.UUID
	if err := tx.QueryRow(ctx, lockDefinitionSQL, tenant, entityKey).Scan(&def); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoEntityType
	} else if err != nil {
		return nil, fmt.Errorf("authz: lock entity type: %w", err)
	}
	tenancy.Forget(tx, policyKey{entityKey})
	return publishRaw(ctx, tx, tenant, def, expected, by, p.Rules, p.Masks)
}
