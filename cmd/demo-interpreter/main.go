// Command demo-interpreter is the S11a demo (TICK-352): it writes the outbox
// rows that drive one nava.interpreter run through the real producer path
// (tenancy.InTenantTx, as nava_app), and the relay hands them to Temporal.
// Run it through scripts/dev/demo-interpreter.sh (the make demo-interpreter-*
// targets), which puts it on the compose network, installs the fixture
// definitions and seeds the records it starts on, with a causation origin
// (ADR-001 A5-3), as the write pipeline would store them.
//
// DEMO_ACTION picks what it writes:
//
//   - start: claims the start key of DEFINITION_ID on record RECORD_ID of
//     TENANT_ID with cascade.ClaimStart and enqueues the start row of
//     nava.interpreter with outbox.EnqueueStart (A5-6), payload
//     {definition_version_id: VERSION_ID, record_id}, and its workflow_ref.
//     The row carries the record's causation. It prints the workflow id the
//     relay gives the run. A key already claimed (a rerun) enqueues nothing
//     and prints the row that holds it.
//   - signal: a signal row on nava.interpreter.signal, payload {signal:
//     SIGNAL_ID}, for the newest open interpreter run of RECORD_ID, through
//     outbox.EnqueueSignal at the record's version (which also writes its
//     workflow_cause row), under a new depth-0 origin (a person's action).
//   - record-id: no database; prints the record id the demo derives from
//     TENANT_ID and DEDUPE_KEY, so the script can write the record's fixture
//     file before the start row exists.
//
// DEDUPE_KEY names the row, so running it again with the same key writes
// nothing and prints the same ids. The record must exist: this command
// writes no record table (PLAN-003 D-3-6).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/relay"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

const (
	// maxKeyLen is outbox's dedupe key limit.
	maxKeyLen = 200

	// versionSQL reads the record's version; RLS keeps it to the
	// transaction's tenant.
	versionSQL = `SELECT version FROM records WHERE id = $1`

	// newestRefSQL is the newest open interpreter run on the record; RLS
	// keeps it to the transaction's tenant.
	newestRefSQL = `SELECT outbox_id FROM workflow_ref WHERE record_id = $1 AND kind = $2 AND closed_at IS NULL
ORDER BY created_at DESC, outbox_id DESC LIMIT 1`

	// claimedSQL finds the start row that holds a key already claimed for
	// the record's origin; RLS keeps it to the transaction's tenant.
	claimedSQL = `SELECT s.outbox_id FROM cascade_start s JOIN records r ON r.tenant_id = s.tenant_id AND r.id = s.record_id
WHERE s.record_id = $1 AND s.definition_id = $2 AND s.causation_origin_id = r.causation_origin_id`
)

func main() {
	if err := run(context.Background(), os.LookupEnv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "demo-interpreter:", err)
		os.Exit(1)
	}
}

// RecordID is the record a demo key names in a tenant: the same key always
// names the same record.
func RecordID(tenant uuid.UUID, key string) uuid.UUID { return uuid.NewSHA1(tenant, []byte(key)) }

// input is the parsed environment.
type input struct {
	dsn, action, key       string
	tenant, record, target uuid.UUID
	definition             uuid.UUID // start only: the cascade key's definition
}

func parse(lookup func(string) (string, bool)) (input, error) {
	get := func(name string) string { v, _ := lookup(name); return v }
	in := input{dsn: get("NAVA_DATABASE_URL"), action: get("DEMO_ACTION"), key: get("DEDUPE_KEY")}
	if in.key == "" || len(in.key) > maxKeyLen {
		return in, fmt.Errorf("DEDUPE_KEY must be 1 to %d bytes", maxKeyLen)
	}
	if in.action == "record-id" {
		id, err := uuid.Parse(get("TENANT_ID"))
		if err != nil || id == uuid.Nil {
			return in, errors.New("TENANT_ID must be a UUID")
		}
		in.tenant = id
		return in, nil
	}
	if in.dsn == "" {
		return in, errors.New("NAVA_DATABASE_URL is not set")
	}
	targetVar := map[string]string{"start": "VERSION_ID", "signal": "SIGNAL_ID"}[in.action]
	if targetVar == "" {
		return in, fmt.Errorf("DEMO_ACTION must be start, signal or record-id, not %q", in.action)
	}
	vars := []struct {
		name string
		dst  *uuid.UUID
	}{{"TENANT_ID", &in.tenant}, {"RECORD_ID", &in.record}, {targetVar, &in.target}}
	if in.action == "start" {
		vars = append(vars, struct {
			name string
			dst  *uuid.UUID
		}{"DEFINITION_ID", &in.definition})
	}
	for _, f := range vars {
		id, err := uuid.Parse(get(f.name))
		if err != nil || id == uuid.Nil {
			return in, fmt.Errorf("%s must be a UUID", f.name)
		}
		*f.dst = id
	}
	return in, nil
}

