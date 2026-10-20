#!/bin/sh
# Creates or upgrades the Temporal schema in the existing temporal and
# temporal_visibility databases (db-roles creates them). Idempotent.
# Environment: PG_HOST, PG_PORT, PG_USER, TEMPORAL_DB_PASSWORD.
set -eu
host=${PG_HOST:-postgres}
port=${PG_PORT:-5432}
user=${PG_USER:-temporal}
schema=/etc/temporal/schema

tool() {
    db=$1; shift
    temporal-sql-tool --plugin postgres12 --ep "$host" -p "$port" -u "$user" \
        --pw "$TEMPORAL_DB_PASSWORD" --db "$db" "$@"
}

tool temporal setup-schema -v 0.0
tool temporal update-schema -d "$schema/temporal/versioned"
tool temporal_visibility setup-schema -v 0.0
tool temporal_visibility update-schema -d "$schema/visibility/versioned"
