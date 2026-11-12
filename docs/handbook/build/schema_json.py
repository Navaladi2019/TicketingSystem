"""Write the schema snapshot (TICK-733) from the output of schema.sql.

Usage: psql ... -f schema.sql | python3 schema_json.py OUT.json

Reads the one JSON document schema.sql prints on stdin, checks it against
contract version 1 (docs/handbook/src/database/schema.md), and writes OUT.json
with a two-space indent, LF line endings and a trailing newline. OUT.json is
written only when every check passes. Standard library only.
"""

import json
import re
import sys

# Every key of version 1, per object, in the order schema.sql writes them.
TABLE_KEYS = ["name", "kind", "partition_of", "partition_by", "partition_bound", "rls",
              "columns", "primary_key", "foreign_keys", "indexes", "policies"]
NESTED_KEYS = {
    "rls": ["enabled", "forced"],
    "columns": ["name", "type", "nullable", "default"],
    "foreign_keys": ["name", "columns", "ref_table", "ref_columns", "on_delete"],
    "indexes": ["name", "method", "unique", "columns", "include", "predicate"],
    "policies": ["name", "command", "roles", "permissive", "using", "check"],
}

# Connection details the file must never hold (the repository is public).
LEAKS = [
    ("a URL or DSN", r"://"),
    ("a keyword DSN", r"(?i)\b(host|hostaddr|port|user|password|dbname|sslmode)\s*="),
    ("a password", r"(?i)passw(or)?d|\btest-pw-|\btest-only\b"),
    ("the bootstrap superuser", r"\bpgadmin\b"),
    ("a pg_hba.conf entry", r"(?i)\bscram-sha-256\b|\b(local|host|hostssl|hostnossl)\s+(all|replication)\s"),
    ("an IPv4 address", r"\b\d{1,3}(\.\d{1,3}){3}\b"),
    ("an IPv6 address", r"::/\d|\b[0-9a-fA-F]{1,4}(:[0-9a-fA-F]{1,4}){7}\b"),
    ("an absolute local path", r"(?i)\b[a-z]:\\|/home/|/Users/"),
]


def byte_sorted(names):
    return names == sorted(names, key=lambda s: s.encode("utf-8"))


def problems(doc):
    out = []
    if list(doc) != ["version", "tables"] or doc["version"] != 1:
        out.append("top level must be exactly {version: 1, tables}")
    tables = doc.get("tables", [])
    if not tables:
        out.append("no tables: is this the migrated platform database?")
    if not byte_sorted([t["name"] for t in tables]):
        out.append("tables are not in byte order of name")
    for t in tables:
        name = t.get("name", "?")
        if list(t) != TABLE_KEYS:
            out.append(f"{name}: keys {list(t)}, want {TABLE_KEYS}")
            continue
        kind = t["kind"]
        part = (t["partition_of"], t["partition_by"], t["partition_bound"])
        if kind == "table" and part != ("", "", ""):
            out.append(f"{name}: a table with partition keys set")
        elif kind == "partitioned" and not t["partition_by"]:
            out.append(f"{name}: partitioned with no partition_by")
        elif kind == "partition" and not (t["partition_of"] and t["partition_bound"]):
            out.append(f"{name}: a partition with no parent or bound")
        elif kind not in ("table", "partitioned", "partition"):
            out.append(f"{name}: kind {kind!r}")
        if list(t["rls"]) != NESTED_KEYS["rls"]:
            out.append(f"{name}: rls keys {list(t['rls'])}")
        for key in ("columns", "foreign_keys", "indexes", "policies"):
            for item in t[key]:
                if list(item) != NESTED_KEYS[key]:
                    out.append(f"{name}.{key}: keys {list(item)}, want {NESTED_KEYS[key]}")
            if key != "columns" and not byte_sorted([i["name"] for i in t[key]]):
                out.append(f"{name}.{key}: not in byte order of name")
        for p in t["policies"]:
            if not byte_sorted(p["roles"]):
                out.append(f"{name}.{p['name']}: roles not sorted")
    return out


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: psql ... -f schema.sql | python3 schema_json.py OUT.json")
    raw = sys.stdin.read().strip()
    if not raw:
        sys.exit("schema_json.py: no input from psql")
    doc = json.loads(raw)
    unsupported = doc.pop("_unsupported", [])
    errs = [f"version 1 cannot name {u}" for u in unsupported] + problems(doc)
    text = json.dumps(doc, indent=2, ensure_ascii=False) + "\n"
    errs += [f"the snapshot holds {what}: {m.group(0)!r}"
             for what, pattern in LEAKS
             for m in [re.search(pattern, text, re.MULTILINE)] if m]
    if errs:
        sys.exit("schema_json.py: not written:\n  " + "\n  ".join(errs))
    with open(sys.argv[1], "w", encoding="utf-8", newline="\n") as f:
        f.write(text)
    print(f"wrote {sys.argv[1]}: {len(doc['tables'])} tables")


if __name__ == "__main__":
    main()
