package authz

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Who values of a workflow policy test (ADR-010 D9): whose access the test
// asks about. They are the values of def.Step.Who; this package does not
// import the workflow packages.
const (
	WhoRequester = "requester"
	WhoAssignee  = "assignee"
)

// PolicyRefActor is the system actor of the audit entry a missing policy
// reference writes.
const PolicyRefActor = "workflow"

// PolicyTest is one workflow policy test (ADR-010 D9, TICK-957): does the
// record's Who principal hold Op on the record under the policy Name of
// EntityType. The tenant is not a field: it is the transaction's.
type PolicyTest struct {
	RecordID   uuid.UUID
	EntityType string
	Policy     string
	Who        string
	Op         string
}

// ErrPolicyTestRecord is a record the transaction's tenant does not have.
var ErrPolicyTestRecord = errors.New("authz: policy test: no such record")

const policyTestRecordSQL = `SELECT d.id, d.key, r.state, r.requester_id, r.assignee_id, r.assignment_group_id
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
WHERE r.tenant_id = $1 AND r.id = $2`

// TestPolicyRef answers a workflow policy test in tx, the tenant transaction
// of the run (never a pool without tenant context). Everything it reads, the
// record, the entity type's current policy, the principal and its groups and
// the shares, is read in tx, so it sees only the tenant's rows and the
// newest published version (D9): a policy narrowed after the run started
// applies at the next call.
//
// It returns false, and never an error, for every refusal:
//   - the record's entity type is not in.EntityType, or the type has no
//     published policy of that name: a missing reference. It is counted
//     (PolicyRefMissingMetric, labelled with the record's own entity type),
//     logged and audited in tx, and never plans another type's policy;
//   - the record's Who column is NULL, or the principal is not a live
//     principal of the tenant: false, not a missing reference;
//   - the policy is voided (D5b): false, counted by PolicyInvalidMetric;
//   - the policy does not grant Op to the principal on the record.
//
// An error is a store failure (retry), or ErrPolicyTestRecord for a record
// the tenant does not have.
func TestPolicyRef(ctx context.Context, tx pgx.Tx, store PolicyReader, in PolicyTest) (bool, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return false, fmt.Errorf("authz: %w", err)
	}
	var (
		defID                         uuid.UUID
		recType, state                string
		requester, assignee, groupRef *uuid.UUID
	)
	switch err := tx.QueryRow(ctx, policyTestRecordSQL, tenant, in.RecordID).
		Scan(&defID, &recType, &state, &requester, &assignee, &groupRef); {
	case errors.Is(err, pgx.ErrNoRows):
		return false, ErrPolicyTestRecord
	case err != nil:
		return false, fmt.Errorf("authz: policy test: read record: %w", err)
	}
	if in.Op != OpRead && in.Op != OpWrite {
		return false, nil
	}
	var who *uuid.UUID
	switch in.Who {
	case WhoRequester:
		who = requester
	case WhoAssignee:
		who = assignee
	default:
		return false, nil
	}

	var pol *Policy
	if recType == in.EntityType {
		if pol, err = store.Current(ctx, tx, recType); err != nil {
			return false, err
		}
	}
	if pol == nil || !slices.ContainsFunc(pol.Document.Policies, func(np NamedPolicy) bool { return np.Name == in.Policy }) {
		return false, policyRefMissing(ctx, tx, in, recType, defID)
	}
	if who == nil {
		return false, nil
	}
	p, found, err := PrincipalOf(ctx, tx, *who)
	if err != nil || !found {
		return false, err
	}
	shared, err := SharesFor(ctx, tx, p, in.RecordID)
	if err != nil {
		return false, err
	}
	row := Row{ID: in.RecordID, State: state, RequesterID: requester, AssigneeID: assignee, AssignmentGroupID: groupRef, SharedWith: shared}
	return NamedPolicyAllows(pol, in.Policy, p, in.Op, row), nil
}

// NamedPolicyAllows reports whether the policy name of pol grants op to p on
// r, judged by that policy alone (the others are not consulted): false for a name pol lacks, a policy
// that does not grant op, and a voided policy, which is counted
// (PolicyInvalidMetric, D5b). It is the pure part of TestPolicyRef.
func NamedPolicyAllows(pol *Policy, name string, p identity.Principal, op string, r Row) bool {
	for _, pp := range plan(pol, p, op) {
		if pp.name != name {
			continue
		}
		if pp.err != nil {
			countInvalid(pol, pp)
			return false
		}
		return pp.tree.eval(r)
	}
	return false // the policy does not grant op, or folds to false
}

// policyRefMissing counts, logs and audits a reference that names nothing
// (D9) in tx. It returns the audit error, which fails the attempt: a denial
// without its audit entry is not recorded.
func policyRefMissing(ctx context.Context, tx pgx.Tx, in PolicyTest, recType string, defID uuid.UUID) error {
	slog.WarnContext(ctx, "authz: workflow policy reference names no policy, it denies",
		slog.String("entity_type", recType), slog.String("ref_entity_type", in.EntityType), slog.String("policy", in.Policy))
	countRefMissing(ctx, recType)
	sys := PolicyRefActor
	// No R2 action means "a reference denied": the closest subject is the access
	// policy of the record's entity type, the thing the reference failed to find,
	// not the record, whose history is the requester's. The content hash is of
	// the reference named, so an auditor can match entries by it (open question
	// on TICK-957: a dedicated action needs a schema change).
	sum := sha256.Sum256([]byte(in.EntityType + "/" + in.Policy))
	e := audit.Entry{
		Domain:             audit.DomainConfig,
		Action:             audit.ActionUpdate,
		SubjectType:        audit.SubjectACLPolicy,
		SubjectID:          defID,
		EntityDefinitionID: &defID,
		Actor:              audit.Actor{System: &sys},
		Source:             audit.SourceWorkflow,
		ContentSHA256:      &sum,
	}
	if id := requestid.FromContext(ctx); id != "" {
		e.RequestID = &id
	}
	if _, err := audit.Append(ctx, tx, e); err != nil {
		return fmt.Errorf("authz: audit policy ref missing: %w", err)
	}
	return nil
}
