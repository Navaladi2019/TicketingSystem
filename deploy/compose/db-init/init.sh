#!/bin/sh
# One-shot, idempotent bootstrap run by the db-init service as the image
# superuser, the only place that credential is used. Safe to re-run: existing
# roles and databases are left alone and the role passwords are re-applied.
#
# To add a service: call ensure_role, then ensure_database, below. T03 grows
# this script into the db-roles job.
set -eu

PGOPTIONS='-c client_min_messages=warning'
export PGOPTIONS
PGPASSWORD=$(cat "$PG_SUPERUSER_PASSWORD_FILE")
export PGPASSWORD

created=0
existing=0

log() { printf 'db-init: %s\n' "$*"; }

# sql <database> <statement> [psql options]: run one statement; reference
# values passed with -v name=value as :'name' (quoted literal) or :"name".
sql() {
    db=$1
    stmt=$2
    shift 2
    printf '%s\n' "$stmt" | psql -X -q -t -A -v ON_ERROR_STOP=1 -d "$db" "$@"
}

# ensure_role <name> <password>
ensure_role() {
    if [ -z "$(sql postgres "SELECT 1 FROM pg_roles WHERE rolname = :'n'" -v n="$1")" ]; then
        sql postgres "SELECT format('CREATE ROLE %I LOGIN', :'n') \gexec" -v n="$1"
        log "created role $1"
        created=$((created + 1))
    else
        existing=$((existing + 1))
    fi
    # Re-applied every run so a changed password in .env reaches the database.
    sql postgres "SELECT format('ALTER ROLE %I PASSWORD %L', :'n', :'pw') \gexec" -v n="$1" -v pw="$2"
}

# ensure_database <name> [owner]
ensure_database() {
    if [ -z "$(sql postgres "SELECT 1 FROM pg_database WHERE datname = :'d'" -v d="$1")" ]; then
        if [ -n "${2:-}" ]; then
            sql postgres "SELECT format('CREATE DATABASE %I OWNER %I', :'d', :'o') \gexec" -v d="$1" -v o="$2"
        else
            sql postgres "SELECT format('CREATE DATABASE %I', :'d') \gexec" -v d="$1"
        fi
        log "created database $1"
        created=$((created + 1))
    else
        existing=$((existing + 1))
    fi
}

ensure_role keycloak "$KEYCLOAK_DB_PASSWORD"
ensure_role temporal "$TEMPORAL_DB_PASSWORD"
ensure_role pgbouncer_auth "$PGBOUNCER_AUTH_PASSWORD"

ensure_database keycloak keycloak
ensure_database temporal temporal
ensure_database temporal_visibility temporal
# The database behind PgBouncer. T03 gives it its roles and owner.
ensure_database nava

# PgBouncer authenticates every other role with this lookup, so it needs no
# copy of their passwords. CREATE OR REPLACE and GRANT are safe to repeat.
sql nava "
CREATE SCHEMA IF NOT EXISTS pgbouncer;
CREATE OR REPLACE FUNCTION pgbouncer.get_auth(p_usename text)
    RETURNS TABLE (usename name, passwd text)
    LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog
    AS 'SELECT usename, passwd FROM pg_catalog.pg_shadow WHERE usename = p_usename';
REVOKE ALL ON FUNCTION pgbouncer.get_auth(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA pgbouncer TO pgbouncer_auth;
GRANT EXECUTE ON FUNCTION pgbouncer.get_auth(text) TO pgbouncer_auth;
"

log "done: $created created, $existing already present"
