package interp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
)

// Activities are NextStep and ApplyStep. Register them with Register: both
// are regular activities, never local, because the guard accepts a Directive
// only as a regular activity's result (internal/workerguard/results.go).
type Activities struct {
	Definitions DefinitionStore
	Records     RecordReader
	Writer      RecordWriter
}

// Register adds the activities to r under their fixed names.
func (a *Activities) Register(r worker.ActivityRegistry) {
	r.RegisterActivityWithOptions(a.NextStep, activity.RegisterOptions{Name: NextStepActivity})
	r.RegisterActivityWithOptions(a.ApplyStep, activity.RegisterOptions{Name: ApplyStepActivity})
}

// errNoTenant: the guard did not put a verified tenant on the context. It is
// not one of the Rejections, so the guard withholds its text; it is
// non-retryable because a retry has the same header.
func errNoTenant() error {
	return temporal.NewNonRetryableApplicationError("interp: no verified tenant", "nava.interp.NoTenant", nil)
}

// rejection maps a port error to its fixed Rejection, or returns err as is
// (the guard then withholds it as ErrTypeOpaque, retryable).
func rejection(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return workerguard.Rejected(workerguard.RejectNotFound)
	case errors.Is(err, ErrValidation):
		return workerguard.Rejected(workerguard.RejectValidation)
	case errors.Is(err, ErrIllegalTransition):
		return workerguard.Rejected(workerguard.RejectTransition)
	case errors.Is(err, ErrForbidden):
		return workerguard.Rejected(workerguard.RejectForbidden)
	}
	return err
}

// loaded is a version with its step-id map inverted.
type loaded struct {
	*Version
	keyOf map[uuid.UUID]string
}

// load reads the version for the header tenant. The tenant comes only from
// the context the activity guard set from the nava-tenant header.
func (a *Activities) load(ctx context.Context, ref StepRef) (uuid.UUID, *loaded, error) {
	tenant, ok := workerguard.TenantFromContext(ctx)
	if !ok {
		return uuid.Nil, nil, errNoTenant()
	}
	if ref.DefinitionVersionID == uuid.Nil || ref.RecordID == uuid.Nil {
		return uuid.Nil, nil, workerguard.Rejected(workerguard.RejectInput)
	}
	v, err := a.Definitions.Load(ctx, tenant, ref.DefinitionVersionID)
	if err != nil {
		return uuid.Nil, nil, rejection(err)
	}
	if v == nil || v.Definition == nil {
		return uuid.Nil, nil, workerguard.Rejected(workerguard.RejectNotFound)
	}
	l := &loaded{Version: v, keyOf: make(map[uuid.UUID]string, len(v.StepIDs))}
	for k, id := range v.StepIDs {
		l.keyOf[id] = k
	}
	return tenant, l, nil
}

// stepKey is the key of a step id; uuid.Nil is the start step. A signal id is
// not a step.
func (l *loaded) stepKey(id uuid.UUID) (string, bool) {
	if id == uuid.Nil {
		return l.Definition.Start, true
	}
	k, ok := l.keyOf[id]
	if !ok {
		return "", false
	}
	_, isStep := l.Definition.Step(k)
	return k, isStep
}

// id is the step-id map entry for key; a version whose map misses a key is
// treated as not found.
func (l *loaded) id(key string) (*uuid.UUID, error) {
	id, ok := l.StepIDs[key]
	if !ok || id == uuid.Nil {
		return nil, workerguard.Rejected(workerguard.RejectNotFound)
	}
	return &id, nil
}

