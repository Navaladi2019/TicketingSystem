// Command demo-outbox is the S7 demo (TICK-275): it writes one start row and
// two signal rows for a record of the seeded tenant tenant-a, through the real
// write path in tenancy.InTenantTx as nava_app. Three pipeline.Write calls
// (PLAN-003 D-3-6) create the record and update it twice; the demo's
// EventSink enqueues one row per write, in the write's transaction: the start
// through cascade.ClaimStart and outbox.EnqueueStart (ADR-001 A5-6), each
// signal through outbox.EnqueueAtVersion at the version its update gave the
// record. The relay service hands the rows to Temporal. Run it through
// scripts/dev/demo-outbox.sh (make demo-outbox), which seeds tenant-a's
// tenants row and the demo entity type (seed-sql below) and puts it on the
// compose network.
//
// All three writes share one transaction, so the rows take target_seq 1, 2
// and 3 in the order below. DEDUPE_KEY names the writes (their idempotency
// keys, D-3-13) and the rows (<key>:start, <key>:signal-1, <key>:signal-2),
// so a second run with the same key writes nothing and reports the same ids.
//
// "demo-outbox seed-sql" only prints the superuser SQL that seeds the demo
// entity type in tenant-a at version 1. The SQL is idempotent.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
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
	// definition is the one the cascade start key names (A5).
	definition = "5e7a0000-0000-4000-8000-0000000000d0"
	// The longest key whose suffixed forms fit outbox's 200-byte dedupe key
	// and the pipeline's 200-byte idempotency key.
	maxKeyLen = 150

	// The demo entity type, seeded by seed-sql with fixed ids so a rerun
	// names the same rows.
	entityKey       = "demo_note"
	entityPrefix    = "DEMO"
	entityDefID     = "5e7a0000-0000-4000-8000-0000000000e1"
	entityVersionID = "5e7a0000-0000-4000-8000-0000000000e2"
	entitySchema    = `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object",` +
		`"additionalProperties":false,"required":["note"],"properties":{` +
		`"status":{"type":"string","enum":["open","closed"]},"note":{"type":"string","minLength":1}}}`

	// rowSQL finds a row by its dedupe key, for a replayed write, whose sink
	// does not run; RLS keeps it to the transaction's tenant.
	rowSQL = `SELECT id FROM outbox WHERE dedupe_key = $1`
)

// entityDefinition is the demo entity type.
func entityDefinition() metadata.Definition {
	return metadata.Definition{
		Key:          entityKey,
		Schema:       json.RawMessage(entitySchema),
		NumberPrefix: entityPrefix,
		FixedColumns: map[string]string{"status": metadata.ColumnState},
		States: metadata.StateModel{
			Initial: "open", States: []string{"open", "closed"},
			Transitions: []metadata.Transition{{From: "open", To: "closed"}},
		},
	}
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "seed-sql" {
		s, err := seedSQL()
		if err != nil {
			fmt.Fprintln(os.Stderr, "demo-outbox:", err)
			os.Exit(1)
		}
		fmt.Print(s)
		return
	}
	if err := run(context.Background(), os.LookupEnv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "demo-outbox:", err)
		os.Exit(1)
	}
}

// seedSQL is the superuser SQL that seeds the demo entity type in tenant-a at
// version 1, as entitytypes.Store.Seed writes a type (no publisher), and
// the workflow definition the start key names (cascade_start's foreign key,
// P3-40; no creator, and no audit entry until P3-41 attaches its trigger). The
// tenants row must exist. Every INSERT is ON CONFLICT DO NOTHING, so a rerun
// changes nothing. The definition is compiled first, so the seed never stores
// a type the pipeline would refuse.
func seedSQL() (string, error) {
	d := entityDefinition()
	if _, err := metadata.Compile(d); err != nil {
		return "", fmt.Errorf("demo entity type: %w", err)
	}
	fixed, err := json.Marshal(d.FixedColumns)
	if err != nil {
		return "", err
	}
	states, err := json.Marshal(d.States)
	if err != nil {
		return "", err
	}
	// Every value below is a constant of this file. The JSON values go in as
	// quoted literals, so none may hold a quote; the schema goes in
	// dollar-quoted, so its bytes are kept, and must not hold the delimiter.
	if strings.Contains(string(fixed)+string(states), "'") {
		return "", errors.New("demo entity type: a quote in a seeded value")
	}
	if strings.Contains(entitySchema, "$nava$") {
		return "", errors.New("demo entity type: the schema holds the literal's delimiter")
	}
	sum := sha256.Sum256([]byte(entitySchema))
	// The seed is a bare fixture, not a publish: like the isolation seed
	// (internal/tenancy/isolation/testdata/seed.sql), it turns the
	// entity-definition (P5-32, migration B) and workflow_definitions (P3-41b)
	// commit-check triggers off for its own explicit transaction and back on before it commits. Only
	// audit.Append writes audit_log (internal/records/arch_test.go).
	return fmt.Sprintf(`BEGIN;
ALTER TABLE entity_definitions DISABLE TRIGGER entity_definitions_audit_commit_check;
ALTER TABLE entity_definition_versions DISABLE TRIGGER entity_definition_versions_audit_commit_check;
ALTER TABLE workflow_definitions DISABLE TRIGGER workflow_definitions_audit_commit_check;
INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
VALUES ('%[1]s', '%[2]s', '%[3]s', 'Demo note', '%[4]s', 1) ON CONFLICT DO NOTHING;
INSERT INTO entity_definition_versions
  (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
VALUES ('%[1]s', '%[5]s', '%[2]s', 1, $nava$%[6]s$nava$::json, '\x%[7]s'::bytea, '%[8]s'::jsonb, '%[9]s'::jsonb, 'initial', NULL)
ON CONFLICT DO NOTHING;
INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version, created_by)
VALUES ('%[1]s', '%[10]s', 'demo_start', '%[2]s', 1, NULL) ON CONFLICT DO NOTHING;
ALTER TABLE entity_definitions ENABLE TRIGGER entity_definitions_audit_commit_check;
ALTER TABLE entity_definition_versions ENABLE TRIGGER entity_definition_versions_audit_commit_check;
ALTER TABLE workflow_definitions ENABLE TRIGGER workflow_definitions_audit_commit_check;
COMMIT;
`, tenantA, entityDefID, entityKey, entityPrefix, entityVersionID, entitySchema, hex.EncodeToString(sum[:]),
		fixed, states, definition), nil
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

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return errors.New("NAVA_DATABASE_URL is not a valid PostgreSQL connection string")
	}
	// Two: the pipeline's own-transaction runner needs a second connection
	// while the demo's transaction is open (pipeline.TxRunner).
	cfg.MaxConns = 2
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec // PgBouncer, transaction mode
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return errors.New("cannot create the PostgreSQL pool")
	}
	defer pool.Close()

	sink := &demoSink{}
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, sink, pipeline.PoolRunner{Pool: pool}, pipeline.SystemClock{})
	var (
		record uuid.UUID
		ids    []uuid.UUID
	)
	err = tenancy.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
		var err error
		record, ids, err = writeAll(ctx, tx, p, sink, tenant, key)
		return err
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

