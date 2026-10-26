//go:build integration

// T32c (TICK-214) and T32f (TICK-240), CI-7 finding T32b-1:
// outbox_payload_check (db/migrations/00007_outbox_payload_keys.sql) holds
// every outbox row to ValidatePayload's references-only rule, keys included,
// whoever writes it.

package outbox_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const (
	payloadCheck    = "outbox_payload_check"
	checkViolation  = "23514"
	invalidJSONText = "22P02"
	// jsonb refuses \u0000 in a string or a key before any CHECK runs.
	untranslatable = "22P05"
	recordDataText = "CI-7 ticket title: printer on fire"
)

// payloadCase is one payload and what both checks must do with it. A refused
// payload is a violation of payloadCheck in the database, unless dbCode names
// the error jsonb input raises first: there the database refuses for a reason
// that is not the CHECK, and the case says so instead of the test guessing.
type payloadCase struct {
	name    string
	payload string
	accept  bool
	dbCode  string
}

// payloadCases is the one table both enforcement points run over: the CI-7
// record-data shapes (TestCI7_ProducerSchemaRejectsRecordDataFields), the
// key channel and every non-canonical form uuid.Parse takes (TICK-240), and
// the edges where JSON decoding in Go and jsonb could part ways.
func payloadCases() []payloadCase {
	ref := "0190a3b2-1c2d-7e8f-9a0b-1c2d3e4f5a6b"
	ok := func(name, p string) payloadCase { return payloadCase{name: name, payload: p, accept: true} }
	bad := func(name, p string) payloadCase { return payloadCase{name: name, payload: p} }
	return []payloadCase{
		// References only.
		ok("empty object", `{}`),
		ok("one reference", `{"record_id":"`+ref+`"}`),
		ok("reference and list", `{"watchers":["`+ref+`"],"record_id":"`+ref+`"}`),
		ok("empty list", `{"watchers":[]}`),
		ok("escaped digit decodes to canonical", `{"record_id":"0190a3b2-1c2d-7e8f-9a0b-1c2d3e4f5a6b"}`),
		ok("escaped key decodes to an allowed key", `{"record_id":"`+ref+`"}`),
		ok("duplicate key, last wins", `{"a":"`+ref+`","a":"`+ref+`"}`),
		ok("duplicate key, text then reference", `{"a":"free text","a":"`+ref+`"}`),
		ok("32-character key", `{"`+strings.Repeat("a", 32)+`":"`+ref+`"}`),
		ok("digits and underscores after the first letter", `{"r2_d2_":"`+ref+`"}`),

		// Keys (TICK-240).
		bad("free-text key", `{"printer on fire":"`+ref+`"}`),
		bad("free-text key with an email", `{"printer on fire, call alice@tenant-a.test":"`+ref+`"}`),
		bad("key with a dot", `{"ticket.id":"`+ref+`"}`),
		bad("key with a capital", `{"Record_id":"`+ref+`"}`),
		bad("key with a hyphen", `{"record-id":"`+ref+`"}`),
		bad("key with a leading digit", `{"1record":"`+ref+`"}`),
		bad("key with a leading underscore", `{"_record":"`+ref+`"}`),
		bad("empty key", `{"":"`+ref+`"}`),
		bad("33-character key", `{"`+strings.Repeat("a", 33)+`":"`+ref+`"}`),
		bad("key with a trailing newline", `{"record_id\n":"`+ref+`"}`),
		bad("key with a non-ASCII letter", `{"récord_id":"`+ref+`"}`),
		bad("bad key next to a good one", `{"record_id":"`+ref+`","alice at tenant a":"`+ref+`"}`),
		{name: "key with NUL", payload: `{"a\u0000":"` + ref + `"}`, dbCode: untranslatable},

		// Non-canonical values (TICK-240): uuid.Parse takes every one.
		bad("uppercase", `{"record_id":"`+strings.ToUpper(ref)+`"}`),
		bad("one uppercase digit", `{"record_id":"`+strings.Replace(ref, "a", "A", 1)+`"}`),
		bad("urn:uuid:", `{"record_id":"urn:uuid:`+ref+`"}`),
		bad("URN:UUID:", `{"record_id":"URN:UUID:`+ref+`"}`),
		bad("38 characters, braces", `{"record_id":"{`+ref+`}"}`),
		bad("38 characters, any wrap", `{"record_id":"x`+ref+`!"}`),
		bad("38 characters, newlines", `{"record_id":"\n`+ref+`\n"}`),
		bad("32 hex digits", `{"record_id":"`+strings.ReplaceAll(ref, "-", "")+`"}`),
		bad("uppercase in a list", `{"watchers":["`+ref+`","`+strings.ToUpper(ref)+`"]}`),
		bad("urn:uuid: in a list", `{"watchers":["urn:uuid:`+ref+`"]}`),

		// Record data and other shapes.
		bad("title", `{"title":"`+recordDataText+`"}`),
		bad("text next to a reference", `{"record_id":"`+ref+`","note":"free text"}`),
		bad("number", `{"amount":42}`),
		bad("bool", `{"approved":true}`),
		bad("nested object", `{"customer":{"email":"alice@tenant-a.test"}}`),
		bad("email in a list", `{"watchers":["`+ref+`","bob@tenant-a.test"]}`),
		bad("uppercase with a trailing space", `{"record_id":"`+strings.ToUpper(ref)+` "}`),
		bad("null value", `{"record_id":null}`),
		bad("null element", `{"watchers":[null]}`),
		bad("nested list", `{"watchers":[["`+ref+`"]]}`),
		bad("object element", `{"watchers":[{"id":"`+ref+`"}]}`),
		bad("empty string", `{"record_id":""}`),
		bad("trailing newline", `{"record_id":"`+ref+`\n"}`),
		bad("non-ASCII wrap", `{"record_id":"é`+ref+`!"}`),
		bad("urn:uuid: with a trailing character", `{"record_id":"urn:uuid:`+ref+`x"}`),
		bad("urn:uuid: and 32 digits", `{"record_id":"urn:uuid:`+strings.ReplaceAll(ref, "-", "")+`"}`),
		bad("hyphen moved", `{"record_id":"`+strings.Replace(ref, "-", "", 1)+`-"}`),
		bad("33 hex digits", `{"record_id":"`+strings.ReplaceAll(ref, "-", "")+`0"}`),
		bad("non-hex digit", `{"record_id":"`+strings.Replace(ref, "a", "g", 1)+`"}`),
		bad("duplicate key, reference then text", `{"a":"`+ref+`","a":"free text"}`),
		{name: "value with NUL", payload: `{"record_id":"\u0000"}`, dbCode: untranslatable},
		bad("array", `["`+ref+`"]`),
		bad("string", `"`+ref+`"`),
		bad("null", `null`),
		bad("number payload", `42`),
		{name: "not JSON", payload: `not json`, dbCode: invalidJSONText},
	}
}

