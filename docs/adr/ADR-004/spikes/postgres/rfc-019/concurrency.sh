# Run against the spike.sql database (container rfc019-pg). C1: VALIDATE held 3 s,
# concurrent INSERT/SELECT with lock_timeout 1 s. C2: migration A held 3 s,
# concurrent INSERT. C3: migration A behind an open reader with lock_timeout 5 s.
P="docker exec -i rfc019-pg psql -U postgres -X -q"
( printf "BEGIN;\nALTER TABLE audit_log VALIDATE CONSTRAINT audit_log_action_check;\nSELECT pg_sleep(3);\nCOMMIT;\n" | $P ) & sleep 1
$P -c "SET lock_timeout='1s'" -c "INSERT INTO audit_log VALUES (gen_random_uuid(), DEFAULT, '2026-06-01', 'number_space_exhausted')" -c "SELECT count(*) FROM audit_log_2026_06"; wait
( printf "BEGIN;\nALTER TABLE audit_log DROP CONSTRAINT audit_log_action_check;\nALTER TABLE audit_log ADD CONSTRAINT audit_log_action_check CHECK (action IN ('create','update','publish','delete','number_space_exhausted')) NOT VALID;\nSELECT pg_sleep(3);\nCOMMIT;\n" | $P ) & sleep 1
$P -c "SET lock_timeout='1s'" -c "INSERT INTO audit_log VALUES (gen_random_uuid(), DEFAULT, '2026-06-01', 'create')"; wait
( printf "BEGIN;\nSELECT count(*) FROM audit_log_2026_06;\nSELECT pg_sleep(7);\nCOMMIT;\n" | $P >/dev/null ) & sleep 1
$P -c "BEGIN" -c "SET LOCAL lock_timeout='5s'" -c "ALTER TABLE audit_log DROP CONSTRAINT audit_log_action_check" -c "COMMIT"; wait
