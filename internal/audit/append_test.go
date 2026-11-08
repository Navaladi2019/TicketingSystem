package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/secrets"
)

// fakeTx records every statement. tenant is what app.tenant_id reads as ("" for
// none); execErr is returned by Exec.
type fakeTx struct {
	pgx.Tx
	tenant  string
	execErr error
	sql     []string
	args    []any
}

func (f *fakeTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.sql = append(f.sql, sql)
	f.args = args
	return pgconn.NewCommandTag("INSERT 0 1"), f.execErr
}

func (f *fakeTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	f.sql = append(f.sql, sql)
	return fakeRow{f.tenant}
}

func (f *fakeTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	f.sql = append(f.sql, sql)
	return nil, errors.New("fakeTx: Query not expected")
}

type fakeRow struct{ s string }

func (r fakeRow) Scan(dest ...any) error {
	*(dest[0].(*string)) = r.s
	return nil
}

func ptr[T any](v T) *T { return &v }

func testTenant() uuid.UUID    { return uuid.MustParse("0190f3a0-0000-7000-8000-00000000000b") }
func testPrincipal() uuid.UUID { return uuid.MustParse("0190f3a0-0000-7000-8000-0000000000a1") }

// validEntry is a record update with a small diff.
func validEntry() Entry {
	return Entry{
		Domain: DomainRecord, Action: ActionUpdate, SubjectType: SubjectRecord,
		SubjectID:    uuid.MustParse("0190f3a0-0000-7000-8000-0000000000d2"),
		VersionAfter: ptr[int64](2),
		Actor:        Actor{PrincipalID: ptr(testPrincipal())},
		Source:       SourceUser,
		Diff:         []byte(`[{"op":"replace","path":"/status","value":"open"}]`),
	}
}

func erasedIDs(n int) []byte {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = uuid.NewString()
	}
	b, _ := json.Marshal(ids)
	return []byte(`[{"op":"add","path":"/erased_ids","value":` + string(b) + `}]`)
}

// Invalid entries fail with ErrInvalidEntry before the transaction is used.
func TestAppendInvalidEntryRunsNoSQL(t *testing.T) {
	cases := map[string]func(e *Entry){
		"two actors":               func(e *Entry) { e.Actor.System = ptr("workflow") },
		"no actor":                 func(e *Entry) { e.Actor = Actor{} },
		"unknown action":           func(e *Entry) { e.Action = "frobnicate" },
		"empty action":             func(e *Entry) { e.Action = "" },
		"unknown domain":           func(e *Entry) { e.Domain = "billing" },
		"unknown source":           func(e *Entry) { e.Source = "cron" },
		"unknown subject type":     func(e *Entry) { e.SubjectType = "ticket" },
		"nil subject":              func(e *Entry) { e.SubjectID = uuid.Nil },
		"nil principal":            func(e *Entry) { e.Actor.PrincipalID = &uuid.Nil },
		"nil change set":           func(e *Entry) { e.ChangeSetID = &uuid.Nil },
		"negative causation depth": func(e *Entry) { e.CausationDepth = -1 },
		"config without content":   func(e *Entry) { e.Domain, e.SubjectType = DomainConfig, SubjectEntityDefinition },
		"ai without source_ref":    func(e *Entry) { e.Source = SourceAI },
		"ai source_ref not a uuid": func(e *Entry) { e.Source, e.SourceRef = SourceAI, ptr("decision-7") },
		"ai source_ref uppercase":  func(e *Entry) { e.Source, e.SourceRef = SourceAI, ptr("0190F3A0-0000-7000-8000-0000000000C1") },
		"empty request_id":         func(e *Entry) { e.RequestID = ptr("") },
		"empty source_ref":         func(e *Entry) { e.SourceRef = ptr("") },
		"NUL in actor_system":      func(e *Entry) { e.Actor = Actor{System: ptr("a\x00b")} },
		"invalid UTF-8 request_id": func(e *Entry) { e.RequestID = ptr("\xff") },
		"version past 2^53-1":      func(e *Entry) { e.VersionAfter = ptr[int64](chain.MaxSafeInteger + 1) },
		"diff not a patch":         func(e *Entry) { e.Diff = []byte(`{"op":"add"}`) },
		"empty diff":               func(e *Entry) { e.Diff = []byte{} },
		"bad secret path":          func(e *Entry) { e.SecretPaths = []patchwalk.Pointer{"no-slash"} },
		"erase without diff":       func(e *Entry) { eraseEntry(e); e.Diff = nil },
		"erase with 0 ids":         func(e *Entry) { eraseEntry(e); e.Diff = erasedIDs(0) },
		"erase with 1,501 ids":     func(e *Entry) { eraseEntry(e); e.Diff = erasedIDs(MaxErasedIDs + 1) },
		"erase with two ops": func(e *Entry) {
			eraseEntry(e)
			e.Diff = []byte(`[{"op":"add","path":"/erased_ids","value":[]},{"op":"remove","path":"/x"}]`)
		},
		"erase with another path": func(e *Entry) {
			eraseEntry(e)
			e.Diff = []byte(`[{"op":"add","path":"/ids","value":["0190f3a0-0000-7000-8000-000000000106"]}]`)
		},
		"erase id uppercase": func(e *Entry) {
			eraseEntry(e)
			e.Diff = []byte(`[{"op":"add","path":"/erased_ids","value":["0190F3A0-0000-7000-8000-000000000106"]}]`)
		},
		"erase id not a string":      func(e *Entry) { eraseEntry(e); e.Diff = []byte(`[{"op":"add","path":"/erased_ids","value":[7]}]`) },
		"erase over 64 KB untrimmed": func(e *Entry) { eraseEntry(e); e.Diff = padded(erasedIDs(1), MaxDiffBytes+1) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := validEntry()
			mutate(&e)
			tx := &fakeTx{tenant: testTenant().String()}
			id, err := Append(context.Background(), tx, e)
			if !errors.Is(err, ErrInvalidEntry) {
				t.Fatalf("err = %v, want ErrInvalidEntry", err)
			}
			if id != uuid.Nil {
				t.Errorf("id = %s, want nil", id)
			}
			if len(tx.sql) != 0 {
				t.Errorf("ran %d statements before rejecting: %q", len(tx.sql), tx.sql)
			}
		})
	}
}

