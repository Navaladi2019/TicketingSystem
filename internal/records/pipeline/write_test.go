package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The Write paths below that need a statement to fail run on scriptTx, a
// pgx.Tx that answers each statement from a script keyed by a fragment of its
// SQL. The database behaviour itself is covered by the integration tests;
// these pin how the pipeline maps each failure and that nothing is written
// after a refusal.

var errBoom = errors.New("boom")

// stmt answers the statements whose SQL contains match: QueryRow scans with
// scan, or fails with err; Exec fails with err.
type stmt struct {
	match string
	scan  func(dest ...any) error
	err   error
}

type scriptTx struct {
	pgx.Tx // nil: any method not overridden panics, which fails the test
	t      *testing.T
	tenant string // what tenancy.TenantOf reads
	stmts  []stmt
	ran    []string
}

type rowFunc func(dest ...any) error

func (f rowFunc) Scan(dest ...any) error { return f(dest...) }

func (x *scriptTx) find(sql string) (stmt, bool) {
	x.ran = append(x.ran, sql)
	for _, s := range x.stmts {
		if strings.Contains(sql, s.match) {
			return s, true
		}
	}
	return stmt{}, false
}

func (x *scriptTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "current_setting(") {
		x.ran = append(x.ran, sql)
		return rowFunc(func(dest ...any) error { *dest[0].(*string) = x.tenant; return nil })
	}
	s, ok := x.find(sql)
	switch {
	case !ok:
		x.t.Errorf("unexpected statement: %s", sql)
		return rowFunc(func(...any) error { return errors.New("unexpected statement") })
	case s.err != nil:
		return rowFunc(func(...any) error { return s.err })
	}
	return rowFunc(s.scan)
}

func (x *scriptTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	s, ok := x.find(sql)
	if !ok {
		x.t.Errorf("unexpected statement: %s", sql)
		return pgconn.CommandTag{}, errors.New("unexpected statement")
	}
	return pgconn.NewCommandTag("OK"), s.err
}

// ran reports whether any statement containing fragment ran.
func (x *scriptTx) ranAny(fragment string) bool {
	for _, s := range x.ran {
		if strings.Contains(s, fragment) {
			return true
		}
	}
	return false
}

// Fragments of the store's statements.
const (
	sqlDefinition = "SELECT id FROM entity_definitions"
	sqlWriteKey   = "FROM record_write_keys"
	sqlGet        = "r.data, r.causation_origin_id"
	sqlLock       = "pg_advisory_xact_lock"
	sqlTaken      = "SELECT EXISTS"
	sqlNumber     = "record_counters"
	sqlInsert     = "INSERT INTO records"
	sqlUpdate     = "UPDATE records"
	sqlPayload    = "::text FROM records"
	sqlAudit      = "INSERT INTO audit_log"
	sqlShares     = "FROM record_shares"
)

func scanUUID(id uuid.UUID) func(...any) error {
	return func(dest ...any) error { *dest[0].(*uuid.UUID) = id; return nil }
}

func scanInt(n int64) func(...any) error {
	return func(dest ...any) error { *dest[0].(*int64) = n; return nil }
}

func scanString(v string) func(...any) error {
	return func(dest ...any) error { *dest[0].(*string) = v; return nil }
}

func scanBool(b bool) func(...any) error {
	return func(dest ...any) error { *dest[0].(*bool) = b; return nil }
}

// storedRecord is a records row as store.Get scans it.
type storedRecord struct {
	tenant, id, def uuid.UUID
	key             string
	version         int64
	schemaVersion   int
	state           string
	data            string
}

func (r storedRecord) scan(dest ...any) error {
	now := time.Now()
	*dest[0].(*uuid.UUID) = r.tenant
	*dest[1].(*uuid.UUID) = r.id
	*dest[2].(*uuid.UUID) = r.def
	*dest[3].(*string) = r.key
	*dest[4].(*int64) = r.version
	*dest[5].(*int) = r.schemaVersion
	*dest[6].(*string) = "AST0000001"
	*dest[7].(*string) = r.state
	*dest[12].(*time.Time) = now
	*dest[13].(*time.Time) = now
	*dest[14].(*[]byte) = []byte(r.data)
	*dest[16].(*int16) = 0
	return nil
}