// demoWrite is one of the demo's three writes and the row its sink enqueues.
type demoWrite struct {
	suffix string
	target outbox.Target
	kind   string
	data   map[string]any
}

func demoWrites() []demoWrite {
	signal := outbox.TargetSignal{Kind: workflowType, SignalName: signalName}
	return []demoWrite{
		{"start", outbox.TargetStart{Kind: workflowType}, workflowType, map[string]any{"note": "created"}},
		{"signal-1", signal, signalName, map[string]any{"note": "first note"}},
		{"signal-2", signal, signalName, map[string]any{"note": "second note"}},
	}
}

// writeAll runs the create and the two updates in tx and returns the record
// and the ids of the start row and the two signal rows.
func writeAll(ctx context.Context, tx pgx.Tx, p *pipeline.Pipeline, sink *demoSink, tenant uuid.UUID, key string) (uuid.UUID, []uuid.UUID, error) {
	// A system writer of the tenant: the demo is no person, and its writes
	// carry idempotency keys (D-3-13), which only a system writer may.
	actor := identity.Principal{TenantID: tenant}
	writes := demoWrites()
	var (
		record  uuid.UUID
		version int64
	)
	ids := make([]uuid.UUID, len(writes))
	for i, w := range writes {
		cmd := pipeline.Command{Op: pipeline.OpCreate, EntityKey: entityKey, Data: w.data, Actor: actor,
			Source: pipeline.SourceSystem, IdempotencyKey: key + ":" + w.suffix}
		if i > 0 {
			cmd.Op, cmd.RecordID, cmd.ExpectedVersion = pipeline.OpUpdate, record, version
		}
		sink.next = outbox.Entry{Kind: w.kind, KindVersion: 1, DedupeKey: key + ":" + w.suffix, Target: w.target}
		sink.id = uuid.Nil
		res, err := p.Write(ctx, tx, cmd)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("write %s: %w", cmd.IdempotencyKey, err)
		}
		record, version = res.RecordID, res.Version
		if ids[i] = sink.id; ids[i] == uuid.Nil {
			// A replay: the pipeline returned the earlier result and called
			// no sink, so the row is the first run's.
			if err := tx.QueryRow(ctx, rowSQL, sink.next.DedupeKey).Scan(&ids[i]); err != nil {
				return uuid.Nil, nil, fmt.Errorf("find row %s: %w", sink.next.DedupeKey, err)
			}
		}
	}
	return record, ids, nil
}

// demoSink is the demo's pipeline.EventSink: it enqueues next for the write
// it is told about, in the write's transaction, and keeps the row's id.
type demoSink struct {
	next outbox.Entry
	id   uuid.UUID
}

// RecordChanged enqueues the start through ClaimStart and EnqueueStart, or a
// signal through EnqueueAtVersion at ev.Version, under the write's causation.
func (s *demoSink) RecordChanged(ctx context.Context, tx pgx.Tx, ev pipeline.Event) error {
	e := s.next
	payload, err := json.Marshal(map[string]string{"record_id": ev.RecordID.String()})
	if err != nil {
		return err
	}
	e.RecordID, e.Payload, e.Causation = ev.RecordID, payload, ev.Causation
	if _, ok := e.Target.(outbox.TargetStart); ok {
		c, claimed, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: ev.RecordID, DefinitionID: uuid.MustParse(definition)})
		if err != nil {
			return fmt.Errorf("claim start: %w", err)
		}
		if !claimed {
			return errors.New("the new record's start key is already claimed")
		}
		s.id, err = outbox.EnqueueStart(ctx, tx, c, e)
		return err
	}
	s.id, err = outbox.EnqueueAtVersion(ctx, tx, e, ev.Version)
	return err
}