func eraseEntry(e *Entry) {
	e.Domain, e.Action, e.SubjectType = DomainAudit, ActionErase, SubjectErasureRequest
	e.Actor = Actor{System: ptr("erasure")}
	e.Source = SourceSystem
	e.VersionAfter = nil
}

// padded inserts spaces after the opening "[" until diff is n bytes long.
func padded(diff []byte, n int) []byte {
	return append(append([]byte("["), bytes.Repeat([]byte(" "), n-len(diff))...), diff[1:]...)
}

func TestAppendValidEntryKinds(t *testing.T) {
	cases := map[string]func(e *Entry){
		"config with content": func(e *Entry) {
			e.Domain, e.SubjectType, e.ContentSHA256 = DomainConfig, SubjectEntityDefinition, &[32]byte{1}
		},
		"config delete": func(e *Entry) {
			e.Domain, e.SubjectType, e.Action = DomainConfig, SubjectEntityDefinition, ActionDelete
		},
		"ai with source_ref":   func(e *Entry) { e.Source, e.SourceRef = SourceAI, ptr("0190f3a0-0000-7000-8000-0000000000c1") },
		"system actor":         func(e *Entry) { e.Actor = Actor{System: ptr("workflow")} },
		"no diff":              func(e *Entry) { e.Diff = nil },
		"erase with 1,500 ids": func(e *Entry) { eraseEntry(e); e.Diff = erasedIDs(MaxErasedIDs) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := validEntry()
			mutate(&e)
			if _, err := Append(context.Background(), &fakeTx{tenant: testTenant().String()}, e); err != nil {
				t.Fatalf("Append: %v", err)
			}
		})
	}
}

// A 1,500-id erase entry stays under the limit, so it is never cut
// (ADR-004-E1: at most 58,545 bytes).
func TestEraseCapFitsUnderLimit(t *testing.T) {
	if n := len(erasedIDs(MaxErasedIDs)); n > MaxDiffBytes || n != 58545 {
		t.Fatalf("a %d-id erase diff is %d bytes, want 58,545", MaxErasedIDs, n)
	}
}

// With no app.tenant_id the entry is refused and nothing is inserted (I2).
func TestAppendNoTenant(t *testing.T) {
	for _, tenant := range []string{"", "not-a-uuid", uuid.Nil.String()} {
		tx := &fakeTx{tenant: tenant}
		_, err := Append(context.Background(), tx, validEntry())
		if !errors.Is(err, ErrInvalidEntry) {
			t.Fatalf("tenant %q: err = %v, want ErrInvalidEntry", tenant, err)
		}
		for _, s := range tx.sql {
			if strings.Contains(strings.ToUpper(s), "INSERT") {
				t.Fatalf("tenant %q: an INSERT ran: %q", tenant, s)
			}
		}
	}
}

const canary = "canary-7f3e-Sup3rS3cret"

