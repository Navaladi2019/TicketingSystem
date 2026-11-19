package audit

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/secrets"
)

// Domain is audit_log.domain (ADR-004 R1).
type Domain string

// The three domains.
const (
	DomainRecord Domain = "record"
	DomainConfig Domain = "config"
	DomainAudit  Domain = "audit"
)

// Action is audit_log.action, the closed R2 list with ADR-004-E1's two
// cascade actions and ADR-004-E9's number_space_exhausted.
// audit_log_action_check mirrors it (TestP510bSubjectTypesMirrorRegistry).
type Action string

// The R2 actions.
const (
	ActionCreate         Action = "create"
	ActionUpdate         Action = "update"
	ActionPublish        Action = "publish"
	ActionEnable         Action = "enable"
	ActionDisable        Action = "disable"
	ActionDelete         Action = "delete"
	ActionPromote        Action = "promote"
	ActionRollback       Action = "rollback"
	ActionStateChange    Action = "state_change"
	ActionExport         Action = "export"
	ActionErase          Action = "erase"
	ActionReseed         Action = "reseed"
	ActionCascadeCycle   Action = "cascade_cycle"
	ActionCascadeTooDeep Action = "cascade_too_deep"
	// ActionNumberSpaceExhausted records a create refused because its
	// number format's scope has no number left (ADR-004-E9).
	ActionNumberSpaceExhausted Action = "number_space_exhausted"
)

// Source is audit_log.source, the closed R2 channel list.
// audit_log_source_check mirrors it.
type Source string

// The R2 sources.
const (
	SourceUser       Source = "user"
	SourceAI         Source = "ai"
	SourceChangeSet  Source = "change_set"
	SourceOnboarding Source = "onboarding"
	SourcePlugin     Source = "plugin"
	SourceRule       Source = "rule"
	SourceWorkflow   Source = "workflow"
	SourceFunction   Source = "function"
	SourceSystem     Source = "system"
)

// Actions returns every R2 action.
func Actions() []Action {
	return []Action{
		ActionCreate, ActionUpdate, ActionPublish, ActionEnable, ActionDisable, ActionDelete,
		ActionPromote, ActionRollback, ActionStateChange, ActionExport, ActionErase, ActionReseed,
		ActionCascadeCycle, ActionCascadeTooDeep, ActionNumberSpaceExhausted,
	}
}

// Sources returns every R2 source.
func Sources() []Source {
	return []Source{
		SourceUser, SourceAI, SourceChangeSet, SourceOnboarding, SourcePlugin,
		SourceRule, SourceWorkflow, SourceFunction, SourceSystem,
	}
}

// Valid reports whether d is one of the three domains.
func (d Domain) Valid() bool { return d == DomainRecord || d == DomainConfig || d == DomainAudit }

// Valid reports whether a is an R2 action.
func (a Action) Valid() bool { return slices.Contains(Actions(), a) }

// Valid reports whether s is an R2 source.
func (s Source) Valid() bool { return slices.Contains(Sources(), s) }

// Actor is who made the change: exactly one of a principal or a named system
// actor (R1).
type Actor struct {
	PrincipalID *uuid.UUID
	System      *string
}

// Entry is one audit_log entry as a writer supplies it (PLAN-005 C2). It has
// no tenant: Append takes the tenant from the transaction's app.tenant_id
// (I2), never from the caller. It has no occurred_at, txid, backfilled, seal
// or erasure field either; the database sets those (R10).
type Entry struct {
	// ID is the entry id; Append mints a UUIDv7 when it is zero.
	ID                 uuid.UUID
	Domain             Domain
	Action             Action
	SubjectType        SubjectType
	SubjectID          uuid.UUID
	EntityDefinitionID *uuid.UUID
	VersionBefore      *int64
	VersionAfter       *int64
	Actor              Actor
	Source             Source
	// SourceRef is required for source = ai: the AI decision record's id, a
	// lowercase UUID (R4).
	SourceRef      *string
	ChangeSetID    *uuid.UUID
	CausationID    *uuid.UUID
	CausationDepth int16
	RequestID      *string
	// ContentSHA256 is the configuration version's content hash, required
	// for domain = config except a delete (R3, Annex A).
	ContentSHA256 *[32]byte
	// Diff is the RFC 6902 patch bytes, stored as given (json, never jsonb);
	// nil for an entry with no diff. Build it with internal/audit/jsondiff.
	Diff []byte
	// SecretPaths are the subject schema's x-nava-secret and writeOnly
	// pointers. They match token by token (no array wildcard).
	SecretPaths []patchwalk.Pointer
}

