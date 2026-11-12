-- TICK-714 spike: which comparison operators are leakproof in this server
-- (a qual is pushed below a forced-RLS policy as an index condition only if
-- its operator function is leakproof).
SELECT o.oprname AS op, format_type(o.oprleft, NULL) AS lhs, format_type(o.oprright, NULL) AS rhs,
       p.proname AS func, p.proleakproof AS leakproof
FROM pg_operator o JOIN pg_proc p ON p.oid = o.oprcode
WHERE (o.oprleft, o.oprright) IN (('text'::regtype, 'text'::regtype), ('numeric'::regtype, 'numeric'::regtype),
                                  ('int8'::regtype, 'int8'::regtype), ('timestamptz'::regtype, 'timestamptz'::regtype),
                                  ('bool'::regtype, 'bool'::regtype), ('uuid'::regtype, 'uuid'::regtype),
                                  ('date'::regtype, 'date'::regtype),
                                  ('tsvector'::regtype, 'tsquery'::regtype), ('jsonb'::regtype, 'jsonb'::regtype),
                                  ('jsonb'::regtype, 'text'::regtype))
  AND o.oprname IN ('=', '<>', '<', '<=', '>', '>=', '@@', '@>', '->>', '?', '~~', '~~*')
ORDER BY lhs, rhs, op;
-- like / ILIKE / trigram operators, if present
SELECT p.proname, p.proleakproof FROM pg_proc p
WHERE p.proname IN ('textlike', 'texticlike', 'ts_match_vq', 'ts_match_tq', 'to_tsvector', 'jsonb_object_field_text',
                    'jsonb_contains', 'jsonb_path_match', 'starts_with', 'lower')
ORDER BY 1;