// assetDef is the compiled asset type: name required, status the state
// column (active -> retired), serial unique. required adds to the schema's
// required list.
func assetDef(t *testing.T, required ...string) *metadata.Compiled {
	t.Helper()
	req, _ := json.Marshal(append([]string{"name"}, required...))
	c, err := metadata.Compile(metadata.Definition{
		Key: "asset",
		Schema: json.RawMessage(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ` + string(req) + `,
  "properties": {
    "name": {"type": "string", "minLength": 1},
    "status": {"type": "string", "enum": ["active", "retired"]},
    "serial": {"type": "string", "x-nava-unique": true},
    "tag": {"type": "number"}
  }
}`),
		NumberPrefix: "AST",
		FixedColumns: map[string]string{"status": metadata.ColumnState},
		States: metadata.StateModel{
			Initial:     "active",
			States:      []string{"active", "retired"},
			Transitions: []metadata.Transition{{From: "active", To: "retired"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// fakeDefs serves cur as the current definition at curVersion.
type fakeDefs struct {
	cur        *metadata.Compiled
	curVersion int
	curErr     error
	verErr     error
}

func (d fakeDefs) Version(context.Context, pgx.Tx, string, int) (*metadata.Compiled, error) {
	if d.verErr != nil {
		return nil, d.verErr
	}
	return d.cur, nil
}

func (d fakeDefs) Current(context.Context, pgx.Tx, string) (*metadata.Compiled, int, error) {
	if d.curErr != nil {
		return nil, 0, d.curErr
	}
	return d.cur, d.curVersion, nil
}

// fakeAuthz answers every write with err and every read with readErr, and
// records the rows it was asked about.
type fakeAuthz struct {
	err, readErr, fieldsErr error
	calls                   *[]authzCall
}

type authzCall struct {
	op                    string
	before, after         *authz.Row
	beforeData, afterData map[string]any
}

func (a fakeAuthz) CanRead(_ context.Context, _ pgx.Tx, _ identity.Principal, _ string, r authz.Row) error {
	if a.calls != nil {
		*a.calls = append(*a.calls, authzCall{op: authz.OpRead, after: &r})
	}
	return a.readErr
}

func (a fakeAuthz) CanWrite(_ context.Context, _ pgx.Tx, _ identity.Principal, _ string, before, after *authz.Row) error {
	if a.calls != nil {
		*a.calls = append(*a.calls, authzCall{op: authz.OpWrite, before: before, after: after})
	}
	return a.err
}

func (a fakeAuthz) CanWriteFields(_ context.Context, _ pgx.Tx, _ identity.Principal, _ string, before, after map[string]any) error {
	if a.calls != nil {
		*a.calls = append(*a.calls, authzCall{op: "fields", beforeData: before, afterData: after})
	}
	return a.fieldsErr
}

// runnerFunc is a TxRunner.
type runnerFunc func(ctx context.Context, tenant uuid.UUID, fn func(pgx.Tx) error) error

func (f runnerFunc) InTenantTx(ctx context.Context, tenant uuid.UUID, fn func(pgx.Tx) error) error {
	return f(ctx, tenant, fn)
}

type fixture struct {
	tenant, defID uuid.UUID
	actor         identity.Principal
}

func newFixture() fixture {
	tenant := uuid.New()
	return fixture{tenant: tenant, defID: uuid.New(), actor: identity.Principal{TenantID: tenant, PrincipalID: uuid.New()}}
}

func (f fixture) tx(t *testing.T, stmts ...stmt) *scriptTx {
	return &scriptTx{t: t, tenant: f.tenant.String(), stmts: append(stmts,
		stmt{match: sqlDefinition, scan: scanUUID(f.defID)},
		stmt{match: sqlShares, scan: func(dest ...any) error { *dest[0].(*[]uuid.UUID) = nil; return nil }})}
}

func (f fixture) create(data map[string]any) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpCreate, EntityKey: "asset", Actor: f.actor, Source: pipeline.SourceUser, Data: data}
}

func (f fixture) update(id uuid.UUID, patch map[string]any) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpUpdate, EntityKey: "asset", RecordID: id, ExpectedVersion: 1,
		Actor: f.actor, Source: pipeline.SourceUser, Data: patch}
}

func unitPipeline(a authz.Authorizer, defs pipeline.DefinitionReader) *pipeline.Pipeline {
	return pipeline.New(a, defs, pipeline.NoopSink{}, runnerFunc(func(context.Context, uuid.UUID, func(pgx.Tx) error) error {
		return errors.New("runner not expected")
	}), pipeline.SystemClock{})
}

// The tenant is the transaction's, never the command's: a transaction with
// no tenant, or a setting that is not a tenant id, is refused before any
// other statement, and a failed read is a wrapped database error.
func TestWriteTakesTheTenantFromTheTransaction(t *testing.T) {
	f := newFixture()
	p := unitPipeline(fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1})
	for name, tenant := range map[string]string{"none": "", "not a uuid": "tenant-a", "nil uuid": uuid.Nil.String()} {
		t.Run(name, func(t *testing.T) {
			tx := f.tx(t)
			tx.tenant = tenant
			_, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x"}))
			if !errors.Is(err, pipeline.ErrInvalidCommand) || !errors.Is(err, tenancy.ErrNoTenant) {
				t.Fatalf("err = %v, want ErrInvalidCommand and tenancy.ErrNoTenant", err)
			}
			if len(tx.ran) != 1 {
				t.Errorf("statements after the tenant read: %q", tx.ran[1:])
			}
		})
	}

	// A failed read of the tenant is a database error, not a refusal.
	tx := &readFailsTx{}
	_, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x"}))
	if !errors.Is(err, errBoom) || errors.Is(err, pipeline.ErrInvalidCommand) {
		t.Fatalf("err = %v, want the read's error and not ErrInvalidCommand", err)
	}
	if want := "pipeline: tenancy: read tenant: boom"; err.Error() != want {
		t.Errorf("message %q, want %q", err, want)
	}
}

type readFailsTx struct{ pgx.Tx }

func (readFailsTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return rowFunc(func(...any) error { return errBoom })
}

// An actor of tenant B is refused in tenant A's transaction, whatever it
// names: the tenant comes from the transaction, not from the command.
func TestWriteRefusesAnActorOfAnotherTenant(t *testing.T) {
	f := newFixture()
	p := unitPipeline(fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1})
	tx := f.tx(t)
	cmd := f.create(map[string]any{"name": "x"})
	cmd.Actor.TenantID = uuid.New()
	if _, err := p.Write(t.Context(), tx, cmd); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("err = %v, want authz.ErrForbidden", err)
	}
	if len(tx.ran) != 1 {
		t.Errorf("statements after the tenant read: %q", tx.ran[1:])
	}
}

// An entity type the reader does not have is ErrNotFound; any other failure
// to read it is a wrapped error, not ErrNotFound.
func TestWriteMapsDefinitionReadErrors(t *testing.T) {
	f := newFixture()
	cmd := f.create(map[string]any{"name": "x"})

	p := unitPipeline(fakeAuthz{}, fakeDefs{curErr: entitytypes.ErrNotFound})
	if _, err := p.Write(t.Context(), f.tx(t), cmd); !errors.Is(err, pipeline.ErrNotFound) {
		t.Fatalf("missing type: %v, want ErrNotFound", err)
	}

	p = unitPipeline(fakeAuthz{}, fakeDefs{curErr: errBoom})
	_, err := p.Write(t.Context(), f.tx(t), cmd)
	if !errors.Is(err, errBoom) || errors.Is(err, pipeline.ErrNotFound) {
		t.Fatalf("read failure: %v, want the cause and not ErrNotFound", err)
	}
	if err.Error() != "pipeline: boom" {
		t.Errorf("message %q", err)
	}
}

// A failed read of an idempotency key is a wrapped error, before anything is
// written.
func TestWriteReplayReadFailure(t *testing.T) {
	f := newFixture()
	p := unitPipeline(fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1})
	cmd := f.create(map[string]any{"name": "x"})
	cmd.Actor.PrincipalID, cmd.Source, cmd.IdempotencyKey = uuid.Nil, pipeline.SourceSystem, "sys:1"
	tx := f.tx(t, stmt{match: sqlWriteKey, err: errBoom})
	_, err := p.Write(t.Context(), tx, cmd)
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want the read's error", err)
	}
	if want := "pipeline: records/store: read write key: boom"; err.Error() != want {
		t.Errorf("message %q, want %q", err, want)
	}
	if tx.ranAny(sqlDefinition) {
		t.Error("the write went on after the failed key read")
	}
}

// Data that does not marshal to JSON is a caller bug, refused before any
// write and reported at the definition version the write used.
func TestWriteRefusesDataThatIsNotJSON(t *testing.T) {
	f := newFixture()
	p := unitPipeline(fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 3})
	tx := f.tx(t)
	_, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": make(chan int)}))
	if !errors.Is(err, pipeline.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidCommand", err)
	}
	if want := "pipeline: invalid command: data does not marshal to JSON"; err.Error() != want {
		t.Errorf("message %q, want %q", err, want)
	}
	var av *pipeline.ErrAtVersion
	if !errors.As(err, &av) || av.SchemaVersion != 3 || av.StateProperty != "status" {
		t.Errorf("err = %#v, want ErrAtVersion at version 3 naming status", err)
	}
	if tx.ranAny(sqlInsert) || tx.ranAny(sqlNumber) {
		t.Error("a statement wrote after the refusal")
	}
}

// Schema validation runs before any write: a create with no data fails on
// the required property, and nothing is inserted or numbered.
func TestCreateValidatesBeforeWriting(t *testing.T) {
	f := newFixture()
	p := unitPipeline(fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1})
	for name, data := range map[string]map[string]any{
		"nil data":   nil,
		"wrong type": {"name": "x", "tag": "seven"},
		"unknown":    {"name": "x", "colour": "red"},
	} {
		t.Run(name, func(t *testing.T) {
			tx := f.tx(t)
			_, err := p.Write(t.Context(), tx, f.create(data))
			var ve *pipeline.ErrValidation
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want *ErrValidation", err)
			}
			want := map[string][]metadata.FieldError{
				"nil data":   {{Path: "/name", Code: "required"}},
				"wrong type": {{Path: "/tag", Code: "type"}},
				"unknown":    {{Path: "/colour", Code: "additionalProperties"}},
			}[name]
			if !reflect.DeepEqual(ve.Fields, want) {
				t.Errorf("fields = %v, want %v", ve.Fields, want)
			}
			for _, s := range []string{sqlLock, sqlNumber, sqlInsert, sqlAudit} {
				if tx.ranAny(s) {
					t.Errorf("%q ran after a validation failure", s)
				}
			}
		})
	}
}

// A refused authorisation writes nothing, on a create and on an update.
func TestWriteRefusedByTheAuthorizer(t *testing.T) {
	f := newFixture()
	p := unitPipeline(fakeAuthz{err: authz.ErrForbidden}, fakeDefs{cur: assetDef(t), curVersion: 1})

	tx := f.tx(t)
	if _, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x"})); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("create: %v, want authz.ErrForbidden", err)
	}
	if tx.ranAny(sqlInsert) || tx.ranAny(sqlNumber) {
		t.Error("create: a statement wrote after the refusal")
	}

	rec := storedRecord{tenant: f.tenant, id: uuid.New(), def: f.defID, key: "asset", version: 1, schemaVersion: 1,
		state: "active", data: `{"name":"x"}`}
	tx = f.tx(t, stmt{match: sqlGet, scan: rec.scan})
	if _, err := p.Write(t.Context(), tx, f.update(rec.id, map[string]any{"name": "y"})); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("update: %v, want authz.ErrForbidden", err)
	}
	if tx.ranAny(sqlUpdate) || tx.ranAny(sqlAudit) {
		t.Error("update: a statement wrote after the refusal")
	}
}

// Each failing statement of a create is returned wrapped, naming its step,
// and stops the write there.
func TestCreateStatementFailures(t *testing.T) {
	f := newFixture()
	p := unitPipeline(fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1})
	ok := []stmt{
		{match: sqlLock}, {match: sqlTaken, scan: scanBool(false)}, {match: sqlNumber, scan: scanInt(1)},
		{match: sqlInsert}, {match: sqlPayload, scan: scanString(`{"name":"x","serial":"SN-1","status":"active"}`)},
		{match: sqlAudit},
	}
	cases := []struct {
		fail     string
		want     string
		notAfter string
	}{
		{sqlLock, "pipeline: records/store: lock unique value: boom", sqlTaken},
		{sqlTaken, "pipeline: records/store: check unique value: boom", sqlNumber},
		{sqlNumber, "pipeline: records/store: take record number: boom", sqlInsert},
		{sqlPayload, "pipeline: records/store: read payload as jsonb: boom", sqlAudit},
		{sqlAudit, "pipeline: audit: audit: insert entry: boom", ""},
	}
	for _, c := range cases {
		t.Run(c.fail, func(t *testing.T) {
			stmts := make([]stmt, 0, len(ok))
			for _, s := range ok {
				if s.match == c.fail {
					s.err = errBoom
				}
				stmts = append(stmts, s)
			}
			tx := f.tx(t, stmts...)
			_, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x", "serial": "SN-1"}))
			if !errors.Is(err, errBoom) {
				t.Fatalf("err = %v, want the statement's error", err)
			}
			if err.Error() != c.want {
				t.Errorf("message %q, want %q", err, c.want)
			}
			var av *pipeline.ErrAtVersion
			if !errors.As(err, &av) || av.SchemaVersion != 1 {
				t.Errorf("err = %#v, want ErrAtVersion at version 1", err)
			}
			if c.notAfter != "" && tx.ranAny(c.notAfter) {
				t.Errorf("%q ran after the failure", c.notAfter)
			}
		})
	}

	// And with every statement answering, the create succeeds.
	res, err := p.Write(t.Context(), f.tx(t, ok...), f.create(map[string]any{"name": "x", "serial": "SN-1"}))
	if err != nil || res.Number != "AST0000001" || res.Version != 1 {
		t.Fatalf("create = %+v, %v", res, err)
	}
}

// An update of a row written under an older version reads that version's
// definition; a failure to read it stops the write.
func TestUpdateStoredDefinitionReadFailure(t *testing.T) {
	f := newFixture()
	p := unitPipeline(fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 2, verErr: errBoom})
	rec := storedRecord{tenant: f.tenant, id: uuid.New(), def: f.defID, key: "asset", version: 1, schemaVersion: 1,
		state: "active", data: `{"name":"x"}`}
	tx := f.tx(t, stmt{match: sqlGet, scan: rec.scan})
	_, err := p.Write(t.Context(), tx, f.update(rec.id, map[string]any{"name": "y"}))
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want the read's error", err)
	}
	if want := "pipeline: read stored definition: boom"; err.Error() != want {
		t.Errorf("message %q, want %q", err, want)
	}
	if tx.ranAny(sqlUpdate) {
		t.Error("the record was updated after the failure")
	}
}

// An update that removes the state property fails validation with
// /status required, merged in path order into the schema's own errors and
// never listed twice.
func TestUpdateValidatesTheStateProperty(t *testing.T) {
	f := newFixture()
	rec := storedRecord{tenant: f.tenant, id: uuid.New(), def: f.defID, key: "asset", version: 1, schemaVersion: 1,
		state: "active", data: `{"name":"x","tag":1}`}
	cases := []struct {
		name     string
		required []string
		patch    map[string]any
		want     []metadata.FieldError
	}{
		{"only the state", nil, map[string]any{"status": nil},
			[]metadata.FieldError{{Path: "/status", Code: "required"}}},
		{"before a later path", nil, map[string]any{"status": nil, "tag": "seven"},
			[]metadata.FieldError{{Path: "/status", Code: "required"}, {Path: "/tag", Code: "type"}}},
		{"after an earlier path", nil, map[string]any{"status": nil, "name": ""},
			[]metadata.FieldError{{Path: "/name", Code: "minLength"}, {Path: "/status", Code: "required"}}},
		{"schema requires it too", []string{"status"}, map[string]any{"status": nil, "tag": "seven"},
			[]metadata.FieldError{{Path: "/status", Code: "required"}, {Path: "/tag", Code: "type"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := unitPipeline(fakeAuthz{}, fakeDefs{cur: assetDef(t, c.required...), curVersion: 1})
			tx := f.tx(t, stmt{match: sqlGet, scan: rec.scan})
			_, err := p.Write(t.Context(), tx, f.update(rec.id, c.patch))
			var ve *pipeline.ErrValidation
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want *ErrValidation", err)
			}
			if !reflect.DeepEqual(ve.Fields, c.want) {
				t.Errorf("fields = %v, want %v", ve.Fields, c.want)
			}
			if tx.ranAny(sqlUpdate) || tx.ranAny(sqlAudit) {
				t.Error("a statement wrote after a validation failure")
			}
		})
	}
}

// A write beyond the depth limit is refused with ErrCascadeTooDeep before
// any statement in tx; when its audit row cannot be written the error wraps
// that cause as well.
func TestTooDeepAuditFailure(t *testing.T) {
	f := newFixture()
	cmd := f.update(uuid.New(), map[string]any{"name": "y"})
	cmd.Actor.PrincipalID, cmd.Source = uuid.Nil, pipeline.SourceWorkflow
	cmd.Causation = outbox.Causation{OriginID: uuid.New(), Depth: 9}

	for name, runner := range map[string]runnerFunc{
		"runner fails": func(context.Context, uuid.UUID, func(pgx.Tx) error) error { return errBoom },
		"record read fails": func(ctx context.Context, tenant uuid.UUID, fn func(pgx.Tx) error) error {
			if tenant != f.tenant {
				t.Errorf("runner tenant %s, want the transaction's %s", tenant, f.tenant)
			}
			return fn(&scriptTx{t: t, tenant: tenant.String(), stmts: []stmt{{match: sqlGet, err: errBoom}}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := pipeline.New(fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
			tx := f.tx(t)
			_, err := p.Write(t.Context(), tx, cmd)
			if !errors.Is(err, outbox.ErrCascadeTooDeep) || !errors.Is(err, errBoom) {
				t.Fatalf("err = %v, want ErrCascadeTooDeep joined with the cause", err)
			}
			if !strings.Contains(err.Error(), "pipeline: audit the rejected cascade:") {
				t.Errorf("message %q does not name the failed audit", err)
			}
			if len(tx.ran) != 1 {
				t.Errorf("statements in tx beyond the tenant read: %q", tx.ran[1:])
			}
		})
	}
}

// AuditSuppressed returns a failed audit insert wrapped.
func TestAuditSuppressedInsertFailure(t *testing.T) {
	tenant, record, def, origin := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	tx := &scriptTx{t: t, tenant: tenant.String(), stmts: []stmt{
		{match: "FOR UPDATE", scan: func(dest ...any) error {
			*dest[0].(*uuid.UUID) = tenant
			*dest[1].(**uuid.UUID) = &origin
			d := int16(1)
			*dest[2].(**int16) = &d
			return nil
		}},
		{match: "INSERT INTO cascade_start", err: pgx.ErrNoRows},
		{match: sqlAudit, err: errBoom},
	}}
	c, claimed, err := cascade.ClaimStart(t.Context(), tx, cascade.Key{RecordID: record, DefinitionID: def})
	if err != nil || claimed {
		t.Fatalf("ClaimStart = %v, %v; want an unclaimed start", claimed, err)
	}
	err = pipeline.AuditSuppressed(t.Context(), tx, c, pipeline.Event{Source: pipeline.SourceWorkflow})
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want the insert's error", err)
	}
	if want := "pipeline: audit the suppressed start: audit: insert entry: boom"; err.Error() != want {
		t.Errorf("message %q, want %q", err, want)
	}
}

func TestRunnerIsTheConfiguredOne(t *testing.T) {
	calls := 0
	r := runnerFunc(func(context.Context, uuid.UUID, func(pgx.Tx) error) error { calls++; return nil })
	p := pipeline.New(fakeAuthz{}, nopDefs{}, pipeline.NoopSink{}, r, pipeline.SystemClock{})
	if err := p.Runner().InTenantTx(t.Context(), uuid.New(), nil); err != nil || calls != 1 {
		t.Fatalf("Runner().InTenantTx = %v after %d calls, want the configured runner", err, calls)
	}
}
