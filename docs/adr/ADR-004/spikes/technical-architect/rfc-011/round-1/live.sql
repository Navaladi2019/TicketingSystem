\set ON_ERROR_STOP 0
\echo '=== L: can the DB refuse a membership in a tombstoned group, and a tombstone with members left? ==='
CREATE TABLE g2 (tenant_id uuid NOT NULL, id uuid NOT NULL, deleted_at timestamptz,
  live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED,
  PRIMARY KEY (tenant_id, id), UNIQUE (tenant_id, id, live));
CREATE TABLE gm2 (tenant_id uuid NOT NULL, group_id uuid NOT NULL, principal_id uuid NOT NULL,
  live boolean NOT NULL DEFAULT true CHECK (live),
  PRIMARY KEY (tenant_id, group_id, principal_id),
  FOREIGN KEY (tenant_id, group_id, live) REFERENCES g2 (tenant_id, id, live));
INSERT INTO g2 (tenant_id, id) VALUES ('11111111-1111-1111-1111-111111111111','00000000-0000-0000-0000-0000000000a1'),
                                     ('11111111-1111-1111-1111-111111111111','00000000-0000-0000-0000-0000000000a2');
INSERT INTO gm2 (tenant_id, group_id, principal_id) VALUES ('11111111-1111-1111-1111-111111111111','00000000-0000-0000-0000-0000000000a1','00000000-0000-0000-0000-0000000000c1');
\echo '--- L1: tombstone a group that still has a member (expect FK error)'
UPDATE g2 SET deleted_at = now() WHERE id = '00000000-0000-0000-0000-0000000000a1';
\echo '--- L2: delete members then tombstone, one transaction (expect success)'
BEGIN;
DELETE FROM gm2 WHERE group_id = '00000000-0000-0000-0000-0000000000a1';
UPDATE g2 SET deleted_at = now() WHERE id = '00000000-0000-0000-0000-0000000000a1';
COMMIT;
\echo '--- L3: add a member to the tombstoned group (expect FK error)'
INSERT INTO gm2 (tenant_id, group_id, principal_id) VALUES ('11111111-1111-1111-1111-111111111111','00000000-0000-0000-0000-0000000000a1','00000000-0000-0000-0000-0000000000c2');
\echo '--- L4: revive and add (expect success)'
UPDATE g2 SET deleted_at = NULL WHERE id = '00000000-0000-0000-0000-0000000000a1';
INSERT INTO gm2 (tenant_id, group_id, principal_id) VALUES ('11111111-1111-1111-1111-111111111111','00000000-0000-0000-0000-0000000000a1','00000000-0000-0000-0000-0000000000c2');
SELECT * FROM gm2;
