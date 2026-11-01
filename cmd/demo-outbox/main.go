// Command demo-outbox is the S7 demo (TICK-275): it writes one start row and
// two signal rows for a record of the seeded tenant tenant-a, through the real
// producer path (outbox.Enqueue in tenancy.InTenantTx) as nava_app. The relay
// service hands them to Temporal. Run it through scripts/dev/demo-outbox.sh
// (make demo-outbox), which puts it on the compose network.
//
// All three rows share one transaction, so they take consecutive versions in
// the order below. DEDUPE_KEY names the rows (<key>:start, <key>:signal-1,
// <key>:signal-2) and picks the record, so a second run with the same key
// writes nothing and reports the same ids.
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
	"github.com/Navaladi2019/TicketingSystem/internal/relay"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const (
	// tenantA is the id of the seeded Keycloak organization tenant-a
	// (deploy/compose/README.md, "Organization ids are fixed").
	tenantA = "00000000-0000-4000-8000-0000000000a1"
	// workflowType is on the Authorizer's allow-list (D9).
	workflowType = "nava.smoke.relay"
	signalName   = "demo.note"
	// The longest key whose suffixed forms fit outbox's 200-byte dedupe key.
	maxKeyLen = 150

	insertRecordSQL = `INSERT INTO records (tenant_id, id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
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
	// cascade, as if one person's write had caused all three.
	origin, err := uuid.NewV7()
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, len(entries))
	err = tenancy.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, insertRecordSQL, tenant, record); err != nil {
			return fmt.Errorf("insert record: %w", err)
		}
		for i, e := range entries {
			e.KindVersion, e.RecordID, e.Payload = 1, record, payload
			e.Causation = outbox.Causation{OriginID: origin}
			if ids[i], err = outbox.Enqueue(ctx, tx, e); err != nil {
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