// A secret is rejected with audit_secret_in_diff, nothing runs, and neither
// the error nor its message carries the value (CI-A5, unit half).
func TestAppendSecretInDiff(t *testing.T) {
	big := strings.Repeat("x", MaxDiffBytes)
	cases := map[string][]byte{
		"top-level":    []byte(`[{"op":"replace","path":"/api_key","value":"` + canary + `"}]`),
		"nested value": []byte(`[{"op":"add","path":"/creds","value":{"api_key":"` + canary + `"}}]`),
		// The second op is dropped by the 64 KB cut; the walk sees it first.
		"in a dropped op": []byte(`[{"op":"add","path":"/notes","value":"` + big + `"},{"op":"add","path":"/api_key","value":"` + canary + `"}]`),
		"url userinfo":    []byte(`[{"op":"add","path":"/hook","value":"https://u:` + canary + `@h.example"}]`),
	}
	for name, diff := range cases {
		t.Run(name, func(t *testing.T) {
			e := validEntry()
			e.Diff = diff
			e.SecretPaths = []patchwalk.Pointer{"/api_key", "/creds/api_key"}
			tx := &fakeTx{tenant: testTenant().String()}
			_, err := Append(context.Background(), tx, e)
			var se *ErrSecretInDiff
			if !errors.As(err, &se) {
				t.Fatalf("err = %v, want *ErrSecretInDiff", err)
			}
			if se.Code() != "audit_secret_in_diff" {
				t.Errorf("code = %q", se.Code())
			}
			if strings.Contains(err.Error(), canary) || strings.Contains(fmt.Sprintf("%+v", se), canary) ||
				strings.Contains(se.LogValue().String(), canary) {
				t.Errorf("the error carries the secret")
			}
			if len(tx.sql) != 0 {
				t.Errorf("ran %d statements", len(tx.sql))
			}
		})
	}
	// secrets.Domain and audit.Domain must agree, or the config-only
	// secret-shape scanner would never run.
	for _, d := range []Domain{DomainRecord, DomainConfig, DomainAudit} {
		if err := secrets.Check([]byte(`[]`), nil, secrets.Domain(d)); err != nil {
			t.Errorf("domain %q unknown to secrets: %v", d, err)
		}
	}
}

// The INSERT carries the transaction's tenant, a minted UUIDv7, the salt and
// diff_sha256 over the stored bytes, and none of the R10 database columns.
func TestAppendInsert(t *testing.T) {
	salt := [16]byte{0x5a, 0x17}
	defer func(f func([]byte)) { readSalt = f }(readSalt)
	readSalt = func(b []byte) { copy(b, salt[:]) }

	tx := &fakeTx{tenant: testTenant().String()}
	e := validEntry()
	id, err := Append(context.Background(), tx, e)
	if err != nil {
		t.Fatal(err)
	}
	if id.Version() != 7 {
		t.Errorf("minted id %s is version %d, want 7", id, id.Version())
	}
	if len(tx.sql) != 2 {
		t.Fatalf("statements = %q, want the tenant read then the insert", tx.sql)
	}
	ins := tx.sql[1]
	for _, col := range []string{"occurred_at", "txid", "backfilled", "seq", "prev_hash", "entry_hash", "sealed_at", "erased_at", "erasure_ref"} {
		if strings.Contains(ins, col) {
			t.Errorf("the insert names %s (R10)", col)
		}
	}
	if tx.args[0] != testTenant() || tx.args[1] != id {
		t.Errorf("tenant, id = %v, %v", tx.args[0], tx.args[1])
	}
	want := chain.DiffSHA256(salt, e.Diff)
	if !bytes.Equal(tx.args[18].([]byte), want[:]) || tx.args[19] != false ||
		*tx.args[20].(*string) != string(e.Diff) || !bytes.Equal(tx.args[21].([]byte), salt[:]) {
		t.Errorf("diff_sha256, diff_truncated, diff, diff_salt = %x, %v, %v, %x", tx.args[18], tx.args[19], tx.args[20], tx.args[21])
	}

	// A caller-supplied id is kept.
	e.ID = uuid.MustParse("0190f3a0-0000-7000-8000-000000000999")
	if got, err := Append(context.Background(), &fakeTx{tenant: testTenant().String()}, e); err != nil || got != e.ID {
		t.Errorf("Append with id = %v, %v; want %v", got, err, e.ID)
	}

	// No diff: diff, salt and diff_sha256 are all NULL.
	e.Diff = nil
	tx = &fakeTx{tenant: testTenant().String()}
	if _, err := Append(context.Background(), tx, e); err != nil {
		t.Fatal(err)
	}
	if tx.args[18].([]byte) != nil || tx.args[20].(*string) != nil || tx.args[21].([]byte) != nil {
		t.Errorf("no diff stored %v, %v, %v; want NULLs", tx.args[18], tx.args[20], tx.args[21])
	}
}

