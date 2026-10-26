-- T32f (TICK-240), CI-7 finding T32b-1 remainder (docs/security/temporal-ci6.md):
-- 00006 checks an outbox payload's values only. A key such as
-- "printer on fire, call alice@tenant-a.test" passed, and so did up to two
-- free characters per value through the forms uuid.Parse tolerates. This
-- constraint replaces outbox_payload_references_check with the stricter rule.
--
-- One rule, two enforcement points. The CHECK accepts exactly what
-- outbox.ValidatePayload accepts: a JSON object whose
--   - every key matches ^[a-z][a-z0-9_]{0,31}$, and
--   - every value is a canonical UUID string (36 characters, lowercase hex,
--     hyphenated) or an array (possibly empty) of them.
-- The key rule is the fallback the TICK-240 contract allows; why the per-kind
-- key allow-list was not workable is in docs/data/outbox-payload-check.md.
-- TestPayloadCheckMatchesValidatePayload (internal/outbox) runs both over one
-- table of cases.
--
-- jsonb_path_exists and like_regex, so no function: nothing new to grant, and
-- nava_relay's UPDATEs, which re-evaluate every CHECK, need no EXECUTE. CASE
-- keeps the path off a non-object, where strict keyvalue() would raise an
-- error instead of failing the CHECK. One keyvalue() item selected fails the
-- row: a bad key, or a value that is neither a canonical UUID string nor an
-- array of them.
--
-- Online, expand then contract, each statement safe to re-run
-- (TestNoTransactionMigrationsAreIdempotent):
--   1. ADD CONSTRAINT ... NOT VALID: ACCESS EXCLUSIVE for a catalog update
--      only (no scan), under lock_timeout; new rows are checked from then on.
--   2. VALIDATE: scans existing rows under SHARE UPDATE EXCLUSIVE, which
--      blocks no reads or writes.
--   3. DROP the 00006 constraint, now implied by this one: ACCESS EXCLUSIVE
--      for a catalog update, under lock_timeout. Dropping it halves the
--      per-row cost the relay's UPDATEs pay.
--
-- An existing row that breaks the rule fails VALIDATE, and so the migration,
-- with "check constraint "outbox_payload_check" of relation "outbox" is
-- violated by some row"; no version row is written, the 00006 constraint is
-- still there and the new one stays NOT VALID, still guarding new rows. Find
-- the rows as the superuser (FORCE RLS hides them from nava_migrator) with
--   SELECT tenant_id, id FROM outbox WHERE NOT (<the CHECK expression>);
-- mark them dead or delete them, then re-run the migrator. Never copy their
-- payload into a ticket or log: it may be the record data this rule keeps out.
--
-- No index, no grant and no policy changes.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint
                   WHERE conrelid = 'public.outbox'::regclass
                     AND conname = 'outbox_payload_check') THEN
        SET LOCAL lock_timeout = '5s';
        ALTER TABLE outbox ADD CONSTRAINT outbox_payload_check CHECK (
            CASE WHEN jsonb_typeof(payload) = 'object' THEN
                NOT jsonb_path_exists(payload, 'strict $.keyvalue() ? (
                    !(@.key like_regex "^[a-z][a-z0-9_]{0,31}$")
                    || !((@.value.type() == "string"
                          && @.value like_regex "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
                         || (@.value.type() == "array"
                             && !exists(@.value[*] ? (
                                 !(@.type() == "string"
                                   && @ like_regex "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"))))))')
            ELSE false
            END
        ) NOT VALID;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE outbox VALIDATE CONSTRAINT outbox_payload_check;

-- +goose StatementBegin
DO $$
BEGIN
    SET LOCAL lock_timeout = '5s';
    ALTER TABLE outbox DROP CONSTRAINT IF EXISTS outbox_payload_references_check;
END
$$;
-- +goose StatementEnd
