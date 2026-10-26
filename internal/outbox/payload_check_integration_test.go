//go:build integration

// T32c (TICK-214), CI-7 finding T32b-1: outbox_payload_references_check
// (db/migrations/00006_outbox_payload_references.sql) holds every outbox row
// to ValidatePayload's references-only rule, whoever writes it.

package outbox_test

import (
	"context"
	"encoding/json"
	"errors"
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
	payloadCheck    = "outbox_payload_references_check"
	checkViolation  = "23514"
	invalidJSONText = "22P02"
	recordDataText  = "CI-7 ticket title: printer on fire"
)

// payloadCases is the one table both enforcement points run over: the CI-7
// record-data shapes (TestCI7_ProducerSchemaRejectsRecordDataFields), every
// form uuid.Parse takes, and the edges where JSON decoding in Go and jsonb
// could part ways.
func payloadCases() []string {
	ref := "0190a3b2-1c2d-7e8f-9a0b-1c2d3e4f5a6b"
	return []string{
		// References only.
		`{}`,
		`{"record_id":"` + ref + `"}`,
		`{"watchers":["` + ref + `"],"record_id":"` + ref + `"}`,
		`{"watchers":[]}`,
		`{"record_id":"` + strings.ToUpper(ref) + `"}`,
		`{"record_id":"urn:uuid:` + ref + `"}`,
		`{"record_id":"URN:UUID:` + ref + `"}`,
		`{"record_id":"{` + ref + `}"}`,
		`{"record_id":"x` + ref + `!"}`, // uuid.Parse ignores both wrapping characters
		`{"record_id":"` + strings.ReplaceAll(ref, "-", "") + `"}`,
		`{"record_id":"\u0030190a3b2-1c2d-7e8f-9a0b-1c2d3e4f5a6b"}`, // JSON escape, decodes to ref
		`{"a":"` + ref + `","a":"` + ref + `"}`,                     // duplicate key, last wins in both
		`{"a":"free text","a":"` + ref + `"}`,
		// Record data and other shapes.
		`{"title":"` + recordDataText + `"}`,
		`{"record_id":"` + ref + `","note":"free text"}`,
		`{"amount":42}`,
		`{"approved":true}`,
		`{"customer":{"email":"alice@tenant-a.test"}}`,
		`{"watchers":["` + ref + `","bob@tenant-a.test"]}`,
		`{"record_id":"` + strings.ToUpper(ref) + ` "}`,
		`{"record_id":null}`,
		`{"watchers":[null]}`,
		`{"watchers":[["` + ref + `"]]}`,
		`{"watchers":[{"id":"` + ref + `"}]}`,
		`{"record_id":""}`,
		`{"record_id":"` + ref + `\n"}`,
		`{"record_id":"\n` + ref + `\n"}`, // 38 characters, both wraps ASCII: accepted by both
		`{"record_id":"é` + ref + `!"}`,
		`{"record_id":"urn:uuid:` + ref + `x"}`,
		`{"record_id":"urn:uuid:` + strings.ReplaceAll(ref, "-", "") + `"}`,
		`{"record_id":"` + strings.Replace(ref, "-", "", 1) + `-"}`,
		`{"record_id":"` + strings.ReplaceAll(ref, "-", "") + `0"}`,
		`{"record_id":"` + strings.Replace(ref, "a", "g", 1) + `"}`,
		`{"a":"` + ref + `","a":"free text"}`,
		`["` + ref + `"]`,
		`"` + ref + `"`,
		`null`,
		`42`,
		`not json`,
	}
}

// One rule, two enforcement points: for every case, a direct INSERT is
// refused by the CHECK exactly when ValidatePayload rejects the payload.
// The INSERT runs as the superuser, which skips RLS but not CHECKs, in a
// transaction that is rolled back.
func TestPayloadCheckMatchesValidatePayload(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	for _, p := range payloadCases() {
		goAccepts := outbox.ValidatePayload(json.RawMessage(p)) == nil
		tx, err := e.super.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, payload)
			VALUES (gen_random_uuid(), gen_random_uuid(), 'k', 'parity', 't', 1, $1::text::jsonb)`, p)
		_ = tx.Rollback(ctx)
		var pgErr *pgconn.PgError
		switch {
		case err == nil:
			if !goAccepts {
				t.Errorf("payload %s: the CHECK accepts it, ValidatePayload rejects it", p)
			}
		case errors.As(err, &pgErr) && pgErr.Code == checkViolation && pgErr.ConstraintName == payloadCheck,
			errors.As(err, &pgErr) && pgErr.Code == invalidJSONText:
			if goAccepts {
				t.Errorf("payload %s: ValidatePayload accepts it, the database refuses it (%v)", p, err)
			}
		default:
			t.Fatalf("payload %s: %v", p, err)
		}
	}
}

// A writer that skips Enqueue cannot store record data: a direct INSERT as
// nava_app or nava_worker, in its own tenant transaction, is refused, and a
// references-only one is not.
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

			err := insert("record-data", `{"record_id":"`+record.String()+`","title":"`+recordDataText+`"}`)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != checkViolation || pgErr.ConstraintName != payloadCheck {
				t.Fatalf("INSERT with record data as %s: %v, want a %s violation", c.role, err, payloadCheck)
			}
			if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant); n != 0 {
				t.Fatalf("%d rows stored after the refused INSERT", n)
			}

			if err := insert("refs", `{"record_id":"`+record.String()+`"}`); err != nil {
				t.Fatalf("references-only INSERT as %s: %v", c.role, err)
			}
		})
	}
}
