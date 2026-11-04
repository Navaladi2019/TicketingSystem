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
	// ADR-001 A.3: one role per system actor. The runtime roles log in
	// through PgBouncer with CONNECT and USAGE on public; nava_backup is
	// LOGIN REPLICATION with no grant at all.
	// nava_audit_reader (ADR-004-A1 A1-1) is the masking reader's own login.
	runtime := []string{"nava_app", "nava_audit_reader", "nava_groupsync", "nava_onboard", "nava_purge", "nava_relay", "nava_worker"}
	if got := m.PgBouncerLogins(); strings.Join(got, ",") != strings.Join(runtime, ",") {
		t.Fatalf("PgBouncer allow-list = %v, want %v", got, runtime)
	}
	// nava_erasure (ADR-001-A6.1) is a runtime role without PgBouncer: its
	// credential lives with the job runner and it connects directly.
	want := append([]string{"keycloak", "nava_backup", "nava_erasure", "nava_migrator", "pgbouncer_auth", "temporal"}, runtime...)
	for _, name := range want {
		if _, ok := m.Roles[name]; !ok {
			t.Errorf("role %s missing", name)
		}
	}
	if len(m.Roles) != len(want) {
		t.Errorf("roles = %v, want exactly %v", m.RoleNames(), want)
	}
	for _, name := range append([]string{"nava_erasure"}, runtime...) {
		r := m.Roles[name]
		if strings.Join(r.Attributes, ",") != "LOGIN" || strings.Join(r.Database, ",") != "CONNECT" ||
			strings.Join(r.Schemas["public"], ",") != "USAGE" || len(r.Schemas) != 1 {
			t.Errorf("runtime role %s: %+v, want LOGIN, CONNECT and USAGE on public", name, r)
		}
	}
	if e := m.Roles["nava_erasure"]; e.PgBouncerLogin || len(e.Settings)+len(e.Functions) != 0 {
		t.Errorf("nava_erasure: %+v, want no PgBouncer login, settings or functions", e)
	}
	b := m.Roles["nava_backup"]
	if strings.Join(b.Attributes, ",") != "LOGIN,REPLICATION" || b.PgBouncerLogin ||
		len(b.Database)+len(b.Schemas)+len(b.Tables)+len(b.Functions) != 0 {
		t.Errorf("nava_backup: %+v, want LOGIN REPLICATION and no grants", b)
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
		{"allowed setting", "database: [CONNECT]", "database: [CONNECT]\n    settings: { plan_cache_mode: force_custom_plan }", ""},
		{"unknown setting", "database: [CONNECT]", "database: [CONNECT]\n    settings: { row_security: \"off\" }", `setting "row_security" is not one of`},
		{"bad setting value", "database: [CONNECT]", "database: [CONNECT]\n    settings: { plan_cache_mode: custom }", `plan_cache_mode = "custom" is not one of`},
		{"owner with settings", "owner: { attributes: [LOGIN] }", "owner: { attributes: [LOGIN], settings: { plan_cache_mode: auto } }", "the owner takes no settings"},
		{"second document", "reason: test\n", "reason: test\n---\nroles: {}\n", "more than one document"},
		{"platform database not an identifier", "platform_database: nava", "platform_database: Nava", `platform_database "Nava" is not a lower-case identifier`},
		{"platform database not listed", "platform_database: nava", "platform_database: other", `platform_database "other" is not listed under databases`},
		{"database not an identifier", "nava: { owner: owner }", "nava: { owner: owner }\n  Keycloak: { owner: owner }", `database "Keycloak" is not a lower-case identifier`},
		{"exception name not an identifier", "  pgbouncer_get_auth:", "  PgBouncer:", `infrastructure_exceptions: "PgBouncer" is not a lower-case identifier`},
		{"exception in public", "    schema: pgbouncer", "    schema: public", `schema "public" must be a dedicated schema`},
		{"exception function malformed", "    function: pgbouncer.get_auth(text)", "    function: get_auth", `infrastructure_exceptions.pgbouncer_get_auth: function "get_auth" is not schema.name(argtypes)`},
		{"exception function outside its schema", "    function: pgbouncer.get_auth(text)", "    function: public.get_auth(text)", `function "public.get_auth(text)" is not in schema "pgbouncer"`},
		{"exception granted to no one", "granted_to: [auth]", "granted_to: []", "pgbouncer_get_auth: granted_to is empty"},
		{"exception granted to an unknown role", "granted_to: [auth]", "granted_to: [auth, ghost]", `granted_to "ghost" is not a role in this manifest`},
		{"repeated attribute", "app:\n    attributes: [LOGIN]", "app:\n    attributes: [LOGIN, LOGIN]", "role app: attributes repeat"},
		{"schema not an identifier", "schemas: { public: [USAGE] }", "schemas: { public: [USAGE], Bad: [USAGE] }", `role app: schema "Bad" is not a lower-case identifier`},
		{"exception schema granted to another role", "schemas: { public: [USAGE] }", "schemas: { public: [USAGE], pgbouncer: [USAGE] }", "role app: schema pgbouncer belongs to infrastructure exception pgbouncer_get_auth"},
		{"column not an identifier", "SELECT: [id, tenant_id]", "SELECT: [id, Tenant]", `column "Tenant" is not a lower-case identifier`},
		{"schema-qualified table", "outbox: {", "public.outbox: {", ""},
		{"bare function name in public", `"public.f(integer)"`, `"f(integer)"`, ""},
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

func TestValidateEmptyManifest(t *testing.T) {
	t.Parallel()
	err := (&Manifest{}).Validate()
	if err == nil {
		t.Fatal("an empty manifest validates")
	}
	for _, want := range []string{`platform_database "" is not a lower-case identifier`, "no roles"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %v, want one containing %q", err, want)
		}
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
