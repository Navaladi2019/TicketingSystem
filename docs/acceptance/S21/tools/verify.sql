-- S21 acceptance (TICK-749): after the Bruno folder, the side tables against
-- a JSON scan of every tenant A incident, for both flagged fields (criteria 1
-- and 4: no record missed across the flip, none surplus), plus the registry
-- rows and tenant B's side rows (criterion 5: tenant B flagged nothing, so it
-- has none). Run by run-kit.sh; it fails on any non-zero `missing`, `surplus`
-- or `wrong` and on any tenant B row.
\set ON_ERROR_STOP on
\pset footer off
SELECT f.tenant_id, f.field_key, f.value_type, f.state, f.progress, f.requested_version, f.updated_at
  FROM record_index_fields f ORDER BY 1, 2;

CREATE TEMP TABLE s21_inc AS
SELECT r.id, r.data
  FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
 WHERE r.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND d.key = 'incident';
CREATE TEMP TABLE s21_f AS
SELECT field_key, id FROM record_index_fields
 WHERE tenant_id = '00000000-0000-4000-8000-0000000000a1' AND field_key IN ('priority', 'due_at');

SELECT 'priority' AS field,
       (SELECT count(*) FROM s21_inc WHERE data ? 'priority') AS scan_rows,
       (SELECT count(*) FROM rfv_num v JOIN s21_f f ON f.id = v.field_id AND f.field_key = 'priority'
         WHERE v.tenant_id = '00000000-0000-4000-8000-0000000000a1') AS side_rows,
       (SELECT count(*) FROM s21_inc i WHERE i.data ? 'priority' AND NOT EXISTS (
          SELECT 1 FROM rfv_num v JOIN s21_f f ON f.id = v.field_id AND f.field_key = 'priority'
           WHERE v.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND v.record_id = i.id)) AS missing,
       (SELECT count(*) FROM rfv_num v JOIN s21_f f ON f.id = v.field_id AND f.field_key = 'priority'
          LEFT JOIN s21_inc i ON i.id = v.record_id
         WHERE v.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND (i.id IS NULL OR NOT i.data ? 'priority')) AS surplus,
       (SELECT count(*) FROM rfv_num v JOIN s21_f f ON f.id = v.field_id AND f.field_key = 'priority'
          JOIN s21_inc i ON i.id = v.record_id
         WHERE v.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND v.value <> (i.data->>'priority')::float8) AS wrong
UNION ALL
SELECT 'due_at',
       (SELECT count(*) FROM s21_inc WHERE data ? 'due_at'),
       (SELECT count(*) FROM rfv_ts v JOIN s21_f f ON f.id = v.field_id AND f.field_key = 'due_at'
         WHERE v.tenant_id = '00000000-0000-4000-8000-0000000000a1'),
       (SELECT count(*) FROM s21_inc i WHERE i.data ? 'due_at' AND NOT EXISTS (
          SELECT 1 FROM rfv_ts v JOIN s21_f f ON f.id = v.field_id AND f.field_key = 'due_at'
           WHERE v.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND v.record_id = i.id)),
       (SELECT count(*) FROM rfv_ts v JOIN s21_f f ON f.id = v.field_id AND f.field_key = 'due_at'
          LEFT JOIN s21_inc i ON i.id = v.record_id
         WHERE v.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND (i.id IS NULL OR NOT i.data ? 'due_at')),
       (SELECT count(*) FROM rfv_ts v JOIN s21_f f ON f.id = v.field_id AND f.field_key = 'due_at'
          JOIN s21_inc i ON i.id = v.record_id
         WHERE v.tenant_id = '00000000-0000-4000-8000-0000000000a1' AND v.value <> (i.data->>'due_at')::timestamptz);

SELECT 'tenant B side rows' AS check,
       (SELECT count(*) FROM record_index_fields WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1')
     + (SELECT count(*) FROM rfv_num WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1')
     + (SELECT count(*) FROM rfv_ts WHERE tenant_id = '00000000-0000-4000-8000-0000000000b1') AS rows;

-- The records criterion 4 wrote, by due_at: created or edited while the
-- fields were backfilling and after they were ready.
SELECT i.data->>'due_at' AS due_at, count(*) AS records, min(r.updated_at) AS first_write, max(r.updated_at) AS last_write
  FROM s21_inc i JOIN records r ON r.id = i.id AND r.tenant_id = '00000000-0000-4000-8000-0000000000a1'
 WHERE i.data->>'due_at' IN ('2026-12-30T12:00:17Z', '2026-12-30T12:00:18Z')
 GROUP BY 1 ORDER BY 1;

-- When the side tables were last analyzed, against the flip times above. The
-- list plans of S21.19 depend on these statistics (TICK-749: one fresh stack
-- missed the >= 40% ACL cell by 2x, the next passed it; see the README).
SELECT relname, n_live_tup, last_analyze, last_autoanalyze, autoanalyze_count
  FROM pg_stat_all_tables WHERE relname IN ('records', 'rfv_num', 'rfv_ts') ORDER BY relname;

-- The visibility map of the side tables when the folder ended, right after
-- S21.19's cells (ADR-006-A2 A2-7: a broad ACL count reads the heap until
-- autovacuum marks pages all-visible, so a low share predicts `unknown`).
-- run-kit.sh's visibility.txt has the same columns sampled every 10 s.
SELECT c.relname, c.relpages, c.relallvisible,
       round(100.0 * c.relallvisible / nullif(c.relpages, 0), 1) AS all_visible_pct,
       s.n_live_tup, s.last_autovacuum, s.autovacuum_count
  FROM pg_class c JOIN pg_stat_all_tables s ON s.relid = c.oid
 WHERE c.relname IN ('rfv_num', 'rfv_ts') ORDER BY 1;
