"""TICK-1125 red proof: runs each test against the documented SQL with one line
removed or broken, and prints which tests fail. The document is restored
afterwards. Every mutant must fail a test; a survivor exits 1.

Needs NAVA_TEST_PG_URL (a throwaway cluster) and the two test binaries:

  go test -c -tags integration -o <dir>/dbtest.exe ./internal/db/dbtest
  go test -c -tags integration -o <dir>/server.exe ./internal/server
  python scratch/tick-1125-repair/mutate.py <dir> > scratch/tick-1125-repair/mutate-out.txt
"""
import os
import re
import subprocess
import sys

# MUTATE_ROOT: a copy of docs/data and of the two package directories, so that
# shards (MUTATE_SHARD=i/n, each with its own cluster) do not share the document.
ROOT = os.environ.get("MUTATE_ROOT") or os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..")
SHARD, SHARDS = (int(x) for x in os.environ.get("MUTATE_SHARD", "0/1").split("/"))
DOC = os.path.join(ROOT, "docs", "data", "entity-versions-org-unit-id-reserved.md")
BIN = os.path.abspath(sys.argv[1])
ONLY = sys.argv[2] if len(sys.argv) > 2 else ""

REFUSES, REPAIRS, PASTES = "TestTICK1125Refuses", "TestTICK1125Repairs", "TestTICK1125Pastes"
UPGRADE, CHECKS, API = "TestTICK1125UpgradeAfterRepair", "TestTICK1125Checks", "TestTICK1125APIAfterTheRepair"

SALT = "substring(sha256(uuid_send(gen_random_uuid()) || uuid_send(gen_random_uuid())) FROM 1 FOR 16)"
RETURN_EMPTY = "        RAISE NOTICE 'nothing to repair';\n        RETURN;\n"


def off(message):
    """Turns the IF whose RAISE starts with message off."""
    return (r"(IF [^;]*? THEN)(\s+RAISE EXCEPTION '" + re.escape(message) + ")", r"IF false THEN\2", True)