// ErrInvalidEntry is wrapped by every rejection of an entry that is wrong in
// itself or has no tenant to belong to. The message names the field and the
// rule, never a value. It is a caller bug, not a client error.
var ErrInvalidEntry = errors.New("audit: invalid entry")

// ErrSecretInDiff is the R6 rejection (code audit_secret_in_diff). It names
// the pointer, the op and the rule, never the value.
type ErrSecretInDiff = secrets.ErrSecretInDiff

// ErrDuplicateEntry is returned when the transaction already holds an entry
// for the same subject and version_after (the R2 commit key).
var ErrDuplicateEntry = errors.New("audit: duplicate entry for this transaction, subject and version")

func invalid(field, rule string) error {
	return fmt.Errorf("%w: %s: %s", ErrInvalidEntry, field, rule)
}

// validate checks everything about e that needs no database. It runs before
// Append reads the tenant or touches the transaction.
func (e Entry) validate() error {
	if !e.Domain.Valid() {
		return invalid("domain", "not one of record, config, audit")
	}
	if !e.Action.Valid() {
		return invalid("action", "not an R2 action")
	}
	if !e.SubjectType.Valid() {
		return invalid("subject_type", "not a registered subject type")
	}
	if e.SubjectID == uuid.Nil {
		return invalid("subject_id", "required")
	}
	if !e.Source.Valid() {
		return invalid("source", "not an R2 source")
	}
	if (e.Actor.PrincipalID == nil) == (e.Actor.System == nil) {
		return invalid("actor", "exactly one of PrincipalID or System is required")
	}
	for _, f := range []struct {
		name string
		id   *uuid.UUID
	}{
		{"actor_principal_id", e.Actor.PrincipalID},
		{"entity_definition_id", e.EntityDefinitionID},
		{"change_set_id", e.ChangeSetID},
		{"causation_id", e.CausationID},
	} {
		if f.id != nil && *f.id == uuid.Nil {
			return invalid(f.name, "the nil UUID (use nil)")
		}
	}
	if e.CausationDepth < 0 {
		return invalid("causation_depth", "negative")
	}
	if e.Source == SourceAI {
		if e.SourceRef == nil {
			return invalid("source_ref", "required for source = ai (R4)")
		}
		if u, err := uuid.Parse(*e.SourceRef); err != nil || u.String() != *e.SourceRef {
			return invalid("source_ref", "must be a lowercase UUID for source = ai (R4)")
		}
	}
	if e.Domain == DomainConfig && e.Action != ActionDelete && e.ContentSHA256 == nil {
		return invalid("content_sha256", "required for domain = config (R3)")
	}
	// The canonical form is what the sealer will hash. Anything it refuses
	// (empty or invalid text, U+0000, an integer past 2^53-1) would be stored
	// but could never be sealed, and would stall the tenant's chain, so it is
	// refused here. The envelope's tenant_id, occurred_at and diff columns
	// are zero placeholders: Append does not know them yet, and they always
	// encode.
	if _, err := chain.CanonicalEnvelope(e.envelope()); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidEntry, err)
	}
	return nil
}

// envelope is e's part of the hashed envelope. tenant_id, occurred_at,
// backfilled, diff_sha256 and diff_truncated are left zero; Append and the
// database supply them.
func (e Entry) envelope() chain.Envelope {
	return chain.Envelope{
		ID:                 e.ID,
		Domain:             string(e.Domain),
		Action:             string(e.Action),
		SubjectType:        string(e.SubjectType),
		SubjectID:          e.SubjectID,
		EntityDefinitionID: e.EntityDefinitionID,
		VersionBefore:      e.VersionBefore,
		VersionAfter:       e.VersionAfter,
		ActorPrincipalID:   e.Actor.PrincipalID,
		ActorSystem:        e.Actor.System,
		Source:             string(e.Source),
		SourceRef:          e.SourceRef,
		ChangeSetID:        e.ChangeSetID,
		CausationID:        e.CausationID,
		CausationDepth:     e.CausationDepth,
		RequestID:          e.RequestID,
		ContentSHA256:      e.ContentSHA256,
	}
}
