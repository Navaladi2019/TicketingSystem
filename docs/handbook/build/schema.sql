-- Schema snapshot, version 1 (TICK-733). Run against a freshly migrated
-- platform database (`nava`) as a role that can read the whole catalogue;
-- docs/handbook/src/database/schema.md has the regenerate command and the
-- contract. Prints one JSON document; schema_json.py checks and formats it.
--
-- Every name is sorted with COLLATE "C" (byte order), so the result does not
-- depend on the database's default collation. json_build_object keeps the
-- key order written here. "_unsupported" lists relations version 1 cannot
-- name (outside schema public, or foreign); schema_json.py fails on any.
WITH platform AS (
    SELECT c.*, n.nspname
    FROM pg_catalog.pg_class c
    JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
      AND c.relkind IN ('r', 'p', 'f')
      AND NOT (n.nspname = 'public' AND c.relname = 'goose_db_version')
),
t AS (
    SELECT * FROM platform WHERE nspname = 'public' AND relkind IN ('r', 'p')
)
SELECT json_build_object(
  'version', 1,
  'tables', coalesce((
    SELECT json_agg(json_build_object(
      'name', t.relname,
      'kind', CASE WHEN t.relispartition THEN 'partition' WHEN t.relkind = 'p' THEN 'partitioned' ELSE 'table' END,
      'partition_of', coalesce((
          SELECT pc.relname FROM pg_catalog.pg_inherits i JOIN pg_catalog.pg_class pc ON pc.oid = i.inhparent
          WHERE i.inhrelid = t.oid AND t.relispartition), ''),
      'partition_by', CASE WHEN t.relkind = 'p' THEN pg_catalog.pg_get_partkeydef(t.oid) ELSE '' END,
      'partition_bound', CASE WHEN t.relispartition THEN pg_catalog.pg_get_expr(t.relpartbound, t.oid) ELSE '' END,
      'rls', json_build_object('enabled', t.relrowsecurity, 'forced', t.relforcerowsecurity),
      'columns', coalesce((
          SELECT json_agg(json_build_object(
                   'name', a.attname,
                   'type', pg_catalog.format_type(a.atttypid, a.atttypmod),
                   'nullable', NOT a.attnotnull,
                   'default', coalesce(pg_catalog.pg_get_expr(d.adbin, d.adrelid), ''))
                 ORDER BY a.attnum)
          FROM pg_catalog.pg_attribute a
          LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
          WHERE a.attrelid = t.oid AND a.attnum > 0 AND NOT a.attisdropped), '[]'::json),
      'primary_key', coalesce((
          SELECT json_agg(a.attname ORDER BY k.ord)
          FROM pg_catalog.pg_constraint con
          CROSS JOIN LATERAL unnest(con.conkey) WITH ORDINALITY k(num, ord)
          JOIN pg_catalog.pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = k.num
          WHERE con.conrelid = t.oid AND con.contype = 'p'), '[]'::json),
      'foreign_keys', coalesce((
          SELECT json_agg(json_build_object(
                   'name', con.conname,
                   'columns', (SELECT json_agg(a.attname ORDER BY k.ord)
                               FROM unnest(con.conkey) WITH ORDINALITY k(num, ord)
                               JOIN pg_catalog.pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = k.num),
                   'ref_table', fc.relname,
                   'ref_columns', (SELECT json_agg(a.attname ORDER BY k.ord)
                                   FROM unnest(con.confkey) WITH ORDINALITY k(num, ord)
                                   JOIN pg_catalog.pg_attribute a ON a.attrelid = con.confrelid AND a.attnum = k.num),
                   'on_delete', CASE con.confdeltype WHEN 'r' THEN 'RESTRICT' WHEN 'c' THEN 'CASCADE'
                                     WHEN 'n' THEN 'SET NULL' WHEN 'd' THEN 'SET DEFAULT' ELSE 'NO ACTION' END)
                 ORDER BY con.conname COLLATE "C")
          FROM pg_catalog.pg_constraint con
          JOIN pg_catalog.pg_class fc ON fc.oid = con.confrelid
          WHERE con.conrelid = t.oid AND con.contype = 'f'), '[]'::json),
      'indexes', coalesce((
          SELECT json_agg(json_build_object(
                   'name', ic.relname,
                   'method', am.amname,
                   'unique', i.indisunique,
                   'columns', coalesce((SELECT json_agg(pg_catalog.pg_get_indexdef(i.indexrelid, k, true) ORDER BY k)
                                        FROM generate_series(1, i.indnkeyatts) k), '[]'::json),
                   'include', coalesce((SELECT json_agg(pg_catalog.pg_get_indexdef(i.indexrelid, k, true) ORDER BY k)
                                        FROM generate_series(i.indnkeyatts + 1, i.indnatts) k), '[]'::json),
                   'predicate', coalesce(pg_catalog.pg_get_expr(i.indpred, i.indrelid, true), ''))
                 ORDER BY ic.relname COLLATE "C")
          FROM pg_catalog.pg_index i
          JOIN pg_catalog.pg_class ic ON ic.oid = i.indexrelid
          JOIN pg_catalog.pg_am am ON am.oid = ic.relam
          WHERE i.indrelid = t.oid), '[]'::json),
      'policies', coalesce((
          SELECT json_agg(json_build_object(
                   'name', p.policyname,
                   'command', p.cmd,
                   'roles', (SELECT json_agg(r ORDER BY r::text COLLATE "C") FROM unnest(p.roles) r),
                   'permissive', p.permissive = 'PERMISSIVE',
                   'using', coalesce(p.qual, ''),
                   'check', coalesce(p.with_check, ''))
                 ORDER BY p.policyname COLLATE "C")
          FROM pg_catalog.pg_policies p
          WHERE p.schemaname = 'public' AND p.tablename = t.relname), '[]'::json)
    ) ORDER BY t.relname COLLATE "C")
    FROM t), '[]'::json),
  '_unsupported', coalesce((
    SELECT json_agg(nspname || '.' || relname || CASE WHEN relkind = 'f' THEN ' (foreign)' ELSE '' END
                    ORDER BY nspname COLLATE "C", relname COLLATE "C")
    FROM platform WHERE nspname <> 'public' OR relkind = 'f'), '[]'::json)
);