# (issue test, what is removed or broken, run pattern, old, new, regex)
M = [
    (1, "entity_definition_id on the record entries", API, "'record', 'update', 'record', r.id, v_type,", "'record', 'update', 'record', r.id, NULL,"),
    (2, "a salt per row (scalar subquery)", REPAIRS + "/salts", SALT + ",\n        d.diff, d.truncated, coalesce", "(SELECT " + SALT + "),\n        d.diff, d.truncated, coalesce"),
    (2, "a salt per row, and paste 2's salt assertion", REPAIRS + "/salts", None, None),
    (2, "ISOLATION LEVEL READ COMMITTED", PASTES + "/a_mask", "BEGIN ISOLATION LEVEL READ COMMITTED;", "BEGIN;"),
    (3, "R1", REFUSES + "/R1", *off("R1:")),
    (3, "R1's new name", REFUSES + "/R1", "m ->> 'field' IN (c_old, v_new)", "m ->> 'field' IN (c_old)"),
    (3, "R1's newest version only", REFUSES + "/repaired", "ORDER BY x.version DESC LIMIT 1) a", "ORDER BY x.version DESC) a"),
    (3, "R2", REFUSES + "/R2", *off("R2:")),
    (3, "R2's set", REFUSES + "/R2", "n.node -> 'set' ?| ARRAY[c_old, v_new]", "false"),
    (3, "R2's field", REFUSES + "/R2", "OR split_part(n.node ->> 'field', '/', 2) IN (c_old, v_new)", ""),
    (3, "R2's new name", REFUSES + "/R2", "split_part(n.node ->> 'field', '/', 2) IN (c_old, v_new)", "split_part(n.node ->> 'field', '/', 2) IN (c_old)"),
    (3, "R2's open run", REFUSES + "/R2", "w.closed_at IS NULL AND r.entity", "false AND r.entity"),
    (3, "R2's closed_at", REFUSES + "/repaired", "w.closed_at IS NULL AND r.entity", "r.entity"),
    (3, "R2's unreadable body", REFUSES + "/R2", "WHEN NOT pg_input_is_valid(v.body::text, 'jsonb') THEN true", "WHEN NOT pg_input_is_valid(v.body::text, 'jsonb') THEN false"),
    (3, "R3", REFUSES + "/R3", *off("R3:")),
    (3, "R3's writeOnly", REFUSES + "/R3", " OR coalesce(p.prop -> 'writeOnly', 'false') <> 'false'", ""),
    (3, "R3's x-nava-secret", REFUSES + "/R3", "coalesce(p.prop -> 'x-nava-secret', 'false') <> 'false' OR ", ""),
    (3, "R4", REFUSES + "/R4", *off("R4:")),
    (3, "R4's properties", REFUSES + "/R4", "v.schema::jsonb -> 'properties' ? v_new", "false"),
    (3, "R4's records", REFUSES + "/R4", "r.data ? v_new", "false"),
    (3, "R4's path", REFUSES + "/R4", "v_new IN (split_part(o.op ->> 'path', '/', 2), ", "v_new IN ("),
    (3, "R4's from", REFUSES + "/R4", ", split_part(o.op ->> 'from', '/', 2))", ")"),
    (3, "R4's root op", REFUSES + "/R4", "OR o.op ->> 'path' = '' AND o.op -> 'value' ? v_new", ""),
    (3, "R5", REFUSES + "/R5", *off("R5:")),
    (3, "R5's keys, old name", REFUSES + "/R5", "strpos(m.key, c_old) > 0 OR ", ""),
    (3, "R5's keys, new name", REFUSES + "/R5", "strpos(m.key, v_new) > 0\n", "false\n"),
    (3, "R5's strings, old name", REFUSES + "/R5", "(strpos(m.value #>> '{}', c_old) > 0 OR ", "("),
    (3, "R5's strings, new name", REFUSES + "/R5", " OR strpos(m.value #>> '{}', v_new) > 0)", ")"),
    (3, "R5's percent sign", REFUSES + "/R5", "m.key IN ('$ref', '$dynamicRef') AND strpos(m.value #>> '{}', '%') > 0", "false"),
    (3, "R5's $dynamicRef", REFUSES + "/R5", "('$ref', '$dynamicRef')", "('$ref')"),
    (3, "R5's propertyNames", REFUSES + "/R5", "('propertyNames', 'patternProperties')", "('patternProperties')"),
    (3, "R5's patternProperties", REFUSES + "/R5", "('propertyNames', 'patternProperties')", "('propertyNames')"),
    (3, "R5 over the property's own schema", REFUSES + "/R5", "            d.doc -> 'properties' -> c_old) AS doc", "            NULL::jsonb) AS doc"),
    (3, "R5 over every version", REFUSES + "/R5", "WHERE v.tenant_id = v_tenant AND v.definition_id = v_type) s", "WHERE v.tenant_id = v_tenant AND v.definition_id = v_type AND v.version = ANY (v_refused)) s"),
    (21, "R5's annotations", REFUSES + "/repaired", "coalesce(m.key NOT IN ('title', 'description', '$comment'), true)", "true"),
    (3, "R6", REFUSES + "/R6", *off("R6:")),
    (3, "R6's duplicate keys", REFUSES + "/R6", " OR v.schema::text IS NOT JSON WITH UNIQUE KEYS", ""),
    (3, "R6's jsonb test", REFUSES + "/R6", "NOT pg_input_is_valid(v.schema::text, 'jsonb') OR ", ""),
    (3, "R7", REFUSES + "/R7", *off("R7:")),
    (3, "R7's colon", REFUSES + "/repaired", "strpos(c.value #>> '{}', ':') > 0 AND ", ""),
    (3, "R7's strings (regex on jsonb text instead)", REFUSES + "/R7", "strpos(c.value #>> '{}', ':') > 0 AND strpos(c.value #>> '{}', '@') > 0", "c.value::text ~ '://[^/?#@]*@'"),
    (3, "R7's objects", REFUSES + "/R7", "WHEN 'object' THEN strpos(c.value::text, '@') > 0", "WHEN 'object' THEN false"),
    (3, "R7's arrays", REFUSES + "/R7", "WHEN 'array' THEN strpos(c.value::text, '@') > 0", "WHEN 'array' THEN false"),
    (3, "R7's older versions", REFUSES + "/R7", "AS prop) p\n        WHERE v.tenant_id = v_tenant AND v.definition_id = v_type\n", "AS prop) p\n        WHERE v.tenant_id = v_tenant AND v.definition_id = v_type AND v.version = 2\n"),
] + [
    (3, "R7's token " + tok, REFUSES + "/R7", "'" + tok + "'", "'no such token'") for tok in
    ("AKIA", "ASIA", "ghp_", "gho_", "ghu_", "ghs_", "ghr_", "github_pat_", "xox", "-----BEGIN", "eyJ")
] + [
    (3, "R8", REFUSES + "/R8", *off("R8: the versions")),
    (3, "R8's clean version", REFUSES + "/R8", "v.version <> ALL (v_refused) AND v.fixed_columns ? c_old\n           OR ", ""),
    (3, "R8's unknown column", REFUSES + "/R8", *off("R8: the property")),
    (3, "R9, entries", PASTES + "/R9_an", *off("R9: a record")),
    (3, "R9, sessions", PASTES + "/R9_a_writer", *off("R9: a service")),
    (3, "R9's database", PASTES + "/R9_a_writer", "s.datname = current_database() AND ", ""),
    (3, "R9's roles", PASTES + "/R9_a_writer", "s.usename IN ('nava_app', 'nava_worker')", "s.usename <> current_user"),
    (3, "R10", REFUSES + "/R10", *off("R10:")),
    (3, "R10's limit", REFUSES + "/repaired", "IF v_count > v_limit THEN", "IF v_count >= v_limit THEN"),
    (3, "R11", REFUSES + "/R11", *off("R11:")),
    (4, "the type belongs to the tenant", REFUSES + "/input", "IF NOT EXISTS (SELECT 1 FROM public.entity_definitions d WHERE d.tenant_id = v_tenant AND d.id = v_type) THEN", "IF false THEN"),
    (5, "the salt in diff_sha256 of the fixed_columns entry", API, "sha256(p.salt || convert_to(p.diff, 'UTF8'))", "sha256(convert_to(p.diff, 'UTF8'))"),
    (5, "the salt in diff_sha256 of the truncated entry", API, "sha256(r.salt || convert_to(r.diff, 'UTF8'))", "sha256(convert_to(r.diff, 'UTF8'))"),
    (6, "older versions are repaired", UPGRADE, "    IF v_refused IS NULL THEN", "    SELECT array_agg(d.current_version) INTO v_refused FROM public.entity_definitions d\n    WHERE d.id = v_type AND d.current_version = ANY (v_refused);\n    IF v_refused IS NULL THEN"),
    (6, "the current version is repaired", UPGRADE, "    IF v_refused IS NULL THEN", "    SELECT array_agg(u) INTO v_refused FROM public.entity_definitions d, unnest(v_refused) u\n    WHERE d.id = v_type AND d.current_version <> u;\n    IF v_refused IS NULL THEN"),
    (6, "every refused version is planned", UPGRADE, "AND v.version = ANY (v_refused);\n\n    IF v_column", "AND v.version = v_refused[1];\n\n    IF v_column"),
    (7, "fixed_columns in the UPDATE", REPAIRS + "/mapped", ", fixed_columns = p.new_fixed_columns", ""),
    (7, "the fixed_columns entry", REPAIRS + "/mapped", "    IF v_column IS NOT NULL THEN\n        INSERT INTO", "    IF false THEN\n        INSERT INTO"),
    (7, "records whose column is not NULL", REPAIRS + "/mapped", "ELSE r.data -> c_old END AS value) x", "ELSE r.data -> c_old END AS value) x0\n    CROSS JOIN LATERAL (SELECT coalesce(x0.value, 'null') AS value) x"),
    (7, "requester_id in the plan", REPAIRS + "/mapped", "THEN to_jsonb(r.state) WHEN 'requester_id' THEN to_jsonb(r.requester_id)\n            WHEN", "THEN to_jsonb(r.state) WHEN 'requester_id' THEN NULL\n            WHEN"),
    (8, "entity_definition_id on the record entries", REPAIRS + "/content", "'record', 'update', 'record', r.id, v_type,", "'record', 'update', 'record', r.id, NULL,"),
    (8, "version_after", REPAIRS + "/content", "r.version, r.version + 1,", "r.version, r.version,"),
    (8, "source_ref", REPAIRS + "/content", "'operator', 'system', v_ref, p.content", "'operator', 'system', 'x', p.content"),
    (8, "content_sha256 of the new schema", REPAIRS + "/content", "public.audit_seed_uuidv7(), sha256(convert_to(s.txt, 'UTF8')),", "public.audit_seed_uuidv7(), v.schema_sha256,"),
    (8, "updated_at stays", REPAIRS + "/content", "    SET version = t.version + 1,\n", "    SET version = t.version + 1, updated_at = now(),\n"),
    (9, "the rename in the record's diff (add as replace)", REPAIRS + "/content", "format('{\"op\":\"add\",\"path\":\"/%s\",\"value\":%s}', v_new, x.value)", "format('{\"op\":\"replace\",\"path\":\"/%s\",\"value\":%s}', v_new, x.value)"),
    (9, "diff_truncated on the record entries", REPAIRS + "/lists", "convert_to(r.diff, 'UTF8')), r.truncated,", "convert_to(r.diff, 'UTF8')), false,"),
    (10, "the rename in records.data", API, "THEN (t.data - c_old) || jsonb_build_object(v_new, t.data -> c_old) ELSE t.data END", "THEN t.data ELSE t.data END"),
    (10, "the key in the root properties", API, "to_json(CASE p.key WHEN c_old THEN v_new ELSE p.key END)::text", "to_json(p.key)::text"),
    (11, "the end when nothing is refused", REPAIRS + "/content", RETURN_EMPTY, "        RAISE NOTICE 'nothing to repair';\n"),
    (12, "exactly 500 entries", REPAIRS + "/lists", "coalesce(h.entries = 500, false)", "coalesce(h.entries >= 500, false)"),
    (12, "the list of records losing their page", REPAIRS + "/lists", "WHERE r.loses_history_page ORDER", "WHERE false ORDER"),
    (12, "the list of truncated records", REPAIRS + "/lists", "WHERE r.truncated ORDER", "WHERE false ORDER"),
    (12, "the truncated flag of a version", REPAIRS + "/lists", "p.truncated AS diff_truncated", "false AS diff_truncated"),
    (12, "the limit of 65,536 bytes", REPAIRS + "/lists", "c_max CONSTANT integer := 65536;", "c_max CONSTANT integer := 65535;"),
    (12, "the closing bracket in the size", REPAIRS + "/lists", "1 + sum(octet_length(u.op) + 1) OVER (ORDER BY u.n) AS size\n            FROM unnest", "sum(octet_length(u.op) + 1) OVER (ORDER BY u.n) AS size\n            FROM unnest"),
    (13, "the lock inside the block", PASTES + "/a_failed_lock", None, None),
    (13, "R1, before paste 2 is pasted on", PASTES + "/paste_2", *off("R1:")),
    (13, "paste 2 reads the plan of this transaction (the plan outlives ROLLBACK)", PASTES + "/paste_2", None, None),
    (14, "the count of updated records", PASTES + "/a_failed_assertion", "    IF v_got <> v_want THEN\n        RAISE EXCEPTION 'updated % records", "    IF false THEN\n        RAISE EXCEPTION 'updated % records"),
    (14, "the record's planned version", PASTES + "/a_record", " AND t.version = r.version;", ";"),
    (14, "nava_app for the records and the entries", PASTES + "/records_are", "    PERFORM set_config('role', 'nava_app', true);\n", ""),
    (15, "session_replication_role in paste 1", PASTES + "/session", "    IF current_setting('session_replication_role') <> 'origin' THEN\n        RAISE EXCEPTION 'session_replication_role is not origin: triggers would not fire';\n    END IF;\n    IF coalesce", "    IF coalesce"),
    (15, "session_replication_role in paste 2", PASTES + "/session", "    -- 1. Triggers fire.\n    IF current_setting('session_replication_role') <> 'origin' THEN", "    IF false THEN"),
    (15, "the trigger modes", PASTES + "/the_trigger", "IS DISTINCT FROM (SELECT p.triggers FROM pg_temp.nava_repair_plan p WHERE p.kind = 'triggers') THEN", "IS NULL THEN"),
    (16, "the lock", PASTES + "/a_writer_that_began_after", "    LOCK TABLE public.entity_definition_versions IN ACCESS EXCLUSIVE MODE;\n", ""),
    (16, "the lock, with a reader", PASTES + "/a_failed_lock", "    LOCK TABLE public.entity_definition_versions IN ACCESS EXCLUSIVE MODE;\n", ""),
    (16, "R9, sessions", PASTES + "/R9_a_writer", *off("R9: a service")),
    (17, "both ids in the one superuser UPDATE", REPAIRS, "WHERE v.tenant_id = v_tenant AND v.definition_id = v_type AND p.kind = 'version'", "WHERE p.kind = 'version'"),
    (17, "both ids in the UPDATE, and its row count", REPAIRS, None, None),
    (18, "the refused versions", CHECKS, "    -> 'properties' -> 'org_unit_id') IS NOT NULL\nORDER BY", "    -> 'properties' -> 'x') IS NOT NULL\nORDER BY"),
    (18, "the NUL escape", CHECKS, "'\\u0000', '?'), '\\ud', '?'), '\\uD', '?')::json\n    -> 'properties' -> 'org_unit_id') IS NOT NULL\nORDER BY", "'~', '?'), '\\ud', '?'), '\\uD', '?')::json\n    -> 'properties' -> 'org_unit_id') IS NOT NULL\nORDER BY"),
    (18, "bytes, not characters", CHECKS, "sum(octet_length(schema::text))", "sum(length(schema::text))"),
    (18, "the version count", CHECKS, "SELECT count(*) AS versions", "SELECT count(DISTINCT tenant_id) AS versions"),
    (19, "the row's tenant", CHECKS, "o.tenant_id = a.tenant_id AND o.id::text", "o.id::text"),
    (19, "the comparison as text", CHECKS, "o.id::text = lower(x.id)", "o.id = x.id::uuid"),
    (19, "upper-case ids", CHECKS, "= lower(x.id)", "= x.id"),
    (20, "tenant_id", REFUSES + "/input", *off("tenant_id is not")),
    (20, "type_id", REFUSES + "/input", *off("type_id is not")),
    (20, "new_name", REFUSES + "/input", *off("new_name does")),
    (20, "source_ref", REFUSES + "/input", *off("source_ref is not")),
    (20, "record_limit", REFUSES + "/input", *off("record_limit is not")),
    (20, "the isolation level", PASTES + "/session", *off("the transaction is not")),
    (20, "lock_timeout", PASTES + "/session", *off("lock_timeout is not")),
    (21, "json, not jsonb, in the rebuild", REPAIRS + "/content", None, None),
    (21, "the order of properties", REPAIRS + "/content", "',' ORDER BY p.n) || '}'", "',' ORDER BY p.key) || '}'"),
    (21, "the order of required", REPAIRS + "/content", "',' ORDER BY q.n), '')", "',' ORDER BY q.value::text), '')"),
    (21, "the entry in required", REPAIRS + "/content", "CASE WHEN q.value #>> '{}' = c_old THEN to_json(v_new)::text ELSE q.value::text END", "q.value::text"),
    (21, "an escaped key is found", REPAIRS + "/content", None, None),
    (21, "compat stays", REPAIRS + "/content", "SET schema = p.new_schema::json,", "SET published_at = now(), schema = p.new_schema::json,"),
    (22, "the order of the record's ops", REPAIRS + "/content", None, None),
    (22, "the replace op of required", REPAIRS + "/content", "\"path\":\"/required/%s\",\"value\":\"%s\"}', q.n - 1, v_new)", "\"path\":\"/required/%s\",\"value\":\"%s\"}', q.n, v_new)"),
    (22, "version + 1", API, "    SET version = t.version + 1,\n", "    SET version = t.version + 0,\n"),
    (22, "unique after the repair: the rename in data", API, "jsonb_build_object(v_new, t.data -> c_old) ELSE", "jsonb_build_object(v_new, 'x'::text) ELSE"),
]

