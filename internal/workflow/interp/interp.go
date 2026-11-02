// Package interp is the workflow interpreter, nava.interpreter (P3-22a,
// PLAN-003 D-3-11, D-3-12, D-3-13, D-3-15): one Temporal workflow type that
// runs any published v1 definition (internal/workflow/def).
//
// It is reference-only (ADR-001 R1, CI-7). Workflow code holds the
// definition version id, the record id, the current step id and the
// workerguard.Directive the NextStep activity returns; never the definition
// or a record value. Every read and write happens in two regular activities:
//
//   - NextStep loads the definition version and the record for the tenant of
//     the nava-tenant header, resolves condition and branch steps, and returns
//     the first step that needs the workflow (timer, wait_signal,
//     update_record, end) as a Directive;
//   - ApplyStep performs an update_record step through RecordWriter, with the
//     idempotency key wf:<workflow id>:<step id> (D-3-13).
//
// Only NextStep builds a Directive, and every Directive value comes from the
// published definition and its step-id map, never from a record field.
//
// See docs/architecture/workflow-interpreter.md.
package interp

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
)

// WorkflowType is the registered name of the interpreter workflow, on the
// nava-tq queue (D-3-11). It is the one workflow type the guard lets receive a
// workerguard.Directive (workerguard.DirectiveWorkflow, TICK-386).
const WorkflowType = workerguard.DirectiveWorkflow

// Activity names. NextStep is the one activity the guard lets record a
// workerguard.Directive (workerguard.DirectiveActivity).
const (
	NextStepActivity  = workerguard.DirectiveActivity
	ApplyStepActivity = "nava.interpreter.ApplyStep"
)

// SignalName is the one Temporal signal name the interpreter listens on
// (D-3-12). Its argument is the relay's outbox.Delivery, whose payload is
// {"signal": <signal id>}; the signal id is the "signal:<name>" entry of the
// version's step-id map.
const SignalName = "nava.interpreter.signal"

// StepRef is the one argument of NextStep and ApplyStep. It holds UUIDs only,
// so the worker guard records it (internal/workerguard/args.go). Step is
// uuid.Nil for the definition's start step. Signal is set only on the
// NextStep call that follows a delivered signal, and is omitted otherwise,
// never null.
type StepRef struct {
	DefinitionVersionID uuid.UUID  `json:"definition_version_id"`
	RecordID            uuid.UUID  `json:"record_id"`
	Step                uuid.UUID  `json:"step"`
	Signal              *uuid.UUID `json:"signal,omitempty"`
}

// Version is one published definition version: the parsed definition and
// the step-id map def.AssignStepIDs assigned at publish (D-3-15).
type Version struct {
	Definition *def.Definition
	StepIDs    map[string]uuid.UUID
}

// DefinitionStore loads a published definition version of tenant. It returns
// ErrNotFound when tenant has no such version. Activity side only; the real
// adapter reads in workerguard.InTenantTx (P3-22b).
type DefinitionStore interface {
	Load(ctx context.Context, tenantID, versionID uuid.UUID) (*Version, error)
}

// RecordReader reads a record of tenant as encoding/json decodes it. It
// returns ErrNotFound when tenant has no such record. Activity side only; the
// real adapter reads in workerguard.InTenantTx.
type RecordReader interface {
	Read(ctx context.Context, tenantID, recordID uuid.UUID) (map[string]any, error)
}

// RecordWriter applies an update_record step's set to a record of tenant,
// once per idempotencyKey: a second call with the same key returns the first
// result and changes nothing (D-3-13). The real adapter writes through the
// records pipeline as the workflow actor (P3-43). Business rejections wrap
// ErrValidation, ErrIllegalTransition or ErrForbidden; a missing record wraps
// ErrNotFound.
type RecordWriter interface {
	Write(ctx context.Context, tenantID, recordID, stepID uuid.UUID, idempotencyKey string, set map[string]any) error
}

// Port errors. Adapters wrap them; the activities map each to a fixed
// workerguard.Rejection, so no adapter text reaches a history.
var (
	ErrNotFound          = errors.New("interp: not found")
	ErrValidation        = errors.New("interp: record write failed validation")
	ErrIllegalTransition = errors.New("interp: record write is an illegal transition")
	ErrForbidden         = errors.New("interp: record write is forbidden")
)

// IdempotencyKey is the key ApplyStep writes a step with (D-3-13). The
// workflow id stays the same across continue-as-new, so a step keeps its key.
func IdempotencyKey(workflowID string, step uuid.UUID) string {
	return "wf:" + workflowID + ":" + step.String()
}
