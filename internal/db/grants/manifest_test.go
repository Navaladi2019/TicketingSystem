package grants

import (
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/db"
)

func TestRepositoryManifestParses(t *testing.T) {
	t.Parallel()
	m, err := Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	if m.Owner() != "nava_migrator" {
		t.Fatalf("platform database owner = %q, want nava_migrator", m.Owner())
	}
	if got := m.PgBouncerLogins(); len(got) != 1 || got[0] != "nava_app" {
		t.Fatalf("PgBouncer allow-list = %v, want [nava_app]", got)
	}
	for _, name := range []string{"nava_migrator", "nava_app"} {
		if _, ok := m.Roles[name]; !ok {
			t.Errorf("role %s missing", name)
		}
	}
}

const base = `
platform_database: nava
databases:
  nava: { owner: owner }
roles:
  owner: { attributes: [LOGIN] }
  app:
    attributes: [LOGIN]
    database: [CONNECT]
    schemas: { public: [USAGE] }
    tables:
      outbox: { SELECT: [id, tenant_id], DELETE: [] }
    functions: { "public.f(integer)": [EXECUTE] }
  auth:
    attributes: [LOGIN]
    schemas: { pgbouncer: [USAGE] }
    functions: { "pgbouncer.get_auth(text)": [EXECUTE] }
infrastructure_exceptions:
  pgbouncer_get_auth:
    schema: pgbouncer
    function: pgbouncer.get_auth(text)
    security_definer: true
    granted_to: [auth]
    reason: test
`

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr string
	}{
		{"base is valid", "", "", ""},
		{"typo'd key", "functions: { \"public", "functons: { \"public", "field functons not found"},
		{"unknown attribute", "app:\n    attributes: [LOGIN]", "app:\n    attributes: [LOGIN, SUPERUSER]", `"SUPERUSER" is not LOGIN or REPLICATION`},
		{"unknown privilege", "database: [CONNECT]", "database: [CONNECT, ALL]", `privilege "ALL"`},
		{"columns on a table-level privilege", "DELETE: []", "DELETE: [id]", "DELETE is table-level only"},
		{"owner missing", "nava: { owner: owner }", "nava: { owner: nobody }", `owner "nobody" is not a role`},
		{"owner with grants", "owner: { attributes: [LOGIN] }", "owner: { attributes: [LOGIN], database: [CONNECT] }", "the owner holds every privilege"},
		{"owner on the pooler", "owner: { attributes: [LOGIN] }", "owner: { attributes: [LOGIN], pgbouncer_login: true }", "may not log in through PgBouncer"},
		{"replication role on the pooler", "app:\n    attributes: [LOGIN]", "app:\n    pgbouncer_login: true\n    attributes: [LOGIN, REPLICATION]", "pgbouncer_login needs LOGIN and no REPLICATION"},
		{"exception function granted to another role", `functions: { "public.f(integer)": [EXECUTE] }`, `functions: { "pgbouncer.get_auth(text)": [EXECUTE] }`, "belongs to infrastructure exception"},
		{"exception without its grants", "schemas: { pgbouncer: [USAGE] }", "schemas: {}", "must list USAGE on schema pgbouncer"},
		{"exception without reason", "reason: test", "reason: ''", "reason is required"},
		{"bad function", `"public.f(integer)"`, `"f; DROP"`, "is not schema.name(argtypes)"},
		{"bad table", "outbox: {", "\"Outbox\": {", "is not schema.table"},
		{"pg_ role", "  app:\n", "  pg_app:\n", "outside pg_"},
		{"repeated privilege", "database: [CONNECT]", "database: [CONNECT, CONNECT]", "privileges repeat"},
		{"repeated column", "SELECT: [id, tenant_id]", "SELECT: [id, id]", "columns repeat"},
		{"duplicate role", "  auth:\n", "  app:\n", `"app" already defined`},
		{"second document", "reason: test\n", "reason: test\n---\nroles: {}\n", "more than one document"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in := base
			if tt.from != "" {
				if !strings.Contains(in, tt.from) {
					t.Fatalf("test fixture does not contain %q", tt.from)
				}
				in = strings.Replace(in, tt.from, tt.to, 1)
			}
			_, err := Parse([]byte(in))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestStatement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		verb       string
		g          Grant
		optionOnly bool
		want       string
	}{
		{"REVOKE", Grant{Kind: "database", Name: "nava", Grantee: "PUBLIC", Privilege: "CONNECT"}, false,
			`REVOKE CONNECT ON DATABASE "nava" FROM PUBLIC CASCADE`},
		{"GRANT", Grant{Kind: "column", Schema: "public", Name: "outbox", Column: "id", Grantee: "nava_relay", Privilege: "SELECT"}, false,
			`GRANT SELECT ("id") ON TABLE "public"."outbox" TO "nava_relay"`},
		{"REVOKE", Grant{Kind: "table", Schema: "public", Name: "s", Grantee: "x", Privilege: "USAGE", Sequence: true}, true,
			`REVOKE GRANT OPTION FOR USAGE ON SEQUENCE "public"."s" FROM "x" CASCADE`},
		{"GRANT", Grant{Kind: "function", Name: "public.f(integer)", Grantee: "x", Privilege: "EXECUTE"}, false,
			`GRANT EXECUTE ON ROUTINE public.f(integer) TO "x"`},
		{"REVOKE", Grant{Kind: "default", Name: "f", Grantee: "PUBLIC", Privilege: "EXECUTE"}, false,
			`ALTER DEFAULT PRIVILEGES FOR ROLE "nava_migrator" REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC CASCADE`},
		{"REVOKE", Grant{Kind: "default", Schema: "public", Name: "r", Grantee: "x", Privilege: "SELECT"}, false,
			`ALTER DEFAULT PRIVILEGES FOR ROLE "nava_migrator" IN SCHEMA "public" REVOKE SELECT ON TABLES FROM "x" CASCADE`},
	}
	for _, tt := range tests {
		got, err := statement(tt.verb, tt.g, "nava_migrator", tt.optionOnly)
		if err != nil || got != tt.want {
			t.Errorf("statement(%s, %s) = %q, %v; want %q", tt.verb, tt.g, got, err, tt.want)
		}
	}
	for _, g := range []Grant{
		{Kind: "database", Name: "nava", Grantee: "x", Privilege: "CONNECT; DROP"},
		{Kind: "default", Name: "f", Grantee: "x", Privilege: "EXECUTE"}, // defaults are never granted
		{Kind: "type", Name: "t", Grantee: "x", Privilege: "USAGE"},
	} {
		if s, err := statement("GRANT", g, "o", false); err == nil {
			t.Errorf("statement(GRANT, %+v) = %q, want an error", g, s)
		}
	}
}