LOCK = "    LOCK TABLE public.entity_definition_versions IN ACCESS EXCLUSIVE MODE;\n"


def special(test, what, doc):
    """Mutants of more than one line."""
    def sub(d, old, new):
        assert d.count(old) == 1, (what, d.count(old), old)
        return d.replace(old, new)
    if what.startswith("a salt per row, and"):
        d = sub(doc, SALT + ",\n        d.diff, d.truncated, coalesce", "(SELECT " + SALT + "),\n        d.diff, d.truncated, coalesce")
        return sub(d, "    IF v_got <> v_want THEN\n        RAISE EXCEPTION '% distinct salts", "    IF false THEN\n        RAISE EXCEPTION '% distinct salts")
    if what == "the lock inside the block":
        return sub(doc, LOCK, "").replace("DO $nava_repair$\nDECLARE\n    c_old CONSTANT text := 'org_unit_id';\n    -- audit",
                                          LOCK.strip() + "\nDO $nava_repair$\nDECLARE\n    c_old CONSTANT text := 'org_unit_id';\n    -- audit")
    if what.startswith("paste 2 reads the plan"):
        d = doc.replace("CREATE TEMP TABLE nava_repair_", "CREATE TEMP TABLE IF NOT EXISTS nava_repair_").replace(") ON COMMIT DROP;", ");")
        return sub(d, "BEGIN ISOLATION LEVEL READ COMMITTED;\n", "").replace("END\n$nava_repair$;\nCOMMIT;", "END\n$nava_repair$;")
    if what.startswith("both ids in the UPDATE, and"):
        d = sub(doc, "WHERE v.tenant_id = v_tenant AND v.definition_id = v_type AND p.kind = 'version'", "WHERE p.kind = 'version'")
        return sub(d, "    IF v_got <> v_want THEN\n        RAISE EXCEPTION 'updated % versions", "    IF false THEN\n        RAISE EXCEPTION 'updated % versions")
    if what.startswith("json, not jsonb"):
        d = sub(doc, "FROM json_each(v.schema) WITH ORDINALITY e", "FROM json_each(v.schema::jsonb::json) WITH ORDINALITY e")
        return d
    if what == "an escaped key is found":
        # The rename on the raw text instead of the decoded key.
        return sub(doc, "SELECT 'version', v.version, s.txt, sha256", "SELECT 'version', v.version, s.txt, sha256").replace(
            "AND (replace(replace(replace(replace(v.schema::text,\n          '\\\\', '?'), '\\u0000', '?'), '\\ud', '?'), '\\uD', '?')::json\n          -> 'properties' -> 'org_unit_id') IS NOT NULL;",
            "AND strpos(v.schema::text, '\"org_unit_id\"') > 0;")
    if what == "the order of the record's ops":
        a = "                format('{\"op\":\"add\",\"path\":\"/%s\",\"value\":%s}', v_new, x.value),\n"
        b = "                format('{\"op\":\"remove\",\"path\":\"/%s\"}', c_old)]) WITH"
        return sub(sub(doc, a, "                format('{\"op\":\"remove\",\"path\":\"/%s\"}', c_old),\n"), b,
                   "                format('{\"op\":\"add\",\"path\":\"/%s\",\"value\":%s}', v_new, x.value)]) WITH")
    raise AssertionError(what)


