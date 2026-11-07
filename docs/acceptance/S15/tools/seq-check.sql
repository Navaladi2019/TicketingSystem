-- S15 acceptance (TICK-443): the per-tenant chain check after a worker kill.
-- Run as the bootstrap superuser (row level security does not apply), with
-- every entry sealed:
--
--   docker compose exec -T postgres sh -c \
--     'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava -v ON_ERROR_STOP=1 -f -' \
--     < docs/acceptance/S15/tools/seq-check.sql
--
-- One row per tenant with a chain head. ok is true only when:
--   unsealed = 0                every entry has a seq
--   dup_seqs = 0                no seq is used twice (R15; seq has no unique key)
--   missing_seqs = 0            seq runs 1..max_seq with nothing skipped
--   broken_links = 0            every entry's prev_hash is the entry_hash of seq - 1
--                               (32 zero bytes for seq 1, R13)
--   head_seq = max_seq          audit_chain_head.seq is the last seq handed out
--   head_hash_ok                audit_chain_head.entry_hash is that entry's hash
-- It does not recompute entry_hash from the envelope: that is the offline
-- verifier's job (cmd/nava verify, on an export, S16).
\pset footer off
WITH e AS (
    SELECT tenant_id,
           count(*)                                AS entries,
           count(*) FILTER (WHERE seq IS NULL)     AS unsealed,
           coalesce(min(seq), 0)                   AS min_seq,
           coalesce(max(seq), 0)                   AS max_seq,
           count(DISTINCT seq)                     AS distinct_seqs,
           count(seq)                              AS sealed
      FROM audit_log
     GROUP BY tenant_id
), d AS (
    SELECT tenant_id, count(*) AS dup_seqs
      FROM (SELECT tenant_id, seq FROM audit_log WHERE seq IS NOT NULL
             GROUP BY tenant_id, seq HAVING count(*) > 1) x
     GROUP BY tenant_id
), l AS (
    SELECT a.tenant_id, count(*) AS broken_links
      FROM audit_log a
      LEFT JOIN audit_log p ON p.tenant_id = a.tenant_id AND p.seq = a.seq - 1
     WHERE a.seq IS NOT NULL
       AND a.prev_hash IS DISTINCT FROM
           CASE WHEN a.seq = 1 THEN decode(repeat('00', 32), 'hex') ELSE p.entry_hash END
     GROUP BY a.tenant_id
), r AS (
    SELECT h.tenant_id,
           coalesce(e.entries, 0)                          AS entries,
           coalesce(e.unsealed, 0)                         AS unsealed,
           coalesce(e.min_seq, 0)                          AS min_seq,
           coalesce(e.max_seq, 0)                          AS max_seq,
           coalesce(d.dup_seqs, 0)                         AS dup_seqs,
           coalesce(e.max_seq, 0) - coalesce(e.distinct_seqs, 0) AS missing_seqs,
           coalesce(l.broken_links, 0)                     AS broken_links,
           h.seq                                           AS head_seq,
           CASE WHEN h.seq = 0 THEN h.entry_hash = decode(repeat('00', 32), 'hex')
                ELSE EXISTS (SELECT 1 FROM audit_log x
                              WHERE x.tenant_id = h.tenant_id AND x.seq = h.seq
                                AND x.entry_hash = h.entry_hash)
           END                                             AS head_hash_ok,
           h.version                                       AS head_version,
           h.seal_interval_s
      FROM audit_chain_head h
      LEFT JOIN e ON e.tenant_id = h.tenant_id
      LEFT JOIN d ON d.tenant_id = h.tenant_id
      LEFT JOIN l ON l.tenant_id = h.tenant_id
)
SELECT tenant_id, entries, unsealed, min_seq, max_seq, dup_seqs, missing_seqs, broken_links,
       head_seq, head_hash_ok, head_version, seal_interval_s,
       (unsealed = 0 AND dup_seqs = 0 AND missing_seqs = 0 AND broken_links = 0
        AND (entries = 0 OR min_seq = 1) AND head_seq = max_seq AND head_hash_ok) AS ok
  FROM r
 ORDER BY tenant_id;

-- Entries of a tenant with no chain head: they can never be sealed.
SELECT count(*) AS entries_without_head
  FROM audit_log a
 WHERE NOT EXISTS (SELECT 1 FROM audit_chain_head h WHERE h.tenant_id = a.tenant_id);
