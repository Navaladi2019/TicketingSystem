# TICK-715: FK validation under FORCE RLS, bug and re-check cost

`bench.sql` runs on a throwaway `postgres:16.10` container (Docker 29.9.2,
Intel i7-1260P, 16 threads, Windows 11 host, `shared_buffers=1GB`). It
seeds 50,000 groups (200 tenants x 250), 2,500,000 memberships and 1,000,000
records (900,000 with an assignment group), all FORCE-RLS tables owned by a
non-superuser role with no policies. It puts one orphan in each table, then:

1. shows the owner's bare `VALIDATE` marks both keys valid over the orphans;
2. shows the P3-40 pattern (migrator SELECT policies inside the transaction)
   fails on each orphan with 23503;
3. times the two re-check migrations' SQL as written on a clean database,
   with the locks they hold.

`orphans.sql` is the orphan scan. Run it as the superuser, which bypasses
RLS. Output: `bench-out.txt`. Summary: `docs/data/fk-validate-force-rls.md`.

    docker run -d --rm --name tick715-bench -e POSTGRES_PASSWORD=x \
      -v "$PWD:/bench" postgres:16.10 -c shared_buffers=1GB
    docker exec -w /bench tick715-bench psql -U postgres -f bench.sql > bench-out.txt 2>&1
