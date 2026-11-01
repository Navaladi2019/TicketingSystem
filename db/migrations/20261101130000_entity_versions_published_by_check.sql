-- P3-06c (TICK-383): every version after v1 names who published it, in the
-- database and not only in entitytypes.Publish. Version 1 alone may have
-- published_by NULL: onboarding seeds it (P3-06, TICK-344) while the tenant is
-- provisioning and has no principal yet. A v1 published through the API
-- names its publisher anyway; the CHECK does not require it to.
--
-- Expand step: the CHECK is added NOT VALID, a catalogue change under a short
-- ACCESS EXCLUSIVE lock with no scan, and every new row is checked from here
-- on. 20261101130100 validates it under SHARE UPDATE EXCLUSIVE, which blocks
-- neither reads nor publishes, the same split as principals_tombstone_check
-- (00010/00011). The table only grows by one row per publish, so the scan is
-- small, but the split keeps the long lock out of the migration whatever the
-- size. lock_timeout makes the migration fail and be re-run instead of
-- queueing reads behind a long publish transaction. No index, no grant
-- change.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions ADD CONSTRAINT entity_definition_versions_published_by_check
    CHECK (published_by IS NOT NULL OR version = 1) NOT VALID;
