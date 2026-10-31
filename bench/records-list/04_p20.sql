-- Page-20 actors and their keyset cursors, filled by `q.sh cursors <q>`.
CREATE TABLE bench_p20 (
    q         text        NOT NULL,
    n         bigint      NOT NULL,
    tenant_id uuid        NOT NULL,
    user_id   uuid        NOT NULL,
    groups    uuid[]      NOT NULL,
    cts       timestamptz NOT NULL,
    cid       uuid        NOT NULL,
    PRIMARY KEY (q, n)
);
GRANT SELECT ON bench_tenant, bench_actor, bench_heavy, bench_p20 TO nava_app;
