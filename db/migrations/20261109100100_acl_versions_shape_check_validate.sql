-- TICK-629: validates the previous migration's rules and masks CHECKs in
-- their own transaction. VALIDATE CONSTRAINT scans the table under SHARE
-- UPDATE EXCLUSIVE, so reads and publishes go on while it runs.
--
-- S10 has not shipped, so a deployed database has no acl_versions rows. A
-- long-lived database built from the story branch and fed by an old
-- isolation seed ('{}' rules) fails here, naming the constraint; rebuild it
-- (story-branch databases are rebuilt, db/migrations/README.md).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE acl_versions VALIDATE CONSTRAINT acl_versions_rules_check;
ALTER TABLE acl_versions VALIDATE CONSTRAINT acl_versions_masks_check;

-- +goose Down

-- Nothing: a constraint cannot be made NOT VALID again, and the previous
-- migration's Down drops it.
SELECT 1;
