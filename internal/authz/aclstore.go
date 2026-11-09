package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// PolicyStore is the PostgreSQL PolicyReader: the newest acl_versions row of
// the entity type, read in the caller's transaction, so under its tenant's
// RLS, with the tenant taken from the transaction, never from a caller.
type PolicyStore struct{}

var _ PolicyReader = PolicyStore{}

// currentPolicySQL is a backward primary-key probe on acl_versions_pkey
// (docs/data/acl-tables.md).
const currentPolicySQL = `
SELECT a.entity_definition_id, a.version, a.rules, a.masks
FROM entity_definitions d
JOIN acl_versions a ON a.tenant_id = d.tenant_id AND a.entity_definition_id = d.id
WHERE d.tenant_id = $1 AND d.key = $2
ORDER BY a.version DESC
LIMIT 1`

// Current returns the entity type's newest published policy, or nil when it
// has none (or the tenant has no such type). A stored rule list that does not
// decode is an error, never an empty policy.
func (PolicyStore) Current(ctx context.Context, tx pgx.Tx, entityKey string) (*Policy, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	var (
		pol          Policy
		rules, masks []byte
	)
	err = tx.QueryRow(ctx, currentPolicySQL, tenant, entityKey).Scan(&pol.EntityDefinitionID, &pol.Version, &rules, &masks)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // nil policy is the contract's "none published" (fail closed)
	}
	if err != nil {
		return nil, fmt.Errorf("authz: read acl: %w", err)
	}
	if err := decodeStrict(rules, &pol.Rules); err != nil {
		return nil, fmt.Errorf("authz: decode acl %d rules: %w", pol.Version, err)
	}
	if err := decodeStrict(masks, &pol.Masks); err != nil {
		return nil, fmt.Errorf("authz: decode acl %d masks: %w", pol.Version, err)
	}
	return &pol, nil
}

