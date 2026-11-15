// Package eventsink is the records pipeline's EventSink from S11b on (PLAN-003
// P3-42, D-3-14). In the write's transaction it:
//
//  1. starts nava.interpreter once for each published definition whose
//     current version's trigger matches the write's entity type and op: the
//     start is keyed by cascade.ClaimStart (ADR-001 A5) and enqueued by
//     outbox.EnqueueStart, with target_key nava.interpreter:<outbox id>
//     (D-3-13 as amended 2026-11-01). On a repeat claim it enqueues nothing
//     and writes the cascade_cycle entry through pipeline.AuditSuppressed
//     (A5-4);
//  2. signals each open interpreter run on the record that is waiting on
//     record_changed (workflow_ref.record_changed_signal_id is set), through
//     outbox.EnqueueSignal, with payload {"signal": <that id>}. It never
//     signals Event.WorkflowRef, the run whose own write this is (D-3-14), and
//     never a run that is not waiting (R9 head-of-line).
//
// outbox.EnqueueStart and outbox.EnqueueSignal write each row's
// workflow_cause row in the same transaction (D-3-29). Nothing here reads a
// payload, a definition body or record data: the tenant comes from the
// transaction, the causation from the record row (ClaimStart) or the write
// (Event.Causation), and record_changed_signal_id is copied as an opaque id.
package eventsink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The trigger_on values of workflow_definition_versions.
const (
	onCreated = "record.created"
	onUpdated = "record.updated"
)

// kindVersion is the kind_version of both interpreter rows.
const kindVersion = 1

const (
	// triggeredSQL finds the current published version of every definition
	// a write fires (index workflow_definition_versions_trigger, then a
	// primary-key probe of workflow_definitions). Ordered by definition, so
	// two writes claim their keys in one order.
	triggeredSQL = `SELECT v.definition_id, v.id FROM workflow_definition_versions v
JOIN workflow_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id AND d.current_version = v.version
WHERE v.tenant_id = $1 AND v.trigger_entity_definition_id = $2 AND v.trigger_on = $3
ORDER BY v.definition_id`
	// createTriggeredSQL is triggeredSQL's question for a create, as a
	// column of the pipeline's prefetch read of the entity type (aliased e):
	// is there anything to start? The same index serves it.
	createTriggeredSQL = `EXISTS (SELECT 1 FROM workflow_definition_versions v
JOIN workflow_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id AND d.current_version = v.version
WHERE v.tenant_id = e.tenant_id AND v.trigger_entity_definition_id = e.id AND v.trigger_on = '` + onCreated + `')`
	// waitingSQL finds the open interpreter runs on the record that wait on
	// record_changed, but the writing run (index workflow_ref_waiting).
	waitingSQL = `SELECT outbox_id, record_changed_signal_id FROM workflow_ref
WHERE tenant_id = $1 AND record_id = $2 AND kind = $3 AND closed_at IS NULL
  AND record_changed_signal_id IS NOT NULL AND outbox_id <> $4
ORDER BY outbox_id`
)

// Sink is the EventSink. It holds no state and is safe for concurrent use.
type Sink struct{}

var (
	_ pipeline.EventSink           = Sink{}
	_ pipeline.CreateTriggerQueuer = Sink{}
)

// CreateTriggeredSQL is true, on the pipeline's read of an entity type e,
// when a create of e fires a published definition. A create has no run on
// its record yet, so with none the sink has nothing to do
// (pipeline.CreateTriggerQueuer).
func (Sink) CreateTriggeredSQL() string { return createTriggeredSQL }

// RecordChanged starts the definitions ev fires and signals the runs waiting
// on the record, in tx (see the package comment). Any error rolls the write
// back.
func (Sink) RecordChanged(ctx context.Context, tx pgx.Tx, ev pipeline.Event) error {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return fmt.Errorf("eventsink: %w", err)
	}
	if err := start(ctx, tx, tenant, ev); err != nil {
		return err
	}
	return signal(ctx, tx, tenant, ev)
}

// triggered is one definition a write fires, at its current version.
type triggered struct{ definition, version uuid.UUID }

