// S14 acceptance (TICK-437), used by upgrade-setup.sh. Dev-only seed credentials.
// Publishes the next incident version as admin@tenant-a.test (pre-upgrade step of the S14 kit).
const KC = 'http://127.0.0.1:18081', CORE = 'http://127.0.0.1:18080';
const tok = await (await fetch(`${KC}/realms/nava/protocol/openid-connect/token`, { method: 'POST',
  headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
  body: new URLSearchParams({ grant_type: 'password', client_id: 'nava-web', scope: 'openid', username: 'admin@tenant-a.test', password: 'dev-only-admin-a' }) })).json();
const H = { Authorization: 'Bearer ' + tok.access_token };
const cur = await (await fetch(`${CORE}/api/v1/entity-types/incident`, { headers: H })).json();
cur.schema.properties.impact_pre_s14 = { type: 'string', title: 'Impact (published before S14)', maxLength: 200 };
const r = await fetch(`${CORE}/api/v1/entity-types/incident/versions`, { method: 'POST',
  headers: { ...H, 'Content-Type': 'application/json', 'Idempotency-Key': 'tick437-pre-s14-publish' },
  body: JSON.stringify({ definition: { schema: cur.schema, fixed_columns: cur.fixed_columns, states: cur.states } }) });
const b = await r.json();
console.log('publish before upgrade: HTTP', r.status, 'version', b.version, 'compat', b.compat);
