-- T18a: write and storage cost of the chosen key (one index) against the
-- tenant_identity_binding shape (PK (tenant_id, domain) + UNIQUE (domain)).
-- The benchmark's 25,000 rows, inserted in random order 5 times per shape
-- (truncated between runs), timed with \timing, as the superuser, no RLS.
-- Run after t18a_email_domain_bench.sql on the same database.
\set ON_ERROR_STOP on
DROP TABLE IF EXISTS bench_one, bench_two;
CREATE TABLE bench_one (tenant_id uuid NOT NULL, domain text NOT NULL, PRIMARY KEY (domain) INCLUDE (tenant_id));
CREATE TABLE bench_two (tenant_id uuid NOT NULL, domain text NOT NULL, PRIMARY KEY (tenant_id, domain), UNIQUE (domain));
CREATE TEMP TABLE src AS SELECT tenant_id, domain FROM tenant_email_domain ORDER BY random();
TRUNCATE bench_one, bench_two;
\timing on
INSERT INTO bench_one SELECT * FROM src;
INSERT INTO bench_two SELECT * FROM src;
\timing off
TRUNCATE bench_one, bench_two;
\timing on
INSERT INTO bench_one SELECT * FROM src;
INSERT INTO bench_two SELECT * FROM src;
\timing off
TRUNCATE bench_one, bench_two;
\timing on
INSERT INTO bench_one SELECT * FROM src;
INSERT INTO bench_two SELECT * FROM src;
\timing off
TRUNCATE bench_one, bench_two;
\timing on
INSERT INTO bench_one SELECT * FROM src;
INSERT INTO bench_two SELECT * FROM src;
\timing off
TRUNCATE bench_one, bench_two;
\timing on
INSERT INTO bench_one SELECT * FROM src;
INSERT INTO bench_two SELECT * FROM src;
\timing off
SELECT c.relname, pg_size_pretty(pg_indexes_size(c.oid)) AS indexes, pg_size_pretty(pg_relation_size(c.oid)) AS heap
FROM pg_class c WHERE c.relname IN ('bench_one', 'bench_two');
DROP TABLE bench_one, bench_two;