// NextStep returns the Directive for ref.Step, or for the first step after it
// that needs the workflow: condition and branch steps are resolved here,
// against the record. The walk is bounded by def.MaxSteps (the graph is
// acyclic). ref.Signal, when set, must be one of the version's signal ids.
func (a *Activities) NextStep(ctx context.Context, ref StepRef) (workerguard.Directive, error) {
	tenant, l, err := a.load(ctx, ref)
	if err != nil {
		return workerguard.Directive{}, err
	}
	if ref.Signal != nil {
		if k, ok := l.keyOf[*ref.Signal]; !ok || len(k) <= len(def.SignalKeyPrefix) || k[:len(def.SignalKeyPrefix)] != def.SignalKeyPrefix {
			return workerguard.Directive{}, workerguard.Rejected(workerguard.RejectNotFound)
		}
	}
	// The record is read on every call, so a record the tenant does not have
	// fails at once, not at the first condition.
	record, err := a.Records.Read(ctx, tenant, ref.RecordID)
	if err != nil {
		return workerguard.Directive{}, rejection(err)
	}
	key, ok := l.stepKey(ref.Step)
	if !ok {
		return workerguard.Directive{}, workerguard.Rejected(workerguard.RejectNotFound)
	}
	for range def.MaxSteps + 1 {
		s, ok := l.Definition.Step(key)
		if !ok {
			return workerguard.Directive{}, workerguard.Rejected(workerguard.RejectNotFound)
		}
		switch s.Type {
		case def.TypeCondition:
			key = s.Else
			if s.Holds(record) {
				key = s.Then
			}
			continue
		case def.TypeBranch:
			key = s.Default
			for _, c := range s.Cases {
				if c.Holds(record) {
					key = c.Next
					break
				}
			}
			continue
		case def.TypeTimer, def.TypeWaitSignal, def.TypeUpdateRecord, def.TypeEnd:
			return l.directive(s)
		}
		return workerguard.Directive{}, workerguard.Rejected(workerguard.RejectStep)
	}
	// Unreachable for a definition Parse accepted.
	return workerguard.Directive{}, workerguard.Rejected(workerguard.RejectStep)
}

// directive builds the Directive for an action step. Every value comes from
// the definition and its step-id map: WaitSeconds is the parsed duration or
// timeout (def.Step.WaitSeconds), never a record field.
func (l *loaded) directive(s *def.Step) (workerguard.Directive, error) {
	step, err := l.id(s.Key)
	if err != nil {
		return workerguard.Directive{}, err
	}
	d := workerguard.Directive{Step: *step}
	switch s.Type {
	case def.TypeTimer:
		d.Kind, d.WaitSeconds = workerguard.KindTimer, s.WaitSeconds()
		d.Next, err = l.id(s.Next)
	case def.TypeWaitSignal:
		d.Kind, d.WaitSeconds = workerguard.KindWaitSignal, s.WaitSeconds()
		if d.Next, err = l.id(s.Next); err == nil {
			if d.OnTimeout, err = l.id(s.OnTimeout); err == nil {
				d.Signal, err = l.id(def.SignalKey(s.Name))
			}
		}
	case def.TypeUpdateRecord:
		d.Kind = workerguard.KindUpdateRecord
		d.Next, err = l.id(s.Next)
	case def.TypeEnd:
		d.Kind = workerguard.KindEnd
	case def.TypeCondition, def.TypeBranch:
		return workerguard.Directive{}, workerguard.Rejected(workerguard.RejectStep)
	}
	if err != nil {
		return workerguard.Directive{}, err
	}
	return d, nil
}

// ApplyStep performs the update_record step ref.Step: it loads the step's set
// itself and calls RecordWriter.Write with IdempotencyKey(workflow id, step).
// A business rejection is a non-retryable Rejection with fixed text, so the
// run fails instead of retrying forever; any other error is retried, and the
// guard withholds its text.
func (a *Activities) ApplyStep(ctx context.Context, ref StepRef) error {
	tenant, l, err := a.load(ctx, ref)
	if err != nil {
		return err
	}
	key, ok := l.keyOf[ref.Step]
	if !ok {
		return workerguard.Rejected(workerguard.RejectNotFound)
	}
	s, ok := l.Definition.Step(key)
	if !ok || s.Type != def.TypeUpdateRecord {
		return workerguard.Rejected(workerguard.RejectStep)
	}
	set, err := decodeSet(s.Set)
	if err != nil {
		return workerguard.Rejected(workerguard.RejectStep)
	}
	idem := IdempotencyKey(activity.GetInfo(ctx).WorkflowExecution.ID, ref.Step)
	if err := a.Writer.Write(ctx, tenant, ref.RecordID, ref.Step, idem, set); err != nil {
		return rejection(err)
	}
	afterCommit(ctx, idem)
	return nil
}

// decodeSet decodes the step's literals, keeping numbers exact.
func decodeSet(raw map[string]json.RawMessage) (map[string]any, error) {
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		dec := json.NewDecoder(bytes.NewReader(v))
		dec.UseNumber()
		var x any
		if err := dec.Decode(&x); err != nil {
			return nil, err
		}
		out[k] = x
	}
	return out, nil
}
