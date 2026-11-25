"""TICK-1125 times: paste 1 + paste 2 + COMMIT of the documented SQL, at 10,000
records (three runs) and 100,000 (two runs), on a throwaway postgres container
with the docs/pins.md tag and default settings.

Needs the dbtest binary, which leaves a database of the release before S31:

  go test -c -tags integration -o <dir>/dbtest.exe ./internal/db/dbtest
  python scratch/tick-1125-repair/bench.py <dir> > scratch/tick-1125-repair/bench-out.txt
"""
import os
import platform
import re
import subprocess
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.join(HERE, "..", "..")
BIN = os.path.abspath(sys.argv[1])
NAME, PORT = f"nava-tick1125-bench-{os.getpid()}", "55199"
TENANT, TYPE, OTHER = ("0190b000-0000-7000-8000-00000000000" + c for c in "abc")


def docker(*args, **kw):
    return subprocess.run(["docker", *args], capture_output=True, text=True, encoding="utf-8", **kw)


def psql(db, sql, *flags):
    p = docker("exec", "-i", NAME, "psql", "-X", "-U", "postgres", "-d", db, "-v", "ON_ERROR_STOP=1", "-At", *flags, input=sql)
    if p.returncode != 0:
        raise SystemExit(p.stdout + p.stderr)
    return p.stdout


def main():
    with open(os.path.join(ROOT, "docs", "pins.md"), encoding="utf-8") as f:
        tag = re.search(r"^\| `postgres` \| (\S+) \|", f.read(), re.M).group(1)
    with open(os.path.join(ROOT, "docs", "data", "entity-versions-org-unit-id-reserved.md"), encoding="utf-8") as f:
        doc = f.read()
    with open(os.path.join(HERE, "gen.sql"), encoding="utf-8") as f:
        gen = f.read()
    blocks = [re.search(r"```sql tick-1125-paste-%d\n(.*?)\n```\n" % i, doc, re.S).group(1) for i in (1, 2)]
    for key, value in (("tenant_id", TENANT), ("type_id", TYPE), ("source_ref", "CHG-1/run-1")):
        blocks[0] = re.sub(r"(nava_repair\.%s', ')[^']*" % key, lambda m, v=value: m.group(1) + v, blocks[0], count=1)
    docker("run", "-d", "--name", NAME, "-e", "POSTGRES_PASSWORD=bench", "-p", f"127.0.0.1:{PORT}:5432", f"postgres:{tag}", check=True)
    try:
        time.sleep(10)
        print("host:", platform.platform(), "|", platform.processor())
        print("docker VM:", docker("info", "--format", "{{.NCPU}} CPUs, {{.MemTotal}} bytes").stdout.strip())
        print("other containers running:", len(docker("ps", "-q").stdout.split()) - 1)
        print("server:", psql("postgres", "SHOW server_version").strip(), "| default settings:", psql(
            "postgres", "SELECT string_agg(n || '=' || current_setting(n), ' ') FROM unnest(ARRAY['shared_buffers',"
            "'work_mem','fsync','synchronous_commit','max_wal_size']) n").strip())
        env = dict(os.environ, NAVA_TEST_PG_URL=f"postgres://postgres:bench@127.0.0.1:{PORT}/postgres?sslmode=disable")
        for n, runs in ((10000, 3), (100000, 2)):
            for run in range(runs):
                p = subprocess.run([os.path.join(BIN, "dbtest.exe"), "-test.run", "^TestTICK1125Refuses$/^input$/^no-such-case$"],
                                   cwd=os.path.join(ROOT, "internal", "db", "dbtest"), env=env, capture_output=True, text=True)
                if p.returncode != 0:
                    raise SystemExit(p.stdout)
                size = psql("nava", gen, "-v", f"n={n}", "-v", f"tenant={TENANT}", "-v", f"type={TYPE}", "-v", f"other={OTHER}").strip().splitlines()[-1]
                out = psql("nava", "\\timing on\n" + blocks[0] + "\n\\echo == paste 2\n" + blocks[1] + "\n")
                ms = [[float(x) for x in re.findall(r"Time: ([\d.]+) ms", part)] for part in out.split("== paste 2")]
                left = psql("nava", "SELECT (SELECT count(*) FROM records WHERE entity_definition_id = '" + TYPE + "' AND data ? 'org_unit_id') || ' old names left, ' || "
                            "(SELECT count(*) FROM audit_log WHERE tenant_id = '" + TENANT + "' AND actor_system = 'operator') || ' entries written'").strip()
                print(f"{n} records, run {run + 1}: paste 1 {sum(ms[0]) / 1000:.1f} s, paste 2 {sum(ms[1][:-1]) / 1000:.1f} s, "
                      f"COMMIT {ms[1][-1] / 1000:.1f} s, total {(sum(ms[0]) + sum(ms[1])) / 1000:.1f} s | {left} | {size}", flush=True)
    finally:
        docker("rm", "-fv", NAME)


main()
