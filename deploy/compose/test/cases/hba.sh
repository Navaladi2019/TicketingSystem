#!/bin/sh
# stack-case: job=stack-db order=10 mode=parallel
set -eu
. "$(dirname "$0")/../case-env.sh"

# db/pg_hba.conf in the running stack (T08: ADR-001 A.3, CI-9; T02 security
# review). Clients run in nava-core's network namespace, so PostgreSQL sees
# nava-core's address. The superuser is refused over TCP even with the right
# password; nava_backup gets no SQL session and no logical walsender, only a
# physical walsender from the backup host (127.0.0.1 in the postgres container).
t_hba() {
    core=$(container_of nava-core)
    [ -n "$core" ] || fail "nava-core is not running"
    # The client image is the one compose runs, so the tag lives only in
    # compose.yaml and docs/pins.md (make check-pins).
    pgimage=$(docker inspect --format '{{.Config.Image}}' "$(container_of postgres)")
    [ -n "$pgimage" ] || fail "postgres is not running"
    # from_core <user> <password> <conninfo> [sql]
    from_core() {
        docker run --rm --network "container:$core" -e PGPASSWORD="$2" "$pgimage" \
            psql -X -t -A "host=postgres user=$1 connect_timeout=10 $3" -c "${4:-SELECT 1}" 2>&1 || true
    }
    # in_pg <user> <password> <conninfo> [sql]: over TCP from inside the postgres container
    in_pg() {
        compose exec -T -e PGPASSWORD="$2" postgres \
            psql -X -t -A "host=127.0.0.1 user=$1 connect_timeout=10 $3" -c "${4:-SELECT 1}" 2>&1 || true
    }
    expect() { # <what> <grep -E pattern> <output>
        printf '%s\n' "$3" | grep -Eq "$2" || fail "$1: expected /$2/, got: $3"
        pass "$1"
    }
    su=${PG_SUPERUSER_PASSWORD:-dev-only-superuser}
    bk=${NAVA_BACKUP_DB_PASSWORD:-dev-only-backup}
    rejected='pg_hba.conf rejects connection'

    expect "control: nava_app from nava-core logs in to nava" '^1$' \
        "$(from_core nava_app "${NAVA_APP_DB_PASSWORD:-dev-only-app}" dbname=nava)"
    expect "control: pgadmin over the Unix socket" '^1$' "$(psql_super postgres 'SELECT 1')"

    expect "pgadmin from nava-core (correct password) refused" "$rejected.*\"pgadmin\"" \
        "$(from_core pgadmin "$su" dbname=postgres)"
    expect "pgadmin from nava-core to nava refused" "$rejected.*\"pgadmin\"" \
        "$(from_core pgadmin "$su" dbname=nava)"
    expect "pgadmin over TCP inside the postgres container refused" "$rejected.*\"pgadmin\"" \
        "$(in_pg pgadmin "$su" dbname=postgres)"

    expect "nava_backup SQL session to nava refused" "$rejected.*\"nava_backup\".*\"nava\"" \
        "$(from_core nava_backup "$bk" dbname=nava)"
    expect "nava_backup SQL session to postgres refused" "$rejected.*\"nava_backup\".*\"postgres\"" \
        "$(from_core nava_backup "$bk" dbname=postgres)"
    expect "nava_backup SQL session from the backup host refused" "$rejected.*\"nava_backup\"" \
        "$(in_pg nava_backup "$bk" dbname=postgres)"
    expect "nava_backup logical walsender (replication=database) refused" "$rejected.*\"nava_backup\"" \
        "$(from_core nava_backup "$bk" "dbname=nava replication=database" IDENTIFY_SYSTEM)"
    expect "nava_backup logical walsender from the backup host refused" "$rejected.*\"nava_backup\"" \
        "$(in_pg nava_backup "$bk" "dbname=nava replication=database" IDENTIFY_SYSTEM)"
    expect "nava_backup physical walsender from nava-core refused" 'no pg_hba.conf entry for replication connection' \
        "$(from_core nava_backup "$bk" replication=true IDENTIFY_SYSTEM)"
    expect "nava_backup physical walsender from the backup host accepted" '^[0-9]+\|1\|' \
        "$(in_pg nava_backup "$bk" replication=true IDENTIFY_SYSTEM)"
}

t_hba
