// Command demo-outbox is the S7 demo (TICK-275): it writes one start row and
// two signal rows for a record of the seeded tenant tenant-a, through the real
// producer path in tenancy.InTenantTx as nava_app: the start through
// cascade.ClaimStart and outbox.EnqueueStart (ADR-001 A5-6), the signals
// through outbox.Enqueue. The relay service hands them to Temporal. Run it
// through scripts/dev/demo-outbox.sh (make demo-outbox), which puts it on the
// compose network.
//
// All three rows share one transaction and the record's causation, so they
// take target_seq in the order below: the start the record's version (it
// bumps nothing, D-3-6), each signal the next. DEDUPE_KEY names the rows
// (<key>:start, <key>:signal-1, <key>:signal-2) and picks the record, so a
// second run with the same key writes nothing and reports the same ids.
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
	// tenantA is the id of the seeded Keycloak organization tenant-a
	// (deploy/compose/README.md, "Organization ids are fixed").
	tenantA = "00000000-0000-4000-8000-0000000000a1"
	// workflowType is the platform type on the Authorizer's allow-list (D9).
	// The stack's always-on nava-worker does not register it, so the demo's
	// workflow stays Running with its two signals in the history. Its
	// payload carries no definition_version_id, so an interpreter worker
	// (nava-worker-interptest) that picked it up would reject it at once
	// (RejectInput) instead of running a step (TICK-374).
	workflowType = interp.WorkflowType
	signalName   = "demo.note"
	// definition is the one the cascade start key names (A5): fixed, so a
	// rerun finds the key it claimed.
	definition = "5e7a0000-0000-4000-8000-0000000000d0"
	// The longest key whose suffixed forms fit outbox's 200-byte dedupe key.
	maxKeyLen = 150

	// insertRecordSQL seeds the record with the depth-0 causation the write
	// pipeline stores on a person's create (A5-3); a rerun keeps the first
	// run's origin.
	insertRecordSQL = `INSERT INTO records (tenant_id, id, causation_origin_id, causation_depth) VALUES ($1, $2, $3, 0)
ON CONFLICT DO NOTHING`
	// claimedSQL finds the start row that holds a key already claimed for
	// the record's origin; RLS keeps it to the transaction's tenant.
	claimedSQL = `SELECT s.outbox_id FROM cascade_start s JOIN records r ON r.tenant_id = s.tenant_id AND r.id = s.record_id
WHERE s.record_id = $1 AND s.definition_id = $2 AND s.causation_origin_id = r.causation_origin_id`
)

func main() {
	if err := run(context.Background(), os.LookupEnv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "demo-outbox:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, lookup func(string) (string, bool), out io.Writer) error {
	dsn, _ := lookup("NAVA_DATABASE_URL")
	key, _ := lookup("DEDUPE_KEY")
	if dsn == "" {
		return errors.New("NAVA_DATABASE_URL is not set")
	}
	if key == "" || len(key) > maxKeyLen {
		return fmt.Errorf("DEDUPE_KEY must be 1 to %d bytes", maxKeyLen)
	}
	tenant := uuid.MustParse(tenantA)
	record := uuid.NewSHA1(tenant, []byte(key))

	cfg, err := pgxpool.ParseConfig(dsn)
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

	payload, err := json.Marshal(map[string]string{"record_id": record.String()})
	if err != nil {
		return err
	}
	entries := []outbox.Entry{
		{Kind: workflowType, DedupeKey: key + ":start", Target: outbox.TargetStart{Kind: workflowType}},
		{Kind: signalName, DedupeKey: key + ":signal-1", Target: outbox.TargetSignal{Kind: workflowType, SignalName: signalName}},
		{Kind: signalName, DedupeKey: key + ":signal-2", Target: outbox.TargetSignal{Kind: workflowType, SignalName: signalName}},
	}
	// Every row carries an origin, depth 0 included (ADR-001 A5-3): one
	// cascade, as if one person's write had caused all three. The record
	// stores it; the start takes it from the claim, the signals likewise.
	origin, err := uuid.NewV7()
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, len(entries))
	err = tenancy.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, insertRecordSQL, tenant, record, origin); err != nil {
			return fmt.Errorf("insert record: %w", err)
		}
		for i := range entries {
			entries[i].KindVersion, entries[i].RecordID, entries[i].Payload = 1, record, payload
		}
		c, err := enqueueStart(ctx, tx, entries[0], &ids[0])
		if err != nil {
			return fmt.Errorf("enqueue %s: %w", entries[0].DedupeKey, err)
		}
		o, depth := c.Causation()
		for i, e := range entries[1:] {
			e.Causation = outbox.Causation{OriginID: o, Depth: depth}
			if ids[i+1], err = outbox.Enqueue(ctx, tx, e); err != nil {
				return fmt.Errorf("enqueue %s: %w", e.DedupeKey, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	wid, ok := relay.WorkflowID(relay.Claimed{ID: ids[0], TenantID: tenant, Kind: workflowType,
		TargetKey: workflowType + ":" + ids[0].String()})
	if !ok {
		return errors.New("the relay would refuse the start row")
	}
	_, _ = fmt.Fprintf(out, "tenant: %s\nrecord: %s\ndedupe key: %s\nstart row: %s\nsignal row 1: %s\nsignal row 2: %s\nworkflow id: %s\n",
		tenant, record, key, ids[0], ids[1], ids[2], wid)
	return nil
}

// enqueueStart claims the start key of the demo's definition on e's record
// and enqueues e with outbox.EnqueueStart (A5-6), which gives the row the
// record's causation. A key already claimed (a rerun) enqueues nothing and
// sets *id to the row that holds it. It returns the claim either way.
func enqueueStart(ctx context.Context, tx pgx.Tx, e outbox.Entry, id *uuid.UUID) (cascade.Claim, error) {
	def := uuid.MustParse(definition)
	c, claimed, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: e.RecordID, DefinitionID: def})
	if err != nil {
		return c, fmt.Errorf("claim start: %w", err)
	}
	if claimed {
		*id, err = outbox.EnqueueStart(ctx, tx, c, e)
		return c, err
	}
	if err := tx.QueryRow(ctx, claimedSQL, e.RecordID, def).Scan(id); err != nil {
		return c, fmt.Errorf("find the claimed start: %w", err)
	}
	return c, nil
}