func start(ctx context.Context, tx pgx.Tx, tenant uuid.UUID, ev pipeline.Event) error {
	on := onUpdated
	if ev.Op == pipeline.OpCreate {
		on = onCreated
	}
	rows, err := tx.Query(ctx, triggeredSQL, tenant, ev.EntityDefinitionID, on)
	if err != nil {
		return fmt.Errorf("eventsink: find triggered definitions: %w", err)
	}
	fired, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (triggered, error) {
		var t triggered
		err := r.Scan(&t.definition, &t.version)
		return t, err
	})
	if err != nil {
		return fmt.Errorf("eventsink: find triggered definitions: %w", err)
	}
	for _, t := range fired {
		c, claimed, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: ev.RecordID, DefinitionID: t.definition})
		if err != nil {
			return fmt.Errorf("eventsink: claim start: %w", err)
		}
		if !claimed {
			if err := pipeline.AuditSuppressed(ctx, tx, c, ev); err != nil {
				return err
			}
			continue
		}
		payload, err := json.Marshal(map[string]uuid.UUID{"definition_version_id": t.version, "record_id": ev.RecordID})
		if err != nil {
			return fmt.Errorf("eventsink: start payload: %w", err)
		}
		// The causation is left empty: EnqueueStart takes the claim's, read
		// from the record row.
		_, err = outbox.EnqueueStart(ctx, tx, c, outbox.Entry{
			Kind:        outbox.InterpreterKind,
			KindVersion: kindVersion,
			DedupeKey:   StartDedupeKey(c.OutboxID()),
			RecordID:    ev.RecordID,
			Target:      outbox.TargetStart{Kind: outbox.InterpreterKind},
			Payload:     payload,
		})
		if err != nil {
			return fmt.Errorf("eventsink: enqueue start: %w", err)
		}
	}
	return nil
}

// waiting is one run waiting on record_changed: its ref and the signal id
// it waits on.
type waiting struct{ ref, signal uuid.UUID }

func signal(ctx context.Context, tx pgx.Tx, tenant uuid.UUID, ev pipeline.Event) error {
	rows, err := tx.Query(ctx, waitingSQL, tenant, ev.RecordID, outbox.InterpreterKind, ev.WorkflowRef)
	if err != nil {
		return fmt.Errorf("eventsink: find waiting runs: %w", err)
	}
	runs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (waiting, error) {
		var w waiting
		err := r.Scan(&w.ref, &w.signal)
		return w, err
	})
	if err != nil {
		return fmt.Errorf("eventsink: find waiting runs: %w", err)
	}
	for _, w := range runs {
		payload, err := json.Marshal(map[string]uuid.UUID{"signal": w.signal})
		if err != nil {
			return fmt.Errorf("eventsink: signal payload: %w", err)
		}
		_, err = outbox.EnqueueSignal(ctx, tx, w.ref, outbox.Entry{
			Kind:        outbox.InterpreterSignalKind,
			KindVersion: kindVersion,
			DedupeKey:   SignalDedupeKey(w.ref, ev.Version),
			RecordID:    ev.RecordID,
			Target:      outbox.TargetSignal{Kind: outbox.InterpreterKind, SignalName: outbox.InterpreterSignalKind},
			Causation:   ev.Causation,
			Payload:     payload,
		}, ev.Version)
		// The run closed after the lookup (a concurrent CloseWorkflowRef):
		// there is nobody to tell, and the write goes ahead.
		if errors.Is(err, outbox.ErrNoOpenWorkflow) {
			continue
		}
		if err != nil {
			return fmt.Errorf("eventsink: enqueue signal: %w", err)
		}
	}
	return nil
}

// StartDedupeKey is the dedupe_key of the start row whose id is the claim's
// outbox id: unique, as the id is.
func StartDedupeKey(outboxID uuid.UUID) string {
	return outbox.InterpreterKind + ":" + outboxID.String()
}

// SignalDedupeKey is the dedupe_key of the record_changed signal to the run
// ref for the write that left its record at version: one per run and
// write, as every write moves the version.
func SignalDedupeKey(ref uuid.UUID, version int64) string {
	return fmt.Sprintf("%s:%s:%d", outbox.InterpreterSignalKind, ref, version)
}