// One rule, two enforcement points: for every case, ValidatePayload and a
// direct INSERT both do what the case says, so they agree. The INSERT runs
// as the superuser, which skips RLS but not CHECKs, in a transaction that is
// rolled back.
func TestPayloadCheckMatchesValidatePayload(t *testing.T) {
	e := migrated(t)
	for i, c := range payloadCases() {
		t.Run(fmt.Sprintf("%02d %s", i, c.name), func(t *testing.T) {
			ctx := context.Background()
			if err := outbox.ValidatePayload(json.RawMessage(c.payload)); (err == nil) != c.accept {
				t.Errorf("ValidatePayload(%s) = %v, want accept %v", c.payload, err, c.accept)
			}
			tx, err := e.super.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			_, err = tx.Exec(ctx, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, payload)
				VALUES (gen_random_uuid(), gen_random_uuid(), 'k', 'parity', 't', 1, $1::text::jsonb)`, c.payload)
			_ = tx.Rollback(ctx)
			var pgErr *pgconn.PgError
			switch {
			case c.accept:
				if err != nil {
					t.Errorf("INSERT %s: %v, want it stored", c.payload, err)
				}
			case c.dbCode != "":
				if !errors.As(err, &pgErr) || pgErr.Code != c.dbCode {
					t.Errorf("INSERT %s: %v, want SQLSTATE %s from jsonb input", c.payload, err, c.dbCode)
				}
			default:
				if !errors.As(err, &pgErr) || pgErr.Code != checkViolation || pgErr.ConstraintName != payloadCheck {
					t.Errorf("INSERT %s: %v, want a %s violation", c.payload, err, payloadCheck)
				}
			}
		})
	}
}

// A writer that skips Enqueue cannot store record data, in a value or in a
// key: a direct INSERT as nava_app or nava_worker, in its own tenant
// transaction, is refused, and a references-only one is not.
func TestPayloadCheckRefusesDirectInsert(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	for _, c := range []struct {
		role string
		pool *pgxpool.Pool
	}{{"nava_app", e.app}, {"nava_worker", e.worker}} {
		t.Run(c.role, func(t *testing.T) {
			tenant, record := uuid.New(), uuid.New()
			seedRecord(t, e.super, tenant, record)
			insert := func(dedupe, payload string) error {
				return tenancy.InTenantTx(ctx, c.pool, tenant, func(tx pgx.Tx) error {
					_, err := tx.Exec(ctx, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, payload)
						VALUES (gen_random_uuid(), $1, $2, $3, $4, 1, $5::text::jsonb)`,
						tenant, approval, dedupe, approval+":"+record.String(), payload)
					return err
				})
			}

			for name, payload := range map[string]string{
				"record-data":   `{"record_id":"` + record.String() + `","title":"` + recordDataText + `"}`,
				"free-text-key": `{"printer on fire, call alice@tenant-a.test":"` + record.String() + `"}`,
				"uppercase-ref": `{"record_id":"` + strings.ToUpper(record.String()) + `"}`,
			} {
				err := insert(name, payload)
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != checkViolation || pgErr.ConstraintName != payloadCheck {
					t.Fatalf("INSERT %s as %s: %v, want a %s violation", name, c.role, err, payloadCheck)
				}
				if strings.Contains(err.Error(), "printer") || strings.Contains(err.Error(), "alice") {
					t.Fatalf("INSERT %s as %s: the error quotes the payload: %v", name, c.role, err)
				}
			}
			if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant); n != 0 {
				t.Fatalf("%d rows stored after the refused INSERTs", n)
			}

			if err := insert("refs", `{"record_id":"`+record.String()+`"}`); err != nil {
				t.Fatalf("references-only INSERT as %s: %v", c.role, err)
			}
		})
	}
}
