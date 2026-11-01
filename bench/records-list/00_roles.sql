-- TICK-339 (P3-01) synthetic bench. Runs as the superuser on database postgres.
-- nava_app mirrors the runtime role: LOGIN, no BYPASSRLS, not the table owner.
CREATE ROLE nava_app LOGIN PASSWORD 'bench' NOSUPERUSER NOBYPASSRLS;
CREATE ROLE bench_owner NOLOGIN;
CREATE DATABASE nava OWNER bench_owner;
