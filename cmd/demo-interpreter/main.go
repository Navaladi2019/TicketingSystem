// Command demo-interpreter is the S11a demo (TICK-352): it writes the outbox
// rows that drive one nava.interpreter run through the real producer path
// (outbox.Enqueue in tenancy.InTenantTx, as nava_app), and the relay hands
// them to Temporal. Run it through scripts/dev/demo-interpreter.sh (the
// make demo-interpreter-* targets), which puts it on the compose network and
// installs the fixture definitions and records the interptest worker reads.
//
// DEMO_ACTION picks what it writes:
//
//   - start: a start row of nava.interpreter for record RECORD_ID of
//     TENANT_ID, payload {definition_version_id: VERSION_ID, record_id}, and
//     its workflow_ref. It prints the workflow id the relay gives the run.
//   - signal: a signal row on nava.interpreter.signal, payload {signal:
//     SIGNAL_ID}, for the newest open interpreter run of RECORD_ID.
//   - record-id: no database; prints the record id the demo derives from
//     TENANT_ID and DEDUPE_KEY, so the script can write the record's fixture
//     file before the start row exists.
//
// DEDUPE_KEY names the row, so running it again with the same key writes
// nothing and prints the same ids.
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
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

const (
	// maxKeyLen is outbox's dedupe key limit.
	maxKeyLen = 200

	insertRecordSQL = `INSERT INTO records (tenant_id, id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
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
	for _, f := range []struct {
		name string
		dst  *uuid.UUID
	}{{"TENANT_ID", &in.tenant}, {"RECORD_ID", &in.record}, {targetVar, &in.target}} {
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
		if in.action == "start" {
			if _, err := tx.Exec(ctx, insertRecordSQL, in.tenant, in.record); err != nil {
				return fmt.Errorf("insert record: %w", err)
			}
		}
		var err error
		id, err = outbox.Enqueue(ctx, tx, e)
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
