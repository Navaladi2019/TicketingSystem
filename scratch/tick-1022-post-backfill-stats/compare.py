"""TICK-1022: per-cell server time of the D13 list statements, by arm.

Reads the *-explain.md files TestFieldIndexPerf800k writes (one EXPLAIN
(ANALYZE) per statement that touches rfv_*, as nava_app) and prints, per
cell, planning + execution of the count and of the page, in ms, and the
count's join plan. Usage:

    python -I compare.py a0=bin/t1022-a0-explain.md b0=bin/t1022-b0-explain.md ...
"""
import re
import sys


def cells(path):
    s = open(path, encoding="utf-8").read()
    out = {}
    for p in re.split(r"^### ", s, flags=re.M)[1:]:
        title = p.splitlines()[0].strip()
        bits = p.split("```")
        sql, plan = bits[1], bits[3]
        if "LIMIT 20000" in sql:
            kind = "pre"  # TICK-1022 spike A': the capped side-index pre-count
        else:
            kind = "count" if "count(*)" in sql[:80] else "page"
        pl = float(re.search(r"Planning Time: ([0-9.]+)", plan).group(1))
        ex = float(re.search(r"Execution Time: ([0-9.]+)", plan).group(1))
        join = ""
        if kind == "count":
            m = re.search(r"(Hash Semi Join|Nested Loop Semi Join|Hash Join|Merge Semi Join)", plan)
            join = m.group(1).replace(" Semi Join", "").replace("Nested Loop", "NL") if m else ""
            if "temp written" in plan:
                join += "+spill"
        c = out.setdefault(title, {})
        c[kind] = pl + ex
        if join:
            c["join"] = join
    return out


def main(args):
    arms = [(a.split("=", 1)[0], cells(a.split("=", 1)[1])) for a in args]
    titles = list(arms[0][1])
    print("| Cell | " + " | ".join(f"{n} pre + count / page ms (count join)" for n, _ in arms) + " |")
    print("| --- |" + " --- |" * len(arms))
    for t in titles:
        row = f"| {t} |"
        for _, c in arms:
            x = c.get(t, {})
            pre = f"{x['pre']:.1f} + " if "pre" in x else ""
            row += f" {pre}{x.get('count', 0):.1f} / {x.get('page', 0):.1f} ({x.get('join', '-')}) |"
        print(row)


if __name__ == "__main__":
    main(sys.argv[1:])