def run(pattern):
    server = pattern == API
    exe = os.path.join(BIN, "server.exe" if server else "dbtest.exe")
    cwd = os.path.join(ROOT, "internal", "server" if server else os.path.join("db", "dbtest"))
    parts = pattern.split("/", 1)
    run_re = "^" + parts[0] + "$" + ("/" + parts[1] if len(parts) > 1 else "")
    p = subprocess.run([exe, "-test.run", run_re, "-test.count=1", "-test.v"], cwd=cwd, capture_output=True, timeout=900)
    out = p.stdout.decode(errors="replace")
    failed = re.findall(r"--- FAIL: (\S+)", out)
    leaf = [f for f in failed if not any(g != f and g.startswith(f + "/") for g in failed)]
    ran = len(re.findall(r"--- (?:PASS|FAIL): ", out))
    return leaf, ran, out


def main():
    with open(DOC, encoding="utf-8", newline="") as f:
        original = f.read()
    doc = original.replace("\r\n", "\n")
    survivors = 0
    try:
        for i, m in enumerate(M):
            if i % SHARDS != SHARD:
                continue
            test, what, pattern = m[:3]
            if ONLY and not re.search(ONLY, what):
                continue
            if m[3] is None:
                mutated = special(test, what, doc)
            elif len(m) > 5 and m[5]:
                mutated, n = re.subn(m[3], m[4], doc, count=1, flags=re.S)
                assert n == 1, what
            else:
                assert doc.count(m[3]) == 1, (what, doc.count(m[3]))
                mutated = doc.replace(m[3], m[4])
            assert mutated != doc, what
            with open(DOC, "w", encoding="utf-8", newline="") as f:
                f.write(mutated)
            leaf, ran, out = run(pattern)
            if not leaf:
                survivors += 1
            first = next((l.strip() for l in out.splitlines() if re.search(r"_test\.go:\d+:", l)), "")
            names = ", ".join(x.replace("TestTICK1125", "") for x in leaf[:3]) + (f" and {len(leaf) - 3} more" if len(leaf) > 3 else "")
            print(f"{test:>2} | {what} | {'RED' if leaf else 'SURVIVED'} | {names} | {first[:200]}", flush=True)
    finally:
        with open(DOC, "w", encoding="utf-8", newline="") as f:
            f.write(original)
    print(f"survivors: {survivors}")
    sys.exit(1 if survivors else 0)


main()