// entry is the outbox row the action writes.
func (in input) entry() (outbox.Entry, error) {
	e := outbox.Entry{KindVersion: 1, DedupeKey: in.key, RecordID: in.record}
	var payload map[string]uuid.UUID
	if in.action == "start" {
		e.Kind, e.Target = interp.WorkflowType, outbox.TargetStart{Kind: interp.WorkflowType}
		payload = map[string]uuid.UUID{"definition_version_id": in.target, "record_id": in.record}
	} else {
		e.Kind = interp.SignalName
		e.Target = outbox.TargetSignal{Kind: interp.WorkflowType, SignalName: interp.SignalName}
		payload = map[string]uuid.UUID{"signal": in.target}
	}
	b, err := json.Marshal(payload)
	e.Payload = b
	return e, err
}

func run(ctx context.Context, lookup func(string) (string, bool), out io.Writer) error {
	in, err := parse(lookup)
	if err != nil {
		return err
	}
	if in.action == "record-id" {
		_, _ = fmt.Fprintln(out, RecordID(in.tenant, in.key))
		return nil
	}
	e, err := in.entry()
	if err != nil {
		return err
	}

	cfg, err := pgxpool.ParseConfig(in.dsn)
	if err != nil {
		return errors.New("NAVA_DATABASE_URL is not a valid PostgreSQL connection string")
	}
	cfg.MaxConns = 1
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec // PgBouncer, transaction mode
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return errors.New("cannot create the PostgreSQL pool")
	}
	defer pool.Close()

	var id uuid.UUID
	err = tenancy.InTenantTx(ctx, pool, in.tenant, func(tx pgx.Tx) error {
		var err error
		if in.action == "start" {
			id, err = enqueueStart(ctx, tx, in.definition, e)
			return err
		}
		// A signal is a person's action: a new cascade at depth 0 (A5-3).
		origin, err := uuid.NewV7()
		if err != nil {
			return err
		}
		e.Causation = outbox.Causation{OriginID: origin}
		// The record's version, unbumped: outbox never moves it and this
		// command writes no record table (D-3-6). The signal shares the
		// start's target_seq; the script sends it once the run waits, so the
		// start is already dispatched.
		var v int64
		switch err := tx.QueryRow(ctx, versionSQL, e.RecordID).Scan(&v); {
		case errors.Is(err, pgx.ErrNoRows):
			return outbox.ErrRecordNotFound
		case err != nil:
			return fmt.Errorf("read record version: %w", err)
		}
		var ref uuid.UUID
		switch err := tx.QueryRow(ctx, newestRefSQL, e.RecordID, interp.WorkflowType).Scan(&ref); {
		case errors.Is(err, pgx.ErrNoRows):
			return outbox.ErrNoOpenWorkflow
		case err != nil:
			return fmt.Errorf("find the open run: %w", err)
		}
		id, err = outbox.EnqueueSignal(ctx, tx, ref, e, v)
		return err
	})
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", in.key, err)
	}

	_, _ = fmt.Fprintf(out, "tenant: %s\nrecord: %s\ndedupe key: %s\n%s row: %s\n", in.tenant, in.record, in.key, in.action, id)
	if in.action == "start" {
		wid, ok := relay.WorkflowID(relay.Claimed{ID: id, TenantID: in.tenant, Kind: interp.WorkflowType,
			TargetKey: interp.WorkflowType + ":" + id.String()})
		if !ok {
			return errors.New("the relay would refuse the start row")
		}
		_, _ = fmt.Fprintf(out, "workflow id: %s\n", wid)
	}
	return nil
}

// enqueueStart claims the start key of definition on e's record and enqueues
// e with outbox.EnqueueStart (A5-6); the row takes the causation the record
// carries. A key already claimed enqueues nothing and returns the id of the
// row that holds it, so a rerun prints the same ids.
func enqueueStart(ctx context.Context, tx pgx.Tx, definition uuid.UUID, e outbox.Entry) (uuid.UUID, error) {
	c, claimed, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: e.RecordID, DefinitionID: definition})
	if err != nil {
		return uuid.Nil, fmt.Errorf("claim start: %w", err)
	}
	if claimed {
		return outbox.EnqueueStart(ctx, tx, c, e)
	}
	var id uuid.UUID
	if err := tx.QueryRow(ctx, claimedSQL, e.RecordID, definition).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("find the claimed start: %w", err)
	}
	return id, nil
}
