// S14 acceptance (TICK-437), used by upgrade-setup.sh. Dev-only seed credentials.
// Applies the user part of the S14 realm-nava.json delta to an upgraded stack's existing realm
// (Keycloak only imports a realm that does not exist yet). The auditors groups are not made here:
// upgrade-setup.sh runs `nava-onboard reconcile-auditors` first (TICK-567). This adds
// auditor@tenant-a.test to A's auditors group and A's organization.
const KC = 'http://127.0.0.1:18081', R = `${KC}/admin/realms/nava`;
const A = '00000000-0000-4000-8000-0000000000a1';
const t = await (await fetch(`${KC}/realms/master/protocol/openid-connect/token`, { method: 'POST',
  body: new URLSearchParams({ grant_type: 'password', client_id: 'admin-cli', username: 'admin', password: 'dev-only-admin' }) })).json();
const H = { Authorization: 'Bearer ' + t.access_token, 'Content-Type': 'application/json' };
const kc = async (m, p, body) => { const r = await fetch(R + p, { method: m, headers: H, body: body && JSON.stringify(body) }); const x = await r.text(); return { s: r.status, j: x ? JSON.parse(x) : null }; };
let u = await kc('POST', '/users', { username: 'auditor@tenant-a.test', email: 'auditor@tenant-a.test', emailVerified: true, firstName: 'Auditor', lastName: 'A', enabled: true,
  credentials: [{ type: 'password', value: 'dev-only-auditor-a', temporary: false }] });
console.log('auditor user: HTTP', u.s);
const uid = (await kc('GET', '/users?username=auditor%40tenant-a.test&exact=true')).j[0].id;
const g = (await kc('GET', `/group-by-path/t/${A}/auditors`)).j;
console.log('join group: HTTP', (await kc('PUT', `/users/${uid}/groups/${g.id}`)).s);
const orgs = (await kc('GET', '/organizations?max=100')).j;
const org = orgs.find(o => o.name === A || o.alias === A || (o.domains || []).some(d => d.name === 'tenant-a.test'));
const mr = await fetch(`${R}/organizations/${org.id}/members`, { method: 'POST', headers: H, body: JSON.stringify(uid) });
console.log('org', org.name, 'member: HTTP', mr.status);