func TestAppendDuplicateAndInsertErrors(t *testing.T) {
	dup := &pgconn.PgError{Code: "23505", ConstraintName: "audit_log_default_commit_key"}
	_, err := Append(context.Background(), &fakeTx{tenant: testTenant().String(), execErr: dup}, validEntry())
	if !errors.Is(err, ErrDuplicateEntry) {
		t.Errorf("commit-key violation: err = %v, want ErrDuplicateEntry", err)
	}
	pk := &pgconn.PgError{Code: "23505", ConstraintName: "audit_log_default_pkey"}
	_, err = Append(context.Background(), &fakeTx{tenant: testTenant().String(), execErr: pk}, validEntry())
	if errors.Is(err, ErrDuplicateEntry) || !errors.As(err, new(*pgconn.PgError)) {
		t.Errorf("pkey violation: err = %v, want the wrapped PgError only", err)
	}
}

// op is one RFC 6902 add of a string value of n bytes, so the op is
// len(prefix)+n+2 bytes.
func op(path string, n int) string {
	return `{"op":"add","path":"` + path + `","value":"` + strings.Repeat("v", n) + `"}`
}

func TestTruncateDiff(t *testing.T) {
	join := func(ops ...string) []byte { return []byte("[" + strings.Join(ops, ",") + "]") }
	size := func(ops ...string) int { return len(join(ops...)) }

	// Under and at the limit: stored unchanged, never re-encoded.
	spaced := []byte(`[ {"op": "add", "path": "/a", "value": 1} ]`)
	if got, cut, err := truncateDiff(spaced); err != nil || cut || !bytes.Equal(got, spaced) {
		t.Errorf("small diff: %q, %v, %v", got, cut, err)
	}
	a := op("/a", 100)
	atLimit := join(a, op("/b", MaxDiffBytes-size(a, op("/b", 0))))
	if len(atLimit) != MaxDiffBytes {
		t.Fatalf("fixture is %d bytes", len(atLimit))
	}
	if got, cut, _ := truncateDiff(atLimit); cut || !bytes.Equal(got, atLimit) {
		t.Errorf("a diff of exactly %d bytes was cut", MaxDiffBytes)
	}

	// One byte over: the last op goes, the first stays byte for byte.
	b := op("/b", MaxDiffBytes-size(a, op("/b", 0))+1)
	got, cut, err := truncateDiff(join(a, b))
	if err != nil || !cut || !bytes.Equal(got, join(a)) {
		t.Errorf("one over: %q, %v, %v", got, cut, err)
	}

	// The cut keeps k ops when k fit exactly, and the raw bytes of each,
	// whitespace inside an op included.
	c := `{ "op" : "remove", "path" : "/c" }`
	fill := op("/f", MaxDiffBytes-size(a, c, op("/f", 0)))
	got, cut, _ = truncateDiff(join(a, c, fill, op("/z", 10)))
	if !cut || !bytes.Equal(got, join(a, c, fill)) || len(got) != MaxDiffBytes {
		t.Errorf("exact fit: %d bytes, cut %v", len(got), cut)
	}

	// The first op alone is over the limit: "[]", flagged.
	got, cut, _ = truncateDiff(join(op("/big", MaxDiffBytes), a))
	if !cut || string(got) != "[]" {
		t.Errorf("first op too big: %q, %v", got, cut)
	}

	// Over the limit only through whitespace between ops: every op is kept,
	// so the flag is false (k = n), though the bytes are rejoined.
	loose := []byte("[" + a + strings.Repeat(" ", MaxDiffBytes) + "," + c + "]")
	got, cut, _ = truncateDiff(loose)
	if cut || !bytes.Equal(got, join(a, c)) {
		t.Errorf("whitespace only: %q, %v", got, cut)
	}
}

// Append stores the cut, flags it, and hashes the stored bytes.
func TestAppendTruncates(t *testing.T) {
	defer func(f func([]byte)) { readSalt = f }(readSalt)
	readSalt = func(b []byte) { clear(b) }

	a := op("/a", 10)
	e := validEntry()
	e.Diff = []byte("[" + a + "," + op("/b", MaxDiffBytes) + "]")
	tx := &fakeTx{tenant: testTenant().String()}
	if _, err := Append(context.Background(), tx, e); err != nil {
		t.Fatal(err)
	}
	stored := "[" + a + "]"
	want := chain.DiffSHA256([16]byte{}, []byte(stored))
	if tx.args[19] != true || *tx.args[20].(*string) != stored || !bytes.Equal(tx.args[18].([]byte), want[:]) {
		t.Errorf("truncated insert: flag %v, diff %q", tx.args[19], *tx.args[20].(*string))
	}
}
