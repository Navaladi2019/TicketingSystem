-- T32c (TICK-214), CI-7 finding T32b-1 (docs/security/temporal-ci6.md): an
-- outbox row's payload holds references only (ADR-001 R1), even when the
-- writer skips outbox.Enqueue. nava_app and nava_worker may INSERT every
-- outbox column and the relay forwards the payload unchecked, so this CHECK
-- is the lowest layer that keeps record data out of the Temporal history.
--
-- One rule, two enforcement points. The CHECK accepts exactly what
-- outbox.ValidatePayload accepts: a JSON object whose every value is a UUID
-- string or an array (possibly empty) of UUID strings. "UUID string" is what
-- github.com/google/uuid.Parse accepts, quirks included:
--   xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx      hex in either case
--   urn:uuid:xxxxxxxx-...                     prefix in either case
--   ?xxxxxxxx-...-xxxxxxxxxxxx?               any one ASCII character each
--                                             side (Parse never checks the
--                                             braces)
--   32 hex digits, no hyphens
-- Any change to either side must keep them equal:
-- TestPayloadCheckMatchesValidatePayload (internal/outbox) runs both over one
-- table of cases.
--
-- jsonb_path_exists and like_regex, so no function: nothing new to grant, and
-- nava_relay's UPDATEs, which re-evaluate every CHECK, need no EXECUTE. In
-- strict mode a value that is neither a matching string nor an array of
-- matching strings is selected, and one selected value fails the row.
--
-- Online: ADD CONSTRAINT ... NOT VALID holds ACCESS EXCLUSIVE only for a
-- catalog update (no scan; new rows are checked from then on), under a
-- lock_timeout so it never queues the relay and producers behind a long
-- transaction. VALIDATE then scans the existing rows under SHARE UPDATE
-- EXCLUSIVE, which blocks no reads or writes. Neither can share a
-- transaction with the other without keeping the strong lock for the scan, so
-- the file is NO TRANSACTION and each statement is safe to re-run
-- (TestNoTransactionMigrationsAreIdempotent).
--
-- An existing row that breaks the rule fails VALIDATE, and so the migration,
-- with "check constraint "outbox_payload_references_check" of relation
-- "outbox" is violated by some row"; no version row is written and the
-- constraint stays NOT VALID, still guarding new rows. Find the rows as the
-- superuser (FORCE RLS hides them from nava_migrator) with
--   SELECT tenant_id, id FROM outbox WHERE NOT (<the CHECK expression>);
-- mark them dead or delete them, then re-run the migrator. Never copy their
-- payload into a ticket or log: it is the record data this rule keeps out.
--
-- No index, no grant and no policy changes.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint
                   WHERE conrelid = 'public.outbox'::regclass
                     AND conname = 'outbox_payload_references_check') THEN
        SET LOCAL lock_timeout = '5s';
        ALTER TABLE outbox ADD CONSTRAINT outbox_payload_references_check CHECK (
            jsonb_typeof(payload) = 'object'
            AND NOT jsonb_path_exists(payload, 'strict $.* ? (
                !(@.type() == "string"
                  && @ like_regex "^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[uU][rR][nN]:[uU][uU][iI][dD]:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[[:ascii:]][0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}[[:ascii:]]|[0-9a-fA-F]{32})$")
                && !(@.type() == "array"
                     && !exists(@[*] ? (
                         !(@.type() == "string"
                           && @ like_regex "^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[uU][rR][nN]:[uU][uU][iI][dD]:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[[:ascii:]][0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}[[:ascii:]]|[0-9a-fA-F]{32})$")))))')
        ) NOT VALID;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE outbox VALIDATE CONSTRAINT outbox_payload_references_check;