func decodeStrict(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

// sharesSQL reads the record's shares that name p, matched on the
// (principal_kind, principal_id) pair (AR-S10-01), as CompileRead does.
const sharesSQL = `
SELECT coalesce(array_agg(principal_id ORDER BY principal_id), '{}') FROM record_shares
WHERE tenant_id = $1 AND record_id = $2
  AND ((principal_kind = 'user' AND principal_id = $3::uuid)
    OR (principal_kind = 'group' AND principal_id = ANY($4::uuid[])))`

// SharesFor is Row.SharedWith for p: the principal ids of the record's
// shares that name p, a user share for p's id or a group share for one of
// p's groups. It reads in tx, under the transaction's tenant. A principal
// with no id (a system writer) has no shares.
func SharesFor(ctx context.Context, tx pgx.Tx, p identity.Principal, recordID uuid.UUID) ([]uuid.UUID, error) {
	if p.PrincipalID == uuid.Nil {
		return nil, nil
	}
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	var ids []uuid.UUID
	if err := tx.QueryRow(ctx, sharesSQL, tenant, recordID, p.PrincipalID, UUIDTexts(p.Groups)).Scan(&ids); err != nil {
		return nil, fmt.Errorf("authz: read shares: %w", err)
	}
	return ids, nil
}

// ACL is the S10 Authorizer: it reads the entity type's current policy with
// Policies, in the caller's transaction, and decides with Allows.
type ACL struct {
	Policies PolicyReader
}

var _ Authorizer = ACL{}

// CanRead implements Authorizer.
func (a ACL) CanRead(ctx context.Context, tx pgx.Tx, p identity.Principal, entity string, r Row) error {
	pol, err := a.policy(ctx, tx, p, entity)
	if errors.Is(err, errOtherTenant) {
		return ErrNotVisible
	}
	if err != nil {
		return err
	}
	if _, ok := Allows(pol, p, OpRead, r); !ok {
		return ErrNotVisible
	}
	return nil
}

// CanWrite implements Authorizer.
func (a ACL) CanWrite(ctx context.Context, tx pgx.Tx, p identity.Principal, entity string, before, after *Row) error {
	if after == nil {
		return fmt.Errorf("authz: CanWrite needs the after row")
	}
	pol, err := a.policy(ctx, tx, p, entity)
	if errors.Is(err, errOtherTenant) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if before != nil {
		if _, ok := Allows(pol, p, OpWrite, *before); !ok {
			return ErrForbidden
		}
	}
	if _, ok := Allows(pol, p, OpWrite, *after); !ok {
		return ErrForbidden
	}
	return nil
}

var errOtherTenant = errors.New("authz: principal of another tenant")

// policy reads the policy after checking that p belongs to the transaction's
// tenant: p's groups and id are only meaningful there.
func (a ACL) policy(ctx context.Context, tx pgx.Tx, p identity.Principal, entity string) (*Policy, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	if p.TenantID == uuid.Nil || p.TenantID != tenant {
		return nil, errOtherTenant
	}
	pol, err := a.Policies.Current(ctx, tx, entity)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	return pol, nil
}

// CanWriteFields implements Authorizer: the mask write check (P3-33), beside
// the CanWrite row check, which it does not change (A1). before is the stored
// payload (nil on a create), after the caller's data (before merged with
// patch, without defaults) and patch the caller's merge patch (nil on a
// create). It returns *ErrFieldNotWritable for a field p may not write,
// ErrForbidden for a principal of another tenant, and a wrapped error when it
// could not decide.
func (a ACL) CanWriteFields(ctx context.Context, tx pgx.Tx, p identity.Principal, entity string, before, after, patch map[string]any) error {
	pol, err := a.policy(ctx, tx, p, entity)
	if errors.Is(err, errOtherTenant) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	return CheckWriteFields(pol, p, before, after, patch)
}

// MaskSource gives the read side the policy the shaping layer applies: the
// records handlers Shape every body with it, and the audit timeline masks
// each record entry's diff with ReadMasks.
type MaskSource interface {
	// Policy is the entity type's current policy for p, read in tx; nil when
	// none is published. A principal of another tenant is ErrNotVisible.
	Policy(ctx context.Context, tx pgx.Tx, p identity.Principal, entity string) (*Policy, error)
	// ReadMasks maps each entity definition of the transaction's tenant whose
	// current policy masks a field from p to those fields, sorted. A
	// principal of another tenant is ErrNotVisible.
	ReadMasks(ctx context.Context, tx pgx.Tx, p identity.Principal) (map[uuid.UUID][]string, error)
}

var _ MaskSource = ACL{}

// Policy implements MaskSource.
func (a ACL) Policy(ctx context.Context, tx pgx.Tx, p identity.Principal, entity string) (*Policy, error) {
	pol, err := a.policy(ctx, tx, p, entity)
	if errors.Is(err, errOtherTenant) {
		return nil, ErrNotVisible
	}
	return pol, err
}

// currentMasksSQL is the masks of every entity type's newest ACL version in
// the tenant: one backward walk of acl_versions_pkey per definition.
const currentMasksSQL = `
SELECT DISTINCT ON (entity_definition_id) entity_definition_id, version, masks
FROM acl_versions
WHERE tenant_id = $1
ORDER BY entity_definition_id, version DESC`

// ReadMasks implements MaskSource. It reads under the transaction's tenant,
// never one a caller names. A stored mask list that does not decode is an
// error, never "no masks".
func (ACL) ReadMasks(ctx context.Context, tx pgx.Tx, p identity.Principal) (map[uuid.UUID][]string, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	if p.TenantID == uuid.Nil || p.TenantID != tenant {
		return nil, ErrNotVisible
	}
	rows, err := tx.Query(ctx, currentMasksSQL, tenant)
	if err != nil {
		return nil, fmt.Errorf("authz: read acl masks: %w", err)
	}
	defer rows.Close()
	out := map[uuid.UUID][]string{}
	for rows.Next() {
		var (
			pol   Policy
			masks []byte
		)
		if err := rows.Scan(&pol.EntityDefinitionID, &pol.Version, &masks); err != nil {
			return nil, fmt.Errorf("authz: scan acl masks: %w", err)
		}
		if err := decodeStrict(masks, &pol.Masks); err != nil {
			return nil, fmt.Errorf("authz: decode acl %d masks: %w", pol.Version, err)
		}
		if fields := MaskedFields(&pol, p); len(fields) > 0 {
			out[pol.EntityDefinitionID] = fields
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("authz: read acl masks: %w", err)
	}
	return out, nil
}
